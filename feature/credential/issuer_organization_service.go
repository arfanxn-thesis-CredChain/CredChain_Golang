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

// CredentialIssuerOrganizationService is the business-logic layer for
// credential_issuer_organizations.
type CredentialIssuerOrganizationService interface {
	// Paginate returns the page and the total matching rows before pagination.
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error)
	Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error)
	Store(ctx context.Context, name string, active *bool) (*domain.CredentialIssuerOrganization, error)
	Submit(ctx context.Context, name string) (*domain.CredentialIssuerOrganization, error)
	Approve(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error)
	Reject(ctx context.Context, id string, reason string) (*domain.CredentialIssuerOrganization, error)
	Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialIssuerOrganization, error)
	// Destroy hard-deletes organizations no credential references. Referenced
	// organizations fail with CodeCredentialIssuerOrganizationDestroyInUse;
	// Update(active=false) is the everyday deactivation path.
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type credentialIssuerOrganizationService struct {
	orgRepo        domain.CredentialIssuerOrganizationRepository
	credentialRepo domain.CredentialRepository
}

type CredentialIssuerOrganizationServiceParams struct {
	fx.In
	OrgRepo        domain.CredentialIssuerOrganizationRepository
	CredentialRepo domain.CredentialRepository
}

func NewCredentialIssuerOrganizationService(p CredentialIssuerOrganizationServiceParams) CredentialIssuerOrganizationService {
	return &credentialIssuerOrganizationService{orgRepo: p.OrgRepo, credentialRepo: p.CredentialRepo}
}

// Paginate returns the requested page and the total number of organizations
// matching the query before pagination, so the handler can build a complete
// pagination envelope.
func (s *credentialIssuerOrganizationService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error) {
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

	return s.orgRepo.Get(ctx, query)
}

func (s *credentialIssuerOrganizationService) Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
	o, err := s.orgRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeIssuerOrganizationNotFound, domain.WithMetadata("issuer_organization_id", id))
		}
		return nil, err
	}
	return o, nil
}

// Store creates a new approved and active issuer organization directly (Issuer+).
// If a matching unrejected name exists:
// - If pending: returns CodeIssuerOrganizationAlreadyPending
// - If approved: returns CodeIssuerOrganizationNameDuplicate
func (s *credentialIssuerOrganizationService) Store(ctx context.Context, name string, active *bool) (*domain.CredentialIssuerOrganization, error) {
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.orgRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeIssuerOrganizationNameDuplicate, domain.WithMetadata("name", trimmed))
	}

	var creatorID *string
	if u, err := httpContext.GetUser(ctx); err == nil && u != nil {
		creatorID = &u.Id
	}
	now := time.Now().UTC()

	row := domain.CredentialIssuerOrganization{
		Name:            trimmed,
		Active:          activeVal,
		SubmitterUserId: creatorID,
		ApproverUserId:  creatorID,
		ActivatedAt:     &now,
	}

	stored, err := s.orgRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	o := stored[0]
	return &o, nil
}

// Submit creates a new pending proposal for an issuer organization (Holder).
// It starts with active=false, without activated_at or rejected_at.
// If a matching unrejected name already exists:
// - If pending: returns CodeIssuerOrganizationAlreadyPending
// - If approved: returns CodeIssuerOrganizationAlreadyApproved
func (s *credentialIssuerOrganizationService) Submit(ctx context.Context, name string) (*domain.CredentialIssuerOrganization, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.orgRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyApproved, domain.WithMetadata("name", trimmed))
	}

	u, err := httpContext.GetUser(ctx)
	if err != nil || u == nil {
		return nil, domain.NewError(domain.CodeAuthUnauthorized)
	}

	row := domain.CredentialIssuerOrganization{
		Name:            trimmed,
		Active:          false,
		SubmitterUserId: &u.Id,
	}

	stored, err := s.orgRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	o := stored[0]
	return &o, nil
}

// Approve transitions a pending issuer organization proposal to approved (active=true).
// Approval applies only to pending rows.
func (s *credentialIssuerOrganizationService) Approve(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
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
		return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeIssuerOrganizationNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.CredentialIssuerOrganization{
		Id:             target.Id,
		Active:         true,
		ApproverUserId: &u.Id,
		ActivatedAt:    &now,
	}
	updated, err := s.orgRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeIssuerOrganizationNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

// Reject transitions a pending issuer organization proposal to rejected with a reason.
// Rejection applies only to pending rows.
func (s *credentialIssuerOrganizationService) Reject(ctx context.Context, id string, reason string) (*domain.CredentialIssuerOrganization, error) {
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
		return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeIssuerOrganizationAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeIssuerOrganizationNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.CredentialIssuerOrganization{
		Id:              target.Id,
		Active:          false,
		RejecterUserId:  &u.Id,
		RejectedAt:      &now,
		RejectionReason: &trimmedReason,
	}
	updated, err := s.orgRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeIssuerOrganizationNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *credentialIssuerOrganizationService) checkNameUnique(ctx context.Context, name string, excludeId string) error {
	matches, err := s.orgRepo.FindByNames(ctx, name)
	if err != nil {
		return err
	}
	for _, o := range matches {
		if o.Id != excludeId {
			return domain.NewError(domain.CodeIssuerOrganizationNameDuplicate, domain.WithMetadata("name", name))
		}
	}
	return nil
}

func (s *credentialIssuerOrganizationService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialIssuerOrganization, error) {
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		if err := s.checkNameUnique(ctx, *name, id); err != nil {
			return nil, err
		}
	}
	u := domain.CredentialIssuerOrganization{Id: target.Id, Name: target.Name}
	if name != nil {
		u.Name = strings.TrimSpace(*name)
	}
	// The repository always emits the `active` CASE branch, so a name-only
	// update must carry the current value forward or it would deactivate.
	if active != nil {
		u.Active = *active
	} else {
		u.Active = target.Active
	}
	updated, err := s.orgRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeIssuerOrganizationNotFound, domain.WithMetadata("issuer_organization_id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *credentialIssuerOrganizationService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := s.credentialRepo.CountByIssuerOrganizationIds(ctx, ids...)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, domain.NewError(domain.CodeCredentialIssuerOrganizationDestroyInUse, domain.WithMetadata("ids", ids))
	}
	return s.orgRepo.Destroy(ctx, ids...)
}

var _ CredentialIssuerOrganizationService = (*credentialIssuerOrganizationService)(nil)
