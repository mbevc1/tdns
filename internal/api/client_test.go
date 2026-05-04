package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		path   string
		token  string
		legacy bool
		query  url.Values
		want   string
	}{
		{
			name: "trailing slash trimmed",
			host: "http://localhost:5380/",
			path: "/api/x",
			want: "http://localhost:5380/api/x",
		},
		{
			name: "leading slash added to path",
			host: "http://localhost:5380",
			path: "api/x",
			want: "http://localhost:5380/api/x",
		},
		{
			name:  "query encoded",
			host:  "http://h",
			path:  "/p",
			query: url.Values{"a": []string{"1 2"}},
			want:  "http://h/p?a=1+2",
		},
		{
			name:   "legacy token appended",
			host:   "http://h",
			path:   "/p",
			token:  "secret",
			legacy: true,
			want:   "http://h/p?token=secret",
		},
		{
			name:   "legacy token not overriding existing",
			host:   "http://h",
			path:   "/p",
			token:  "secret",
			legacy: true,
			query:  url.Values{"token": []string{"keep"}},
			want:   "http://h/p?token=keep",
		},
		{
			name:   "legacy off does not append",
			host:   "http://h",
			path:   "/p",
			token:  "secret",
			legacy: false,
			want:   "http://h/p",
		},
		{
			name:   "legacy on but empty token",
			host:   "http://h",
			path:   "/p",
			token:  "",
			legacy: true,
			want:   "http://h/p",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{Host: tt.host, Token: tt.token, LegacyToken: tt.legacy}
			got := c.buildURL(tt.path, tt.query)
			if got != tt.want {
				t.Errorf("buildURL = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSendsBearerHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, Token: "tok", HTTP: srv.Client(), Timeout: time.Second}
	resp, err := c.Get("/api/x", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
}

func TestGetNoTokenNoAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	resp, err := c.Get("/api/x", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestDoPreservesExistingAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, Token: "newtok", HTTP: srv.Client(), Timeout: time.Second}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/x", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer existing")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "Bearer existing" {
		t.Errorf("Authorization = %q, want preserved %q", gotAuth, "Bearer existing")
	}
}

func TestDoAddsAuthWhenMissing(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, Token: "tok", HTTP: srv.Client(), Timeout: time.Second}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/x", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
}

func TestPostSetsContentTypeAndBody(t *testing.T) {
	var (
		gotCT   string
		gotBody string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	resp, err := c.Post("/x", nil, strings.NewReader("hello"), "application/json")
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	resp.Body.Close()
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotCT, "application/json")
	}
	if gotBody != "hello" {
		t.Errorf("body = %q, want %q", gotBody, "hello")
	}
}

func TestPostOmitsContentTypeWhenEmpty(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	resp, err := c.Post("/x", nil, nil, "")
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	resp.Body.Close()
	if gotCT != "" {
		t.Errorf("Content-Type = %q, want empty", gotCT)
	}
}

func TestGetJSONOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","response":{"foo":"bar"}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	envelope, response, err := c.GetJSON("/x", nil)
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if envelope["status"] != "ok" {
		t.Errorf("envelope status = %v", envelope["status"])
	}
	if response["foo"] != "bar" {
		t.Errorf("response[foo] = %v, want bar", response["foo"])
	}
}

func TestGetJSONAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"error","errorMessage":"nope"}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	_, _, err := c.GetJSON("/x", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Error() != "nope" {
		t.Errorf("APIError.Error() = %q, want %q", apiErr.Error(), "nope")
	}
	if apiErr.Status != "error" {
		t.Errorf("APIError.Status = %q", apiErr.Status)
	}
}

func TestAPIErrorFallbackMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"error"}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	_, _, err := c.GetJSON("/x", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Error() != "unexpected API error" {
		t.Errorf("APIError.Error() = %q, want fallback", apiErr.Error())
	}
}

func TestGetJSONMalformed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	_, _, err := c.GetJSON("/x", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid response") {
		t.Errorf("expected invalid response error, got %v", err)
	}
}

func TestServerVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/settings/get" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`{"status":"ok","response":{"version":"13.6"}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	v, err := c.ServerVersion()
	if err != nil {
		t.Fatalf("ServerVersion: %v", err)
	}
	if v != "13.6" {
		t.Errorf("ServerVersion = %q, want %q", v, "13.6")
	}
}

func TestServerVersionMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","response":{}}`))
	}))
	defer srv.Close()

	c := &Client{Host: srv.URL, HTTP: srv.Client(), Timeout: time.Second}
	_, err := c.ServerVersion()
	if err == nil {
		t.Fatal("expected error when version is missing")
	}
}

func TestTimeoutErrorWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	timeout := 30 * time.Millisecond
	c := &Client{
		Host:    srv.URL,
		HTTP:    &http.Client{Timeout: timeout},
		Timeout: timeout,
	}
	_, err := c.Get("/x", nil)
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeoutError, got %T (%v)", err, err)
	}
	if te.Timeout != timeout {
		t.Errorf("TimeoutError.Timeout = %v, want %v", te.Timeout, timeout)
	}
	if !strings.Contains(te.Error(), timeout.String()) {
		t.Errorf("TimeoutError.Error() = %q, missing timeout duration", te.Error())
	}
	if te.Unwrap() == nil {
		t.Error("TimeoutError.Unwrap() returned nil")
	}
}

func TestIsTimeoutFalseOnNil(t *testing.T) {
	if isTimeout(nil) {
		t.Error("isTimeout(nil) = true, want false")
	}
}

func TestNewFromViper(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("host", "http://example:5380")
	viper.Set("token", "abc")
	viper.Set("legacy_token", true)
	viper.Set("timeout", 7*time.Second)

	c := New()
	if c.Host != "http://example:5380" {
		t.Errorf("Host = %q", c.Host)
	}
	if c.Token != "abc" {
		t.Errorf("Token = %q", c.Token)
	}
	if !c.LegacyToken {
		t.Error("LegacyToken = false, want true")
	}
	if c.Timeout != 7*time.Second {
		t.Errorf("Timeout = %v, want 7s", c.Timeout)
	}
	if c.HTTP == nil || c.HTTP.Timeout != 7*time.Second {
		t.Errorf("HTTP client timeout not propagated: %+v", c.HTTP)
	}
}

func TestNewDefaultsTimeout(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	c := New()
	if c.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want DefaultTimeout %v", c.Timeout, DefaultTimeout)
	}
	if c.HTTP.Timeout != DefaultTimeout {
		t.Errorf("HTTP.Timeout = %v, want %v", c.HTTP.Timeout, DefaultTimeout)
	}
}
