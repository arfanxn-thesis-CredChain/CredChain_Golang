package notification

import (
	"context"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"
)

func notificationScope(ctx context.Context, userID string, authority chain.AuthorityService) domain.NotificationScope {
	scope := domain.NotificationScope{UserID: userID}
	user, err := httpContext.GetUser(ctx)
	if err == nil && user.Id == userID && user.DeletedAt == nil && user.Role.Rank() >= domain.RoleIssuer.Rank() && authority != nil {
		scope.CanReview = authority.HasRoleOrAbove(ctx, user.WalletAddress, domain.RoleIssuer)
	}
	return scope
}
