package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var importFile string
var importJSON bool

var importCmd = &cobra.Command{
	Use:     "import [zone]",
	Aliases: []string{"im"},
	Short:   "Import a DNS zone",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("token")
		host := viper.GetString("host")

		zone := args[0]

		data, err := ioutil.ReadFile(importFile)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			os.Exit(1)
		}

		url := fmt.Sprintf("%s/api/zones/import?token=%s&zone=%s", host, token, zone)
		req, err := http.NewRequest("POST", url, strings.NewReader(string(data)))
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if status, ok := result["status"].(string); !ok || status != "ok" {
				if msg, ok := result["errorMessage"].(string); ok {
					fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
				} else {
					fmt.Fprintln(os.Stderr, "❌ Unexpected API error")
				}
				os.Exit(1)
			}
		}

		if importJSON {
			fmt.Println(string(body))
		} else {
			fmt.Printf("✅ Zone '%s' imported successfully.\n", zone)
		}
	},
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "data.txt", "Zone file to import")
	importCmd.Flags().BoolVar(&importJSON, "json", false, "Print raw JSON response")
	rootCmd.AddCommand(importCmd)
}
