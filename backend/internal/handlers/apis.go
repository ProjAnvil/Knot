package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetAPI returns a single API with parameters
func GetAPI(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		var api models.API
		result := db.Preload("Group").
			Preload("Parameters", func(db *gorm.DB) *gorm.DB {
				return db.Order("`order` ASC")
			}).
			First(&api, id)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return response.NotFound(c, "API not found")
			}
			return response.InternalError(c, "Failed to fetch API")
		}

		return response.Success(c, api)
	}
}

// GetAPIsByGroup returns all APIs in a group
func GetAPIsByGroup(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		groupID, err := strconv.ParseUint(c.Params("groupId"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid group ID")
		}

		var apis []models.API
		result := db.Where("group_id = ?", groupID).
			Preload("Parameters", func(db *gorm.DB) *gorm.DB {
				return db.Order("`order` ASC")
			}).
			Order("`order` ASC").
			Find(&apis)

		if result.Error != nil {
			return response.InternalError(c, "Failed to fetch APIs")
		}

		return response.Success(c, apis)
	}
}

// CreateAPI creates a new API
func CreateAPI(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			GroupID  uint    `json:"groupId"`
			Name     string  `json:"name"`
			Endpoint string  `json:"endpoint"`
			Method   string  `json:"method"`
			Type     string  `json:"type"`
			Note     *string `json:"note"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.GroupID == 0 || body.Name == "" || body.Endpoint == "" || body.Type == "" {
			return response.BadRequest(c, "Missing required fields")
		}

		if body.Type == "HTTP" && body.Method == "" {
			return response.BadRequest(c, "Method is required for HTTP APIs")
		}

		// Get max order for this group
		var maxOrder int
		db.Model(&models.API{}).Where("group_id = ?", body.GroupID).Select("COALESCE(MAX(`order`), 0)").Scan(&maxOrder)

		api := models.API{
			GroupID:  body.GroupID,
			Name:     body.Name,
			Endpoint: body.Endpoint,
			Method:   body.Method,
			Type:     body.Type,
			Note:     body.Note,
			Order:    maxOrder + 1,
		}

		if err := db.Create(&api).Error; err != nil {
			return response.InternalError(c, "Failed to create API")
		}

		return response.Success(c, api)
	}
}

// UpdateAPI updates API basic info
func UpdateAPI(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		var body struct {
			Name     *string `json:"name"`
			Endpoint *string `json:"endpoint"`
			Method   *string `json:"method"`
			Type     *string `json:"type"`
			Note     *string `json:"note"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		var api models.API
		if err := db.First(&api, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return response.NotFound(c, "API not found")
			}
			return response.InternalError(c, "Failed to fetch API")
		}

		if body.Name != nil {
			api.Name = *body.Name
		}
		if body.Endpoint != nil {
			api.Endpoint = *body.Endpoint
		}
		if body.Method != nil {
			api.Method = *body.Method
		}
		if body.Type != nil {
			api.Type = *body.Type
		}
		if body.Note != nil {
			api.Note = body.Note
		}

		if err := db.Save(&api).Error; err != nil {
			return response.InternalError(c, "Failed to update API")
		}

		return response.Success(c, api)
	}
}

// UpdateAPINote updates API note
func UpdateAPINote(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		var body struct {
			Note *string `json:"note"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		var api models.API
		if err := db.First(&api, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return response.NotFound(c, "API not found")
			}
			return response.InternalError(c, "Failed to fetch API")
		}

		api.Note = body.Note
		if err := db.Save(&api).Error; err != nil {
			return response.InternalError(c, "Failed to update API note")
		}

		return response.Success(c, api)
	}
}

// UpdateAPIOrders updates the order of multiple APIs
func UpdateAPIOrders(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			APIOrders []struct {
				ID    uint `json:"id"`
				Order int  `json:"order"`
			} `json:"apiOrders"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		// Update each API's order in a transaction
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, item := range body.APIOrders {
				if err := tx.Model(&models.API{}).Where("id = ?", item.ID).Update("order", item.Order).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return response.InternalError(c, "Failed to update API orders")
		}

		return response.Success(c, nil)
	}
}

// DeleteAPI deletes an API and all its parameters
func DeleteAPI(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		// Delete the API (parameters will be cascade deleted via foreign key constraint)
		result := db.Delete(&models.API{}, id)
		if result.Error != nil {
			return response.InternalError(c, "Failed to delete API")
		}

		if result.RowsAffected == 0 {
			return response.NotFound(c, "API not found")
		}

		return response.Success(c, nil)
	}
}

// UpdateParameters updates parameters from structure
func UpdateParameters(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		var body struct {
			ParamType  string          `json:"paramType"`
			Parameters json.RawMessage `json:"parameters"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.ParamType != "request" && body.ParamType != "response" {
			return response.BadRequest(c, "Invalid paramType")
		}

		// Parse parameters
		var params []map[string]interface{}
		if err := json.Unmarshal(body.Parameters, &params); err != nil {
			return response.BadRequest(c, "Invalid parameters format")
		}

		// Delete existing parameters of this type
		if err := db.Where("api_id = ? AND param_type = ?", id, body.ParamType).Delete(&models.Parameter{}).Error; err != nil {
			return response.InternalError(c, "Failed to delete existing parameters")
		}

		if len(params) == 0 {
			return response.Success(c, fiber.Map{"count": 0})
		}

		// Insert parameters recursively
		order := 0
		insertedCount := 0

		var insertParams func(params []map[string]interface{}, parentID *uint) error
		insertParams = func(params []map[string]interface{}, parentID *uint) error {
			for _, param := range params {
				name, _ := param["name"].(string)
				paramType, _ := param["type"].(string)
				description, _ := param["description"].(string)
				required, _ := param["required"].(bool)

				p := models.Parameter{
					APIID:     uint(id),
					ParentID:  parentID,
					Name:      name,
					Type:      paramType,
					Required:  required,
					ParamType: body.ParamType,
					Order:     order,
				}

				if description != "" {
					p.Description = &description
				}

				if err := db.Create(&p).Error; err != nil {
					return err
				}

				order++
				insertedCount++

				// Handle children
				if children, ok := param["children"].([]interface{}); ok && len(children) > 0 {
					childParams := make([]map[string]interface{}, 0, len(children))
					for _, child := range children {
						if childMap, ok := child.(map[string]interface{}); ok {
							childParams = append(childParams, childMap)
						}
					}
					if len(childParams) > 0 {
						if err := insertParams(childParams, &p.ID); err != nil {
							return err
						}
					}
				}
			}
			return nil
		}

		if err := insertParams(params, nil); err != nil {
			return response.InternalError(c, "Failed to insert parameters")
		}

		return response.Success(c, fiber.Map{"count": insertedCount})
	}
}

// UpdateParametersFromJSON updates parameters from JSON
func UpdateParametersFromJSON(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid API ID")
		}

		var body struct {
			ParamType string                 `json:"paramType"`
			JSON      map[string]interface{} `json:"json"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.ParamType != "request" && body.ParamType != "response" {
			return response.BadRequest(c, "Invalid paramType. Must be 'request' or 'response'")
		}

		if body.JSON == nil {
			return response.BadRequest(c, "Invalid json object")
		}

		// Get existing parameters to preserve required status and descriptions
		var existingParams []models.Parameter
		db.Where("api_id = ? AND param_type = ?", id, body.ParamType).Find(&existingParams)

		// Build a map of existing parameters by name
		existingMap := make(map[string]*models.Parameter)
		for i := range existingParams {
			existingMap[existingParams[i].Name] = &existingParams[i]
		}

		// Delete existing parameters
		db.Where("api_id = ? AND param_type = ?", id, body.ParamType).Delete(&models.Parameter{})

		// Convert JSON to parameters
		orderCounter := 0
		var convertJSON func(obj map[string]interface{}, parentID *uint) error
		convertJSON = func(obj map[string]interface{}, parentID *uint) error {
			for key, value := range obj {
				existing := existingMap[key]

				var paramType string
				var children map[string]interface{}

				switch v := value.(type) {
				case []interface{}:
					paramType = "array"
					if len(v) > 0 {
						if obj, ok := v[0].(map[string]interface{}); ok {
							children = obj
						}
					}
				case map[string]interface{}:
					paramType = "object"
					children = v
				case string:
					paramType = "string"
				case float64:
					paramType = "number"
				case bool:
					paramType = "boolean"
				default:
					paramType = "string"
				}

				param := models.Parameter{
					APIID:     uint(id),
					ParentID:  parentID,
					Name:      key,
					Type:      paramType,
					ParamType: body.ParamType,
					Order:     orderCounter,
					Required:  false,
				}

				if existing != nil {
					param.Required = existing.Required
					param.Description = existing.Description
				}

				if err := db.Create(&param).Error; err != nil {
					return err
				}

				orderCounter++

				if children != nil {
					if err := convertJSON(children, &param.ID); err != nil {
						return err
					}
				}
			}
			return nil
		}

		if err := convertJSON(body.JSON, nil); err != nil {
			return response.InternalError(c, "Failed to convert JSON to parameters")
		}

		return response.Success(c, fiber.Map{"parameterCount": len(body.JSON)})
	}
}
