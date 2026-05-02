package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var resyncZoneCmd = &cobra.Command{
	Use:   "resync [zone]...",
	Short: "Resynchronize one or more DNS zones",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold).SprintFunc()
		amber := color.New(color.FgYellow).SprintFunc()

		client := api.New()
		for _, zone := range args {
			_, _, err := client.GetJSON("/api/zones/resync", url.Values{"zone": {zone}})
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to resync zone %s: %v\n", amber(zone), err)
				os.Exit(1)
			}
			fmt.Printf("✅ Zone %v resynced successfully.\n", bold(zone))
		}
	},
}

func init() {
	rootCmd.AddCommand(resyncZoneCmd)
}
