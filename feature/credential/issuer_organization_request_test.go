package credential

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssuerOrganizationStoreRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := IssuerOrganizationStoreRequest{Name: "Faculty of Computer Science"}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty name", func(t *testing.T) {
		r := IssuerOrganizationStoreRequest{Name: ""}
		assert.Error(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		r := IssuerOrganizationStoreRequest{Name: strings.Repeat("a", 257)}
		assert.Error(t, r.Validate())
	})
}

func TestIssuerOrganizationUpdateRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		name := "Faculty of Computer Science and Engineering"
		r := IssuerOrganizationUpdateRequest{Name: &name}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty update", func(t *testing.T) {
		r := IssuerOrganizationUpdateRequest{}
		assert.NoError(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 257)
		r := IssuerOrganizationUpdateRequest{Name: &name}
		assert.Error(t, r.Validate())
	})
}
