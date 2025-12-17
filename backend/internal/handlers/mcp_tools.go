package handlers

import (
	"strings"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/internal/services"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// HandleMCPTools handles all MCP tool calls
func HandleMCPTools(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Tool string                 `json:"tool"`
			Args map[string]interface{} `json:"args"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.Tool == "" {
			return response.BadRequest(c, "Tool name is required")
		}

		// Route to appropriate tool handler
		switch body.Tool {
		case "list_groups":
			return handleListGroups(c, db)
		case "get_group":
			return handleGetGroup(c, db, body.Args)
		case "list_apis_by_group":
			return handleListAPIsByGroup(c, db, body.Args)
		case "get_api":
			return handleGetAPI(c, db, body.Args)
		case "search_apis":
			return handleSearchAPIs(c, db, body.Args)
		case "get_api_json_example":
			return handleGetAPIJSONExample(c, db, body.Args)
		default:
			return response.BadRequest(c, "Unknown tool: "+body.Tool)
		}
	}
}

// handleListGroups lists all API groups
func handleListGroups(c *fiber.Ctx, db *gorm.DB) error {
	var groups []models.Group
	if err := db.Order("created_at DESC").Find(&groups).Error; err != nil {
		return response.InternalError(c, "Failed to fetch groups")
	}

	data := make([]map[string]interface{}, len(groups))
	for i, g := range groups {
		data[i] = map[string]interface{}{
			"id":        g.ID,
			"name":      g.Name,
			"createdAt": g.CreatedAt,
		}
	}

	return c.JSON(fiber.Map{"data": data})
}

// handleGetGroup gets group details with API count
func handleGetGroup(c *fiber.Ctx, db *gorm.DB, args map[string]interface{}) error {
	groupName, ok := args["groupName"].(string)
	if !ok || groupName == "" {
		return response.BadRequest(c, "groupName is required")
	}

	var groups []models.Group
	if err := db.Where("name LIKE ?", "%"+groupName+"%").Preload("APIs").Find(&groups).Error; err != nil {
		return response.InternalError(c, "Failed to fetch groups")
	}

	if len(groups) == 0 {
		return response.NotFound(c, "No group found matching: "+groupName)
	}

	group := groups[0]
	return c.JSON(fiber.Map{
		"data": map[string]interface{}{
			"id":        group.ID,
			"name":      group.Name,
			"apiCount":  len(group.APIs),
			"createdAt": group.CreatedAt,
		},
	})
}

// handleListAPIsByGroup lists all APIs in a group
func handleListAPIsByGroup(c *fiber.Ctx, db *gorm.DB, args map[string]interface{}) error {
	groupName, ok := args["groupName"].(string)
	if !ok || groupName == "" {
		return response.BadRequest(c, "groupName is required")
	}

	var groups []models.Group
	if err := db.Where("name LIKE ?", "%"+groupName+"%").Preload("APIs", func(db *gorm.DB) *gorm.DB {
		return db.Order("`order` ASC")
	}).Find(&groups).Error; err != nil {
		return response.InternalError(c, "Failed to fetch groups")
	}

	if len(groups) == 0 {
		return response.NotFound(c, "No group found matching: "+groupName)
	}

	group := groups[0]
	apis := make([]map[string]interface{}, len(group.APIs))
	for i, api := range group.APIs {
		apis[i] = map[string]interface{}{
			"id":       api.ID,
			"name":     api.Name,
			"endpoint": api.Endpoint,
			"method":   api.Method,
			"type":     api.Type,
		}
	}

	return c.JSON(fiber.Map{
		"data": map[string]interface{}{
			"group": map[string]interface{}{
				"id":   group.ID,
				"name": group.Name,
			},
			"apis": apis,
		},
	})
}

// handleGetAPI gets full API details with parameters
func handleGetAPI(c *fiber.Ctx, db *gorm.DB, args map[string]interface{}) error {
	apiID, ok := args["apiId"].(float64)
	if !ok {
		return response.BadRequest(c, "apiId (number) is required")
	}

	var api models.API
	if err := db.Preload("Group").Preload("Parameters", func(db *gorm.DB) *gorm.DB {
		return db.Order("`order` ASC")
	}).First(&api, uint(apiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(c, "API not found")
		}
		return response.InternalError(c, "Failed to fetch API")
	}

	// Separate request and response parameters
	requestParams := make([]models.Parameter, 0)
	responseParams := make([]models.Parameter, 0)

	for _, p := range api.Parameters {
		if p.ParamType == "request" {
			requestParams = append(requestParams, p)
		} else if p.ParamType == "response" {
			responseParams = append(responseParams, p)
		}
	}

	// Build parameter trees
	requestTree := services.BuildParameterTree(requestParams)
	responseTree := services.BuildParameterTree(responseParams)

	return c.JSON(fiber.Map{
		"data": map[string]interface{}{
			"id":                 api.ID,
			"name":               api.Name,
			"endpoint":           api.Endpoint,
			"method":             api.Method,
			"type":               api.Type,
			"note":               api.Note,
			"group":              map[string]interface{}{"id": api.Group.ID, "name": api.Group.Name},
			"requestParameters":  requestTree,
			"responseParameters": responseTree,
		},
	})
}

// handleSearchAPIs searches APIs by name or endpoint
func handleSearchAPIs(c *fiber.Ctx, db *gorm.DB, args map[string]interface{}) error {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return response.BadRequest(c, "query is required")
	}

	var apis []models.API
	searchPattern := "%" + strings.ToLower(query) + "%"
	if err := db.Where("LOWER(name) LIKE ? OR LOWER(endpoint) LIKE ?", searchPattern, searchPattern).
		Preload("Group").
		Limit(50).
		Find(&apis).Error; err != nil {
		return response.InternalError(c, "Failed to search APIs")
	}

	results := make([]map[string]interface{}, len(apis))
	for i, api := range apis {
		results[i] = map[string]interface{}{
			"id":       api.ID,
			"name":     api.Name,
			"endpoint": api.Endpoint,
			"method":   api.Method,
			"type":     api.Type,
			"group": map[string]interface{}{
				"id":   api.Group.ID,
				"name": api.Group.Name,
			},
		}
	}

	return c.JSON(fiber.Map{
		"data": map[string]interface{}{
			"count": len(results),
			"apis":  results,
		},
	})
}

// handleGetAPIJSONExample generates example JSON for API
func handleGetAPIJSONExample(c *fiber.Ctx, db *gorm.DB, args map[string]interface{}) error {
	apiID, ok := args["apiId"].(float64)
	if !ok {
		return response.BadRequest(c, "apiId (number) is required")
	}

	var api models.API
	if err := db.Preload("Parameters", func(db *gorm.DB) *gorm.DB {
		return db.Order("`order` ASC")
	}).First(&api, uint(apiID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(c, "API not found")
		}
		return response.InternalError(c, "Failed to fetch API")
	}

	// Separate request and response parameters
	requestParams := make([]models.Parameter, 0)
	responseParams := make([]models.Parameter, 0)

	for _, p := range api.Parameters {
		if p.ParamType == "request" {
			requestParams = append(requestParams, p)
		} else if p.ParamType == "response" {
			responseParams = append(responseParams, p)
		}
	}

	// Build trees
	requestTree := services.BuildParameterTree(requestParams)
	responseTree := services.BuildParameterTree(responseParams)

	// Generate example JSON
	var requestExample, responseExample interface{}
	if len(requestTree) > 0 {
		requestExample = services.GenerateExampleJSON(requestTree)
	}
	if len(responseTree) > 0 {
		responseExample = services.GenerateExampleJSON(responseTree)
	}

	return c.JSON(fiber.Map{
		"data": map[string]interface{}{
			"apiName":         api.Name,
			"endpoint":        api.Endpoint,
			"method":          api.Method,
			"requestExample":  requestExample,
			"responseExample": responseExample,
		},
	})
}
