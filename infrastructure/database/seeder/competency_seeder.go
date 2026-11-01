package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"

	"github.com/samber/lo"
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
	issuerID := deterministicULID(3)
	holderID := deterministicULID(4)

	names := []string{
		"Pemrograman Backend", "Pemrograman Frontend", "Analisis Data", "Keamanan Siber",
		"Jaringan Komputer", "Manajemen Proyek", "Basis Data", "Rekayasa Perangkat Lunak",
		"Kecerdasan Buatan", "Analisis Sistem", "Tata Kelola Teknologi Informasi", "Sistem Tertanam",
		"Elektronika", "Sistem Kendali", "Manajemen Pemasaran", "Manajemen Sumber Daya Manusia",
		"Manajemen Keuangan", "Kewirausahaan", "Akuntansi Keuangan", "Audit", "Perpajakan",
		"Psikologi Klinis", "Psikologi Industri dan Organisasi", "Psikometri", "Hukum Perdata",
		"Hukum Pidana", "Hukum Tata Negara", "Hukum Bisnis", "Jurnalistik", "Humas", "Komunikasi Pemasaran",
	}

	type compDef struct {
		name            string
		active          bool
		submitterID     *string
		approverID      *string
		rejecterID      *string
		rejectionReason *string
		isApproved      bool
		isRejected      bool
	}

	defs := make([]compDef, 0, len(names)+3)
	for _, n := range names {
		defs = append(defs, compDef{
			name:       n,
			active:     true,
			approverID: &issuerID,
			isApproved: true,
		})
	}

	// Approved but deactivated
	defs = append(defs, compDef{
		name:       "Bahasa Pemrograman Pascal",
		active:     false,
		approverID: &issuerID,
		isApproved: true,
	})

	// Pending proposal requested by holder
	defs = append(defs, compDef{
		name:        "Kompetensi Belum Terdaftar",
		active:      false,
		submitterID: &holderID,
	})

	// Rejected proposal requested by holder and reviewed by issuer
	defs = append(defs, compDef{
		name:            "Keahlian Supranatural",
		active:          false,
		submitterID:     &holderID,
		rejecterID:      &issuerID,
		isRejected:      true,
		rejectionReason: lo.ToPtr("Bukan kompetensi profesional atau akademik yang valid"),
	})

	competencies := make([]domain.Competency, len(defs))
	for i, d := range defs {
		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)
		comp := domain.Competency{
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
			comp.ActivatedAt = &act
		}
		if d.isRejected {
			rej := createdAt.Add(2 * time.Hour)
			comp.RejectedAt = &rej
		}
		competencies[i] = comp
	}
	return competencies
}
