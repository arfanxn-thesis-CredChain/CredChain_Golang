package model

import (
	"time"

	"CredChain_Golang/domain"
)

// Competency maps to competencies. Note: the column is `active` (NOT
// `is_active`). The GORM tag intentionally omits `default:true` so GORM inserts
// an explicit value for Active=false instead of letting the DB default silently
// override it.
type Competency struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	Name      string     `gorm:"type:varchar(256);column:name;not null;uniqueIndex"`
	Active    bool       `gorm:"type:boolean;column:active;not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (Competency) TableName() string { return "competencies" }

func (m Competency) ToDomain() domain.Competency {
	return domain.Competency{
		Id:        m.Id,
		Name:      m.Name,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainCompetency(c domain.Competency) Competency {
	return Competency{
		Id:        c.Id,
		Name:      c.Name,
		Active:    c.Active,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
