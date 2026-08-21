package seeder_test

import (
	"context"
	"testing"

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

	orgs, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, orgs, 6)

	names := map[string]bool{}
	for _, o := range orgs {
		names[o.Name] = true
	}
	expected := []string{
		"Universitas Harkat Negeri",
		"Badan Nasional Sertifikasi Profesi (BNSP)",
		"Udemy",
		"DataCamp",
		"Huawei",
		"Cyfrin Updraft",
	}
	for _, n := range expected {
		assert.True(t, names[n], "missing org: %s", n)
	}
}

func TestCredentialIssuerOrganizationSeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := credential.NewGormCredentialIssuerOrganizationRepository(gormDB1)
	repo2 := credential.NewGormCredentialIssuerOrganizationRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewCredentialIssuerOrganizationSeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewCredentialIssuerOrganizationSeeder(repo2).Seed(ctx))

	o1, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	o2, err := repo2.Get(ctx, nil)
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
