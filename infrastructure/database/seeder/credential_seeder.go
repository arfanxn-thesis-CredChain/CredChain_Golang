package seeder

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	infraCrypto "CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/storage"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/samber/lo"
)

type CredentialSeeder struct {
	credRepo       domain.CredentialRepository
	userRepo       domain.UserRepository
	typeRepo       domain.CredentialTypeRepository
	issuerOrgRepo  domain.CredentialIssuerOrganizationRepository
	competencyRepo domain.CompetencyRepository
	compCredRepo   domain.CompetencyCredentialRepository
	extractRepo    domain.CredentialExtractionRepository
	storage        *storage.Storage
	cfg            *config.Config
}

func NewCredentialSeeder(
	credRepo domain.CredentialRepository,
	userRepo domain.UserRepository,
	typeRepo domain.CredentialTypeRepository,
	issuerOrgRepo domain.CredentialIssuerOrganizationRepository,
	competencyRepo domain.CompetencyRepository,
	compCredRepo domain.CompetencyCredentialRepository,
	extractRepo domain.CredentialExtractionRepository,
	storage *storage.Storage,
	cfg *config.Config,
) *CredentialSeeder {
	return &CredentialSeeder{
		credRepo:       credRepo,
		userRepo:       userRepo,
		typeRepo:       typeRepo,
		issuerOrgRepo:  issuerOrgRepo,
		competencyRepo: competencyRepo,
		compCredRepo:   compCredRepo,
		extractRepo:    extractRepo,
		storage:        storage,
		cfg:            cfg,
	}
}

func (s *CredentialSeeder) Name() string { return "credential" }

// credWorkflow is the code path a seeded row pretends to have come from.
// Every seeded credential must be reachable by one of the two, otherwise it is
// a shape no API call can produce and worthless as a fixture.
type credWorkflow int

const (
	// workflowIssue is POST /api/credentials/batch/issue — an Issuer+ writes a
	// credential for a holder. The row is born approved: submitter, issuer and
	// approver are all the officer, and extraction is enqueued at creation
	// (credential_service.go issuePrepareCredentials).
	workflowIssue credWorkflow = iota
	// workflowSubmit is POST /api/credentials/batch/submit — a holder files
	// their own document. holder, submitter and issuer all start as the holder;
	// approval later reassigns issuer_user_id to the reviewing officer
	// (credential_service.go submitPrepareCredentials / Approve).
	workflowSubmit
)

// credOutcome is where the row ends up once the workflow has played out.
// It maps onto domain.CredentialStatus, which is derived from timestamps.
type credOutcome int

const (
	outcomePending credOutcome = iota
	outcomeApproved
	outcomeRejected
	outcomeRevoked
)

// credExtract mirrors domain.ExtractState. Extraction is enqueued at approval,
// so anything that never got approved is unextracted.
type credExtract int

const (
	extractUnextracted credExtract = iota
	extractPending
	extractSucceeded
	extractFailed
)

// credMetadata is the type/organization shape of the row.
type credMetadata int

const (
	// metadataResolved — FKs set, nothing staged. Direct issuance always, and
	// a submission whose names matched existing taxonomy rows.
	metadataResolved credMetadata = iota
	// metadataStaged — free text only, FKs null. A submission that named
	// nothing the taxonomy knows; illegal on an approved row
	// (chk_credentials_approved_metadata_resolved).
	metadataStaged
	// metadataResolvedFromStaged — a reviewer ran ResolveMetadata: the FKs are
	// filled in but the submitted name survives for audit, since ResolveMetadata
	// never clears it.
	metadataResolvedFromStaged
)

// seedCredSpec describes one credential as a workflow plus an end state,
// instead of as a bag of timestamps. Everything else (participants, staged
// names, competency links, extraction) is derived from these fields, so an
// impossible combination cannot be written by accident.
type seedCredSpec struct {
	name     string
	workflow credWorkflow
	outcome  credOutcome
	extract  credExtract
	metadata credMetadata

	holderID string // credential subject
	// officerID is the Issuer+ who issued it directly, or who reviewed the
	// submission. Empty only for rows still pending review.
	officerID string
	// revokerID is the Issuer+ who revoked it (outcomeRevoked only).
	revokerID string

	// expired gives the row a past expires_at. Expiry is an attribute, not a
	// status — it is evaluated only on the verify path.
	expired bool
	// fileTwin reuses the named spec's file bytes verbatim, producing a second
	// row with an identical file_hash. Legal only when one of the pair is
	// rejected or revoked (idx_credentials_file_hash_active is partial).
	fileTwin string
}

// seedCredActors are the fixed users the scenario matrix addresses by name.
// They are looked up by the deterministic ULID the user seeder derives from
// the wallet index: userRepo.Get sorts by updated_at, so slice position is not
// a stable handle.
type seedCredActors struct {
	superAdmin    string
	admin         string
	issuer        string
	holder        string
	deletedHolder string
}

// Staged free text deliberately matches no taxonomy row — a submit that hit an
// existing name would have resolved it to the FK instead of staging it.
const (
	seedStagedTypeName       = "Sertifikat Pelatihan Mandiri"
	seedStagedOrgName        = "Lembaga Pelatihan Swadaya Nusantara"
	seedStagedCompetencyName = "Kompetensi Belum Terdaftar"
)

func (s *CredentialSeeder) Seed(ctx context.Context) error {
	users, _, err := s.userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch users: %w", err)
	}
	usersByID := lo.SliceToMap(users, func(u domain.User) (string, domain.User) { return u.Id, u })

	actors := seedCredActors{
		superAdmin:    deterministicULID(1),
		admin:         deterministicULID(2),
		issuer:        deterministicULID(3),
		holder:        deterministicULID(4),
		deletedHolder: deterministicULID(5),
	}
	if err := seedCredValidateActors(usersByID, actors); err != nil {
		return err
	}

	var holders, issuers []domain.User
	for _, u := range users {
		if u.DeletedAt != nil {
			continue
		}
		switch u.Role {
		case domain.RoleHolder:
			holders = append(holders, u)
		case domain.RoleIssuer:
			issuers = append(issuers, u)
		}
	}
	if len(holders) == 0 || len(issuers) == 0 {
		return fmt.Errorf("credential seeder: need active holders and issuers to seed credentials")
	}

	types, _, err := s.typeRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch types: %w", err)
	}
	activeTypes := lo.Filter(types, func(t domain.CredentialType, _ int) bool { return t.Active })
	if len(activeTypes) == 0 {
		return fmt.Errorf("credential seeder: no active credential types found")
	}

	orgs, _, err := s.issuerOrgRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch orgs: %w", err)
	}
	activeOrgs := lo.Filter(orgs, func(o domain.CredentialIssuerOrganization, _ int) bool { return o.Active })
	if len(activeOrgs) == 0 {
		return fmt.Errorf("credential seeder: no active issuer orgs found")
	}

	comps, _, err := s.competencyRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch competencies: %w", err)
	}
	activeComps := lo.Filter(comps, func(c domain.Competency, _ int) bool { return c.Active })

	specs := append(seedCredFixedSpecs(actors), seedCredFillerSpecs(holders, issuers)...)
	if err := seedCredValidateSpecs(specs, usersByID); err != nil {
		return err
	}

	seed := hashToSeed("credchain-seed-credential")
	rng := rand.New(rand.NewSource(seed))

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	credentials := make([]domain.Credential, len(specs))
	var extractions []domain.CredentialExtraction
	var compCredLinks []domain.CompetencyCredential

	// Plaintext of every row built so far, so a fileTwin spec can reuse bytes
	// verbatim and land on the same file_hash.
	type seedCredFile struct {
		bytes []byte
		lines []string
	}
	filesByName := make(map[string]seedCredFile, len(specs))

	for i, spec := range specs {
		credID := deterministicULID(uint32(1000 + i))
		credType := activeTypes[i%len(activeTypes)]
		org := activeOrgs[i%len(activeOrgs)]

		issuedAt := baseTime.Add(time.Duration(rng.Int63n(180)) * 24 * time.Hour)
		number := fmt.Sprintf("CRED/%d/%04d", issuedAt.Year(), i+1)

		holderName := "Unknown"
		if h, ok := usersByID[spec.holderID]; ok && h.Name != nil {
			holderName = *h.Name
		}

		file := seedCredFile{
			lines: []string{
				fmt.Sprintf("Identifier: %s", credID),
				fmt.Sprintf("Holder: %s (ID: %s)", holderName, spec.holderID),
				fmt.Sprintf("Number: %s", number),
				fmt.Sprintf("Issuer Organization: %s", org.Name),
				fmt.Sprintf("Type: %s", credType.Name),
				fmt.Sprintf("Issued Date: %s", issuedAt.Format("2006-01-02")),
				fmt.Sprintf("Seed-Marker: seed-%d", i+1),
			},
		}
		file.bytes = seedPDFBytes(spec.name, file.lines)
		if spec.fileTwin != "" {
			twin, ok := filesByName[spec.fileTwin]
			if !ok {
				return fmt.Errorf("credential seeder: %q reuses unknown file twin %q", spec.name, spec.fileTwin)
			}
			file = twin
		}
		filesByName[spec.name] = file

		fileHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(file.bytes))

		encBytes, encErr := infraCrypto.Encrypt(file.bytes, []byte(*s.cfg.FileEncryptionKey))
		if encErr != nil {
			return fmt.Errorf("credential seeder: encrypt file: %w", encErr)
		}

		// A resubmitted document is a second upload: same bytes, own file.
		filename := fmt.Sprintf("%s.pdf", credID)
		filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
		if _, saveErr := s.storage.SaveBytes([]byte(encBytes), filePath); saveErr != nil {
			return fmt.Errorf("credential seeder: save file %s: %w", filename, saveErr)
		}

		c := domain.Credential{
			ID:           credID,
			HolderUserID: spec.holderID,
			Number:       &number,
			Name:         spec.name,
			FileHash:     fileHash,
			FileURI:      lo.ToPtr(filename),
			IssuedAt:     issuedAt,
		}

		// Participants and creation time. Direct issuance writes the row at
		// issuance; a self-submission files a document issued earlier.
		officerID := spec.officerID
		switch spec.workflow {
		case workflowIssue:
			c.SubmitterUserID = officerID
			c.IssuerUserID = officerID
			c.CreatedAt = issuedAt
		case workflowSubmit:
			c.SubmitterUserID = spec.holderID
			c.IssuerUserID = spec.holderID
			c.CreatedAt = issuedAt.Add(30 * 24 * time.Hour)
		}

		switch spec.metadata {
		case metadataResolved:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
		case metadataStaged:
			c.SubmittedTypeName = lo.ToPtr(seedStagedTypeName)
			c.SubmittedIssuerOrganizationName = lo.ToPtr(seedStagedOrgName)
		case metadataResolvedFromStaged:
			c.SubmittedTypeName = lo.ToPtr(seedStagedTypeName)
			c.SubmittedIssuerOrganizationName = lo.ToPtr(seedStagedOrgName)
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
		}

		// Competencies. A submission stages the names the holder typed and gets
		// resolved_id stamped once a reviewer links them; direct issuance takes
		// competency ids and only writes join rows. The join row and the
		// resolved_id are always written together (ResolveMetadata does both).
		if len(activeComps) > 0 {
			comp := activeComps[i%len(activeComps)]
			resolved := spec.metadata != metadataStaged
			if spec.workflow == workflowSubmit {
				sc := domain.SubmittedCompetency{Name: seedStagedCompetencyName}
				if resolved {
					sc = domain.SubmittedCompetency{Name: comp.Name, ResolvedID: &comp.Id}
				}
				c.SubmittedCompetencies = domain.SubmittedCompetencies{sc}
			}
			if resolved {
				compCredLinks = append(compCredLinks, domain.CompetencyCredential{
					CompetencyId: comp.Id,
					CredentialId: credID,
				})
			}
		}

		// Review. Direct issuance stamps approval at creation; a submission
		// waits for a reviewer, who also takes over issuer_user_id because they
		// are the wallet that writes the mint on chain.
		reviewedAt := c.CreatedAt
		if spec.workflow == workflowSubmit {
			reviewedAt = c.CreatedAt.Add(2 * 24 * time.Hour)
		}
		switch spec.outcome {
		case outcomeApproved, outcomeRevoked:
			c.ApprovedAt = &reviewedAt
			c.ApproverUserID = &officerID
			c.IssuerUserID = officerID
		case outcomeRejected:
			c.RejectedAt = &reviewedAt
			c.RejecterUserID = &officerID
			c.RejectionReason = lo.ToPtr("Dokumen tidak sesuai dengan standar verifikasi atau data tidak valid")
		case outcomePending:
		}
		if spec.outcome == outcomeRevoked {
			revokerID := spec.revokerID
			revokedAt := reviewedAt.Add(60 * 24 * time.Hour)
			c.RevokedAt = &revokedAt
			c.RevokerUserID = &revokerID
		}

		// Extraction, enqueued by the same call that approved the row.
		enqueuedAt := reviewedAt.Add(time.Minute)
		switch spec.extract {
		case extractPending:
			c.ExtractEnqueuedAt = &enqueuedAt
		case extractSucceeded:
			extractedAt := enqueuedAt.Add(2 * time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractedAt = &extractedAt
			extractions = append(extractions, domain.CredentialExtraction{
				CredentialID: credID,
				FileHash:     fileHash,
				Text:         fmt.Sprintf("%s\n%s", spec.name, strings.Join(file.lines, "\n")),
				IDs: []domain.CredentialExtractedID{
					{Type: "number", Value: number},
					{Type: "holder_id", Value: spec.holderID},
				},
				Embedding: seedEmbedding(rng, 768),
				CreatedAt: extractedAt,
				UpdatedAt: extractedAt,
			})
		case extractFailed:
			failedAt := enqueuedAt.Add(2 * time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractFailedAt = &failedAt
			c.ExtractError = lo.ToPtr("document text extraction failed: unreadable or corrupted OCR stream")
		case extractUnextracted:
		}

		if spec.expired {
			c.ExpiresAt = lo.ToPtr(issuedAt.Add(90 * 24 * time.Hour))
		}

		c.UpdatedAt = lo.ToPtr(seedCredLastTouched(c))
		credentials[i] = c
	}

	if _, err := s.credRepo.Store(ctx, credentials...); err != nil {
		return fmt.Errorf("credential seeder: store credentials: %w", err)
	}

	if len(compCredLinks) > 0 {
		if _, err := s.compCredRepo.Store(ctx, compCredLinks...); err != nil {
			return fmt.Errorf("credential seeder: store competency credentials: %w", err)
		}
	}

	// One Mongo document per row whose extraction actually succeeded — an
	// extracted_at with no document behind it is unverifiable through the
	// fuzzy path.
	if s.extractRepo != nil {
		for _, ext := range extractions {
			if err := s.extractRepo.Store(ctx, ext); err != nil {
				return fmt.Errorf("credential seeder: store extraction: %w", err)
			}
		}
	}

	return nil
}

// seedCredFixedSpecs is the scenario matrix: the rows that exist to be test
// cases rather than volume. A covers direct issuance, B the submit-then-review
// workflow, C the file_hash reuse the partial unique index allows.
func seedCredFixedSpecs(a seedCredActors) []seedCredSpec {
	return []seedCredSpec{
		// ── A · direct issuance (Issuer+ writes it, born approved) ──────────
		{
			name: "Ijazah Sarjana Komputer", workflow: workflowIssue,
			outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.holder, officerID: a.issuer,
		},
		// Admin and SuperAdmin outrank Issuer, so they issue directly too.
		{
			name: "Transkrip Akademik S1", workflow: workflowIssue,
			outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.holder, officerID: a.admin,
		},
		{
			name: "SKPI Sarjana Sistem Informasi", workflow: workflowIssue,
			outcome: outcomeApproved, extract: extractPending, metadata: metadataResolved,
			holderID: a.holder, officerID: a.superAdmin,
		},
		// Expiry is an attribute of an approved row, not a status of its own.
		{
			name: "Sertifikasi Lisensi Tahunan", workflow: workflowIssue,
			outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.holder, officerID: a.issuer, expired: true,
		},
		{
			name: "Ijazah Dibatalkan Rektorat", workflow: workflowIssue,
			outcome: outcomeRevoked, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.holder, officerID: a.issuer, revokerID: a.admin,
		},
		// Held by a soft-deleted holder: the verify path answers holder_disabled.
		// The issuer stays active on purpose — seed-chain signs with the
		// issuer's wallet and a deregistered signer would revert on chain.
		{
			name: "Sertifikat Kompetensi Alumni Nonaktif", workflow: workflowIssue,
			outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.deletedHolder, officerID: a.issuer,
		},

		// ── B · self-submission, then review ────────────────────────────────
		{
			name: "Sertifikat Keahlian Golang Backend", workflow: workflowSubmit,
			outcome: outcomePending, extract: extractUnextracted, metadata: metadataResolved,
			holderID: a.holder,
		},
		{
			name: "Workshop Natural Language Processing", workflow: workflowSubmit,
			outcome: outcomePending, extract: extractUnextracted, metadata: metadataStaged,
			holderID: a.holder,
		},
		{
			name: "Certified Kubernetes Administrator", workflow: workflowSubmit,
			outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolvedFromStaged,
			holderID: a.holder, officerID: a.issuer,
		},
		// Approved but extraction failed — the only ReExtract-eligible state.
		{
			name: "Sertifikat Pindai Kualitas Rendah", workflow: workflowSubmit,
			outcome: outcomeApproved, extract: extractFailed, metadata: metadataResolved,
			holderID: a.holder, officerID: a.admin,
		},
		// Rejected with its staged names never resolved, the way Reject leaves them.
		{
			name: "Pengajuan Ijazah Duplikat", workflow: workflowSubmit,
			outcome: outcomeRejected, extract: extractUnextracted, metadata: metadataStaged,
			holderID: a.holder, officerID: a.issuer,
		},
		{
			name: "Sertifikasi Kompetensi Dicabut", workflow: workflowSubmit,
			outcome: outcomeRevoked, extract: extractSucceeded, metadata: metadataResolved,
			holderID: a.holder, officerID: a.issuer, revokerID: a.issuer,
		},

		// ── C · file_hash reuse after a terminal state ──────────────────────
		// idx_credentials_file_hash_active is unique only WHERE revoked_at IS
		// NULL AND rejected_at IS NULL, so resubmitting a rejected document is
		// legal. Neither row is approved, so seed-chain skips both.
		{
			name: "Berkas Ganda - Pengajuan Ditolak", workflow: workflowSubmit,
			outcome: outcomeRejected, extract: extractUnextracted, metadata: metadataResolved,
			holderID: a.holder, officerID: a.issuer,
		},
		{
			name: "Berkas Ganda - Pengajuan Ulang", workflow: workflowSubmit,
			outcome: outcomePending, extract: extractUnextracted, metadata: metadataResolved,
			holderID: a.holder, fileTwin: "Berkas Ganda - Pengajuan Ditolak",
		},
	}
}

// seedCredFillerSpecs repeats the same workflow shapes across the randomized
// user pool, so list endpoints and pagination have volume to work with. Shapes
// only — every invariant worth asserting is covered by the fixed matrix.
func seedCredFillerSpecs(holders, issuers []domain.User) []seedCredSpec {
	names := []string{
		"Sarjana Teknik Informatika", "Magister Sistem Informasi", "Sertifikasi DevOps Engineer",
		"Certified Cloud Architect", "Pelatihan Fullstack Web", "Sertifikat Ethical Hacking",
		"Kursus Machine Learning Terapan", "Sertifikat Data Science Profesional",
		"Sertifikasi Scrum Master", "Pelatihan Cybersecurity Fundamentals",
		"Ijazah Sarjana Manajemen", "AWS Certified Solutions Architect",
		"Sertifikat BNSP Pemrogram Senior", "Sertifikat Pelatihan AI dan Robotika",
		"Ijazah Sarjana Akuntansi", "Sertifikasi Database Administrator",
		"Certified Network Associate", "Sertifikasi Blockchain Developer",
		"Pelatihan Cyber Threat Intelligence", "Transkrip Magister Teknik",
		"Dokumen Terenkripsi Eksternal", "Berkas Format Khusus OCR",
		"Sertifikat Tidak Terverifikasi", "Transkrip Nilai Tidak Lengkap",
		"Pengajuan Dokumen Non-Akademik", "Sertifikat Kadaluarsa Pemegang",
		"Surat Keputusan Gugur",
	}

	shapes := []seedCredSpec{
		{workflow: workflowIssue, outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolved},
		{workflow: workflowSubmit, outcome: outcomeApproved, extract: extractSucceeded, metadata: metadataResolvedFromStaged, expired: true},
		{workflow: workflowSubmit, outcome: outcomePending, extract: extractUnextracted, metadata: metadataStaged},
		{workflow: workflowIssue, outcome: outcomeApproved, extract: extractPending, metadata: metadataResolved, expired: true},
		{workflow: workflowSubmit, outcome: outcomePending, extract: extractUnextracted, metadata: metadataResolved},
		{workflow: workflowSubmit, outcome: outcomeRejected, extract: extractUnextracted, metadata: metadataStaged},
		{workflow: workflowIssue, outcome: outcomeRevoked, extract: extractSucceeded, metadata: metadataResolved},
		{workflow: workflowSubmit, outcome: outcomeApproved, extract: extractFailed, metadata: metadataResolved},
	}

	specs := make([]seedCredSpec, len(names))
	for i, name := range names {
		spec := shapes[i%len(shapes)]
		officer := issuers[i%len(issuers)]
		spec.name = name
		spec.holderID = holders[i%len(holders)].Id
		if spec.outcome != outcomePending {
			spec.officerID = officer.Id
		}
		if spec.outcome == outcomeRevoked {
			spec.revokerID = officer.Id
		}
		specs[i] = spec
	}
	return specs
}

// seedCredValidateActors fails loudly when the user seeder's fixed slots have
// drifted — the matrix is written against these five identities and silently
// falling back would reintroduce nonsense rows.
func seedCredValidateActors(byID map[string]domain.User, a seedCredActors) error {
	for _, want := range []struct {
		id      string
		label   string
		role    domain.Role
		deleted bool
	}{
		{id: a.superAdmin, label: "super admin", role: domain.RoleSuperAdmin},
		{id: a.admin, label: "admin", role: domain.RoleAdmin},
		{id: a.issuer, label: "issuer", role: domain.RoleIssuer},
		{id: a.holder, label: "holder", role: domain.RoleHolder},
		{id: a.deletedHolder, label: "soft-deleted holder", role: domain.RoleHolder, deleted: true},
	} {
		u, ok := byID[want.id]
		if !ok {
			return fmt.Errorf("credential seeder: %s user %s not seeded", want.label, want.id)
		}
		if u.Role != want.role {
			return fmt.Errorf("credential seeder: %s user %s has role %s", want.label, want.id, u.Role)
		}
		if want.deleted != (u.DeletedAt != nil) {
			return fmt.Errorf("credential seeder: %s user %s has unexpected deleted_at", want.label, want.id)
		}
	}
	return nil
}

// seedCredValidateSpecs rejects a spec that no code path could have produced,
// or that would break something downstream:
//
//   - an approved row with staged metadata violates a CHECK constraint;
//   - extraction only exists once a row has been approved;
//   - seed-chain signs each mint and revoke batch with the officer's wallet, and
//     CredentialRegistry reverts RoleBelowIssuerError for a signer below Issuer.
func seedCredValidateSpecs(specs []seedCredSpec, byID map[string]domain.User) error {
	isOfficer := func(id string) bool {
		u, ok := byID[id]
		return ok && u.DeletedAt == nil && u.Role.Rank() >= domain.RoleIssuer.Rank()
	}
	for _, spec := range specs {
		if _, ok := byID[spec.holderID]; !ok {
			return fmt.Errorf("credential seeder: %q has unknown holder %s", spec.name, spec.holderID)
		}
		approved := spec.outcome == outcomeApproved || spec.outcome == outcomeRevoked
		if approved && spec.metadata == metadataStaged {
			return fmt.Errorf("credential seeder: %q is approved with unresolved metadata", spec.name)
		}
		if !approved && spec.extract != extractUnextracted {
			return fmt.Errorf("credential seeder: %q extracts without ever being approved", spec.name)
		}
		if spec.outcome == outcomePending {
			if spec.officerID != "" {
				return fmt.Errorf("credential seeder: %q is pending but names a reviewer", spec.name)
			}
			continue
		}
		if !isOfficer(spec.officerID) {
			return fmt.Errorf("credential seeder: %q officer %s is not an active issuer+", spec.name, spec.officerID)
		}
		if spec.outcome == outcomeRevoked && !isOfficer(spec.revokerID) {
			return fmt.Errorf("credential seeder: %q revoker %s is not an active issuer+", spec.name, spec.revokerID)
		}
	}
	return nil
}

// seedCredLastTouched returns the timestamp of the row's most recent mutation,
// which is what updated_at would hold had the row been written through the API.
func seedCredLastTouched(c domain.Credential) time.Time {
	last := c.CreatedAt
	for _, t := range []*time.Time{
		c.ApprovedAt, c.RejectedAt, c.RevokedAt,
		c.ExtractEnqueuedAt, c.ExtractedAt, c.ExtractFailedAt,
	} {
		if t != nil && t.After(last) {
			last = *t
		}
	}
	return last
}

// seedEmbedding produces a synthetic L2-normalized vector of dimension dim.
func seedEmbedding(rng *rand.Rand, dim int) []float64 {
	vec := make([]float64, dim)
	var normSq float64
	for i := range vec {
		val := rng.NormFloat64()
		vec[i] = val
		normSq += val * val
	}
	norm := math.Sqrt(normSq)
	if norm == 0 {
		norm = 1
	}
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}
