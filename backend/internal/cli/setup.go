package cli

import (
	"fmt"

	"github.com/ProjAnvil/knot/backend/internal/config"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize configuration file",
	Long:  `Initialize the configuration file with default values at ~/.knot/config.json`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Initializing Knot configuration...")

		if err := config.InitConfig(); err != nil {
			fmt.Printf("❌ Failed to initialize config: %v\n", err)
			return
		}

		fmt.Println("\n✅ Setup complete! Run 'knot start' to start the server.")
	},
}
