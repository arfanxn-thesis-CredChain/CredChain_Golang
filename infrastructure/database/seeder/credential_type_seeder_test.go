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

	types, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, types, 9)

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
		assert.Equal(t, domain.ReferenceStatusApproved, c.Status())
		assert.NotNil(t, c.ApproverUserId)
		assert.NotNil(t, c.ActivatedAt)
	}

	khs, ok := byName["Kartu Hasil Studi (KHS)"]
	require.True(t, ok)
	assert.False(t, khs.Active)
	assert.Equal(t, domain.ReferenceStatusApproved, khs.Status(), "deactivated approved type remains approved")
	assert.NotNil(t, khs.ActivatedAt)

	pending, ok := byName["Sertifikat Pelatihan Mandiri"]
	require.True(t, ok)
	assert.False(t, pending.Active)
	assert.Equal(t, domain.ReferenceStatusPending, pending.Status())
	assert.NotNil(t, pending.SubmitterUserId)
	assert.Nil(t, pending.ActivatedAt)
	assert.Nil(t, pending.RejectedAt)

	rejected, ok := byName["Piagam Penghargaan Kegiatan"]
	require.True(t, ok)
	assert.False(t, rejected.Active)
	assert.Equal(t, domain.ReferenceStatusRejected, rejected.Status())
	assert.NotNil(t, rejected.SubmitterUserId)
	assert.NotNil(t, rejected.RejecterUserId)
	assert.NotNil(t, rejected.RejectedAt)
	assert.NotNil(t, rejected.RejectionReason)
}

func TestCredentialTypeSeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := credential.NewGormCredentialTypeRepository(gormDB1)
	repo2 := credential.NewGormCredentialTypeRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewCredentialTypeSeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewCredentialTypeSeeder(repo2).Seed(ctx))

	t1, _, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	t2, _, err := repo2.Get(ctx, nil)
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
