package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

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

// createZone calls /api/zones/create and returns the domain name reported by
// the server. It is shared by the `create` command and `import --create`.
func createZone(client *api.Client, zone, zoneType string, useSoaSerialDateScheme bool, primaryNameServerAddresses string) (string, error) {
	q := url.Values{
		"zone":                   {zone},
		"type":                   {zoneType},
		"useSoaSerialDateScheme": {strconv.FormatBool(useSoaSerialDateScheme)},
	}
	if primaryNameServerAddresses != "" {
		q.Set("primaryNameServerAddresses", primaryNameServerAddresses)
	}

	_, response, err := client.GetJSON("/api/zones/create", q)
	if err != nil {
		return "", err
	}
	domain, _ := response["domain"].(string)
	return domain, nil
}

// isZoneExistsError reports whether err is the API's "Zone already exists"
// rejection, which `import --create` treats as success.
func isZoneExistsError(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "zone already exists")
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
			domain, err := createZone(client, zone, zoneType, useSerial, nameServers)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to create zone %s: %v\n", zone, err)
				os.Exit(1)
			}
			fmt.Printf("✅ Zone %v created successfully.\n", domain)
		}
	},
}

func init() {
	createCmd.Flags().StringP("type", "y", "Primary", "Zone type")
	createCmd.Flags().Bool("useSoaSerialDateScheme", true, "Use date-based SOA serial scheme")
	createCmd.Flags().String("primaryNameServerAddresses", "", "Comma-separated list of primary name server IPs")
	rootCmd.AddCommand(createCmd)
}
