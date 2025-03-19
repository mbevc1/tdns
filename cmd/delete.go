package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [zone]",
	Short: "Delete a DNS zone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		zone := args[0]
		url := fmt.Sprintf("%s/api/zones/delete?token=%s&zone=%s", host, token, zone)
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			os.Exit(1)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Delete failed for '%s': %v\n", zone, err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Invalid response: %v\n", err)
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

		fmt.Printf("✅ Zone '%s' deleted successfully.\n", zone)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
