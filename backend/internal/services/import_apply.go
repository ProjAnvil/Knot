package services

import (
	"errors"
	"fmt"

	models "github.com/ProjAnvil/knot/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ImportResult is relocated to the models module (models v1.1); this alias
// keeps every services/handler reference and test compiling unchanged while
// the canonical definition ships with the shared data model.
type ImportResult = models.ImportResult

// importGroupState tracks the running resolution of one group name within a
// single ApplyImport run. Keeping the state in memory (instead of re-querying
// the database) makes dry runs compute exactly the same continuation orders
// and counts as real runs, including when the same group name appears twice
// in one plan.
type importGroupState struct {
	groupID      uint
	nextAPIOrder int
}

// ApplyImport persists a models.ImportPlan into the database with merge semantics,
// per spec §8.4. The whole import runs inside one transaction: any failure
// rolls back every row written so far (all-or-nothing). For each plan group,
// a same-name group is merged into (APIs appended after that group's current
// max API order, the group's own order untouched); a missing group is created
// with an order continuing after the current max group order. Parameter
// trees are flattened to rows with ParentID links and a per-API, per-side
// sequential order starting at 0. There is no API-level deduplication:
// importing the same file twice duplicates APIs inside merged groups.
//
// With dryRun=true the exact same resolution logic runs but no row is
// written, and the returned counts are the ones a real run would produce.
// Plan warnings are surfaced in Result.Warnings in both modes.
func ApplyImport(db *gorm.DB, format models.ImportFormat, plan *models.ImportPlan, dryRun bool) (*ImportResult, error) {
	if plan == nil {
		return nil, errors.New("import plan is nil")
	}

	result := &ImportResult{
		Format:   string(format),
		Warnings: append([]string{}, plan.Warnings...),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return applyImportPlan(tx, plan, dryRun, result)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// applyImportPlan resolves and (unless dryRun) writes the whole plan inside
// an open transaction.
func applyImportPlan(tx *gorm.DB, plan *models.ImportPlan, dryRun bool, result *ImportResult) error {
	// The next order for a newly created group continues after the current
	// max group order across all groups. The COALESCE default of -1 makes an
	// empty table start at 0. The tracker is bumped in memory so a dry run
	// (which never inserts) yields the same orders as a real run, where each
	// insert raises the database max inside the transaction.
	maxGroupOrder, err := maxOrderOf(tx, &models.Group{}, "")
	if err != nil {
		return err
	}
	groupsByName := make(map[string]*importGroupState, len(plan.Groups))

	for i := range plan.Groups {
		imported := &plan.Groups[i]

		state, seen := groupsByName[imported.Name]
		switch {
		case seen:
			// The same name appeared earlier in this plan. In a real run the
			// database lookup below would find the row inserted by that
			// earlier occurrence, so this is a merge continuing on its state.
			result.GroupsMerged++
		default:
			var existing models.Group
			err := tx.Where("name = ?", imported.Name).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				// Missing group: create it with the next group order.
				maxGroupOrder++
				state = &importGroupState{nextAPIOrder: 0}
				if !dryRun {
					group := models.Group{Name: imported.Name, Order: maxGroupOrder}
					if err := tx.Create(&group).Error; err != nil {
						return fmt.Errorf("failed to create group %q: %w", imported.Name, err)
					}
					state.groupID = group.ID
				}
				result.GroupsCreated++
			case err != nil:
				return fmt.Errorf("failed to look up group %q: %w", imported.Name, err)
			default:
				// Existing group: merge. Imported API orders continue after
				// the group's current max API order; the group's own order
				// and rows are left untouched.
				maxAPIOrder, err := maxOrderOf(tx, &models.API{}, "group_id = ?", existing.ID)
				if err != nil {
					return err
				}
				state = &importGroupState{groupID: existing.ID, nextAPIOrder: maxAPIOrder + 1}
				result.GroupsMerged++
			}
			groupsByName[imported.Name] = state
		}

		for j := range imported.APIs {
			api := &imported.APIs[j]
			apiOrder := state.nextAPIOrder
			state.nextAPIOrder++

			if !dryRun {
				row := models.API{
					GroupID:  state.groupID,
					Name:     api.Name,
					Endpoint: api.Endpoint,
					Method:   api.Method,
					Type:     api.Type,
					Note:     api.Note,
					Order:    apiOrder,
				}
				if err := tx.Create(&row).Error; err != nil {
					return fmt.Errorf("failed to create API %q: %w", api.Name, err)
				}
				if err := insertParamTree(tx, row.ID, "request", api.Request); err != nil {
					return err
				}
				if err := insertParamTree(tx, row.ID, "response", api.Response); err != nil {
					return err
				}
			}
			result.APIsImported++
			result.ParametersImported += models.CountNodes(api.Request) + models.CountNodes(api.Response)
		}
	}
	return nil
}

// insertParamTree flattens a models.ParamNode tree into parameter rows inside the
// transaction. Each parent row is created first and its database-assigned ID
// becomes the children's ParentID. Order is a single per-API, per-side
// sequence starting at 0, assigned in depth-first traversal order (the same
// convention ParameterService.UpdateParameters follows).
func insertParamTree(tx *gorm.DB, apiID uint, paramType string, nodes []models.ParamNode) error {
	order := 0

	var insert func(nodes []models.ParamNode, parentID *uint) error
	insert = func(nodes []models.ParamNode, parentID *uint) error {
		for i := range nodes {
			node := &nodes[i]
			row := models.Parameter{
				APIID:       apiID,
				ParentID:    parentID,
				Name:        node.Name,
				Type:        node.Type,
				Description: node.Description,
				Required:    node.Required,
				ParamType:   paramType,
				Order:       order,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("failed to create parameter %q: %w", node.Name, err)
			}
			order++
			if len(node.Children) > 0 {
				if err := insert(node.Children, &row.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return insert(nodes, nil)
}

// maxOrderOf returns the maximum value of the reserved-word column "order"
// for the given model, or -1 when the table (or the filtered set) is empty,
// so callers can continue the sequence with max+1 starting at 0. The
// clause.Column placeholder keeps the query dialect-safe across
// SQLite/MySQL/PostgreSQL ("order" is quoted per the active dialector).
func maxOrderOf(tx *gorm.DB, model interface{}, where string, args ...interface{}) (int, error) {
	query := tx.Model(model)
	if where != "" {
		query = query.Where(where, args...)
	}
	var maxOrder int
	if err := query.Select("COALESCE(MAX(?), -1)", clause.Column{Name: "order"}).Scan(&maxOrder).Error; err != nil {
		return 0, fmt.Errorf("failed to query max order for %T: %w", model, err)
	}
	return maxOrder, nil
}
