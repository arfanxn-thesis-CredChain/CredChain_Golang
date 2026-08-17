package user

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

// UserUnitService is the business-logic layer for user_units — the
// self-referential organizational tree (faculty / study program / department).
type UserUnitService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.UserUnit, error)
	Find(ctx context.Context, id string) (*domain.UserUnit, error)
	Store(ctx context.Context, name string, parentId *string) (*domain.UserUnit, error)
	Update(ctx context.Context, id string, name *string, parentId *string) (*domain.UserUnit, error)
	// Destroy hard-deletes units neither users nor child units reference.
	// Referenced units fail with CodeUserUnitDestroyInUse.
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type userUnitService struct {
	unitRepo domain.UserUnitRepository
	userRepo domain.UserRepository
}

type UserUnitServiceParams struct {
	fx.In
	UnitRepo domain.UserUnitRepository
	UserRepo domain.UserRepository
}

func NewUserUnitService(p UserUnitServiceParams) UserUnitService {
	return &userUnitService{unitRepo: p.UnitRepo, userRepo: p.UserRepo}
}

func (s *userUnitService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.UserUnit, error) {
	return s.unitRepo.Get(ctx, query)
}

func (s *userUnitService) Find(ctx context.Context, id string) (*domain.UserUnit, error) {
	u, err := s.unitRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeUserUnitNotFound, domain.WithMetadata("user_unit_id", id))
		}
		return nil, err
	}
	return u, nil
}

// Store is a plain create — unit names are not unique (D17 upsert applies to
// types/orgs/competencies only). Parent is validated when provided.
func (s *userUnitService) Store(ctx context.Context, name string, parentId *string) (*domain.UserUnit, error) {
	if err := s.validateParent(ctx, parentId, ""); err != nil {
		return nil, err
	}
	stored, err := s.unitRepo.Store(ctx, domain.UserUnit{Name: strings.TrimSpace(name), ParentId: parentId})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	u := stored[0]
	return &u, nil
}

func (s *userUnitService) Update(ctx context.Context, id string, name *string, parentId *string) (*domain.UserUnit, error) {
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateParent(ctx, parentId, target.Id); err != nil {
		return nil, err
	}
	u := domain.UserUnit{Id: target.Id, Name: target.Name, ParentId: target.ParentId}
	if name != nil {
		u.Name = strings.TrimSpace(*name)
	}
	if parentId != nil {
		u.ParentId = parentId
	}
	updated, err := s.unitRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeUserUnitNotFound, domain.WithMetadata("user_unit_id", id))
	}
	out := updated[0]
	return &out, nil
}

// validateParent ensures parentId references an existing unit that is neither
// the unit itself nor one of its descendants (cycle check). Store passes
// selfId ""; Update passes the unit id.
func (s *userUnitService) validateParent(ctx context.Context, parentId *string, selfId string) error {
	if parentId == nil {
		return nil
	}
	if _, err := s.unitRepo.Find(ctx, *parentId); err != nil {
		return domain.NewError(domain.CodeUserUnitParentInvalid, domain.WithMetadata("parent_id", *parentId))
	}
	if selfId == "" {
		return nil
	}
	descendants, err := s.unitRepo.FindWithDescendants(ctx, selfId)
	if err != nil {
		return err
	}
	if lo.ContainsBy(descendants, func(u domain.UserUnit) bool { return u.Id == *parentId }) {
		return domain.NewError(domain.CodeUserUnitParentInvalid, domain.WithMetadata("parent_id", *parentId))
	}
	return nil
}

func (s *userUnitService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := s.userRepo.CountByUnitIds(ctx, ids...)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, domain.NewError(domain.CodeUserUnitDestroyInUse, domain.WithMetadata("ids", ids))
	}
	children, err := s.unitRepo.CountByParentIds(ctx, ids...)
	if err != nil {
		return 0, err
	}
	if children > 0 {
		return 0, domain.NewError(domain.CodeUserUnitDestroyInUse, domain.WithMetadata("ids", ids))
	}
	return s.unitRepo.Destroy(ctx, ids...)
}

var _ UserUnitService = (*userUnitService)(nil)
