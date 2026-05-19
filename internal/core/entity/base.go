package entity

import (
	"time"

	"gorm.io/gorm"
)

type BaseEntity struct {
	ID        string `gorm:"primaryKey;type:varchar(50)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
