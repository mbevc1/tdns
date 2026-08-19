package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"15.0", "15.0", 0},
		{"15.0", "15.0.0", 0}, // missing components count as zero
		{"15.0.1", "15.0", 1},
		{"14.9.9", "15.0", -1},
		{"15.1", "15.0", 1},
		{"9.0", "15.0", -1}, // numeric, not lexicographic
		{"15.0.1.2", "15.0.1", 1},
		{"15.0-beta", "15.0", 0}, // suffixes are ignored
		{"v15.0", "15.0", 0},     // a leading "v" is tolerated
	}
	for _, tt := range tests {
		got, err := CompareVersions(tt.a, tt.b)
		if err != nil {
			t.Errorf("CompareVersions(%q, %q): %v", tt.a, tt.b, err)
			continue
		}
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}

	if _, err := CompareVersions("", "15.0"); err == nil {
		t.Error("empty version should error")
	}
	if _, err := CompareVersions("unknown", "15.0"); err == nil {
		t.Error("non-numeric version should error")
	}
}

func TestServerAtLeast(t *testing.T) {
	serve := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
	}

	srv := serve(`{"status":"ok","response":{"version":"15.0.1"}}`)
	defer srv.Close()
	ok, version, err := (&Client{Host: srv.URL, HTTP: srv.Client()}).ServerAtLeast("15.0")
	if err != nil || !ok || version != "15.0.1" {
		t.Errorf("ServerAtLeast = %v, %q, %v; want true, \"15.0.1\", nil", ok, version, err)
	}

	old := serve(`{"status":"ok","response":{"version":"14.1"}}`)
	defer old.Close()
	ok, version, err = (&Client{Host: old.URL, HTTP: old.Client()}).ServerAtLeast("15.0")
	if err != nil || ok || version != "14.1" {
		t.Errorf("ServerAtLeast = %v, %q, %v; want false, \"14.1\", nil", ok, version, err)
	}

	denied := serve(`{"status":"error","errorMessage":"Access was denied."}`)
	defer denied.Close()
	if ok, _, err = (&Client{Host: denied.URL, HTTP: denied.Client()}).ServerAtLeast("15.0"); err == nil || ok {
		t.Errorf("ServerAtLeast on an error response = %v, %v; want false and an error", ok, err)
	}
}
