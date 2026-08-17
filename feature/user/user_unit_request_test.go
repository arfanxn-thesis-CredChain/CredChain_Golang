package user

import (
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestUserUnitStoreRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := UserUnitStoreRequest{Name: "Informatics"}
		assert.NoError(t, r.Validate())
	})
	t.Run("valid with parent", func(t *testing.T) {
		r := UserUnitStoreRequest{Name: "Informatics", ParentID: lo.ToPtr(strings.Repeat("a", 26))}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty name", func(t *testing.T) {
		r := UserUnitStoreRequest{Name: ""}
		assert.Error(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		r := UserUnitStoreRequest{Name: strings.Repeat("a", 257)}
		assert.Error(t, r.Validate())
	})
	t.Run("parent too long", func(t *testing.T) {
		r := UserUnitStoreRequest{Name: "Informatics", ParentID: lo.ToPtr(strings.Repeat("a", 27))}
		assert.Error(t, r.Validate())
	})
}

func TestUserUnitUpdateRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		name := "Informatics"
		r := UserUnitUpdateRequest{Name: &name}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty update", func(t *testing.T) {
		r := UserUnitUpdateRequest{}
		assert.NoError(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 257)
		r := UserUnitUpdateRequest{Name: &name}
		assert.Error(t, r.Validate())
	})
	t.Run("parent too long", func(t *testing.T) {
		r := UserUnitUpdateRequest{ParentID: lo.ToPtr(strings.Repeat("a", 27))}
		assert.Error(t, r.Validate())
	})
}
