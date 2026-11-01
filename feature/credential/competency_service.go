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

// CompetencyService is the business-logic layer for competencies.
type CompetencyService interface {
	// Paginate returns the page and the total matching rows before pagination.
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, int, error)
	Find(ctx context.Context, id string) (*domain.Competency, error)
	Store(ctx context.Context, name string, active *bool) (*domain.Competency, error)
	Submit(ctx context.Context, name string) (*domain.Competency, error)
	Approve(ctx context.Context, id string) (*domain.Competency, error)
	Reject(ctx context.Context, id string, reason string) (*domain.Competency, error)
	Update(ctx context.Context, id string, name *string, active *bool) (*domain.Competency, error)
	// Destroy hard-deletes competencies no credential references. Referenced
	// competencies fail with CodeCompetencyDestroyInUse; Update(active=false)
	// is the everyday deactivation path.
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type competencyService struct {
	competencyRepo           domain.CompetencyRepository
	competencyCredentialRepo domain.CompetencyCredentialRepository
}

type CompetencyServiceParams struct {
	fx.In
	CompetencyRepo           domain.CompetencyRepository
	CompetencyCredentialRepo domain.CompetencyCredentialRepository
}

func NewCompetencyService(p CompetencyServiceParams) CompetencyService {
	return &competencyService{competencyRepo: p.CompetencyRepo, competencyCredentialRepo: p.CompetencyCredentialRepo}
}

// Paginate returns the requested page and the total number of competencies
// matching the query before pagination, so the handler can build a complete
// pagination envelope.
func (s *competencyService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, int, error) {
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

	return s.competencyRepo.Get(ctx, query)
}

func (s *competencyService) Find(ctx context.Context, id string) (*domain.Competency, error) {
	c, err := s.competencyRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCompetencyNotFound, domain.WithMetadata("competency_id", id))
		}
		return nil, err
	}
	return c, nil
}

// Store creates a new approved and active competency directly (Issuer+).
// If a matching unrejected name exists:
// - If pending: returns CodeCompetencyAlreadyPending
// - If approved: returns CodeCompetencyNameDuplicate
func (s *competencyService) Store(ctx context.Context, name string, active *bool) (*domain.Competency, error) {
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.competencyRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeCompetencyAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeCompetencyNameDuplicate, domain.WithMetadata("name", trimmed))
	}

	var creatorID *string
	if u, err := httpContext.GetUser(ctx); err == nil && u != nil {
		creatorID = &u.Id
	}
	now := time.Now().UTC()

	row := domain.Competency{
		Name:            trimmed,
		Active:          activeVal,
		SubmitterUserId: creatorID,
		ApproverUserId:  creatorID,
		ActivatedAt:     &now,
	}

	stored, err := s.competencyRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	c := stored[0]
	return &c, nil
}

// Submit creates a new pending proposal for a competency (Holder).
// It starts with active=false, without activated_at or rejected_at.
// If a matching unrejected name already exists:
// - If pending: returns CodeCompetencyAlreadyPending
// - If approved: returns CodeCompetencyAlreadyApproved
func (s *competencyService) Submit(ctx context.Context, name string) (*domain.Competency, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.NewError(domain.CodeSystemValidation)
	}

	matches, err := s.competencyRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		m := matches[0]
		if m.Status() == domain.ReferenceStatusPending {
			return nil, domain.NewError(domain.CodeCompetencyAlreadyPending, domain.WithMetadata("name", trimmed))
		}
		return nil, domain.NewError(domain.CodeCompetencyAlreadyApproved, domain.WithMetadata("name", trimmed))
	}

	u, err := httpContext.GetUser(ctx)
	if err != nil || u == nil {
		return nil, domain.NewError(domain.CodeAuthUnauthorized)
	}

	row := domain.Competency{
		Name:            trimmed,
		Active:          false,
		SubmitterUserId: &u.Id,
	}

	stored, err := s.competencyRepo.Store(ctx, row)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	c := stored[0]
	return &c, nil
}

// Approve transitions a pending competency proposal to approved (active=true).
// Approval applies only to pending rows.
func (s *competencyService) Approve(ctx context.Context, id string) (*domain.Competency, error) {
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
		return nil, domain.NewError(domain.CodeCompetencyAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeCompetencyAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeCompetencyNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.Competency{
		Id:             target.Id,
		Active:         true,
		ApproverUserId: &u.Id,
		ActivatedAt:    &now,
	}
	updated, err := s.competencyRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCompetencyNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

// Reject transitions a pending competency proposal to rejected with a reason.
// Rejection applies only to pending rows.
func (s *competencyService) Reject(ctx context.Context, id string, reason string) (*domain.Competency, error) {
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
		return nil, domain.NewError(domain.CodeCompetencyAlreadyApproved, domain.WithMetadata("id", id))
	case domain.ReferenceStatusRejected:
		return nil, domain.NewError(domain.CodeCompetencyAlreadyRejected, domain.WithMetadata("id", id))
	case domain.ReferenceStatusPending:
		// ok
	default:
		return nil, domain.NewError(domain.CodeCompetencyNotPending, domain.WithMetadata("id", id))
	}

	now := time.Now().UTC()
	updatePayload := domain.Competency{
		Id:              target.Id,
		Active:          false,
		RejecterUserId:  &u.Id,
		RejectedAt:      &now,
		RejectionReason: &trimmedReason,
	}
	updated, err := s.competencyRepo.Update(ctx, updatePayload)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCompetencyNotFound, domain.WithMetadata("id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *competencyService) checkNameUnique(ctx context.Context, name string, excludeId string) error {
	matches, err := s.competencyRepo.FindByNames(ctx, name)
	if err != nil {
		return err
	}
	for _, c := range matches {
		if c.Id != excludeId {
			return domain.NewError(domain.CodeCompetencyNameDuplicate, domain.WithMetadata("name", name))
		}
	}
	return nil
}

func (s *competencyService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.Competency, error) {
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		if err := s.checkNameUnique(ctx, *name, id); err != nil {
			return nil, err
		}
	}
	u := domain.Competency{Id: target.Id, Name: target.Name}
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
	updated, err := s.competencyRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeCompetencyNotFound, domain.WithMetadata("competency_id", id))
	}
	out := updated[0]
	return &out, nil
}

func (s *competencyService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := s.competencyCredentialRepo.CountByCompetencyIds(ctx, ids...)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, domain.NewError(domain.CodeCompetencyDestroyInUse, domain.WithMetadata("ids", ids))
	}
	return s.competencyRepo.Destroy(ctx, ids...)
}

var _ CompetencyService = (*competencyService)(nil)
