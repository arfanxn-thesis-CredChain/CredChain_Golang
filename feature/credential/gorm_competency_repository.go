package credential

import (
	"context"
	"sort"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

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

func (r *gormCompetencyRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, error) {
	db := r.db.WithContext(ctx).Model(&model.Competency{})
	db = gormhelpers.ApplySorts(db, query, nil, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
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

// Delete hard-deletes rows by ID (batch). Rows referenced by credentials are
// protected by the FK constraint; reference pre-checks belong to the service
// layer (step 3). Unconsumed plumbing in this step.
func (r *gormCompetencyRepository) Delete(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.Competency{}, "id IN ?", ids)
	return result.RowsAffected, result.Error
}

var _ domain.CompetencyRepository = (*gormCompetencyRepository)(nil)
