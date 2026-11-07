package notification

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNotificationScope_Unauthenticated(t *testing.T) {
	authMock := new(mocks.MockAuthorityService)
	scope := notificationScope(context.Background(), "user-123", authMock)

	assert.Equal(t, "user-123", scope.UserID)
	assert.False(t, scope.CanReview)
	authMock.AssertNotCalled(t, "HasRoleOrAbove", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationScope_NilAuthority(t *testing.T) {
	u := &domain.User{Id: "user-123", Role: domain.RoleIssuer, WalletAddress: "0x123"}
	ctx := httpContext.WithUser(context.Background(), u)

	scope := notificationScope(ctx, "user-123", nil)
	assert.Equal(t, "user-123", scope.UserID)
	assert.False(t, scope.CanReview)
}

func TestNotificationScope_RecipientMismatch(t *testing.T) {
	authMock := new(mocks.MockAuthorityService)
	u := &domain.User{Id: "user-1", Role: domain.RoleIssuer, WalletAddress: "0x123"}
	ctx := httpContext.WithUser(context.Background(), u)

	scope := notificationScope(ctx, "user-2", authMock)
	assert.Equal(t, "user-2", scope.UserID)
	assert.False(t, scope.CanReview)
	authMock.AssertNotCalled(t, "HasRoleOrAbove", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationScope_SoftDeletedUser(t *testing.T) {
	authMock := new(mocks.MockAuthorityService)
	now := time.Now()
	u := &domain.User{Id: "user-123", Role: domain.RoleIssuer, WalletAddress: "0x123", DeletedAt: &now}
	ctx := httpContext.WithUser(context.Background(), u)

	scope := notificationScope(ctx, "user-123", authMock)
	assert.Equal(t, "user-123", scope.UserID)
	assert.False(t, scope.CanReview)
	authMock.AssertNotCalled(t, "HasRoleOrAbove", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationScope_HolderRole(t *testing.T) {
	authMock := new(mocks.MockAuthorityService)
	u := &domain.User{Id: "user-123", Role: domain.RoleHolder, WalletAddress: "0x123"}
	ctx := httpContext.WithUser(context.Background(), u)

	scope := notificationScope(ctx, "user-123", authMock)
	assert.Equal(t, "user-123", scope.UserID)
	assert.False(t, scope.CanReview)
	authMock.AssertNotCalled(t, "HasRoleOrAbove", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationScope_ReviewerRoles_ChainAllowed(t *testing.T) {
	roles := []domain.Role{domain.RoleIssuer, domain.RoleAdmin, domain.RoleSuperAdmin}
	for _, r := range roles {
		t.Run(string(r), func(t *testing.T) {
			authMock := new(mocks.MockAuthorityService)
			u := &domain.User{Id: "user-123", Role: r, WalletAddress: "0x123"}
			ctx := httpContext.WithUser(context.Background(), u)

			authMock.On("HasRoleOrAbove", ctx, "0x123", domain.RoleIssuer).Return(true).Once()

			scope := notificationScope(ctx, "user-123", authMock)
			assert.Equal(t, "user-123", scope.UserID)
			assert.True(t, scope.CanReview)
			authMock.AssertExpectations(t)
		})
	}
}

func TestNotificationScope_ReviewerRole_ChainDenied(t *testing.T) {
	authMock := new(mocks.MockAuthorityService)
	u := &domain.User{Id: "user-123", Role: domain.RoleIssuer, WalletAddress: "0x123"}
	ctx := httpContext.WithUser(context.Background(), u)

	authMock.On("HasRoleOrAbove", ctx, "0x123", domain.RoleIssuer).Return(false).Once()

	scope := notificationScope(ctx, "user-123", authMock)
	assert.Equal(t, "user-123", scope.UserID)
	assert.False(t, scope.CanReview)
	authMock.AssertExpectations(t)
}
