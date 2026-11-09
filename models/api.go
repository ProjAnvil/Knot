package models

import (
	"encoding/json"
	"time"
)

// API represents an API endpoint
type API struct {
	ID         uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID    uint        `gorm:"not null;index:idx_group_id" json:"groupId"`
	Group      *Group      `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Name       string      `gorm:"not null" json:"name"`
	Endpoint   string      `gorm:"not null" json:"endpoint"`
	Method     string      `gorm:"type:varchar(10)" json:"method"` // GET, POST, PUT, DELETE, PATCH
	Type       string      `gorm:"not null" json:"type"`           // HTTP or RPC
	Order      int         `gorm:"default:0" json:"order"`
	Note       *string     `gorm:"type:text" json:"note"`
	Parameters []Parameter `gorm:"foreignKey:APIID;constraint:OnDelete:CASCADE" json:"parameters,omitempty"`
	CreatedAt  int64       `gorm:"autoCreateTime" json:"-"`
	UpdatedAt  int64       `gorm:"autoUpdateTime" json:"-"`
}

// TableName specifies the table name for API
func (API) TableName() string {
	return "apis"
}

// MarshalJSON customizes JSON serialization to convert timestamps to ISO 8601 strings
func (a API) MarshalJSON() ([]byte, error) {
	type Alias API
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
	}{
		Alias:     (*Alias)(&a),
		CreatedAt: time.Unix(a.CreatedAt, 0).UTC().Format(time.RFC3339),
		UpdatedAt: time.Unix(a.UpdatedAt, 0).UTC().Format(time.RFC3339),
	})
}
