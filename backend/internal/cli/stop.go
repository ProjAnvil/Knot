package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running server",
	Long:  `Stop the Knot server that is running in the background`,
	Run: func(cmd *cobra.Command, args []string) {
		pid, err := ReadPID()
		if err != nil {
			fmt.Printf("❌ Failed to read PID file: %v\n", err)
			return
		}

		if pid == 0 {
			fmt.Println("No PID file found. Server may not be running.")
			return
		}

		if !IsProcessRunning(pid) {
			fmt.Printf("Server is not running (PID %d not found).\n", pid)
			if err := RemovePID(); err != nil {
				fmt.Printf("⚠️  Failed to remove stale PID file: %v\n", err)
			}
			return
		}

		fmt.Printf("Stopping server (PID %d)...\n", pid)

		// Try graceful shutdown first (SIGTERM)
		if err := KillProcess(pid, true); err != nil {
			fmt.Printf("❌ Failed to send SIGTERM: %v\n", err)
			return
		}

		// Wait for graceful shutdown
		time.Sleep(2 * time.Second)

		// Check if still running
		if IsProcessRunning(pid) {
			fmt.Println("Server did not stop gracefully, forcing shutdown...")
			if err := KillProcess(pid, false); err != nil {
				fmt.Printf("❌ Failed to force kill: %v\n", err)
				return
			}
			time.Sleep(500 * time.Millisecond)
		}

		// Remove PID file
		if err := RemovePID(); err != nil {
			fmt.Printf("⚠️  Server stopped but failed to remove PID file: %v\n", err)
		}

		fmt.Println("✅ Server stopped successfully")
	},
}
