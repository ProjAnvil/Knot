package handlers

import (
	"strconv"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/ProjAnvil/knot/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetGroups returns all groups
func GetGroups(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var groups []models.Group
		result := db.Order("`order` ASC").Find(&groups)
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
			return db.Order("`order` ASC")
		}).Order("`order` ASC").Find(&groups)

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

		// Get max order
		var maxOrder int
		db.Model(&models.Group{}).Select("COALESCE(MAX(`order`), 0)").Scan(&maxOrder)

		group := models.Group{
			Name:  body.Name,
			Order: maxOrder + 1,
		}

		if err := db.Create(&group).Error; err != nil {
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

		group.Name = body.Name
		if err := db.Save(&group).Error; err != nil {
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
