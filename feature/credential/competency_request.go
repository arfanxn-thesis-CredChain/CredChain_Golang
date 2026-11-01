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

type CompetencySubmitRequest struct {
	Name string `json:"name"`
}

func (r CompetencySubmitRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 256)),
	)
}

type CompetencyRejectRequest struct {
	Reason string `json:"reason"`
}

func (r CompetencyRejectRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Reason, validation.Required, validation.Length(1, 1024)),
	)
}
