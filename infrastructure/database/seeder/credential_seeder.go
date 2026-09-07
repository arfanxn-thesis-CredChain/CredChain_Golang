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

type credBucketType int

const (
	bucketPendingResolved credBucketType = iota
	bucketPendingUnresolved
	bucketApprovedExtracted
	bucketApprovedExtractPending
	bucketApprovedExtractFailed
	bucketRejected
	bucketRevoked
	bucketExpired
)

type seedCredSpec struct {
	nameBucket string
	bucket     credBucketType
}

func (s *CredentialSeeder) Seed(ctx context.Context) error {
	// Fetch taxonomies and users
	users, _, err := s.userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch users: %w", err)
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

	seed := hashToSeed("credchain-seed-credential")
	rng := rand.New(rand.NewSource(seed))

	// Specification of ~41 credentials covering all lifecycle states
	specs := []seedCredSpec{
		// 6 pending, fully resolved
		{"Sarjana Teknik Informatika", bucketPendingResolved},
		{"Magister Sistem Informasi", bucketPendingResolved},
		{"Sertifikasi DevOps Engineer", bucketPendingResolved},
		{"Certified Cloud Architect", bucketPendingResolved},
		{"Pelatihan Fullstack Web", bucketPendingResolved},
		{"Sertifikat Ethical Hacking", bucketPendingResolved},

		// 5 pending, unresolved metadata
		{"Kursus Machine Learning Terapan", bucketPendingUnresolved},
		{"Sertifikat Data Science Profesional", bucketPendingUnresolved},
		{"Sertifikasi Scrum Master", bucketPendingUnresolved},
		{"Pelatihan Cybersecurity Fundamentals", bucketPendingUnresolved},
		{"Workshop Natural Language Processing", bucketPendingUnresolved},

		// 12 approved, extract succeeded (has Mongo extraction)
		{"Ijazah Sarjana Komputer", bucketApprovedExtracted},
		{"Ijazah Sarjana Manajemen", bucketApprovedExtracted},
		{"Transkrip Akademik S1", bucketApprovedExtracted},
		{"Sertifikat Keahlian Golang Backend", bucketApprovedExtracted},
		{"Certified Kubernetes Administrator", bucketApprovedExtracted},
		{"AWS Certified Solutions Architect", bucketApprovedExtracted},
		{"Sertifikat BNSP Pemrogram Senior", bucketApprovedExtracted},
		{"Sertifikat Pelatihan AI dan Robotika", bucketApprovedExtracted},
		{"SKPI Sarjana Sistem Informasi", bucketApprovedExtracted},
		{"Ijazah Sarjana Akuntansi", bucketApprovedExtracted},
		{"Sertifikasi Database Administrator", bucketApprovedExtracted},
		{"Certified Network Associate", bucketApprovedExtracted},

		// 3 approved, extract pending
		{"Sertifikasi Blockchain Developer", bucketApprovedExtractPending},
		{"Pelatihan Cyber Threat Intelligence", bucketApprovedExtractPending},
		{"Transkrip Magister Teknik", bucketApprovedExtractPending},

		// 4 approved, extract failed
		{"Dokumen Terenkripsi Eksternal", bucketApprovedExtractFailed},
		{"Sertifikat Pindai Kualitas Rendah", bucketApprovedExtractFailed},
		{"Berkas Format Khusus OCR", bucketApprovedExtractFailed},
		{"Dokumen Ekstraksi Bermasalah", bucketApprovedExtractFailed},

		// 5 rejected
		{"Pengajuan Ijazah Duplikat", bucketRejected},
		{"Sertifikat Tidak Terverifikasi", bucketRejected},
		{"Transkrip Nilai Tidak Lengkap", bucketRejected},
		{"Pengajuan Dokumen Non-Akademik", bucketRejected},
		{"Sertifikat Kedaluwarsa", bucketRejected},

		// 4 revoked
		{"Ijazah Dibatalkan Rektorat", bucketRevoked},
		{"Sertifikasi Kompetensi Dicabut", bucketRevoked},
		{"Sertifikat Kadaluarsa Pemegang", bucketRevoked},
		{"Surat Keputusan Gugur", bucketRevoked},

		// 2 expired
		{"Sertifikasi Lisensi 2024", bucketExpired},
		{"Sertifikat Kepatuhan Tahunan", bucketExpired},
	}

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	credentials := make([]domain.Credential, len(specs))
	var extractions []domain.CredentialExtraction
	var compCredLinks []domain.CompetencyCredential

	for i, spec := range specs {
		credID := deterministicULID(uint32(1000 + i))
		holder := holders[i%len(holders)]
		issuer := issuers[i%len(issuers)]
		credType := activeTypes[i%len(activeTypes)]
		org := activeOrgs[i%len(activeOrgs)]

		issuedAt := baseTime.Add(time.Duration(rng.Int63n(180)) * 24 * time.Hour)
		number := fmt.Sprintf("CRED/%d/%04d", issuedAt.Year(), i+1)

		holderName := "Unknown"
		if holder.Name != nil {
			holderName = *holder.Name
		}

		lines := []string{
			fmt.Sprintf("Identifier: %s", credID),
			fmt.Sprintf("Holder: %s (ID: %s)", holderName, holder.Id),
			fmt.Sprintf("Number: %s", number),
			fmt.Sprintf("Issuer Organization: %s", org.Name),
			fmt.Sprintf("Type: %s", credType.Name),
			fmt.Sprintf("Issued Date: %s", issuedAt.Format("2006-01-02")),
			fmt.Sprintf("Seed-Marker: seed-%d", i+1),
		}

		plainBytes := seedPDFBytes(spec.nameBucket, lines)
		fileHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(plainBytes))

		encBytes, encErr := infraCrypto.Encrypt(plainBytes, []byte(*s.cfg.FileEncryptionKey))
		if encErr != nil {
			return fmt.Errorf("credential seeder: encrypt file: %w", encErr)
		}

		filename := fmt.Sprintf("%s.pdf", credID)
		filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
		if _, saveErr := s.storage.SaveBytes([]byte(encBytes), filePath); saveErr != nil {
			return fmt.Errorf("credential seeder: save file %s: %w", filename, saveErr)
		}

		c := domain.Credential{
			ID:              credID,
			HolderUserID:    holder.Id,
			SubmitterUserID: holder.Id,
			IssuerUserID:    issuer.Id,
			Number:          &number,
			Name:            spec.nameBucket,
			FileHash:        fileHash,
			FileURI:         lo.ToPtr(filename),
			IssuedAt:        issuedAt,
			CreatedAt:       issuedAt.Add(-2 * 24 * time.Hour),
		}

		// Bucket specific states & timestamps
		switch spec.bucket {
		case bucketPendingResolved:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			if len(activeComps) > 0 {
				c.SubmittedCompetencies = domain.SubmittedCompetencies{
					{Name: activeComps[i%len(activeComps)].Name, ResolvedID: &activeComps[i%len(activeComps)].Id},
				}
			}

		case bucketPendingUnresolved:
			stageTypeName := "Unstaged Type " + credType.Name
			stageOrgName := "Unstaged Org " + org.Name
			c.SubmittedTypeName = &stageTypeName
			c.SubmittedIssuerOrganizationName = &stageOrgName
			c.SubmittedCompetencies = domain.SubmittedCompetencies{
				{Name: "Custom Unstaged Competency"},
			}

		case bucketApprovedExtracted:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			approvedAt := issuedAt.Add(time.Hour)
			c.ApprovedAt = &approvedAt
			c.ApproverUserID = &issuer.Id

			enqueuedAt := approvedAt.Add(time.Minute)
			extractedAt := enqueuedAt.Add(2 * time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractedAt = &extractedAt

			// Assign competencies
			if len(activeComps) > 0 {
				comp := activeComps[i%len(activeComps)]
				compCredLinks = append(compCredLinks, domain.CompetencyCredential{
					CompetencyId: comp.Id,
					CredentialId: credID,
				})
			}

			// Add Mongo extraction doc
			docText := fmt.Sprintf("%s\n%s", spec.nameBucket, strings.Join(lines, "\n"))
			extractions = append(extractions, domain.CredentialExtraction{
				CredentialID: credID,
				FileHash:     fileHash,
				Text:         docText,
				IDs: []domain.CredentialExtractedID{
					{Type: "number", Value: number},
					{Type: "holder_id", Value: holder.Id},
				},
				Embedding: seedEmbedding(rng, 768),
				CreatedAt: extractedAt,
				UpdatedAt: extractedAt,
			})

		case bucketApprovedExtractPending:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			approvedAt := issuedAt.Add(time.Hour)
			c.ApprovedAt = &approvedAt
			c.ApproverUserID = &issuer.Id

			enqueuedAt := approvedAt.Add(time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt

		case bucketApprovedExtractFailed:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			approvedAt := issuedAt.Add(time.Hour)
			c.ApprovedAt = &approvedAt
			c.ApproverUserID = &issuer.Id

			enqueuedAt := approvedAt.Add(time.Minute)
			failedAt := enqueuedAt.Add(time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractFailedAt = &failedAt
			c.ExtractError = lo.ToPtr("document text extraction failed: unreadable or corrupted OCR stream")

		case bucketRejected:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			rejectedAt := issuedAt.Add(time.Hour)
			c.RejectedAt = &rejectedAt
			c.RejecterUserID = &issuer.Id
			c.RejectionReason = lo.ToPtr("Dokumen tidak sesuai dengan standar verifikasi atau data tidak valid")

		case bucketRevoked:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			approvedAt := issuedAt.Add(time.Hour)
			c.ApprovedAt = &approvedAt
			c.ApproverUserID = &issuer.Id

			enqueuedAt := approvedAt.Add(time.Minute)
			extractedAt := enqueuedAt.Add(time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractedAt = &extractedAt

			revokedAt := approvedAt.Add(30 * 24 * time.Hour)
			c.RevokedAt = &revokedAt
			c.RevokerUserID = &issuer.Id

		case bucketExpired:
			c.TypeID = &credType.Id
			c.IssuerOrganizationID = &org.Id
			approvedAt := issuedAt.Add(time.Hour)
			c.ApprovedAt = &approvedAt
			c.ApproverUserID = &issuer.Id

			enqueuedAt := approvedAt.Add(time.Minute)
			extractedAt := enqueuedAt.Add(time.Minute)
			c.ExtractEnqueuedAt = &enqueuedAt
			c.ExtractedAt = &extractedAt

			pastExpiry := issuedAt.Add(-24 * time.Hour)
			c.ExpiresAt = &pastExpiry
		}

		credentials[i] = c
	}

	// Store PostgreSQL credentials
	if _, err := s.credRepo.Store(ctx, credentials...); err != nil {
		return fmt.Errorf("credential seeder: store credentials: %w", err)
	}

	// Store competency_credential joins
	if len(compCredLinks) > 0 {
		if _, err := s.compCredRepo.Store(ctx, compCredLinks...); err != nil {
			return fmt.Errorf("credential seeder: store competency credentials: %w", err)
		}
	}

	// Store Mongo extractions (succeeded rows only)
	if s.extractRepo != nil {
		for _, ext := range extractions {
			if err := s.extractRepo.Store(ctx, ext); err != nil {
				return fmt.Errorf("credential seeder: store extraction: %w", err)
			}
		}
	}

	return nil
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
