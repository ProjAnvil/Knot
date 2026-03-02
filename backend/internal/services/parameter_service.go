package services

import (
	"encoding/json"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"gorm.io/gorm"
)

// ParameterService handles parameter-related operations
type ParameterService struct {
	db *gorm.DB
}

// NewParameterService creates a new ParameterService
func NewParameterService(db *gorm.DB) *ParameterService {
	return &ParameterService{db: db}
}

// UpdateParameters updates parameters from structure for a specific API
func (s *ParameterService) UpdateParameters(apiID uint, paramType string, parameters json.RawMessage) (int, error) {
	// Parse parameters
	var params []map[string]interface{}
	if err := json.Unmarshal(parameters, &params); err != nil {
		return 0, err
	}

	// Delete existing parameters of this type
	if err := s.db.Where("api_id = ? AND param_type = ?", apiID, paramType).Delete(&models.Parameter{}).Error; err != nil {
		return 0, err
	}

	if len(params) == 0 {
		return 0, nil
	}

	// Insert parameters recursively
	order := 0
	insertedCount := 0

	var insertParams func(params []map[string]interface{}, parentID *uint) error
	insertParams = func(params []map[string]interface{}, parentID *uint) error {
		for _, param := range params {
			name, _ := param["name"].(string)
			paramTypeValue, _ := param["type"].(string)
			description, _ := param["description"].(string)
			required, _ := param["required"].(bool)

			p := models.Parameter{
				APIID:     apiID,
				ParentID:  parentID,
				Name:      name,
				Type:      paramTypeValue,
				Required:  required,
				ParamType: paramType,
				Order:     order,
			}

			if description != "" {
				p.Description = &description
			}

			if err := s.db.Create(&p).Error; err != nil {
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
		return 0, err
	}

	return insertedCount, nil
}

// UpdateParametersFromJSON updates parameters from JSON example for a specific API
func (s *ParameterService) UpdateParametersFromJSON(apiID uint, paramType string, jsonData map[string]interface{}) (int, error) {
	// Get existing parameters to preserve required status and descriptions
	var existingParams []models.Parameter
	s.db.Where("api_id = ? AND param_type = ?", apiID, paramType).Find(&existingParams)

	// Build a map of existing parameters by name
	existingMap := make(map[string]*models.Parameter)
	for i := range existingParams {
		existingMap[existingParams[i].Name] = &existingParams[i]
	}

	// Delete existing parameters
	s.db.Where("api_id = ? AND param_type = ?", apiID, paramType).Delete(&models.Parameter{})

	// Convert JSON to parameters
	orderCounter := 0
	var convertJSON func(obj map[string]interface{}, parentID *uint) error
	convertJSON = func(obj map[string]interface{}, parentID *uint) error {
		for key, value := range obj {
			existing := existingMap[key]

			var paramTypeValue string
			var children map[string]interface{}

			switch v := value.(type) {
			case []interface{}:
				paramTypeValue = "array"
				if len(v) > 0 {
					if obj, ok := v[0].(map[string]interface{}); ok {
						children = obj
					}
				}
			case map[string]interface{}:
				paramTypeValue = "object"
				children = v
			case string:
				paramTypeValue = "string"
			case float64:
				paramTypeValue = "number"
			case bool:
				paramTypeValue = "boolean"
			default:
				paramTypeValue = "string"
			}

			param := models.Parameter{
				APIID:     apiID,
				ParentID:  parentID,
				Name:      key,
				Type:      paramTypeValue,
				ParamType: paramType,
				Order:     orderCounter,
				Required:  false,
			}

			if existing != nil {
				param.Required = existing.Required
				param.Description = existing.Description
			}

			if err := s.db.Create(&param).Error; err != nil {
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

	if err := convertJSON(jsonData, nil); err != nil {
		return 0, err
	}

	return len(jsonData), nil
}
