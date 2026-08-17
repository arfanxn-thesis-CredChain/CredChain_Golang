package response

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainCompetency_AllFieldsSet(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	got := FromDomainCompetency(domain.Competency{
		Id:        "cpt-1",
		Name:      "Blockchain Fundamentals",
		CreatedAt: createdAt,
		UpdatedAt: &updatedAt,
	})

	assert.Equal(t, "cpt-1", got.ID)
	assert.Equal(t, "Blockchain Fundamentals", got.Name)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestFromDomainCompetency_NilUpdatedAt(t *testing.T) {
	got := FromDomainCompetency(domain.Competency{
		Id:   "cpt-2",
		Name: "Data Structures",
	})

	assert.Equal(t, "cpt-2", got.ID)
	assert.Equal(t, "Data Structures", got.Name)
	assert.Nil(t, got.UpdatedAt)
}
