package overview

import (
	"context"
	"time"

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
				 WHERE revoked_at IS NULL AND rejected_at IS NULL AND activated_at IS NOT NULL AND (expires_at IS NULL OR expires_at > ?)
				 AND issued_at BETWEEN ? AND ?) AS active,
				(SELECT COUNT(*) FROM credentials
				 WHERE revoked_at IS NOT NULL
				 AND issued_at BETWEEN ? AND ?) AS revoked,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_enqueued_at IS NOT NULL AND extracted_at IS NULL AND extract_failed_at IS NULL AND issued_at BETWEEN ? AND ?) AS pending,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_failed_at IS NOT NULL AND issued_at BETWEEN ? AND ?) AS failed
		`, dateFrom, dateTo, time.Now(), dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo).Scan(&result).Error
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
			 WHERE holder_user_id = ? AND revoked_at IS NULL AND rejected_at IS NULL AND activated_at IS NOT NULL AND (expires_at IS NULL OR expires_at > ?)
			 AND issued_at BETWEEN ? AND ?) AS active,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND revoked_at IS NOT NULL
			 AND issued_at BETWEEN ? AND ?) AS revoked,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_enqueued_at IS NOT NULL AND extracted_at IS NULL AND extract_failed_at IS NULL AND issued_at BETWEEN ? AND ?) AS pending,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_failed_at IS NOT NULL AND issued_at BETWEEN ? AND ?) AS failed
	`,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, time.Now(), dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &domain.OverviewCredentialCounts{Total: result.Total, Active: result.Active, Revoked: result.Revoked, Pending: result.Pending, Failed: result.Failed}, nil
}

func (r *gormOverviewRepository) CredentialRecap(ctx context.Context, asOf time.Time) ([]domain.CredentialRecapRow, error) {
	rows := make([]domain.CredentialRecapRow, 0)
	// Aggregate per owner before joining holders so zero-credential holders remain
	// present and each holder contributes exactly once to its current unit/year.
	err := r.db.WithContext(ctx).Raw(`
  WITH credential_counts AS (
   SELECT c.holder_user_id,
    COUNT(*) AS total_credentials,
    SUM(CASE WHEN c.revoked_at IS NOT NULL THEN 1 ELSE 0 END) AS revoked_credentials,
    SUM(CASE WHEN c.revoked_at IS NULL AND c.rejected_at IS NOT NULL THEN 1 ELSE 0 END) AS rejected_credentials,
    SUM(CASE WHEN c.revoked_at IS NULL AND c.rejected_at IS NULL AND c.expires_at IS NOT NULL AND c.expires_at <= ? THEN 1 ELSE 0 END) AS expired_credentials,
    SUM(CASE WHEN c.revoked_at IS NULL AND c.rejected_at IS NULL AND (c.expires_at IS NULL OR c.expires_at > ?) AND c.activated_at IS NOT NULL THEN 1 ELSE 0 END) AS active_credentials,
    SUM(CASE WHEN c.revoked_at IS NULL AND c.rejected_at IS NULL AND (c.expires_at IS NULL OR c.expires_at > ?) AND c.activated_at IS NULL THEN 1 ELSE 0 END) AS pending_credentials
   FROM credentials c GROUP BY c.holder_user_id
  )
  SELECT u.unit_id, u.joined_year,
   COUNT(*) AS total_holders,
   SUM(CASE WHEN cc.total_credentials IS NOT NULL THEN 1 ELSE 0 END) AS holders_with_credentials,
   SUM(CASE WHEN cc.total_credentials IS NULL THEN 1 ELSE 0 END) AS holders_without_credentials,
   SUM(CASE WHEN COALESCE(cc.active_credentials, 0) > 0 THEN 1 ELSE 0 END) AS holders_with_active_credentials,
   SUM(CASE WHEN COALESCE(cc.active_credentials, 0) = 0 THEN 1 ELSE 0 END) AS holders_without_active_credentials,
   SUM(COALESCE(cc.total_credentials, 0)) AS total_credentials,
   SUM(COALESCE(cc.active_credentials, 0)) AS active_credentials,
   SUM(COALESCE(cc.pending_credentials, 0)) AS pending_credentials,
   SUM(COALESCE(cc.rejected_credentials, 0)) AS rejected_credentials,
   SUM(COALESCE(cc.expired_credentials, 0)) AS expired_credentials,
   SUM(COALESCE(cc.revoked_credentials, 0)) AS revoked_credentials
  FROM users u LEFT JOIN credential_counts cc ON cc.holder_user_id = u.id
  WHERE u.role = 'holder' AND u.deleted_at IS NULL
  GROUP BY u.unit_id, u.joined_year
  ORDER BY u.unit_id, u.joined_year
 `, asOf, asOf, asOf).Scan(&rows).Error
	return rows, err
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
