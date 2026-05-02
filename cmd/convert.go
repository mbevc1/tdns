package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var convertCmd = &cobra.Command{
	Use:     "convert [zone]",
	Aliases: []string{"co"},
	Short:   "Convert zone type",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		zoneType, _ := cmd.Flags().GetString("type")

		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		if zoneType == "" {
			fmt.Fprintln(os.Stderr, "❌ --type is required")
			os.Exit(1)
		}

		q := url.Values{"zone": {zone}, "type": {zoneType}}
		if _, _, err := api.New().GetJSON("/api/zones/convert", q); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Zone %v converted to %v successfully.\n", bold(zone), cyan(zoneType))
	},
}

func init() {
	convertCmd.Flags().StringP("zone", "z", "", "Zone to convert")
	convertCmd.Flags().StringP("type", "y", "", "Target zone type (Primary, Secondary, etc.)")
	rootCmd.AddCommand(convertCmd)
}
