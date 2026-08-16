package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// Role defines the allowed role types mapped to the Postgres ENUM
type Role string

const (
	RoleNone       Role = "none"
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleIssuer     Role = "issuer"
	RoleHolder     Role = "holder"
)

// Rank returns the numeric hierarchy level of the role (higher is more privileged)
func (r Role) Rank() int {
	switch r {
	case RoleSuperAdmin:
		return 4
	case RoleAdmin:
		return 3
	case RoleIssuer:
		return 2
	case RoleHolder:
		return 1
	case RoleNone:
		return 0
	default:
		return 0
	}
}

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}

// ToUint8 converts domain.Role to Solidity Role enum value
func (r Role) ToUint8() uint8 {
	switch r {
	case RoleHolder:
		return 1
	case RoleIssuer:
		return 2
	case RoleAdmin:
		return 3
	case RoleSuperAdmin:
		return 4
	case RoleNone:
		return 0
	default:
		return 0
	}
}

// RoleFromUint8 converts Solidity Role enum value to domain.Role
func RoleFromUint8(v uint8) Role {
	switch v {
	case 1:
		return RoleHolder
	case 2:
		return RoleIssuer
	case 3:
		return RoleAdmin
	case 4:
		return RoleSuperAdmin
	case 0:
		return RoleNone
	default:
		return RoleNone
	}
}

// Gender defines the allowed gender values mapped to the Postgres ENUM
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// String returns the string representation of the gender
func (g Gender) String() string {
	return string(g)
}

// UserRoleUpdate defines a single target role update for a user
type UserRoleUpdate struct {
	UserID string
	Role   Role
}

// User represents a row in the users table
type User struct {
	Id                        string         `json:"id"`
	Name                      *string        `json:"name"`
	Number                    *string        `json:"number"`
	UnitID                    *string        `json:"unit_id"`
	JoinedYear                *int           `json:"joined_year"`
	Email                     string         `json:"email"`
	BirthDate                 *time.Time     `json:"birth_date"`
	Gender                    *Gender        `json:"gender"`
	Meta                      map[string]any `json:"meta"`
	Role                      Role           `json:"role"`
	WalletAddress             string         `json:"wallet_address"`
	EncryptedWalletPrivateKey string         `json:"-"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 *time.Time     `json:"updated_at"`
	DeletedAt                 *time.Time     `json:"deleted_at"`
}

// UserRepository defines the database contract for the User Domain
type UserRepository interface {
	// Query-based retrieval
	Get(ctx context.Context, query *domainQuery.Query) ([]User, int, error)

	// Single item lookups
	Find(ctx context.Context, id string) (*User, error)
	FindByEmails(ctx context.Context, emails ...string) ([]User, error)

	// Role-based lookups
	FindByRole(ctx context.Context, role Role) ([]User, error)

	// Multiple item lookups
	FindByIds(ctx context.Context, ids ...string) ([]User, error)

	// CRUD operations
	Update(ctx context.Context, users ...User) ([]User, error)
	Store(ctx context.Context, users ...User) ([]User, error)
	Delete(ctx context.Context, ids ...string) (int64, error)

	// Restore unsets deleted_at for trashed users (batch operation).
	Restore(ctx context.Context, ids ...string) (int64, error)

	// Specialized operations
	UpdateRole(ctx context.Context, users ...User) ([]User, int64, error)

	// CountByUnitIds counts users (including trashed) referencing any of the
	// given unit ids. Pure read primitive for the step-3 deletion guard.
	CountByUnitIds(ctx context.Context, unitIds ...string) (int64, error)
}

// UserTokenType defines the type of user token
type UserTokenType string

const (
	UserTokenTypeRefresh UserTokenType = "refresh"
)

// UserToken represents a row in the user_tokens table
type UserToken struct {
	Id         string
	UserId     string
	Type       UserTokenType
	Token      string
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  *time.Time
}

// UserTokenRepository defines the database contract for user tokens
type UserTokenRepository interface {
	// Store creates new user tokens
	Store(ctx context.Context, tokens ...UserToken) ([]UserToken, error)

	// Find retrieves a token by ID
	Find(ctx context.Context, id string) (*UserToken, error)

	// FindByToken retrieves a token by its token string
	FindByToken(ctx context.Context, token string) (*UserToken, error)

	// FindByUserId retrieves all tokens for a user
	FindByUserId(ctx context.Context, userId string) ([]UserToken, error)

	// Revoke marks tokens as revoked, returns number of revoked tokens
	Revoke(ctx context.Context, tokenIDs ...string) (int64, error)

	// RevokeByUserIdAndType revokes all tokens of a specific type for a user, returns rows affected
	RevokeByUserIdAndType(ctx context.Context, userId string, tokenType UserTokenType) (int64, error)

	// Update updates a single user token, returns the updated entity
	Update(ctx context.Context, token UserToken) (*UserToken, error)
}
