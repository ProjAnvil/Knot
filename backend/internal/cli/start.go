package cli

import (
	"fmt"
	"os"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server in background",
	Long:  `Start the Knot server in the background`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Knot server...")

		// Check if server is already running
		pid, err := ReadPID()
		if err != nil {
			fmt.Printf("❌ Failed to read PID file: %v\n", err)
			return
		}

		if pid > 0 && IsProcessRunning(pid) {
			fmt.Printf("❌ Server is already running with PID %d\n", pid)
			fmt.Println("   Use 'knot stop' to stop it first.")
			return
		}

		// Clean up stale PID file
		if pid > 0 {
			fmt.Println("Cleaning up stale PID file...")
			if err := RemovePID(); err != nil {
				fmt.Printf("⚠️  Failed to remove stale PID file: %v\n", err)
			}
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("❌ Failed to load config: %v\n", err)
			return
		}

		// Get path to self (the knot CLI binary)
		selfPath, err := os.Executable()
		if err != nil {
			fmt.Printf("❌ Failed to get executable path: %v\n", err)
			return
		}

		// Start server using self with __serve command
		pid, err = StartServerWithServeCommand(selfPath, cfg.Port, cfg.Host)
		if err != nil {
			fmt.Printf("❌ Failed to start server: %v\n", err)
			return
		}

		// Save PID
		if err := SavePID(pid); err != nil {
			fmt.Printf("⚠️  Server started but failed to save PID: %v\n", err)
		}

		fmt.Printf("✅ Server started with PID %d\n", pid)
		fmt.Printf("   URL: http://%s:%d\n", cfg.Host, cfg.Port)
		fmt.Println("   Use 'knot stop' to stop the server")
		fmt.Println("   Use 'knot status' to check server status")
	},
}
