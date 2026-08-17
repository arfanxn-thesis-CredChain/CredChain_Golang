package response

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainCredentialType_AllFieldsSet(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	got := FromDomainCredentialType(domain.CredentialType{
		Id:        "type-1",
		Name:      "Degree",
		Active:    true,
		CreatedAt: createdAt,
		UpdatedAt: &updatedAt,
	})

	assert.Equal(t, "type-1", got.ID)
	assert.Equal(t, "Degree", got.Name)
	assert.True(t, got.Active)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestFromDomainCredentialType_NilUpdatedAt(t *testing.T) {
	got := FromDomainCredentialType(domain.CredentialType{
		Id:   "type-2",
		Name: "Certificate",
	})

	assert.Equal(t, "type-2", got.ID)
	assert.False(t, got.Active)
	assert.Nil(t, got.UpdatedAt)
}
