package model

import (
	"time"

	"CredChain_Golang/domain"
)

type Competency struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	Name      string     `gorm:"type:varchar(256);column:name;not null;uniqueIndex"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (Competency) TableName() string { return "competencies" }

func (m Competency) ToDomain() domain.Competency {
	return domain.Competency{
		Id:        m.Id,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainCompetency(c domain.Competency) Competency {
	return Competency{
		Id:        c.Id,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
