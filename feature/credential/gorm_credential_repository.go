package credential

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// ── Repository struct & factory ──────────────────────────────────────────

type gormCredentialRepository struct {
	db *gorm.DB
}

// NewGormCredentialRepository is the exported factory for FX injection.
func NewGormCredentialRepository(db *gorm.DB) domain.CredentialRepository {
	return &gormCredentialRepository{db: db}
}

// ── Column allowlists (dialect-agnostic; Postgres + SQLite) ─────────────

// allowedFilterColumns whitelists credential columns clients may filter on.
// holder_user_id and issuer_user_id are intentionally included so the
// user-detail UI can scope credentials to a specific holder or issuer.
var allowedFilterColumns = map[string]bool{
	"name":                   true,
	"issued_at":              true,
	"revoked_at":             true,
	"holder_user_id":         true,
	"issuer_user_id":         true,
	"extract_status":         true,
	"approved_at":            true,
	"rejected_at":            true,
	"type_id":                true,
	"issuer_organization_id": true,
	"number":                 true,
	"expires_at":             true,
}

// allowedSortColumns whitelists credential columns plus virtual joined-user
// columns (prefixed "holder_") that clients may sort on. Joined columns
// trigger an additional LEFT JOIN users AS holder at query build time.
// Sorts on non-allowlisted columns are silently ignored.
var allowedSortColumns = map[string]bool{
	"name":          true,
	"issued_at":     true,
	"revoked_at":    true,
	"holder_name":   true,
	"holder_email":  true,
	"holder_number": true,
}

// ── Preload helper ────────────────────────────────────────────────────────

// preloadByIncludes applies GORM Preload for each include key present in the
// query. Supported keys: "holder", "issuer", "revoker". A single batch
// IN-clause query runs per Preload regardless of result size (no N+1).
func preloadByIncludes(db *gorm.DB, query *domainQuery.Query) *gorm.DB {
	if query == nil {
		return db
	}
	for _, inc := range query.Includes {
		switch inc {
		case "holder":
			db = db.Preload("HolderUser", func(db *gorm.DB) *gorm.DB {
				return db.Unscoped()
			})
		case "issuer":
			db = db.Preload("IssuerUser", func(db *gorm.DB) *gorm.DB {
				return db.Unscoped()
			})
		case "revoker":
			db = db.Preload("RevokerUser", func(db *gorm.DB) *gorm.DB {
				return db.Unscoped()
			})
		}
	}
	return db
}

// ── Join helper ───────────────────────────────────────────────────────────

// needsHolderJoin reports whether we must LEFT JOIN users AS holder for the
// given query. Search always needs the holder join (name/email/number/phone
// search predicates); sorts on holder_* columns also require it.
func needsHolderJoin(query *domainQuery.Query) bool {
	if query == nil {
		return false
	}
	return query.HasSearch() ||
		lo.ContainsBy(query.Sorts, func(s domainQuery.Sort) bool { return strings.HasPrefix(s.Column, "holder_") })
}

func needsIssuerJoin(query *domainQuery.Query) bool {
	return query != nil && query.HasSearch()
}

func needsRevokerJoin(query *domainQuery.Query) bool {
	return query != nil && query.HasSearch()
}

// mapSortColumn translates a user-facing sort column into a DB-qualified
// column expression (e.g. "holder_name" → "holder.name").
func mapSortColumn(col string) string {
	switch col {
	case "name", "issued_at", "revoked_at":
		return "credentials." + col
	case "holder_name":
		return "holder.name"
	case "holder_email":
		return "holder.email"
	case "holder_number":
		return "holder.number"
	default:
		return col
	}
}

// ── Pagination ────────────────────────────────────────────────────────────

// Get retrieves credentials with pagination, search, filters, sorts, and
// optional includes. Search spans credentials identity fields (id, token_id,
// file_hash, name, meta) plus the holder/issuer/revoker users' name/email/number
// via LEFT JOINs (all activated when HasSearch is true).
//
// When query.Includes contains "holder", "issuer", or "revoker", the
// corresponding GORM Preload runs — a single batch IN-clause query per
// Preload regardless of result size.
func (r *gormCredentialRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{})

	if needsHolderJoin(query) {
		db = db.Joins("LEFT JOIN users AS holder ON holder.id = credentials.holder_user_id")
	}

	if needsIssuerJoin(query) {
		db = db.Joins("LEFT JOIN users AS issuer ON issuer.id = credentials.issuer_user_id")
	}

	if needsRevokerJoin(query) {
		db = db.Joins("LEFT JOIN users AS revoker ON revoker.id = credentials.revoker_user_id")
	}

	if query != nil {
		if query.HasSearch() {
			needle := "%" + query.Search + "%"
			db = db.Where(
				"LOWER(credentials.name) LIKE LOWER(?) OR "+
					"LOWER(CAST(credentials.meta AS TEXT)) LIKE LOWER(?) OR "+
					"LOWER(credentials.id) LIKE LOWER(?) OR "+
					"LOWER(credentials.token_id) LIKE LOWER(?) OR "+
					"LOWER(credentials.file_hash) LIKE LOWER(?) OR "+
					"LOWER(holder.name) LIKE LOWER(?) OR "+
					"LOWER(holder.email) LIKE LOWER(?) OR "+
					"LOWER(holder.number) LIKE LOWER(?) OR "+
					"LOWER(issuer.name) LIKE LOWER(?) OR "+
					"LOWER(issuer.email) LIKE LOWER(?) OR "+
					"LOWER(issuer.number) LIKE LOWER(?) OR "+
					"LOWER(revoker.name) LIKE LOWER(?) OR "+
					"LOWER(revoker.email) LIKE LOWER(?) OR "+
					"LOWER(revoker.number) LIKE LOWER(?)",
				needle, needle, needle, needle, needle, // 5 credential cols
				needle, needle, needle, // 3 holder cols
				needle, needle, needle, // 3 issuer cols
				needle, needle, needle, // 3 revoker cols
			)
		}

		if query.HasFilters() {
			db = gormhelpers.ApplyFilters(db, query.Filters, allowedFilterColumns, "credentials.")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedSortColumns, "credentials.issued_at DESC", mapSortColumn, "credentials.id ASC")
	db = preloadByIncludes(db, query)
	db = gormhelpers.ApplyPagination(db, query)

	var credentials []model.Credential
	if err := db.Find(&credentials).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.Credential, len(credentials))
	for i, c := range credentials {
		out[i] = c.ToDomain()
	}
	return out, int(total), nil
}

// ── Single-row lookups ────────────────────────────────────────────────────

// Find retrieves a single credential by ID, applying optional Preloads
// from query.Includes.
func (r *gormCredentialRepository) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{})
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var c model.Credential
	if err := db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := c.ToDomain()
	return &d, nil
}

// FindVerifiableById retrieves a single approved credential by ID.
// Verification path only: rows with approved_at IS NULL are invisible.
func (r *gormCredentialRepository) FindVerifiableById(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{}).Where("approved_at IS NOT NULL")
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var c model.Credential
	if err := db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := c.ToDomain()
	return &d, nil
}

// FindVerifiableByIds retrieves approved credentials by ID list.
// Verification path only: rows with approved_at IS NULL are invisible.
func (r *gormCredentialRepository) FindVerifiableByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
	if len(ids) == 0 {
		return []domain.Credential{}, nil
	}
	db := r.db.WithContext(ctx).Where("approved_at IS NOT NULL")
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// ── Batch lookups ─────────────────────────────────────────────────────────

// FindByIds retrieves credentials by ID list (batch lookup).
func (r *gormCredentialRepository) FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
	if len(ids) == 0 {
		return []domain.Credential{}, nil
	}
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// FindByHolderId retrieves all credentials owned by a single holder.
func (r *gormCredentialRepository) FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]domain.Credential, error) {
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("holder_user_id = ?", holderID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// FindByFileHashes retrieves approved credentials whose file_hash matches any
// of the supplied hashes. Verification path only: rows with approved_at IS
// NULL are invisible.
func (r *gormCredentialRepository) FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]domain.Credential, error) {
	if len(hashes) == 0 {
		return []domain.Credential{}, nil
	}
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("file_hash IN ?", hashes).Where("approved_at IS NOT NULL").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// ── Mutations ─────────────────────────────────────────────────────────────

// Store batch-inserts new credentials. Generates ULIDs for any missing IDs
// and defaults ExtractStatus to pending.
func (r *gormCredentialRepository) Store(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	if len(credentials) == 0 {
		return []domain.Credential{}, nil
	}
	for i := range credentials {
		if credentials[i].ID == "" {
			credentials[i].ID = ulid.Make().String()
		}
		if credentials[i].ExtractStatus == "" {
			credentials[i].ExtractStatus = domain.ExtractStatusPending
		}
	}
	rows := make([]model.Credential, len(credentials))
	for i, c := range credentials {
		rows[i] = model.FromDomainCredential(c)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.NewError(domain.CodeCredentialIssueDuplicateFileHash,
				domain.WithError(err))
		}
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Update partially updates one or more credentials using a single batched
// UPDATE with per-column CASE expressions. Only non-nil / non-zero fields
// are touched; unspecified columns fall through to ELSE column (preserving
// the existing value). This eliminates the N+1 per-row UPDATE pattern.
func (r *gormCredentialRepository) Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	if len(credentials) == 0 {
		return []domain.Credential{}, nil
	}
	if err := r.updateBatchCase(ctx, credentials); err != nil {
		return nil, err
	}
	ids := make([]string, len(credentials))
	for i, c := range credentials {
		ids[i] = c.ID
	}
	return r.FindByIds(ctx, ids, nil)
}

// updateBatchCase builds and executes a single UPDATE statement using CASE
// expressions for each column that at least one credential provides. Users
// sorted by ID for deterministic arg ordering.
func (r *gormCredentialRepository) updateBatchCase(ctx context.Context, items []domain.Credential) error {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	var clauses []string
	var allArgs [][]interface{}

	addCol := func(col string, getValue func(domain.Credential) (interface{}, bool)) {
		var pairs []interface{}
		for _, c := range items {
			if v, ok := getValue(c); ok {
				pairs = append(pairs, c.ID, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}

	addCol("name", func(c domain.Credential) (interface{}, bool) {
		if c.Name != "" {
			return c.Name, true
		}
		return nil, false
	})
	addCol("number", func(c domain.Credential) (interface{}, bool) {
		if c.Number == nil {
			return nil, false
		}
		return *c.Number, true
	})
	addCol("submitter_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.SubmitterUserID == "" {
			return nil, false
		}
		return c.SubmitterUserID, true
	})
	addCol("issuer_organization_id", func(c domain.Credential) (interface{}, bool) {
		if c.IssuerOrganizationID == "" {
			return nil, false
		}
		return c.IssuerOrganizationID, true
	})
	addCol("type_id", func(c domain.Credential) (interface{}, bool) {
		if c.TypeID == "" {
			return nil, false
		}
		return c.TypeID, true
	})
	addCol("expires_at", func(c domain.Credential) (interface{}, bool) {
		if c.ExpiresAt == nil {
			return nil, false
		}
		return *c.ExpiresAt, true
	})
	addCol("approver_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.ApproverUserID == nil {
			return nil, false
		}
		return *c.ApproverUserID, true
	})
	addCol("approved_at", func(c domain.Credential) (interface{}, bool) {
		if c.ApprovedAt == nil {
			return nil, false
		}
		return *c.ApprovedAt, true
	})
	addCol("rejecter_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.RejecterUserID == nil {
			return nil, false
		}
		return *c.RejecterUserID, true
	})
	addCol("rejected_at", func(c domain.Credential) (interface{}, bool) {
		if c.RejectedAt == nil {
			return nil, false
		}
		return *c.RejectedAt, true
	})
	addCol("rejection_reason", func(c domain.Credential) (interface{}, bool) {
		if c.RejectionReason == nil {
			return nil, false
		}
		return *c.RejectionReason, true
	})
	addCol("meta", func(c domain.Credential) (interface{}, bool) {
		if c.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(c.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCol("token_id", func(c domain.Credential) (interface{}, bool) {
		if c.TokenID == nil {
			return nil, false
		}
		return *c.TokenID, true
	})
	addCol("file_uri", func(c domain.Credential) (interface{}, bool) {
		if c.FileURI == nil {
			return nil, false
		}
		return *c.FileURI, true
	})
	addCol("revoked_at", func(c domain.Credential) (interface{}, bool) {
		if c.RevokedAt == nil {
			return nil, false
		}
		return *c.RevokedAt, true
	})
	addCol("revoker_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.RevokerUserID == nil {
			return nil, false
		}
		return *c.RevokerUserID, true
	})
	addCol("extract_status", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractStatus == "" {
			return nil, false
		}
		return string(c.ExtractStatus), true
	})
	addCol("extract_error", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractError == nil {
			return nil, false
		}
		return *c.ExtractError, true
	})
	addCol("extracted_at", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractedAt == nil {
			return nil, false
		}
		return *c.ExtractedAt, true
	})

	if len(clauses) == 0 {
		return nil
	}

	ids := make([]interface{}, len(items))
	for i, c := range items {
		ids[i] = c.ID
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("credentials", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}

// ── Count primitives (deletion guards) ─────────────────────────────────────

// CountByTypeIds counts credentials referencing any of the given type ids.
func (r *gormCredentialRepository) CountByTypeIds(ctx context.Context, typeIds ...string) (int64, error) {
	if len(typeIds) == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Credential{}).Where("type_id IN ?", typeIds).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByIssuerOrganizationIds counts credentials referencing any of the
// given issuer organization ids.
func (r *gormCredentialRepository) CountByIssuerOrganizationIds(ctx context.Context, organizationIds ...string) (int64, error) {
	if len(organizationIds) == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Credential{}).Where("issuer_organization_id IN ?", organizationIds).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveByFileHashes counts credentials whose file_hash matches any of
// the given hashes AND which are neither revoked nor rejected (mirrors the
// partial unique index). Pure read primitive for submit-time duplicate
// detection — pending rows are invisible to the approved-gated finders.
func (r *gormCredentialRepository) CountActiveByFileHashes(ctx context.Context, hashes ...string) (int64, error) {
	if len(hashes) == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Credential{}).
		Where("file_hash IN ?", hashes).
		Where("revoked_at IS NULL").
		Where("rejected_at IS NULL").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ── Compile-time interface check ──────────────────────────────────────────

var _ domain.CredentialRepository = (*gormCredentialRepository)(nil)
