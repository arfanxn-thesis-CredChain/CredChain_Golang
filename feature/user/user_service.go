package user

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/oauth"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UserService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error)
	Find(ctx context.Context, id string) (*domain.User, error)
	FindByIds(ctx context.Context, ids ...string) ([]domain.User, error)
	Update(ctx context.Context, users ...domain.User) ([]domain.User, error)
	UpdateEmail(ctx context.Context, id string, email string, idToken string) (string, error)
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error)
	Store(ctx context.Context, users ...domain.User) ([]domain.User, error)
	Delete(ctx context.Context, ids ...string) (int64, error)
	TransferSuperAdmin(ctx context.Context, targetId string) error
	Restore(ctx context.Context, ids []string) ([]domain.User, int64, error)
}

type userService struct {
	userRepo         domain.UserRepository
	unitRepo         domain.UserUnitRepository
	uow              domain.UnitOfWork
	cfg              *config.Config
	authorityService chain.AuthorityService
	logger           *zap.Logger
	policy           UserPolicy
	oauthClient      oauth.GoogleOAuthClient
}

type UserServiceParams struct {
	fx.In
	UserRepo         domain.UserRepository
	UnitRepo         domain.UserUnitRepository
	UoW              domain.UnitOfWork
	Config           *config.Config
	AuthorityService chain.AuthorityService
	Logger           *zap.Logger
	Policy           UserPolicy
	OAuthClient      oauth.GoogleOAuthClient
}

func NewUserService(p UserServiceParams) UserService {
	return &userService{
		userRepo:         p.UserRepo,
		unitRepo:         p.UnitRepo,
		uow:              p.UoW,
		cfg:              p.Config,
		authorityService: p.AuthorityService,
		logger:           p.Logger,
		policy:           p.Policy,
		oauthClient:      p.OAuthClient,
	}
}

func (s *userService) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if err := s.policy.Store(ctx, users...); err != nil {
		return nil, err
	}

	if verrs := s.storeValidateEmails(ctx, users); len(verrs) > 0 {
		return nil, verrs
	}

	if err := s.storeGenerateWallets(users); err != nil {
		return nil, err
	}

	return s.storeUsersAndSyncBlockchainRoles(ctx, users)
}

func (s *userService) storeValidateEmails(ctx context.Context, users []domain.User) validation.Errors {
	verrs := validation.Errors{}

	seen := map[string]int{}
	for i, u := range users {
		if prev, ok := seen[u.Email]; ok {
			verrs[fmt.Sprintf("users.%d.email", i)] = validation.NewError(
				"validation_store_email_duplicate_batch",
				"",
			)
			verrs[fmt.Sprintf("users.%d.email", prev)] = validation.NewError(
				"validation_store_email_duplicate_batch",
				"",
			)
		}
		seen[u.Email] = i
	}

	if len(verrs) == 0 {
		emails := make([]string, len(users))
		for i, u := range users {
			emails[i] = u.Email
		}
		existing, _ := s.userRepo.FindByEmails(ctx, emails...)
		existingSet := lo.SliceToMap(existing, func(e domain.User) (string, bool) { return e.Email, true })
		for i, u := range users {
			if existingSet[u.Email] {
				verrs[fmt.Sprintf("users.%d.email", i)] = validation.NewError(
					"validation_store_email_duplicate_db",
					"",
				)
			}
		}
	}

	return verrs
}

func (s *userService) storeGenerateWallets(users []domain.User) error {
	for i := range users {
		key, err := ethCrypto.GenerateKey()
		if err != nil {
			return domain.NewError(domain.CodeUserStoreWalletGenerationFailed, domain.WithError(err))
		}
		privateKeyHex := hex.EncodeToString(ethCrypto.FromECDSA(key))
		address := ethCrypto.PubkeyToAddress(key.PublicKey).Hex()
		encrypted, err := cryptoInfra.Encrypt([]byte(privateKeyHex), []byte(*s.cfg.WalletEncryptionKey))
		if err != nil {
			return domain.NewError(domain.CodeUserStoreWalletGenerationFailed, domain.WithError(err))
		}
		users[i].WalletAddress = address
		users[i].EncryptedWalletPrivateKey = encrypted
	}
	return nil
}

func (s *userService) storeUsersAndSyncBlockchainRoles(ctx context.Context, users []domain.User) ([]domain.User, error) {
	var created []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		created, err = uow.User().Store(ctx, users...)
		if err != nil {
			return err
		}
		return s.syncBlockchainRoles(ctx, users, domain.CodeUserStoreBlockchainSyncFailed)
	})
	return created, err
}

// syncBlockchainRoles posts users to the on-chain authority and translates
// raw chain errors into the supplied domain code. Caller is responsible for
// transactional context; chain failure rolls back the surrounding UoW.
func (s *userService) syncBlockchainRoles(ctx context.Context, users []domain.User, errCode int) error {
	authUser := httpContext.MustGetUser(ctx)
	wallet := domain.WalletFromUser(*authUser)
	if err := s.authorityService.UpdateUserRole(ctx, wallet, users...); err != nil {
		return domain.NewError(errCode, domain.WithError(err))
	}
	return nil
}

// Paginate returns a paginated user list. A unit_id filter (equal) is
// rewritten into unit_id IN (unit + all descendants) via the recursive
// user_units query — faculty-level filtering without denormalization.
func (s *userService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	if query != nil {
		for i, f := range query.Filters {
			if f.Column == "unit_id" && f.Operator == domainQuery.OperatorEqual {
				units, err := s.unitRepo.FindWithDescendants(ctx, f.GetValue())
				if err != nil {
					return nil, 0, err
				}
				ids := lo.Map(units, func(u domain.UserUnit, _ int) string { return u.Id })
				query.Filters[i] = domainQuery.NewFilter("unit_id", domainQuery.OperatorIn, ids...)
			}
		}
	}
	return s.userRepo.Get(ctx, query)
}

func (s *userService) Find(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.userRepo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", id))
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	return s.userRepo.FindByEmails(ctx, emails...)
}

func (s *userService) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	return s.userRepo.FindByIds(ctx, ids...)
}

func (s *userService) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if err := s.policy.UpdatePreFetch(ctx, users...); err != nil {
		return nil, err
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	var updated []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if len(targets) != len(users) {
			return domain.NewError(domain.CodeUserUpdateNotFound)
		}
		if err := s.policy.UpdatePostFetch(ctx, targets, users); err != nil {
			return err
		}

		// Cross-user email conflict check
		emailOwnerMap := make(map[string]string)
		var emailsToCheck []string
		for _, u := range users {
			if u.Email != "" {
				emailsToCheck = append(emailsToCheck, u.Email)
				emailOwnerMap[u.Email] = u.Id
			}
		}
		if len(emailsToCheck) > 0 {
			existing, err := uow.User().FindByEmails(ctx, emailsToCheck...)
			if err != nil {
				return err
			}
			for _, e := range existing {
				if ownerID := emailOwnerMap[e.Email]; e.Id != ownerID {
					return domain.NewError(domain.CodeUserEmailConflict, domain.WithMetadata("email", e.Email))
				}
			}
		}

		// Build target map; apply same-role no-op filter; collect role-changed users
		targetMap := lo.SliceToMap(targets, func(t domain.User) (string, domain.User) { return t.Id, t })
		filteredUsers := make([]domain.User, len(users))
		copy(filteredUsers, users)
		var roleChainUsers []domain.User
		for i, u := range filteredUsers {
			if u.Role == "" {
				continue
			}
			target := targetMap[u.Id]
			if u.Role == target.Role {
				filteredUsers[i].Role = ""
				continue
			}
			roleChainUsers = append(roleChainUsers, domain.User{
				WalletAddress:             target.WalletAddress,
				EncryptedWalletPrivateKey: target.EncryptedWalletPrivateKey,
				Role:                      u.Role,
			})
		}

		// DB update
		updated, err = uow.User().Update(ctx, filteredUsers...)
		if err != nil {
			return err
		}

		// Chain sync for role changes
		if len(roleChainUsers) > 0 {
			if err := s.syncBlockchainRoles(ctx, roleChainUsers, domain.CodeUserUpdateBlockchainSyncFailed); err != nil {
				return err
			}
		}

		// Token revocation for email changes
		for _, u := range filteredUsers {
			if u.Email != "" {
				if _, err := uow.UserToken().RevokeByUserIdAndType(ctx, u.Id, domain.UserTokenTypeRefresh); err != nil {
					return err
				}
			}
		}

		return nil
	})
	return updated, err
}

func (s *userService) UpdateEmail(ctx context.Context, id string, email string, idToken string) (string, error) {
	payload, err := s.oauthClient.Validate(ctx, idToken, *s.cfg.GoogleClientID)
	if err != nil {
		return "", domain.NewError(domain.CodeUserEmailInvalidIdToken, domain.WithError(err))
	}
	tokenEmail, _ := payload.Claims["email"].(string)
	if tokenEmail != email {
		return "", domain.NewError(domain.CodeUserEmailMismatchedIdToken,
			domain.WithMetadata("token_email", tokenEmail),
			domain.WithMetadata("requested_email", email))
	}
	existing, err := s.userRepo.FindByEmails(ctx, email)
	if err != nil {
		return "", err
	}
	for _, u := range existing {
		if u.Id != id {
			return "", domain.NewError(domain.CodeUserEmailConflict, domain.WithMetadata("email", email))
		}
	}
	var updatedEmail string
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		updated, err := uow.User().Update(ctx, domain.User{Id: id, Email: email})
		if err != nil {
			return err
		}
		if len(updated) == 0 {
			return domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", id))
		}
		updatedEmail = updated[0].Email
		_, err = uow.UserToken().RevokeByUserIdAndType(ctx, id, domain.UserTokenTypeRefresh)
		return err
	})
	if err != nil {
		return "", err
	}
	return updatedEmail, nil
}

func (s *userService) updateRoleValidateAndPrepare(ctx context.Context, updates []domain.UserRoleUpdate, uow domain.UnitOfWork) ([]domain.User, error) {
	userIDs := make([]string, len(updates))
	for i, u := range updates {
		userIDs[i] = u.UserID
	}
	targetUsers, err := uow.User().FindByIds(ctx, userIDs...)
	if err != nil {
		return nil, err
	}
	if err := s.policy.UpdateRolePostFetch(ctx, targetUsers, updates...); err != nil {
		return nil, err
	}
	targetMap := lo.SliceToMap(targetUsers, func(t domain.User) (string, domain.User) { return t.Id, t })
	usersToUpdate := make([]domain.User, 0, len(updates))
	for _, u := range updates {
		target := targetMap[u.UserID]
		target.Role = u.Role
		usersToUpdate = append(usersToUpdate, target)
	}
	return usersToUpdate, nil
}

func (s *userService) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) {
	if err := s.policy.UpdateRolePreFetch(ctx, updates...); err != nil {
		return nil, 0, err
	}
	// Single UoW: fetch + post-fetch policy + DB update + chain sync all run in
	// one transaction so a chain-sync failure rolls back the DB write and there
	// is no read-then-write race window. Mirrors the Update flow.
	var updatedUsers []domain.User
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		usersToUpdate, err := s.updateRoleValidateAndPrepare(ctx, updates, uow)
		if err != nil {
			return err
		}
		updatedUsers, rowsAffected, err = uow.User().UpdateRole(ctx, usersToUpdate...)
		if err != nil {
			return err
		}
		return s.syncBlockchainRoles(ctx, usersToUpdate, domain.CodeUserRoleBlockchainSyncFailed)
	})
	if err != nil {
		return nil, 0, err
	}
	return updatedUsers, rowsAffected, err
}

func (s *userService) Delete(ctx context.Context, ids ...string) (int64, error) {
	if err := s.policy.DeletePreFetch(ctx, ids...); err != nil {
		return 0, err
	}
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targetUsers, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if err := s.policy.DeletePostFetch(ctx, targetUsers); err != nil {
			return err
		}
		if len(targetUsers) == 0 {
			return nil
		}
		var err2 error
		rowsAffected, err2 = uow.User().Delete(ctx, ids...)
		if err2 != nil {
			return err2
		}
		liveTargets := make([]domain.User, 0, len(targetUsers))
		for _, t := range targetUsers {
			if t.DeletedAt == nil {
				liveTargets = append(liveTargets, t)
			}
		}
		if len(liveTargets) == 0 {
			return nil
		}
		revocationUsers := make([]domain.User, len(liveTargets))
		for i, t := range liveTargets {
			revocationUsers[i] = domain.User{
				WalletAddress:             t.WalletAddress,
				EncryptedWalletPrivateKey: t.EncryptedWalletPrivateKey,
				Role:                      domain.RoleNone,
			}
		}
		// TODO: revoke user's credentials in DB (mark revoked_at, revoker_user_id)
		// and on-chain via CredentialRegistry.batchRevokeCredentialsWithSignature
		// once the credential feature is implemented. Without this, deleted users'
		// credentials remain active in the database and on-chain indefinitely.
		return s.syncBlockchainRoles(ctx, revocationUsers, domain.CodeUserDeleteBlockchainSyncFailed)
	})
	return rowsAffected, err
}

func (s *userService) TransferSuperAdmin(ctx context.Context, targetId string) error {
	if err := s.policy.TransferSuperAdminPreFetch(ctx, targetId); err != nil {
		return err
	}
	authUser := httpContext.MustGetUser(ctx)
	return s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.User().FindByIds(ctx, targetId)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return domain.NewError(domain.CodeUserTransferSuperAdminTargetNotFound,
				domain.WithMetadata("user_id", targetId))
		}
		target := targets[0]
		if target.DeletedAt != nil {
			return domain.NewError(domain.CodeUserTransferSuperAdminTrashedForbidden,
				domain.WithMetadata("user_id", targetId))
		}

		wallet := domain.WalletFromUser(*authUser)
		if err := s.authorityService.TransferSuperAdmin(ctx, wallet, target); err != nil {
			return domain.NewError(domain.CodeUserTransferSuperAdminBlockchainSyncFailed,
				domain.WithError(err))
		}

		if _, _, err := uow.User().UpdateRole(ctx,
			domain.User{
				Id:                        authUser.Id,
				Role:                      domain.RoleAdmin,
				WalletAddress:             authUser.WalletAddress,
				EncryptedWalletPrivateKey: authUser.EncryptedWalletPrivateKey,
			},
			domain.User{
				Id:                        target.Id,
				Role:                      domain.RoleSuperAdmin,
				WalletAddress:             target.WalletAddress,
				EncryptedWalletPrivateKey: target.EncryptedWalletPrivateKey,
			},
		); err != nil {
			return err
		}

		if _, err := uow.UserToken().RevokeByUserIdAndType(ctx, authUser.Id, domain.UserTokenTypeRefresh); err != nil {
			return err
		}
		if _, err := uow.UserToken().RevokeByUserIdAndType(ctx, target.Id, domain.UserTokenTypeRefresh); err != nil {
			return err
		}
		return nil
	})
}

func (s *userService) Restore(ctx context.Context, ids []string) ([]domain.User, int64, error) {
	if err := s.policy.RestorePreFetch(ctx, ids...); err != nil {
		return nil, 0, err
	}
	var restored []domain.User
	var count int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if err := s.policy.RestorePostFetch(ctx, targets); err != nil {
			return err
		}
		count, err = uow.User().Restore(ctx, ids...)
		if err != nil {
			return err
		}
		restored, err = uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		return s.syncBlockchainRoles(ctx, restored, domain.CodeUserRestoreBlockchainSyncFailed)
	})
	return restored, count, err
}
