package credential

import (
	"mime/multipart"
	"time"

	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
		IssuerOrganizationID: n.IssuerOrganizationID,
		TypeID:               n.TypeID,
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
	Name                 string         `form:"name"`
	TypeID               string         `form:"type_id"`
	IssuerOrganizationID string         `form:"issuer_organization_id"`
	Number               *string        `form:"number"`
	IssuedAt             *string        `form:"issued_at"`
	ExpiresAt            *string        `form:"expires_at"`
	CompetencyIDs        []string       `form:"-"`
	Meta                 map[string]any `form:"meta"`
	File                 *multipart.FileHeader
}

func (n CredentialSubmitInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&n.TypeID, validation.Required),
		validation.Field(&n.IssuerOrganizationID, validation.Required),
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

// CredentialApproveRequest is the JSON body for POST /api/credentials/batch/approve.
type CredentialApproveRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialApproveRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids, validation.Required, validation.Length(1, 100)),
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
