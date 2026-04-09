package models

import (
	"time"
)

// DailyUsage melacak jumlah request dan token yang dipakai per hari untuk masing-masing model.
type DailyUsage struct {
	ID       uint      `gorm:"primaryKey"`
	Date     time.Time `gorm:"type:date;uniqueIndex:idx_date_model"`
	AIModel  string    `gorm:"size:50;uniqueIndex:idx_date_model"`
	Requests int       `gorm:"default:0"`
	Tokens   int       `gorm:"default:0"`
}
