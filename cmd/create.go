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

var validZoneTypes = map[string]bool{
	"Primary":            true,
	"Secondary":          true,
	"Stub":               true,
	"Forwarder":          true,
	"SecondaryForwarder": true,
	"Catalog":            true,
	"SecondaryCatalog":   true,
}

var createCmd = &cobra.Command{
	Use:     "create [zones...]",
	Aliases: []string{"cr"},
	Short:   "Create one or more DNS zones",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		zoneType, _ := cmd.Flags().GetString("type")
		useSerial, _ := cmd.Flags().GetBool("useSoaSerialDateScheme")
		nameServers, _ := cmd.Flags().GetString("primaryNameServerAddresses")

		bold := color.New(color.Bold).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()

		if !validZoneTypes[zoneType] {
			fmt.Fprintf(os.Stderr, "%s Invalid zone type: %s\n", red("❌"), bold(zoneType))
			fmt.Fprintf(os.Stderr, "Valid types are: Primary, Secondary, Stub, Forwarder, SecondaryForwarder, Catalog, SecondaryCatalog\n")
			os.Exit(1)
		}

		for _, zone := range args {
			url := fmt.Sprintf("%s/api/zones/create?token=%s&zone=%s&type=%s&useSoaSerialDateScheme=%t",
				host, token, zone, zoneType, useSerial)

			if nameServers != "" {
				url += fmt.Sprintf("&primaryNameServerAddresses=%s", nameServers)
			}

			resp, err := http.Get(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to create zone %s: %v\n", zone, err)
				continue
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

			response := result["response"].(map[string]interface{})
			fmt.Printf("✅ Zone %v created successfully.\n", response["domain"])
		}
	},
}

func init() {
	createCmd.Flags().StringP("type", "y", "Primary", "Zone type")
	createCmd.Flags().String("useSoaSerialDateScheme", "true", "Use date-based SOA serial scheme (true|false)")
	createCmd.Flags().String("primaryNameServerAddresses", "", "Comma-separated list of primary name server IPs")
	rootCmd.AddCommand(createCmd)
}
