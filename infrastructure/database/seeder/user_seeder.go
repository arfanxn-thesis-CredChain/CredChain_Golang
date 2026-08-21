package seeder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/samber/lo"
)

type UserSeeder struct {
	repo     domain.UserRepository
	unitRepo domain.UserUnitRepository
	cfg      *config.Config
}

func NewUserSeeder(repo domain.UserRepository, unitRepo domain.UserUnitRepository, cfg *config.Config) *UserSeeder {
	return &UserSeeder{repo: repo, unitRepo: unitRepo, cfg: cfg}
}

func (s *UserSeeder) Name() string { return "user" }

func (s *UserSeeder) Seed(ctx context.Context) error {
	units, err := s.unitRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("user seeder: fetch units: %w", err)
	}
	programs := seedLeafUnits(units)

	seed := hashToSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))

	users := s.seedBuildUsers(rng)
	seedAssignUnitIDs(users, programs)

	_, err = s.repo.Store(ctx, users...)
	if err != nil {
		return fmt.Errorf("user seeder: store: %w", err)
	}

	return nil
}

func (s *UserSeeder) seedBuildUsers(rng *rand.Rand) []domain.User {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var nipSeq, nimSeq int
	users := make([]domain.User, 15)

	createdAt0 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[0] = s.seedBuildUser(seedBuildUserParams{
		index: 1, name: "Muhammad Arfan", email: "arfan2173@gmail.com",
		birthDate: seedMustParseDate("2003-07-21"),
		gender:    seedGenderPtr(domain.GenderMale),
		meta:      map[string]any{"key": "A1B2C3D4"},
		role:      domain.RoleSuperAdmin,
		number:    seedGenerateNIP(time.Date(2003, 7, 21, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderMale), &nipSeq),
		createdAt: createdAt0,
		updatedAt: lo.ToPtr(createdAt0.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt1 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[1] = s.seedBuildUser(seedBuildUserParams{
		index: 2, name: "Project", email: "arfanforproject@gmail.com",
		birthDate: seedMustParseDate("1992-05-15"),
		role:      domain.RoleAdmin,
		number:    seedGenerateNIP(time.Date(1992, 5, 15, 0, 0, 0, 0, time.UTC), nil, &nipSeq),
		createdAt: createdAt1,
		updatedAt: lo.ToPtr(createdAt1.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt2 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[2] = s.seedBuildUser(seedBuildUserParams{
		index: 3, name: "Edy Susilo", email: "edysusilo17580@gmail.com",
		birthDate: seedMustParseDate("1980-05-17"),
		gender:    seedGenderPtr(domain.GenderMale),
		meta:      map[string]any{"key": "E5F6G7H8"},
		role:      domain.RoleIssuer,
		number:    seedGenerateNIP(time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderMale), &nipSeq),
		createdAt: createdAt2,
		updatedAt: lo.ToPtr(createdAt2.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt3 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[3] = s.seedBuildUser(seedBuildUserParams{
		index: 4, name: "Liesbeth Stifanny", email: "liesbethsh19@gmail.com",
		birthDate: seedMustParseDate("2003-09-19"),
		gender:    seedGenderPtr(domain.GenderFemale),
		role:      domain.RoleHolder,
		number:    seedGenerateNIM(&nimSeq),
		createdAt: createdAt3,
		updatedAt: &createdAt3,
	})

	createdAt4 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[4] = s.seedBuildUser(seedBuildUserParams{
		index: 5, name: "Anna Sorokin", email: "annasorokin2173@gmail.com",
		gender:    seedGenderPtr(domain.GenderFemale),
		meta:      map[string]any{"key": "I9J0K1L2"},
		role:      domain.RoleHolder,
		number:    seedGenerateNIM(&nimSeq),
		createdAt: createdAt4,
		updatedAt: &createdAt4,
		deletedAt: lo.ToPtr(createdAt4.Add(time.Duration(1+rng.Int63n(180)) * 24 * time.Hour)),
	})

	for i := range 10 {
		idx := i + 5
		walletIdx := uint32(i + 6)
		role := seedRandomUserRole(rng)
		name := seedRandomIndonesianName(rng)
		email := seedNameToEmail(name, idx)
		birthDate := seedRandomBirthDate(rng)
		gender := seedRandomGender(rng)

		var meta map[string]any
		if i%2 == 0 {
			meta = map[string]any{"key": seedRandomAlphaKey(rng)}
		}

		var number string
		if role == domain.RoleIssuer {
			number = seedGenerateNIP(birthDate, &gender, &nipSeq)
		} else {
			number = seedGenerateNIM(&nimSeq)
		}

		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		var updatedAt *time.Time
		if role == domain.RoleIssuer {
			updatedAt = lo.ToPtr(createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour))
		} else {
			updatedAt = &createdAt
		}

		var deletedAt *time.Time
		if i >= 5 && i <= 8 {
			t := updatedAt
			if t == nil {
				t = &createdAt
			}
			deletedAt = lo.ToPtr((*t).Add(time.Duration(1+rng.Int63n(180)) * 24 * time.Hour))
		}

		users[idx] = s.seedBuildUser(seedBuildUserParams{
			index:     walletIdx,
			name:      name,
			email:     email,
			birthDate: &birthDate,
			gender:    &gender,
			meta:      meta,
			role:      role,
			number:    number,
			createdAt: createdAt,
			updatedAt: updatedAt,
			deletedAt: deletedAt,
		})
	}

	return users
}

type seedBuildUserParams struct {
	index     uint32
	name      string
	email     string
	birthDate *time.Time
	gender    *domain.Gender
	meta      map[string]any
	role      domain.Role
	number    string
	createdAt time.Time
	updatedAt *time.Time
	deletedAt *time.Time
}

func (s *UserSeeder) seedBuildUser(p seedBuildUserParams) domain.User {
	mnemonic := seedMnemonic(s.cfg)
	privKeyHex, address, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, p.index)
	if err != nil {
		panic(fmt.Sprintf("failed to derive key for index %d: %v", p.index, err))
	}
	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(*s.cfg.WalletEncryptionKey))
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt key for index %d: %v", p.index, err))
	}
	updatedAt := p.updatedAt
	if updatedAt == nil {
		updatedAt = &p.createdAt
	}
	return domain.User{
		Id:   deterministicULID(p.index),
		Name: lo.ToPtr(p.name), Number: lo.ToPtr(p.number),
		Email:  p.email,
		Gender: p.gender, BirthDate: p.birthDate,
		Meta: p.meta, Role: p.role,
		WalletAddress: address, EncryptedWalletPrivateKey: encryptedKey,
		CreatedAt: p.createdAt, UpdatedAt: updatedAt, DeletedAt: p.deletedAt,
	}
}

func seedMnemonic(cfg *config.Config) string {
	if cfg.HardhatMnemonic != nil && *cfg.HardhatMnemonic != "" {
		return *cfg.HardhatMnemonic
	}
	return "test test test test test test test test test test test junk"
}

// seedLeafUnits returns units with a ParentId that are not themselves parents
// (i.e., the study programs at the leaves of the tree).
func seedLeafUnits(units []domain.UserUnit) []domain.UserUnit {
	parentSet := make(map[string]bool, len(units))
	for _, u := range units {
		if u.ParentId != nil {
			parentSet[*u.ParentId] = true
		}
	}
	leaves := make([]domain.UserUnit, 0, len(units))
	for _, u := range units {
		if u.ParentId != nil && !parentSet[u.Id] {
			leaves = append(leaves, u)
		}
	}
	return leaves
}

// seedAssignUnitIDs assigns study programs round-robin to Holder users.
// Non-Holder users keep a nil UnitID.
func seedAssignUnitIDs(users []domain.User, programs []domain.UserUnit) {
	if len(programs) == 0 {
		return
	}
	idx := 0
	for i := range users {
		if users[i].Role == domain.RoleHolder {
			users[i].UnitID = lo.ToPtr(programs[idx%len(programs)].Id)
			idx++
		}
	}
}

func seedGenerateNIP(dob time.Time, gender *domain.Gender, seq *int) string {
	recruit := dob.AddDate(21, 0, 0)
	genderDigit := '0'
	if gender != nil {
		switch *gender {
		case domain.GenderMale:
			genderDigit = '1'
		case domain.GenderFemale:
			genderDigit = '2'
		}
	}
	*seq++
	return fmt.Sprintf("%s%04d%02d%c%03d",
		dob.Format("20060102"), recruit.Year(), recruit.Month(), genderDigit, *seq)
}

func seedGenerateNIM(seq *int) string {
	*seq++
	return fmt.Sprintf("2209%04d", *seq)
}

func seedRandomAlphaKey(rng *rand.Rand) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

func seedMustParseDate(date string) *time.Time {
	t, _ := time.Parse(time.DateOnly, date)
	return &t
}

func seedGenderPtr(g domain.Gender) *domain.Gender { return &g }

func hashToSeed(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

var seedIndoFirstNames = []string{
	"Ahmad", "Budi", "Citra", "Dewi", "Eko", "Fitri", "Gunawan", "Hadi", "Indah", "Joko",
	"Kartika", "Lestari", "Mega", "Nur", "Putri", "Rizky", "Sari", "Tono", "Wati", "Yanto",
}
var seedIndoLastNames = []string{
	"Santoso", "Wijaya", "Pratama", "Kusuma", "Hidayat", "Saputra", "Nugroho", "Permana",
	"Mahendra", "Setiawan", "Purnama", "Gunawan", "Hartono", "Wibowo", "Kurniawan",
}

func seedRandomIndonesianName(rng *rand.Rand) string {
	return seedIndoFirstNames[rng.Intn(len(seedIndoFirstNames))] + " " + seedIndoLastNames[rng.Intn(len(seedIndoLastNames))]
}

func seedNameToEmail(name string, idx int) string {
	parts := seedSplitName(name)
	email := ""
	for i, p := range parts {
		if i > 0 {
			email += "."
		}
		email += seedToLower(p)
	}
	return fmt.Sprintf("%s.%d@gmail.com", email, idx)
}

func seedSplitName(name string) []string {
	var parts []string
	current := ""
	for _, r := range name {
		if r == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func seedToLower(s string) string {
	b := make([]byte, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			b[i] = byte(r + 32)
		} else {
			b[i] = byte(r)
		}
	}
	return string(b)
}

func seedRandomBirthDate(rng *rand.Rand) time.Time {
	min := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2005, 12, 31, 0, 0, 0, 0, time.UTC)
	delta := max.Unix() - min.Unix()
	return min.Add(time.Duration(rng.Int63n(delta)) * time.Second)
}

var seedGenders = []domain.Gender{domain.GenderMale, domain.GenderFemale}

func seedRandomGender(rng *rand.Rand) domain.Gender {
	return seedGenders[rng.Intn(len(seedGenders))]
}

var seedUserRoles = []domain.Role{
	domain.RoleHolder, domain.RoleHolder, domain.RoleHolder,
	domain.RoleIssuer, domain.RoleIssuer,
}

func seedRandomUserRole(rng *rand.Rand) domain.Role {
	return seedUserRoles[rng.Intn(len(seedUserRoles))]
}
