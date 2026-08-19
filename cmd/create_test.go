package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
)

// runCreateCmd executes `tdns create <args>` against a stub server and returns
// the query values the server received.
func runCreateCmd(t *testing.T, args ...string) map[string][]string {
	t.Helper()

	var gotQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok","response":{"domain":"example.com"}}`)
	}))
	defer srv.Close()

	oldHost := viper.GetString("host")
	viper.Set("host", srv.URL)
	defer viper.Set("host", oldHost)

	oldStdout := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	defer func() { os.Stdout = oldStdout }()
	go func() { _, _ = io.Copy(io.Discard, rp) }()

	rootCmd.SetArgs(append([]string{"create"}, args...))
	defer rootCmd.SetArgs(nil)
	// Flags persist across Execute calls; restore the declared default.
	defer func() { _ = createCmd.Flags().Set("useSoaSerialDateScheme", "true") }()
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create command failed: %v", err)
	}
	wp.Close()

	return gotQuery
}

// useSoaSerialDateScheme was declared as a string flag but read with GetBool,
// so it always resolved to false regardless of what the user passed.
func TestCreateCmdUseSoaSerialDateScheme(t *testing.T) {
	if got := runCreateCmd(t, "example.com")["useSoaSerialDateScheme"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("default useSoaSerialDateScheme = %v, want true", got)
	}
	if got := runCreateCmd(t, "example.com", "--useSoaSerialDateScheme=false")["useSoaSerialDateScheme"]; len(got) != 1 || got[0] != "false" {
		t.Errorf("useSoaSerialDateScheme=false = %v, want false", got)
	}
	if got := runCreateCmd(t, "example.com", "--useSoaSerialDateScheme=true")["useSoaSerialDateScheme"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("useSoaSerialDateScheme=true = %v, want true", got)
	}
}
