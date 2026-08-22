package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func testUserSeederConfig(mnemonic, encKey string) *config.Config {
	return &config.Config{
		HardhatMnemonic:     lo.ToPtr(mnemonic),
		WalletEncryptionKey: lo.ToPtr(encKey),
	}
}

func TestUserSeeder_Seeds15Users(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	userUnitRepo := user.NewGormUserUnitRepository(gormDB)
	ctx := context.Background()

	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())

	assert.NoError(t, seeder.NewUserUnitSeeder(userUnitRepo).Seed(ctx))

	s := seeder.NewUserSeeder(userRepo, userUnitRepo, testUserSeederConfig(seedMnemonic, encKey))

	err := s.Seed(ctx)
	assert.NoError(t, err)

	superAdmins, err := userRepo.FindByRole(ctx, domain.RoleSuperAdmin)
	assert.NoError(t, err)
	assert.Len(t, superAdmins, 1)
	assert.Equal(t, "arfan2173@gmail.com", superAdmins[0].Email)
	assert.Equal(t, "Muhammad Arfan", *superAdmins[0].Name)
	assert.NotNil(t, superAdmins[0].Number)
	assert.True(t, len(*superAdmins[0].Number) == 18)
	assert.NotNil(t, superAdmins[0].Meta)
	assert.Equal(t, "A1B2C3D4", superAdmins[0].Meta["key"])
	assert.Nil(t, superAdmins[0].DeletedAt)
	assert.Nil(t, superAdmins[0].UnitID, "super admin must not have a unit")

	admins, err := userRepo.FindByRole(ctx, domain.RoleAdmin)
	assert.NoError(t, err)
	assert.Len(t, admins, 1)
	assert.Equal(t, "arfanforproject@gmail.com", admins[0].Email)
	assert.NotNil(t, admins[0].Number)
	assert.Nil(t, admins[0].Meta)
	assert.Nil(t, admins[0].DeletedAt)
	assert.Nil(t, admins[0].UnitID, "admin must not have a unit")

	issuers, err := userRepo.FindByRole(ctx, domain.RoleIssuer)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(issuers), 1)
	hasEdy := false
	for _, u := range issuers {
		assert.NotNil(t, u.Number, "all users must have Number")
		assert.True(t, len(*u.Number) == 18, "issuer number must be 18-digit NIP")
		assert.Nil(t, u.UnitID, "issuer must not have a unit")
		if u.Email == "edysusilo17580@gmail.com" {
			hasEdy = true
			assert.NotNil(t, u.Meta)
			assert.Equal(t, "E5F6G7H8", u.Meta["key"])
			assert.Nil(t, u.DeletedAt)
		}
	}
	assert.True(t, hasEdy, "Edy Susilo should be an issuer")

	holders, err := userRepo.FindByRole(ctx, domain.RoleHolder)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(holders), 2)
	for _, u := range holders {
		assert.NotNil(t, u.Number, "all users must have Number")
		assert.True(t, len(*u.Number) == 8)
		assert.True(t, (*u.Number)[:4] == "2209")
		assert.NotNil(t, u.UnitID, "holder must have a unit")
	}

	total := len(superAdmins) + len(admins) + len(issuers) + len(holders)
	assert.Equal(t, 15, total)

	allUsers := append(append(superAdmins, admins...), append(issuers, holders...)...)
	for _, u := range allUsers {
		assert.NotNil(t, u.JoinedYear, "all seeded users must have a joined year: %s", u.Email)
		if u.JoinedYear != nil {
			allowed := []int{u.CreatedAt.Year() - 1, u.CreatedAt.Year(), u.CreatedAt.Year() + 1}
			assert.Contains(t, allowed, *u.JoinedYear, "joined year must be within +/-1 of created year: %s", u.Email)
		}
	}

	deletedCount := 0
	for _, groups := range [][]domain.User{superAdmins, admins, issuers, holders} {
		for _, u := range groups {
			if u.DeletedAt != nil {
				deletedCount++
			}
		}
	}
	assert.Equal(t, 5, deletedCount)

	annaDeleted := false
	for _, u := range holders {
		if u.Email == "annasorokin2173@gmail.com" && u.DeletedAt != nil {
			annaDeleted = true
		}
	}
	assert.True(t, annaDeleted, "Anna Sorokin should be soft-deleted")
}

func TestUserSeeder_Name(t *testing.T) {
	s := seeder.NewUserSeeder(nil, nil, nil)
	assert.Equal(t, "user", s.Name())
}

func TestUserSeeder_DeterministicRandomUsers(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)

	repo1 := user.NewGormUserRepository(gormDB1)
	repo2 := user.NewGormUserRepository(gormDB2)
	unitRepo1 := user.NewGormUserUnitRepository(gormDB1)
	unitRepo2 := user.NewGormUserUnitRepository(gormDB2)

	encKey := string(fixtures.TestWalletEncryptionKey())
	mnemonic := "test test test test test test test test test test test junk"

	ctx := context.Background()

	assert.NoError(t, seeder.NewUserUnitSeeder(unitRepo1).Seed(ctx))
	assert.NoError(t, seeder.NewUserUnitSeeder(unitRepo2).Seed(ctx))

	s1 := seeder.NewUserSeeder(repo1, unitRepo1, testUserSeederConfig(mnemonic, encKey))
	err := s1.Seed(ctx)
	assert.NoError(t, err)

	s2 := seeder.NewUserSeeder(repo2, unitRepo2, testUserSeederConfig(mnemonic, encKey))
	err = s2.Seed(ctx)
	assert.NoError(t, err)

	issuers1, _ := repo1.FindByRole(ctx, domain.RoleIssuer)
	issuers2, _ := repo2.FindByRole(ctx, domain.RoleIssuer)
	holders1, _ := repo1.FindByRole(ctx, domain.RoleHolder)
	holders2, _ := repo2.FindByRole(ctx, domain.RoleHolder)

	assert.Equal(t, len(issuers1), len(issuers2))
	assert.Equal(t, len(holders1), len(holders2))

	for i := range issuers1 {
		assert.Equal(t, issuers1[i].Email, issuers2[i].Email)
		assert.Equal(t, issuers1[i].Number, issuers2[i].Number)
		assert.Equal(t, issuers1[i].Id, issuers2[i].Id, "user id must be deterministic")
		assert.Equal(t, issuers1[i].JoinedYear, issuers2[i].JoinedYear, "joined year must be deterministic")
	}
	for i := range holders1 {
		assert.Equal(t, holders1[i].Email, holders2[i].Email)
		assert.Equal(t, holders1[i].Number, holders2[i].Number)
		assert.Equal(t, holders1[i].Id, holders2[i].Id, "user id must be deterministic")
		assert.Equal(t, holders1[i].JoinedYear, holders2[i].JoinedYear, "joined year must be deterministic")
	}
}
