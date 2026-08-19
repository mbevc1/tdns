package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var (
	importFile string
	importJSON bool

	// import options mapped onto /api/zones/import query parameters
	importOverwrite          bool
	importOverwriteZone      bool
	importOverwriteSoaSerial bool

	// create the zone before importing into it
	importCreate     bool
	importCreateType string
)

// importableZoneTypes are the zone types the server accepts records for via
// /api/zones/import, so the only types --create may produce.
var importableZoneTypes = map[string]bool{
	"Primary":   true,
	"Forwarder": true,
}

// minOverwriteZoneVersion is the first server release that understands the
// `overwriteZone` parameter. Older servers ignore unknown parameters, so
// without this check the import would silently not overwrite the zone.
const minOverwriteZoneVersion = "15.0"

// checkOverwriteZoneSupported exits when the server is too old to honor
// `overwriteZone`. A server that won't report its version (for example when
// the token lacks permission to read settings) only produces a warning, since
// refusing the import outright would be worse than the silent no-op we are
// trying to prevent.
func checkOverwriteZoneSupported(client *api.Client) {
	ok, version, err := client.ServerAtLeast(minOverwriteZoneVersion)
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "⚠️  Could not verify that the server supports --overwrite-zone (needs v%s+): %v\n", minOverwriteZoneVersion, err)
	case !ok:
		fmt.Fprintf(os.Stderr, "❌ --overwrite-zone requires Technitium DNS Server v%s or later, but the server reports v%s.\n", minOverwriteZoneVersion, version)
		fmt.Fprintln(os.Stderr, "Older servers ignore the option and would import without clearing the zone.")
		os.Exit(1)
	}
}

// buildImportQuery maps the import flags onto /api/zones/import query
// parameters. `overwriteZone` is only sent when enabled so that the request
// stays identical to previous releases against servers older than v15.0.
func buildImportQuery(zone string, overwrite, overwriteZone, overwriteSoaSerial bool) url.Values {
	q := url.Values{
		"zone":               {zone},
		"overwrite":          {strconv.FormatBool(overwrite)},
		"overwriteSoaSerial": {strconv.FormatBool(overwriteSoaSerial)},
	}
	if overwriteZone {
		q.Set("overwriteZone", "true")
	}
	return q
}

var importCmd = &cobra.Command{
	Use:     "import [zone]",
	Aliases: []string{"im"},
	Short:   "Import a DNS zone",
	Long: `Import records from an RFC 1035 (BIND style) zone file into an existing zone.

The zone must already exist and be of type Primary or Forwarder; pass --create
to create it first.

With --overwrite-zone (Technitium v15.0+) every existing record in the zone is
deleted before the import, so only the imported records remain. Note that this
includes the zone's apex NS records, so the zone file must contain them. The
zone's SOA record is kept, as are DNSSEC records (DNSKEY/RRSIG/NSEC/NSEC3/
NSEC3PARAM), which the server always manages itself and ignores on import.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]

		if importCreate && !importableZoneTypes[importCreateType] {
			fmt.Fprintf(os.Stderr, "❌ Invalid zone type: %s\n", importCreateType)
			fmt.Fprintln(os.Stderr, "Only Primary and Forwarder zones can be imported into.")
			os.Exit(1)
		}

		data, err := os.ReadFile(importFile)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			os.Exit(1)
		}

		client := api.New()

		if importOverwriteZone {
			// Verify support before prompting, so the user isn't asked to
			// confirm something the server would silently ignore.
			checkOverwriteZoneSupported(client)

			if !assumeYes {
				fmt.Printf("This deletes all existing records in zone '%s' before importing. Are you sure? (yes/no): ", zone)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("❌ Aborted.")
					return
				}
			}
		}

		if importCreate {
			// The zone file supplies the SOA serial, so leave the server's
			// date-based serial scheme off.
			if _, err := createZone(client, zone, importCreateType, false, ""); err != nil {
				if !isZoneExistsError(err) {
					fmt.Fprintf(os.Stderr, "❌ Failed to create zone %s: %v\n", zone, err)
					os.Exit(1)
				}
				if !importJSON {
					fmt.Printf("ℹ️  Zone '%s' already exists, importing into it.\n", zone)
				}
			} else if !importJSON {
				fmt.Printf("✅ Zone '%s' created successfully.\n", zone)
			}
		}

		q := buildImportQuery(zone, importOverwrite, importOverwriteZone, importOverwriteSoaSerial)
		resp, err := client.Post("/api/zones/import", q, bytes.NewReader(data), "text/plain")
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if importJSON {
				raw, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(raw))
				return
			}
			if status, ok := result["status"].(string); !ok || status != "ok" {
				if msg, ok := result["errorMessage"].(string); ok {
					fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
				} else {
					fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
				}
				os.Exit(1)
			}
		}

		fmt.Printf("✅ Zone '%s' imported successfully.\n", zone)
	},
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "data.txt", "Zone file to import")
	importCmd.Flags().BoolVar(&importJSON, "json", false, "Print raw JSON response")
	importCmd.Flags().BoolVar(&importOverwrite, "overwrite", true, "Overwrite existing record sets for the records being imported")
	importCmd.Flags().BoolVar(&importOverwriteZone, "overwrite-zone", false, "Delete all existing records in the zone before importing (Technitium v15.0+)")
	importCmd.Flags().BoolVar(&importOverwriteSoaSerial, "overwrite-soa-serial", true, "Take the SOA serial from the imported file. Warning: a serial lower than the current one makes secondary zones fail to sync")
	importCmd.Flags().BoolVar(&importCreate, "create", false, "Create the zone first if it does not exist")
	importCmd.Flags().StringVar(&importCreateType, "type", "Primary", "Zone type to use with --create (Primary or Forwarder)")
	importCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Assume yes when asking for confirmation")
	rootCmd.AddCommand(importCmd)
}
