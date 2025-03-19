package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var exportOutputDir string
var exportJSON bool

var exportCmd = &cobra.Command{
	Use:   "export [zones...]",
	Short: "Export one or more DNS zones",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		for _, zone := range args {
			url := fmt.Sprintf("%s/api/zones/export?token=%s&zone=%s", host, token, zone)
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Export failed for %s: %v\n", zone, err)
				continue
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
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
				if exportJSON {
					fmt.Println(string(body))
				} else {
					fmt.Printf("Zone: %s\n", zone)
					fmt.Println(string(body))
				}
			}
		}
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutputDir, "output-dir", "o", "", "Directory to save exported zone files")
	exportCmd.Flags().BoolVar(&exportJSON, "json", false, "Print raw JSON output instead of zone file text")
	rootCmd.AddCommand(exportCmd)
}
