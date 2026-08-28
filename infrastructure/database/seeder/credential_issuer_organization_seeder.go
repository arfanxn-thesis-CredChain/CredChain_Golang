package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
)

type CredentialIssuerOrganizationSeeder struct {
	repo domain.CredentialIssuerOrganizationRepository
}

func NewCredentialIssuerOrganizationSeeder(repo domain.CredentialIssuerOrganizationRepository) *CredentialIssuerOrganizationSeeder {
	return &CredentialIssuerOrganizationSeeder{repo: repo}
}

func (s *CredentialIssuerOrganizationSeeder) Name() string { return "credential-issuer-organization" }

func (s *CredentialIssuerOrganizationSeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed-issuer-organization")
	rng := rand.New(rand.NewSource(seed))

	orgs := s.seedBuildOrgs(rng)

	_, err := s.repo.Store(ctx, orgs...)
	if err != nil {
		return fmt.Errorf("credential issuer organization seeder: store: %w", err)
	}

	return nil
}

func (s *CredentialIssuerOrganizationSeeder) seedBuildOrgs(rng *rand.Rand) []domain.CredentialIssuerOrganization {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	names := []string{
		"Universitas Harkat Negeri",
		"Badan Nasional Sertifikasi Profesi (BNSP)",
		"Udemy",
		"DataCamp",
		"Huawei",
		"Cyfrin Updraft",
	}

	orgs := make([]domain.CredentialIssuerOrganization, len(names))
	for i, n := range names {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		orgs[i] = domain.CredentialIssuerOrganization{
			Id:   deterministicULID(uint32(i + 1)),
			Name: n,
			// Explicit: the GORM tag omits `default:true`, so the zero value
			// would insert every seeded row as inactive.
			Active:    true,
			CreatedAt: createdAt,
			UpdatedAt: &updatedAt,
		}
	}
	return orgs
}
