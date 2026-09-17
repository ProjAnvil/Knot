package handlers

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEmbeddedFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html><body>knot</body></html>"),
		},
		"assets/index-abc123.js": &fstest.MapFile{
			// Large enough to be compressed by the middleware.
			Data: []byte(strings.Repeat("console.log('knot');", 200)),
		},
	}
}

func newStaticTestApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(ServeEmbeddedFiles(testEmbeddedFS()))
	return app
}

func TestServeEmbeddedFilesHashedAssetIsImmutable(t *testing.T) {
	app := newStaticTestApp()

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/assets/index-abc123.js", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "public, max-age=31536000, immutable", resp.Header.Get(fiber.HeaderCacheControl))
	assert.Contains(t, resp.Header.Get("Content-Type"), "javascript")
}

func TestServeEmbeddedFilesEntryDocumentRevalidates(t *testing.T) {
	app := newStaticTestApp()

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "no-cache", resp.Header.Get(fiber.HeaderCacheControl))

	etag := resp.Header.Get(fiber.HeaderETag)
	require.NotEmpty(t, etag)

	// A second request with the ETag must not resend the body.
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, etag)
	conditional, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotModified, conditional.StatusCode)
}

func TestServeEmbeddedFilesSPAFallbackRevalidates(t *testing.T) {
	app := newStaticTestApp()

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/some/client/route", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "no-cache", resp.Header.Get(fiber.HeaderCacheControl))
	assert.NotEmpty(t, resp.Header.Get(fiber.HeaderETag))
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestServeEmbeddedFilesCompressesAsset(t *testing.T) {
	app := newStaticTestApp()

	req := httptest.NewRequest(fiber.MethodGet, "/assets/index-abc123.js", nil)
	req.Header.Set(fiber.HeaderAcceptEncoding, "gzip")
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get(fiber.HeaderContentEncoding))
	assert.Contains(t, resp.Header.Get(fiber.HeaderVary), "Accept-Encoding")

	reader, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, string(testEmbeddedFS()["assets/index-abc123.js"].Data), string(body))
}

func TestETagMatches(t *testing.T) {
	assert.True(t, etagMatches(`"abc"`, `"abc"`))
	assert.True(t, etagMatches(`W/"abc"`, `"abc"`))
	assert.True(t, etagMatches(`"x", "abc"`, `"abc"`))
	assert.True(t, etagMatches(`*`, `"abc"`))
	assert.False(t, etagMatches(``, `"abc"`))
	assert.False(t, etagMatches(`"other"`, `"abc"`))
}
