package cli

import (
	"fmt"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings for Knot`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.ShowConfig(); err != nil {
			fmt.Printf("❌ Failed to show config: %v\n", err)
		}
	},
}
