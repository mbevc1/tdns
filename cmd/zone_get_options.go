package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type sprinter func(a ...interface{}) string

var (
	includeAvailableKeys bool
	optionsJSON          bool
)

var getZoneOptionsCmd = &cobra.Command{
	Use:     "get-options [zone]",
	Aliases: []string{"go"},
	Short:   "Get zone options",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		token := viper.GetString("token")
		host := viper.GetString("host")

		// Colors & styles to match existing CLI look-and-feel
		bold := color.New(color.Bold).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		gray := color.New(color.FgHiBlack).SprintFunc()

		url := fmt.Sprintf("%s/api/zones/options/get?token=%s&zone=%s&includeAvailableTsigKeyNames=%t",
			host, token, zone, includeAvailableKeys)

		resp, err := http.Get(url)
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

		respObj, ok := result["response"].(map[string]interface{})
		if !ok {
			fmt.Fprintln(os.Stderr, red("❌ Unexpected response structure"))
			os.Exit(1)
		}

		if optionsJSON {
			raw, _ := json.MarshalIndent(respObj, "", "  ")
			fmt.Println(string(raw))
			return
		}

		// Header
		fmt.Printf("%s %s\n", bold("Zone:"), blue(str(respObj["name"])))
		fmt.Printf("%s %s\n", bold("Type:"), str(respObj["type"]))
		fmt.Printf("%s %s\n", bold("DNSSEC:"), str(respObj["dnssecStatus"]))
		fmt.Printf("%s %s\n", bold("Status:"), onOff(!toBool(respObj["disabled"]), green, red)) // Enabled/Disabled
		if v, ok := respObj["internal"]; ok {
			fmt.Printf("%s %s\n", bold("Scope:"), boolWord(v, blue, gray, "Internal", "External"))
		}
		if v := strOrEmpty(respObj["catalog"]); v != "" {
			fmt.Printf("%s %s\n", bold("Catalog:"), v)
		}
		fmt.Println()

		// Notify status
		if v, ok := respObj["notifyFailed"].(bool); ok {
			fmt.Printf("%s %s\n", bold("Notify Failed:"), boolColor(v, red, green))
		}
		printStringSlice("Notify Failed For", respObj["notifyFailedFor"], gray, 2)
		fmt.Println()

		// Overrides & access
		printBool("Override Catalog Query Access", respObj["overrideCatalogQueryAccess"], green, yellow)
		printBool("Override Catalog Zone Transfer", respObj["overrideCatalogZoneTransfer"], green, yellow)
		printBool("Override Catalog Notify", respObj["overrideCatalogNotify"], green, yellow)
		fmt.Println()

		// Query access
		fmt.Printf("%s %s\n", bold("Query Access:"), str(respObj["queryAccess"]))
		printStringSlice("Query Access ACL", respObj["queryAccessNetworkACL"], gray, 2)
		fmt.Println()

		// Zone transfer
		fmt.Printf("%s %s\n", bold("Zone Transfer:"), str(respObj["zoneTransfer"]))
		printStringSlice("Zone Transfer ACL", respObj["zoneTransferNetworkACL"], gray, 2)
		printStringSlice("Zone Transfer TSIG Keys", respObj["zoneTransferTsigKeyNames"], gray, 2)
		fmt.Println()

		// Notify
		fmt.Printf("%s %s\n", bold("Notify:"), str(respObj["notify"]))
		printStringSlice("Notify Name Servers", respObj["notifyNameServers"], gray, 2)
		fmt.Println()

		// Update policy
		fmt.Printf("%s %s\n", bold("Update Policy:"), str(respObj["update"]))
		printStringSlice("Update Network ACL", respObj["updateNetworkACL"], gray, 2)

		// Update Security Policies
		if usp, ok := respObj["updateSecurityPolicies"].([]interface{}); ok {
			fmt.Println(bold("Update Security Policies:"))
			if len(usp) == 0 {
				fmt.Println("  (none)")
			} else {
				for _, row := range usp {
					m, _ := row.(map[string]interface{})
					tn := str(m["tsigKeyName"])
					dom := str(m["domain"])
					types := sliceToString(m["allowedTypes"])
					fmt.Printf("  TSIG: %s  Domain: %s  Types: %s\n", blue(tn), blue(dom), types)
				}
			}
			fmt.Println()
		}

		// Available choices (catalogs & TSIG keys)
		printStringSlice("Available Catalog Zones", respObj["availableCatalogZoneNames"], gray, 0)
		printStringSlice("Available TSIG Keys", respObj["availableTsigKeyNames"], gray, 0)
	},
}

func init() {
	getZoneOptionsCmd.Flags().BoolVar(&includeAvailableKeys, "include-available-keys", true, "Include available TSIG key names")
	getZoneOptionsCmd.Flags().BoolVar(&optionsJSON, "json", false, "Print raw JSON response")
	rootCmd.AddCommand(getZoneOptionsCmd)
}

// ---------- helpers (SprintFunc-based) ----------

func str(v interface{}) string {
	if v == nil {
		return "None"
	}
	return fmt.Sprintf("%v", v)
}
func strOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	s := strings.ToLower(fmt.Sprintf("%v", v))
	return s == "true" || s == "1" || s == "yes"
}

func boolColor(b bool, yes sprinter, no sprinter) string {
	if b {
		return yes("true")
	}
	return no("false")
}
func boolWord(v interface{}, yes sprinter, no sprinter, yWord, nWord string) string {
	if toBool(v) {
		return yes(yWord)
	}
	return no(nWord)
}
func onOff(enabled bool, yes sprinter, no sprinter) string {
	if enabled {
		return yes("Enabled")
	}
	return no("Disabled")
}

func sliceToString(v interface{}) string {
	if v == nil {
		return "[]"
	}
	if arr, ok := v.([]interface{}); ok {
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			out = append(out, fmt.Sprintf("%v", x))
		}
		sort.Strings(out)
		return "[" + strings.Join(out, ", ") + "]"
	}
	return fmt.Sprintf("%v", v)
}

func printStringSlice(title string, v interface{}, tint sprinter, indent int) {
	prefix := strings.Repeat(" ", indent)
	bold := color.New(color.Bold).SprintFunc()
	if v == nil {
		return
	}
	arr, ok := v.([]interface{})
	if !ok {
		return
	}
	fmt.Println(bold(title + ":"))
	if len(arr) == 0 {
		fmt.Println(prefix + "(none)")
		return
	}
	items := make([]string, 0, len(arr))
	for _, x := range arr {
		items = append(items, fmt.Sprintf("%v", x))
	}
	sort.Strings(items)
	for _, it := range items {
		fmt.Println(prefix + "• " + tint(it))
	}
}

func printBool(title string, v interface{}, yes sprinter, no sprinter) {
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s %s\n", bold(title+":"), boolColor(toBool(v), yes, no))
}
