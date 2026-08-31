package credential

import (
	"mime/multipart"
	"time"

	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
)

var allowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"image/tiff":      true,
}

const maxFileBytes = 10 * 1024 * 1024 // 10 MB

// parseDatePtr parses a "2006-01-02" string into *time.Time, returning nil for
// empty/invalid input. Shared by CredentialIssueInput.ToDomain and the Issue
// handler (which maps inputs straight into service-layer CredentialIssuance).
func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", *s); err == nil {
		return &t
	}
	return nil
}

// CredentialIssueInput is one item in a batch issue request.
type CredentialIssueInput struct {
	HolderUserID         string         `form:"holder_user_id"`
	TypeID               string         `form:"type_id"`
	IssuerOrganizationID string         `form:"issuer_organization_id"`
	Number               *string        `form:"number"`
	IssuedAt             *string        `form:"issued_at"`
	ExpiresAt            *string        `form:"expires_at"`
	Name                 string         `form:"name"`
	Meta                 map[string]any `form:"meta"`
	CompetencyIDs        []string       `form:"-"`
	File                 *multipart.FileHeader
}

func (n CredentialIssueInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.HolderUserID, validation.Required),
		validation.Field(&n.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&n.TypeID, validation.Required),
		validation.Field(&n.IssuerOrganizationID, validation.Required),
		validation.Field(&n.Number, validation.Length(0, 256)),
		validation.Field(&n.IssuedAt, validation.Date("2006-01-02")),
		validation.Field(&n.ExpiresAt, validation.Date("2006-01-02")),
	)
}

func (n CredentialIssueInput) ToDomain() domain.Credential {
	issuedAt := time.Time{}
	if t := parseDatePtr(n.IssuedAt); t != nil {
		issuedAt = *t
	}
	return domain.Credential{
		HolderUserID:         n.HolderUserID,
		IssuerOrganizationID: lo.ToPtr(n.IssuerOrganizationID),
		TypeID:               lo.ToPtr(n.TypeID),
		Number:               n.Number,
		Name:                 n.Name,
		Meta:                 n.Meta,
		IssuedAt:             issuedAt,
		ExpiresAt:            parseDatePtr(n.ExpiresAt),
	}
}

// CredentialIssueRequest is the parsed multipart batch issue request.
// Gin does not support nested multipart structs, so the handler builds this
// manually from c.MultipartForm().
type CredentialIssueRequest struct {
	Credentials []CredentialIssueInput
}

func (r CredentialIssueRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Credentials,
			validation.Required,
			validation.Length(1, 100),
			validation.Each(validation.By(func(v any) error {
				return v.(CredentialIssueInput).Validate()
			})),
		),
	)
}

func (r CredentialIssueRequest) ToDomain() []domain.Credential {
	out := make([]domain.Credential, len(r.Credentials))
	for i, item := range r.Credentials {
		out[i] = item.ToDomain()
	}
	return out
}

// CredentialUpdateInput is one item in a batch credential update request.
// id is required; every other field is optional (nil = unchanged).
type CredentialUpdateInput struct {
	Id                   string         `json:"id"`
	Name                 *string        `json:"name"`
	Number               *string        `json:"number"`
	TypeID               *string        `json:"type_id"`
	IssuerOrganizationID *string        `json:"issuer_organization_id"`
	IssuedAt             *string        `json:"issued_at"`
	ExpiresAt            *string        `json:"expires_at"`
	Meta                 map[string]any `json:"meta"`
}

func (n CredentialUpdateInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Id, validation.Required),
		validation.Field(&n.Name, validation.Length(0, 256)),
		validation.Field(&n.Number, validation.Length(0, 256)),
		validation.Field(&n.IssuedAt, validation.Date("2006-01-02")),
		validation.Field(&n.ExpiresAt, validation.Date("2006-01-02")),
	)
}

func (n CredentialUpdateInput) ToDomain() domain.Credential {
	c := domain.Credential{ID: n.Id}
	if n.Name != nil {
		c.Name = *n.Name
	}
	if n.Number != nil {
		c.Number = n.Number
	}
	if n.TypeID != nil {
		c.TypeID = n.TypeID
	}
	if n.IssuerOrganizationID != nil {
		c.IssuerOrganizationID = n.IssuerOrganizationID
	}
	if t := parseDatePtr(n.IssuedAt); t != nil {
		c.IssuedAt = *t
	}
	c.ExpiresAt = parseDatePtr(n.ExpiresAt)
	c.Meta = n.Meta
	return c
}

// CredentialUpdateRequest is the JSON body for PUT /api/credentials/batch.
type CredentialUpdateRequest struct {
	Credentials []CredentialUpdateInput `json:"credentials"`
}

func (r CredentialUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Credentials,
			validation.Required,
			validation.Length(1, 100),
			validation.Each(validation.By(func(v any) error {
				return v.(CredentialUpdateInput).Validate()
			})),
		),
	)
}

func (r CredentialUpdateRequest) ToDomain() []domain.Credential {
	if r.Credentials == nil {
		return []domain.Credential{}
	}
	out := make([]domain.Credential, len(r.Credentials))
	for i, item := range r.Credentials {
		out[i] = item.ToDomain()
	}
	return out
}

// CredentialRevokeRequest is the JSON body for POST /api/credentials/batch/revoke.
type CredentialRevokeRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialRevokeRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids,
			validation.Required,
			validation.Length(1, 100),
		),
	)
}

// CredentialReExtractRequest is the JSON body for POST /api/credentials/batch/reextract.
type CredentialReExtractRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialReExtractRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids, validation.Required, validation.Length(1, 100)),
	)
}

// CredentialSubmitInput is one item in a batch self-submission request.
type CredentialSubmitInput struct {
	Name                            string         `form:"name"`
	TypeID                          *string        `form:"type_id"`
	SubmittedTypeName               *string        `form:"submitted_type_name"`
	IssuerOrganizationID            *string        `form:"issuer_organization_id"`
	SubmittedIssuerOrganizationName *string        `form:"submitted_issuer_organization_name"`
	Number                          *string        `form:"number"`
	IssuedAt                        *string        `form:"issued_at"`
	ExpiresAt                       *string        `form:"expires_at"`
	CompetencyIDs                   []string       `form:"-"`
	SubmittedCompetencyNames        []string       `form:"submitted_competency_names"`
	Meta                            map[string]any `form:"meta"`
	File                            *multipart.FileHeader
}

func (n CredentialSubmitInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required, validation.Length(1, 256)),
		// Type and organization are id-or-name; submitValidate enforces the
		// "exactly one of" rule because it needs the taxonomy lookup anyway.
		validation.Field(&n.IssuedAt, validation.Required, validation.Date("2006-01-02")),
		validation.Field(&n.ExpiresAt, validation.Date("2006-01-02")),
		validation.Field(&n.Number, validation.Length(0, 256)),
	)
}

// CredentialSubmitRequest is the parsed multipart batch self-submission
// request. Gin does not support nested multipart structs, so the handler
// builds this manually from c.MultipartForm().
type CredentialSubmitRequest struct {
	Credentials []CredentialSubmitInput
}

func (r CredentialSubmitRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Credentials,
			validation.Required,
			validation.Length(1, 100),
			validation.Each(validation.By(func(v any) error {
				return v.(CredentialSubmitInput).Validate()
			})),
		),
	)
}

// CredentialResolveMetadataRequest is the reviewer's resolution payload for
// PUT /api/credentials/:id/metadata. Every field is optional; per kind, at
// most one of the link-existing and create-new fields may be set.
type CredentialResolveMetadataRequest struct {
	TypeID         *string `json:"type_id"`
	CreateTypeName *string `json:"create_type_name"`

	IssuerOrganizationID   *string `json:"issuer_organization_id"`
	CreateOrganizationName *string `json:"create_organization_name"`

	CompetencyIDs         []string `json:"competency_ids"`
	CreateCompetencyNames []string `json:"create_competency_names"`
}

func (n CredentialResolveMetadataRequest) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.TypeID,
			validation.When(n.CreateTypeName != nil, validation.Nil.Error("validation_resolve_type_id_and_name"))),
		validation.Field(&n.IssuerOrganizationID,
			validation.When(n.CreateOrganizationName != nil, validation.Nil.Error("validation_resolve_org_id_and_name"))),
	)
}

func (n CredentialResolveMetadataRequest) ToResolution(credentialID string) CredentialMetadataResolution {
	return CredentialMetadataResolution{
		CredentialID:           credentialID,
		TypeID:                 n.TypeID,
		CreateTypeName:         n.CreateTypeName,
		OrganizationID:         n.IssuerOrganizationID,
		CreateOrganizationName: n.CreateOrganizationName,
		CompetencyIDs:          n.CompetencyIDs,
		CreateCompetencyNames:  n.CreateCompetencyNames,
	}
}

// CredentialApproveRequest is the JSON body for POST /api/credentials/batch/approve.
type CredentialApproveRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialApproveRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids, validation.Required, validation.Length(1, 100)),
	)
}

// CredentialLinkCompetenciesRequest is the JSON body for
// PUT /api/credentials/:id/competencies.
type CredentialLinkCompetenciesRequest struct {
	CompetencyIDs []string `json:"competency_ids"`
}

func (r CredentialLinkCompetenciesRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.CompetencyIDs, validation.Length(0, 100)),
	)
}

// CredentialRejectionInput is one per-credential rejection in a batch reject
// request. The API always carries per-item reasons (the frontend copies one
// reason across items when the user wants a single batch reason).
type CredentialRejectionInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// CredentialRejectRequest is the JSON body for POST /api/credentials/batch/reject.
type CredentialRejectRequest struct {
	Rejections []CredentialRejectionInput `json:"rejections"`
}

func (r CredentialRejectRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Rejections,
			validation.Required,
			validation.Length(1, 100),
			validation.Each(validation.By(func(v any) error {
				in := v.(CredentialRejectionInput)
				return validation.ValidateStruct(&in,
					validation.Field(&in.ID, validation.Required),
					validation.Field(&in.Reason, validation.Required, validation.Length(1, 1000)),
				)
			})),
		),
	)
}
