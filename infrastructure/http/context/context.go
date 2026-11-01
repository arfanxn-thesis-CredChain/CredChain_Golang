package context

import (
	"CredChain_Golang/domain"
	"context"
	"fmt"
)

// contextKey is a private type for context keys (type-safe)
type contextKey string

const (
	// UserKey is the key for user entity in context
	UserKey contextKey = "user"
)

// WithUser stores user in context.
func WithUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

// GetUser retrieves user from context
func GetUser(ctx context.Context) (*domain.User, error) {
	user, ok := ctx.Value(UserKey).(*domain.User)
	if !ok || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// MustGetUser retrieves user or panics if not found.
// This should only be used in handlers protected by AuthMiddleware.
func MustGetUser(ctx context.Context) *domain.User {
	user, err := GetUser(ctx)
	if err != nil {
		panic("user missing in authenticated context - AuthMiddleware not configured?")
	}
	return user
}
