package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "example.org", false},
		{"EXAMPLE.com", "example.COM", true}, // case-insensitive
		{"*", "anything.example.com", true},
		{"*", "", true},
		{"*.example.com", "sub.example.com", true},
		{"*.example.com", "example.com", false},
		{"example*", "example.com", true},
		{"exampl?.com", "example.com", true},
		{"exampl?.com", "exampl.com", false}, // ? is exactly one char
		{"ex?mple.*", "example.org", true},
		{"a+b.com", "a+b.com", true},  // regex metachars in pattern are literal
		{"a+b.com", "aab.com", false}, // and must not act as regex
		{"", "", true},
		{"", "example.com", false},
	}
	for _, tt := range tests {
		if got := matchWildcard(tt.pattern, tt.name); got != tt.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestCanonicalZoneType(t *testing.T) {
	for in, want := range map[string]string{
		"primary":            "Primary",
		"SECONDARY":          "Secondary",
		"secondaryforwarder": "SecondaryForwarder",
		"Catalog":            "Catalog",
	} {
		got, ok := canonicalZoneType(in)
		if !ok || got != want {
			t.Errorf("canonicalZoneType(%q) = %q, %v; want %q, true", in, got, ok, want)
		}
	}
	if _, ok := canonicalZoneType("bogus"); ok {
		t.Errorf("canonicalZoneType(\"bogus\") should not match")
	}
}

func TestBuildZonesListQuery(t *testing.T) {
	if q := buildZonesListQuery("", "", 0, 0); q != nil {
		t.Errorf("expected nil query when no options are set, got %v", q)
	}
	q := buildZonesListQuery("ex*", "Primary", 2, 5)
	for k, want := range map[string]string{
		"filterName":   "ex*",
		"filterType":   "Primary",
		"pageNumber":   "2",
		"zonesPerPage": "5",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

func TestFilterZonesClientSideFallback(t *testing.T) {
	// Simulates a pre-v15.3 server that ignored filterName/filterType and
	// returned every zone.
	zones := []interface{}{
		map[string]interface{}{"name": "example.com", "type": "Primary"},
		map[string]interface{}{"name": "sub.example.com", "type": "Secondary"},
		map[string]interface{}{"name": "other.org", "type": "Primary"},
		"not-a-zone-object",
	}

	got := filterZones(zones, "*.example.com", "")
	if len(got) != 1 || got[0]["name"] != "sub.example.com" {
		t.Errorf("name filter: got %v, want only sub.example.com", got)
	}

	got = filterZones(zones, "", "Primary")
	if len(got) != 2 {
		t.Errorf("type filter: got %d zones, want 2", len(got))
	}

	got = filterZones(zones, "*example*", "primary")
	if len(got) != 1 || got[0]["name"] != "example.com" {
		t.Errorf("combined filter: got %v, want only example.com", got)
	}

	if got = filterZones(zones, "", ""); len(got) != 3 {
		t.Errorf("no filter: got %d zones, want 3", len(got))
	}
}

func TestFormatZonesListTolerant(t *testing.T) {
	// Minimal zone objects (e.g. from a hypothetical older/newer server)
	// must render without panicking.
	response := map[string]interface{}{
		"zones": []interface{}{
			map[string]interface{}{"name": "bare.example.com"},
		},
	}
	out := formatZonesList(response, "", "", false)
	if !strings.Contains(out, "bare.example.com") {
		t.Errorf("output missing zone name: %q", out)
	}
	if strings.Contains(out, "SOA Serial") || strings.Contains(out, "Last Modified") {
		t.Errorf("output should omit absent fields: %q", out)
	}

	// A response without a zones field must not panic either.
	out = formatZonesList(map[string]interface{}{}, "", "", false)
	if !strings.Contains(out, "No zones found.") {
		t.Errorf("expected empty-list message, got %q", out)
	}
}

func TestFormatZonesListFull(t *testing.T) {
	response := map[string]interface{}{
		"pageNumber": float64(1),
		"totalPages": float64(2),
		"totalZones": float64(12),
		"zones": []interface{}{
			map[string]interface{}{
				"name":         "example.com",
				"type":         "Secondary",
				"dnssecStatus": "SignedWithNSEC",
				"soaSerial":    float64(7),
				"lastModified": "2022-02-26T07:57:08.1842183Z",
				"disabled":     true,
				"isExpired":    true,
				"syncFailed":   true,
				"notifyFailed": true,
			},
		},
	}
	out := formatZonesList(response, "", "", true)
	for _, want := range []string{
		"example.com", "Secondary", "SignedWithNSEC", "SOA Serial: 7",
		"Last Modified: 2022-02-26T07:57:08.1842183Z", "Disabled",
		"Expired", "Sync failed", "Notify failed", "Page 1/2 | 12 zones total",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// Footer suppressed when pagination flags were not used.
	if out := formatZonesList(response, "", "", false); strings.Contains(out, "zones total") {
		t.Errorf("footer should be hidden when showFooter=false:\n%s", out)
	}
}

// listRequest records what the stub server received for one API call.
type listRequest struct {
	path  string
	query map[string][]string
}

// runListCmd executes `tdns list <args>` against a stub server reporting a
// v15.3+ version, and returns the query values of the zones/list call.
func runListCmd(t *testing.T, args ...string) map[string][]string {
	t.Helper()
	for _, r := range runListCmdRequests(t, "15.3", args...) {
		if r.path == "/api/zones/list" {
			return r.query
		}
	}
	t.Fatal("no request to /api/zones/list")
	return nil
}

// runListCmdRequests executes `tdns list <args>` against a stub server that
// reports serverVersion, and returns every request it received in order. An
// empty serverVersion omits the field, standing in for a server whose version
// cannot be read.
func runListCmdRequests(t *testing.T, serverVersion string, args ...string) []listRequest {
	t.Helper()

	var got []listRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, listRequest{path: r.URL.Path, query: r.URL.Query()})
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/api/settings/get") {
			fmt.Fprintf(w, `{"status":"ok","response":{"version":%q}}`, serverVersion)
			return
		}
		fmt.Fprint(w, `{"status":"ok","response":{"zones":[]}}`)
	}))
	defer srv.Close()

	oldHost := viper.GetString("host")
	viper.Set("host", srv.URL)
	defer viper.Set("host", oldHost)

	// Reset flag-bound package vars from any previous invocation.
	listJSON = false
	listFilterName = ""
	listFilterType = ""
	listPage = 0
	listPerPage = 0

	// Silence the command's stdout while it runs.
	oldStdout := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	defer func() { os.Stdout = oldStdout }()
	go func() { _, _ = io.Copy(io.Discard, rp) }()

	rootCmd.SetArgs(append([]string{"list"}, args...))
	defer rootCmd.SetArgs(nil)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	wp.Close()

	return got
}

// paths returns the request paths in the order they were received.
func paths(reqs []listRequest) []string {
	out := make([]string, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, r.path)
	}
	return out
}

func TestListCmdChecksVersionOnlyWhenFilteringPaginatedResults(t *testing.T) {
	// The version lookup is an extra round-trip, so it must not happen unless a
	// filter and pagination are combined.
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"plain list", nil},
		{"filter only", []string{"--name", "ex*"}},
		{"type filter only", []string{"--type", "Primary"}},
		{"pagination only", []string{"--page", "2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := paths(runListCmdRequests(t, "15.3", tt.args...))
			if len(got) != 1 || got[0] != "/api/zones/list" {
				t.Errorf("requests = %v, want only /api/zones/list", got)
			}
		})
	}

	for _, args := range [][]string{
		{"--name", "ex*", "--page", "2"},
		{"--type", "Primary", "--per-page", "5"},
	} {
		got := paths(runListCmdRequests(t, "15.3", args...))
		want := []string{"/api/settings/get", "/api/zones/list"}
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("`list %v` requests = %v, want %v", args, got, want)
		}
	}
}

func TestListCmdWarnsWhenServerCannotFilterPaginatedResults(t *testing.T) {
	// The listing itself must still be requested and rendered; the warning is
	// advisory, not a refusal.
	for _, version := range []string{"15.2", "14.0", ""} {
		reqs := runListCmdRequests(t, version, "--name", "ex*", "--page", "2")
		if got := paths(reqs); len(got) == 0 || got[len(got)-1] != "/api/zones/list" {
			t.Errorf("server v%q: requests = %v, want the listing to still run", version, got)
		}
		for _, r := range reqs {
			if r.path != "/api/zones/list" {
				continue
			}
			if got := r.query["filterName"]; len(got) != 1 || got[0] != "ex*" {
				t.Errorf("server v%q: filterName = %v, want it sent anyway", version, got)
			}
		}
	}
}

func TestListCmdSendsNoParamsByDefault(t *testing.T) {
	q := runListCmd(t)
	if len(q) != 0 {
		t.Errorf("plain `tdns list` should send no query params, got %v", q)
	}
}

func TestListCmdSendsFilterAndPaginationParams(t *testing.T) {
	q := runListCmd(t, "--name", "ex*", "--type", "primary", "--page", "2", "--per-page", "5")
	for k, want := range map[string]string{
		"filterName":   "ex*",
		"filterType":   "Primary", // canonicalized from "primary"
		"pageNumber":   "2",
		"zonesPerPage": "5",
	} {
		if got := q[k]; len(got) != 1 || got[0] != want {
			t.Errorf("query %s = %v, want %q", k, got, want)
		}
	}
}
