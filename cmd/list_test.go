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

// runListCmd executes `tdns list <args>` against a stub server and returns
// the query values the server received.
func runListCmd(t *testing.T, args ...string) map[string][]string {
	t.Helper()

	var gotQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
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

	return gotQuery
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
