package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
)

type CompetencySeeder struct {
	repo domain.CompetencyRepository
}

func NewCompetencySeeder(repo domain.CompetencyRepository) *CompetencySeeder {
	return &CompetencySeeder{repo: repo}
}

func (s *CompetencySeeder) Name() string { return "competency" }

func (s *CompetencySeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed-competency")
	rng := rand.New(rand.NewSource(seed))

	competencies := s.seedBuildCompetencies(rng)

	_, err := s.repo.Store(ctx, competencies...)
	if err != nil {
		return fmt.Errorf("competency seeder: store: %w", err)
	}

	return nil
}

func (s *CompetencySeeder) seedBuildCompetencies(rng *rand.Rand) []domain.Competency {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	names := []string{
		"Pemrograman Backend", "Pemrograman Frontend", "Analisis Data", "Keamanan Siber",
		"Jaringan Komputer", "Manajemen Proyek", "Basis Data", "Rekayasa Perangkat Lunak",
		"Kecerdasan Buatan", "Analisis Sistem", "Tata Kelola Teknologi Informasi", "Sistem Tertanam",
		"Elektronika", "Sistem Kendali", "Manajemen Pemasaran", "Manajemen Sumber Daya Manusia",
		"Manajemen Keuangan", "Kewirausahaan", "Akuntansi Keuangan", "Audit", "Perpajakan",
		"Psikologi Klinis", "Psikologi Industri dan Organisasi", "Psikometri", "Hukum Perdata",
		"Hukum Pidana", "Hukum Tata Negara", "Hukum Bisnis", "Jurnalistik", "Humas", "Komunikasi Pemasaran",
	}

	competencies := make([]domain.Competency, len(names))
	for i, n := range names {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		competencies[i] = domain.Competency{
			Id:        deterministicULID(uint32(i + 1)),
			Name:      n,
			CreatedAt: createdAt,
			UpdatedAt: &updatedAt,
		}
	}
	return competencies
}
