package model

import (
	"time"

	"CredChain_Golang/domain"
)

type UserUnit struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	ParentId  *string    `gorm:"type:char(26);column:parent_id;index"`
	Name      string     `gorm:"type:varchar(256);column:name;not null"`
	Active    bool       `gorm:"type:boolean;column:active;not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (UserUnit) TableName() string { return "user_units" }

func (m UserUnit) ToDomain() domain.UserUnit {
	return domain.UserUnit{
		Id:        m.Id,
		ParentId:  m.ParentId,
		Name:      m.Name,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainUserUnit(u domain.UserUnit) UserUnit {
	return UserUnit{
		Id:        u.Id,
		ParentId:  u.ParentId,
		Name:      u.Name,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
