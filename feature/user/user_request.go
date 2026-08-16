package user

import (
	"time"

	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type UserStoreInput struct {
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Role      domain.Role    `json:"role"`
	Number    *string        `json:"number"`
	BirthDate *string        `json:"birth_date"`
	Gender    *string        `json:"gender"`
	Meta      map[string]any `json:"meta"`
}

func (n UserStoreInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&n.Email, validation.Required, is.Email, validation.Length(1, 256)),
		validation.Field(&n.Role, validation.Required, validation.In(domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder)),
		validation.Field(&n.Number, validation.Length(0, 256)),
		validation.Field(&n.BirthDate, validation.Date("2006-01-02")),
		validation.Field(&n.Gender, validation.In("male", "female")),
	)
}

func (n UserStoreInput) ToDomain() domain.User {
	var birthDate *time.Time
	if n.BirthDate != nil && *n.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", *n.BirthDate); err == nil {
			birthDate = &t
		}
	}
	var gender *domain.Gender
	if n.Gender != nil && *n.Gender != "" {
		g := domain.Gender(*n.Gender)
		gender = &g
	}
	return domain.User{
		Name:        &n.Name,
		Email:       n.Email,
		Role:        n.Role,
		Number:      n.Number,
		BirthDate:   birthDate,
		Gender:      gender,
		Meta:        n.Meta,
	}
}

type UserStoreRequest struct {
	Users []UserStoreInput `json:"users"`
}

func (r UserStoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Users, validation.Required, validation.Each(validation.By(func(value any) error {
			u := value.(UserStoreInput)
			return u.Validate()
		}))),
	)
}

func (r UserStoreRequest) ToDomain() []domain.User {
	if r.Users == nil {
		return []domain.User{}
	}
	users := make([]domain.User, len(r.Users))
	for i, u := range r.Users {
		users[i] = u.ToDomain()
	}
	return users
}

type UserUpdateSelfEmailRequest struct {
	Email   string `json:"email"`
	IdToken string `json:"id_token"`
}

func (r UserUpdateSelfEmailRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, is.Email, validation.Length(1, 256)),
		validation.Field(&r.IdToken, validation.Required),
	)
}

type UserRoleUpdateInput struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
}

func (r UserRoleUpdateInput) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.Role, validation.Required, validation.In(domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder)),
	)
}

type UserUpdateRoleRequest struct {
	UserRoles []UserRoleUpdateInput `json:"user_roles"`
}

func (r UserUpdateRoleRequest) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.UserRoles, validation.Required))
}

type UserDeleteRequest struct {
	Ids []string `json:"ids"`
}

func (r UserDeleteRequest) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.Ids, validation.Required))
}

type UserUpdateInput struct {
	Id          string         `json:"id"`
	Name        *string        `json:"name"`
	Number      *string        `json:"number"`
	BirthDate   *string        `json:"birth_date"`
	Gender      *string        `json:"gender"`
	Meta        map[string]any `json:"meta"`
	Email       *string        `json:"email"`
	Role        *domain.Role   `json:"role"`
}

func (n UserUpdateInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Id, validation.Required),
		validation.Field(&n.Name, validation.Length(0, 256)),
		validation.Field(&n.Number, validation.Length(0, 256)),
		validation.Field(&n.BirthDate, validation.Date("2006-01-02")),
		validation.Field(&n.Gender, validation.In("male", "female")),
		validation.Field(&n.Email, is.Email, validation.Length(1, 256)),
		validation.Field(&n.Role, validation.In(domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder)),
	)
}

func (n UserUpdateInput) ToDomain() domain.User {
	var birthDate *time.Time
	if n.BirthDate != nil && *n.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", *n.BirthDate); err == nil {
			birthDate = &t
		}
	}
	var gender *domain.Gender
	if n.Gender != nil && *n.Gender != "" {
		g := domain.Gender(*n.Gender)
		gender = &g
	}
	u := domain.User{
		Id:          n.Id,
		Name:        n.Name,
		Number:      n.Number,
		BirthDate:   birthDate,
		Gender:      gender,
		Meta:        n.Meta,
	}
	if n.Email != nil {
		u.Email = *n.Email
	}
	if n.Role != nil {
		u.Role = *n.Role
	}
	return u
}

type UserUpdateRequest struct {
	Users []UserUpdateInput `json:"users"`
}

func (r UserUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Users, validation.Required, validation.Each(validation.By(func(value any) error {
			u := value.(UserUpdateInput)
			return u.Validate()
		}))),
	)
}

func (r UserUpdateRequest) ToDomain() []domain.User {
	if r.Users == nil {
		return []domain.User{}
	}
	users := make([]domain.User, len(r.Users))
	for i, u := range r.Users {
		users[i] = u.ToDomain()
	}
	return users
}

type UserTransferSuperAdminRequest struct {
	Id string `json:"id"`
}

func (r UserTransferSuperAdminRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Id, validation.Required),
	)
}

type UserRestoreRequest struct {
	IDs []string `json:"ids"`
}

func (r UserRestoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.IDs, validation.Required, validation.Length(1, 0)),
	)
}
