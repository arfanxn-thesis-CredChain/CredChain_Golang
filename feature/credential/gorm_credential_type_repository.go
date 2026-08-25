package credential

import (
	"context"
	"errors"
	"sort"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type gormCredentialTypeRepository struct {
	db *gorm.DB
}

func NewGormCredentialTypeRepository(db *gorm.DB) domain.CredentialTypeRepository {
	return &gormCredentialTypeRepository{db: db}
}

func (r *gormCredentialTypeRepository) Store(ctx context.Context, types ...domain.CredentialType) ([]domain.CredentialType, error) {
	if len(types) == 0 {
		return []domain.CredentialType{}, nil
	}
	for i := range types {
		if types[i].Id == "" {
			types[i].Id = ulid.Make().String()
		}
	}
	rows := make([]model.CredentialType, len(types))
	for i, t := range types {
		rows[i] = model.FromDomainCredentialType(t)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialType, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormCredentialTypeRepository) Find(ctx context.Context, id string) (*domain.CredentialType, error) {
	var m model.CredentialType
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := m.ToDomain()
	return &d, nil
}

func (r *gormCredentialTypeRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.CredentialType, error) {
	if len(ids) == 0 {
		return []domain.CredentialType{}, nil
	}
	var rows []model.CredentialType
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CredentialType, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// allowedCredentialTypeFilterColumns whitelists columns clients may filter on.
// "id" is included so the UI can resolve a specific set of types by id
// (?filters=id$a,b,c) — selected chips may live outside the loaded page.
var allowedCredentialTypeFilterColumns = map[string]bool{
	"id":         true,
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

var allowedCredentialTypeSortColumns = map[string]bool{
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

// Get retrieves credential types with pagination, search, filters, and sorts.
// Returns: ([]CredentialType, int, error) — the page, the total matching the
// criteria before pagination, and an error.
func (r *gormCredentialTypeRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, int, error) {
	db := r.db.WithContext(ctx).Model(&model.CredentialType{})

	if query != nil {
		if query.HasSearch() {
			db = db.Where("LOWER(name) LIKE LOWER(?)", "%"+query.Search+"%")
		}
		if query.HasFilters() {
			db = gormhelpers.ApplyFilters(db, query.Filters, allowedCredentialTypeFilterColumns, "")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedCredentialTypeSortColumns, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
	var rows []model.CredentialType
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CredentialType, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, int(total), nil
}

func (r *gormCredentialTypeRepository) Update(ctx context.Context, types ...domain.CredentialType) ([]domain.CredentialType, error) {
	if len(types) == 0 {
		return []domain.CredentialType{}, nil
	}
	sort.Slice(types, func(i, j int) bool { return types[i].Id < types[j].Id })

	var clauses []string
	var allArgs [][]interface{}
	addCol := func(col string, getValue func(domain.CredentialType) (interface{}, bool)) {
		var pairs []interface{}
		for _, t := range types {
			if v, ok := getValue(t); ok {
				pairs = append(pairs, t.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}
	addCol("name", func(t domain.CredentialType) (interface{}, bool) {
		if t.Name != "" {
			return t.Name, true
		}
		return nil, false
	})
	// Active is a bool without a pointer; callers toggling it include the
	// field explicitly, so always emit the CASE branch for update calls.
	addCol("active", func(t domain.CredentialType) (interface{}, bool) {
		return t.Active, true
	})
	if len(clauses) == 0 {
		return []domain.CredentialType{}, nil
	}
	ids := make([]interface{}, len(types))
	for i, t := range types {
		ids[i] = t.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("credential_types", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	if err := r.db.WithContext(ctx).Exec(sql, finalArgs...).Error; err != nil {
		return nil, err
	}
	idStrs := make([]string, len(types))
	for i, t := range types {
		idStrs[i] = t.Id
	}
	return r.FindByIds(ctx, idStrs...)
}

// Destroy hard-deletes rows by ID (batch). Referenced rows are rejected by the
// database FK (23503) — translated to CodeCredentialTypeDestroyInUse. The
// reference pre-check belongs to the step-3 service; this method is a pure
// primitive. Setting active = false via Update is the intended everyday
// deactivation path; hard deletion is the exception.
func (r *gormCredentialTypeRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.CredentialType{}, "id IN ?", ids)
	if err := result.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, domain.NewError(domain.CodeCredentialTypeDestroyInUse,
				domain.WithMetadata("ids", ids), domain.WithError(err))
		}
		return 0, err
	}
	return result.RowsAffected, nil
}

var _ domain.CredentialTypeRepository = (*gormCredentialTypeRepository)(nil)
