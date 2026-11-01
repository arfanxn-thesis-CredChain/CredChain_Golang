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

func TestCredentialIssuerOrganizationSeeder_SeedsOrgs(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	repo := credential.NewGormCredentialIssuerOrganizationRepository(gormDB)
	s := seeder.NewCredentialIssuerOrganizationSeeder(repo)
	ctx := context.Background()

	err := s.Seed(ctx)
	require.NoError(t, err)

	orgs, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, orgs, 9)

	byName := map[string]domain.CredentialIssuerOrganization{}
	for _, o := range orgs {
		byName[o.Name] = o
	}

	expectedActive := []string{
		"Universitas Harkat Negeri",
		"Badan Nasional Sertifikasi Profesi (BNSP)",
		"Udemy",
		"DataCamp",
		"Huawei",
		"Cyfrin Updraft",
	}
	for _, n := range expectedActive {
		o, ok := byName[n]
		require.True(t, ok, "missing org: %s", n)
		assert.True(t, o.Active, "seeded organization must be active: %s", o.Name)
		assert.Equal(t, domain.ReferenceStatusApproved, o.Status())
		assert.NotNil(t, o.ApproverUserId)
		assert.NotNil(t, o.ActivatedAt)
	}

	deactivated, ok := byName["Lembaga Sertifikasi Profesi Vokasi"]
	require.True(t, ok)
	assert.False(t, deactivated.Active)
	assert.Equal(t, domain.ReferenceStatusApproved, deactivated.Status(), "deactivated approved org remains approved")
	assert.NotNil(t, deactivated.ActivatedAt)

	pending, ok := byName["Lembaga Pelatihan Swadaya Nusantara"]
	require.True(t, ok)
	assert.False(t, pending.Active)
	assert.Equal(t, domain.ReferenceStatusPending, pending.Status())
	assert.NotNil(t, pending.SubmitterUserId)
	assert.Nil(t, pending.ActivatedAt)
	assert.Nil(t, pending.RejectedAt)

	rejected, ok := byName["Organisasi Penerbit Tak Dikenal"]
	require.True(t, ok)
	assert.False(t, rejected.Active)
	assert.Equal(t, domain.ReferenceStatusRejected, rejected.Status())
	assert.NotNil(t, rejected.SubmitterUserId)
	assert.NotNil(t, rejected.RejecterUserId)
	assert.NotNil(t, rejected.RejectedAt)
	assert.NotNil(t, rejected.RejectionReason)
}

func TestCredentialIssuerOrganizationSeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := credential.NewGormCredentialIssuerOrganizationRepository(gormDB1)
	repo2 := credential.NewGormCredentialIssuerOrganizationRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewCredentialIssuerOrganizationSeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewCredentialIssuerOrganizationSeeder(repo2).Seed(ctx))

	o1, _, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	o2, _, err := repo2.Get(ctx, nil)
	require.NoError(t, err)
	require.Len(t, o1, len(o2))

	byName := map[string]string{}
	for _, o := range o1 {
		byName[o.Name] = o.Id
	}
	for _, o := range o2 {
		assert.Equal(t, byName[o.Name], o.Id, "id must be stable across seeds: %s", o.Name)
	}
}

func TestCredentialIssuerOrganizationSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialIssuerOrganizationSeeder(nil)
	assert.Equal(t, "credential-issuer-organization", s.Name())
}
