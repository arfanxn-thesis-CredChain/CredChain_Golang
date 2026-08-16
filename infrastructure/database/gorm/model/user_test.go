package model

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUser_RoundTrip_AllFields(t *testing.T) {
	name := "Alice"
	number := "12345"
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	upd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	meta := map[string]any{"x": "y"}

	d := domain.User{
		Id: "u1", Name: &name, Number: &number,
		Email: "a@x.com", BirthDate: &bd, Meta: meta,
		Role: domain.RoleAdmin, WalletAddress: "0xaa",
		EncryptedWalletPrivateKey: "enc", CreatedAt: time.Now(), UpdatedAt: &upd,
	}
	m := FromDomainUser(d)
	roundtrip := m.ToDomain()

	assert.Equal(t, d.Id, roundtrip.Id)
	assert.Equal(t, &name, roundtrip.Name)
	assert.Equal(t, &number, roundtrip.Number)
	assert.Equal(t, "a@x.com", roundtrip.Email)
	assert.Equal(t, &bd, roundtrip.BirthDate)
	assert.Equal(t, meta, roundtrip.Meta)
	assert.Equal(t, domain.RoleAdmin, roundtrip.Role)
	assert.Equal(t, "0xaa", roundtrip.WalletAddress)
	assert.Equal(t, "enc", roundtrip.EncryptedWalletPrivateKey)
	assert.Equal(t, &upd, roundtrip.UpdatedAt)
}

func TestUser_RoundTrip_NilOptionals(t *testing.T) {
	d := domain.User{
		Id: "u2", Email: "b@x.com",
		Role: domain.RoleHolder, WalletAddress: "0xbb",
		CreatedAt: time.Now(),
	}
	m2 := FromDomainUser(d)
	roundtrip := m2.ToDomain()
	assert.Nil(t, roundtrip.Name)
	assert.Nil(t, roundtrip.Number)
	assert.Nil(t, roundtrip.BirthDate)
	assert.Nil(t, roundtrip.Meta)
	assert.Nil(t, roundtrip.UpdatedAt)
}

func TestUser_RoleStringConversion(t *testing.T) {
	d := domain.User{Role: domain.RoleSuperAdmin, Email: "s@x", WalletAddress: "0x", CreatedAt: time.Now()}
	m := FromDomainUser(d)
	assert.Equal(t, "super_admin", m.Role)
	back := m.ToDomain()
	assert.Equal(t, domain.RoleSuperAdmin, back.Role)
}

func TestUser_ToDomain_PreservesDeletedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := User{DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}
	d := m.ToDomain()
	assert.NotNil(t, d.DeletedAt)
	assert.Equal(t, now, *d.DeletedAt)
}

func TestUser_ToDomain_NilDeletedAt_WhenInvalid(t *testing.T) {
	m := User{DeletedAt: gorm.DeletedAt{Valid: false}}
	d := m.ToDomain()
	assert.Nil(t, d.DeletedAt)
}

func TestUser_FromDomain_PreservesDeletedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := domain.User{DeletedAt: &now}
	m := FromDomainUser(d)
	assert.True(t, m.DeletedAt.Valid)
	assert.Equal(t, now, m.DeletedAt.Time)
}

func TestUser_FromDomain_NilDeletedAt(t *testing.T) {
	d := domain.User{}
	m := FromDomainUser(d)
	assert.False(t, m.DeletedAt.Valid)
}

func TestUserModel_UnitID_JoinedYear_RoundTrip(t *testing.T) {
	unitID := "unit-abc"
	year := 2023
	d := domain.User{
		Id: "u1", Email: "a@x.com", Role: domain.RoleHolder,
		WalletAddress: "0xaa", UnitID: &unitID, JoinedYear: &year,
	}
	m := FromDomainUser(d)
	assert.Equal(t, unitID, *m.UnitID)
	assert.Equal(t, year, *m.JoinedYear)

	roundtrip := m.ToDomain()
	assert.Equal(t, unitID, *roundtrip.UnitID)
	assert.Equal(t, year, *roundtrip.JoinedYear)
}
