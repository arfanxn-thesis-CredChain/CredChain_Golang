package credential

import validation "github.com/go-ozzo/ozzo-validation/v4"

type IssuerOrganizationStoreRequest struct {
	Name   string `json:"name"`
	Active *bool  `json:"active"`
}

func (r IssuerOrganizationStoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 256)),
	)
}

type IssuerOrganizationUpdateRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

func (r IssuerOrganizationUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Length(0, 256)),
	)
}
