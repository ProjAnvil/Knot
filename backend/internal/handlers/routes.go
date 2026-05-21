package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupRoutes registers all API routes on the given Fiber app.
// Both the standalone server and CLI serve command use this to avoid duplication.
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	api := app.Group("/api")

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

	// MCP Tools routes
	mcpTools := api.Group("/mcp-tools")
	mcpTools.Post("/", HandleMCPTools(db))
}
