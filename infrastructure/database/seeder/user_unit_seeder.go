package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"

	"github.com/samber/lo"
)

type UserUnitSeeder struct {
	repo domain.UserUnitRepository
}

func NewUserUnitSeeder(repo domain.UserUnitRepository) *UserUnitSeeder {
	return &UserUnitSeeder{repo: repo}
}

func (s *UserUnitSeeder) Name() string { return "user-unit" }

func (s *UserUnitSeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed-user-unit")
	rng := rand.New(rand.NewSource(seed))

	units := s.seedBuildUnits(rng)

	_, err := s.repo.Store(ctx, units...)
	if err != nil {
		return fmt.Errorf("user unit seeder: store: %w", err)
	}

	return nil
}

func (s *UserUnitSeeder) seedBuildUnits(rng *rand.Rand) []domain.UserUnit {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var seq uint32
	newUnit := func(name string, parentId *string) domain.UserUnit {
		seq++
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		return domain.UserUnit{
			Id:        deterministicULID(seq),
			ParentId:  parentId,
			Name:      name,
			CreatedAt: createdAt,
			UpdatedAt: &updatedAt,
		}
	}

	faculties := []struct {
		name     string
		programs []string
	}{
		{
			name: "Fakultas Teknik",
			programs: []string{
				"Program Studi Teknik Informatika",
				"Program Studi Sistem Informasi",
				"Program Studi Teknik Elektro",
			},
		},
		{
			name: "Fakultas Ekonomi dan Bisnis",
			programs: []string{
				"Program Studi Manajemen",
				"Program Studi Akuntansi",
			},
		},
		{
			name: "Fakultas Ilmu Sosial",
			programs: []string{
				"Program Studi Psikologi",
				"Program Studi Hukum",
				"Program Studi Ilmu Komunikasi",
			},
		},
	}

	units := make([]domain.UserUnit, 0, 11)
	for _, f := range faculties {
		fac := newUnit(f.name, nil)
		units = append(units, fac)
		for _, p := range f.programs {
			units = append(units, newUnit(p, lo.ToPtr(fac.Id)))
		}
	}

	return units
}
