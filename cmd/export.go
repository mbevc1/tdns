package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var exportOutputDir string

var exportCmd = &cobra.Command{
	Use:     "export [zones...]",
	Aliases: []string{"ex"},
	Short:   "Export one or more DNS zones",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.New()
		for _, zone := range args {
			resp, err := client.Get("/api/zones/export", url.Values{"zone": {zone}})
			if err != nil {
				fmt.Printf("Export failed for %s: %v\n", zone, err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("Failed to read response for %s: %v\n", zone, err)
				continue
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil {
				if status, ok := result["status"].(string); ok && status != "ok" {
					if msg, ok := result["errorMessage"].(string); ok {
						fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
					} else {
						fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
					}
					continue
				}
			}

			if exportOutputDir != "" {
				outPath := filepath.Join(exportOutputDir, fmt.Sprintf("%s.zone", zone))
				if err := os.WriteFile(outPath, body, 0644); err != nil {
					fmt.Printf("Failed to write to %s: %v\n", outPath, err)
					continue
				}
				fmt.Printf("✅ Zone '%s' exported to %s\n", zone, outPath)
			} else {
				fmt.Printf("-----\n"+bold("Zone:")+" %s\n-----\n", blue(zone))
				fmt.Println(string(body))
			}
		}
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutputDir, "output-dir", "o", "", "Directory to save exported zone files")
	rootCmd.AddCommand(exportCmd)
}
