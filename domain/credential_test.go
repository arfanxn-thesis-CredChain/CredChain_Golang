package domain

import (
	"testing"

	"github.com/samber/lo"
)

// No local pointer helper here — samber/lo is already a direct dependency
// (used throughout feature/credential/) and provides lo.ToPtr for exactly
// this. Do not reintroduce a hand-rolled strp/ptr helper.

func TestUnresolvedMetadata(t *testing.T) {
	tests := []struct {
		name string
		c    Credential
		want []string
	}{
		{
			name: "fully resolved",
			c: Credential{
				TypeID:               lo.ToPtr("01J0TYPE"),
				IssuerOrganizationID: lo.ToPtr("01J0ORG"),
				SubmittedCompetencies: []SubmittedCompetency{
					{Name: "Discrete Math", ResolvedID: lo.ToPtr("01J0COMP")},
				},
			},
			want: nil,
		},
		{
			name: "unresolved type",
			c: Credential{
				SubmittedTypeName:    lo.ToPtr("Micro-credential"),
				IssuerOrganizationID: lo.ToPtr("01J0ORG"),
			},
			want: []string{"type"},
		},
		{
			name: "unresolved organization and competency",
			c: Credential{
				TypeID:                          lo.ToPtr("01J0TYPE"),
				SubmittedIssuerOrganizationName: lo.ToPtr("Cyfrin Updraft"),
				SubmittedCompetencies: []SubmittedCompetency{
					{Name: "Discrete Math"},
				},
			},
			want: []string{"issuer_organization", "competency"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.UnresolvedMetadata()
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
