package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"

	"github.com/samber/lo"
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
		// Standard approved active types
		{name: "Ijazah", active: true, approverID: &issuerID, isApproved: true},
		{name: "Transkrip Nilai", active: true, approverID: &issuerID, isApproved: true},
		{name: "Sertifikat Kompetensi", active: true, approverID: &issuerID, isApproved: true},
		{name: "Sertifikat Pelatihan", active: true, approverID: &issuerID, isApproved: true},
		{name: "Surat Keterangan Pendamping Ijazah (SKPI)", active: true, approverID: &issuerID, isApproved: true},
		{name: "Surat Keterangan Lulus (SKL)", active: true, approverID: &issuerID, isApproved: true},
		// Approved but deactivated
		{name: "Kartu Hasil Studi (KHS)", active: false, approverID: &issuerID, isApproved: true},
		// Pending proposal requested by holder
		{name: "Sertifikat Pelatihan Mandiri", active: false, submitterID: &holderID},
		// Rejected proposal requested by holder and reviewed by issuer
		{
			name:            "Piagam Penghargaan Kegiatan",
			active:          false,
			submitterID:     &holderID,
			rejecterID:      &issuerID,
			isRejected:      true,
			rejectionReason: lo.ToPtr("Bukan jenis kredensial formal yang diakui"),
		},
	}

	types := make([]domain.CredentialType, len(defs))
	for i, d := range defs {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		ct := domain.CredentialType{
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
			ct.ActivatedAt = &act
		}
		if d.isRejected {
			rej := createdAt.Add(2 * time.Hour)
			ct.RejectedAt = &rej
		}
		types[i] = ct
	}
	return types
}
