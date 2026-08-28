package credential

import validation "github.com/go-ozzo/ozzo-validation/v4"

type CompetencyStoreRequest struct {
	Name   string `json:"name"`
	Active *bool  `json:"active"`
}

func (r CompetencyStoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 256)),
	)
}

type CompetencyUpdateRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

func (r CompetencyUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Length(0, 256)),
	)
}
