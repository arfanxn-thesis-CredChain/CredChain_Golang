package cmd

import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/seeder"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var seedNames []string

func init() {
	rootCmd.AddCommand(seedCmd)
	seedCmd.Flags().StringArrayVar(&seedNames, "names", nil, "Specific seeder names to run (default: run all)")
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run database seeders",
	Long: `Run database seeders to populate the database with initial data.

Use --names to run specific seeders (e.g., --names user). Without --names, all
registered seeders execute in registration order. Can be specified multiple times:

  --names user --names credential

The Hardhat mnemonic used to derive wallet keys defaults to the standard
Hardhat mnemonic ("test test test test test test test test test test test junk").
Override it via the HARDHAT_MNEMONIC environment variable.

Examples:
  go run main.go seed --env .env
  go run main.go seed --env .env --names user`,
	Run: func(cmd *cobra.Command, args []string) {
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				user.NewGormUserUnitRepository,
				credential.NewGormCredentialTypeRepository,
				credential.NewGormCredentialIssuerOrganizationRepository,
				credential.NewGormCompetencyRepository,
				func(
					userRepo domain.UserRepository,
					userUnitRepo domain.UserUnitRepository,
					credentialTypeRepo domain.CredentialTypeRepository,
					issuerOrgRepo domain.CredentialIssuerOrganizationRepository,
					competencyRepo domain.CompetencyRepository,
					cfg *config.Config,
				) *seeder.Registry {
					return seeder.NewRegistry(
						seeder.NewUserUnitSeeder(userUnitRepo),
						seeder.NewUserSeeder(userRepo, userUnitRepo, cfg),
						seeder.NewCredentialTypeSeeder(credentialTypeRepo),
						seeder.NewCredentialIssuerOrganizationSeeder(issuerOrgRepo),
						seeder.NewCompetencySeeder(competencyRepo),
					)
				},
			),
			fx.Invoke(func(shutdowner fx.Shutdowner, registry *seeder.Registry, logger *zap.Logger) {
				go func() {
					if err := seedRun(registry, seedNames, logger); err != nil {
						logger.Error("seed failed", zap.Error(err))
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

// seedRun executes the registry with the given seeder names.
func seedRun(registry *seeder.Registry, names []string, logger *zap.Logger) error {
	ctx := context.Background()

	if len(names) > 0 {
		logger.Info("running seeders", zap.Strings("names", names))
	} else {
		logger.Info("running all seeders")
	}

	if err := registry.Run(ctx, names...); err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	logger.Info("seed completed successfully")
	return nil
}
