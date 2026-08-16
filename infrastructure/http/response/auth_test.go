package response

import (
	"CredChain_Golang/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuth(t *testing.T) {
	name := "Alice"
	now := time.Now()

	user := User{
		ID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:          &name,
		Number:        nil,
		Email:         "alice@example.com",
		BirthDate:     nil,
		Meta:          nil,
		Role:          domain.RoleHolder,
		WalletAddress: "0xabc123",
		CreatedAt:     now,
		UpdatedAt:     nil,
	}

	got := NewAuth(user, "access_tok_123", "refresh_tok_456", 900, 86400)

	assert.Equal(t, "access_tok_123", got.AccessToken)
	assert.Equal(t, "refresh_tok_456", got.RefreshToken)
	assert.Equal(t, 900, got.AccessTokenExpiresIn)
	assert.Equal(t, 86400, got.RefreshTokenExpiresIn)
	assert.Equal(t, "Bearer", got.TokenType)
	assert.Equal(t, user, got.User)
}
