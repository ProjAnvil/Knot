package models

import (
	"encoding/json"
	"time"
)

// Parameter represents request/response parameters with support for nested structures
type Parameter struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	APIID       uint        `gorm:"not null;index:idx_api_param" json:"apiId"`
	API         *API        `gorm:"foreignKey:APIID" json:"-"`
	ParentID    *uint       `gorm:"index:idx_parent" json:"parentId"`
	Parent      *Parameter  `gorm:"foreignKey:ParentID" json:"-"`
	Children    []Parameter `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Name        string      `gorm:"not null" json:"name"`
	Type        string      `gorm:"not null" json:"type"` // string, number, boolean, array, object
	Description *string     `gorm:"type:text" json:"description"`
	Required    bool        `gorm:"default:false" json:"required"`
	ParamType   string      `gorm:"not null;index:idx_api_param" json:"paramType"` // request or response
	Order       int         `gorm:"not null;index:idx_api_param" json:"order"`
	CreatedAt   int64       `gorm:"autoCreateTime" json:"-"`
	UpdatedAt   int64       `gorm:"autoUpdateTime" json:"-"`
}

// TableName specifies the table name for Parameter: TablePrefix + "parameters".
// The default empty prefix keeps the v1.0 table name "parameters".
func (Parameter) TableName() string {
	return TablePrefix + "parameters"
}

// MarshalJSON customizes JSON serialization to convert timestamps to ISO 8601 strings
func (p Parameter) MarshalJSON() ([]byte, error) {
	// The alias is unexported so code generators (tygo) do not pick the
	// local type up as an exported declaration.
	type alias Parameter
	return json.Marshal(&struct {
		*alias
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
	}{
		alias:     (*alias)(&p),
		CreatedAt: time.Unix(p.CreatedAt, 0).UTC().Format(time.RFC3339),
		UpdatedAt: time.Unix(p.UpdatedAt, 0).UTC().Format(time.RFC3339),
	})
}
