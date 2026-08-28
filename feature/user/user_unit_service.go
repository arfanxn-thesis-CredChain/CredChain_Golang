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
	// Update changes name, parent, and/or active. setParent distinguishes "clear
	// parent to root" (parentId nil, setParent true) from "leave parent untouched"
	// (setParent false); the handler derives it from parent_id key presence. active
	// is nil for "unchanged".
	Update(ctx context.Context, id string, name *string, parentId *string, setParent bool, active *bool) (*domain.UserUnit, error)
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
// types/orgs/competencies only). Parent is validated when provided; a child
// under an inactive parent is born inactive.
func (s *userUnitService) Store(ctx context.Context, name string, parentId *string) (*domain.UserUnit, error) {
	if err := s.validateParent(ctx, parentId, ""); err != nil {
		return nil, err
	}
	active := true
	if parentId != nil {
		parent, err := s.unitRepo.Find(ctx, *parentId)
		if err != nil {
			return nil, err
		}
		active = parent.Active
	}
	stored, err := s.unitRepo.Store(ctx, domain.UserUnit{Name: strings.TrimSpace(name), ParentId: parentId, Active: active})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	u := stored[0]
	return &u, nil
}

func (s *userUnitService) Update(ctx context.Context, id string, name *string, parentId *string, setParent bool, active *bool) (*domain.UserUnit, error) {
	target, err := s.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		// Active must be carried forward: the batch Update always emits it.
		if _, err := s.unitRepo.Update(ctx, domain.UserUnit{Id: target.Id, Name: strings.TrimSpace(*name), Active: target.Active}); err != nil {
			return nil, err
		}
	}

	if active != nil {
		if !*active {
			// Deactivation cascades to every descendant (rule 1).
			if err := s.cascadeOff(ctx, target.Id, nil); err != nil {
				return nil, err
			}
		} else {
			// Activation is never cascading and is rejected under an inactive parent.
			if target.ParentId != nil {
				parent, err := s.unitRepo.Find(ctx, *target.ParentId)
				if err != nil {
					return nil, err
				}
				if !parent.Active {
					return nil, domain.NewError(domain.CodeUserUnitParentInactive, domain.WithMetadata("parent_id", *target.ParentId))
				}
			}
			if _, err := s.unitRepo.Update(ctx, domain.UserUnit{Id: target.Id, Active: true}); err != nil {
				return nil, err
			}
		}
	}

	if setParent {
		if err := s.validateParent(ctx, parentId, target.Id); err != nil {
			return nil, err
		}
		if parentId == nil {
			// Promote to root — never cascades (rule 5).
			if err := s.unitRepo.UpdateParent(ctx, target.Id, nil); err != nil {
				return nil, err
			}
		} else {
			newParent, err := s.unitRepo.Find(ctx, *parentId)
			if err != nil {
				return nil, err
			}
			if newParent.Active {
				if err := s.unitRepo.UpdateParent(ctx, target.Id, parentId); err != nil {
					return nil, err
				}
			} else {
				// Move under an inactive parent cascades the moved subtree off,
				// atomically with the reparent (rule 4).
				if err := s.cascadeOff(ctx, target.Id, parentId); err != nil {
					return nil, err
				}
			}
		}
	}

	return s.Find(ctx, id)
}

// cascadeOff sets the unit and every descendant inactive in one batch UPDATE
// (atomic). When reparentTo is non-nil it also reparents the top node — used
// by the move-under-inactive-parent path.
func (s *userUnitService) cascadeOff(ctx context.Context, id string, reparentTo *string) error {
	desc, err := s.unitRepo.FindWithDescendants(ctx, id)
	if err != nil {
		return err
	}
	updates := make([]domain.UserUnit, 0, len(desc))
	for _, d := range desc {
		u := domain.UserUnit{Id: d.Id, Active: false}
		if reparentTo != nil && d.Id == id {
			u.ParentId = reparentTo
		}
		updates = append(updates, u)
	}
	_, err = s.unitRepo.Update(ctx, updates...)
	return err
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
