package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ProjAnvil/knot/backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupImportApplyTestDB creates an in-memory SQLite database with the full
// Group/API/Parameter schema that ApplyImport writes to. The shared
// setupTestDB helper only migrates models.Parameter, so import apply tests
// need this local variant.
func setupImportApplyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Group{}, &models.API{}, &models.Parameter{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
	return db
}

// applyTestPlan builds a plan with two groups covering nested request and
// response parameter trees, an API without parameters, and a note pointer.
func applyTestPlan() *ImportPlan {
	return &ImportPlan{
		Groups: []ImportGroup{
			{
				Name: "Auth",
				APIs: []ImportAPI{
					{
						Name:     "Login",
						Endpoint: "/auth/login",
						Method:   "POST",
						Type:     "HTTP",
						Note:     strPtr("Login with credentials"),
						Request: []ParamNode{
							{
								Name: "user", Type: "object", Required: true,
								Children: []ParamNode{
									{Name: "name", Type: "string", Required: true},
									{Name: "age", Type: "number"},
								},
							},
							{Name: "remember", Type: "boolean"},
						},
						Response: []ParamNode{
							{Name: "token", Type: "string"},
							{Name: "expiresIn", Type: "number"},
						},
					},
					{
						Name:     "Logout",
						Endpoint: "/auth/logout",
						Method:   "POST",
						Type:     "HTTP",
					},
				},
			},
			{
				Name: "Billing",
				APIs: []ImportAPI{
					{
						Name:     "ListInvoices",
						Endpoint: "/billing/invoices",
						Method:   "GET",
						Type:     "HTTP",
						Response: []ParamNode{
							{
								Name: "items", Type: "array",
								Children: []ParamNode{
									{Name: "id", Type: "string"},
									{Name: "total", Type: "number"},
								},
							},
						},
					},
				},
			},
		},
		Warnings: []string{"sample parse warning"},
	}
}

func TestApplyImport(t *testing.T) {
	t.Run("givenFreshDB_whenApplyImport_thenGroupsApisAndNestedParamsCreated", func(t *testing.T) {
		// Arrange
		db := setupImportApplyTestDB(t)

		// Act
		result, err := ApplyImport(db, ImportFormatKnot, applyTestPlan(), false)

		// Assert: exact counts and warnings surfaced from the plan.
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result.Format != "knot-json" {
			t.Errorf("Expected format %q, got %q", "knot-json", result.Format)
		}
		if result.GroupsCreated != 2 {
			t.Errorf("Expected 2 groups created, got %d", result.GroupsCreated)
		}
		if result.GroupsMerged != 0 {
			t.Errorf("Expected 0 groups merged, got %d", result.GroupsMerged)
		}
		if result.APIsImported != 3 {
			t.Errorf("Expected 3 APIs imported, got %d", result.APIsImported)
		}
		if result.ParametersImported != 9 {
			t.Errorf("Expected 9 parameters imported, got %d", result.ParametersImported)
		}
		if !reflect.DeepEqual(result.Warnings, []string{"sample parse warning"}) {
			t.Errorf("Expected plan warnings surfaced, got: %v", result.Warnings)
		}

		// Assert: groups created with sequential orders starting at 0.
		var groups []models.Group
		if err := db.Order("id ASC").Find(&groups).Error; err != nil {
			t.Fatalf("Failed to load groups: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("Expected 2 groups in database, got %d", len(groups))
		}
		if groups[0].Name != "Auth" || groups[0].Order != 0 {
			t.Errorf("First group should be Auth with order 0, got %q order %d", groups[0].Name, groups[0].Order)
		}
		if groups[1].Name != "Billing" || groups[1].Order != 1 {
			t.Errorf("Second group should be Billing with order 1, got %q order %d", groups[1].Name, groups[1].Order)
		}

		// Assert: APIs persisted with per-group sequential orders.
		var authAPIs []models.API
		if err := db.Where("group_id = ?", groups[0].ID).Order("id ASC").Find(&authAPIs).Error; err != nil {
			t.Fatalf("Failed to load Auth APIs: %v", err)
		}
		if len(authAPIs) != 2 {
			t.Fatalf("Expected 2 APIs in Auth group, got %d", len(authAPIs))
		}
		login := authAPIs[0]
		if login.Name != "Login" || login.Order != 0 {
			t.Errorf("Login should have order 0, got %q order %d", login.Name, login.Order)
		}
		if login.Note == nil || *login.Note != "Login with credentials" {
			t.Errorf("Login note not preserved, got: %v", login.Note)
		}
		if authAPIs[1].Name != "Logout" || authAPIs[1].Order != 1 {
			t.Errorf("Logout should have order 1, got %q order %d", authAPIs[1].Name, authAPIs[1].Order)
		}
		if authAPIs[1].Note != nil {
			t.Errorf("Logout note should stay nil, got: %v", *authAPIs[1].Note)
		}

		var billingAPIs []models.API
		if err := db.Where("group_id = ?", groups[1].ID).Find(&billingAPIs).Error; err != nil {
			t.Fatalf("Failed to load Billing APIs: %v", err)
		}
		if len(billingAPIs) != 1 || billingAPIs[0].Name != "ListInvoices" || billingAPIs[0].Order != 0 {
			t.Fatalf("Expected ListInvoices with order 0 in Billing, got %+v", billingAPIs)
		}

		// Assert: request parameter tree flattened with ParentID chain and a
		// single per-API per-side order sequence starting at 0.
		var requestParams []models.Parameter
		if err := db.Where("api_id = ? AND param_type = ?", login.ID, "request").
			Order("id ASC").Find(&requestParams).Error; err != nil {
			t.Fatalf("Failed to load request params: %v", err)
		}
		if len(requestParams) != 4 {
			t.Fatalf("Expected 4 request params, got %d", len(requestParams))
		}
		byName := make(map[string]models.Parameter, len(requestParams))
		for _, p := range requestParams {
			byName[p.Name] = p
		}
		user := byName["user"]
		if user.ParentID != nil || user.Order != 0 {
			t.Errorf("user should be a root with order 0, got parent %v order %d", user.ParentID, user.Order)
		}
		for _, child := range []string{"name", "age"} {
			p := byName[child]
			if p.ParentID == nil || *p.ParentID != user.ID {
				t.Errorf("%s should reference user's ID %d, got %v", child, user.ID, p.ParentID)
			}
		}
		if byName["name"].Order != 1 || byName["age"].Order != 2 || byName["remember"].Order != 3 {
			t.Errorf("Request orders should be sequential depth-first, got name=%d age=%d remember=%d",
				byName["name"].Order, byName["age"].Order, byName["remember"].Order)
		}

		// Assert: response side has its own order sequence restarting at 0.
		var responseParams []models.Parameter
		if err := db.Where("api_id = ? AND param_type = ?", login.ID, "response").
			Order("id ASC").Find(&responseParams).Error; err != nil {
			t.Fatalf("Failed to load response params: %v", err)
		}
		if len(responseParams) != 2 {
			t.Fatalf("Expected 2 response params, got %d", len(responseParams))
		}
		if responseParams[0].Name != "token" || responseParams[0].Order != 0 {
			t.Errorf("token should have response order 0, got %q order %d", responseParams[0].Name, responseParams[0].Order)
		}
		if responseParams[1].Name != "expiresIn" || responseParams[1].Order != 1 {
			t.Errorf("expiresIn should have response order 1, got %q order %d", responseParams[1].Name, responseParams[1].Order)
		}

		// Assert: nested array parameter in Billing flattened correctly.
		var itemsParams []models.Parameter
		if err := db.Where("api_id = ? AND param_type = ?", billingAPIs[0].ID, "response").
			Order("id ASC").Find(&itemsParams).Error; err != nil {
			t.Fatalf("Failed to load Billing response params: %v", err)
		}
		if len(itemsParams) != 3 {
			t.Fatalf("Expected 3 response params for ListInvoices, got %d", len(itemsParams))
		}
		items := itemsParams[0]
		if items.ParentID != nil || items.Order != 0 {
			t.Errorf("items should be a root with order 0, got parent %v order %d", items.ParentID, items.Order)
		}
		if itemsParams[1].ParentID == nil || *itemsParams[1].ParentID != items.ID || itemsParams[1].Order != 1 {
			t.Errorf("id child of items incorrect: %+v", itemsParams[1])
		}
		if itemsParams[2].ParentID == nil || *itemsParams[2].ParentID != items.ID || itemsParams[2].Order != 2 {
			t.Errorf("total child of items incorrect: %+v", itemsParams[2])
		}
	})

	t.Run("givenExistingSameNameGroup_whenApplyImport_thenGroupMergedAndApisAppended", func(t *testing.T) {
		// Arrange: an existing Auth group with two APIs and an untouched
		// sibling group; the merged group's order must not change.
		db := setupImportApplyTestDB(t)
		auth := models.Group{Name: "Auth", Order: 7}
		if err := db.Create(&auth).Error; err != nil {
			t.Fatalf("Failed to seed Auth group: %v", err)
		}
		other := models.Group{Name: "Other", Order: 3}
		if err := db.Create(&other).Error; err != nil {
			t.Fatalf("Failed to seed Other group: %v", err)
		}
		for i, name := range []string{"Existing A", "Existing B"} {
			api := models.API{GroupID: auth.ID, Name: name, Endpoint: "/old", Method: "GET", Type: "HTTP", Order: i}
			if err := db.Create(&api).Error; err != nil {
				t.Fatalf("Failed to seed API: %v", err)
			}
		}

		plan := &ImportPlan{
			Groups: []ImportGroup{
				{
					Name: "Auth",
					APIs: []ImportAPI{
						{Name: "New X", Endpoint: "/new-x", Method: "POST", Type: "HTTP"},
						{Name: "New Y", Endpoint: "/new-y", Method: "GET", Type: "RPC",
							Request: []ParamNode{{Name: "p", Type: "string"}}},
					},
				},
			},
		}

		// Act
		result, err := ApplyImport(db, ImportFormatPostman, plan, false)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result.GroupsCreated != 0 || result.GroupsMerged != 1 {
			t.Errorf("Expected 0 created / 1 merged, got %d / %d", result.GroupsCreated, result.GroupsMerged)
		}
		if result.APIsImported != 2 || result.ParametersImported != 1 {
			t.Errorf("Expected 2 APIs / 1 param, got %d / %d", result.APIsImported, result.ParametersImported)
		}
		if result.Format != "postman" {
			t.Errorf("Expected format %q, got %q", "postman", result.Format)
		}

		var groups []models.Group
		if err := db.Find(&groups).Error; err != nil {
			t.Fatalf("Failed to load groups: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("Expected still 2 groups, got %d", len(groups))
		}
		var merged models.Group
		if err := db.Where("name = ?", "Auth").First(&merged).Error; err != nil {
			t.Fatalf("Failed to load merged group: %v", err)
		}
		if merged.Order != 7 {
			t.Errorf("Merged group order should stay 7, got %d", merged.Order)
		}

		// Assert: APIs appended after the group's current max order.
		var apis []models.API
		if err := db.Where("group_id = ?", merged.ID).Order("id ASC").Find(&apis).Error; err != nil {
			t.Fatalf("Failed to load Auth APIs: %v", err)
		}
		if len(apis) != 4 {
			t.Fatalf("Expected 4 APIs in merged group, got %d", len(apis))
		}
		expectedOrders := map[string]int{"Existing A": 0, "Existing B": 1, "New X": 2, "New Y": 3}
		for _, api := range apis {
			if api.Order != expectedOrders[api.Name] {
				t.Errorf("API %q should have order %d, got %d", api.Name, expectedOrders[api.Name], api.Order)
			}
		}
	})

	t.Run("givenExistingGroups_whenApplyImport_thenNewGroupOrderContinuesAfterMax", func(t *testing.T) {
		// Arrange: groups with a gap in orders; a new group must continue
		// after the current maximum, not after count.
		db := setupImportApplyTestDB(t)
		for _, g := range []models.Group{{Name: "A", Order: 0}, {Name: "B", Order: 5}} {
			if err := db.Create(&g).Error; err != nil {
				t.Fatalf("Failed to seed group: %v", err)
			}
		}

		plan := &ImportPlan{
			Groups: []ImportGroup{
				{Name: "C", APIs: []ImportAPI{{Name: "c1", Endpoint: "/c", Method: "GET", Type: "HTTP"}}},
			},
		}

		// Act
		result, err := ApplyImport(db, ImportFormatOpenAPI, plan, false)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result.GroupsCreated != 1 || result.GroupsMerged != 0 {
			t.Errorf("Expected 1 created / 0 merged, got %d / %d", result.GroupsCreated, result.GroupsMerged)
		}
		var created models.Group
		if err := db.Where("name = ?", "C").First(&created).Error; err != nil {
			t.Fatalf("Failed to load created group: %v", err)
		}
		if created.Order != 6 {
			t.Errorf("New group order should continue after max (6), got %d", created.Order)
		}
	})

	t.Run("givenDryRun_whenApplyImport_thenCountsMatchRealRunAndNothingWritten", func(t *testing.T) {
		// Arrange: two identically seeded databases; the plan exercises both
		// the merge and the create paths.
		seed := func(t *testing.T) *gorm.DB {
			db := setupImportApplyTestDB(t)
			auth := models.Group{Name: "Auth", Order: 2}
			if err := db.Create(&auth).Error; err != nil {
				t.Fatalf("Failed to seed Auth group: %v", err)
			}
			other := models.Group{Name: "Other", Order: 1}
			if err := db.Create(&other).Error; err != nil {
				t.Fatalf("Failed to seed Other group: %v", err)
			}
			api := models.API{GroupID: auth.ID, Name: "Existing", Endpoint: "/e", Method: "GET", Type: "HTTP", Order: 0}
			if err := db.Create(&api).Error; err != nil {
				t.Fatalf("Failed to seed API: %v", err)
			}
			param := models.Parameter{APIID: api.ID, Name: "seed", Type: "string", ParamType: "request", Order: 0}
			if err := db.Create(&param).Error; err != nil {
				t.Fatalf("Failed to seed param: %v", err)
			}
			return db
		}
		buildPlan := func() *ImportPlan {
			return &ImportPlan{
				Groups: []ImportGroup{
					{
						Name: "Auth", // merge path
						APIs: []ImportAPI{
							{Name: "Merged API", Endpoint: "/m", Method: "POST", Type: "HTTP",
								Request:  []ParamNode{{Name: "q", Type: "string"}},
								Response: []ParamNode{{Name: "r", Type: "number"}}},
						},
					},
					{
						Name: "Fresh", // create path
						APIs: []ImportAPI{
							{Name: "Fresh API", Endpoint: "/f", Method: "GET", Type: "HTTP"},
						},
					},
				},
				Warnings: []string{"dry-run comparison warning"},
			}
		}

		realDB := seed(t)
		dryDB := seed(t)

		// Act
		realResult, err := ApplyImport(realDB, ImportFormatOpenAPI, buildPlan(), false)
		if err != nil {
			t.Fatalf("Expected no error from real run, got: %v", err)
		}
		dryResult, err := ApplyImport(dryDB, ImportFormatOpenAPI, buildPlan(), true)
		if err != nil {
			t.Fatalf("Expected no error from dry run, got: %v", err)
		}

		// Assert: the dry run reports exactly the counts the real run produced.
		if !reflect.DeepEqual(dryResult, realResult) {
			t.Errorf("Dry-run result should match real run:\nreal: %+v\ndry:  %+v", realResult, dryResult)
		}
		if realResult.GroupsCreated != 1 || realResult.GroupsMerged != 1 ||
			realResult.APIsImported != 2 || realResult.ParametersImported != 2 {
			t.Fatalf("Unexpected real-run counts: %+v", realResult)
		}

		// Assert: the dry-run database is byte-for-byte unchanged.
		assertRowCounts(t, dryDB, 2, 1, 1)
		var authGroup models.Group
		if err := dryDB.Where("name = ?", "Auth").First(&authGroup).Error; err != nil {
			t.Fatalf("Failed to load Auth group: %v", err)
		}
		var authAPIs []models.API
		if err := dryDB.Where("group_id = ?", authGroup.ID).Find(&authAPIs).Error; err != nil {
			t.Fatalf("Failed to load Auth APIs: %v", err)
		}
		if len(authAPIs) != 1 || authAPIs[0].Name != "Existing" || authAPIs[0].Order != 0 {
			t.Errorf("Dry run must not append APIs, got: %+v", authAPIs)
		}

		// Sanity: the real run did write on its own database.
		assertRowCounts(t, realDB, 3, 3, 3)
	})

	t.Run("givenPlanWarnings_whenApplyImport_thenWarningsSurfacedInBothModes", func(t *testing.T) {
		// Arrange
		plan := &ImportPlan{
			Groups: []ImportGroup{
				{Name: "G", APIs: []ImportAPI{{Name: "a", Endpoint: "/a", Method: "GET", Type: "HTTP"}}},
			},
			Warnings: []string{"warning one", "warning two"},
		}

		// Act
		realResult, err := ApplyImport(setupImportApplyTestDB(t), ImportFormatKnot, plan, false)
		if err != nil {
			t.Fatalf("Expected no error from real run, got: %v", err)
		}
		dryResult, err := ApplyImport(setupImportApplyTestDB(t), ImportFormatKnot, plan, true)
		if err != nil {
			t.Fatalf("Expected no error from dry run, got: %v", err)
		}

		// Assert
		if !reflect.DeepEqual(realResult.Warnings, plan.Warnings) {
			t.Errorf("Real run should surface plan warnings, got: %v", realResult.Warnings)
		}
		if !reflect.DeepEqual(dryResult.Warnings, plan.Warnings) {
			t.Errorf("Dry run should surface plan warnings, got: %v", dryResult.Warnings)
		}
	})

	t.Run("givenFailureMidImport_whenApplyImport_thenRolledBackAndDbUnchanged", func(t *testing.T) {
		// Arrange: seed one group/API/param, then import a plan whose create
		// calls proceed in this order: group(1), API "First"(2), its param(3),
		// API "Second"(4). The callback fails on the 4th create so several
		// rows exist mid-import and only a rollback can remove them.
		db := setupImportApplyTestDB(t)
		existing := models.Group{Name: "Existing", Order: 0}
		if err := db.Create(&existing).Error; err != nil {
			t.Fatalf("Failed to seed group: %v", err)
		}
		keepAPI := models.API{GroupID: existing.ID, Name: "Keep", Endpoint: "/keep", Method: "GET", Type: "HTTP", Order: 0}
		if err := db.Create(&keepAPI).Error; err != nil {
			t.Fatalf("Failed to seed API: %v", err)
		}
		keepParam := models.Parameter{APIID: keepAPI.ID, Name: "keepParam", Type: "string", ParamType: "request", Order: 0}
		if err := db.Create(&keepParam).Error; err != nil {
			t.Fatalf("Failed to seed param: %v", err)
		}

		plan := &ImportPlan{
			Groups: []ImportGroup{
				{
					Name: "New",
					APIs: []ImportAPI{
						{Name: "First", Endpoint: "/first", Method: "GET", Type: "HTTP",
							Request: []ParamNode{{Name: "p", Type: "string"}}},
						{Name: "Second", Endpoint: "/second", Method: "GET", Type: "HTTP"},
					},
				},
			},
		}

		const callbackName = "test:inject_import_failure"
		injected := errors.New("injected create failure")
		createCalls := 0
		if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
			createCalls++
			if createCalls == 4 {
				_ = tx.AddError(injected)
			}
		}); err != nil {
			t.Fatalf("Failed to register failure callback: %v", err)
		}
		// Callbacks are global on the DB instance; always remove it again.
		defer db.Callback().Create().Remove(callbackName)

		// Act
		result, err := ApplyImport(db, ImportFormatKnot, plan, false)

		// Assert
		if err == nil {
			t.Fatal("Expected an error from the injected failure, got nil")
		}
		if !errors.Is(err, injected) {
			t.Errorf("Expected the injected error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result on failure, got: %+v", result)
		}
		assertRowCounts(t, db, 1, 1, 1)
		var newGroup models.Group
		if err := db.Where("name = ?", "New").First(&newGroup).Error; err == nil {
			t.Errorf("Rolled-back group New still exists: %+v", newGroup)
		}
	})
}

// assertRowCounts verifies the exact number of group, API, and parameter rows.
func assertRowCounts(t *testing.T, db *gorm.DB, groups, apis, params int) {
	t.Helper()
	var groupCount, apiCount, paramCount int64
	if err := db.Model(&models.Group{}).Count(&groupCount).Error; err != nil {
		t.Fatalf("Failed to count groups: %v", err)
	}
	if err := db.Model(&models.API{}).Count(&apiCount).Error; err != nil {
		t.Fatalf("Failed to count APIs: %v", err)
	}
	if err := db.Model(&models.Parameter{}).Count(&paramCount).Error; err != nil {
		t.Fatalf("Failed to count parameters: %v", err)
	}
	if groupCount != int64(groups) {
		t.Errorf("Expected %d groups, got %d", groups, groupCount)
	}
	if apiCount != int64(apis) {
		t.Errorf("Expected %d APIs, got %d", apis, apiCount)
	}
	if paramCount != int64(params) {
		t.Errorf("Expected %d parameters, got %d", params, paramCount)
	}
}

// TestApplyImport_InPlanDuplicateGroup covers a plan naming the same group
// twice: the second occurrence continues on the first's in-memory state,
// counts as a merge, and its APIs continue the group's order sequence.
func TestApplyImport_InPlanDuplicateGroup(t *testing.T) {
	// Arrange - one plan listing Auth twice with one API each
	db := setupImportApplyTestDB(t)
	plan := &ImportPlan{
		Groups: []ImportGroup{
			{
				Name: "Auth",
				APIs: []ImportAPI{{Name: "Login", Endpoint: "/login", Method: "POST", Type: "HTTP"}},
			},
			{
				Name: "Auth",
				APIs: []ImportAPI{{Name: "Logout", Endpoint: "/logout", Method: "POST", Type: "HTTP"}},
			},
		},
	}

	// Act
	result, err := ApplyImport(db, ImportFormatPostman, plan, false)

	// Assert - one created group plus one in-plan merge
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.GroupsCreated != 1 || result.GroupsMerged != 1 {
		t.Errorf("Expected 1 created / 1 merged, got %d / %d", result.GroupsCreated, result.GroupsMerged)
	}

	// Assert - a single group row holds both APIs with sequential orders
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		t.Fatalf("Failed to load groups: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Auth" {
		t.Fatalf("Expected a single Auth group, got %+v", groups)
	}
	var apis []models.API
	if err := db.Where("group_id = ?", groups[0].ID).Order("id ASC").Find(&apis).Error; err != nil {
		t.Fatalf("Failed to load APIs: %v", err)
	}
	if len(apis) != 2 {
		t.Fatalf("Expected 2 APIs in the merged group, got %+v", apis)
	}
	if apis[0].Name != "Login" || apis[0].Order != 0 {
		t.Errorf("Login should have order 0, got %q order %d", apis[0].Name, apis[0].Order)
	}
	if apis[1].Name != "Logout" || apis[1].Order != 1 {
		t.Errorf("Logout should have order 1, got %q order %d", apis[1].Name, apis[1].Order)
	}
}

// TestApplyImport_ErrorPropagation covers database failures: with the
// groups table missing, the very first order query fails and ApplyImport
// returns the error (and no result) instead of panicking.
func TestApplyImport_ErrorPropagation(t *testing.T) {
	// Arrange
	db := setupImportApplyTestDB(t)
	if err := db.Migrator().DropTable(&models.Group{}); err != nil {
		t.Fatalf("Failed to drop groups table: %v", err)
	}
	plan := &ImportPlan{
		Groups: []ImportGroup{{
			Name: "Auth",
			APIs: []ImportAPI{{Name: "Login", Endpoint: "/login", Method: "POST", Type: "HTTP"}},
		}},
	}

	// Act
	result, err := ApplyImport(db, ImportFormatPostman, plan, false)

	// Assert
	if err == nil {
		t.Fatal("Expected an error when the groups table is missing, got nil")
	}
	if result != nil {
		t.Errorf("Expected a nil result on error, got %+v", result)
	}
}
