package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

type OverviewCredentialCounts struct {
	Total   int
	Active  int
	Revoked int
	Pending int
	Failed  int
}

type OverviewUserCounts struct {
	Total      int
	Holder     int
	Issuer     int
	Admin      int
	SuperAdmin int
	Active     int
	Trashed    int
}

// CredentialRecapRow contains additive counts for one exact unit/year combination.
type CredentialRecapRow struct {
	UnitID                          *string `json:"unit_id"`
	JoinedYear                      *int    `json:"joined_year"`
	TotalHolders                    int     `json:"total_holders"`
	HoldersWithCredentials          int     `json:"holders_with_credentials"`
	HoldersWithoutCredentials       int     `json:"holders_without_credentials"`
	HoldersWithActiveCredentials    int     `json:"holders_with_active_credentials"`
	HoldersWithoutActiveCredentials int     `json:"holders_without_active_credentials"`
	TotalCredentials                int     `json:"total_credentials"`
	ActiveCredentials               int     `json:"active_credentials"`
	PendingCredentials              int     `json:"pending_credentials"`
	RejectedCredentials             int     `json:"rejected_credentials"`
	ExpiredCredentials              int     `json:"expired_credentials"`
	RevokedCredentials              int     `json:"revoked_credentials"`
}

type OverviewRepository interface {
	CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*OverviewCredentialCounts, error)
	UserCounts(ctx context.Context, q *domainQuery.Query) (*OverviewUserCounts, error)
	CredentialRecap(ctx context.Context, asOf time.Time) ([]CredentialRecapRow, error)
}
