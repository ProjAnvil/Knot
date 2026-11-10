package models

// TablePrefix is prepended to every model table name (see Group.TableName,
// API.TableName and Parameter.TableName). The default empty string keeps the
// v1.0 physical schema unchanged: groups, apis, parameters.
//
// TablePrefix is process-global state. Embedders that share one database
// with other software set it (e.g. "knot_") so Knot's tables are namespaced;
// everything else leaves it empty. Use SetTablePrefix to assign it — never
// write the variable from multiple goroutines.
var TablePrefix string

// SetTablePrefix sets the process-wide prefix for all model table names.
//
// Contract: call it exactly once, before the first database use (before
// opening a connection or running any migration/query), and run a single
// engine instance per process. GORM resolves each model's table name when
// its schema is first parsed and caches the result per *gorm.DB, so a
// running database never observes later prefix changes — changing the prefix
// after tables exist will not rename them and will strand existing data
// under the old names.
func SetTablePrefix(p string) {
	TablePrefix = p
}
