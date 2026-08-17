package credential

import validation "github.com/go-ozzo/ozzo-validation/v4"

type CredentialTypeStoreRequest struct {
	Name   string `json:"name"`
	Active *bool  `json:"active"`
}

func (r CredentialTypeStoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 256)),
	)
}

type CredentialTypeUpdateRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

func (r CredentialTypeUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Length(0, 256)),
	)
}
