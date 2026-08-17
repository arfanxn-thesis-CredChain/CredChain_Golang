package response

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainUserUnit_AllFieldsSet(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	parentId := "parent-1"

	got := FromDomainUserUnit(domain.UserUnit{
		Id:        "unit-1",
		ParentId:  &parentId,
		Name:      "Informatics",
		CreatedAt: createdAt,
		UpdatedAt: &updatedAt,
	})

	assert.Equal(t, "unit-1", got.ID)
	assert.Equal(t, &parentId, got.ParentID)
	assert.Equal(t, "Informatics", got.Name)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, &updatedAt, got.UpdatedAt)
}

func TestFromDomainUserUnit_NilParentID(t *testing.T) {
	got := FromDomainUserUnit(domain.UserUnit{
		Id:   "unit-2",
		Name: "Faculty of Engineering",
	})

	assert.Equal(t, "unit-2", got.ID)
	assert.Nil(t, got.ParentID)
	assert.Equal(t, "Faculty of Engineering", got.Name)
}
