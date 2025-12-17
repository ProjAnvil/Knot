package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the server",
	Long:  `Stop the running server and start it again`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Restarting Knot server...")

		// Stop first
		pid, err := ReadPID()
		if err != nil {
			fmt.Printf("❌ Failed to read PID file: %v\n", err)
			return
		}

		if pid > 0 && IsProcessRunning(pid) {
			fmt.Printf("Stopping server (PID %d)...\n", pid)

			// Try graceful shutdown
			if err := KillProcess(pid, true); err != nil {
				fmt.Printf("⚠️  Failed to send SIGTERM: %v\n", err)
			}

			time.Sleep(2 * time.Second)

			// Force kill if still running
			if IsProcessRunning(pid) {
				if err := KillProcess(pid, false); err != nil {
					fmt.Printf("⚠️  Failed to force kill: %v\n", err)
				}
				time.Sleep(500 * time.Millisecond)
			}

			if err := RemovePID(); err != nil {
				fmt.Printf("⚠️  Failed to remove PID file: %v\n", err)
			}
		} else {
			fmt.Println("Note: Server was not running")
		}

		// Wait a bit before starting
		time.Sleep(1 * time.Second)

		// Start server
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
		newPid, err := StartServerWithServeCommand(selfPath, cfg.Port, cfg.Host)
		if err != nil {
			fmt.Printf("❌ Failed to start server: %v\n", err)
			return
		}

		if err := SavePID(newPid); err != nil {
			fmt.Printf("⚠️  Server started but failed to save PID: %v\n", err)
		}

		fmt.Printf("✅ Server restarted with PID %d\n", newPid)
		fmt.Printf("   URL: http://%s:%d\n", cfg.Host, cfg.Port)
	},
}
