// cmd/zone_options_set.go
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	optionsDataFile string
	optionsStdin    bool

	// flags
	flagDisabled                       bool
	flagCatalog                        string
	flagPrimaryNameServerAddresses     string // CSV -> comma-joined
	flagPrimaryZoneTransferTsigKeyName string
	flagValidateZone                   bool
	flagNotify                         string // flexible as requested
	flagNotifyNameServers              string // CSV -> comma-joined
)

var setZoneOptionsCmd = &cobra.Command{
	Use:   "set-options [zone]",
	Short: "Set zone options (Technitium /api/zones/options/set) via query parameters",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		token := viper.GetString("token")
		host := viper.GetString("host")

		// styles
		bold := color.New(color.Bold).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()

		// 1) Start with payload from JSON (file|stdin) if provided.
		// We'll treat it as a generic map and then fold into query params.
		var base map[string]interface{} = map[string]interface{}{}

		if optionsDataFile != "" && optionsStdin {
			fmt.Fprintln(os.Stderr, red("❌ --data-file and --stdin are mutually exclusive"))
			os.Exit(1)
		}

		if optionsDataFile != "" {
			body, err := os.ReadFile(optionsDataFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s failed to read %s: %v\n", red("❌"), optionsDataFile, err)
				os.Exit(1)
			}
			if len(bytes.TrimSpace(body)) > 0 {
				if err := json.Unmarshal(body, &base); err != nil {
					fmt.Fprintf(os.Stderr, "%s invalid JSON in %s: %v\n", red("❌"), optionsDataFile, err)
					os.Exit(1)
				}
			}
		} else if optionsStdin {
			body, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s failed to read stdin: %v\n", red("❌"), err)
				os.Exit(1)
			}
			if len(bytes.TrimSpace(body)) == 0 {
				fmt.Fprintln(os.Stderr, red("❌ no data received on stdin"))
				os.Exit(1)
			}
			if err := json.Unmarshal(body, &base); err != nil {
				fmt.Fprintf(os.Stderr, "%s invalid JSON on stdin: %v\n", red("❌"), err)
				os.Exit(1)
			}
		}

		// 2) Build query params from base payload first
		q := url.Values{}
		q.Set("token", token)
		q.Set("zone", zone)

		// Helper to set a key from "base" map if present
		setFromBase := func(key string) {
			if v, ok := base[key]; ok {
				switch vv := v.(type) {
				case []interface{}:
					// join arrays by comma
					q.Set(key, joinInterfaceCSV(vv))
				case []string:
					q.Set(key, strings.Join(vv, ","))
				case bool:
					if vv {
						q.Set(key, "true")
					} else {
						q.Set(key, "false")
					}
				default:
					q.Set(key, fmt.Sprintf("%v", vv))
				}
			}
		}

		// Known keys we support commonly
		setFromBase("disabled")
		setFromBase("catalog")
		setFromBase("primaryNameServerAddresses")
		setFromBase("primaryZoneTransferTsigKeyName")
		setFromBase("validateZone")
		setFromBase("notify")
		setFromBase("notifyNameServers")

		// 3) Override with CLI flags when provided (only if user set the flag)
		if cmd.Flags().Changed("disabled") {
			q.Set("disabled", boolToStr(flagDisabled))
		}
		if cmd.Flags().Changed("catalog") && flagCatalog != "" {
			q.Set("catalog", flagCatalog)
		}
		if cmd.Flags().Changed("primaryNameServerAddresses") && flagPrimaryNameServerAddresses != "" {
			q.Set("primaryNameServerAddresses", joinCSV(flagPrimaryNameServerAddresses))
		}
		if cmd.Flags().Changed("primaryZoneTransferTsigKeyName") && flagPrimaryZoneTransferTsigKeyName != "" {
			q.Set("primaryZoneTransferTsigKeyName", flagPrimaryZoneTransferTsigKeyName)
		}
		if cmd.Flags().Changed("validateZone") {
			q.Set("validateZone", boolToStr(flagValidateZone))
		}
		if cmd.Flags().Changed("notify") && flagNotify != "" {
			q.Set("notify", flagNotify)
		}
		if cmd.Flags().Changed("notifyNameServers") && flagNotifyNameServers != "" {
			q.Set("notifyNameServers", joinCSV(flagNotifyNameServers))
		}

		// If only token/zone are present, nothing was set
		if len(q) <= 2 {
			fmt.Fprintln(os.Stderr, red("❌ no options provided — use flags and/or --data-file/--stdin"))
			os.Exit(1)
		}

		// 4) Call API with query params (GET, per API expectation)
		endpoint := fmt.Sprintf("%s/api/zones/options/set?%s", host, q.Encode())
		resp, err := http.Get(endpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s request failed: %v\n", red("❌"), err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "%s invalid response: %v\n", red("❌"), err)
			os.Exit(1)
		}

		// If status != ok, print only errorMessage (if present), else generic
		if status, ok := result["status"].(string); !ok || status != "ok" {
			if msg, ok := result["errorMessage"].(string); ok && strings.TrimSpace(msg) != "" {
				fmt.Fprintln(os.Stderr, red("❌ ")+msg)
			} else {
				fmt.Fprintln(os.Stderr, red("❌ Unexpected API error"))
			}
			os.Exit(1)
		}

		fmt.Printf("%s %s %s\n", green("✅"), bold("Zone options updated for:"), zone)
	},
}

func init() {
	setZoneOptionsCmd.Flags().StringVarP(&optionsDataFile, "data-file", "f", "", "Path to JSON file; keys become query parameters")
	setZoneOptionsCmd.Flags().BoolVar(&optionsStdin, "stdin", false, "Read JSON from stdin; keys become query parameters")

	setZoneOptionsCmd.Flags().BoolVar(&flagDisabled, "disabled", false, "Set zone disabled state (true|false)")
	setZoneOptionsCmd.Flags().StringVar(&flagCatalog, "catalog", "", "Catalog zone name")

	setZoneOptionsCmd.Flags().StringVar(&flagPrimaryNameServerAddresses, "primaryNameServerAddresses", "", "Comma-separated primary name server IPs")
	setZoneOptionsCmd.Flags().StringVar(&flagPrimaryZoneTransferTsigKeyName, "primaryZoneTransferTsigKeyName", "", "Primary zone transfer TSIG key name")

	setZoneOptionsCmd.Flags().BoolVar(&flagValidateZone, "validateZone", false, "Validate zone after applying options (true|false)")

	// Flexible notify flags
	setZoneOptionsCmd.Flags().StringVar(&flagNotify, "notify", "", "Notify setting (e.g. None, ZoneNameServers, UseSpecified)")
	setZoneOptionsCmd.Flags().StringVar(&flagNotifyNameServers, "notifyNameServers", "", "Comma-separated list of notify name servers (used with --notify=UseSpecified)")

	rootCmd.AddCommand(setZoneOptionsCmd)
}

// ---- helpers ----

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func joinCSV(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, ",")
}

func joinInterfaceCSV(arr []interface{}) string {
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		out = append(out, fmt.Sprintf("%v", v))
	}
	return strings.Join(out, ",")
}
