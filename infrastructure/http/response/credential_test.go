package response

import (
	"encoding/json"
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainVerificationCredential_NullablePublicFacts(t *testing.T) {
	out := FromDomainVerificationCredential(domain.Credential{ID: "c1", HolderUserID: "h1", SubmitterUserID: "s1"})
	payload, err := json.Marshal(out)
	assert.NoError(t, err)
	var fields map[string]any
	assert.NoError(t, json.Unmarshal(payload, &fields))
	assert.Len(t, fields, 11)
	assert.Contains(t, fields, "issuer_organization_name")
	assert.Nil(t, fields["issuer_organization_name"])
	assert.Contains(t, fields, "credential_type_name")
	assert.Nil(t, fields["credential_type_name"])
	assert.Contains(t, fields, "number")
	assert.Contains(t, fields, "expires_at")
	assert.Nil(t, fields["number"])
	assert.Nil(t, fields["expires_at"])
	assert.Equal(t, false, fields["is_submission"])
}

func TestFromDomainVerificationCredential_OrganizationNameAllowlist(t *testing.T) {
	secret := "private-workflow-value"
	out := FromDomainVerificationCredential(domain.Credential{
		ID: "record", HolderUserID: secret, FileURI: &secret, ExtractError: &secret,
		Meta:               map[string]any{"private": secret},
		IssuerOrganization: &domain.CredentialIssuerOrganization{Name: "Linked School", SubmitterUserId: &secret, RejectionReason: &secret},
		Holder:             &domain.User{Email: "hidden@example.test"},
	})
	payload, err := json.Marshal(out)
	assert.NoError(t, err)
	var fields map[string]any
	assert.NoError(t, json.Unmarshal(payload, &fields))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, []string{"id", "name", "status", "is_submission", "issuer_organization_id", "issuer_organization_name", "credential_type_name", "number", "file_hash", "issued_at", "expires_at"}, keys)
	assert.Equal(t, "Linked School", fields["issuer_organization_name"])
	assert.Nil(t, fields["credential_type_name"])

	typeName := "Diploma"
	withType, err := json.Marshal(FromDomainVerificationCredential(domain.Credential{
		Type: &domain.CredentialType{Name: typeName},
	}))
	assert.NoError(t, err)
	assert.Contains(t, string(withType), `"credential_type_name":"Diploma"`)
	assert.NotContains(t, string(payload), secret)
	assert.NotContains(t, string(payload), "hidden@example.test")
}

func TestFromDomainVerificationPublicDetails_NullableAllowlist(t *testing.T) {
	out := FromDomainVerificationPublicDetails(domain.Credential{ID: "record", HolderUserID: "private-holder", SubmitterUserID: "private-submitter"}, nil)
	payload, err := json.Marshal(out)
	assert.NoError(t, err)
	var fields map[string]any
	assert.NoError(t, json.Unmarshal(payload, &fields))
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, []string{"id", "name", "status", "is_submission", "issuer_organization_id", "number", "file_hash", "issued_at", "expires_at", "issuer_organization_name", "credential_type_name", "revoked_at", "holder", "competencies", "meta", "token_id", "created_at", "activated_at", "rejected_at"}, keys)
	for _, key := range []string{"number", "expires_at", "issuer_organization_name", "revoked_at", "holder", "meta", "token_id", "activated_at", "rejected_at"} {
		assert.Nil(t, fields[key])
	}
	assert.Equal(t, []any{}, fields["competencies"])
	assert.NotContains(t, string(payload), "private-holder")
	assert.NotContains(t, string(payload), "private-submitter")
}

func TestVerificationPublicDetails_HolderAllowlist(t *testing.T) {
	name, number, unit := "Synthetic Holder", "123", "Engineering"
	year := 2026
	c := domain.Credential{
		ID: "record", Meta: map[string]any{"grade": "A"},
		Holder:       &domain.User{Id: "hidden-user", Name: &name, Number: &number, JoinedYear: &year, Email: "synthetic@example.test", Role: domain.RoleHolder, WalletAddress: "0xabc", Meta: map[string]any{"hidden": "private-meta"}, EncryptedWalletPrivateKey: "secret"},
		Issuer:       &domain.User{Email: "hidden-staff@example.test"},
		Competencies: []domain.Competency{{Name: "Design"}},
	}
	payload, err := json.Marshal(FromDomainVerificationPublicDetails(c, &unit))
	assert.NoError(t, err)
	var fields map[string]any
	assert.NoError(t, json.Unmarshal(payload, &fields))
	holder, ok := fields["holder"].(map[string]any)
	if !assert.True(t, ok) {
		return
	}
	keys := make([]string, 0, len(holder))
	for key := range holder {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, []string{"name", "number", "unit_name", "joined_year", "email", "gender", "role", "wallet_address", "created_at", "updated_at", "deleted_at"}, keys)
	assert.Equal(t, name, holder["name"])
	assert.Nil(t, holder["updated_at"])
	assert.Equal(t, []any{map[string]any{"name": "Design"}}, fields["competencies"])
	assert.Equal(t, map[string]any{"grade": "A"}, fields["meta"])
	for _, hidden := range []string{"hidden-user", "private-meta", "secret", "hidden-staff"} {
		assert.NotContains(t, string(payload), hidden)
	}
	assert.Equal(t, unit, holder["unit_name"])
}

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
		ActivatedAt:          &now,
		RejectedAt:           nil,
		CreatedAt:            now,
		UpdatedAt:            &now,
	})
	assert.Equal(t, "s1", out.SubmitterUserID)
	assert.Equal(t, "o1", out.IssuerOrganizationID)
	assert.Equal(t, "t1", out.TypeID)
	assert.Equal(t, &num, out.Number)
	assert.Equal(t, &now, out.ExpiresAt)
	assert.Equal(t, &now, out.ActivatedAt)
	assert.Nil(t, out.RejectedAt)
}

func TestFromDomainCredential_StatusDerivation(t *testing.T) {
	now := time.Now()
	ts := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name     string
		cred     domain.Credential
		expected domain.CredentialStatus
	}{
		{"pending when no timestamps", domain.Credential{}, domain.CredentialStatusPending},
		{"approved when activated_at set", domain.Credential{ActivatedAt: ts(now)}, domain.CredentialStatusApproved},
		{"rejected when rejected_at set", domain.Credential{RejectedAt: ts(now)}, domain.CredentialStatusRejected},
		{"revoked when revoked_at set", domain.Credential{RevokedAt: ts(now)}, domain.CredentialStatusRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := FromDomainCredential(tt.cred)
			assert.Equal(t, tt.expected, out.Status)
		})
	}
}
