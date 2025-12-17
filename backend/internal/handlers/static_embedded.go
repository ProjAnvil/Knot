package handlers

import (
	"io/fs"
	"mime"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ServeEmbeddedFiles serves static files from an embedded filesystem
func ServeEmbeddedFiles(embedFS fs.FS) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip API routes
		if strings.HasPrefix(path, "/api/") {
			return c.Next()
		}

		// Normalize path
		if path == "" || path == "/" {
			path = "/index.html"
		}

		// Remove leading slash for fs.FS
		filePath := strings.TrimPrefix(path, "/")

		// Try to read the file
		data, err := fs.ReadFile(embedFS, filePath)
		if err != nil {
			// If file not found, try index.html for SPA routing
			indexData, indexErr := fs.ReadFile(embedFS, "index.html")
			if indexErr != nil {
				return c.Status(404).SendString("Not Found")
			}
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.Send(indexData)
		}

		// Set content type based on file extension
		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Set("Content-Type", contentType)

		return c.Send(data)
	}
}
