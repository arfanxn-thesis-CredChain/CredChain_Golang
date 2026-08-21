package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialTypeSeeder_SeedsTypes(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	repo := credential.NewGormCredentialTypeRepository(gormDB)
	s := seeder.NewCredentialTypeSeeder(repo)
	ctx := context.Background()

	err := s.Seed(ctx)
	require.NoError(t, err)

	types, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, types, 7)

	byName := map[string]domain.CredentialType{}
	for _, c := range types {
		byName[c.Name] = c
	}

	activeNames := []string{
		"Ijazah",
		"Transkrip Nilai",
		"Sertifikat Kompetensi",
		"Sertifikat Pelatihan",
		"Surat Keterangan Pendamping Ijazah (SKPI)",
		"Surat Keterangan Lulus (SKL)",
	}
	for _, n := range activeNames {
		c, ok := byName[n]
		require.True(t, ok)
		assert.True(t, c.Active)
	}

	khs, ok := byName["Kartu Hasil Studi (KHS)"]
	require.True(t, ok)
	assert.False(t, khs.Active)
}

func TestCredentialTypeSeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := credential.NewGormCredentialTypeRepository(gormDB1)
	repo2 := credential.NewGormCredentialTypeRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewCredentialTypeSeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewCredentialTypeSeeder(repo2).Seed(ctx))

	t1, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	t2, err := repo2.Get(ctx, nil)
	require.NoError(t, err)
	require.Len(t, t1, len(t2))

	byName := map[string]domain.CredentialType{}
	for _, c := range t1 {
		byName[c.Name] = c
	}
	for _, c := range t2 {
		assert.Equal(t, byName[c.Name].Id, c.Id, "id must be stable across seeds: %s", c.Name)
	}
}

func TestCredentialTypeSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialTypeSeeder(nil)
	assert.Equal(t, "credential-type", s.Name())
}
