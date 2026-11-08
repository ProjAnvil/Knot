package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// assetsPrefix is the directory Vite writes content-hashed bundles into
// (e.g. /assets/index-ByJUNkX8.js). Because the file name changes whenever the
// content changes, these responses can be cached forever by the browser.
const assetsPrefix = "assets/"

// embeddedETags caches computed ETags per embedded file path.
var embeddedETags sync.Map // map[string]string

// setImmutableAssetCache marks content-hashed bundles as cacheable for a year.
// It returns false when the path is not a hashed asset, so the caller can fall
// back to revalidating cache semantics.
func setImmutableAssetCache(c *fiber.Ctx, filePath string) bool {
	normalized := strings.TrimPrefix(filePath, "/")
	if !strings.HasPrefix(normalized, assetsPrefix) {
		return false
	}
	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	return true
}

// setRevalidatingCache marks entry documents (index.html and SPA fallbacks) as
// always revalidated and answers conditional requests with 304 Not Modified.
// It returns true when the client's cached copy is still fresh.
func setRevalidatingCache(c *fiber.Ctx, cacheKey string, data []byte) bool {
	c.Set(fiber.HeaderCacheControl, "no-cache")

	etag, ok := embeddedETags.Load(cacheKey)
	if !ok {
		sum := sha256.Sum256(data)
		etag = `"` + hex.EncodeToString(sum[:16]) + `"`
		embeddedETags.Store(cacheKey, etag)
	}
	tag := etag.(string)

	c.Set(fiber.HeaderETag, tag)
	return etagMatches(c.Get(fiber.HeaderIfNoneMatch), tag)
}

// etagMatches reports whether an If-None-Match header value matches the tag.
func etagMatches(header, tag string) bool {
	if header == "" {
		return false
	}
	if strings.TrimSpace(header) == "*" {
		return true
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == tag {
			return true
		}
	}
	return false
}
