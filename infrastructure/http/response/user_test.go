package response

import (
	"CredChain_Golang/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainUser_AllFieldsSet(t *testing.T) {
	name := "Alice"
	number := "12345"
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	meta := map[string]any{"key": "value"}

	u := domain.User{
		Id:                        "user-1",
		Name:                      &name,
		Number:                    &number,
		Email:                     "alice@example.com",
		BirthDate:                 &bd,
		Meta:                      meta,
		Role:                      domain.RoleHolder,
		WalletAddress:             "0xabc",
		EncryptedWalletPrivateKey: "secret",
		CreatedAt:                 time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:                 &updatedAt,
	}

	got := FromDomainUser(u)

	assert.Equal(t, "user-1", got.ID)
	assert.Equal(t, &name, got.Name)
	assert.Equal(t, &number, got.Number)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.Equal(t, &bd, got.BirthDate)
	assert.Equal(t, meta, got.Meta)
	assert.Equal(t, domain.RoleHolder, got.Role)
	assert.Equal(t, "0xabc", got.WalletAddress)
	assert.Equal(t, u.CreatedAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestFromDomainUser_NilOptionalFields(t *testing.T) {
	u := domain.User{
		Id:                        "user-2",
		Name:                      nil,
		Number:                    nil,
		Email:                     "bob@example.com",
		BirthDate:                 nil,
		Meta:                      nil,
		Role:                      domain.RoleIssuer,
		WalletAddress:             "0xdef",
		EncryptedWalletPrivateKey: "",
		CreatedAt:                 time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:                 nil,
	}

	got := FromDomainUser(u)

	assert.Equal(t, "user-2", got.ID)
	assert.Nil(t, got.Name)
	assert.Nil(t, got.Number)
	assert.Equal(t, "bob@example.com", got.Email)
	assert.Nil(t, got.BirthDate)
	assert.Nil(t, got.Meta)
	assert.Equal(t, domain.RoleIssuer, got.Role)
	assert.Equal(t, "0xdef", got.WalletAddress)
	assert.Nil(t, got.UpdatedAt)
}

func TestToDomain_AllFieldsSet(t *testing.T) {
	name := "Alice"
	number := "12345"
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	meta := map[string]any{"key": "value"}

	r := User{
		ID:            "user-1",
		Name:          &name,
		Number:        &number,
		Email:         "alice@example.com",
		BirthDate:     &bd,
		Meta:          meta,
		Role:          domain.RoleHolder,
		WalletAddress: "0xabc",
		CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     &updatedAt,
	}

	got := r.ToDomain()

	assert.Equal(t, "user-1", got.Id)
	assert.Equal(t, &name, got.Name)
	assert.Equal(t, &number, got.Number)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.Equal(t, &bd, got.BirthDate)
	assert.Equal(t, meta, got.Meta)
	assert.Equal(t, domain.RoleHolder, got.Role)
	assert.Equal(t, "0xabc", got.WalletAddress)
	assert.Equal(t, r.CreatedAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestToDomain_NilOptionalFields(t *testing.T) {
	r := User{
		ID:            "user-2",
		Name:          nil,
		Number:        nil,
		Email:         "bob@example.com",
		BirthDate:     nil,
		Meta:          nil,
		Role:          domain.RoleIssuer,
		WalletAddress: "0xdef",
		CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     nil,
	}

	got := r.ToDomain()

	assert.Equal(t, "user-2", got.Id)
	assert.Nil(t, got.Name)
	assert.Nil(t, got.Number)
	assert.Equal(t, "bob@example.com", got.Email)
	assert.Nil(t, got.BirthDate)
	assert.Nil(t, got.Meta)
	assert.Equal(t, domain.RoleIssuer, got.Role)
	assert.Equal(t, "0xdef", got.WalletAddress)
	assert.Nil(t, got.UpdatedAt)
}

func TestFromDomainUser_PreservesDeletedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := domain.User{Id: "u1", Email: "x@x.com", DeletedAt: &now}
	r := FromDomainUser(d)
	assert.NotNil(t, r.DeletedAt)
	assert.Equal(t, now, *r.DeletedAt)
}

func TestFromDomainUser_NilDeletedAt(t *testing.T) {
	d := domain.User{Id: "u1", Email: "x@x.com"}
	r := FromDomainUser(d)
	assert.Nil(t, r.DeletedAt)
}

func TestUser_ToDomain_PreservesDeletedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := User{ID: "u1", Email: "x@x.com", DeletedAt: &now}
	d := r.ToDomain()
	assert.NotNil(t, d.DeletedAt)
	assert.Equal(t, now, *d.DeletedAt)
}
