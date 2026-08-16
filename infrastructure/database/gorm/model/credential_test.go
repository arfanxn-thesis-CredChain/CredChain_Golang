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
		IssuerOrganizationID: "org-1",
		TypeID:               "type-1",
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
	assert.Equal(t, "org-1", m.IssuerOrganizationId)
	assert.Equal(t, "type-1", m.TypeId)
	assert.Equal(t, &num, m.Number)
	assert.Equal(t, &now, m.ExpiresAt)
	assert.Equal(t, &now, m.ApprovedAt)
	assert.Nil(t, m.RejectedAt)

	d := m.ToDomain()
	assert.Equal(t, "s1", d.SubmitterUserID)
	assert.Equal(t, "org-1", d.IssuerOrganizationID)
	assert.Equal(t, "type-1", d.TypeID)
	assert.Equal(t, num, *d.Number)
	assert.Equal(t, now, *d.ApprovedAt)
	assert.Equal(t, now, *d.ExpiresAt)
}

func strPtrModel(s string) *string { return &s }
