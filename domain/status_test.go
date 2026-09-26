package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredential_Status(t *testing.T) {
	ts := func(t time.Time) *time.Time { return &t }
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		cred     Credential
		expected CredentialStatus
	}{
		{"pending when no timestamps", Credential{}, CredentialStatusPending},
		{"approved when only approved_at", Credential{ApprovedAt: ts(now)}, CredentialStatusApproved},
		{"rejected when only rejected_at", Credential{RejectedAt: ts(now)}, CredentialStatusRejected},
		{"revoked when only revoked_at", Credential{RevokedAt: ts(now)}, CredentialStatusRevoked},
		{"revoked beats approved", Credential{ApprovedAt: ts(now), RevokedAt: ts(now)}, CredentialStatusRevoked},
		{"revoked beats rejected", Credential{RejectedAt: ts(now), RevokedAt: ts(now)}, CredentialStatusRevoked},
		{"rejected beats approved", Credential{ApprovedAt: ts(now), RejectedAt: ts(now)}, CredentialStatusRejected},
		{"revoked beats expired", Credential{ApprovedAt: ts(now), ExpiresAt: ts(past), RevokedAt: ts(now)}, CredentialStatusRevoked},
		{"rejected beats expired", Credential{RejectedAt: ts(now), ExpiresAt: ts(past)}, CredentialStatusRejected},
		{"expired when approved and expires_at in past", Credential{ApprovedAt: ts(now), ExpiresAt: ts(past)}, CredentialStatusExpired},
		{"approved when approved and expires_at in future", Credential{ApprovedAt: ts(now), ExpiresAt: ts(future)}, CredentialStatusApproved},
		{"expired when pending and expires_at in past", Credential{ExpiresAt: ts(past)}, CredentialStatusExpired},
		{"pending when pending and expires_at in future", Credential{ExpiresAt: ts(future)}, CredentialStatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cred.Status())
		})
	}
}

func TestCredential_ExtractState(t *testing.T) {
	ts := func(t time.Time) *time.Time { return &t }
	now := time.Now()

	tests := []struct {
		name     string
		cred     Credential
		expected ExtractState
	}{
		{"unextracted when no timestamps", Credential{}, ExtractStateUnextracted},
		{"pending when only enqueued_at", Credential{ExtractEnqueuedAt: ts(now)}, ExtractStatePending},
		{"succeeded when extracted_at set", Credential{ExtractEnqueuedAt: ts(now), ExtractedAt: ts(now)}, ExtractStateSucceeded},
		{"failed when failed_at set", Credential{ExtractEnqueuedAt: ts(now), ExtractFailedAt: ts(now)}, ExtractStateFailed},
		{"failed beats succeeded (re-extract failed again)", Credential{ExtractEnqueuedAt: ts(now), ExtractedAt: ts(now), ExtractFailedAt: ts(now)}, ExtractStateFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cred.ExtractState())
		})
	}
}
