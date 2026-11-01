package domain

import (
	"testing"
)

func TestInactiveMetadata(t *testing.T) {
	tests := []struct {
		name string
		c    Credential
		want []string
	}{
		{
			name: "no relations preloaded",
			c:    Credential{},
			want: nil,
		},
		{
			name: "all relations active",
			c: Credential{
				Type:               &CredentialType{Id: "t1", Active: true},
				IssuerOrganization: &CredentialIssuerOrganization{Id: "o1", Active: true},
				Competencies:       []Competency{{Id: "c1", Active: true}},
			},
			want: nil,
		},
		{
			name: "inactive type",
			c: Credential{
				Type:               &CredentialType{Id: "t1", Active: false},
				IssuerOrganization: &CredentialIssuerOrganization{Id: "o1", Active: true},
			},
			want: []string{"type"},
		},
		{
			name: "inactive organization and competency",
			c: Credential{
				Type:               &CredentialType{Id: "t1", Active: true},
				IssuerOrganization: &CredentialIssuerOrganization{Id: "o1", Active: false},
				Competencies:       []Competency{{Id: "c1", Active: false}},
			},
			want: []string{"issuer_organization", "competency"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.InactiveMetadata()
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
