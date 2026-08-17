package credential

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompetencyStoreRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CompetencyStoreRequest{Name: "Blockchain Fundamentals"}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty name", func(t *testing.T) {
		r := CompetencyStoreRequest{Name: ""}
		assert.Error(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		r := CompetencyStoreRequest{Name: strings.Repeat("a", 257)}
		assert.Error(t, r.Validate())
	})
}

func TestCompetencyUpdateRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		name := "Blockchain Fundamentals and Applications"
		r := CompetencyUpdateRequest{Name: &name}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty update", func(t *testing.T) {
		r := CompetencyUpdateRequest{}
		assert.NoError(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 257)
		r := CompetencyUpdateRequest{Name: &name}
		assert.Error(t, r.Validate())
	})
}
