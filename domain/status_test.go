package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredential_Status(t *testing.T) {
	ts := func(t time.Time) *time.Time { return &t }
	now := time.Now()

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
		{"expiry does not affect status", Credential{ApprovedAt: ts(now), ExpiresAt: ts(now)}, CredentialStatusApproved},
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
