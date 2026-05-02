package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var enableZoneCmd = &cobra.Command{
	Use:     "enable [zone]...",
	Aliases: []string{"en"},
	Short:   "Enable DNS zones(s)",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold).SprintFunc()

		client := api.New()
		for _, zone := range args {
			if _, _, err := client.GetJSON("/api/zones/enable", url.Values{"zone": {zone}}); err != nil {
				fmt.Fprintf(os.Stderr, "❌ %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Zone %v enabled successfully.\n", bold(zone))
		}
	},
}

func init() {
	rootCmd.AddCommand(enableZoneCmd)
}
