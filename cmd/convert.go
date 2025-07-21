package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var convertCmd = &cobra.Command{
	Use:     "convert [zone]",
	Aliases: []string{"co"},
	Short:   "Convert zone type",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		zone := args[0]
		zoneType, _ := cmd.Flags().GetString("type")

		bold := color.New(color.Bold).SprintFunc()
		amber := color.New(color.FgYellow).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		if zoneType == "" {
			fmt.Fprintln(os.Stderr, "❌ --type is required")
			os.Exit(1)
		}

		url := fmt.Sprintf("%s/api/zones/convert?token=%s&zone=%s&type=%s", host, token, zone, zoneType)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to convert zone %s: %v\n", amber(zone), err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid response: %v\n", err)
			os.Exit(1)
		}

		if status, ok := result["status"].(string); !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok {
				fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
			} else {
				fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
			}
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
