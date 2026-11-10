package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupRoutes registers all API routes on the given Fiber app under the
// default "/api" prefix. Both the standalone server and CLI serve command
// use this to avoid duplication.
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	RegisterRoutes(app, db, "/api")
}

// RegisterRoutes registers all API routes under the given prefix. It is the
// single routing source shared by SetupRoutes and the embeddable engine
// facade (knot.Knot.Mount), which mounts the same routes under a
// host-chosen prefix. It must be exported because the facade lives in the
// module-root package knot, a different package from this one.
func RegisterRoutes(app *fiber.App, db *gorm.DB, prefix string) {
	api := app.Group(prefix)

	// Groups routes
	groups := api.Group("/groups")
	groups.Get("/", GetGroups(db))
	groups.Get("/with-apis", GetGroupsWithAPIs(db))
	groups.Post("/orders", UpdateGroupOrders(db))
	groups.Post("/", CreateGroup(db))
	groups.Patch("/:id", UpdateGroup(db))
	groups.Delete("/:id", DeleteGroup(db))

	// APIs routes
	apis := api.Group("/apis")
	// The flat list must be registered before (and independently of) the
	// "/:id" detail route: Fiber's router prefers the static "/" segment, so
	// GET /apis lists and GET /apis/5 resolves the detail handler.
	apis.Get("/", ListAPIs(db))
	apis.Get("/:id", GetAPI(db))
	apis.Get("/group/:groupId", GetAPIsByGroup(db))
	apis.Post("/", CreateAPI(db))
	apis.Patch("/:id", UpdateAPI(db))
	apis.Patch("/:id/note", UpdateAPINote(db))
	apis.Post("/orders", UpdateAPIOrders(db))
	apis.Post("/:id/duplicate", DuplicateAPI(db))
	apis.Delete("/:id", DeleteAPI(db))
	apis.Put("/:id/parameters", UpdateParameters(db))
	apis.Post("/:id/parameters/from-json", UpdateParametersFromJSON(db))

	// Export routes
	export := api.Group("/export")
	export.Post("/", ExportAPIs(db))

	// Import routes
	api.Post("/import", ImportDocument(db))

	// MCP Tools routes
	mcpTools := api.Group("/mcp-tools")
	mcpTools.Post("/", HandleMCPTools(db))
}
