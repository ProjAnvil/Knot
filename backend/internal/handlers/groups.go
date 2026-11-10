package handlers

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/ProjAnvil/knot/backend/pkg/response"
	models "github.com/ProjAnvil/knot/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// uniqueViolationPattern matches duplicate-key errors across every supported
// database dialect:
//   - SQLite:     "UNIQUE constraint failed: groups.name"
//   - MySQL:      "Duplicate entry 'x' for key 'groups.name'"
//   - PostgreSQL: "... SQLSTATE 23505 (unique_violation)"
//
// The tokens are intentionally broad so a driver wording change cannot break
// the race backstop; do not prune them.
var uniqueViolationPattern = regexp.MustCompile(`UNIQUE|Duplicate|23505`)

// isUniqueViolation reports whether err looks like a duplicate-key violation.
func isUniqueViolation(err error) bool {
	return err != nil && uniqueViolationPattern.MatchString(err.Error())
}

// duplicateGroupName sends the shared 409 response for name conflicts.
func duplicateGroupName(c *fiber.Ctx) error {
	return response.Conflict(c, "Group name already exists")
}

// GetGroups returns all groups
func GetGroups(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var groups []models.Group
		// Use clause.OrderByColumn so GORM quotes the reserved `order`
		// column per dialect (backticks break on PostgreSQL).
		result := db.Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).Find(&groups)
		if result.Error != nil {
			return response.InternalError(c, "Failed to fetch groups")
		}

		// Ensure groups is never nil
		if groups == nil {
			groups = []models.Group{}
		}

		return response.Success(c, groups)
	}
}

// GetGroupsWithAPIs returns all groups with their APIs
func GetGroupsWithAPIs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var groups []models.Group
		result := db.Preload("APIs", func(db *gorm.DB) *gorm.DB {
			return db.Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}})
		}).Order(clause.OrderByColumn{Column: clause.Column{Name: "order"}}).Find(&groups)

		if result.Error != nil {
			return response.InternalError(c, "Failed to fetch groups with APIs")
		}

		// Ensure groups is never nil
		if groups == nil {
			groups = []models.Group{}
		}

		// Ensure each group's APIs slice is never nil
		for i := range groups {
			if groups[i].APIs == nil {
				groups[i].APIs = []models.API{}
			}
		}

		return response.Success(c, groups)
	}
}

// CreateGroup creates a new group
func CreateGroup(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.Name == "" {
			return response.BadRequest(c, "Group name is required")
		}

		// Pre-check the name so clients get a friendly 409 instead of a 500.
		var existing models.Group
		if err := db.Where("name = ?", body.Name).First(&existing).Error; err == nil {
			return duplicateGroupName(c)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return response.InternalError(c, "Failed to create group")
		}

		// Get max order
		var maxOrder int
		db.Model(&models.Group{}).Select("COALESCE(MAX(?), 0)", clause.Column{Name: "order"}).Scan(&maxOrder)

		group := models.Group{
			Name:  body.Name,
			Order: maxOrder + 1,
		}

		if err := db.Create(&group).Error; err != nil {
			// Backstop for the race where a duplicate name slips past the
			// pre-check and the database rejects the insert.
			if isUniqueViolation(err) {
				return duplicateGroupName(c)
			}
			return response.InternalError(c, "Failed to create group")
		}

		return response.Success(c, group)
	}
}

// UpdateGroup updates a group name
func UpdateGroup(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid group ID")
		}

		var body struct {
			Name string `json:"name"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		if body.Name == "" {
			return response.BadRequest(c, "Group name is required")
		}

		var group models.Group
		if err := db.First(&group, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return response.NotFound(c, "Group not found")
			}
			return response.InternalError(c, "Failed to fetch group")
		}

		// Only check for conflicts when the name actually changes; renaming a
		// group to its own current name must succeed.
		if body.Name != group.Name {
			var existing models.Group
			if err := db.Where("name = ?", body.Name).First(&existing).Error; err == nil {
				if existing.ID != group.ID {
					return duplicateGroupName(c)
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return response.InternalError(c, "Failed to fetch group")
			}
		}

		group.Name = body.Name
		if err := db.Save(&group).Error; err != nil {
			// Backstop for the race where a conflicting rename slips past the
			// pre-check and the database rejects the save.
			if isUniqueViolation(err) {
				return duplicateGroupName(c)
			}
			return response.InternalError(c, "Failed to update group")
		}

		return response.Success(c, group)
	}
}

// UpdateGroupOrders updates the order of multiple groups
func UpdateGroupOrders(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			GroupOrders []struct {
				ID    uint `json:"id"`
				Order int  `json:"order"`
			} `json:"groupOrders"`
		}

		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "Invalid request body")
		}

		// Update each group's order in a transaction
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, item := range body.GroupOrders {
				if err := tx.Model(&models.Group{}).Where("id = ?", item.ID).Update("order", item.Order).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return response.InternalError(c, "Failed to update group orders")
		}

		return response.Success(c, nil)
	}
}

// DeleteGroup deletes a group and all its APIs
func DeleteGroup(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 32)
		if err != nil {
			return response.BadRequest(c, "Invalid group ID")
		}

		// Delete the group (APIs will be cascade deleted via foreign key constraint)
		result := db.Delete(&models.Group{}, id)
		if result.Error != nil {
			return response.InternalError(c, "Failed to delete group")
		}

		if result.RowsAffected == 0 {
			return response.NotFound(c, "Group not found")
		}

		return response.Success(c, nil)
	}
}
