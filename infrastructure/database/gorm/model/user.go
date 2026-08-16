package model

import (
	"CredChain_Golang/domain"
	"time"

	"gorm.io/gorm"
)

type User struct {
	Id                        string         `gorm:"primaryKey;type:varchar(255);column:id" json:"id"`
	Name                      *string        `gorm:"type:varchar(255);column:name" json:"name"`
	Number                    *string        `gorm:"type:varchar(50);column:number" json:"number"`
	UnitID                    *string        `gorm:"type:char(26);column:unit_id;index" json:"unit_id"`
	JoinedYear                *int           `gorm:"column:joined_year;index" json:"joined_year"`
	Email                     string         `gorm:"type:varchar(255);uniqueIndex;column:email" json:"email"`
	Gender                    *string        `gorm:"type:gender;column:gender" json:"gender"`
	BirthDate                 *time.Time     `gorm:"column:birth_date" json:"birth_date"`
	Meta                      map[string]any `gorm:"type:jsonb;serializer:json;column:meta" json:"meta"`
	Role                      string         `gorm:"type:varchar(50);column:role" json:"role"`
	WalletAddress             string         `gorm:"type:varchar(255);column:wallet_address" json:"wallet_address"`
	EncryptedWalletPrivateKey string         `gorm:"type:varchar(255);column:encrypted_wallet_private_key" json:"-"`
	CreatedAt                 time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt                 *time.Time     `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`
}

func (m *User) ToDomain() domain.User {
	var deletedAt *time.Time
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		deletedAt = &t
	}
	var gender *domain.Gender
	if m.Gender != nil {
		g := domain.Gender(*m.Gender)
		gender = &g
	}
	return domain.User{
		Id:                        m.Id,
		Name:                      m.Name,
		Number:                    m.Number,
		UnitID:                    m.UnitID,
		JoinedYear:                m.JoinedYear,
		Email:                     m.Email,
		Gender:                    gender,
		BirthDate:                 m.BirthDate,
		Meta:                      m.Meta,
		Role:                      domain.Role(m.Role),
		WalletAddress:             m.WalletAddress,
		EncryptedWalletPrivateKey: m.EncryptedWalletPrivateKey,
		CreatedAt:                 m.CreatedAt,
		UpdatedAt:                 m.UpdatedAt,
		DeletedAt:                 deletedAt,
	}
}

func FromDomainUser(u domain.User) User {
	var deletedAt gorm.DeletedAt
	if u.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *u.DeletedAt, Valid: true}
	}
	var gender *string
	if u.Gender != nil {
		g := string(*u.Gender)
		gender = &g
	}
	return User{
		Id:                        u.Id,
		Name:                      u.Name,
		Number:                    u.Number,
		UnitID:                    u.UnitID,
		JoinedYear:                u.JoinedYear,
		Email:                     u.Email,
		Gender:                    gender,
		BirthDate:                 u.BirthDate,
		Meta:                      u.Meta,
		Role:                      string(u.Role),
		WalletAddress:             u.WalletAddress,
		EncryptedWalletPrivateKey: u.EncryptedWalletPrivateKey,
		CreatedAt:                 u.CreatedAt,
		UpdatedAt:                 u.UpdatedAt,
		DeletedAt:                 deletedAt,
	}
}
