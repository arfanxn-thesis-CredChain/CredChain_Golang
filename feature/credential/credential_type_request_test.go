package credential

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialTypeStoreRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialTypeStoreRequest{Name: "Degree"}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty name", func(t *testing.T) {
		r := CredentialTypeStoreRequest{Name: ""}
		assert.Error(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		r := CredentialTypeStoreRequest{Name: strings.Repeat("a", 257)}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialTypeUpdateRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		name := "Degree Certificate"
		r := CredentialTypeUpdateRequest{Name: &name}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty update", func(t *testing.T) {
		r := CredentialTypeUpdateRequest{}
		assert.NoError(t, r.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 257)
		r := CredentialTypeUpdateRequest{Name: &name}
		assert.Error(t, r.Validate())
	})
}
