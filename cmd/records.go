package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var recordType string
var jsonOutput bool

var recordsGetCmd = &cobra.Command{
	Use:     "get [zone]",
	Aliases: []string{"ge"},
	Short:   "List all DNS records for a zone",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		zone := args[0]
		url := fmt.Sprintf("%s/api/zones/records/get?token=%s&domain=%s&zone=%s&listZone=true", host, token, zone, zone)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
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

		if jsonOutput {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

		response, ok := result["response"].(map[string]interface{})
		if !ok {
			fmt.Println("Unexpected response structure")
			os.Exit(1)
		}

		records, ok := response["records"].([]interface{})
		if !ok || len(records) == 0 {
			fmt.Printf("No records found for %s.\n", zone)
			return
		}

		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()

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
				recordValue = fmt.Sprintf("%v", rVal)
			}

			fmt.Printf("%s  %s  %v  %s\n",
				green(name),
				rtype,
				ttl,
				recordValue,
			)
		}
	},
}

var recordsCmd = &cobra.Command{
	Use:     "records",
	Aliases: []string{"re"},
	Short:   "Manage zone records",
}

var (
	overwrite  bool
	zoneName   string
	recordTTL  int
	rTTL       string
	ipAddress  string
	cnameValue string
	domainName string
)

var recordsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new record to a zone",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		if zoneName == "" || recordType == "" {
			fmt.Fprintln(os.Stderr, "❌ --zone and --type are required")
			os.Exit(1)
		}

		if recordTTL < 0 {
			rTTL = ""
		} else {
			rTTL = fmt.Sprintf("&ttl=%d", recordTTL)
		}

		url := fmt.Sprintf("%s/api/zones/records/add?token=%s&domain=%s&zone=%s&type=%s%s",
			host, token, domainName, zoneName, recordType, rTTL)
		url += fmt.Sprintf("&overwrite=%t", overwrite)

		if ipAddress != "" {
			url += fmt.Sprintf("&ipAddress=%s", ipAddress)
		}
		if cnameValue != "" {
			url += fmt.Sprintf("&cname=%s", cnameValue)
		}

		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to parse response: %v\n", err)
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

		if jsonOutput {
			raw, _ := json.MarshalIndent(result["response"], "", "  ")
			fmt.Println(string(raw))
			return
		}
	},
}

func init() {
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
