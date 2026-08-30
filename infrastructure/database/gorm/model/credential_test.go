package model

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestCredentialModel_NewFields_RoundTrip(t *testing.T) {
	now := time.Now()
	num := "CRED-0001"
	c := domain.Credential{
		ID:                   "cred-1",
		HolderUserID:         "h1",
		SubmitterUserID:      "s1",
		IssuerUserID:         "i1",
		IssuerOrganizationID: strPtrModel("org-1"),
		TypeID:               strPtrModel("type-1"),
		Number:               &num,
		Name:                 "Degree",
		FileHash:             "0xabc",
		ApproverUserID:       strPtrModel("a1"),
		ApprovedAt:           &now,
		ExpiresAt:            &now,
		RejectionReason:      nil,
	}

	m := FromDomainCredential(c)
	assert.Equal(t, "s1", m.SubmitterUserId)
	assert.Equal(t, "org-1", *m.IssuerOrganizationId)
	assert.Equal(t, "type-1", *m.TypeId)
	assert.Equal(t, &num, m.Number)
	assert.Equal(t, &now, m.ExpiresAt)
	assert.Equal(t, &now, m.ApprovedAt)
	assert.Nil(t, m.RejectedAt)

	d := m.ToDomain()
	assert.Equal(t, "s1", d.SubmitterUserID)
	assert.Equal(t, "org-1", *d.IssuerOrganizationID)
	assert.Equal(t, "type-1", *d.TypeID)
	assert.Equal(t, num, *d.Number)
	assert.Equal(t, now, *d.ApprovedAt)
	assert.Equal(t, now, *d.ExpiresAt)
}

// Regression: a revoked credential has revoker_user_id set, but verify never
// preloads the revoker. ToDomain must NOT fabricate an empty Revoker from the
// zero-value association — doing so surfaced a phantom blank avatar in the UI.
func TestCredential_ToDomain_NoPhantomRelationsWhenUnpreloaded(t *testing.T) {
	revokerID := "01REVOKER"
	m := Credential{
		Id:            "01CRED",
		HolderUserId:  "01HOLDER",
		IssuerUserId:  "01ISSUER",
		RevokerUserId: &revokerID,
		// HolderUser / IssuerUser / RevokerUser left as zero User{} (not preloaded).
	}

	d := m.ToDomain()

	assert.Equal(t, &revokerID, d.RevokerUserID, "FK still mapped")
	assert.Nil(t, d.Holder, "un-preloaded holder must be nil")
	assert.Nil(t, d.Issuer, "un-preloaded issuer must be nil")
	assert.Nil(t, d.Revoker, "un-preloaded revoker must be nil (no phantom empty user)")
}

func TestCredential_ToDomain_MapsPreloadedRelations(t *testing.T) {
	revokerID := "01REVOKER"
	m := Credential{
		Id:            "01CRED",
		HolderUserId:  "01HOLDER",
		IssuerUserId:  "01ISSUER",
		RevokerUserId: &revokerID,
		HolderUser:    User{Id: "01HOLDER"},
		IssuerUser:    User{Id: "01ISSUER"},
		RevokerUser:   User{Id: "01REVOKER"},
	}

	d := m.ToDomain()

	assert.NotNil(t, d.Holder)
	assert.NotNil(t, d.Issuer)
	assert.NotNil(t, d.Revoker)
	assert.Equal(t, "01REVOKER", d.Revoker.Id)
}

func strPtrModel(s string) *string { return &s }
