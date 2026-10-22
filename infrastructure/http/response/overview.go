package response

import (
	"CredChain_Golang/domain"
	"time"
)

type CredentialRecap struct {
	AsOf  time.Time                   `json:"as_of"`
	Rows  []domain.CredentialRecapRow `json:"rows"`
	Units []CredentialRecapUnit       `json:"units"`
}

type CredentialRecapUnit struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
	Active   bool    `json:"active"`
}

type Overview struct {
	CredentialCounts *OverviewCredentialCounts `json:"credential_counts,omitempty"`
	UserCounts       *OverviewUserCounts       `json:"user_counts,omitempty"`
	Recents          *OverviewRecents          `json:"recents,omitempty"`
	ChainDetails     *OverviewChainDetails     `json:"chain_details,omitempty"`
}

type OverviewCredentialCounts struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Revoked int `json:"revoked"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
}

func FromDomainOverviewCredentialCounts(d domain.OverviewCredentialCounts) OverviewCredentialCounts {
	return OverviewCredentialCounts{
		Total: d.Total, Active: d.Active, Revoked: d.Revoked,
		Pending: d.Pending, Failed: d.Failed,
	}
}

type OverviewUserCounts struct {
	Total      int `json:"total"`
	Holder     int `json:"holder"`
	Issuer     int `json:"issuer"`
	Admin      int `json:"admin"`
	SuperAdmin int `json:"super_admin"`
	Active     int `json:"active"`
	Trashed    int `json:"trashed"`
}

func FromDomainOverviewUserCounts(d domain.OverviewUserCounts) OverviewUserCounts {
	return OverviewUserCounts{
		Total: d.Total, Holder: d.Holder, Issuer: d.Issuer,
		Admin: d.Admin, SuperAdmin: d.SuperAdmin,
		Active: d.Active, Trashed: d.Trashed,
	}
}

type OverviewRecents struct {
	ActiveCredentials  []Credential `json:"active_credentials,omitempty"`
	RevokedCredentials []Credential `json:"revoked_credentials,omitempty"`
	StoredUsers        []User       `json:"stored_users,omitempty"`
}

type OverviewChainDetails struct {
	AuthorityContract string `json:"authority_contract"`
	RegistryContract  string `json:"registry_contract"`
	LastBlock         uint64 `json:"last_block"`
	RelayerAddress    string `json:"relayer_address"`
	RelayerBalance    string `json:"relayer_balance"`
}
