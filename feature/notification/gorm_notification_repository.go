package notification

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

type gormNotificationRepository struct{ db *gorm.DB }

func NewGormNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &gormNotificationRepository{db: db}
}

func (r *gormNotificationRepository) Store(ctx context.Context, notifications ...domain.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	rows := make([]model.Notification, len(notifications))
	for i, n := range notifications {
		if err := n.Validate(); err != nil {
			return err
		}
		if n.ID == "" {
			n.ID = ulid.Make().String()
		} else if _, err := ulid.ParseStrict(n.ID); err != nil {
			return domain.NewError(domain.CodeSystemValidation, domain.WithError(err))
		}
		rows[i] = model.FromDomainNotification(n)
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "notified_user_id"}, {Name: "subject_type"}, {Name: "subject_id"}, {Name: "type"}}, DoNothing: true}).Create(&rows).Error
}

func (r *gormNotificationRepository) scoped(ctx context.Context, scope domain.NotificationScope) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.Notification{}).Where("notified_user_id = ?", scope.UserID)
	if !scope.CanReview {
		q = q.Where("type NOT IN ?", []domain.NotificationType{domain.NotificationTypeCredentialTypeSubmitted, domain.NotificationTypeCredentialIssuerOrganizationSubmitted, domain.NotificationTypeCompetencySubmitted})
	}
	return q
}

func (r *gormNotificationRepository) List(ctx context.Context, scope domain.NotificationScope, options domain.NotificationListOptions) (domain.NotificationPage, error) {
	status, limit := options.Status, options.Limit
	if status == "" {
		status = domain.NotificationStatusAll
	}
	if limit == 0 {
		limit = 20
	}
	invalid := func() (domain.NotificationPage, error) {
		return domain.NotificationPage{}, domain.NewError(domain.CodeSystemValidation)
	}
	if (status != domain.NotificationStatusAll && status != domain.NotificationStatusUnread) || limit < 1 || limit > 100 {
		return invalid()
	}
	var cursorTime time.Time
	var cursorID string
	if options.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(options.Cursor)
		if err != nil {
			return invalid()
		}
		parts := strings.Split(string(decoded), "|")
		if len(parts) != 2 {
			return invalid()
		}
		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return invalid()
		}
		if _, err = ulid.ParseStrict(parts[1]); err != nil {
			return invalid()
		}
		cursorID = parts[1]
	}
	base := func() *gorm.DB {
		q := r.scoped(ctx, scope)
		if status == domain.NotificationStatusUnread {
			q = q.Where("read_at IS NULL")
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return domain.NotificationPage{}, err
	}
	q := base()
	if cursorID != "" {
		q = q.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}
	var rows []model.Notification
	if err := q.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return domain.NotificationPage{}, err
	}
	var next *string
	if len(rows) > limit {
		last := rows[limit-1]
		val := base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%s|%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID))
		next = &val
		rows = rows[:limit]
	}
	items := make([]domain.Notification, len(rows))
	for i, row := range rows {
		items[i] = row.ToDomain()
	}
	if err := r.loadSubjects(ctx, scope, items); err != nil {
		return domain.NotificationPage{}, err
	}
	return domain.NotificationPage{Items: items, Total: total, NextCursor: next}, nil
}

// Each branch is a fixed allowlisted table. All subjects resolve in one query regardless of page composition.
func (r *gormNotificationRepository) loadSubjects(ctx context.Context, scope domain.NotificationScope, items []domain.Notification) error {
	ids := map[domain.NotificationSubjectType][]string{}
	for _, n := range items {
		if n.SubjectType != nil && n.SubjectID != nil {
			ids[*n.SubjectType] = append(ids[*n.SubjectType], *n.SubjectID)
		}
	}
	var queries []string
	var args []any
	if values := ids[domain.NotificationSubjectCredential]; len(values) > 0 {
		queries = append(queries, "SELECT 'credential' AS subject_type, id, name, number, NULL AS status FROM credentials WHERE id IN ? AND holder_user_id = ?")
		args = append(args, values, scope.UserID)
	}
	if scope.CanReview {
		for _, branch := range []struct {
			typ   domain.NotificationSubjectType
			table string
		}{{domain.NotificationSubjectCredentialType, "credential_types"}, {domain.NotificationSubjectCredentialIssuerOrganization, "credential_issuer_organizations"}, {domain.NotificationSubjectCompetency, "competencies"}} {
			if values := ids[branch.typ]; len(values) > 0 {
				queries = append(queries, "SELECT ? AS subject_type, id, name, NULL AS number, CASE WHEN rejected_at IS NOT NULL THEN 'rejected' WHEN activated_at IS NOT NULL THEN 'approved' ELSE 'pending' END AS status FROM "+branch.table+" WHERE id IN ?")
				args = append(args, string(branch.typ), values)
			}
		}
	}
	if len(queries) == 0 {
		return nil
	}
	var rows []struct {
		SubjectType domain.NotificationSubjectType
		ID          string
		Name        string
		Number      *string
		Status      *string
	}
	if err := r.db.WithContext(ctx).Raw(strings.Join(queries, " UNION ALL "), args...).Scan(&rows).Error; err != nil {
		return err
	}
	byType := map[domain.NotificationSubjectType]map[string]domain.NotificationSubject{}
	for _, row := range rows {
		if byType[row.SubjectType] == nil {
			byType[row.SubjectType] = map[string]domain.NotificationSubject{}
		}
		byType[row.SubjectType][row.ID] = domain.NotificationSubject{ID: row.ID, Name: row.Name, Number: row.Number, Status: row.Status}
	}
	for i, n := range items {
		if n.SubjectType != nil && n.SubjectID != nil {
			if subject, ok := byType[*n.SubjectType][*n.SubjectID]; ok {
				items[i].Subject = &subject
			}
		}
	}
	return nil
}

func (r *gormNotificationRepository) UnreadCount(ctx context.Context, scope domain.NotificationScope) (int64, error) {
	var count int64
	err := r.scoped(ctx, scope).Where("read_at IS NULL").Count(&count).Error
	return count, err
}

func (r *gormNotificationRepository) readTimestamp() clause.Expr {
	if r.db.Dialector.Name() == "postgres" {
		return gorm.Expr("statement_timestamp()")
	}
	return gorm.Expr("CURRENT_TIMESTAMP")
}

func (r *gormNotificationRepository) MarkRead(ctx context.Context, scope domain.NotificationScope, id string) (*domain.Notification, error) {
	stamp := r.readTimestamp()
	if err := r.scoped(ctx, scope).Where("id = ? AND read_at IS NULL", id).Updates(map[string]any{"read_at": stamp, "updated_at": stamp}).Error; err != nil {
		return nil, err
	}
	var row model.Notification
	if err := r.scoped(ctx, scope).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	items := []domain.Notification{row.ToDomain()}
	if err := r.loadSubjects(ctx, scope, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *gormNotificationRepository) MarkAllRead(ctx context.Context, scope domain.NotificationScope) (int64, error) {
	stamp := r.readTimestamp()
	res := r.scoped(ctx, scope).Where("read_at IS NULL").Updates(map[string]any{"read_at": stamp, "updated_at": stamp})
	return res.RowsAffected, res.Error
}
