package overview

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"gorm.io/gorm"
)

type gormOverviewRepository struct {
	db *gorm.DB
}

func NewGormOverviewRepository(db *gorm.DB) domain.OverviewRepository {
	return &gormOverviewRepository{db: db}
}

func (r *gormOverviewRepository) CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*domain.OverviewCredentialCounts, error) {
	dateFrom, dateTo := extractDateRange(q)

	type row struct{ Total, Active, Revoked, Pending, Failed int }

	if holderUserID == nil {
		var result row
		err := r.db.WithContext(ctx).Raw(`
			SELECT
				(SELECT COUNT(*) FROM credentials WHERE issued_at BETWEEN ? AND ?) AS total,
				(SELECT COUNT(*) FROM credentials
				 WHERE revoked_at IS NULL AND extract_enqueued_at IS NOT NULL AND extract_failed_at IS NULL
				 AND issued_at BETWEEN ? AND ?) AS active,
				(SELECT COUNT(*) FROM credentials
				 WHERE revoked_at IS NOT NULL AND extract_enqueued_at IS NOT NULL AND extract_failed_at IS NULL
				 AND issued_at BETWEEN ? AND ?) AS revoked,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_enqueued_at IS NOT NULL AND extracted_at IS NULL AND extract_failed_at IS NULL AND issued_at BETWEEN ? AND ?) AS pending,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_failed_at IS NOT NULL AND issued_at BETWEEN ? AND ?) AS failed
		`, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo).Scan(&result).Error
		if err != nil {
			return nil, err
		}
		return &domain.OverviewCredentialCounts{Total: result.Total, Active: result.Active, Revoked: result.Revoked, Pending: result.Pending, Failed: result.Failed}, nil
	}

	var result row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM credentials WHERE holder_user_id = ? AND issued_at BETWEEN ? AND ?) AS total,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND revoked_at IS NULL AND extract_enqueued_at IS NOT NULL AND extract_failed_at IS NULL
			 AND issued_at BETWEEN ? AND ?) AS active,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND revoked_at IS NOT NULL AND extract_enqueued_at IS NOT NULL AND extract_failed_at IS NULL
			 AND issued_at BETWEEN ? AND ?) AS revoked,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_enqueued_at IS NOT NULL AND extracted_at IS NULL AND extract_failed_at IS NULL AND issued_at BETWEEN ? AND ?) AS pending,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_failed_at IS NOT NULL AND issued_at BETWEEN ? AND ?) AS failed
	`,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &domain.OverviewCredentialCounts{Total: result.Total, Active: result.Active, Revoked: result.Revoked, Pending: result.Pending, Failed: result.Failed}, nil
}

func (r *gormOverviewRepository) UserCounts(ctx context.Context, q *domainQuery.Query) (*domain.OverviewUserCounts, error) {
	dateFrom, dateTo := extractDateRange(q)

	type row struct{ Total, Holder, Issuer, Admin, SuperAdmin, Active, Trashed int }

	var result row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM users WHERE created_at BETWEEN ? AND ?) AS total,
			(SELECT COUNT(*) FROM users WHERE role = 'holder' AND created_at BETWEEN ? AND ?) AS holder,
			(SELECT COUNT(*) FROM users WHERE role = 'issuer' AND created_at BETWEEN ? AND ?) AS issuer,
			(SELECT COUNT(*) FROM users WHERE role = 'admin' AND created_at BETWEEN ? AND ?) AS admin,
			(SELECT COUNT(*) FROM users WHERE role = 'super_admin' AND created_at BETWEEN ? AND ?) AS super_admin,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at BETWEEN ? AND ?) AS active,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NOT NULL AND created_at BETWEEN ? AND ?) AS trashed
	`,
		dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo,
		dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &domain.OverviewUserCounts{Total: result.Total, Holder: result.Holder, Issuer: result.Issuer, Admin: result.Admin, SuperAdmin: result.SuperAdmin, Active: result.Active, Trashed: result.Trashed}, nil
}
