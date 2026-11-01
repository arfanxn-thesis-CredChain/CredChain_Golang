package credential

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormCredentialNotificationRepository struct {
	db *gorm.DB
}

// NewGormCredentialNotificationRepository creates a new GORM-backed credential notification repository.
func NewGormCredentialNotificationRepository(db *gorm.DB) domain.CredentialNotificationRepository {
	return &gormCredentialNotificationRepository{db: db}
}

func (r *gormCredentialNotificationRepository) Store(ctx context.Context, notifications ...domain.CredentialNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	models := make([]model.CredentialNotification, len(notifications))
	for i, n := range notifications {
		if n.ID == "" {
			n.ID = ulid.Make().String()
		} else {
			if _, err := ulid.ParseStrict(n.ID); err != nil {
				return domain.NewError(domain.CodeSystemValidation, domain.WithError(err))
			}
		}
		models[i] = model.FromDomainCredentialNotification(n)
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "notified_user_id"},
				{Name: "credential_id"},
				{Name: "type"},
			},
			DoNothing: true,
		}).
		Create(&models).Error
}

func (r *gormCredentialNotificationRepository) List(
	ctx context.Context,
	notifiedUserID string,
	options domain.CredentialNotificationListOptions,
) (domain.CredentialNotificationPage, error) {
	status := options.Status
	if status == "" {
		status = domain.CredentialNotificationStatusAll
	}
	if status != domain.CredentialNotificationStatusAll && status != domain.CredentialNotificationStatusUnread {
		return domain.CredentialNotificationPage{}, domain.NewError(
			domain.CodeSystemValidation,
			domain.WithMetadata("field", "status"),
		)
	}

	limit := options.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return domain.CredentialNotificationPage{}, domain.NewError(
			domain.CodeSystemValidation,
			domain.WithMetadata("field", "limit"),
		)
	}

	var cursorTime time.Time
	var cursorID string
	if options.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(options.Cursor)
		if err != nil {
			return domain.CredentialNotificationPage{}, domain.NewError(
				domain.CodeSystemValidation,
				domain.WithMetadata("field", "cursor"),
			)
		}
		parts := strings.Split(string(decoded), "|")
		if len(parts) != 2 {
			return domain.CredentialNotificationPage{}, domain.NewError(
				domain.CodeSystemValidation,
				domain.WithMetadata("field", "cursor"),
			)
		}
		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return domain.CredentialNotificationPage{}, domain.NewError(
				domain.CodeSystemValidation,
				domain.WithMetadata("field", "cursor"),
			)
		}
		if _, err := ulid.ParseStrict(parts[1]); err != nil {
			return domain.CredentialNotificationPage{}, domain.NewError(
				domain.CodeSystemValidation,
				domain.WithMetadata("field", "cursor"),
			)
		}
		cursorID = parts[1]
	}

	baseQuery := r.db.WithContext(ctx).
		Model(&model.CredentialNotification{}).
		Where("notified_user_id = ?", notifiedUserID)

	if status == domain.CredentialNotificationStatusUnread {
		baseQuery = baseQuery.Where("read_at IS NULL")
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return domain.CredentialNotificationPage{}, err
	}

	dataQuery := r.db.WithContext(ctx).
		Model(&model.CredentialNotification{}).
		Where("notified_user_id = ?", notifiedUserID)

	if status == domain.CredentialNotificationStatusUnread {
		dataQuery = dataQuery.Where("read_at IS NULL")
	}

	if !cursorTime.IsZero() && cursorID != "" {
		dataQuery = dataQuery.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}

	var rows []model.CredentialNotification
	if err := dataQuery.
		Order("created_at DESC, id DESC").
		Limit(limit + 1).
		Find(&rows).Error; err != nil {
		return domain.CredentialNotificationPage{}, err
	}

	var nextCursor *string
	if len(rows) > limit {
		last := rows[limit-1]
		val := base64.RawURLEncoding.EncodeToString(
			fmt.Appendf(nil, "%s|%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID),
		)
		nextCursor = &val
		rows = rows[:limit]
	}

	items := make([]domain.CredentialNotification, len(rows))
	credIDs := make([]string, 0, len(rows))
	for i, m := range rows {
		items[i] = m.ToDomain()
		credIDs = append(credIDs, m.CredentialID)
	}

	if len(credIDs) > 0 {
		var credRows []model.Credential
		err := r.db.WithContext(ctx).
			Model(&model.Credential{}).
			Select("id", "holder_user_id", "name", "number").
			Where("id IN (?) AND holder_user_id = ?", credIDs, notifiedUserID).
			Find(&credRows).Error
		if err != nil {
			return domain.CredentialNotificationPage{}, err
		}

		credMap := make(map[string]domain.Credential, len(credRows))
		for _, c := range credRows {
			credMap[c.Id] = domain.Credential{
				ID:           c.Id,
				HolderUserID: c.HolderUserId,
				Name:         c.Name,
				Number:       c.Number,
			}
		}

		for i := range items {
			if c, ok := credMap[items[i].CredentialID]; ok {
				cCopy := c
				items[i].Credential = &cCopy
			}
		}
	}

	return domain.CredentialNotificationPage{
		Items:      items,
		Total:      total,
		NextCursor: nextCursor,
	}, nil
}

func (r *gormCredentialNotificationRepository) UnreadCount(ctx context.Context, notifiedUserID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CredentialNotification{}).
		Where("notified_user_id = ? AND read_at IS NULL", notifiedUserID).
		Count(&count).Error
	return count, err
}

func (r *gormCredentialNotificationRepository) MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.CredentialNotification, error) {
	stmtTime := gorm.Expr("CURRENT_TIMESTAMP")
	if r.db.Dialector.Name() == "postgres" {
		stmtTime = gorm.Expr("statement_timestamp()")
	}

	res := r.db.WithContext(ctx).
		Model(&model.CredentialNotification{}).
		Where("id = ? AND notified_user_id = ? AND read_at IS NULL", id, notifiedUserID).
		Update("read_at", stmtTime)
	if res.Error != nil {
		return nil, res.Error
	}

	var row model.CredentialNotification
	err := r.db.WithContext(ctx).
		Where("id = ? AND notified_user_id = ?", id, notifiedUserID).
		First(&row).Error
	if err != nil {
		return nil, err
	}

	domainNotif := row.ToDomain()

	var cred model.Credential
	err = r.db.WithContext(ctx).
		Select("id", "holder_user_id", "name", "number").
		Where("id = ? AND holder_user_id = ?", row.CredentialID, notifiedUserID).
		First(&cred).Error
	if err == nil {
		domainNotif.Credential = &domain.Credential{
			ID:           cred.Id,
			HolderUserID: cred.HolderUserId,
			Name:         cred.Name,
			Number:       cred.Number,
		}
	}

	return &domainNotif, nil
}

func (r *gormCredentialNotificationRepository) MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error) {
	stmtTime := gorm.Expr("CURRENT_TIMESTAMP")
	if r.db.Dialector.Name() == "postgres" {
		stmtTime = gorm.Expr("statement_timestamp()")
	}

	res := r.db.WithContext(ctx).
		Model(&model.CredentialNotification{}).
		Where("notified_user_id = ? AND read_at IS NULL", notifiedUserID).
		Update("read_at", stmtTime)
	return res.RowsAffected, res.Error
}

var _ domain.CredentialNotificationRepository = (*gormCredentialNotificationRepository)(nil)
