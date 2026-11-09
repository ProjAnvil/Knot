package models

// WireFormat is the format identifier written into every Knot JSON export
// document ("knot"). Consumers use it to recognize Knot files; the export
// builder and the format detector must agree on this value.
const WireFormat = "knot"

// WireVersion is the version of the Knot JSON export envelope currently
// produced and accepted for import.
const WireVersion = 1

// KnotExportDocument is the top-level structure of a Knot JSON v1 export file.
type KnotExportDocument struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	ExportedAt string            `json:"exportedAt"`
	Groups     []KnotExportGroup `json:"groups"`
}

// KnotExportGroup is one API group within a Knot export document.
type KnotExportGroup struct {
	Name  string          `json:"name"`
	Order int             `json:"order"`
	APIs  []KnotExportAPI `json:"apis"`
}

// KnotExportAPI is one API entry within a group.
type KnotExportAPI struct {
	Name       string           `json:"name"`
	Endpoint   string           `json:"endpoint"`
	Method     string           `json:"method"`
	Type       string           `json:"type"`
	Order      int              `json:"order"`
	Note       *string          `json:"note"`
	Parameters KnotExportParams `json:"parameters"`
}

// KnotExportParams holds the request and response parameter trees of one API.
type KnotExportParams struct {
	Request  []ParamNode `json:"request"`
	Response []ParamNode `json:"response"`
}
