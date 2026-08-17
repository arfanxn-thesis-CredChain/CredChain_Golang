package response

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainCredential_MapsNewFields(t *testing.T) {
	now := time.Now()
	num := "N-1"
	out := FromDomainCredential(domain.Credential{
		ID:                   "c1",
		SubmitterUserID:      "s1",
		IssuerOrganizationID: "o1",
		TypeID:               "t1",
		Number:               &num,
		ExpiresAt:            &now,
		ApprovedAt:           &now,
		RejectedAt:           nil,
		CreatedAt:            now,
		UpdatedAt:            &now,
	})
	assert.Equal(t, "s1", out.SubmitterUserID)
	assert.Equal(t, "o1", out.IssuerOrganizationID)
	assert.Equal(t, "t1", out.TypeID)
	assert.Equal(t, &num, out.Number)
	assert.Equal(t, &now, out.ExpiresAt)
	assert.Equal(t, &now, out.ApprovedAt)
	assert.Nil(t, out.RejectedAt)
}

func TestFromDomainCredential_LifecycleStatusDerivation(t *testing.T) {
	now := time.Now()
	ts := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name     string
		cred     domain.Credential
		expected domain.CredentialLifecycleStatus
	}{
		{"pending when no timestamps", domain.Credential{}, domain.CredentialLifecycleStatusPending},
		{"approved when approved_at set", domain.Credential{ApprovedAt: ts(now)}, domain.CredentialLifecycleStatusApproved},
		{"rejected when rejected_at set", domain.Credential{RejectedAt: ts(now)}, domain.CredentialLifecycleStatusRejected},
		{"revoked when revoked_at set", domain.Credential{RevokedAt: ts(now)}, domain.CredentialLifecycleStatusRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := FromDomainCredential(tt.cred)
			assert.Equal(t, tt.expected, out.LifecycleStatus)
		})
	}
}
