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

func (r *gormIssuerOrganizationRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, error) {
	db := r.db.WithContext(ctx).Model(&model.CredentialIssuerOrganization{})
	db = gormhelpers.ApplySorts(db, query, nil, "name ASC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)
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

// Delete hard-deletes rows by ID (batch). Referenced rows are rejected by the
// database FK (23503) — translated to CodeCredentialIssuerOrganizationDeleteInUse.
// The reference pre-check belongs to the step-3 service; this method is a pure
// primitive.
func (r *gormIssuerOrganizationRepository) Delete(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Delete(&model.CredentialIssuerOrganization{}, "id IN ?", ids)
	if err := result.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, domain.NewError(domain.CodeCredentialIssuerOrganizationDeleteInUse,
				domain.WithMetadata("ids", ids), domain.WithError(err))
		}
		return 0, err
	}
	return result.RowsAffected, nil
}

var _ domain.CredentialIssuerOrganizationRepository = (*gormIssuerOrganizationRepository)(nil)
