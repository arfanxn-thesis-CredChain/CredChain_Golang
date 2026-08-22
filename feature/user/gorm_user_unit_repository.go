package user

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

type gormUserUnitRepository struct {
	db *gorm.DB
}

func NewGormUserUnitRepository(db *gorm.DB) domain.UserUnitRepository {
	return &gormUserUnitRepository{db: db}
}

func (r *gormUserUnitRepository) Store(ctx context.Context, units ...domain.UserUnit) ([]domain.UserUnit, error) {
	if len(units) == 0 {
		return []domain.UserUnit{}, nil
	}
	for i := range units {
		if units[i].Id == "" {
			units[i].Id = ulid.Make().String()
		}
	}
	rows := make([]model.UserUnit, len(units))
	for i, u := range units {
		rows[i] = model.FromDomainUserUnit(u)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserUnit, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormUserUnitRepository) Find(ctx context.Context, id string) (*domain.UserUnit, error) {
	var m model.UserUnit
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := m.ToDomain()
	return &d, nil
}

func (r *gormUserUnitRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.UserUnit, error) {
	if len(ids) == 0 {
		return []domain.UserUnit{}, nil
	}
	var rows []model.UserUnit
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserUnit, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormUserUnitRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.UserUnit, error) {
	db := r.db.WithContext(ctx).Model(&model.UserUnit{})
	db = gormhelpers.ApplySorts(db, query, nil, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
	var rows []model.UserUnit
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserUnit, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormUserUnitRepository) Update(ctx context.Context, units ...domain.UserUnit) ([]domain.UserUnit, error) {
	if len(units) == 0 {
		return []domain.UserUnit{}, nil
	}
	sort.Slice(units, func(i, j int) bool { return units[i].Id < units[j].Id })

	var clauses []string
	var allArgs [][]interface{}
	addCol := func(col string, getValue func(domain.UserUnit) (interface{}, bool)) {
		var pairs []interface{}
		for _, u := range units {
			if v, ok := getValue(u); ok {
				pairs = append(pairs, u.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}
	addCol("name", func(u domain.UserUnit) (interface{}, bool) {
		if u.Name != "" {
			return u.Name, true
		}
		return nil, false
	})
	addCol("parent_id", func(u domain.UserUnit) (interface{}, bool) {
		if u.ParentId != nil {
			return *u.ParentId, true
		}
		return nil, false
	})
	if len(clauses) == 0 {
		return []domain.UserUnit{}, nil
	}
	ids := make([]interface{}, len(units))
	for i, u := range units {
		ids[i] = u.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("user_units", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	if err := r.db.WithContext(ctx).Exec(sql, finalArgs...).Error; err != nil {
		return nil, err
	}
	idStrs := make([]string, len(units))
	for i, u := range units {
		idStrs[i] = u.Id
	}
	return r.FindByIds(ctx, idStrs...)
}

// FindWithDescendants returns the unit with the given id plus all of its
// descendants using a single WITH RECURSIVE query. Supported by both Postgres
// and SQLite (test dialect). The unit tree is assumed acyclic.
func (r *gormUserUnitRepository) FindWithDescendants(ctx context.Context, id string) ([]domain.UserUnit, error) {
	const sql = `
WITH RECURSIVE descendants AS (
	SELECT id, parent_id, name, created_at, updated_at FROM user_units WHERE id = ?
	UNION ALL
	SELECT uu.id, uu.parent_id, uu.name, uu.created_at, uu.updated_at
	FROM user_units uu
	JOIN descendants d ON uu.parent_id = d.id
)
SELECT id, parent_id, name, created_at, updated_at FROM descendants`

	var rows []model.UserUnit
	if err := r.db.WithContext(ctx).Raw(sql, id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserUnit, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Destroy hard-deletes rows by ID (batch). Referenced rows are rejected by the
// database FK (23503) — translated to CodeUserUnitDestroyInUse. The reference
// pre-check belongs to the step-3 service; this method is a pure primitive.
func (r *gormUserUnitRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.UserUnit{}, "id IN ?", ids)
	if err := result.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, domain.NewError(domain.CodeUserUnitDestroyInUse,
				domain.WithMetadata("ids", ids), domain.WithError(err))
		}
		return 0, err
	}
	return result.RowsAffected, nil
}

// CountByParentIds counts units whose parent_id is any of the given ids.
func (r *gormUserUnitRepository) CountByParentIds(ctx context.Context, parentIds ...string) (int64, error) {
	if len(parentIds) == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.UserUnit{}).Where("parent_id IN ?", parentIds).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateParent sets a single unit's parent_id, writing SQL NULL when parentId is
// nil (promote to root). Kept separate from the batch Update because that CASE
// builder skips a nil parent_id — it can't tell "clear" from "leave untouched".
func (r *gormUserUnitRepository) UpdateParent(ctx context.Context, id string, parentId *string) error {
	var value interface{}
	if parentId != nil {
		value = *parentId
	}
	return r.db.WithContext(ctx).Model(&model.UserUnit{}).
		Where("id = ?", id).
		Update("parent_id", value).Error
}

var _ domain.UserUnitRepository = (*gormUserUnitRepository)(nil)
