package secretmessage

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Secret struct {
	gorm.Model
	ID        string
	ExpiresAt time.Time
	Value     string
}

type Team struct {
	gorm.Model
	ID          string
	AccessToken string
	Scope       string
	Name        string
	Paid        sql.NullBool `gorm:"default:false"`
}
