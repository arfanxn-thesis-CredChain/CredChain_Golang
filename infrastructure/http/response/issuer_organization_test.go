package response

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainIssuerOrganization_AllFieldsSet(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	got := FromDomainIssuerOrganization(domain.CredentialIssuerOrganization{
		Id:        "org-1",
		Name:      "Faculty of Computer Science",
		CreatedAt: createdAt,
		UpdatedAt: &updatedAt,
	})

	assert.Equal(t, "org-1", got.ID)
	assert.Equal(t, "Faculty of Computer Science", got.Name)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestFromDomainIssuerOrganization_NilUpdatedAt(t *testing.T) {
	got := FromDomainIssuerOrganization(domain.CredentialIssuerOrganization{
		Id:   "org-2",
		Name: "Faculty of Engineering",
	})

	assert.Equal(t, "org-2", got.ID)
	assert.Equal(t, "Faculty of Engineering", got.Name)
	assert.Nil(t, got.UpdatedAt)
}
