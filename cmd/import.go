package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var importFile string
var importJSON bool

var importCmd = &cobra.Command{
	Use:     "import [zone]",
	Aliases: []string{"im"},
	Short:   "Import a DNS zone",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]

		data, err := os.ReadFile(importFile)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			os.Exit(1)
		}

		q := url.Values{
			"zone":               {zone},
			"overwrite":          {"true"},
			"overwriteSoaSerial": {"true"},
		}
		resp, err := api.New().Post("/api/zones/import", q, bytes.NewReader(data), "text/plain")
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
	rootCmd.AddCommand(importCmd)
}
