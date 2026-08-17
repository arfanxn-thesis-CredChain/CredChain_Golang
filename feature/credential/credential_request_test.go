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

func TestCredentialUpdateInput_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		assert.NoError(t, CredentialUpdateInput{Id: "01J0000000000000000000000A"}.Validate())
	})
	t.Run("valid full", func(t *testing.T) {
		name := "Degree"
		number := "N-001"
		typeID := "type-1"
		orgID := "org-1"
		issued := "2026-08-01"
		expires := "2026-09-01"
		in := CredentialUpdateInput{
			Id: "01J0000000000000000000000A", Name: &name, Number: &number,
			TypeID: &typeID, IssuerOrganizationID: &orgID, IssuedAt: &issued, ExpiresAt: &expires,
		}
		assert.NoError(t, in.Validate())
	})
	t.Run("missing id", func(t *testing.T) {
		assert.Error(t, CredentialUpdateInput{}.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 257)
		assert.Error(t, CredentialUpdateInput{Id: "c1", Name: &name}.Validate())
	})
	t.Run("number too long", func(t *testing.T) {
		number := strings.Repeat("1", 257)
		assert.Error(t, CredentialUpdateInput{Id: "c1", Number: &number}.Validate())
	})
	t.Run("bad issued_at date", func(t *testing.T) {
		bad := "not-a-date"
		assert.Error(t, CredentialUpdateInput{Id: "c1", IssuedAt: &bad}.Validate())
	})
	t.Run("bad expires_at date", func(t *testing.T) {
		bad := "01/02/2026"
		assert.Error(t, CredentialUpdateInput{Id: "c1", ExpiresAt: &bad}.Validate())
	})
}

func TestCredentialUpdateInput_ToDomain(t *testing.T) {
	name := "Degree"
	number := "N-001"
	typeID := "type-1"
	orgID := "org-1"
	issued := "2026-08-01"
	expires := "2026-09-01"
	in := CredentialUpdateInput{
		Id: "c1", Name: &name, Number: &number,
		TypeID: &typeID, IssuerOrganizationID: &orgID, IssuedAt: &issued, ExpiresAt: &expires,
		Meta: map[string]any{"k": "v"},
	}
	got := in.ToDomain()
	assert.Equal(t, "c1", got.ID)
	assert.Equal(t, "Degree", got.Name)
	assert.Equal(t, "N-001", *got.Number)
	assert.Equal(t, "type-1", got.TypeID)
	assert.Equal(t, "org-1", got.IssuerOrganizationID)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), got.IssuedAt)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *got.ExpiresAt)
	assert.Equal(t, map[string]any{"k": "v"}, got.Meta)
}

func TestCredentialUpdateInput_ToDomain_Partial(t *testing.T) {
	got := CredentialUpdateInput{Id: "c1"}.ToDomain()
	assert.Equal(t, "c1", got.ID)
	assert.Equal(t, "", got.Name)
	assert.Nil(t, got.Number)
	assert.Nil(t, got.ExpiresAt)
	assert.True(t, got.IssuedAt.IsZero())
}

func TestCredentialUpdateRequest_Validate(t *testing.T) {
	validItem := func() CredentialUpdateInput { return CredentialUpdateInput{Id: "c1"} }
	t.Run("valid", func(t *testing.T) {
		r := CredentialUpdateRequest{Credentials: []CredentialUpdateInput{validItem()}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty items", func(t *testing.T) {
		r := CredentialUpdateRequest{Credentials: []CredentialUpdateInput{}}
		assert.Error(t, r.Validate())
	})
	t.Run("too many items", func(t *testing.T) {
		items := make([]CredentialUpdateInput, 101)
		for i := range items {
			items[i] = validItem()
		}
		assert.Error(t, CredentialUpdateRequest{Credentials: items}.Validate())
	})
	t.Run("invalid nested item", func(t *testing.T) {
		r := CredentialUpdateRequest{Credentials: []CredentialUpdateInput{{}}}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialUpdateRequest_ToDomain(t *testing.T) {
	r := CredentialUpdateRequest{Credentials: []CredentialUpdateInput{
		{Id: "c1"},
		{Id: "c2"},
	}}
	out := r.ToDomain()
	assert.Len(t, out, 2)
	assert.Equal(t, "c1", out[0].ID)
	assert.Equal(t, "c2", out[1].ID)
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

func TestCredentialApproveRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, CredentialApproveRequest{Ids: []string{"01J0"}}.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialApproveRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialApproveRequest{Ids: ids}.Validate())
	})
}

func TestCredentialRejectRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialRejectRequest{Rejections: []CredentialRejectionInput{{ID: "01J0", Reason: "bad scan"}}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialRejectRequest{Rejections: []CredentialRejectionInput{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		rejections := make([]CredentialRejectionInput, 101)
		for i := range rejections {
			rejections[i] = CredentialRejectionInput{ID: "x", Reason: "y"}
		}
		assert.Error(t, CredentialRejectRequest{Rejections: rejections}.Validate())
	})
	t.Run("missing id", func(t *testing.T) {
		r := CredentialRejectRequest{Rejections: []CredentialRejectionInput{{ID: "", Reason: "bad scan"}}}
		assert.Error(t, r.Validate())
	})
	t.Run("missing reason", func(t *testing.T) {
		r := CredentialRejectRequest{Rejections: []CredentialRejectionInput{{ID: "01J0", Reason: ""}}}
		assert.Error(t, r.Validate())
	})
	t.Run("reason too long", func(t *testing.T) {
		r := CredentialRejectRequest{Rejections: []CredentialRejectionInput{{ID: "01J0", Reason: strings.Repeat("a", 1001)}}}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialLinkCompetenciesRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, CredentialLinkCompetenciesRequest{CompetencyIDs: []string{"comp-1", "comp-2"}}.Validate())
	})
	t.Run("empty set allowed", func(t *testing.T) {
		assert.NoError(t, CredentialLinkCompetenciesRequest{}.Validate())
		assert.NoError(t, CredentialLinkCompetenciesRequest{CompetencyIDs: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialLinkCompetenciesRequest{CompetencyIDs: ids}.Validate())
	})
	t.Run("100 allowed", func(t *testing.T) {
		ids := make([]string, 100)
		for i := range ids {
			ids[i] = "x"
		}
		assert.NoError(t, CredentialLinkCompetenciesRequest{CompetencyIDs: ids}.Validate())
	})
}
