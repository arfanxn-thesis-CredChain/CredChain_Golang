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

func TestCompetencySeeder_SeedsCompetencies(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	repo := credential.NewGormCompetencyRepository(gormDB)
	s := seeder.NewCompetencySeeder(repo)
	ctx := context.Background()

	err := s.Seed(ctx)
	require.NoError(t, err)

	competencies, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, competencies, 31)

	names := map[string]bool{}
	for _, c := range competencies {
		names[c.Name] = true
	}
	expected := []string{
		"Pemrograman Backend", "Pemrograman Frontend", "Analisis Data", "Keamanan Siber",
		"Jaringan Komputer", "Manajemen Proyek", "Basis Data", "Rekayasa Perangkat Lunak",
		"Kecerdasan Buatan", "Analisis Sistem", "Tata Kelola Teknologi Informasi", "Sistem Tertanam",
		"Elektronika", "Sistem Kendali", "Manajemen Pemasaran", "Manajemen Sumber Daya Manusia",
		"Manajemen Keuangan", "Kewirausahaan", "Akuntansi Keuangan", "Audit", "Perpajakan",
		"Psikologi Klinis", "Psikologi Industri dan Organisasi", "Psikometri", "Hukum Perdata",
		"Hukum Pidana", "Hukum Tata Negara", "Hukum Bisnis", "Jurnalistik", "Humas", "Komunikasi Pemasaran",
	}
	for _, n := range expected {
		assert.True(t, names[n], "missing competency: %s", n)
	}
}

func TestCompetencySeeder_DeterministicIDs(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)
	repo1 := credential.NewGormCompetencyRepository(gormDB1)
	repo2 := credential.NewGormCompetencyRepository(gormDB2)
	ctx := context.Background()

	require.NoError(t, seeder.NewCompetencySeeder(repo1).Seed(ctx))
	require.NoError(t, seeder.NewCompetencySeeder(repo2).Seed(ctx))

	c1, _, err := repo1.Get(ctx, nil)
	require.NoError(t, err)
	c2, _, err := repo2.Get(ctx, nil)
	require.NoError(t, err)
	require.Len(t, c1, len(c2))

	byName := map[string]string{}
	for _, c := range c1 {
		byName[c.Name] = c.Id
	}
	for _, c := range c2 {
		assert.Equal(t, byName[c.Name], c.Id, "id must be stable across seeds: %s", c.Name)
	}
}

func TestCompetencySeeder_Name(t *testing.T) {
	s := seeder.NewCompetencySeeder(nil)
	assert.Equal(t, "competency", s.Name())
}
