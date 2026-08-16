package user

import (
	"strings"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestUserStoreInput_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserStoreInput
		shouldErr bool
	}{
		{name: "Valid Request", req: UserStoreInput{Name: "John Doe", Email: "john@example.com", Role: domain.RoleHolder}, shouldErr: false},
		{name: "Valid Gender Male", req: UserStoreInput{Name: "John", Email: "john1@example.com", Role: domain.RoleHolder, Gender: lo.ToPtr("male")}, shouldErr: false},
		{name: "Valid Gender Female", req: UserStoreInput{Name: "Jane", Email: "jane@example.com", Role: domain.RoleHolder, Gender: lo.ToPtr("female")}, shouldErr: false},
		{name: "Invalid Gender Other", req: UserStoreInput{Name: "Alex", Email: "alex@example.com", Role: domain.RoleHolder, Gender: lo.ToPtr("other")}, shouldErr: true},
		{name: "Invalid Gender", req: UserStoreInput{Name: "Pat", Email: "pat@example.com", Role: domain.RoleHolder, Gender: lo.ToPtr("unknown")}, shouldErr: true},
		{name: "Nil Gender is valid", req: UserStoreInput{Name: "Sam", Email: "sam@example.com", Role: domain.RoleHolder, Gender: nil}, shouldErr: false},
		{name: "Missing Name", req: UserStoreInput{Email: "john@example.com", Role: domain.RoleHolder}, shouldErr: true},
		{name: "Invalid Email", req: UserStoreInput{Name: "John Doe", Email: "john-example.com", Role: domain.RoleHolder}, shouldErr: true},
		{name: "Invalid Role", req: UserStoreInput{Name: "John Doe", Email: "john@example.com", Role: domain.Role("invalid_role")}, shouldErr: true},
		{
			name: "Valid with all optional fields",
			req: UserStoreInput{
				Name:      "Jane Doe",
				Email:     "jane@example.com",
				Role:      domain.RoleIssuer,
				Number:    lo.ToPtr("12345"),
				BirthDate: lo.ToPtr("1990-01-01"),
				Meta:      map[string]any{"key": "value"},
			},
			shouldErr: false,
		},
		{
			name: "Invalid BirthDate format",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: lo.ToPtr("1990/01/01"),
			},
			shouldErr: true,
		},
		{
			name: "Invalid BirthDate not a date",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: lo.ToPtr("not-a-date"),
			},
			shouldErr: true,
		},
		{
			name: "Empty BirthDate string is valid",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: lo.ToPtr(""),
			},
			shouldErr: false,
		},
		{name: "Valid UnitID and JoinedYear", req: UserStoreInput{Name: "Alex", Email: "alex@example.com", Role: domain.RoleHolder, UnitID: lo.ToPtr("unit-1"), JoinedYear: lo.ToPtr(2023)}, shouldErr: false},
		{name: "UnitID Too Long", req: UserStoreInput{Name: "Alex", Email: "alex@example.com", Role: domain.RoleHolder, UnitID: lo.ToPtr(strings.Repeat("a", 27))}, shouldErr: true},
		{name: "Gender Other Still Rejected", req: UserStoreInput{Name: "Alex", Email: "alex@example.com", Role: domain.RoleHolder, Gender: lo.ToPtr("other")}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserStoreInput_ToDomain(t *testing.T) {
	t.Run("All fields set", func(t *testing.T) {
		number := "12345"
		birthDate := "1990-01-01"
		meta := map[string]any{"key": "value"}
		input := UserStoreInput{
			Name:      "John Doe",
			Email:     "john@example.com",
			Role:      domain.RoleHolder,
			Number:    &number,
			BirthDate: &birthDate,
			Meta:      meta,
		}

		got := input.ToDomain()

		assert.NotNil(t, got.Name)
		assert.Equal(t, "John Doe", *got.Name)
		assert.Equal(t, "john@example.com", got.Email)
		assert.Equal(t, domain.RoleHolder, got.Role)
		assert.Equal(t, &number, got.Number)
		assert.NotNil(t, got.BirthDate)
		assert.Equal(t, time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), *got.BirthDate)
		assert.Equal(t, meta, got.Meta)
	})

	t.Run("Only required fields", func(t *testing.T) {
		input := UserStoreInput{
			Name:  "John Doe",
			Email: "john@example.com",
			Role:  domain.RoleHolder,
		}

		got := input.ToDomain()

		assert.NotNil(t, got.Name)
		assert.Equal(t, "John Doe", *got.Name)
		assert.Equal(t, "john@example.com", got.Email)
		assert.Equal(t, domain.RoleHolder, got.Role)
		assert.Nil(t, got.Number)
		assert.Nil(t, got.BirthDate)
		assert.Nil(t, got.Meta)
	})

	t.Run("Empty BirthDate string yields nil domain BirthDate", func(t *testing.T) {
		empty := ""
		input := UserStoreInput{
			Name:      "John Doe",
			Email:     "john@example.com",
			Role:      domain.RoleHolder,
			BirthDate: &empty,
		}

		got := input.ToDomain()

		assert.Nil(t, got.BirthDate)
	})

	t.Run("Gender set yields domain.Gender pointer", func(t *testing.T) {
		input := UserStoreInput{
			Name:   "Test",
			Email:  "test@example.com",
			Role:   domain.RoleHolder,
			Gender: lo.ToPtr("female"),
		}
		got := input.ToDomain()
		assert.NotNil(t, got.Gender)
		assert.Equal(t, domain.GenderFemale, *got.Gender)
	})

	t.Run("Empty gender string yields nil domain.Gender", func(t *testing.T) {
		empty := ""
		input := UserStoreInput{
			Name:   "Test",
			Email:  "test@example.com",
			Role:   domain.RoleHolder,
			Gender: &empty,
		}
		got := input.ToDomain()
		assert.Nil(t, got.Gender)
	})
}

func TestUserUpdateSelfEmailRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserUpdateSelfEmailRequest
		shouldErr bool
	}{
		{name: "Valid Email with IdToken", req: UserUpdateSelfEmailRequest{Email: "alice@gmail.com", IdToken: "token"}, shouldErr: false},
		{name: "Invalid Email", req: UserUpdateSelfEmailRequest{Email: "alicetest.com", IdToken: "token"}, shouldErr: true},
		{name: "Empty Email", req: UserUpdateSelfEmailRequest{Email: "", IdToken: "token"}, shouldErr: true},
		{name: "Missing IdToken", req: UserUpdateSelfEmailRequest{Email: "alice@gmail.com"}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserRoleUpdateInput_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserRoleUpdateInput
		shouldErr bool
	}{
		{name: "Valid Role Update", req: UserRoleUpdateInput{UserID: "01HXXXXX...", Role: domain.RoleIssuer}, shouldErr: false},
		{name: "Missing UserID", req: UserRoleUpdateInput{Role: domain.RoleIssuer}, shouldErr: true},
		{name: "Invalid Role", req: UserRoleUpdateInput{UserID: "01HXXXXX...", Role: domain.Role("fake_role")}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserStoreRequest_Validate_EmptyUsers(t *testing.T) {
	r := UserStoreRequest{Users: nil}
	assert.Error(t, r.Validate())
}

func TestUserStoreRequest_Validate_InvalidNested(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Bob", Email: "not-an-email", Role: domain.RoleHolder},
	}}
	assert.Error(t, r.Validate())
}

func TestUserStoreRequest_Validate_Valid(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Bob", Email: "bob@x.com", Role: domain.RoleHolder},
	}}
	assert.NoError(t, r.Validate())
}

func TestUserStoreRequest_ToDomain_NilSlice(t *testing.T) {
	r := UserStoreRequest{Users: nil}
	got := r.ToDomain()
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestUserStoreRequest_ToDomain_MultiUser(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Alice", Email: "a@x.com", Role: domain.RoleHolder},
		{Name: "Bob", Email: "b@x.com", Role: domain.RoleIssuer},
	}}
	got := r.ToDomain()
	assert.Len(t, got, 2)
	assert.Equal(t, "a@x.com", got[0].Email)
	assert.Equal(t, "b@x.com", got[1].Email)
}

func TestUserUpdateRoleRequest_Validate_Empty(t *testing.T) {
	r := UserUpdateRoleRequest{UserRoles: nil}
	assert.Error(t, r.Validate())
}

func TestUserUpdateRoleRequest_Validate_Valid(t *testing.T) {
	r := UserUpdateRoleRequest{UserRoles: []UserRoleUpdateInput{
		{UserID: "u1", Role: domain.RoleIssuer},
	}}
	assert.NoError(t, r.Validate())
}

func TestUserDeleteRequest_Validate_Empty(t *testing.T) {
	r := UserDeleteRequest{Ids: nil}
	assert.Error(t, r.Validate())
}

func TestUserDeleteRequest_Validate_Valid(t *testing.T) {
	r := UserDeleteRequest{Ids: []string{"u1", "u2"}}
	assert.NoError(t, r.Validate())
}

func TestUserStoreInput_NameOverMax(t *testing.T) {
	over := strings.Repeat("a", 257)
	in := UserStoreInput{Name: over, Email: "ok@x.com", Role: domain.RoleHolder}
	assert.Error(t, in.Validate())
}

func TestUserStoreInput_NameAtMax(t *testing.T) {
	atMax := strings.Repeat("a", 256)
	in := UserStoreInput{Name: atMax, Email: "ok@x.com", Role: domain.RoleHolder}
	assert.NoError(t, in.Validate())
}

func TestUserStoreInput_EmailOverMax(t *testing.T) {
	longLocal := strings.Repeat("a", 250)
	in := UserStoreInput{Name: "n", Email: longLocal + "@x.com", Role: domain.RoleHolder}
	assert.Error(t, in.Validate())
}

func TestUserUpdateSelfEmailRequest_EmailOverMax(t *testing.T) {
	longLocal := strings.Repeat("a", 250)
	r := UserUpdateSelfEmailRequest{Email: longLocal + "@x.com"}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfEmailRequest_Validate_RequiresIdToken(t *testing.T) {
	r := UserUpdateSelfEmailRequest{Email: "ok@x.com", IdToken: ""}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfEmailRequest_Validate_ValidWithIdToken(t *testing.T) {
	r := UserUpdateSelfEmailRequest{Email: "ok@x.com", IdToken: "some-token"}
	assert.NoError(t, r.Validate())
}

func TestUserUpdateInput_Validate_RequiresId(t *testing.T) {
	in := UserUpdateInput{}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_Validate_Valid(t *testing.T) {
	n := "Alice"
	in := UserUpdateInput{Id: "u1", Name: &n}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateRequest_Validate_Empty(t *testing.T) {
	r := UserUpdateRequest{Users: nil}
	assert.Error(t, r.Validate())
}

func TestUserUpdateRequest_ToDomain(t *testing.T) {
	n := "Alice"
	r := UserUpdateRequest{Users: []UserUpdateInput{{Id: "u1", Name: &n}}}
	users := r.ToDomain()
	assert.Len(t, users, 1)
	assert.Equal(t, "u1", users[0].Id)
}

func TestUserUpdateInput_Validate_EmailValid(t *testing.T) {
	email := "ok@x.com"
	in := UserUpdateInput{Id: "u1", Email: &email}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateInput_Validate_EmailInvalid(t *testing.T) {
	email := "not-an-email"
	in := UserUpdateInput{Id: "u1", Email: &email}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_Validate_RoleValid(t *testing.T) {
	role := domain.RoleIssuer
	in := UserUpdateInput{Id: "u1", Role: &role}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateInput_Validate_RoleSuperAdminRejected(t *testing.T) {
	role := domain.RoleSuperAdmin
	in := UserUpdateInput{Id: "u1", Role: &role}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_ToDomain_PropagatesEmailAndRole(t *testing.T) {
	email := "new@x.com"
	role := domain.RoleIssuer
	in := UserUpdateInput{Id: "u1", Email: &email, Role: &role}
	u := in.ToDomain()
	assert.Equal(t, "new@x.com", u.Email)
	assert.Equal(t, domain.RoleIssuer, u.Role)
}

func TestUserTransferSuperAdminRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UserTransferSuperAdminRequest
		wantErr bool
	}{
		{"valid ULID", UserTransferSuperAdminRequest{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, false},
		{"valid UUID (accepted as opaque string)", UserTransferSuperAdminRequest{Id: "123e4567-e89b-12d3-a456-426614174000"}, false},
		{"any non-empty string", UserTransferSuperAdminRequest{Id: "not-a-uuid"}, false},
		{"plain number string", UserTransferSuperAdminRequest{Id: "12345"}, false},
		{"empty id", UserTransferSuperAdminRequest{Id: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserRestoreRequest_Validate_EmptyIDs(t *testing.T) {
	req := UserRestoreRequest{IDs: []string{}}
	err := req.Validate()
	assert.Error(t, err)
}

func TestUserRestoreRequest_Validate_Valid(t *testing.T) {
	req := UserRestoreRequest{IDs: []string{"01J123456789012345678901"}}
	err := req.Validate()
	assert.NoError(t, err)
}
