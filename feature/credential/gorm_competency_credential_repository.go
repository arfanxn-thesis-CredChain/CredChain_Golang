package credential

import (
	"context"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"gorm.io/gorm"
)

type gormCompetencyCredentialRepository struct {
	db *gorm.DB
}

func NewGormCompetencyCredentialRepository(db *gorm.DB) domain.CompetencyCredentialRepository {
	return &gormCompetencyCredentialRepository{db: db}
}

func (r *gormCompetencyCredentialRepository) Store(ctx context.Context, links ...domain.CompetencyCredential) ([]domain.CompetencyCredential, error) {
	if len(links) == 0 {
		return []domain.CompetencyCredential{}, nil
	}
	rows := make([]model.CompetencyCredential, len(links))
	for i, l := range links {
		rows[i] = model.FromDomainCompetencyCredential(l)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CompetencyCredential, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormCompetencyCredentialRepository) Destroy(ctx context.Context, links ...domain.CompetencyCredential) (int64, error) {
	if len(links) == 0 {
		return 0, nil
	}
	rows := make([]model.CompetencyCredential, len(links))
	for i, l := range links {
		rows[i] = model.FromDomainCompetencyCredential(l)
	}
	result := r.db.WithContext(ctx).Delete(&rows)
	return result.RowsAffected, result.Error
}

func (r *gormCompetencyCredentialRepository) FindByCredentialId(ctx context.Context, credentialId string) ([]domain.CompetencyCredential, error) {
	var rows []model.CompetencyCredential
	if err := r.db.WithContext(ctx).Where("credential_id = ?", credentialId).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CompetencyCredential, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

func (r *gormCompetencyCredentialRepository) FindByCompetencyId(ctx context.Context, competencyId string) ([]domain.CompetencyCredential, error) {
	var rows []model.CompetencyCredential
	if err := r.db.WithContext(ctx).Where("competency_id = ?", competencyId).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CompetencyCredential, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// CountByCompetencyIds counts join rows referencing any of the given
// competency ids.
func (r *gormCompetencyCredentialRepository) CountByCompetencyIds(ctx context.Context, competencyIds ...string) (int64, error) {
	if len(competencyIds) == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.CompetencyCredential{}).Where("competency_id IN ?", competencyIds).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

var _ domain.CompetencyCredentialRepository = (*gormCompetencyCredentialRepository)(nil)
