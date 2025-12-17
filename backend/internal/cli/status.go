package cli

import (
	"fmt"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status",
	Long:  `Display the current status of the Knot server`,
	Run: func(cmd *cobra.Command, args []string) {
		pid, err := ReadPID()
		if err != nil {
			fmt.Printf("❌ Failed to read PID file: %v\n", err)
			return
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("❌ Failed to load config: %v\n", err)
			return
		}

		fmt.Println("\n📊 Knot Server Status\n")

		if pid == 0 {
			fmt.Println("Status:  ⭕ Not running")
			fmt.Println()
			return
		}

		if IsProcessRunning(pid) {
			fmt.Println("Status:  ✅ Running")
			fmt.Printf("PID:     %d\n", pid)
			fmt.Printf("URL:     http://%s:%d\n", cfg.Host, cfg.Port)
		} else {
			fmt.Println("Status:  ⭕ Not running (stale PID file)")
			if err := RemovePID(); err != nil {
				fmt.Printf("⚠️  Failed to remove stale PID file: %v\n", err)
			}
		}

		fmt.Println()
	},
}
