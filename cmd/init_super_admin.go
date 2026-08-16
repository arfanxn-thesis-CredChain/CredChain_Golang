package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"
	"github.com/samber/lo"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	initSuperAdminName      string
	initSuperAdminNumber    string
	initSuperAdminEmail     string
	initSuperAdminPrivKey   string
	initSuperAdminBirthDate string
	initSuperAdminGender    string
	initSuperAdminMeta      string
)

func init() {
	rootCmd.AddCommand(initSuperAdminCmd)

	initSuperAdminCmd.Flags().StringVar(&initSuperAdminName, "name", "", "Super admin name (optional)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminNumber, "number", "", "Super admin number/ID (optional)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminEmail, "email", "", "Super admin email (required)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminPrivKey, "private-key", "", "Super admin wallet private key (required)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminBirthDate, "birth-date", "", "Super admin birth date in ISO 8601 format (YYYY-MM-DD, optional)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminGender, "gender", "", "Super admin gender: male or female (optional)")
	initSuperAdminCmd.Flags().StringVar(&initSuperAdminMeta, "meta", "", "Super admin meta as JSON string (optional)")
}

// initSuperAdminValidateConfig validates required configuration for super admin initialization.
// Flags take priority over environment variables.
//
// Parameters:
//   - cfg: Config from environment variables
//   - email: Email from CLI flag (takes priority over cfg.InitialSuperAdminEmail)
//   - privKey: Private key from CLI flag (takes priority over cfg.InitialSuperAdminPrivKey)
//
// Returns:
//   - finalEmail: Resolved email address (flag > env)
//   - finalPrivKey: Resolved private key (flag > env)
//   - error: If required fields are missing
func initSuperAdminValidateConfig(cfg *config.Config, email, privKey string) (string, string, error) {
	finalEmail := email
	if finalEmail == "" {
		if cfg.InitialSuperAdminEmail == nil {
			return "", "", fmt.Errorf("email is required (use --email flag or INITIAL_SUPER_ADMIN_EMAIL env var)")
		}
		finalEmail = *cfg.InitialSuperAdminEmail
	}

	finalPrivKey := privKey
	if finalPrivKey == "" {
		if cfg.InitialSuperAdminPrivKey == nil {
			return "", "", fmt.Errorf("private key is required (use --private-key flag or INITIAL_SUPER_ADMIN_PRIVATE_KEY env var)")
		}
		finalPrivKey = *cfg.InitialSuperAdminPrivKey
	}

	if cfg.WalletEncryptionKey == nil {
		return "", "", fmt.Errorf("wallet encryption key is required (WALLET_ENCRYPTION_KEY env var)")
	}

	return finalEmail, finalPrivKey, nil
}

// initSuperAdminParseWallet derives wallet address from hex private key.
//
// Parameters:
//   - privKeyHex: Hexadecimal private key string (with or without 0x prefix)
//
// Returns:
//   - string: Ethereum wallet address in hexadecimal format
//   - error: If private key format is invalid
func initSuperAdminParseWallet(privKeyHex string) (string, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid private key format: %w", err)
	}

	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to cast public key to ecdsa")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA).Hex(), nil
}

// initSuperAdminEncryptKey encrypts private key for storage.
//
// Parameters:
//   - privKey: Raw private key hex string to encrypt
//   - encryptionKey: 32-byte encryption key from WALLET_ENCRYPTION_KEY env var
//
// Returns:
//   - string: Encrypted private key for database storage
//   - error: If encryption fails
func initSuperAdminEncryptKey(privKey, encryptionKey string) (string, error) {
	encrypted, err := cryptoInfra.Encrypt([]byte(privKey), []byte(encryptionKey))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %w", err)
	}

	return encrypted, nil
}

// initSuperAdminBuildUser constructs a domain.User with super admin role and all provided fields.
//
// Parameters:
//   - email: User email address (required)
//   - walletAddress: Ethereum wallet address derived from private key (required)
//   - encryptedKey: Encrypted wallet private key for storage (required)
//   - name: User name (optional, may be nil)
//   - number: User number/ID like employee or student number (optional, may be nil)
//   - birthDate: User birth date in ISO 8601 format (optional, may be nil)
//   - gender: User gender (optional, may be nil)
//   - meta: User metadata as JSON object (optional, may be nil)
//
// Returns:
//   - domain.User: Complete user entity with RoleSuperAdmin
func initSuperAdminBuildUser(
	email string,
	walletAddress string,
	encryptedKey string,
	name *string,
	number *string,
	birthDate *time.Time,
	gender *domain.Gender,
	meta map[string]any,
) domain.User {
	return domain.User{
		Name:                      name,
		Number:                    number,
		Email:                     email,
		BirthDate:                 birthDate,
		Gender:                    gender,
		Meta:                      meta,
		Role:                      domain.RoleSuperAdmin,
		WalletAddress:             walletAddress,
		EncryptedWalletPrivateKey: encryptedKey,
	}
}

// initSuperAdminGetBirthDate resolves birth date from CLI flag or environment variable.
// Flag takes priority over environment variable.
func initSuperAdminGetBirthDate(cfg *config.Config, birthDateFlag string) *time.Time {
	if birthDateFlag != "" {
		t, err := time.Parse(time.DateOnly, birthDateFlag)
		if err == nil {
			return &t
		}
	}
	return cfg.InitialSuperAdminBirthDate
}

// initSuperAdminGetMeta resolves meta JSON from CLI flag or environment variable.
// Flag takes priority over environment variable.
func initSuperAdminGetMeta(cfg *config.Config, metaFlag string) map[string]any {
	if metaFlag != "" {
		var result map[string]any
		if err := json.Unmarshal([]byte(metaFlag), &result); err == nil {
			return result
		}
	}
	return cfg.InitialSuperAdminMeta
}

// initSuperAdminGetGender resolves gender from CLI flag or environment variable.
// Flag takes priority over environment variable. Returns nil if not set.
// Fatals if an invalid value is provided.
func initSuperAdminGetGender(cfg *config.Config, genderFlag string) *domain.Gender {
	raw := genderFlag
	if raw == "" && cfg.InitialSuperAdminGender != nil {
		raw = *cfg.InitialSuperAdminGender
	}
	if raw == "" {
		return nil
	}
	switch raw {
	case "male", "female":
		g := domain.Gender(raw)
		return &g
	default:
		log.Fatalf("invalid gender %q: must be one of: male, female", raw)
		return nil
	}
}

// initSuperAdminGetString resolves a string field from CLI flag or environment variable.
// Flag takes priority over environment variable.
func initSuperAdminGetString(cfgVal *string, flagVal string) *string {
	if flagVal != "" {
		return &flagVal
	}
	return cfgVal
}

// getStringValue returns the string value or "(not set)" for nil pointers.
func getStringValue(s *string) string {
	if s == nil {
		return "(not set)"
	}
	return *s
}

// getTimeValue returns the time or zero time for nil pointers.
func getTimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// initSuperAdminVerifyOnChainRole verifies that the wallet address has SuperAdmin role on-chain.
// Returns an error if the wallet does not have the required role.
//
// Parameters:
//   - ctx: Context for timeout/cancellation control
//   - authorityService: AuthorityService for blockchain role lookup
//   - walletAddress: Ethereum wallet address to verify (hex string with "0x" prefix)
//
// Returns:
//   - error: nil if verification passes, error if wallet doesn't have SuperAdmin role
func initSuperAdminVerifyOnChainRole(ctx context.Context, authorityService chain.AuthorityService, walletAddress string) error {
	onChainRole, err := authorityService.FindRole(ctx, walletAddress)
	if err != nil {
		return fmt.Errorf("failed to verify on-chain role: %w", err)
	}

	if onChainRole != domain.RoleSuperAdmin {
		return fmt.Errorf("wallet address %s does not have SuperAdmin role on-chain (current role: %s). Please ensure the wallet is registered as SuperAdmin in the CredentialAuthority contract before initializing", walletAddress, onChainRole)
	}

	return nil
}

// initSuperAdmin is the main FX-invoked function
func initSuperAdmin(cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, logger *zap.Logger) error {
	birthDate := initSuperAdminGetBirthDate(cfg, initSuperAdminBirthDate)
	gender := initSuperAdminGetGender(cfg, initSuperAdminGender)
	meta := initSuperAdminGetMeta(cfg, initSuperAdminMeta)
	name := initSuperAdminGetString(cfg.InitialSuperAdminName, initSuperAdminName)
	number := initSuperAdminGetString(cfg.InitialSuperAdminNumber, initSuperAdminNumber)

	email, privKey, err := initSuperAdminValidateConfig(cfg, initSuperAdminEmail, initSuperAdminPrivKey)
	if err != nil {
		return err
	}

	walletAddress, err := initSuperAdminParseWallet(privKey)
	if err != nil {
		return err
	}

	// Verify the wallet address has SuperAdmin role on-chain
	if err := initSuperAdminVerifyOnChainRole(context.Background(), authorityService, walletAddress); err != nil {
		return err
	}

	// Check if any super admin already exists in the database.
	// FindByRole is unscoped (returns trashed users too), so filter to live rows
	// only — a system whose previous SuperAdmin was soft-deleted must be able to
	// re-initialize.
	existingSuperAdmins, err := userRepo.FindByRole(context.Background(), domain.RoleSuperAdmin)
	if err != nil {
		return err
	}
	liveSuperAdmins := lo.CountBy(existingSuperAdmins, func(u domain.User) bool {
		return u.DeletedAt == nil
	})

	if liveSuperAdmins > 0 {
		msg := "super admin already exists in database"
		logger.Error(msg)
		return fmt.Errorf("%s", msg)
	}

	encryptedKey, err := initSuperAdminEncryptKey(privKey, *cfg.WalletEncryptionKey)
	if err != nil {
		return err
	}

	adminUser := initSuperAdminBuildUser(email, walletAddress, encryptedKey, name, number, birthDate, gender, meta)

	_, err = userRepo.Store(context.Background(), adminUser)
	if err != nil {
		return err
	}

	logger.Info("super admin initialized",
		zap.String("email", adminUser.Email),
		zap.String("walletAddress", adminUser.WalletAddress),
		zap.String("name", getStringValue(adminUser.Name)),
		zap.String("number", getStringValue(adminUser.Number)),
		zap.Time("birthDate", getTimeValue(adminUser.BirthDate)),
		zap.Any("gender", adminUser.Gender),
		zap.Any("meta", adminUser.Meta),
	)
	return nil
}

var initSuperAdminCmd = &cobra.Command{
	Use:   "init-super-admin",
	Short: "Initializes the Super Admin based on .env config",
	Long: `Creates the inaugural Super Admin securely in Postgres, parsing the Ethereum Wallet automatically from the given private key.

Pre-Initialization Checks:
  1. On-Chain Verification: Verifies the provided wallet address already has
     the SuperAdmin role in the CredentialAuthority smart contract. If the
     wallet does not have SuperAdmin role on-chain, initialization fails.
  2. Database Check: Verifies no SuperAdmin user exists in the database.
     Only one SuperAdmin is allowed per database instance.

Environment Variables:
  INITIAL_SUPER_ADMIN_NAME         Super admin name (optional)
  INITIAL_SUPER_ADMIN_NUMBER       Super admin number/ID like employee or student number (optional)
  INITIAL_SUPER_ADMIN_EMAIL        Super admin email address (required)
  INITIAL_SUPER_ADMIN_PRIVATE_KEY  Super admin wallet private key, 64-char hex with 0x prefix (required)
  INITIAL_SUPER_ADMIN_BIRTH_DATE   Super admin birth date in ISO 8601 format YYYY-MM-DD (optional)
  INITIAL_SUPER_ADMIN_GENDER       Super admin gender: male or female (optional)
  INITIAL_SUPER_ADMIN_META         Super admin metadata as JSON string (optional)

CLI Flags (take priority over env vars):
  --name          Super admin name (optional)
  --number        Super admin number/ID (optional)
  --email         Super admin email (required)
  --private-key   Super admin wallet private key (required)
  --birth-date    Super admin birth date (YYYY-MM-DD, optional)
  --gender        Super admin gender: male or female (optional)
  --meta          Super admin meta as JSON string (optional)

Examples:
  # Use environment variables only
  make init-super-admin

  # Override email with CLI flag
  go run main.go init-super-admin --email admin@example.com --private-key 0x...

  # Set all fields via flags
  go run main.go init-super-admin \
    --email admin@example.com \
    --private-key 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d \
    --name "Admin Name" \
    --number 1234 \
    --birth-date 2000-01-01 \
    --meta '{"department":"engineering"}'

  # Mix env and flags (flags take priority)
  INITIAL_SUPER_ADMIN_EMAIL=admin@example.com go run main.go init-super-admin --name "Custom Name" --private-key 0x...`,
	Run: func(cmd *cobra.Command, args []string) {
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				chain.NewClient,
				chain.NewAuthorityService,
			),
			fx.Invoke(func(shutdowner fx.Shutdowner, cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, logger *zap.Logger) {
				go func() {
					if err := initSuperAdmin(cfg, userRepo, authorityService, logger); err != nil {
						logger.Error("init-super-admin failed", zap.Error(err))
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
