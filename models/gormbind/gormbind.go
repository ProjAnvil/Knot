// Package gormbind binds the persistence-free domain models of the models
// module to GORM. It is the only place in the module allowed to import
// gorm.io/gorm, keeping the root models package stdlib-only while still
// exposing a single canonical migration list shared by the backend and
// any other consumer.
package gormbind

import (
	models "github.com/ProjAnvil/knot/models"
	"gorm.io/gorm"
)

// MigrateModels returns the canonical migration list, same order as
// backend/internal/database uses today: Group, API, Parameter.
func MigrateModels() []any {
	return []any{
		&models.Group{},
		&models.API{},
		&models.Parameter{},
	}
}

// AutoMigrate runs gorm's AutoMigrate over MigrateModels().
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(MigrateModels()...)
}
