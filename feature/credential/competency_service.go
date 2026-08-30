package credential

import (
	"context"
	"errors"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

// CompetencyService is the business-logic layer for competencies.
type CompetencyService interface {
	// Paginate returns the page and the total matching rows before pagination.
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, int, error)
	Find(ctx context.Context, id string) (*domain.Competency, error)
	Store(ctx context.Context, name string, active *bool) (*domain.Competency, error)
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

// Store is an idempotent upsert-by-name (D17): names are trimmed and matched
// case-insensitively; an existing name returns the existing row (HTTP 200),
// so submitters can reference a competency that does not exist yet without
// ever creating duplicates.
func (s *competencyService) Store(ctx context.Context, name string, active *bool) (*domain.Competency, error) {
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	trimmed := strings.TrimSpace(name)

	matches, err := s.competencyRepo.FindByNames(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		return &matches[0], nil
	}

	stored, err := s.competencyRepo.Store(ctx, domain.Competency{Name: trimmed, Active: activeVal})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	c := stored[0]
	return &c, nil
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
