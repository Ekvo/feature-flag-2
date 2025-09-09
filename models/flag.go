//go:generate reform
package models

import (
	"github.com/google/uuid"
	"time"
)

//reform:public.flags
type Flag struct {
	FlagName    string    `json:"flag_name" reform:"flag_name,pk"`
	IsDeleted   bool      `json:"is_deleted" reform:"is_deleted"`
	IsEnabled   bool      `json:"is_enabled" reform:"is_enabled"`
	ActiveFrom  time.Time `json:"active_from" reform:"active_from"`
	Data        JSONmap   `json:"data" reform:"data"`
	DefaultData JSONmap   `json:"default_data" reform:"default_data"`
	CreatedBy   uuid.UUID `json:"created_by" reform:"created_by"`
	CreatedAt   time.Time `json:"created_at" reform:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" reform:"updated_at"`
}

// TableName возвращает имя таблицы
func (f *Flag) TableName() string {
	return "flags"
}
