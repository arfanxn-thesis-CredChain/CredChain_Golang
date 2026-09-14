package cmd

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"slices"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var seedChainNames []string

func init() {
	rootCmd.AddCommand(seedChainCmd)
	seedChainCmd.Flags().StringArrayVar(&seedChainNames, "names", nil, "Specific seeder names to run (default: run all)")
}

var seedChainCmd = &cobra.Command{
	Use:   "seed-chain",
	Short: "Register seeded users and credentials on the blockchain",
	Long: `Reads users and credentials from the database and registers roles and issuances on-chain.

Prerequisites:
  1. The 'seed' command must have been run first (users & credentials in DB)
  2. The SuperAdmin wallet must already have the SuperAdmin role on-chain
  3. The RPC_URL must point to a running Hardhat node

Examples:
  go run main.go seed-chain --env .env
  go run main.go seed-chain --env .env --names user
  go run main.go seed-chain --env .env --names credential`,
	Run: func(cmd *cobra.Command, args []string) {
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				credential.NewGormCredentialRepository,
				chain.NewClient,
				chain.NewAuthorityService,
				chain.NewRegistryService,
			),
			fx.Invoke(func(
				shutdowner fx.Shutdowner,
				cfg *config.Config,
				userRepo domain.UserRepository,
				credRepo domain.CredentialRepository,
				authorityService chain.AuthorityService,
				registryService chain.RegistryService,
				logger *zap.Logger,
			) {
				go func() {
					if err := seedChainRun(cfg, userRepo, credRepo, authorityService, registryService, seedChainNames, logger); err != nil {
						logger.Error("seed-chain failed", zap.Error(err))
					}
					shutdowner.Shutdown()
				}()
			}),
		)

		if err := app.Start(context.Background()); err != nil {
			log.Fatal(err)
		}

		<-app.Done()
	},
}

// seedChainRun orchestrates blockchain registration for users and credentials.
// It reads flags to decide which legs to run (default: both "user" and "credential").
func seedChainRun(
	cfg *config.Config,
	userRepo domain.UserRepository,
	credRepo domain.CredentialRepository,
	authorityService chain.AuthorityService,
	registryService chain.RegistryService,
	names []string,
	logger *zap.Logger,
) error {
	runAll := len(names) == 0
	runUser := runAll || slices.Contains(names, "user")
	runCredential := runAll || slices.Contains(names, "credential")

	ctx := context.Background()

	if runUser {
		if err := seedChainUsers(ctx, cfg, userRepo, authorityService, logger); err != nil {
			return err
		}
	}

	if runCredential {
		if err := seedChainCredentials(ctx, userRepo, credRepo, registryService, logger); err != nil {
			return err
		}
	}

	logger.Info("seed-chain completed successfully")
	return nil
}

// seedChainUsers reads all users from PostgreSQL via a single Get query,
// derives the SuperAdmin wallet (mnemonic index 1), and registers every
// non-None-role user on-chain in chunked batches of ≤100.
func seedChainUsers(
	ctx context.Context,
	cfg *config.Config,
	userRepo domain.UserRepository,
	authorityService chain.AuthorityService,
	logger *zap.Logger,
) error {
	mnemonic := seedGetHardhatMnemonic(cfg)

	logger.Info("reading seeded users from database for chain registration")

	allUsers, total, err := userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain users: read users: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("seed-chain users: no users in database — run 'seed' first")
	}

	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		if update.Role == domain.RoleNone {
			continue
		}
		usersToRegister = append(usersToRegister, update)
	}

	logger.Info("users loaded for chain registration",
		zap.Int("total_in_db", total),
		zap.Int("to_register", len(usersToRegister)),
	)

	privKey, _, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 1)
	if err != nil {
		return fmt.Errorf("seed-chain users: derive super admin key: %w", err)
	}

	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKey), []byte(*cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("seed-chain users: encrypt super admin key: %w", err)
	}

	addr, err := cryptoInfra.DeriveAddressFromPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("seed-chain users: derive super admin address: %w", err)
	}

	superAdminWallet := domain.Wallet{
		Address:             addr,
		EncryptedPrivateKey: encryptedKey,
	}

	const maxBatchRole = 100
	registeredCount := 0
	for start := 0; start < len(usersToRegister); start += maxBatchRole {
		end := min(start+maxBatchRole, len(usersToRegister))
		chunk := usersToRegister[start:end]

		logger.Info("registering users on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(usersToRegister)),
			zap.String("signer", superAdminWallet.Address),
		)

		if err := authorityService.UpdateUserRole(ctx, superAdminWallet, chunk...); err != nil {
			return fmt.Errorf("seed-chain users: on-chain registration chunk [%d:%d]: %w", start, end, err)
		}
		registeredCount += len(chunk)
	}

	logger.Info("users chain registration completed",
		zap.Int("users_registered", registeredCount),
	)

	return nil
}

// seedChainCredentials mints approved and revoked credentials on-chain using
// the respective issuer's wallet, writes back token IDs to Postgres, and
// revokes credentials where revoked_at is set.
func seedChainCredentials(
	ctx context.Context,
	userRepo domain.UserRepository,
	credRepo domain.CredentialRepository,
	registryService chain.RegistryService,
	logger *zap.Logger,
) error {
	logger.Info("reading credentials from database for on-chain issuance")

	users, _, err := userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain credentials: read users: %w", err)
	}
	userByID := make(map[string]domain.User, len(users))
	for _, u := range users {
		userByID[u.Id] = u
	}

	creds, _, err := credRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain credentials: read credentials: %w", err)
	}

	// Filter credentials that need to be on-chain: ApprovedAt != nil
	// Pending and rejected credentials do not exist on chain.
	var onChainCreds []domain.Credential
	for _, c := range creds {
		if c.ApprovedAt != nil {
			onChainCreds = append(onChainCreds, c)
		}
	}

	if len(onChainCreds) == 0 {
		logger.Info("no approved credentials to mint on-chain")
		return nil
	}

	logger.Info("credentials loaded for chain minting",
		zap.Int("total_creds", len(creds)),
		zap.Int("to_mint", len(onChainCreds)),
	)

	// Group credentials by issuer user ID so each batch is signed by that issuer's wallet
	credsByIssuer := make(map[string][]domain.Credential)
	for _, c := range onChainCreds {
		if c.IssuerUserID == nil {
			continue
		}
		credsByIssuer[*c.IssuerUserID] = append(credsByIssuer[*c.IssuerUserID], c)
	}

	const maxBatchIssue = 100
	totalMinted := 0
	var revokedCredsToSync []domain.Credential

	for issuerID, issuerCreds := range credsByIssuer {
		issuer, ok := userByID[issuerID]
		if !ok {
			return fmt.Errorf("seed-chain credentials: issuer %s not found in DB", issuerID)
		}
		issuerWallet := domain.WalletFromUser(issuer)

		for start := 0; start < len(issuerCreds); start += maxBatchIssue {
			end := min(start+maxBatchIssue, len(issuerCreds))
			chunk := issuerCreds[start:end]

			issuances := make([]chain.CredentialIssuance, len(chunk))
			for i, c := range chunk {
				holder, ok := userByID[c.HolderUserID]
				if !ok {
					return fmt.Errorf("seed-chain credentials: holder %s not found in DB", c.HolderUserID)
				}
				var exp uint64
				if c.ExpiresAt != nil {
					exp = uint64(c.ExpiresAt.Unix())
				}
				issuances[i] = chain.CredentialIssuance{
					HolderAddress: holder.WalletAddress,
					Hash:          c.FileHash,
					URI:           c.ID,
					IssuedAt:      uint64(c.IssuedAt.Unix()),
					ExpiresAt:     exp,
				}
			}

			logger.Info("minting credentials on-chain",
				zap.String("issuer_id", issuerID),
				zap.String("issuer_wallet", issuerWallet.Address),
				zap.Int("chunk_size", len(chunk)),
			)

			tokenIDs, err := registryService.IssueCredentials(ctx, issuerWallet, issuances...)
			if err != nil {
				return fmt.Errorf("seed-chain credentials: batch issue by issuer %s: %w", issuerID, err)
			}

			// Update token IDs in Postgres
			updates := make([]domain.Credential, len(chunk))
			for i, c := range chunk {
				tokStr := tokenIDs[i].String()
				c.TokenID = &tokStr
				updates[i] = domain.Credential{
					ID:      c.ID,
					TokenID: &tokStr,
				}
				if c.RevokedAt != nil {
					revokedCredsToSync = append(revokedCredsToSync, c)
				}
			}

			if _, err := credRepo.Update(ctx, updates...); err != nil {
				return fmt.Errorf("seed-chain credentials: update token IDs in DB: %w", err)
			}

			totalMinted += len(chunk)
		}
	}

	logger.Info("credentials minted on-chain", zap.Int("total_minted", totalMinted))

	// Revoke on-chain credentials for revoked rows
	if len(revokedCredsToSync) > 0 {
		logger.Info("revoking credentials on-chain", zap.Int("to_revoke", len(revokedCredsToSync)))

		revokedByRevoker := make(map[string][]domain.Credential)
		for _, c := range revokedCredsToSync {
			var revokerID string
			if c.RevokerUserID != nil && *c.RevokerUserID != "" {
				revokerID = *c.RevokerUserID
			} else if c.IssuerUserID != nil {
				revokerID = *c.IssuerUserID
			}
			if revokerID == "" {
				continue
			}
			revokedByRevoker[revokerID] = append(revokedByRevoker[revokerID], c)
		}

		for revokerID, revCreds := range revokedByRevoker {
			revoker, ok := userByID[revokerID]
			if !ok {
				return fmt.Errorf("seed-chain credentials: revoker %s not found in DB", revokerID)
			}
			revokerWallet := domain.WalletFromUser(revoker)

			tokenIDs := make([]*big.Int, 0, len(revCreds))
			for _, c := range revCreds {
				if c.TokenID == nil {
					continue
				}
				bi, ok := new(big.Int).SetString(*c.TokenID, 10)
				if !ok {
					return fmt.Errorf("seed-chain credentials: invalid token ID %s", *c.TokenID)
				}
				tokenIDs = append(tokenIDs, bi)
			}

			if len(tokenIDs) == 0 {
				continue
			}

			logger.Info("executing on-chain revocation",
				zap.String("revoker_id", revokerID),
				zap.String("revoker_wallet", revokerWallet.Address),
				zap.Int("count", len(tokenIDs)),
			)

			if err := registryService.RevokeCredentials(ctx, revokerWallet, tokenIDs...); err != nil {
				return fmt.Errorf("seed-chain credentials: batch revoke by revoker %s: %w", revokerID, err)
			}
		}

		logger.Info("on-chain revocations completed", zap.Int("revoked_count", len(revokedCredsToSync)))
	}

	return nil
}

// seedGetHardhatMnemonic resolves the mnemonic from config or returns the
// standard Hardhat default mnemonic.
func seedGetHardhatMnemonic(cfg *config.Config) string {
	if cfg.HardhatMnemonic != nil && *cfg.HardhatMnemonic != "" {
		return *cfg.HardhatMnemonic
	}
	return "test test test test test test test test test test test junk"
}
