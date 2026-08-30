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

type gormCompetencyRepository struct {
	db *gorm.DB
}

func NewGormCompetencyRepository(db *gorm.DB) domain.CompetencyRepository {
	return &gormCompetencyRepository{db: db}
}

func (r *gormCompetencyRepository) Store(ctx context.Context, competencies ...domain.Competency) ([]domain.Competency, error) {
	if len(competencies) == 0 {
		return []domain.Competency{}, nil
	}
	for i := range competencies {
		if competencies[i].Id == "" {
			competencies[i].Id = ulid.Make().String()
		}
	}
	rows := make([]model.Competency, len(competencies))
	for i, c := range competencies {
		rows[i] = model.FromDomainCompetency(c)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Competency, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormCompetencyRepository) Find(ctx context.Context, id string) (*domain.Competency, error) {
	var m model.Competency
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := m.ToDomain()
	return &d, nil
}

func (r *gormCompetencyRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.Competency, error) {
	if len(ids) == 0 {
		return []domain.Competency{}, nil
	}
	var rows []model.Competency
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Competency, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// FindByNames resolves names case-insensitively in a single query. Names are
// trimmed and lowercased before matching, mirroring uq_competencies_lower_name.
func (r *gormCompetencyRepository) FindByNames(ctx context.Context, names ...string) ([]domain.Competency, error) {
	if len(names) == 0 {
		return []domain.Competency{}, nil
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
		return []domain.Competency{}, nil
	}
	var rows []model.Competency
	if err := r.db.WithContext(ctx).
		Where("LOWER(name) IN ?", needles).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Competency, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// SuggestByName ranks rows by trigram similarity on Postgres. Repository tests
// run on SQLite, which has no pg_trgm, so a substring match stands in there —
// the ordering guarantee is only meaningful on Postgres.
func (r *gormCompetencyRepository) SuggestByName(ctx context.Context, name string, limit int) ([]domain.Competency, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return []domain.Competency{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	db := r.db.WithContext(ctx).Model(&model.Competency{}).Limit(limit)
	if r.db.Dialector.Name() == "postgres" {
		db = db.Where("similarity(name, ?) > 0.1", trimmed).
			Order(gorm.Expr("similarity(name, ?) DESC", trimmed))
	} else {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(trimmed)+"%").
			Order("LENGTH(name) ASC")
	}

	var rows []model.Competency
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Competency, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// allowedCompetencyFilterColumns whitelists columns clients may filter on. "id"
// is included so the UI can resolve a specific set of competencies by id
// (?filters=id$a,b,c) — selected chips may live outside the loaded page.
var allowedCompetencyFilterColumns = map[string]bool{
	"id":         true,
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

var allowedCompetencySortColumns = map[string]bool{
	"name":       true,
	"active":     true,
	"created_at": true,
	"updated_at": true,
}

// Get retrieves competencies with pagination, search, filters, and sorts.
// Returns: ([]Competency, int, error) — the page, the total matching the
// criteria before pagination, and an error.
func (r *gormCompetencyRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Competency{})

	if query != nil {
		if query.HasSearch() {
			db = db.Where("LOWER(name) LIKE LOWER(?)", "%"+query.Search+"%")
		}
		if query.HasFilters() {
			db = gormhelpers.ApplyFilters(db, query.Filters, allowedCompetencyFilterColumns, "")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedCompetencySortColumns, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
	var rows []model.Competency
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Competency, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, int(total), nil
}

func (r *gormCompetencyRepository) Update(ctx context.Context, competencies ...domain.Competency) ([]domain.Competency, error) {
	if len(competencies) == 0 {
		return []domain.Competency{}, nil
	}
	sort.Slice(competencies, func(i, j int) bool { return competencies[i].Id < competencies[j].Id })

	var clauses []string
	var allArgs [][]interface{}
	addCol := func(col string, getValue func(domain.Competency) (interface{}, bool)) {
		var pairs []interface{}
		for _, c := range competencies {
			if v, ok := getValue(c); ok {
				pairs = append(pairs, c.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}
	addCol("name", func(c domain.Competency) (interface{}, bool) {
		if c.Name != "" {
			return c.Name, true
		}
		return nil, false
	})
	// Active is a bool without a pointer; callers toggling it include the
	// field explicitly, so always emit the CASE branch for update calls.
	addCol("active", func(c domain.Competency) (interface{}, bool) {
		return c.Active, true
	})
	if len(clauses) == 0 {
		return []domain.Competency{}, nil
	}
	ids := make([]interface{}, len(competencies))
	for i, c := range competencies {
		ids[i] = c.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("competencies", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	if err := r.db.WithContext(ctx).Exec(sql, finalArgs...).Error; err != nil {
		return nil, err
	}
	idStrs := make([]string, len(competencies))
	for i, c := range competencies {
		idStrs[i] = c.Id
	}
	return r.FindByIds(ctx, idStrs...)
}

// Destroy hard-deletes rows by ID (batch). Referenced rows are rejected by the
// database FK (23503) — translated to CodeCompetencyDestroyInUse. The reference
// pre-check belongs to the step-3 service; this method is a pure primitive.
func (r *gormCompetencyRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.Competency{}, "id IN ?", ids)
	if err := result.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, domain.NewError(domain.CodeCompetencyDestroyInUse,
				domain.WithMetadata("ids", ids), domain.WithError(err))
		}
		return 0, err
	}
	return result.RowsAffected, nil
}

var _ domain.CompetencyRepository = (*gormCompetencyRepository)(nil)
