package handlers

import (
	"strconv"
	"strings"

	"github.com/ProjAnvil/knot/backend/pkg/response"
	models "github.com/ProjAnvil/knot/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListAPIs pagination defaults and limits.
const (
	defaultListPage     = 1
	defaultListPageSize = 50
	maxListPageSize     = 100
)

// listAPIsData is the GET /apis response payload: one page of the flat API
// listing plus the pagination metadata clients need to render a pager.
type listAPIsData struct {
	Items    []models.API `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// parseListPagination parses the page/pageSize query parameters. Empty values
// fall back to the defaults; non-numeric values are an error; numeric values
// are clamped (page >= 1, pageSize 1..100) so a hostile or stale client
// cannot request unbounded pages.
func parseListPagination(pageRaw, pageSizeRaw string) (page, pageSize int, err error) {
	page, pageSize = defaultListPage, defaultListPageSize

	if pageRaw != "" {
		if page, err = strconv.Atoi(pageRaw); err != nil {
			return 0, 0, err
		}
	}
	if pageSizeRaw != "" {
		if pageSize, err = strconv.Atoi(pageSizeRaw); err != nil {
			return 0, 0, err
		}
	}

	if page < 1 {
		page = 1
	}
	switch {
	case pageSize < 1:
		pageSize = 1
	case pageSize > maxListPageSize:
		pageSize = maxListPageSize
	}
	return page, pageSize, nil
}

// ListAPIs returns a flat, paginated, searchable listing of every API
// ordered by its parent group's position first (group order, then API order,
// then id for a stable total ordering).
//
// Supported query parameters:
//
//	page     default 1, clamped >= 1 (non-numeric -> 400)
//	pageSize default 50, clamped 1..100 (non-numeric -> 400)
//	keyword  case-insensitive substring match on name OR endpoint
//	groupId  exact group filter (non-numeric -> 400)
//	method   exact HTTP method filter, input is upper-cased first
func ListAPIs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize, err := parseListPagination(c.Query("page"), c.Query("pageSize"))
		if err != nil {
			return response.BadRequest(c, "Invalid pagination parameters")
		}

		// Resolve table names through the models so hosts using a table
		// prefix (e.g. "knot_") get working SQL; a raw "groups" string would
		// break the JOIN on prefixed deployments.
		groupsTable := (&models.Group{}).TableName()
		apisTable := (&models.API{}).TableName()

		// Inner-join the parent group purely to order by its position; the
		// join cannot fan out rows because it targets the group primary key.
		// clause.Table args are rendered as dialect-quoted identifiers.
		query := db.Model(&models.API{}).
			Joins("JOIN ? ON ?.id = ?.group_id",
				clause.Table{Name: groupsTable},
				clause.Table{Name: groupsTable},
				clause.Table{Name: apisTable})

		if raw := c.Query("groupId"); raw != "" {
			groupID, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return response.BadRequest(c, "Invalid group ID")
			}
			query = query.Where(clause.Eq{
				Column: clause.Column{Table: apisTable, Name: "group_id"},
				Value:  groupID,
			})
		}

		if method := c.Query("method"); method != "" {
			query = query.Where(clause.Eq{
				Column: clause.Column{Table: apisTable, Name: "method"},
				Value:  strings.ToUpper(method),
			})
		}

		if keyword := c.Query("keyword"); keyword != "" {
			// The pattern is lowercased in Go so both sides of the
			// comparison share one case. LIKE wildcards inside the keyword
			// are intentionally NOT escaped, mirroring the MCP search tool
			// so both search surfaces behave identically; the keyword is
			// still bound as a SQL parameter, so injection is impossible.
			// Columns are qualified through clause.Column because the JOIN
			// above makes a bare "name" ambiguous between the two tables.
			pattern := "%" + strings.ToLower(keyword) + "%"
			query = query.Where(
				"LOWER(?) LIKE ? OR LOWER(?) LIKE ?",
				clause.Column{Table: apisTable, Name: "name"}, pattern,
				clause.Column{Table: apisTable, Name: "endpoint"}, pattern,
			)
		}

		// Session-scoped clones let Count and Find share the filters without
		// polluting each other's SELECT/ORDER BY/LIMIT clauses.
		var total int64
		if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			return response.InternalError(c, "Failed to count APIs")
		}

		var apis []models.API
		if err := query.Session(&gorm.Session{}).
			Preload("Group").
			// OrderByColumn quotes the reserved word "order" per dialect and
			// keeps the table references prefix-aware.
			Order(clause.OrderByColumn{Column: clause.Column{Table: groupsTable, Name: "order"}}).
			Order(clause.OrderByColumn{Column: clause.Column{Table: apisTable, Name: "order"}}).
			Order(clause.OrderByColumn{Column: clause.Column{Table: apisTable, Name: "id"}}).
			Limit(pageSize).
			Offset((page - 1) * pageSize).
			Find(&apis).Error; err != nil {
			return response.InternalError(c, "Failed to fetch APIs")
		}

		// An empty page must serialize as [], not null.
		if apis == nil {
			apis = []models.API{}
		}

		return response.Success(c, listAPIsData{
			Items:    apis,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}
