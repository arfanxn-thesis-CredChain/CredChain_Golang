package overview

import (
	"context"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/response"

	"go.uber.org/fx"
)

type OverviewService interface {
	Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error)
	CredentialRecap(ctx context.Context) (*response.CredentialRecap, error)
}

type overviewService struct {
	overviewRepo domain.OverviewRepository
	credRepo     domain.CredentialRepository
	userRepo     domain.UserRepository
	unitRepo     domain.UserUnitRepository
	cfg          *config.Config
	chainClient  *chain.Client
}

type OverviewServiceParams struct {
	fx.In
	OverviewRepo domain.OverviewRepository
	CredRepo     domain.CredentialRepository
	UserRepo     domain.UserRepository
	UnitRepo     domain.UserUnitRepository
	Config       *config.Config
	ChainClient  *chain.Client
}

func NewOverviewService(p OverviewServiceParams) OverviewService {
	return &overviewService{
		overviewRepo: p.OverviewRepo,
		credRepo:     p.CredRepo,
		userRepo:     p.UserRepo,
		unitRepo:     p.UnitRepo,
		cfg:          p.Config,
		chainClient:  p.ChainClient,
	}
}

func (s *overviewService) CredentialRecap(ctx context.Context) (*response.CredentialRecap, error) {
	if httpContext.MustGetUser(ctx).Role.Rank() < domain.RoleIssuer.Rank() {
		return nil, domain.NewError(domain.CodeAuthForbidden)
	}
	asOf := time.Now().UTC()
	rows, err := s.overviewRepo.CredentialRecap(ctx, asOf)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}
	units, err := s.unitRepo.Get(ctx, nil)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}
	result := &response.CredentialRecap{AsOf: asOf, Rows: rows, Units: make([]response.CredentialRecapUnit, len(units))}
	if result.Rows == nil {
		result.Rows = []domain.CredentialRecapRow{}
	}
	for i, u := range units {
		result.Units[i] = response.CredentialRecapUnit{ID: u.Id, ParentID: u.ParentId, Name: u.Name, Active: u.Active}
	}
	return result, nil
}

func (s *overviewService) Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error) {
	authUser := httpContext.MustGetUser(ctx)
	isIssuer := authUser.Role.Rank() >= domain.RoleIssuer.Rank()

	dateFrom, dateTo := extractDateRange(q)

	limit := q.Limit
	if limit <= 0 {
		limit = 5
	}

	var holderID *string
	if !isIssuer {
		holderID = &authUser.Id
	}

	credCounts, err := s.overviewRepo.CredentialCounts(ctx, q, holderID)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	activeQ := buildRecentActiveCredentialQuery(dateFrom, dateTo, limit, holderID)
	activeCreds, _, err := s.credRepo.Get(ctx, activeQ)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	revokedQ := buildRecentRevokedCredentialQuery(dateFrom, dateTo, limit, holderID)
	revokedCreds, _, err := s.credRepo.Get(ctx, revokedQ)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	dtoCredCounts := response.FromDomainOverviewCredentialCounts(*credCounts)
	ov := &response.Overview{
		CredentialCounts: &dtoCredCounts,
		Recents: &response.OverviewRecents{
			ActiveCredentials:  mapCredentials(activeCreds),
			RevokedCredentials: mapCredentials(revokedCreds),
		},
	}

	if isIssuer {
		userCounts, err := s.overviewRepo.UserCounts(ctx, q)
		if err != nil {
			return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
		}
		dtoUserCounts := response.FromDomainOverviewUserCounts(*userCounts)
		ov.UserCounts = &dtoUserCounts

		recentQ := buildRecentUsersQuery(dateFrom, dateTo, limit)
		recentUsers, _, err := s.userRepo.Get(ctx, recentQ)
		if err != nil {
			return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
		}
		ov.Recents.StoredUsers = mapUsers(recentUsers)

		var lastBlock uint64
		var relayerAddress string
		var relayerBalance = "0.00"
		if s.chainClient != nil {
			lastBlock, err = s.chainClient.BlockNumber(ctx)
			if err != nil {
				lastBlock = 0
			}
			relayerAddress = s.chainClient.Relayer.From.Hex()
			if balance, balErr := s.chainClient.RelayerBalance(ctx); balErr == nil {
				relayerBalance = balance
			}
		}
		ov.ChainDetails = &response.OverviewChainDetails{
			AuthorityContract: *s.cfg.AuthorityContract,
			RegistryContract:  *s.cfg.RegistryContract,
			LastBlock:         lastBlock,
			RelayerAddress:    relayerAddress,
			RelayerBalance:    relayerBalance,
		}
	}

	return ov, nil
}

func buildRecentActiveCredentialQuery(dateFrom, dateTo time.Time, limit int, holderID *string) *domainQuery.Query {
	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "status", Operator: domainQuery.OperatorEqual, Values: []string{"approved"}},
			{Column: "issued_at", Operator: domainQuery.OperatorBetween, Values: []string{
				dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"),
			}},
		},
		Includes: []string{"holder", "issuer"},
		Sorts:    []domainQuery.Sort{{Column: "issued_at", Order: domainQuery.SortDesc}},
		Limit:    limit,
		Page:     1,
	}
	if holderID != nil {
		q.Filters = append(q.Filters, domainQuery.Filter{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{*holderID}})
	}
	return q
}

func buildRecentRevokedCredentialQuery(dateFrom, dateTo time.Time, limit int, holderID *string) *domainQuery.Query {
	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "revoked_at", Operator: domainQuery.OperatorNotNull},
			{Column: "revoked_at", Operator: domainQuery.OperatorBetween, Values: []string{
				dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"),
			}},
		},
		Includes: []string{"holder", "revoker"},
		Sorts:    []domainQuery.Sort{{Column: "revoked_at", Order: domainQuery.SortDesc}},
		Limit:    limit,
		Page:     1,
	}
	if holderID != nil {
		q.Filters = append(q.Filters, domainQuery.Filter{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{*holderID}})
	}
	return q
}

func buildRecentUsersQuery(dateFrom, dateTo time.Time, limit int) *domainQuery.Query {
	return &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "created_at", Operator: domainQuery.OperatorBetween, Values: []string{
				dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"),
			}},
		},
		Sorts: []domainQuery.Sort{{Column: "created_at", Order: domainQuery.SortDesc}},
		Limit: limit,
		Page:  1,
	}
}

func mapCredentials(creds []domain.Credential) []response.Credential {
	out := make([]response.Credential, len(creds))
	for i, c := range creds {
		out[i] = response.FromDomainCredential(c)
	}
	return out
}

func mapUsers(users []domain.User) []response.User {
	out := make([]response.User, len(users))
	for i, u := range users {
		out[i] = response.FromDomainUser(u)
	}
	return out
}
