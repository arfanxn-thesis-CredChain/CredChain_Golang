package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredential_LifecycleStatus(t *testing.T) {
	ts := func(t time.Time) *time.Time { return &t }
	now := time.Now()

	tests := []struct {
		name     string
		cred     Credential
		expected CredentialLifecycleStatus
	}{
		{"pending when no timestamps", Credential{}, CredentialLifecycleStatusPending},
		{"approved when only approved_at", Credential{ApprovedAt: ts(now)}, CredentialLifecycleStatusApproved},
		{"rejected when only rejected_at", Credential{RejectedAt: ts(now)}, CredentialLifecycleStatusRejected},
		{"revoked when only revoked_at", Credential{RevokedAt: ts(now)}, CredentialLifecycleStatusRevoked},
		{"revoked beats approved", Credential{ApprovedAt: ts(now), RevokedAt: ts(now)}, CredentialLifecycleStatusRevoked},
		{"revoked beats rejected", Credential{RejectedAt: ts(now), RevokedAt: ts(now)}, CredentialLifecycleStatusRevoked},
		{"rejected beats approved", Credential{ApprovedAt: ts(now), RejectedAt: ts(now)}, CredentialLifecycleStatusRejected},
		{"expiry does not affect status", Credential{ApprovedAt: ts(now), ExpiresAt: ts(now)}, CredentialLifecycleStatusApproved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cred.LifecycleStatus())
		})
	}
}
