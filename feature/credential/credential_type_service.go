package credential

import (
	"context"
	"errors"
	"strings"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

// CredentialTypeService is the business-logic layer for credential_types.
type CredentialTypeService interface {
	// Paginate returns the page and the total matching rows before pagination.
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, int, error)
	Find(ctx context.Context, id string) (*domain.CredentialType, error)
	Store(ctx context.Context, name string, active *bool) (*domain.CredentialType, error)
	Submit(ctx context.Context, name string) (*domain.CredentialType, error)
	Approve(ctx context.Context, id string) (*domain.CredentialType, error)
	Reject(ctx context.Context, id string, reason string) (*domain.CredentialType, error)
	Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialType, error)
	// Destroy hard-deletes types no credential references. Referenced types
	// fail with CodeCredentialTypeDestroyInUse; Update(active=false) is the
	// everyday deactivation path.
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type credentialTypeService struct {
	typeRepo       domain.CredentialTypeRepository
	credentialRepo domain.CredentialRepository
}

type CredentialTypeServiceParams struct {
	fx.In
	TypeRepo       domain.CredentialTypeRepository
	CredentialRepo domain.CredentialRepository
}

func NewCredentialTypeService(p CredentialTypeServiceParams) CredentialTypeService {
	return &credentialTypeService{typeRepo: p.TypeRepo, credentialRepo: p.CredentialRepo}
}

// Paginate returns the requested page and the total number of credential types
// matching the query before pagination, so the handler can build a complete
// pagination envelope.
func (s *credentialTypeService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, int, error) {
	if query == nil {
		q := domainQuery.NewQuery()
		query = &q
	} else {
		cloned := *query
		cloned.Filters = make([]domainQuery.Filter, len(query.Filters))
		copy(cloned.Filters, query.Filters)
		query = &cloned
	}

	remaining := query.Filters[:0]
	for _, f := range query.Filters {
		if f.Column != "holder_user_id" {
			remaining = append(remaining, f)
		}
	}
	query.Filters = remaining

	if u, err := httpContext.GetUser(ctx); err == nil && u != nil {
		if u.Role.Rank() < domain.RoleIssuer.Rank() {
			query.Filters = append(query.Filters, domainQuery.NewFilter("holder_user_id", domainQuery.OperatorEqual, u.Id))
		}
	}

	return s.typeRepo.Get(ctx, query)
}

func (s *credentialTypeService) Find(ctx context.Context, id string) (*domain.CredentialType, error) {
	t, err := s.typeRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCredentialTypeNotFound, domain.WithMetadata("credential_type_id", id))
		}
		return nil, err
	}
	return t, nil
}

// Store creates a new approved and active credential type directly (Issuer+).
// If a matching unrejected name exists:
// - If pending: returns CodeCredentialTypeAlreadyPending
// - If approved: returns CodeCredentialTypeNameDuplicate
func (s *credentialTypeService) Store(ctx context.Context, name string, active *bool) (*domain.CredentialType, error) {
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.typeRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeCredentialTypeAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeCredentialTypeNameDuplicate, domain.WithMetadata("name", trimmed))
	}

	var creatorID *string
	if u, err := httpContext.GetUser(ctx); err == nil && u != nil {
		creatorID = &u.Id
	}
	now := time.Now().UTC()

	row := domain.CredentialType{
		Name:            trimmed,
		Active:          activeVal,
		SubmitterUserId: creatorID,
		ApproverUserId:  creatorID,
		ActivatedAt:     &now,
	}

	stored, err := s.typeRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	t := stored[0]
	return &t, nil
}

// Submit creates a new pending proposal for a credential type (Holder).
// It starts with active=false, without activated_at or rejected_at.
// If a matching unrejected name already exists:
// - If pending: returns CodeCredentialTypeAlreadyPending
// - If approved: returns CodeCredentialTypeAlreadyApproved
func (s *credentialTypeService) Submit(ctx context.Context, name string) (*domain.CredentialType, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.typeRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeCredentialTypeAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeCredentialTypeAlreadyApproved, domain.WithMetadata("name", trimmed))
	}

	u, err := httpContext.GetUser(ctx)
	if err != nil || u == nil {
		return nil, domain.NewError(domain.CodeAuthUnauthorized)
	}

	row := domain.CredentialType{
		Name:            trimmed,
		Active:          false,
		SubmitterUserId: &u.Id,
	}

	stored, err := s.typeRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	t := stored[0]
	return &t, nil
}

// Approve transitions a pending credential type proposal to approved (active=true).
// Approval applies only to pending rows.
func (s *credentialTypeService) Approve(ctx context.Context, id string) (*domain.CredentialType, error) {
	u, err := httpContext.GetUser(ctx)
	if err != nil || u == nil {
		return nil, domain.NewError(domain.CodeAuthUnauthorized)
	}
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	switch target.Status() {
	case domain.ReferenceStatusApproved:
		return nil, domain.NewError(domain.CodeCredentialTypeAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeCredentialTypeAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeCredentialTypeNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.CredentialType{
		Id:             target.Id,
		Active:         true,
		ApproverUserId: &u.Id,
		ActivatedAt:    &now,
	}
	updated, err := s.typeRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCredentialTypeNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

// Reject transitions a pending credential type proposal to rejected with a reason.
// Rejection applies only to pending rows.
func (s *credentialTypeService) Reject(ctx context.Context, id string, reason string) (*domain.CredentialType, error) {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}
	u, err := httpContext.GetUser(ctx)
	if err != nil || u == nil {
		return nil, domain.NewError(domain.CodeAuthUnauthorized)
	}
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	switch target.Status() {
	case domain.ReferenceStatusApproved:
		return nil, domain.NewError(domain.CodeCredentialTypeAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeCredentialTypeAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeCredentialTypeNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.CredentialType{
		Id:              target.Id,
		Active:          false,
		RejecterUserId:  &u.Id,
		RejectedAt:      &now,
		RejectionReason: &trimmedReason,
	}
	updated, err := s.typeRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCredentialTypeNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *credentialTypeService) checkNameUnique(ctx context.Context, name string, excludeId string) error {
	matches, err := s.typeRepo.FindByNames(ctx, name)
	if err != nil {
		return err
	}
	for _, t := range matches {
		if t.Id != excludeId {
			return domain.NewError(domain.CodeCredentialTypeNameDuplicate, domain.WithMetadata("name", name))
		}
	}
	return nil
}

func (s *credentialTypeService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialType, error) {
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		if err := s.checkNameUnique(ctx, *name, id); err != nil {
			return nil, err
		}
	}
	u := domain.CredentialType{Id: target.Id}
	if name != nil {
		u.Name = strings.TrimSpace(*name)
	}
	if active != nil {
		u.Active = *active
	} else {
		u.Active = target.Active
	}
	updated, err := s.typeRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCredentialTypeNotFound, domain.WithMetadata("credential_type_id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *credentialTypeService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := s.credentialRepo.CountByTypeIds(ctx, ids...)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, domain.NewError(domain.CodeCredentialTypeDestroyInUse, domain.WithMetadata("ids", ids))
	}
	return s.typeRepo.Destroy(ctx, ids...)
}

var _ CredentialTypeService = (*credentialTypeService)(nil)
