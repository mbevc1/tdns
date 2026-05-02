package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all DNS zones",
	Run: func(cmd *cobra.Command, args []string) {
		result, response, err := api.New().GetJSON("/api/zones/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if listJSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

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
