package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"

	"github.com/samber/lo"
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
	issuerID := deterministicULID(3)
	holderID := deterministicULID(4)

	defs := []struct {
		name            string
		active          bool
		submitterID     *string
		approverID      *string
		rejecterID      *string
		rejectionReason *string
		isApproved      bool
		isRejected      bool
	}{
		// Standard approved active orgs
		{name: "Universitas Harkat Negeri", active: true, approverID: &issuerID, isApproved: true},
		{name: "Badan Nasional Sertifikasi Profesi (BNSP)", active: true, approverID: &issuerID, isApproved: true},
		{name: "Udemy", active: true, approverID: &issuerID, isApproved: true},
		{name: "DataCamp", active: true, approverID: &issuerID, isApproved: true},
		{name: "Huawei", active: true, approverID: &issuerID, isApproved: true},
		{name: "Cyfrin Updraft", active: true, approverID: &issuerID, isApproved: true},
		// Approved but deactivated
		{name: "Lembaga Sertifikasi Profesi Vokasi", active: false, approverID: &issuerID, isApproved: true},
		// Pending proposal requested by holder
		{name: "Lembaga Pelatihan Swadaya Nusantara", active: false, submitterID: &holderID},
		// Rejected proposal requested by holder and reviewed by issuer
		{
			name:            "Organisasi Penerbit Tak Dikenal",
			active:          false,
			submitterID:     &holderID,
			rejecterID:      &issuerID,
			isRejected:      true,
			rejectionReason: lo.ToPtr("Organisasi penerbit tidak terverifikasi atau tidak memiliki akreditasi resmi"),
		},
	}

	orgs := make([]domain.CredentialIssuerOrganization, len(defs))
	for i, d := range defs {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		org := domain.CredentialIssuerOrganization{
			Id:              deterministicULID(uint32(i + 1)),
			SubmitterUserId: d.submitterID,
			ApproverUserId:  d.approverID,
			RejecterUserId:  d.rejecterID,
			Name:            d.name,
			Active:          d.active,
			RejectionReason: d.rejectionReason,
			CreatedAt:       createdAt,
			UpdatedAt:       &updatedAt,
		}
		if d.isApproved {
			act := createdAt.Add(time.Hour)
			org.ActivatedAt = &act
		}
		if d.isRejected {
			rej := createdAt.Add(2 * time.Hour)
			org.RejectedAt = &rej
		}
		orgs[i] = org
	}
	return orgs
}
