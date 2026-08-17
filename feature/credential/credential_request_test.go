package credential

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCredentialReExtractRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialReExtractRequest{Ids: []string{"01J0000000000000000000000A"}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialReExtractRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialReExtractRequest{Ids: ids}.Validate())
	})
}

func TestCredentialIssueInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := CredentialIssueInput{
			HolderUserID:         "01J0000000000000000000000A",
			Name:                 "Degree",
			TypeID:               "type-1",
			IssuerOrganizationID: "org-1",
		}
		assert.NoError(t, in.Validate())
	})
	t.Run("missing holder", func(t *testing.T) {
		in := CredentialIssueInput{Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1"}
		assert.Error(t, in.Validate())
	})
	t.Run("missing name", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "01J0000000000000000000000A", TypeID: "type-1", IssuerOrganizationID: "org-1"}
		assert.Error(t, in.Validate())
	})
	t.Run("missing type", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "h", Name: "Degree", IssuerOrganizationID: "org-1"}
		assert.Error(t, in.Validate())
	})
	t.Run("missing org", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "h", Name: "Degree", TypeID: "type-1"}
		assert.Error(t, in.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "h", Name: strings.Repeat("a", 257), TypeID: "type-1", IssuerOrganizationID: "org-1"}
		assert.Error(t, in.Validate())
	})
	t.Run("number too long", func(t *testing.T) {
		tooLong := strings.Repeat("1", 257)
		in := CredentialIssueInput{HolderUserID: "h", Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1", Number: &tooLong}
		assert.Error(t, in.Validate())
	})
	t.Run("bad issued_at date", func(t *testing.T) {
		bad := "not-a-date"
		in := CredentialIssueInput{HolderUserID: "h", Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1", IssuedAt: &bad}
		assert.Error(t, in.Validate())
	})
	t.Run("bad expires_at date", func(t *testing.T) {
		bad := "01/02/2026"
		in := CredentialIssueInput{HolderUserID: "h", Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1", ExpiresAt: &bad}
		assert.Error(t, in.Validate())
	})
}

func TestCredentialIssueInput_ToDomain(t *testing.T) {
	meta := map[string]any{"k": "v"}
	number := "N-001"
	issued := "2026-08-01"
	expires := "2026-09-01"
	in := CredentialIssueInput{
		HolderUserID:         "holder-1",
		Name:                 "Degree",
		Meta:                 meta,
		TypeID:               "type-1",
		IssuerOrganizationID: "org-1",
		Number:               &number,
		IssuedAt:             &issued,
		ExpiresAt:            &expires,
	}
	got := in.ToDomain()
	assert.Equal(t, "holder-1", got.HolderUserID)
	assert.Equal(t, "Degree", got.Name)
	assert.Equal(t, meta, got.Meta)
	assert.Equal(t, "type-1", got.TypeID)
	assert.Equal(t, "org-1", got.IssuerOrganizationID)
	assert.Equal(t, "N-001", *got.Number)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), got.IssuedAt)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *got.ExpiresAt)
}

func TestCredentialIssueRequest_Validate(t *testing.T) {
	validItem := func() CredentialIssueInput {
		return CredentialIssueInput{HolderUserID: "h", Name: "n", TypeID: "type-1", IssuerOrganizationID: "org-1"}
	}
	t.Run("valid", func(t *testing.T) {
		r := CredentialIssueRequest{Credentials: []CredentialIssueInput{validItem()}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty items", func(t *testing.T) {
		r := CredentialIssueRequest{Credentials: []CredentialIssueInput{}}
		assert.Error(t, r.Validate())
	})
	t.Run("too many items", func(t *testing.T) {
		items := make([]CredentialIssueInput, 101)
		for i := range items {
			items[i] = validItem()
		}
		r := CredentialIssueRequest{Credentials: items}
		assert.Error(t, r.Validate())
	})
	t.Run("invalid nested item", func(t *testing.T) {
		r := CredentialIssueRequest{Credentials: []CredentialIssueInput{{Name: "no-holder"}}}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialIssueRequest_ToDomain(t *testing.T) {
	r := CredentialIssueRequest{Credentials: []CredentialIssueInput{
		{HolderUserID: "h1", Name: "n1"},
		{HolderUserID: "h2", Name: "n2"},
	}}
	got := r.ToDomain()
	assert.Len(t, got, 2)
	assert.Equal(t, "h1", got[0].HolderUserID)
	assert.Equal(t, "n2", got[1].Name)
}

func TestCredentialRevokeRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, CredentialRevokeRequest{Ids: []string{"01J0"}}.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialRevokeRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialRevokeRequest{Ids: ids}.Validate())
	})
}

func TestCredentialSubmitInput_Validate(t *testing.T) {
	issued := "2026-08-01"
	valid := func() CredentialSubmitInput {
		return CredentialSubmitInput{Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1", IssuedAt: &issued}
	}
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid().Validate())
	})
	t.Run("missing name", func(t *testing.T) {
		in := valid()
		in.Name = ""
		assert.Error(t, in.Validate())
	})
	t.Run("missing type", func(t *testing.T) {
		in := valid()
		in.TypeID = ""
		assert.Error(t, in.Validate())
	})
	t.Run("missing org", func(t *testing.T) {
		in := valid()
		in.IssuerOrganizationID = ""
		assert.Error(t, in.Validate())
	})
	t.Run("missing issued_at", func(t *testing.T) {
		in := valid()
		in.IssuedAt = nil
		assert.Error(t, in.Validate())
	})
	t.Run("bad issued_at date", func(t *testing.T) {
		in := valid()
		bad := "not-a-date"
		in.IssuedAt = &bad
		assert.Error(t, in.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		in := valid()
		in.Name = strings.Repeat("a", 257)
		assert.Error(t, in.Validate())
	})
}

func TestCredentialSubmitRequest_Validate(t *testing.T) {
	issued := "2026-08-01"
	validItem := func() CredentialSubmitInput {
		return CredentialSubmitInput{Name: "Degree", TypeID: "type-1", IssuerOrganizationID: "org-1", IssuedAt: &issued}
	}
	t.Run("valid", func(t *testing.T) {
		r := CredentialSubmitRequest{Credentials: []CredentialSubmitInput{validItem()}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty items", func(t *testing.T) {
		r := CredentialSubmitRequest{Credentials: []CredentialSubmitInput{}}
		assert.Error(t, r.Validate())
	})
	t.Run("too many items", func(t *testing.T) {
		items := make([]CredentialSubmitInput, 101)
		for i := range items {
			items[i] = validItem()
		}
		r := CredentialSubmitRequest{Credentials: items}
		assert.Error(t, r.Validate())
	})
	t.Run("invalid nested item", func(t *testing.T) {
		r := CredentialSubmitRequest{Credentials: []CredentialSubmitInput{{Name: "no-issued-at"}}}
		assert.Error(t, r.Validate())
	})
}
