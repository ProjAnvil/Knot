package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "knot",
	Short: "Knot - API Documentation Management System",
	Long: `Knot is an API documentation management system with MCP support.

Features:
  - Manage API documentation with groups and parameters
  - Export documentation to HTML
  - MCP (Model Context Protocol) integration
  - Support for multiple databases (SQLite, PostgreSQL, MySQL)`,
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add all subcommands
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(serveCmd) // Hidden command for internal use
}
