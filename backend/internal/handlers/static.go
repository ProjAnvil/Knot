package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ServeStaticFiles serves static files from the frontend dist directory
// It supports both development mode (reading from ../frontend/dist) and production mode
func ServeStaticFiles(distPath string) fiber.Handler {
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

		// Build file path
		filePath := filepath.Join(distPath, path)

		// Check if file exists
		stat, err := os.Stat(filePath)
		if err != nil || stat.IsDir() {
			// If file not found or is a directory, try index.html for SPA routing
			indexPath := filepath.Join(distPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				return c.SendFile(indexPath)
			}
			return c.Status(404).SendString("Not Found")
		}

		// Serve the file
		return c.SendFile(filePath)
	}
}

// GetDistPath returns the path to the frontend dist directory
// It tries multiple locations in order:
// 1. ./web/dist (embedded or production location)
// 2. ./frontend/dist (development mode in same directory)
// 3. ../frontend/dist (development mode from backend-go directory)
func GetDistPath() (string, error) {
	// Get the executable directory
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)

	// Try multiple locations
	candidates := []string{
		filepath.Join(exeDir, "web", "dist"),            // Production/embedded
		filepath.Join(exeDir, "frontend", "dist"),       // Development (same dir)
		filepath.Join(exeDir, "..", "frontend", "dist"), // Development (parent has frontend)
		"./web/dist",       // Relative to CWD
		"./frontend/dist",  // Relative to CWD
		"../frontend/dist", // Relative to CWD
	}

	for _, path := range candidates {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			// Check if index.html exists
			indexPath := filepath.Join(path, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				absPath, _ := filepath.Abs(path)
				return absPath, nil
			}
		}
	}

	return "", os.ErrNotExist
}
