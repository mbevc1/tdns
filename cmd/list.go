package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DNS zones",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		url := fmt.Sprintf("%s/api/zones/list?token=%s", host, token)

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

		if listJSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

		response := result["response"].(map[string]interface{})
		zones := response["zones"].([]interface{})

		bold := color.New(color.Bold).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()
		gray := color.New(color.FgHiBlack).SprintFunc()
		amber := color.New(color.FgYellow).SprintFunc()

		names := make([]string, 0, len(zones))
		zoneMap := make(map[string]map[string]interface{})

		for _, z := range zones {
			zone := z.(map[string]interface{})
			name := zone["name"].(string)
			names = append(names, name)
			zoneMap[name] = zone
		}

		sort.Strings(names)

		for _, name := range names {
			zone := zoneMap[name]
			ztype := zone["type"].(string)
			serial := int64(zone["soaSerial"].(float64))
			modified := zone["lastModified"].(string)

			status := green("Enabled")
			if zone["disabled"].(bool) {
				status = red("Disabled")
			}

			scope := gray("External")
			if internal, ok := zone["internal"].(bool); ok && internal {
				scope = blue("Internal")
			}

			dnssec := amber("Unsigned")
			if s, ok := zone["dnssecStatus"].(string); ok && s != "" {
				dnssec = amber(s)
			}

			fmt.Printf("%s (%s)\n", bold(name), blue(ztype))
			fmt.Printf("  Last Modified: %s\n", modified)
			fmt.Printf("  SOA Serial: %d\n", serial)
			fmt.Printf("  Status: %s | %s\n", status, scope)
			fmt.Printf("  DNSSEC: %s\n", dnssec)
			fmt.Println()
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output raw JSON response")
	rootCmd.AddCommand(listCmd)
}
