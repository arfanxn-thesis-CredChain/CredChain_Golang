package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserUnitSeeder_SeedsTree(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	repo := user.NewGormUserUnitRepository(gormDB)
	s := seeder.NewUserUnitSeeder(repo)
	ctx := context.Background()

	err := s.Seed(ctx)
	require.NoError(t, err)

	units, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, units, 11)

	byName := map[string]domain.UserUnit{}
	for _, u := range units {
		byName[u.Name] = u
	}

	faculties := map[string][]string{
		"Fakultas Teknik": {
			"Program Studi Teknik Informatika",
			"Program Studi Sistem Informasi",
			"Program Studi Teknik Elektro",
		},
		"Fakultas Ekonomi dan Bisnis": {
			"Program Studi Manajemen",
			"Program Studi Akuntansi",
		},
		"Fakultas Ilmu Sosial": {
			"Program Studi Psikologi",
			"Program Studi Hukum",
			"Program Studi Ilmu Komunikasi",
		},
	}
	for facName, programs := range faculties {
		fac, ok := byName[facName]
		require.True(t, ok)
		assert.Nil(t, fac.ParentId)
		for _, progName := range programs {
			prog, ok := byName[progName]
			require.True(t, ok)
			require.NotNil(t, prog.ParentId)
			assert.Equal(t, fac.Id, *prog.ParentId)
		}
	}
}

func TestUserUnitSeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := user.NewGormUserUnitRepository(gormDB1)
	repo2 := user.NewGormUserUnitRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewUserUnitSeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewUserUnitSeeder(repo2).Seed(ctx))

	u1, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	u2, err := repo2.Get(ctx, nil)
	require.NoError(t, err)
	require.Len(t, u1, len(u2))

	byName1 := map[string]domain.UserUnit{}
	for _, u := range u1 {
		byName1[u.Name] = u
	}
	for _, u := range u2 {
		assert.Equal(t, byName1[u.Name].Id, u.Id, "id must be stable across seeds: %s", u.Name)
	}
}

func TestUserUnitSeeder_Name(t *testing.T) {
	s := seeder.NewUserUnitSeeder(nil)
	assert.Equal(t, "user-unit", s.Name())
}
