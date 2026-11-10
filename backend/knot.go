// Package knot embeds the Knot API-documentation engine into any Go host.
//
// A host opens its own *gorm.DB, constructs the engine with New, optionally
// mounts the HTTP routes onto its Fiber app with Mount, and otherwise calls
// the service methods (Export, Import) directly. The engine owns no global
// state of its own beyond the process-wide models.TablePrefix that
// Config.TablePrefix installs (only when non-empty, so a host-set prefix is
// never clobbered).
//
// Ownership rules: the database connection belongs to the host (Close is a
// documented no-op); Mount only registers routes — the host keeps ownership
// of static file serving, health endpoints and any other middleware.
package knot

import (
	"errors"
	"fmt"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/handlers"
	"github.com/ProjAnvil/knot/backend/internal/services"
	models "github.com/ProjAnvil/knot/models"
	"github.com/ProjAnvil/knot/models/gormbind"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DefaultAPIPrefix is the route prefix used when Config.APIPrefix is empty,
// identical to the prefix the standalone Knot server serves.
const DefaultAPIPrefix = "/api"

// Sentinel errors reported by the service methods. Hosts map them to HTTP
// statuses with errors.Is: ErrUnsupportedFormat is the 415 case (mirroring
// the import handler), ErrNoAPIIDs and ErrImportParse are user-input (400)
// cases, and ErrExportFailed and ErrImportApply are internal (500) cases.
// Non-sentinel detail is preserved by wrapping, so err.Error() stays
// log-worthy while errors.Is keeps the mapping stable.
var (
	// ErrUnsupportedFormat reports an export format or import document the
	// engine does not recognize.
	ErrUnsupportedFormat = errors.New("knot: unsupported format")
	// ErrNoAPIIDs reports an Export call with an empty apiIDs slice.
	ErrNoAPIIDs = errors.New("knot: no API IDs provided")
	// ErrExportFailed wraps database or encoding failures during Export.
	ErrExportFailed = errors.New("knot: export failed")
	// ErrImportParse wraps failures to parse a detected import document.
	ErrImportParse = errors.New("knot: failed to parse import document")
	// ErrImportApply wraps failures to persist an import plan.
	ErrImportApply = errors.New("knot: failed to apply import document")
)

// Config configures an engine instance. DB is required. APIPrefix defaults
// to DefaultAPIPrefix when empty. TablePrefix, when non-empty, is installed
// as the process-wide models.TablePrefix before migration so every engine
// table is namespaced (e.g. "knot_" → knot_groups); when empty, any prefix
// the host already set is left untouched.
type Config struct {
	DB          *gorm.DB
	APIPrefix   string
	TablePrefix string
}

// Knot is an instance of the embeddable engine. Construct with New.
type Knot struct {
	db      *gorm.DB
	prefix  string
	mounted bool
}

// New validates the configuration, installs the table prefix (only when
// configured — a host-set prefix is never clobbered), runs the canonical
// AutoMigrate via models/gormbind, and returns a ready engine. All errors
// are wrapped with the failing step.
func New(cfg Config) (*Knot, error) {
	if cfg.DB == nil {
		return nil, errors.New("knot: Config.DB must not be nil")
	}

	if cfg.TablePrefix != "" {
		models.SetTablePrefix(cfg.TablePrefix)
	}

	if err := gormbind.AutoMigrate(cfg.DB); err != nil {
		return nil, fmt.Errorf("knot: auto-migrate failed: %w", err)
	}

	prefix := cfg.APIPrefix
	if prefix == "" {
		prefix = DefaultAPIPrefix
	}

	return &Knot{db: cfg.DB, prefix: prefix}, nil
}

// Mount registers the engine's HTTP routes on the app under Config.APIPrefix
// (default "/api"). Mount is idempotent: a second call is a no-op, so hosts
// can mount defensively. Static file serving, health checks and host routes
// are not touched — those stay with the host.
func (k *Knot) Mount(app *fiber.App) {
	if k.mounted {
		return
	}
	k.mounted = true
	handlers.RegisterRoutes(app, k.db, k.prefix)
}

// Export renders the selected APIs in the given format and returns the
// payload, the download filename and the Content-Type, byte-identical to the
// HTTP export handler: format "" and "html" produce the HTML document,
// "json", "openapi" and "postman" the respective JSON documents. locale
// selects the HTML language and defaults to "zh" when empty. The payload and
// filename are stamped with the current time at second granularity, exactly
// like the handler.
func (k *Knot) Export(apiIDs []uint, format string, locale string) ([]byte, string, error) {
	payload, filename, _, err := services.BuildExportDocument(k.db, apiIDs, format, locale, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNoAPIIDs):
			return nil, "", ErrNoAPIIDs
		case errors.Is(err, services.ErrUnsupportedExportFormat):
			return nil, "", ErrUnsupportedFormat
		default:
			return nil, "", fmt.Errorf("%w: %w", ErrExportFailed, err)
		}
	}
	return payload, filename, nil
}

// Import detects the format of the given document bytes (Knot JSON, Postman
// Collection v2.x, OpenAPI 3.x), parses it and applies it in one
// transaction. With dryRun=true the same resolution logic runs but nothing
// is written and the returned counts are the ones a real run would produce.
// Error mapping mirrors the HTTP import handler: an undetectable or
// unsupported document wraps ErrUnsupportedFormat (the 415 case), a parse
// failure wraps ErrImportParse, and a persistence failure wraps
// ErrImportApply.
func (k *Knot) Import(data []byte, dryRun bool) (*models.ImportResult, error) {
	format, err := services.DetectImportFormat(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnsupportedFormat, err)
	}

	plan, err := parseImportDocument(format, data)
	if err != nil {
		return nil, err
	}

	result, err := services.ApplyImport(k.db, format, plan, dryRun)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrImportApply, err)
	}
	return result, nil
}

// parseImportDocument dispatches to the importer of a detected format and
// parses the bytes into a plan. Parse failures are returned wrapped in
// ErrImportParse; a format with no importer (defensive: DetectImportFormat
// only returns known formats) is returned wrapped in ErrUnsupportedFormat,
// mirroring the handler's 415 default case.
func parseImportDocument(format models.ImportFormat, data []byte) (*models.ImportPlan, error) {
	var parse func([]byte) (*models.ImportPlan, error)
	switch format {
	case models.ImportFormatKnot:
		parse = services.ParseKnotJSON
	case models.ImportFormatPostman:
		parse = services.ParsePostmanCollection
	case models.ImportFormatOpenAPI:
		parse = services.ParseOpenAPIDocument
	default:
		return nil, fmt.Errorf("%w: unsupported file format", ErrUnsupportedFormat)
	}

	plan, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrImportParse, err)
	}
	return plan, nil
}

// Close releases engine resources. The engine owns none — the database
// connection belongs to the host — so Close is a documented no-op that
// hosts may defer symmetrically with New.
func (k *Knot) Close() error {
	return nil
}
