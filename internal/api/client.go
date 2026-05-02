// Package api is a thin HTTP client for the Technitium DNS Server API.
//
// It targets v15+, sending the session token via the
// `Authorization: Bearer <token>` header. For older servers (or endpoints
// that haven't been updated to honor the header), set viper key
// `legacy_token` (CLI flag `--legacy-token`) to also emit the token as a
// `token=` query/form parameter.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// DefaultTimeout is used when neither the `timeout` viper key nor a custom
// HTTP client is provided. It bounds total request time including connect,
// TLS handshake, redirects, and reading the response body.
const DefaultTimeout = 5 * time.Second

// Client talks to a Technitium DNS Server HTTP API.
type Client struct {
	Host        string
	Token       string
	HTTP        *http.Client
	Timeout     time.Duration
	LegacyToken bool // when true, also append `token=` to the query string
}

// New builds a Client from viper config (host, token, legacy_token, timeout).
func New() *Client {
	timeout := DefaultTimeout
	if d := viper.GetDuration("timeout"); d > 0 {
		timeout = d
	}
	return &Client{
		Host:        viper.GetString("host"),
		Token:       viper.GetString("token"),
		LegacyToken: viper.GetBool("legacy_token"),
		Timeout:     timeout,
		HTTP:        &http.Client{Timeout: timeout},
	}
}

func (c *Client) buildURL(path string, q url.Values) string {
	host := strings.TrimRight(c.Host, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if c.LegacyToken && c.Token != "" {
		if q == nil {
			q = url.Values{}
		}
		if q.Get("token") == "" {
			q.Set("token", c.Token)
		}
	}
	if len(q) == 0 {
		return host + path
	}
	return host + path + "?" + q.Encode()
}

func (c *Client) newRequest(method, path string, q url.Values, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.buildURL(path, q), body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return req, nil
}

// Do executes a pre-built request after attaching the Bearer auth header.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.Token != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.do(req)
}

// Get issues a GET against the API.
func (c *Client) Get(path string, q url.Values) (*http.Response, error) {
	req, err := c.newRequest(http.MethodGet, path, q, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Post issues a POST with the given body.
func (c *Client) Post(path string, q url.Values, body io.Reader, contentType string) (*http.Response, error) {
	req, err := c.newRequest(http.MethodPost, path, q, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.do(req)
}

// do is the single chokepoint where transport errors are inspected. It wraps
// timeout errors in a TimeoutError so the CLI can print a friendly message.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil && isTimeout(err) {
		return resp, &TimeoutError{Timeout: c.Timeout, Err: err}
	}
	return resp, err
}

// TimeoutError indicates the request exceeded the client timeout.
type TimeoutError struct {
	Timeout time.Duration
	Err     error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("request timed out after %s (set --timeout to override)", e.Timeout)
}

func (e *TimeoutError) Unwrap() error { return e.Err }

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	var ue *url.Error
	if errors.As(err, &ue) && ue.Timeout() {
		return true
	}
	type timeoutIface interface{ Timeout() bool }
	var t timeoutIface
	if errors.As(err, &t) && t.Timeout() {
		return true
	}
	return false
}

// APIError is returned when the API responds with status != "ok".
type APIError struct {
	Status  string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unexpected API error"
}

// GetJSON issues a GET, decodes the envelope, and returns both the full
// envelope and the inner "response" object. It returns *APIError when the
// server reports a non-ok status.
func (c *Client) GetJSON(path string, q url.Values) (map[string]interface{}, map[string]interface{}, error) {
	resp, err := c.Get(path, q)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return decodeEnvelope(resp.Body)
}

// DoJSON executes the request and decodes the JSON envelope.
func (c *Client) DoJSON(req *http.Request) (map[string]interface{}, map[string]interface{}, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return decodeEnvelope(resp.Body)
}

// ServerVersion fetches the connected server's version string by calling
// /api/settings/get and reading its `version` field.
func (c *Client) ServerVersion() (string, error) {
	_, response, err := c.GetJSON("/api/settings/get", nil)
	if err != nil {
		return "", err
	}
	v, _ := response["version"].(string)
	if v == "" {
		return "", fmt.Errorf("server did not return a version field")
	}
	return v, nil
}

func decodeEnvelope(r io.Reader) (map[string]interface{}, map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("invalid response: %w", err)
	}
	status, _ := result["status"].(string)
	if status != "ok" {
		msg, _ := result["errorMessage"].(string)
		return result, nil, &APIError{Status: status, Message: msg}
	}
	response, _ := result["response"].(map[string]interface{})
	return result, response, nil
}
