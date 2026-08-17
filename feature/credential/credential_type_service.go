package credential

import (
	"context"
	"errors"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"github.com/samber/lo"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// CredentialTypeService is the business-logic layer for credential_types.
type CredentialTypeService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, error)
	Find(ctx context.Context, id string) (*domain.CredentialType, error)
	Store(ctx context.Context, name string, active *bool) (*domain.CredentialType, error)
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

func (s *credentialTypeService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, error) {
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

// Store is an idempotent upsert-by-name (D17): names are trimmed and matched
// case-insensitively; an existing name returns the existing row (HTTP 200),
// so submitters can reference a type that does not exist yet without ever
// creating duplicates.
func (s *credentialTypeService) Store(ctx context.Context, name string, active *bool) (*domain.CredentialType, error) {
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	trimmed := strings.TrimSpace(name)

	existing, err := s.typeRepo.Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	if found, ok := lo.Find(existing, func(t domain.CredentialType) bool {
		return strings.EqualFold(t.Name, trimmed)
	}); ok {
		return &found, nil
	}

	stored, err := s.typeRepo.Store(ctx, domain.CredentialType{Name: trimmed, Active: activeVal})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	t := stored[0]
	return &t, nil
}

func (s *credentialTypeService) checkNameUnique(ctx context.Context, name string, excludeId string) error {
	existing, err := s.typeRepo.Get(ctx, nil)
	if err != nil {
		return err
	}
	if lo.ContainsBy(existing, func(t domain.CredentialType) bool {
		return strings.EqualFold(t.Name, strings.TrimSpace(name)) && t.Id != excludeId
	}) {
		return domain.NewError(domain.CodeCredentialTypeNameDuplicate, domain.WithMetadata("name", name))
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
		u.Name = *name
	}
	if active != nil {
		u.Active = *active
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
