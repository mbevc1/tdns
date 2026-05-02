package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var (
	recordType string
	jsonOutput bool
	overwrite  bool
	assumeYes  bool
	zoneName   string
	recordTTL  int
	ipAddress  string
	cnameValue string
	domainName string
)

var recordsGetCmd = &cobra.Command{
	Use:     "get [zone]",
	Aliases: []string{"ge"},
	Short:   "List all DNS records for a zone",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		q := url.Values{
			"domain":   {zone},
			"zone":     {zone},
			"listZone": {"true"},
		}
		result, response, err := api.New().GetJSON("/api/zones/records/get", q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

		records, ok := response["records"].([]interface{})
		if !ok || len(records) == 0 {
			fmt.Printf("No records found for %s.\n", zone)
			return
		}

		fmt.Printf("%s %s\n\n", bold("Records for zone:"), cyan(zone))
		for _, r := range records {
			rec := r.(map[string]interface{})
			rtype := rec["type"].(string)
			if recordType != "" && strings.ToUpper(recordType) != rtype {
				continue
			}

			name := rec["name"]
			ttl := rec["ttl"]
			recordValue := "None"
			if rVal, ok := rec["rData"]; ok && rVal != nil {
				recordValue = FormatMap(rVal)
			}

			fmt.Printf("%s  %s  %g  %s\n", greenL(name), rtype, ttl, recordValue)
		}
	},
}

var recordsCmd = &cobra.Command{
	Use:     "records",
	Aliases: []string{"re"},
	Short:   "Manage zone records",
}

func recordQuery() url.Values {
	q := url.Values{
		"domain": {domainName},
		"zone":   {zoneName},
		"type":   {recordType},
	}
	if recordTTL >= 0 {
		q.Set("ttl", strconv.Itoa(recordTTL))
	}
	if ipAddress != "" {
		q.Set("ipAddress", ipAddress)
	}
	if cnameValue != "" {
		q.Set("cname", cnameValue)
	}
	return q
}

var recordsAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a"},
	Short:   "Add a new record to a zone",
	Run: func(cmd *cobra.Command, args []string) {
		if zoneName == "" || recordType == "" {
			fmt.Fprintln(os.Stderr, "❌ --zone and --type are required")
			os.Exit(1)
		}

		q := recordQuery()
		q.Set("overwrite", strconv.FormatBool(overwrite))

		result, _, err := api.New().GetJSON("/api/zones/records/add", q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			raw, _ := json.MarshalIndent(result["response"], "", "  ")
			fmt.Println(string(raw))
			return
		}
	},
}

var recordsDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"d", "del"},
	Short:   "Delete a record from a zone",
	Run: func(cmd *cobra.Command, args []string) {
		if zoneName == "" || recordType == "" {
			fmt.Fprintln(os.Stderr, "❌ --zone and --type are required")
			os.Exit(1)
		}

		if !assumeYes {
			fmt.Printf("Are you sure you want to delete this record (%s)? (yes/no): ", domainName)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("❌ Aborted.")
				return
			}
		}

		if _, _, err := api.New().GetJSON("/api/zones/records/delete", recordQuery()); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Record deleted successfully.")
	},
}

func init() {
	recordsDeleteCmd.Flags().StringVarP(&zoneName, "zone", "z", "", "Zone name")
	recordsDeleteCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Assume yes when asking for confirmation")
	recordsDeleteCmd.Flags().StringVarP(&domainName, "domain", "n", "", "Domain name")
	recordsDeleteCmd.Flags().StringVarP(&recordType, "type", "r", "", "Record type")
	recordsDeleteCmd.Flags().IntVarP(&recordTTL, "ttl", "", -1, "Time to live")
	recordsDeleteCmd.Flags().StringVar(&ipAddress, "ipAddress", "", "IP address for A/AAAA records")
	recordsDeleteCmd.Flags().StringVar(&cnameValue, "cname", "", "CNAME target")
	recordsCmd.AddCommand(recordsDeleteCmd)
	recordsAddCmd.Flags().StringVarP(&zoneName, "zone", "z", "", "Zone name")
	recordsAddCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite existing record if present")
	recordsAddCmd.Flags().StringVarP(&domainName, "domain", "n", "", "Domain name")
	recordsAddCmd.Flags().StringVarP(&recordType, "type", "r", "", "Record type")
	recordsAddCmd.Flags().IntVarP(&recordTTL, "ttl", "", -1, "Time to live")
	recordsAddCmd.Flags().StringVar(&ipAddress, "ipAddress", "", "IP address for A/AAAA records")
	recordsAddCmd.Flags().StringVar(&cnameValue, "cname", "", "CNAME target")
	recordsAddCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON of response")
	recordsCmd.AddCommand(recordsAddCmd)
	recordsGetCmd.Flags().StringVarP(&recordType, "filter", "f", "", "Filter by record type (e.g. A, MX, TXT)")
	recordsGetCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output raw JSON instead of formatted text")
	recordsCmd.AddCommand(recordsGetCmd)
	rootCmd.AddCommand(recordsCmd)
}
