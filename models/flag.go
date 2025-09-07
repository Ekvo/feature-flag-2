//go:generate reform
package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

//reform:public.flags
type Flag struct {
	FlagName    string          `json:"flag_name" reform:"flag_name,pk"`
	IsEnable    bool            `json:"is_enable" reform:"is_enable"`
	ActiveFrom  time.Time       `json:"active_from" reform:"active_from"`
	Data        json.RawMessage `json:"data" reform:"data"`
	DefaultData json.RawMessage `json:"default_data" reform:"default_data"`
	CreatedUser uuid.UUID       `json:"created_user" reform:"created_user"`
	CreatedAt   time.Time       `json:"created_at" reform:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" reform:"updated_at"`
}

// TableName возвращает имя таблицы
func (f *Flag) TableName() string {
	return "flags"
}
