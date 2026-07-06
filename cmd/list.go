package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"tdns/internal/api"
)

var (
	listJSON       bool
	listFilterName string
	listFilterType string
	listPage       int
	listPerPage    int
)

// zoneTypes are the valid values for the API's filterType parameter.
var zoneTypes = []string{"Primary", "Secondary", "Stub", "Forwarder", "SecondaryForwarder", "Catalog", "SecondaryCatalog"}

// canonicalZoneType matches t against zoneTypes case-insensitively and
// returns the server's canonical spelling.
func canonicalZoneType(t string) (string, bool) {
	for _, v := range zoneTypes {
		if strings.EqualFold(v, t) {
			return v, true
		}
	}
	return "", false
}

// matchWildcard reports whether name matches pattern case-insensitively,
// where `*` matches zero or more characters and `?` matches exactly one.
// It mirrors the server-side filterName semantics (added in v15.3) so that
// filtering also works against older servers that ignore the parameter.
func matchWildcard(pattern, name string) bool {
	var sb strings.Builder
	sb.WriteString("(?i)^")
	for _, r := range pattern {
		switch r {
		case '*':
			sb.WriteString(".*")
		case '?':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")
	re, err := regexp.Compile(sb.String())
	if err != nil {
		return false
	}
	return re.MatchString(name)
}

// buildZonesListQuery translates the list flags into /api/zones/list query
// parameters. It returns nil when no flag is set so a plain `tdns list`
// sends the same request as before these options existed.
func buildZonesListQuery(filterName, filterType string, page, perPage int) url.Values {
	q := url.Values{}
	if filterName != "" {
		q.Set("filterName", filterName)
	}
	if filterType != "" {
		q.Set("filterType", filterType)
	}
	if page > 0 {
		q.Set("pageNumber", strconv.Itoa(page))
	}
	if perPage > 0 {
		q.Set("zonesPerPage", strconv.Itoa(perPage))
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// filterZones applies filterName/filterType to zones client-side. Servers
// v15.3+ already filter, making this a no-op; older servers ignore the
// parameters and return everything, so this keeps the flags working there.
func filterZones(zones []interface{}, filterName, filterType string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(zones))
	for _, z := range zones {
		zone, ok := z.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := zone["name"].(string)
		if filterName != "" && !matchWildcard(filterName, name) {
			continue
		}
		if filterType != "" {
			if zt, _ := zone["type"].(string); !strings.EqualFold(zt, filterType) {
				continue
			}
		}
		out = append(out, zone)
	}
	return out
}

// formatZonesList renders the zones/list response body. Fields are read
// tolerantly so responses from older or newer server versions render
// without panicking. showFooter adds the pagination summary when the
// response carries one.
func formatZonesList(response map[string]interface{}, filterName, filterType string, showFooter bool) string {
	bold := color.New(color.Bold).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()
	amber := color.New(color.FgYellow).SprintFunc()

	rawZones, _ := response["zones"].([]interface{})
	zones := filterZones(rawZones, filterName, filterType)

	names := make([]string, 0, len(zones))
	zoneMap := make(map[string]map[string]interface{})
	for _, zone := range zones {
		name, _ := zone["name"].(string)
		names = append(names, name)
		zoneMap[name] = zone
	}
	sort.Strings(names)

	var sb strings.Builder
	if len(names) == 0 {
		sb.WriteString("No zones found.\n")
	}

	for _, name := range names {
		zone := zoneMap[name]
		ztype, _ := zone["type"].(string)

		status := green("Enabled")
		if disabled, _ := zone["disabled"].(bool); disabled {
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

		health := ""
		if expired, _ := zone["isExpired"].(bool); expired {
			health += " | " + red("Expired")
		}
		if failed, _ := zone["syncFailed"].(bool); failed {
			health += " | " + red("Sync failed")
		}
		if failed, _ := zone["notifyFailed"].(bool); failed {
			health += " | " + amber("Notify failed")
		}

		fmt.Fprintf(&sb, "%s (%s)\n", bold(name), blue(ztype))
		if modified, ok := zone["lastModified"].(string); ok {
			fmt.Fprintf(&sb, "  Last Modified: %s\n", modified)
		}
		if serial, ok := zone["soaSerial"].(float64); ok {
			fmt.Fprintf(&sb, "  SOA Serial: %d\n", int64(serial))
		}
		fmt.Fprintf(&sb, "  Status: %s | %s%s\n", status, scope, health)
		fmt.Fprintf(&sb, "  DNSSEC: %s\n", dnssec)
		sb.WriteString("\n")
	}

	if showFooter {
		if totalPages, ok := response["totalPages"].(float64); ok {
			pageNumber, _ := response["pageNumber"].(float64)
			totalZones, _ := response["totalZones"].(float64)
			fmt.Fprintf(&sb, "%s\n", gray(fmt.Sprintf("Page %.0f/%.0f | %.0f zones total", pageNumber, totalPages, totalZones)))
		}
	}

	return sb.String()
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List DNS zones, optionally filtered by name or type",
	Long: `List DNS zones.

Zones can be filtered by name (--name, supporting * and ? wildcards) and by
type (--type). Servers v15.3+ filter server-side; on older servers the same
filter is applied client-side, so the flags work either way. Results can be
paginated with --page and --per-page.`,
	Run: func(cmd *cobra.Command, args []string) {
		filterType := ""
		if listFilterType != "" {
			ct, ok := canonicalZoneType(listFilterType)
			if !ok {
				fmt.Fprintf(os.Stderr, "❌ invalid zone type %q (valid: %s)\n", listFilterType, strings.Join(zoneTypes, ", "))
				os.Exit(1)
			}
			filterType = ct
		}

		q := buildZonesListQuery(listFilterName, filterType, listPage, listPerPage)
		result, response, err := api.New().GetJSON("/api/zones/list", q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if listJSON {
			raw, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(raw))
			return
		}

		showFooter := listPage > 0 || listPerPage > 0
		fmt.Print(formatZonesList(response, listFilterName, filterType, showFooter))
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output raw JSON response")
	listCmd.Flags().StringVarP(&listFilterName, "name", "n", "", "Filter zones by name; supports * and ? wildcards")
	listCmd.Flags().StringVarP(&listFilterType, "type", "y", "", fmt.Sprintf("Filter zones by type (%s)", strings.Join(zoneTypes, ", ")))
	listCmd.Flags().IntVar(&listPage, "page", 0, "Page number of paginated results (default: all zones)")
	listCmd.Flags().IntVar(&listPerPage, "per-page", 0, "Zones per page; server default 10 when --page is set")
	rootCmd.AddCommand(listCmd)
}
