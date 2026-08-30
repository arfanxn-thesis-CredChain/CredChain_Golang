package credential

import (
	"context"
	"errors"
	"sort"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type gormIssuerOrganizationRepository struct {
	db *gorm.DB
}

func NewGormCredentialIssuerOrganizationRepository(db *gorm.DB) domain.CredentialIssuerOrganizationRepository {
	return &gormIssuerOrganizationRepository{db: db}
}

func (r *gormIssuerOrganizationRepository) Store(ctx context.Context, orgs ...domain.CredentialIssuerOrganization) ([]domain.CredentialIssuerOrganization, error) {
	if len(orgs) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	for i := range orgs {
		if orgs[i].Id == "" {
			orgs[i].Id = ulid.Make().String()
		}
	}
	rows := make([]model.CredentialIssuerOrganization, len(orgs))
	for i, o := range orgs {
		rows[i] = model.FromDomainCredentialIssuerOrganization(o)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialIssuerOrganization, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormIssuerOrganizationRepository) Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
	var m model.CredentialIssuerOrganization
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := m.ToDomain()
	return &d, nil
}

func (r *gormIssuerOrganizationRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.CredentialIssuerOrganization, error) {
	if len(ids) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	var rows []model.CredentialIssuerOrganization
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialIssuerOrganization, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// FindByNames resolves names case-insensitively in a single query. Names are
// trimmed and lowercased before matching, mirroring
// uq_credential_issuer_organizations_lower_name.
func (r *gormIssuerOrganizationRepository) FindByNames(ctx context.Context, names ...string) ([]domain.CredentialIssuerOrganization, error) {
	if len(names) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	needles := make([]string, 0, len(names))
	for _, n := range names {
		trimmed := strings.TrimSpace(n)
		if trimmed == "" {
			continue
		}
		needles = append(needles, strings.ToLower(trimmed))
	}
	if len(needles) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	var rows []model.CredentialIssuerOrganization
	if err := r.db.WithContext(ctx).
		Where("LOWER(name) IN ?", needles).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialIssuerOrganization, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// SuggestByName ranks rows by trigram similarity on Postgres. Repository tests
// run on SQLite, which has no pg_trgm, so a substring match stands in there —
// the ordering guarantee is only meaningful on Postgres.
func (r *gormIssuerOrganizationRepository) SuggestByName(ctx context.Context, name string, limit int) ([]domain.CredentialIssuerOrganization, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	db := r.db.WithContext(ctx).Model(&model.CredentialIssuerOrganization{}).Limit(limit)
	if r.db.Dialector.Name() == "postgres" {
		db = db.Where("similarity(name, ?) > 0.1", trimmed).
			Order(gorm.Expr("similarity(name, ?) DESC", trimmed))
	} else {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(trimmed)+"%").
			Order("LENGTH(name) ASC")
	}

	var rows []model.CredentialIssuerOrganization
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialIssuerOrganization, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// allowedIssuerOrganizationFilterColumns whitelists columns clients may filter
// on. "id" is included so the UI can resolve a specific set of organizations by
// id (?filters=id$a,b,c) — selected chips may live outside the loaded page.
var allowedIssuerOrganizationFilterColumns = map[string]bool{
	"id":         true,
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

var allowedIssuerOrganizationSortColumns = map[string]bool{
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

// Get retrieves issuer organizations with pagination, search, filters, and
// sorts. Returns: ([]CredentialIssuerOrganization, int, error) — the page, the
// total matching the criteria before pagination, and an error.
func (r *gormIssuerOrganizationRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error) {
	db := r.db.WithContext(ctx).Model(&model.CredentialIssuerOrganization{})

	if query != nil {
		if query.HasSearch() {
			db = db.Where("LOWER(name) LIKE LOWER(?)", "%"+query.Search+"%")
		}
		if query.HasFilters() {
			db = gormhelpers.ApplyFilters(db, query.Filters, allowedIssuerOrganizationFilterColumns, "")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedIssuerOrganizationSortColumns, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
	var rows []model.CredentialIssuerOrganization
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CredentialIssuerOrganization, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, int(total), nil
}

func (r *gormIssuerOrganizationRepository) Update(ctx context.Context, orgs ...domain.CredentialIssuerOrganization) ([]domain.CredentialIssuerOrganization, error) {
	if len(orgs) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	sort.Slice(orgs, func(i, j int) bool { return orgs[i].Id < orgs[j].Id })

	var clauses []string
	var allArgs [][]interface{}
	addCol := func(col string, getValue func(domain.CredentialIssuerOrganization) (interface{}, bool)) {
		var pairs []interface{}
		for _, o := range orgs {
			if v, ok := getValue(o); ok {
				pairs = append(pairs, o.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}
	addCol("name", func(o domain.CredentialIssuerOrganization) (interface{}, bool) {
		if o.Name != "" {
			return o.Name, true
		}
		return nil, false
	})
	// Active is a bool without a pointer; callers toggling it include the
	// field explicitly, so always emit the CASE branch for update calls.
	addCol("active", func(o domain.CredentialIssuerOrganization) (interface{}, bool) {
		return o.Active, true
	})
	if len(clauses) == 0 {
		return []domain.CredentialIssuerOrganization{}, nil
	}
	ids := make([]interface{}, len(orgs))
	for i, o := range orgs {
		ids[i] = o.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("credential_issuer_organizations", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	if err := r.db.WithContext(ctx).Exec(sql, finalArgs...).Error; err != nil {
		return nil, err
	}
	idStrs := make([]string, len(orgs))
	for i, o := range orgs {
		idStrs[i] = o.Id
	}
	return r.FindByIds(ctx, idStrs...)
}

// Destroy hard-deletes rows by ID (batch). Referenced rows are rejected by the
// database FK (23503) — translated to CodeCredentialIssuerOrganizationDestroyInUse.
// The reference pre-check belongs to the step-3 service; this method is a pure
// primitive.
func (r *gormIssuerOrganizationRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.CredentialIssuerOrganization{}, "id IN ?", ids)
	if err := result.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, domain.NewError(domain.CodeCredentialIssuerOrganizationDestroyInUse,
				domain.WithMetadata("ids", ids), domain.WithError(err))
		}
		return 0, err
	}
	return result.RowsAffected, nil
}

var _ domain.CredentialIssuerOrganizationRepository = (*gormIssuerOrganizationRepository)(nil)
