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
