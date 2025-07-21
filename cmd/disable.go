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

var disableZoneCmd = &cobra.Command{
	Use:     "disable [zone]...",
	Aliases: []string{"en"},
	Short:   "Disable DNS zone(s)",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		bold := color.New(color.Bold).SprintFunc()
		amber := color.New(color.FgYellow).SprintFunc()

		for _, zone := range args {
			url := fmt.Sprintf("%s/api/zones/disable?token=%s&zone=%s", host, token, zone)
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

			fmt.Printf("✅ Zone %v disabled successfully.\n", bold(zone))
		}
	},
}

func init() {
	rootCmd.AddCommand(disableZoneCmd)
}
