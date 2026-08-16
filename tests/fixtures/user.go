package fixtures

import (
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
)

type UserOption func(*domain.User)

func WithID(id string) UserOption           { return func(u *domain.User) { u.Id = id } }
func WithEmail(e string) UserOption         { return func(u *domain.User) { u.Email = e } }
func WithRole(r domain.Role) UserOption     { return func(u *domain.User) { u.Role = r } }
func WithName(n string) UserOption          { return func(u *domain.User) { u.Name = &n } }
func WithMeta(m map[string]any) UserOption  { return func(u *domain.User) { u.Meta = m } }
func WithWalletAddress(a string) UserOption { return func(u *domain.User) { u.WalletAddress = a } }
func WithNumber(n string) UserOption        { return func(u *domain.User) { u.Number = &n } }
func WithUnitID(id string) UserOption       { return func(u *domain.User) { u.UnitID = &id } }
func WithJoinedYear(y int) UserOption       { return func(u *domain.User) { u.JoinedYear = &y } }
func WithEncryptedKey(k string) UserOption {
	return func(u *domain.User) { u.EncryptedWalletPrivateKey = k }
}

// NewDomainUser returns a domain.User with sensible defaults: random ULID id,
// RoleHolder, valid 0x-prefixed wallet address (zero address), current CreatedAt.
func NewDomainUser(opts ...UserOption) domain.User {
	u := domain.User{
		Id:            ulid.Make().String(),
		Email:         "test-" + ulid.Make().String() + "@example.com",
		Role:          domain.RoleHolder,
		WalletAddress: "0x0000000000000000000000000000000000000000",
		CreatedAt:     time.Now(),
	}
	for _, opt := range opts {
		opt(&u)
	}
	return u
}

// NewModelUser maps a NewDomainUser through model.FromDomainUser.
func NewModelUser(opts ...UserOption) model.User {
	return model.FromDomainUser(NewDomainUser(opts...))
}
