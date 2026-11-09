// Command gen-schema generates the JSON Schema (draft 2020-12) for the Knot
// v1 export document by reflecting over models.KnotExportDocument.
//
// The golden output is committed at models/schema/knot-v1.schema.json so
// consumers can validate Knot files without a Go toolchain, and so CI can
// fail when the schema drifts from the Go types. Run from the repo root:
//
//	go run ./tools/cmd/gen-schema
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"

	"github.com/ProjAnvil/knot/models"
)

const (
	// schemaID is the canonical $id of the generated document. It doubles as
	// the public URL under which the schema will be published.
	schemaID = "https://github.com/ProjAnvil/knot/models/schema/knot-v1.schema.json"

	// outputPath is the golden file location, relative to the repo root.
	outputPath = "models/schema/knot-v1.schema.json"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-schema: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Anonymous hides the auto-generated per-type $id values so every
	// definition is referenced locally via #/$defs/<Name>; the root $id is
	// pinned explicitly below. ExpandedStruct inlines the root document
	// type's properties into the root schema instead of leaving a $ref to
	// its own $defs entry.
	reflector := &jsonschema.Reflector{
		Anonymous:      true,
		ExpandedStruct: true,
	}

	schema := reflector.Reflect(&models.KnotExportDocument{})

	// Pin the document identity: exact $id and draft 2020-12 dialect.
	schema.ID = jsonschema.ID(schemaID)
	schema.Version = jsonschema.Version

	// Pin the envelope discriminator fields as constants so any document
	// validated against this schema must carry format "knot" and version 1.
	// The values come from the same constants the export builder uses.
	if err := pinProperty(schema, "format", models.WireFormat); err != nil {
		return err
	}
	if err := pinProperty(schema, "version", models.WireVersion); err != nil {
		return err
	}

	// json.Marshal is deterministic here: struct fields serialize in
	// declaration order, the Properties ordered map preserves struct field
	// order, and the Definitions map is serialized with sorted keys.
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write schema: %w", err)
	}

	fmt.Printf("wrote %s\n", outputPath)
	return nil
}

// pinProperty sets a JSON Schema "const" on a top-level property of the root
// schema, failing loudly if reflection ever renames or drops the property.
func pinProperty(schema *jsonschema.Schema, name string, value any) error {
	prop, ok := schema.Properties.Get(name)
	if !ok || prop == nil {
		return fmt.Errorf("root schema has no %q property to pin", name)
	}
	prop.Const = value
	return nil
}
