package credential

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
)

func recordReferenceSubmission(ctx context.Context, uow domain.UnitOfWork, submitterID, subjectID string, kind domain.NotificationType) error {
	query := &domainQuery.Query{Filters: []domainQuery.Filter{
		domainQuery.NewFilter("role", domainQuery.OperatorIn, string(domain.RoleIssuer), string(domain.RoleAdmin), string(domain.RoleSuperAdmin)),
		domainQuery.NewFilter("deleted_at", domainQuery.OperatorNull),
	}}
	reviewers, _, err := uow.User().Get(ctx, query)
	if err != nil {
		return err
	}
	subjectType := kind.SubjectType()
	rows := make([]domain.Notification, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if reviewer.Id == submitterID {
			continue
		}
		rows = append(rows, domain.Notification{
			NotifiedUserID: reviewer.Id,
			SubjectType:    &subjectType,
			SubjectID:      &subjectID,
			Type:           kind,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return uow.Notification().Store(ctx, rows...)
}
