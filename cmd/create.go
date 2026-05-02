package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
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

		client := api.New()
		for _, zone := range args {
			q := url.Values{
				"zone":                   {zone},
				"type":                   {zoneType},
				"useSoaSerialDateScheme": {strconv.FormatBool(useSerial)},
			}
			if nameServers != "" {
				q.Set("primaryNameServerAddresses", nameServers)
			}

			_, response, err := client.GetJSON("/api/zones/create", q)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to create zone %s: %v\n", zone, err)
				os.Exit(1)
			}
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
