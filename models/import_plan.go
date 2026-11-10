package models

// ImportPlan is the shared intermediate representation that every importer
// (Knot JSON, Postman, OpenAPI) produces from an uploaded file. The import
// apply step consumes it; Warnings collects non-fatal issues encountered
// while parsing so they can be surfaced to the user after import.
type ImportPlan struct {
	Groups   []ImportGroup
	Warnings []string
}

// ImportGroup is one API group within an import plan.
type ImportGroup struct {
	Name string
	APIs []ImportAPI
}

// ImportAPI is one API entry within an import plan group. Note is a pointer
// so a missing note stays distinguishable from an empty one.
type ImportAPI struct {
	Name     string
	Endpoint string
	Method   string
	Type     string
	Note     *string
	Request  []ParamNode
	Response []ParamNode
}

// ImportResult reports what an applied import plan did (or, for a dry run,
// what it would do). It is the payload of the import preview/confirm flow.
type ImportResult struct {
	Format             string   `json:"format"`
	GroupsCreated      int      `json:"groupsCreated"`
	GroupsMerged       int      `json:"groupsMerged"`
	APIsImported       int      `json:"apisImported"`
	ParametersImported int      `json:"parametersImported"`
	Warnings           []string `json:"warnings"`
}

// ImportFormat identifies the exchange format of an uploaded import file.
// The string values are stable identifiers used by the import handler.
type ImportFormat string

const (
	ImportFormatKnot    ImportFormat = "knot-json"
	ImportFormatPostman ImportFormat = "postman"
	ImportFormatOpenAPI ImportFormat = "openapi"
)
