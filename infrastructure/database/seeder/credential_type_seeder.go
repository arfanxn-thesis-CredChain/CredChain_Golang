package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
)

type CredentialTypeSeeder struct {
	repo domain.CredentialTypeRepository
}

func NewCredentialTypeSeeder(repo domain.CredentialTypeRepository) *CredentialTypeSeeder {
	return &CredentialTypeSeeder{repo: repo}
}

func (s *CredentialTypeSeeder) Name() string { return "credential-type" }

func (s *CredentialTypeSeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed-credential-type")
	rng := rand.New(rand.NewSource(seed))

	types := s.seedBuildTypes(rng)

	_, err := s.repo.Store(ctx, types...)
	if err != nil {
		return fmt.Errorf("credential type seeder: store: %w", err)
	}

	return nil
}

func (s *CredentialTypeSeeder) seedBuildTypes(rng *rand.Rand) []domain.CredentialType {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	defs := []struct {
		name   string
		active bool
	}{
		{"Ijazah", true},
		{"Transkrip Nilai", true},
		{"Sertifikat Kompetensi", true},
		{"Sertifikat Pelatihan", true},
		{"Surat Keterangan Pendamping Ijazah (SKPI)", true},
		{"Surat Keterangan Lulus (SKL)", true},
		{"Kartu Hasil Studi (KHS)", false},
	}

	types := make([]domain.CredentialType, len(defs))
	for i, d := range defs {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		types[i] = domain.CredentialType{
			Id:        deterministicULID(uint32(i + 1)),
			Name:      d.name,
			Active:    d.active,
			CreatedAt: createdAt,
			UpdatedAt: &updatedAt,
		}
	}
	return types
}
