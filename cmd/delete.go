package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [zone]...",
	Aliases: []string{"de", "rm"},
	Short:   "Delete DNS zone(s)",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.New()
		for _, zone := range args {
			if _, _, err := client.GetJSON("/api/zones/delete", url.Values{"zone": {zone}}); err != nil {
				fmt.Fprintf(os.Stderr, "❌ %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Zone '%s' deleted successfully.\n", zone)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
