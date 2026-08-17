package credential

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/chain/contracts"
	infraCrypto "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/jobs"
	"CredChain_Golang/infrastructure/storage"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Interface ─────────────────────────────────────────────────────────────

// CredentialService is the business-logic layer for credential operations.
// It orchestrates the GORM repository, the on-chain CredentialRegistry
// (via chain.RegistryService), the Python AI service (via pyai.PythonAIClient),
// local file storage, and the asynchronous extract worker queue.
type CredentialService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error)
	Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error)
	SelfPaginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error)
	SelfFind(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error)
	Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error)
	Submit(ctx context.Context, items []CredentialSubmission) ([]domain.Credential, error)
	Approve(ctx context.Context, ids ...string) ([]domain.Credential, error)
	Reject(ctx context.Context, rejections []CredentialRejection) ([]domain.Credential, error)
	Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error)
	Revoke(ctx context.Context, ids ...string) ([]domain.Credential, error)
	Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error)
	ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error)
	DownloadFile(ctx context.Context, id string) (data []byte, filename string, mimeType string, err error)
}

// CredentialIssuance is the service-layer input for one credential issuance. File
// bytes are already in memory (the handler reads multipart upload bytes
// before calling the service).
type CredentialIssuance struct {
	HolderUserID         string
	TypeID               string
	IssuerOrganizationID string
	Number               *string
	IssuedAt             *time.Time
	ExpiresAt            *time.Time
	Name                 string
	Meta                 map[string]any
	CompetencyIDs        []string
	Filename             string
	MIMEType             string
	FileBytes            []byte
}

// CredentialSubmission is the service-layer input for one self-submitted
// credential. The submitter is always the holder (auth user), so no holder
// field exists. File bytes are already in memory (the handler reads multipart
// upload bytes before calling the service).
type CredentialSubmission struct {
	Name                 string
	TypeID               string
	IssuerOrganizationID string
	Number               *string
	IssuedAt             *time.Time
	ExpiresAt            *time.Time
	CompetencyIDs        []string
	Meta                 map[string]any
	Filename             string
	MIMEType             string
	FileBytes            []byte
}

// CredentialRejection is the service-layer input for one rejected credential.
type CredentialRejection struct {
	ID     string
	Reason string
}

// ── Implementation struct & constructor ───────────────────────────────────

type credentialService struct {
	repo             domain.CredentialRepository
	uow              domain.UnitOfWork
	cfg              *config.Config
	registryService  chain.RegistryService
	aiClient         pyai.PythonAIClient
	extractionRepo   domain.CredentialExtractionRepository
	verificationRepo domain.CredentialVerificationRepository
	storage          *storage.Storage
	policy           CredentialPolicy
	userRepo         domain.UserRepository
	typeRepo         domain.CredentialTypeRepository
	orgRepo          domain.CredentialIssuerOrganizationRepository
	competencyRepo   domain.CompetencyRepository
	logger           *zap.Logger
	enqueuer         jobs.Enqueuer
}

type CredentialServiceParams struct {
	fx.In
	Repo             domain.CredentialRepository
	UoW              domain.UnitOfWork
	Config           *config.Config
	RegistryService  chain.RegistryService
	AIClient         pyai.PythonAIClient
	ExtractionRepo   domain.CredentialExtractionRepository
	VerificationRepo domain.CredentialVerificationRepository
	Storage          *storage.Storage
	Policy           CredentialPolicy
	UserRepo         domain.UserRepository
	TypeRepo         domain.CredentialTypeRepository
	OrgRepo          domain.CredentialIssuerOrganizationRepository
	CompetencyRepo   domain.CompetencyRepository
	Logger           *zap.Logger
	Enqueuer         jobs.Enqueuer
}

// NewCredentialService is the exported factory for FX injection.
func NewCredentialService(p CredentialServiceParams) CredentialService {
	return &credentialService{
		repo:             p.Repo,
		uow:              p.UoW,
		cfg:              p.Config,
		registryService:  p.RegistryService,
		aiClient:         p.AIClient,
		extractionRepo:   p.ExtractionRepo,
		verificationRepo: p.VerificationRepo,
		storage:          p.Storage,
		policy:           p.Policy,
		userRepo:         p.UserRepo,
		typeRepo:         p.TypeRepo,
		orgRepo:          p.OrgRepo,
		competencyRepo:   p.CompetencyRepo,
		logger:           p.Logger,
		enqueuer:         p.Enqueuer,
	}
}

// ── Paginate ──────────────────────────────────────────────────────────────

// Paginate returns a paginated list of credentials, optionally including
// holder/issuer/revoker user expansions via query.Includes.
func (s *credentialService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	return s.repo.Get(ctx, query)
}

// ── Find ──────────────────────────────────────────────────────────────────

// Find retrieves a single credential by ID with optional user expansions
// from query.Includes.
func (s *credentialService) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	c, err := s.repo.Find(ctx, id, query)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCredentialFetchNotFound,
				domain.WithMetadata("credential_id", id))
		}
		return nil, err
	}
	return c, nil
}

// ── Self (Holder) ───────────────────────────────────────────────────────────

// SelfPaginate returns a paginated list of credentials owned by the
// authenticated user. It injects a holder_user_id filter scoped to the auth
// user before delegating to the repository (single query; no N+1).
func (s *credentialService) SelfPaginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	authUser := httpContext.MustGetUser(ctx)
	if query == nil {
		query = &domainQuery.Query{}
	}
	query.Filters = append(query.Filters,
		domainQuery.NewFilter("holder_user_id", domainQuery.OperatorEqual, authUser.Id))
	return s.repo.Get(ctx, query)
}

// SelfFind retrieves a single credential by ID, scoped to the authenticated
// user. Returns CodeCredentialFetchNotFound (404) when the credential does not
// exist OR when it is owned by another user — ownership mismatch is reported
// as not-found so credential IDs are not leaked across holders.
func (s *credentialService) SelfFind(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)
	c, err := s.repo.Find(ctx, id, query)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCredentialFetchNotFound,
				domain.WithMetadata("credential_id", id))
		}
		return nil, err
	}
	if c.HolderUserID != authUser.Id {
		return nil, domain.NewError(domain.CodeCredentialFetchNotFound,
			domain.WithMetadata("credential_id", id))
	}
	return c, nil
}

// ── Issue ─────────────────────────────────────────────────────────────────

// issueValidate performs batch input-driven validation that belongs at the
// service layer: holder existence, on-chain duplicate file hash, in-batch
// duplicate hash, credential type (exists + active), issuer organization
// existence, number uniqueness within the org, and competency existence.
// Returns validation.Errors keyed by "credentials.N.field".
// Server-side failures (encryption, storage, chain mint) are NOT collected
// here — those remain *domain.Error from the Issue orchestrator.
func (s *credentialService) issueValidate(
	ctx context.Context,
	items []CredentialIssuance,
	holderSet map[string]bool,
	statuses []contracts.CredentialRegistryCredentialHashStatus,
) validation.Errors {
	verrs := validation.Errors{}

	hashes := make([]string, len(items))
	for i, it := range items {
		hash := ethCrypto.Keccak256(it.FileBytes)
		hashes[i] = "0x" + hex.EncodeToString(hash)
	}

	onChainActive := map[string]bool{}
	for i, st := range statuses {
		if st.Status == 1 {
			onChainActive[hashes[i]] = true
		}
	}

	// Competency existence: one FindByIds across the whole batch (NO-N+1).
	var allCompIDs []string
	for _, it := range items {
		allCompIDs = append(allCompIDs, it.CompetencyIDs...)
	}
	allCompIDs = lo.Uniq(allCompIDs)
	existingComp := map[string]bool{}
	if len(allCompIDs) > 0 {
		if found, err := s.competencyRepo.FindByIds(ctx, allCompIDs...); err == nil {
			for _, c := range found {
				existingComp[c.Id] = true
			}
		}
	}

	// Number uniqueness: one Get per DISTINCT org (bounded by the batch's org
	// count, never per input item — NO-N+1). Each org query uses a single
	// number IN (...) so collisions are attributed back to the right items.
	// The duplicate set is keyed by org+number (composite) so an existing
	// number in org A never flags an org-B item carrying the same number.
	numbersByOrg := map[string][]string{}
	for _, it := range items {
		if it.Number != nil && *it.Number != "" {
			numbersByOrg[it.IssuerOrganizationID] = append(numbersByOrg[it.IssuerOrganizationID], *it.Number)
		}
	}
	duplicateNumbers := map[string]bool{}
	for org, numbers := range numbersByOrg {
		dupQuery := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				domainQuery.NewFilter("issuer_organization_id", domainQuery.OperatorEqual, org),
				domainQuery.NewFilter("number", domainQuery.OperatorIn, numbers...),
			},
		}
		rows, _, err := s.repo.Get(ctx, dupQuery)
		if err != nil {
			continue
		}
		for _, r := range rows {
			if r.Number != nil {
				duplicateNumbers[org+"\x00"+*r.Number] = true
			}
		}
	}

	seenHash := map[string]bool{}
	for i, it := range items {
		prefix := fmt.Sprintf("credentials.%d", i)

		if !holderSet[it.HolderUserID] {
			verrs[prefix+".holder_user_id"] = validation.NewError(
				"validation_issue_holder_not_found",
				"holder not found",
			)
			continue
		}

		if onChainActive[hashes[i]] || seenHash[hashes[i]] {
			verrs[prefix+".file"] = validation.NewError(
				"validation_issue_duplicate_file_hash", "duplicate file hash",
			)
			continue
		}
		seenHash[hashes[i]] = true

		if t, err := s.typeRepo.Find(ctx, it.TypeID); err != nil || t == nil {
			verrs[prefix+".type_id"] = validation.NewError(
				"validation_issue_type_not_found", "credential type not found",
			)
		} else if !t.Active {
			verrs[prefix+".type_id"] = validation.NewError(
				"validation_issue_type_inactive", "credential type inactive",
			)
		}

		if _, err := s.orgRepo.Find(ctx, it.IssuerOrganizationID); err != nil {
			verrs[prefix+".issuer_organization_id"] = validation.NewError(
				"validation_issue_org_not_found", "issuer organization not found",
			)
		}

		if it.Number != nil && *it.Number != "" && duplicateNumbers[it.IssuerOrganizationID+"\x00"+*it.Number] {
			verrs[prefix+".number"] = validation.NewError(
				"validation_issue_number_duplicate", "number already in use",
			)
		}

		for _, cid := range it.CompetencyIDs {
			if !existingComp[cid] {
				verrs[prefix+".competency_ids"] = validation.NewError(
					"validation_issue_competency_not_found", "competency not found",
				)
				break
			}
		}
	}

	return verrs
}

// issuePrepareCredentials encrypts files, persists them to storage, and builds
// domain.Credential entities with extract_status=pending. Direct issuance is
// approved at creation: the submitting issuer stamps themselves as approver so
// the row never sits in pending-review. Returns *domain.Error on encryption or
// storage failure (caller cleans up orphan files).
func (s *credentialService) issuePrepareCredentials(
	ctx context.Context,
	items []CredentialIssuance,
) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)

	creds := make([]domain.Credential, len(items))
	for i, it := range items {
		issuedAt := time.Now()
		if it.IssuedAt != nil {
			issuedAt = *it.IssuedAt
		}
		ext := strings.ToLower(filepath.Ext(it.Filename))
		if ext == "" {
			ext = ".bin"
		}
		encryptedHex, encErr := infraCrypto.Encrypt(it.FileBytes, []byte(*s.cfg.FileEncryptionKey))
		if encErr != nil {
			return nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(encErr))
		}
		filename := ulid.Make().String() + ext
		filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
		if _, err := s.storage.SaveBytes([]byte(encryptedHex), filePath); err != nil {
			return nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(err))
		}
		hash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
		creds[i] = domain.Credential{
			ID:                   ulid.Make().String(),
			HolderUserID:         it.HolderUserID,
			SubmitterUserID:      authUser.Id,
			IssuerUserID:         authUser.Id,
			IssuerOrganizationID: it.IssuerOrganizationID,
			TypeID:               it.TypeID,
			Number:               it.Number,
			Name:                 it.Name,
			Meta:                 it.Meta,
			FileHash:             hash,
			FileURI:              &filename,
			ExtractStatus:        domain.ExtractStatusPending,
			IssuedAt:             issuedAt,
			ExpiresAt:            it.ExpiresAt,
			ApproverUserID:       &authUser.Id,
			ApprovedAt:           &issuedAt,
		}
	}
	return creds, nil
}

func (s *credentialService) cleanupOrphanCredentialFiles(creds []domain.Credential) {
	paths := make([]string, 0, len(creds))
	for _, c := range creds {
		if c.FileURI != nil {
			paths = append(paths, filepath.Join(*s.cfg.CredentialFileStoragePath, *c.FileURI))
		}
	}
	s.cleanupOrphanFiles(paths)
}

// credentialIssuedAtToChain converts the credential's issue date to the
// on-chain seconds value. The issue flow stamps IssuedAt explicitly, so the
// same value reaches the database and the chain.
func credentialIssuedAtToChain(issuedAt time.Time) uint64 {
	if issuedAt.IsZero() {
		return 0
	}
	return uint64(issuedAt.Unix())
}

// credentialExpiresAtToChain maps the credential's DB expiry to the on-chain
// seconds value. NULL in the database and 0 on chain both mean no expiry
// (FINAL: no sentinel value).
func credentialExpiresAtToChain(expiresAt *time.Time) uint64 {
	if expiresAt == nil {
		return 0
	}
	return uint64(expiresAt.Unix())
}

// issueCommit runs the UoW transaction: Store credentials, mint on-chain,
// update token IDs, persist competency links, and enqueue River extraction
// jobs. Chain failure or enqueue failure rolls back the entire transaction.
// competencyIDsByCredID maps prepared credential IDs to their requested
// competency IDs (empty entries are skipped).
func (s *credentialService) issueCommit(
	ctx context.Context,
	authWallet domain.Wallet,
	creds []domain.Credential,
	competencyIDsByCredID map[string][]string,
) ([]domain.Credential, error) {
	holderIDs := lo.Map(creds, func(c domain.Credential, _ int) string { return c.HolderUserID })
	holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
	if err != nil {
		return nil, err
	}
	holderByID := lo.SliceToMap(holders, func(h domain.User) (string, domain.User) { return h.Id, h })

	if err := s.policy.IssuePostFetch(ctx, creds, holders); err != nil {
		return nil, err
	}

	var committed []domain.Credential
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		stored, err := uow.Credential().Store(ctx, creds...)
		if err != nil {
			return err
		}
		for _, c := range stored {
			if c.FileURI == nil {
				return domain.NewError(domain.CodeCredentialIssueStorageFailed,
					domain.WithMetadata("credential_id", c.ID))
			}
		}
		if err := s.mintCredentials(ctx, uow, authWallet, stored, holderByID,
			domain.CodeCredentialIssueBlockchainSyncFailed); err != nil {
			return err
		}
		for _, c := range stored {
			compIDs := competencyIDsByCredID[c.ID]
			if len(compIDs) == 0 {
				continue
			}
			links := make([]domain.CompetencyCredential, len(compIDs))
			for j, compID := range compIDs {
				links[j] = domain.CompetencyCredential{CompetencyId: compID, CredentialId: c.ID}
			}
			if _, err := uow.CompetencyCredential().Store(ctx, links...); err != nil {
				return err
			}
		}
		committed = stored
		return nil
	})
	return committed, err
}

// Issue performs the synchronous batch credential issuance flow.
//
// Architecture (Option A — sync chain, async embeddings):
//  1. IssuePreFetch policy (signer is Issuer+)
//  2. issueValidate — input-driven checks (holders, duplicate hashes)
//  3. issuePrepareCredentials — encrypt, store, build entities
//  4. issueCommit — UoW: Store → chain mint → update token IDs → enqueue
//
// All-or-nothing: any validation failure returns validation.Errors;
// any server-side failure rolls back the entire batch.
func (s *credentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)

	holderIDs := lo.Map(items, func(it CredentialIssuance, _ int) string { return it.HolderUserID })
	holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
	if err != nil {
		return nil, err
	}
	holderSet := lo.SliceToMap(holders, func(h domain.User) (string, bool) { return h.Id, true })

	hashBytes := make([][32]byte, len(items))
	for i, it := range items {
		rawHash := ethCrypto.Keccak256(it.FileBytes)
		hashStr := "0x" + hex.EncodeToString(rawHash)
		copy(hashBytes[i][:], ethCrypto.Keccak256([]byte(hashStr)))
	}
	statuses, err := s.registryService.GetCredentialHashStatuses(ctx, hashBytes)
	if err != nil {
		return nil, domain.NewError(domain.CodeCredentialIssueBlockchainSyncFailed, domain.WithError(err))
	}

	if verrs := s.issueValidate(ctx, items, holderSet, statuses); len(verrs) > 0 {
		return nil, verrs
	}

	creds, err := s.issuePrepareCredentials(ctx, items)
	if err != nil {
		s.cleanupOrphanCredentialFiles(creds)
		return nil, err
	}

	competencyIDsByCredID := make(map[string][]string, len(creds))
	for i, c := range creds {
		if len(items[i].CompetencyIDs) > 0 {
			competencyIDsByCredID[c.ID] = items[i].CompetencyIDs
		}
	}

	authWallet := domain.WalletFromUser(*authUser)
	committed, err := s.issueCommit(ctx, authWallet, creds, competencyIDsByCredID)
	if err != nil {
		s.cleanupOrphanCredentialFiles(creds)
		return nil, err
	}

	return committed, nil
}

// ── Submit (self-submission) ──────────────────────────────────────────────

// Submit stores holder-submitted credentials as pending rows with
// ExtractStatus unextracted and no mint (no on-chain interaction). Holder
// and submitter are both the authenticated user. Submitted rows are rejected
// at approval time or activated by an Issuer via the review flow (step 4).
func (s *credentialService) Submit(ctx context.Context, items []CredentialSubmission) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)

	verrs, err := s.submitValidate(ctx, items)
	if err != nil {
		return nil, err
	}
	if len(verrs) > 0 {
		return nil, verrs
	}

	creds, err := s.submitPrepareCredentials(ctx, authUser.Id, items)
	if err != nil {
		s.cleanupOrphanCredentialFiles(creds)
		return nil, err
	}

	var committed []domain.Credential
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		stored, err := uow.Credential().Store(ctx, creds...)
		if err != nil {
			return err
		}
		for i, c := range stored {
			if len(items[i].CompetencyIDs) == 0 {
				continue
			}
			links := make([]domain.CompetencyCredential, len(items[i].CompetencyIDs))
			for j, compID := range items[i].CompetencyIDs {
				links[j] = domain.CompetencyCredential{CompetencyId: compID, CredentialId: c.ID}
			}
			if _, err := uow.CompetencyCredential().Store(ctx, links...); err != nil {
				return err
			}
		}
		committed = stored
		return nil
	})
	if err != nil {
		s.cleanupOrphanCredentialFiles(creds)
		return nil, err
	}
	return committed, nil
}

// submitValidate performs batch input-driven validation for self-submission:
// credential type (exists + active), issuer organization existence, number
// uniqueness within the org (same batched per-distinct-org approach as
// issueValidate), competency existence (one FindByIds), and active-duplicate
// file-hash detection (one CountActiveByFileHashes per batch). The DB partial
// unique index is the final authority for concurrent duplicates. Returns
// validation.Errors keyed by "credentials.N.field"; the count error is a
// server-side failure returned as a plain error.
func (s *credentialService) submitValidate(
	ctx context.Context,
	items []CredentialSubmission,
) (validation.Errors, error) {
	verrs := validation.Errors{}

	// Active-duplicate detection: one count per batch (NO-N+1).
	hashes := make([]string, len(items))
	for i, it := range items {
		hashes[i] = "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
	}
	n, err := s.repo.CountActiveByFileHashes(ctx, hashes...)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		verrs["credentials.0.file"] = validation.NewError("validation_issue_duplicate_file_hash", "")
		return verrs, nil
	}

	// Competency existence: one FindByIds across the whole batch (NO-N+1).
	var allCompIDs []string
	for _, it := range items {
		allCompIDs = append(allCompIDs, it.CompetencyIDs...)
	}
	allCompIDs = lo.Uniq(allCompIDs)
	existingComp := map[string]bool{}
	if len(allCompIDs) > 0 {
		if found, err := s.competencyRepo.FindByIds(ctx, allCompIDs...); err == nil {
			for _, c := range found {
				existingComp[c.Id] = true
			}
		}
	}

	// Number uniqueness: one Get per DISTINCT org (bounded by the batch's org
	// count, never per input item — NO-N+1). Each org query uses a single
	// number IN (...) so collisions are attributed back to the right items.
	numbersByOrg := map[string][]string{}
	for _, it := range items {
		if it.Number != nil && *it.Number != "" {
			numbersByOrg[it.IssuerOrganizationID] = append(numbersByOrg[it.IssuerOrganizationID], *it.Number)
		}
	}
	duplicateNumbers := map[string]bool{}
	for org, numbers := range numbersByOrg {
		dupQuery := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				domainQuery.NewFilter("issuer_organization_id", domainQuery.OperatorEqual, org),
				domainQuery.NewFilter("number", domainQuery.OperatorIn, numbers...),
			},
		}
		rows, _, err := s.repo.Get(ctx, dupQuery)
		if err != nil {
			continue
		}
		for _, r := range rows {
			if r.Number != nil {
				duplicateNumbers[org+"\x00"+*r.Number] = true
			}
		}
	}

	for i, it := range items {
		prefix := fmt.Sprintf("credentials.%d", i)

		if t, err := s.typeRepo.Find(ctx, it.TypeID); err != nil || t == nil {
			verrs[prefix+".type_id"] = validation.NewError(
				"validation_issue_type_not_found", "credential type not found",
			)
		} else if !t.Active {
			verrs[prefix+".type_id"] = validation.NewError(
				"validation_issue_type_inactive", "credential type inactive",
			)
		}

		if _, err := s.orgRepo.Find(ctx, it.IssuerOrganizationID); err != nil {
			verrs[prefix+".issuer_organization_id"] = validation.NewError(
				"validation_issue_org_not_found", "issuer organization not found",
			)
		}

		if it.Number != nil && *it.Number != "" && duplicateNumbers[it.IssuerOrganizationID+"\x00"+*it.Number] {
			verrs[prefix+".number"] = validation.NewError(
				"validation_issue_number_duplicate", "number already in use",
			)
		}

		for _, cid := range it.CompetencyIDs {
			if !existingComp[cid] {
				verrs[prefix+".competency_ids"] = validation.NewError(
					"validation_issue_competency_not_found", "competency not found",
				)
				break
			}
		}
	}

	return verrs, nil
}

// submitPrepareCredentials encrypts files, persists them to storage, and
// builds domain.Credential entities with ExtractStatus unextracted, holder
// and submitter both equal to the auth user, and NO approver fields (the row
// enters the pending-review pool). IssuerUserID is the auth user as a D5
// placeholder (the submitting holder is not an issuer; the real issuer is
// stamped at approval). Returns *domain.Error on encryption or storage
// failure (caller cleans up orphan files).
func (s *credentialService) submitPrepareCredentials(
	ctx context.Context,
	holderID string,
	items []CredentialSubmission,
) ([]domain.Credential, error) {
	creds := make([]domain.Credential, len(items))
	for i, it := range items {
		ext := strings.ToLower(filepath.Ext(it.Filename))
		if ext == "" {
			ext = ".bin"
		}
		encryptedHex, encErr := infraCrypto.Encrypt(it.FileBytes, []byte(*s.cfg.FileEncryptionKey))
		if encErr != nil {
			return nil, domain.NewError(domain.CodeCredentialSubmitStorageFailed,
				domain.WithError(encErr))
		}
		filename := ulid.Make().String() + ext
		filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
		if _, err := s.storage.SaveBytes([]byte(encryptedHex), filePath); err != nil {
			return nil, domain.NewError(domain.CodeCredentialSubmitStorageFailed,
				domain.WithError(err))
		}
		hash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
		creds[i] = domain.Credential{
			ID:                   ulid.Make().String(),
			HolderUserID:         holderID,
			SubmitterUserID:      holderID,
			IssuerUserID:         holderID,
			IssuerOrganizationID: it.IssuerOrganizationID,
			TypeID:               it.TypeID,
			Number:               it.Number,
			Name:                 it.Name,
			Meta:                 it.Meta,
			FileHash:             hash,
			FileURI:              &filename,
			ExtractStatus:        domain.ExtractStatusUnextracted,
			IssuedAt:             *it.IssuedAt,
			ExpiresAt:            it.ExpiresAt,
		}
	}
	return creds, nil
}

// ── Update ────────────────────────────────────────────────────────────────

// Update batch-updates credentials. PENDING-ONLY (D12): every target must be
// pending (submitted, not yet minted); approved/revoked/rejected rows are
// fully immutable — revoke + reissue is the only fix. Stamps and the file
// are never editable.
func (s *credentialService) Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	if len(credentials) == 0 {
		return []domain.Credential{}, nil
	}
	ids := lo.Map(credentials, func(c domain.Credential, _ int) string { return c.ID })
	targets, err := s.repo.FindByIds(ctx, ids, nil)
	if err != nil {
		return nil, err
	}
	targetByID := lo.SliceToMap(targets, func(c domain.Credential) (string, domain.Credential) { return c.ID, c })
	targetIDs := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
	if missing, _ := lo.Difference(ids, targetIDs); len(missing) > 0 {
		return nil, domain.NewError(domain.CodeCredentialUpdateNotFound, domain.WithMetadata("credential_ids", missing))
	}

	for i := range credentials {
		in := &credentials[i]
		target := targetByID[in.ID]
		if target.LifecycleStatus() != domain.CredentialLifecycleStatusPending {
			return nil, domain.NewError(domain.CodeCredentialUpdateNotPending,
				domain.WithMetadata("credential_ids", []string{in.ID}))
		}
		if in.TypeID != "" && in.TypeID != target.TypeID {
			t, err := s.typeRepo.Find(ctx, in.TypeID)
			if err != nil || !t.Active {
				return nil, domain.NewError(domain.CodeCredentialIssueTypeInactive,
					domain.WithMetadata("credential_ids", []string{in.ID}))
			}
		}
		if in.IssuerOrganizationID != "" && in.IssuerOrganizationID != target.IssuerOrganizationID {
			if _, err := s.orgRepo.Find(ctx, in.IssuerOrganizationID); err != nil {
				return nil, domain.NewError(domain.CodeCredentialIssueOrganizationNotFound,
					domain.WithMetadata("credential_ids", []string{in.ID}))
			}
		}
	}

	if err := s.updateValidateNumberUniqueness(ctx, credentials, targetByID); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, credentials...)
}

// updateValidateNumberUniqueness checks that every number change is unique per
// (issuer_organization_id, number). One Get per DISTINCT org (bounded by the
// batch's org count, never per input item — NO-N+1, per the established B2
// pattern). Collisions are attributed back to the offending items via an
// org+"\x00"+number composite key so a number in org A never flags an org-B
// item carrying the same number.
func (s *credentialService) updateValidateNumberUniqueness(
	ctx context.Context,
	credentials []domain.Credential,
	targetByID map[string]domain.Credential,
) error {
	numbersByOrg := map[string][]string{}
	for i := range credentials {
		in := &credentials[i]
		target := targetByID[in.ID]
		if in.Number == nil || *in.Number == derefString(target.Number) {
			continue
		}
		orgID := target.IssuerOrganizationID
		if in.IssuerOrganizationID != "" {
			orgID = in.IssuerOrganizationID
		}
		numbersByOrg[orgID] = append(numbersByOrg[orgID], *in.Number)
	}

	duplicateNumbers := map[string]bool{}
	for org, numbers := range numbersByOrg {
		dupQuery := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				domainQuery.NewFilter("issuer_organization_id", domainQuery.OperatorEqual, org),
				domainQuery.NewFilter("number", domainQuery.OperatorIn, numbers...),
			},
		}
		rows, _, err := s.repo.Get(ctx, dupQuery)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if r.Number != nil {
				duplicateNumbers[org+"\x00"+*r.Number] = true
			}
		}
	}

	for i := range credentials {
		in := &credentials[i]
		target := targetByID[in.ID]
		if in.Number == nil || *in.Number == derefString(target.Number) {
			continue
		}
		orgID := target.IssuerOrganizationID
		if in.IssuerOrganizationID != "" {
			orgID = in.IssuerOrganizationID
		}
		if duplicateNumbers[orgID+"\x00"+*in.Number] {
			return domain.NewError(domain.CodeCredentialIssueNumberDuplicate,
				domain.WithMetadata("credential_ids", []string{in.ID}))
		}
	}
	return nil
}

// derefString returns "" for a nil *string, the value otherwise.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ── Review: Approve / Reject ─────────────────────────────────────────────

// Approve batch-approves pending self-submissions. Approval is the on-chain
// mint trigger: the approval update and the mint run in the SAME unit of
// work, so a failed mint rolls the approval back and the rows stay pending.
// No queue, no background worker, no deferred mint.
func (s *credentialService) Approve(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)
	now := time.Now()
	approverID := authUser.Id

	var approved []domain.Credential
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids, nil)
		if err != nil {
			return err
		}
		targetIDs := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
		if missing, _ := lo.Difference(ids, targetIDs); len(missing) > 0 {
			return domain.NewError(domain.CodeCredentialReviewNotFound, domain.WithMetadata("credential_ids", missing))
		}
		for _, t := range targets {
			switch t.LifecycleStatus() {
			case domain.CredentialLifecycleStatusApproved:
				return domain.NewError(domain.CodeCredentialReviewAlreadyApproved, domain.WithMetadata("credential_ids", []string{t.ID}))
			case domain.CredentialLifecycleStatusRejected:
				return domain.NewError(domain.CodeCredentialReviewAlreadyRejected, domain.WithMetadata("credential_ids", []string{t.ID}))
			case domain.CredentialLifecycleStatusRevoked:
				return domain.NewError(domain.CodeCredentialReviewAlreadyRevoked, domain.WithMetadata("credential_ids", []string{t.ID}))
			}
		}

		holderIDs := lo.Map(targets, func(c domain.Credential, _ int) string { return c.HolderUserID })
		holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
		if err != nil {
			return err
		}
		holderByID := lo.SliceToMap(holders, func(h domain.User) (string, domain.User) { return h.Id, h })

		updates := make([]domain.Credential, len(targets))
		for i, t := range targets {
			updates[i] = domain.Credential{
				ID:             t.ID,
				ApproverUserID: &approverID,
				ApprovedAt:     &now,
				IssuerUserID:   approverID,                  // the officer who writes to chain
				ExtractStatus:  domain.ExtractStatusPending, // job enqueued below
			}
		}
		updated, err := uow.Credential().Update(ctx, updates...)
		if err != nil {
			return err
		}

		if err := s.mintCredentials(ctx, uow, domain.WalletFromUser(*authUser), updated, holderByID,
			domain.CodeCredentialReviewBlockchainSyncFailed); err != nil {
			return err // UoW rolls back the approval update — rows stay pending
		}

		approved = updated
		return nil
	})
	return approved, err
}

// Reject batch-rejects pending submissions with per-credential reasons.
// Rejected rows are historical: they are never edited back into pending —
// re-submission creates a new row.
func (s *credentialService) Reject(ctx context.Context, rejections []CredentialRejection) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)
	now := time.Now()
	rejecterID := authUser.Id

	ids := lo.Map(rejections, func(r CredentialRejection, _ int) string { return r.ID })
	reasonByID := lo.SliceToMap(rejections, func(r CredentialRejection) (string, string) { return r.ID, r.Reason })

	var rejected []domain.Credential
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids, nil)
		if err != nil {
			return err
		}
		targetIDs := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
		if missing, _ := lo.Difference(ids, targetIDs); len(missing) > 0 {
			return domain.NewError(domain.CodeCredentialReviewNotFound, domain.WithMetadata("credential_ids", missing))
		}
		for _, t := range targets {
			switch t.LifecycleStatus() {
			case domain.CredentialLifecycleStatusApproved:
				return domain.NewError(domain.CodeCredentialReviewAlreadyApproved, domain.WithMetadata("credential_ids", []string{t.ID}))
			case domain.CredentialLifecycleStatusRejected:
				return domain.NewError(domain.CodeCredentialReviewAlreadyRejected, domain.WithMetadata("credential_ids", []string{t.ID}))
			case domain.CredentialLifecycleStatusRevoked:
				return domain.NewError(domain.CodeCredentialReviewAlreadyRevoked, domain.WithMetadata("credential_ids", []string{t.ID}))
			}
		}
		updates := make([]domain.Credential, len(targets))
		for i, t := range targets {
			reason := reasonByID[t.ID]
			updates[i] = domain.Credential{
				ID:              t.ID,
				RejecterUserID:  &rejecterID,
				RejectedAt:      &now,
				RejectionReason: &reason,
			}
		}
		rejected, err = uow.Credential().Update(ctx, updates...)
		return err
	})
	return rejected, err
}

// issueEnqueueExtractJob enqueues a River extraction job.
// River jobs live in Postgres (river_jobs table) but use a separate connection
// pool (pgx) from GORM's (database/sql + pgx). They cannot share a transaction.
// This means a credential can be committed without its extraction job (rare:
// server crash between Update and Insert). Mitigation: the credential stays in
// extract_status=pending and the reextract endpoint can recover it.
func (s *credentialService) issueEnqueueExtractJob(ctx context.Context, credentialID, fileURI string) error {
	return s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{
		CredentialID: credentialID,
		FileURI:      fileURI,
	})
}

// ── Revoke ────────────────────────────────────────────────────────────────

// Revoke batch-revokes credentials by ID. Sets revoked_at, revoker_user_id
// in the database and syncs the revocation on-chain via the CredentialRegistry.
// Uses Update (CASE-based) — there is no separate Revoke method on the repository.
func (s *credentialService) Revoke(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)
	now := time.Now()
	revokerID := authUser.Id

	var (
		revoked    []domain.Credential
		fileHashes []string
	)
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids, nil)
		if err != nil {
			return err
		}

		fileHashes = lo.Map(targets, func(c domain.Credential, _ int) string { return c.FileHash })

		targetIds := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
		missing, _ := lo.Difference(ids, targetIds)
		if len(missing) > 0 {
			return domain.NewError(domain.CodeCredentialRevokeNotFound,
				domain.WithMetadata("credential_ids", missing))
		}

		alreadyRevoked := []string{}
		for _, t := range targets {
			if t.LifecycleStatus() == domain.CredentialLifecycleStatusRevoked {
				alreadyRevoked = append(alreadyRevoked, t.ID)
			}
		}
		if len(alreadyRevoked) > 0 {
			return domain.NewError(domain.CodeCredentialRevokeAlreadyRevoked,
				domain.WithMetadata("credential_ids", alreadyRevoked))
		}

		if err := s.policy.RevokePostFetch(ctx, targets); err != nil {
			return err
		}

		updates := make([]domain.Credential, len(targets))
		tokenIds := make([]string, 0, len(targets))
		for i, t := range targets {
			updates[i] = domain.Credential{
				ID:            t.ID,
				RevokedAt:     &now,
				RevokerUserID: &revokerID,
			}
			if t.TokenID != nil {
				tokenIds = append(tokenIds, *t.TokenID)
			}
		}

		revoked, err = uow.Credential().Update(ctx, updates...)
		if err != nil {
			return err
		}

		if len(tokenIds) > 0 {
			signerWallet := domain.WalletFromUser(*authUser)
			if err := s.syncBlockchainRevoke(ctx, signerWallet, tokenIds); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil && len(fileHashes) > 0 && s.verificationRepo != nil {
		if delErr := s.verificationRepo.DeleteByUploadedFileHashes(ctx, fileHashes); delErr != nil {
			s.logger.Warn("failed to delete verification cache entries after revoke", zap.Error(delErr), zap.Strings("file_hashes", fileHashes))
		}
	}
	return revoked, err
}

// ── Verify ────────────────────────────────────────────────────────────────

// Verify runs the cache → exact hash → fuzzy pipeline against the uploaded
// file. Returns (verdictCode, matchedCredential, score, percent, error).
func (s *credentialService) Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error) {
	uploadedHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(file.Data))

	verifyQuery := &domainQuery.Query{Includes: []string{"holder", "issuer", "revoker"}}

	// CACHE LOOKUP
	cached, err := s.verificationRepo.FindByUploadedFileHash(ctx, uploadedHash)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if cached != nil {
		var cred *domain.Credential
		if cached.MatchedCredentialID != nil {
			cred, _ = s.repo.FindVerifiableById(ctx, *cached.MatchedCredentialID, verifyQuery)
		}
		code := cached.VerdictCode
		if code == domain.CodeCredentialVerifyAuthentic {
			if cred != nil && cred.LifecycleStatus() == domain.CredentialLifecycleStatusRevoked {
				code = domain.CodeCredentialVerifyRevoked
			} else if cred != nil {
				holderGone := cred.Holder == nil || cred.Holder.DeletedAt != nil
				issuerGone := cred.Issuer == nil || cred.Issuer.DeletedAt != nil
				if holderGone && issuerGone {
					code = domain.CodeCredentialVerifyPartyDisabled
				} else if holderGone {
					code = domain.CodeCredentialVerifyHolderDisabled
				} else if issuerGone {
					code = domain.CodeCredentialVerifyIssuerDisabled
				}
			} else {
				code = domain.CodeCredentialVerifyIntegrityWarning
			}
		}
		return code, cred, cached.SimilarityScore, cached.SimilarityPercent, nil
	}

	// EXACT-HASH PATH: Postgres bridge → on-chain cross-check
	existing, err := s.repo.FindByFileHashes(ctx, []string{uploadedHash}, verifyQuery)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(existing) > 0 {
		tokenIds := make([]*big.Int, 0, len(existing))
		credByTokenId := make(map[string]*domain.Credential, len(existing))
		for i := range existing {
			if existing[i].TokenID != nil {
				tid, ok := new(big.Int).SetString(*existing[i].TokenID, 10)
				if ok {
					tokenIds = append(tokenIds, tid)
					credByTokenId[tid.String()] = &existing[i]
				}
			}
		}
		if len(tokenIds) > 0 {
			onChainCreds, chainErr := s.registryService.GetCredentialsByIds(ctx, tokenIds)
			if chainErr == nil {
				var best *domain.Credential
				var bestOnChain *contracts.CredentialRegistryCredential
				bestIsRevoked := true
				for i := range onChainCreds {
					if onChainCreds[i].Hash != uploadedHash {
						continue
					}
					revoked := onChainCreds[i].RevokedAt != nil && onChainCreds[i].RevokedAt.Cmp(big.NewInt(0)) > 0
					if best == nil || (bestIsRevoked && !revoked) ||
						(bestIsRevoked == revoked && onChainCreds[i].IssuedAt.Cmp(bestOnChain.IssuedAt) > 0) {
						idStr := onChainCreds[i].Id.String()
						best = credByTokenId[idStr]
						bestOnChain = &onChainCreds[i]
						bestIsRevoked = revoked
					}
				}
				if best != nil {
					code := domain.CodeCredentialVerifyAuthentic
					if bestIsRevoked {
						code = domain.CodeCredentialVerifyRevoked
					}
					if code == domain.CodeCredentialVerifyAuthentic {
						holderGone := best.Holder == nil || best.Holder.DeletedAt != nil
						issuerGone := best.Issuer == nil || best.Issuer.DeletedAt != nil
						if holderGone && issuerGone {
							code = domain.CodeCredentialVerifyPartyDisabled
						} else if holderGone {
							code = domain.CodeCredentialVerifyHolderDisabled
						} else if issuerGone {
							code = domain.CodeCredentialVerifyIssuerDisabled
						}
					}
					s.verifyCacheVerdict(ctx, uploadedHash, code, &best.ID, nil, nil)
					return code, best, nil, nil, nil
				}
			}
		}
		// No valid on-chain match found
		code := domain.CodeCredentialVerifyIntegrityWarning
		s.verifyCacheVerdict(ctx, uploadedHash, code, &existing[0].ID, nil, nil)
		return code, &existing[0], nil, nil, nil
	}

	// FUZZY PATH
	ids, err := s.aiClient.ExtractIDs(ctx, file)
	if err != nil {
		return 0, nil, nil, nil, domain.NewError(domain.CodeCredentialVerifyAiServiceFailed, domain.WithError(err))
	}
	if len(ids) == 0 {
		code := domain.CodeCredentialVerifyNoIdentifiers
		s.verifyCacheVerdict(ctx, uploadedHash, code, nil, nil, nil)
		return code, nil, nil, nil, nil
	}

	values := lo.Map(ids, func(id pyai.ExtractedID, _ int) string { return id.Value })
	ranked, err := s.extractionRepo.FindRankedByIds(ctx, values, 10)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(ranked) == 0 {
		code := domain.CodeCredentialVerifyNoMatch
		s.verifyCacheVerdict(ctx, uploadedHash, code, nil, nil, nil)
		return code, nil, nil, nil, nil
	}

	best := s.verifyPickBestMatch(ctx, ranked, values)
	result, err := s.aiClient.Verify(ctx, file, best.Embedding)
	if err != nil {
		if errors.Is(err, pyai.ErrVerifyFileUnprocessable) {
			return 0, nil, nil, nil, domain.NewError(domain.CodeCredentialVerifyDocumentUnreadable, domain.WithError(err))
		}
		return 0, nil, nil, nil, domain.NewError(domain.CodeCredentialVerifyAiServiceFailed, domain.WithError(err))
	}

	code := s.verifyVerdictToCode(result.Verdict)
	cred, _ := s.repo.FindVerifiableById(ctx, best.CredentialID, verifyQuery)
	if code == domain.CodeCredentialVerifyAuthentic && cred != nil {
		holderGone := cred.Holder == nil || cred.Holder.DeletedAt != nil
		issuerGone := cred.Issuer == nil || cred.Issuer.DeletedAt != nil
		if holderGone && issuerGone {
			code = domain.CodeCredentialVerifyPartyDisabled
		} else if holderGone {
			code = domain.CodeCredentialVerifyHolderDisabled
		} else if issuerGone {
			code = domain.CodeCredentialVerifyIssuerDisabled
		}
	}
	s.verifyCacheVerdict(ctx, uploadedHash, code, &best.CredentialID, &result.SimilarityScore, &result.SimilarityPercent)
	return code, cred, &result.SimilarityScore, &result.SimilarityPercent, nil
}

// verifyPickBestMatch selects the best-matching extraction from ranked
// results. Uses one FindVerifiableByIds IN-query for tie-breaking (no
// per-candidate queries). Prefers non-revoked credentials; then prefers
// newer IssuedAt.
func (s *credentialService) verifyPickBestMatch(ctx context.Context, ranked []domain.CredentialExtraction, values []string) domain.CredentialExtraction {
	maxCount := s.verifyCountIntersection(ranked[0].IDs, values)
	var tied []domain.CredentialExtraction
	for _, r := range ranked {
		if s.verifyCountIntersection(r.IDs, values) == maxCount {
			tied = append(tied, r)
		}
	}
	if len(tied) == 1 {
		return tied[0]
	}
	ids := lo.Map(tied, func(e domain.CredentialExtraction, _ int) string { return e.CredentialID })
	creds, err := s.repo.FindVerifiableByIds(ctx, ids, nil)
	if err != nil {
		s.logger.Warn("verifyPickBestMatch: FindVerifiableByIds failed, falling back to first candidate",
			zap.Error(err),
			zap.Int("candidate_count", len(tied)),
		)
		return tied[0]
	}
	credByID := lo.SliceToMap(creds, func(c domain.Credential) (string, domain.Credential) { return c.ID, c })
	best, bestCred, bestOK := tied[0], credByID[tied[0].CredentialID], true
	for _, t := range tied[1:] {
		tc, ok := credByID[t.CredentialID]
		if !ok {
			continue
		}
		if !bestOK {
			best, bestCred, bestOK = t, tc, true
			continue
		}
		bestRevoked := bestCred.LifecycleStatus() == domain.CredentialLifecycleStatusRevoked
		tRevoked := tc.LifecycleStatus() == domain.CredentialLifecycleStatusRevoked
		if bestRevoked && !tRevoked {
			best, bestCred = t, tc
		} else if bestRevoked == tRevoked && tc.IssuedAt.After(bestCred.IssuedAt) {
			best, bestCred = t, tc
		}
	}
	return best
}

// verifyCountIntersection counts how many of the extraction's IDs appear in
// the uploaded file's value set.
func (s *credentialService) verifyCountIntersection(ids []domain.CredentialExtractedID, values []string) int {
	set := lo.SliceToMap(values, func(v string) (string, struct{}) { return v, struct{}{} })
	count := 0
	for _, id := range ids {
		if _, ok := set[id.Value]; ok {
			count++
		}
	}
	return count
}

// verifyVerdictToCode maps the Python /verify verdict string to a domain code.
func (s *credentialService) verifyVerdictToCode(verdict string) int {
	switch verdict {
	case "authentic":
		return domain.CodeCredentialVerifyAuthentic
	case "tampered":
		return domain.CodeCredentialVerifyTampered
	case "suspicious":
		return domain.CodeCredentialVerifySuspicious
	case "low_similarity":
		return domain.CodeCredentialVerifyLowSimilarity
	default:
		return domain.CodeCredentialVerifyNotSimilar
	}
}

// verifyCacheVerdict stores the verify result in the MongoDB cache. Logs
// failures (non-fatal — the next call will recompute).
func (s *credentialService) verifyCacheVerdict(ctx context.Context, hash string, code int, credID *string, score *float64, percent *string) {
	if err := s.verificationRepo.Store(ctx, domain.CredentialVerification{
		UploadedFileHash:    hash,
		VerdictCode:         code,
		MatchedCredentialID: credID,
		SimilarityScore:     score,
		SimilarityPercent:   percent,
	}); err != nil {
		s.logger.Warn("failed to cache verification", zap.String("hash", hash), zap.Error(err))
	}
}

// ── ReExtract ─────────────────────────────────────────────────────────────

// ReExtract resets failed credentials to pending and enqueues new extract
// jobs. All-or-nothing within one UoW: validates all exist + are failed with
// file_uri, resets to pending, then enqueues River jobs after commit.
func (s *credentialService) ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	var updated []domain.Credential
	var toEnqueue []domain.Credential
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids, nil)
		if err != nil {
			return err
		}
		if err := s.reExtractValidate(ids, targets); err != nil {
			return err
		}
		updates := make([]domain.Credential, len(targets))
		for i, t := range targets {
			emptyErr := ""
			updates[i] = domain.Credential{
				ID:            t.ID,
				ExtractStatus: domain.ExtractStatusPending,
				ExtractError:  &emptyErr,
			}
		}
		updated, err = uow.Credential().Update(ctx, updates...)
		if err != nil {
			return err
		}
		toEnqueue = targets
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Enqueue River jobs. If enqueue fails, compensate: re-stamp the failed
	// credential back to failed so the operator can retry the whole batch.
	// Earlier credentials in this batch remain in pending with active jobs and
	// will be extracted normally by the River worker.
	for _, t := range toEnqueue {
		if err := s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{
			CredentialID: t.ID, FileURI: filepath.Join(*s.cfg.CredentialFileStoragePath, *t.FileURI),
		}); err != nil {
			if compErr := s.reExtractCompensate(ctx, t); compErr != nil {
				s.logger.Error("reextract compensate failed",
					zap.String("credential_id", t.ID), zap.Error(compErr))
			}
			return nil, err
		}
	}
	return updated, nil
}

// reExtractCompensate restamps a credential back to ExtractStatusFailed after a
// failed enqueue attempt so the operator can retry reextraction.
func (s *credentialService) reExtractCompensate(ctx context.Context, t domain.Credential) error {
	errMsg := "reenqueue failed"
	if t.ExtractError != nil {
		errMsg = *t.ExtractError
	}
	_, err := s.repo.Update(ctx, domain.Credential{
		ID:            t.ID,
		ExtractStatus: domain.ExtractStatusFailed,
		ExtractError:  &errMsg,
	})
	return err
}

// reExtractValidate ensures all targets exist and are in failed state with a
// file URI. Helper prefixed with the method name "reExtract".
func (s *credentialService) reExtractValidate(ids []string, targets []domain.Credential) error {
	targetIds := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
	if missing, _ := lo.Difference(ids, targetIds); len(missing) > 0 {
		return domain.NewError(domain.CodeCredentialReExtractNotFound,
			domain.WithMetadata("credential_ids", missing))
	}
	notFailed := []string{}
	for _, t := range targets {
		if t.ExtractStatus != domain.ExtractStatusFailed || t.FileURI == nil {
			notFailed = append(notFailed, t.ID)
		}
	}
	if len(notFailed) > 0 {
		return domain.NewError(domain.CodeCredentialReExtractNotEligible,
			domain.WithMetadata("credential_ids", notFailed))
	}
	return nil
}

// ── DownloadFile ──────────────────────────────────────────────────────────

// DownloadFile retrieves a single credential file, validates authorization,
// decrypts with FILE_ENCRYPTION_KEY, and returns the plaintext bytes along
// with the filename and MIME type for HTTP response.
func (s *credentialService) DownloadFile(ctx context.Context, id string) ([]byte, string, string, error) {
	query := &domainQuery.Query{Includes: []string{"holder"}}
	target, err := s.repo.Find(ctx, id, query)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", domain.NewError(domain.CodeCredentialFileDownloadNotFound,
				domain.WithMetadata("credential_id", id))
		}
		return nil, "", "", err
	}
	if err := s.policy.DownloadFilePreFetch(ctx, *target); err != nil {
		return nil, "", "", err
	}
	if target.FileURI == nil {
		return nil, "", "", domain.NewError(domain.CodeCredentialFileDownloadNoFile,
			domain.WithMetadata("credential_id", id))
	}
	filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, *target.FileURI)
	encryptedHex, err := s.storage.ReadBytes(filePath)
	if err != nil {
		return nil, "", "", err
	}
	key := []byte(*s.cfg.FileEncryptionKey)
	decrypted, err := infraCrypto.Decrypt(string(encryptedHex), key)
	if err != nil {
		return nil, "", "", domain.NewError(domain.CodeCredentialFileDownloadDecryptionFailed,
			domain.WithError(err))
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(*target.FileURI)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return decrypted, *target.FileURI, mimeType, nil
}

// ── Blockchain sync helpers ───────────────────────────────────────────────

// mintCredentials mints stored credentials on chain (signer wallet), persists
// the returned token ids, and enqueues extract jobs. Called inside a UoW —
// any error rolls back the caller's writes. chainFailCode is the domain code
// for a failed registry call (issue uses 400244, approve uses 401244);
// the contract's IssuedCredentialError always maps to 400242.
func (s *credentialService) mintCredentials(
	ctx context.Context,
	uow domain.UnitOfWork,
	signer domain.Wallet,
	stored []domain.Credential,
	holderByID map[string]domain.User,
	chainFailCode int,
) error {
	issuances := make([]chain.CredentialIssuance, len(stored))
	for i, c := range stored {
		issuances[i] = chain.CredentialIssuance{
			HolderAddress: holderByID[c.HolderUserID].WalletAddress,
			Hash:          c.FileHash,
			URI:           c.ID,
			IssuedAt:      credentialIssuedAtToChain(c.IssuedAt),
			ExpiresAt:     credentialExpiresAtToChain(c.ExpiresAt),
		}
	}
	tokenIds, err := s.registryService.IssueCredentials(ctx, signer, issuances...)
	if err != nil {
		if strings.Contains(err.Error(), "IssuedCredentialError") {
			return domain.NewError(domain.CodeCredentialIssueDuplicateFileHash, domain.WithError(err))
		}
		return domain.NewError(chainFailCode, domain.WithError(err))
	}
	updates := make([]domain.Credential, len(stored))
	for i, c := range stored {
		tok := tokenIds[i].String()
		stored[i].TokenID = &tok
		updates[i] = domain.Credential{ID: c.ID, TokenID: &tok}
	}
	if _, err := uow.Credential().Update(ctx, updates...); err != nil {
		return err
	}
	for _, c := range stored {
		fileURI := filepath.Join(*s.cfg.CredentialFileStoragePath, *c.FileURI)
		if err := s.issueEnqueueExtractJob(ctx, c.ID, fileURI); err != nil {
			return err
		}
	}
	return nil
}

// syncBlockchainRevoke calls RegistryService.RevokeCredentials with the
// given token ID strings (decimal) and translates raw chain errors so the
// UoW transaction rolls back the DB revoke.
func (s *credentialService) syncBlockchainRevoke(ctx context.Context, signer domain.Wallet, tokenIDStrings []string) error {
	if len(tokenIDStrings) == 0 {
		return nil
	}
	tokenIDs := make([]*big.Int, 0, len(tokenIDStrings))
	for _, idStr := range tokenIDStrings {
		bi, ok := new(big.Int).SetString(idStr, 10)
		if !ok {
			return domain.NewError(domain.CodeCredentialRevokeBlockchainSyncFailed,
				domain.WithError(fmt.Errorf("invalid token id: %s", idStr)))
		}
		tokenIDs = append(tokenIDs, bi)
	}
	if err := s.registryService.RevokeCredentials(ctx, signer, tokenIDs...); err != nil {
		return domain.NewError(domain.CodeCredentialRevokeBlockchainSyncFailed,
			domain.WithError(err))
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

// cleanupOrphanFiles deletes files that were persisted to storage but whose
// credential rows were never committed (e.g. because of a validation error
// or a chain failure rollback). Best-effort — log-and-continue.
func (s *credentialService) cleanupOrphanFiles(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := s.storage.Delete(p); err != nil {
			s.logger.Warn("failed to clean up orphan file",
				zap.String("path", p),
				zap.Error(err))
		}
	}
}
