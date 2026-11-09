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
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

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

	rootType := reflect.TypeOf(models.KnotExportDocument{})
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

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	// Widen nullable properties to accept null; see widenNullableProperties.
	data, err = widenNullableProperties(data, rootType.Name())
	if err != nil {
		return err
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

// widenNullableProperties post-processes the marshaled schema so every
// property backed by a Go *string field whose json tag lacks omitempty is
// typed ["string", "null"] instead of "string". Go's encoding/json always
// serializes such nil pointers as null, so the un-widened form would reject
// Knot's own exports (for example an API without a note).
//
// invopop's Schema.Type is a plain string and cannot express a type array,
// so the widening runs on the decoded document. The final re-encode is
// deterministic: encoding/json serializes map keys in sorted order.
//
// Like pinProperty, it fails loudly if an expected property is missing or
// unexpectedly typed, so schema-shape changes surface at generation time.
func widenNullableProperties(data []byte, rootTypeName string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode schema for widening: %w", err)
	}

	nullable := nullableStringProps(reflect.TypeOf(models.KnotExportDocument{}))
	for _, typeName := range slices.Sorted(maps.Keys(nullable)) {
		props, err := propertiesOf(doc, rootTypeName, typeName)
		if err != nil {
			return nil, err
		}
		for _, prop := range nullable[typeName] {
			p, ok := props[prop].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s has no %q property to widen", typeName, prop)
			}
			if t, _ := p["type"].(string); t != "string" {
				return nil, fmt.Errorf("%s.%s is typed %q, expected %q", typeName, prop, t, "string")
			}
			p["type"] = []any{"string", "null"}
		}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("re-encode widened schema: %w", err)
	}
	return out, nil
}

// propertiesOf returns the properties container for typeName in the decoded
// schema document: the document root for the reflected root type (which
// ExpandedStruct inlines), otherwise its $defs entry.
func propertiesOf(doc map[string]any, rootTypeName, typeName string) (map[string]any, error) {
	if typeName == rootTypeName {
		props, ok := doc["properties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("root schema has no properties object")
		}
		return props, nil
	}
	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema has no $defs object")
	}
	def, ok := defs[typeName].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema has no %q definition", typeName)
	}
	props, ok := def["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("$defs.%s has no properties object", typeName)
	}
	return props, nil
}

// nullableStringProps walks t and every type reachable from it, returning
// for each struct type the JSON property names backed by a *string field
// whose json tag lacks omitempty - exactly the properties the wire format
// always emits, as null when the pointer is nil. New nullable fields are
// therefore picked up automatically by the next regeneration.
func nullableStringProps(t reflect.Type) map[string][]string {
	nullable := make(map[string][]string)
	seen := make(map[reflect.Type]bool)
	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		if seen[t] {
			return
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(t.Elem())
		case reflect.Struct:
			for _, f := range reflect.VisibleFields(t) {
				if !f.IsExported() {
					continue
				}
				name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
				if name == "-" {
					continue
				}
				if name == "" {
					name = f.Name
				}
				if f.Type.Kind() == reflect.Pointer &&
					f.Type.Elem().Kind() == reflect.String &&
					!strings.Contains(opts, "omitempty") {
					nullable[t.Name()] = append(nullable[t.Name()], name)
				}
				walk(f.Type)
			}
		}
	}
	walk(t)
	for _, names := range nullable {
		slices.Sort(names)
	}
	return nullable
}
