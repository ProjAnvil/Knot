package models

import (
	"encoding/json"
	"time"
)

// Group represents an API group/collection
type Group struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"unique;not null" json:"name"`
	Order     int    `gorm:"default:0" json:"order"`
	APIs      []API  `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"apis,omitempty"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"-"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"-"`
}

// TableName specifies the table name for Group
func (Group) TableName() string {
	return "groups"
}

// MarshalJSON customizes JSON serialization to convert timestamps to ISO 8601 strings
func (g Group) MarshalJSON() ([]byte, error) {
	// Ensure APIs is never nil, always an empty array
	apis := g.APIs
	if apis == nil {
		apis = []API{}
	}

	return json.Marshal(&struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		APIs      []API  `json:"apis"`
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
	}{
		ID:        g.ID,
		Name:      g.Name,
		APIs:      apis,
		CreatedAt: time.Unix(g.CreatedAt, 0).UTC().Format(time.RFC3339),
		UpdatedAt: time.Unix(g.UpdatedAt, 0).UTC().Format(time.RFC3339),
	})
}
