package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"tdns/internal/api"
)

func TestBuildImportQuery(t *testing.T) {
	// Defaults must keep sending the parameters previous releases sent, and
	// must not mention overwriteZone at all.
	q := buildImportQuery("example.com", true, false, true)
	for k, want := range map[string]string{
		"zone":               "example.com",
		"overwrite":          "true",
		"overwriteSoaSerial": "true",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
	if _, ok := q["overwriteZone"]; ok {
		t.Errorf("overwriteZone should be omitted when disabled, got %v", q)
	}

	q = buildImportQuery("example.com", false, true, false)
	for k, want := range map[string]string{
		"overwrite":          "false",
		"overwriteZone":      "true",
		"overwriteSoaSerial": "false",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

func TestIsZoneExistsError(t *testing.T) {
	if !isZoneExistsError(&api.APIError{Message: "Zone already exists: example.com"}) {
		t.Error("should match the API's zone-exists rejection")
	}
	if !isZoneExistsError(fmt.Errorf("wrapped: %w", &api.APIError{Message: "Zone already exists: example.com"})) {
		t.Error("should match through a wrapped error")
	}
	if isZoneExistsError(&api.APIError{Message: "Access was denied."}) {
		t.Error("should not match unrelated API errors")
	}
	if isZoneExistsError(errors.New("Zone already exists: example.com")) {
		t.Error("should only match *api.APIError, not any error mentioning it")
	}
	if isZoneExistsError(nil) {
		t.Error("nil is not a zone-exists error")
	}
}

// importRequest records what the stub server received for one API call.
type importRequest struct {
	path  string
	query map[string][]string
	body  string
}

// runImportCmd executes `tdns import <args>` against a stub server and returns
// the requests it received, in order.
func runImportCmd(t *testing.T, handler func(r *http.Request) string, args ...string) []importRequest {
	t.Helper()

	var got []importRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, importRequest{
			path:  r.URL.Path,
			query: r.URL.Query(),
			body:  string(body),
		})
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, handler(r))
	}))
	defer srv.Close()

	oldHost := viper.GetString("host")
	viper.Set("host", srv.URL)
	defer viper.Set("host", oldHost)

	// Reset flag-bound package vars from any previous invocation.
	importJSON = false
	importOverwrite = true
	importOverwriteZone = false
	importOverwriteSoaSerial = true
	importCreate = false
	importCreateType = "Primary"
	assumeYes = false

	// Silence the command's stdout while it runs.
	oldStdout := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	defer func() { os.Stdout = oldStdout }()
	go func() { _, _ = io.Copy(io.Discard, rp) }()

	rootCmd.SetArgs(append([]string{"import"}, args...))
	defer rootCmd.SetArgs(nil)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import command failed: %v", err)
	}
	wp.Close()

	return got
}

// writeZoneFile creates a throwaway zone file and returns its path.
func writeZoneFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "zone.txt")
	contents := "example.com. 3600 IN NS ns1.example.com.\n"
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write zone file: %v", err)
	}
	return path
}

// okResponse answers every call with success, reporting a server version new
// enough for --overwrite-zone.
func okResponse(r *http.Request) string {
	if strings.HasSuffix(r.URL.Path, "/api/settings/get") {
		return `{"status":"ok","response":{"version":"15.0.1"}}`
	}
	return `{"status":"ok","response":{}}`
}

func TestImportCmdDefaultsMatchPreviousBehaviour(t *testing.T) {
	reqs := runImportCmd(t, okResponse, "example.com", "--file", writeZoneFile(t))
	if len(reqs) != 1 {
		t.Fatalf("got %d requests, want 1: %v", len(reqs), reqs)
	}
	if reqs[0].path != "/api/zones/import" {
		t.Errorf("path = %q, want /api/zones/import", reqs[0].path)
	}
	for k, want := range map[string]string{
		"zone":               "example.com",
		"overwrite":          "true",
		"overwriteSoaSerial": "true",
	} {
		if got := reqs[0].query[k]; len(got) != 1 || got[0] != want {
			t.Errorf("query %s = %v, want %q", k, got, want)
		}
	}
	if _, ok := reqs[0].query["overwriteZone"]; ok {
		t.Error("overwriteZone must not be sent by default")
	}
	if !strings.Contains(reqs[0].body, "ns1.example.com") {
		t.Errorf("zone file was not posted as the body: %q", reqs[0].body)
	}
}

// findRequest returns the first recorded request for path.
func findRequest(t *testing.T, reqs []importRequest, path string) importRequest {
	t.Helper()
	for _, r := range reqs {
		if r.path == path {
			return r
		}
	}
	t.Fatalf("no request to %s, got %v", path, reqs)
	return importRequest{}
}

func TestImportCmdOverwriteZone(t *testing.T) {
	reqs := runImportCmd(t, okResponse,
		"example.com", "--file", writeZoneFile(t), "--overwrite-zone", "--yes")

	// The version gate must run before the destructive import.
	if len(reqs) != 2 || reqs[0].path != "/api/settings/get" {
		t.Fatalf("want a version check followed by the import, got %v", reqs)
	}
	imp := findRequest(t, reqs, "/api/zones/import")
	if got := imp.query["overwriteZone"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("overwriteZone = %v, want true", got)
	}
}

func TestImportCmdSkipsVersionCheckWithoutOverwriteZone(t *testing.T) {
	reqs := runImportCmd(t, okResponse, "example.com", "--file", writeZoneFile(t))
	for _, r := range reqs {
		if r.path == "/api/settings/get" {
			t.Error("version check should only run for --overwrite-zone")
		}
	}
}

func TestImportCmdProceedsWhenVersionUnavailable(t *testing.T) {
	// A token that cannot read settings must not block the import; the command
	// warns and carries on rather than refusing outright.
	handler := func(r *http.Request) string {
		if strings.HasSuffix(r.URL.Path, "/api/settings/get") {
			return `{"status":"error","errorMessage":"Access was denied."}`
		}
		return `{"status":"ok","response":{}}`
	}
	reqs := runImportCmd(t, handler,
		"example.com", "--file", writeZoneFile(t), "--overwrite-zone", "--yes")
	imp := findRequest(t, reqs, "/api/zones/import")
	if got := imp.query["overwriteZone"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("overwriteZone = %v, want true", got)
	}
}

func TestImportCmdOverwriteFlagsCanBeDisabled(t *testing.T) {
	reqs := runImportCmd(t, okResponse, "example.com", "--file", writeZoneFile(t),
		"--overwrite=false", "--overwrite-soa-serial=false")
	if len(reqs) != 1 {
		t.Fatalf("got %d requests, want 1: %v", len(reqs), reqs)
	}
	for k, want := range map[string]string{
		"overwrite":          "false",
		"overwriteSoaSerial": "false",
	} {
		if got := reqs[0].query[k]; len(got) != 1 || got[0] != want {
			t.Errorf("query %s = %v, want %q", k, got, want)
		}
	}
}

func TestImportCmdCreateCreatesZoneFirst(t *testing.T) {
	reqs := runImportCmd(t, okResponse,
		"example.com", "--file", writeZoneFile(t), "--create")
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2 (create then import): %v", len(reqs), reqs)
	}
	if reqs[0].path != "/api/zones/create" {
		t.Errorf("first call = %q, want /api/zones/create", reqs[0].path)
	}
	for k, want := range map[string]string{
		"zone":                   "example.com",
		"type":                   "Primary",
		"useSoaSerialDateScheme": "false",
	} {
		if got := reqs[0].query[k]; len(got) != 1 || got[0] != want {
			t.Errorf("create query %s = %v, want %q", k, got, want)
		}
	}
	if reqs[1].path != "/api/zones/import" {
		t.Errorf("second call = %q, want /api/zones/import", reqs[1].path)
	}
}

func TestImportCmdCreateIsIdempotent(t *testing.T) {
	// An existing zone must not abort the import.
	handler := func(r *http.Request) string {
		if strings.HasSuffix(r.URL.Path, "/create") {
			return `{"status":"error","errorMessage":"Zone already exists: example.com"}`
		}
		return `{"status":"ok","response":{}}`
	}
	reqs := runImportCmd(t, handler,
		"example.com", "--file", writeZoneFile(t), "--create")
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2: %v", len(reqs), reqs)
	}
	if reqs[1].path != "/api/zones/import" {
		t.Errorf("import must still run after a zone-exists error, got %q", reqs[1].path)
	}
}

func TestImportCmdCreateForwarderType(t *testing.T) {
	reqs := runImportCmd(t, okResponse,
		"example.com", "--file", writeZoneFile(t), "--create", "--type", "Forwarder")
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2: %v", len(reqs), reqs)
	}
	if got := reqs[0].query["type"]; len(got) != 1 || got[0] != "Forwarder" {
		t.Errorf("create type = %v, want Forwarder", got)
	}
}
