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

// CredentialIssuerOrganizationService is the business-logic layer for
// credential_issuer_organizations.
type CredentialIssuerOrganizationService interface {
	// Paginate returns the page and the total matching rows before pagination.
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error)
	Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error)
	Store(ctx context.Context, name string) (*domain.CredentialIssuerOrganization, error)
	Update(ctx context.Context, id string, name *string) (*domain.CredentialIssuerOrganization, error)
	// Destroy hard-deletes organizations no credential references. Referenced
	// organizations fail with CodeCredentialIssuerOrganizationDestroyInUse.
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

// Store is an idempotent upsert-by-name (D17): names are trimmed and matched
// case-insensitively; an existing name returns the existing row (HTTP 200),
// so submitters can reference an organization that does not exist yet without
// ever creating duplicates.
func (s *credentialIssuerOrganizationService) Store(ctx context.Context, name string) (*domain.CredentialIssuerOrganization, error) {
	trimmed := strings.TrimSpace(name)

	existing, _, err := s.orgRepo.Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	if found, ok := lo.Find(existing, func(o domain.CredentialIssuerOrganization) bool {
		return strings.EqualFold(o.Name, trimmed)
	}); ok {
		return &found, nil
	}

	stored, err := s.orgRepo.Store(ctx, domain.CredentialIssuerOrganization{Name: trimmed})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, domain.NewError(domain.CodeSystemInternal)
	}
	o := stored[0]
	return &o, nil
}

func (s *credentialIssuerOrganizationService) checkNameUnique(ctx context.Context, name string, excludeId string) error {
	existing, _, err := s.orgRepo.Get(ctx, nil)
	if err != nil {
		return err
	}
	if lo.ContainsBy(existing, func(o domain.CredentialIssuerOrganization) bool {
		return strings.EqualFold(o.Name, strings.TrimSpace(name)) && o.Id != excludeId
	}) {
		return domain.NewError(domain.CodeIssuerOrganizationNameDuplicate, domain.WithMetadata("name", name))
	}
	return nil
}

func (s *credentialIssuerOrganizationService) Update(ctx context.Context, id string, name *string) (*domain.CredentialIssuerOrganization, error) {
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
