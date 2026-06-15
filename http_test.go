package mudrex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientInitWithAPISecret(t *testing.T) {
	client, err := newHTTPClient(clientConfig{
		apiSecret: "test_secret",
		skipPing:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.apiSecret != "test_secret" {
		t.Fatalf("got secret %q", client.apiSecret)
	}
}

func TestHTTPClientInitFromEnv(t *testing.T) {
	t.Setenv("MUDREX_API_SECRET", "env_secret")
	client, err := newHTTPClient(clientConfig{skipPing: true})
	if err != nil {
		t.Fatal(err)
	}
	if client.apiSecret != "env_secret" {
		t.Fatalf("got secret %q", client.apiSecret)
	}
}

func TestHTTPClientInitNoSecretRaises(t *testing.T) {
	t.Setenv("MUDREX_API_SECRET", "")
	_, err := newHTTPClient(clientConfig{skipPing: true})
	if err == nil || !strings.Contains(err.Error(), "API secret is required") {
		t.Fatalf("expected missing secret error, got %v", err)
	}
}

func TestHTTPClientInitDefaults(t *testing.T) {
	client, err := newHTTPClient(clientConfig{apiSecret: "s", skipPing: true})
	if err != nil {
		t.Fatal(err)
	}
	if client.timeout != defaultTimeout {
		t.Fatalf("timeout=%v", client.timeout)
	}
	if client.maxRetries != defaultMaxRetry {
		t.Fatalf("maxRetries=%d", client.maxRetries)
	}
	if client.logRequests {
		t.Fatal("expected logRequests false")
	}
}

func TestHTTPClientInitCustomParams(t *testing.T) {
	client, err := newHTTPClient(clientConfig{
		apiSecret:   "s",
		timeout:     30 * time.Second,
		maxRetries:  5,
		logRequests: true,
		skipPing:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.timeout != 30*time.Second || client.maxRetries != 5 || !client.logRequests {
		t.Fatalf("unexpected config: timeout=%v maxRetries=%d log=%v", client.timeout, client.maxRetries, client.logRequests)
	}
}

func TestCleanParams(t *testing.T) {
	if cleanParams(nil) != nil {
		t.Fatal("expected nil")
	}
	if cleanParams(map[string]any{}) != nil {
		t.Fatal("expected nil for empty map")
	}
	got := cleanParams(map[string]any{"a": 1, "b": nil, "c": "x"})
	if got.Get("a") != "1" || got.Get("c") != "x" || got.Has("b") {
		t.Fatalf("unexpected values: %v", got)
	}
	got = cleanParams(map[string]any{"is_symbol": "true", "x": nil})
	if got.Get("is_symbol") != "true" || got.Has("x") {
		t.Fatalf("unexpected values: %v", got)
	}
	got = cleanParams(map[string]any{"offset": 0})
	if got.Get("offset") != "0" {
		t.Fatalf("expected offset=0, got %v", got)
	}
	if cleanParams(map[string]any{"a": nil, "b": nil}) != nil {
		t.Fatal("expected nil when all values nil")
	}
}

func TestPrepareBody(t *testing.T) {
	if prepareBody(nil) != nil {
		t.Fatal("expected nil")
	}
	if prepareBody(map[string]any{}) != nil {
		t.Fatal("expected nil for empty map")
	}
	got := prepareBody(map[string]any{"a": 1, "b": nil})
	if len(got) != 1 || got["a"] != 1 {
		t.Fatalf("unexpected body: %v", got)
	}
	got = prepareBody(map[string]any{
		"qty":  "0.001",
		"lev":  10,
		"flag": true,
	})
	if got["qty"] != "0.001" || got["lev"] != 10 || got["flag"] != true {
		t.Fatalf("unexpected body: %v", got)
	}
	got = prepareBody(map[string]any{"reduce_only": false})
	if got["reduce_only"] != false {
		t.Fatal("expected false preserved")
	}
	if prepareBody(map[string]any{"a": nil}) != nil {
		t.Fatal("expected nil when all values nil")
	}
}

func TestHandleResponseSuccessDict(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]any{"id": "abc"},
	})
	result, err := handleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := result.(Response)
	if !ok {
		t.Fatalf("expected Response, got %T", result)
	}
	id, err := r.GetString("id")
	if err != nil || id != "abc" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestHandleResponseSuccessList(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": true,
		"data":    []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}},
	})
	result, err := handleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	items, ok := result.([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected list of 2, got %T", result)
	}
	if _, ok := items[0].(Response); !ok {
		t.Fatalf("expected Response item, got %T", items[0])
	}
}

func TestHandleResponseSuccessScalar(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": true,
		"data":    "62888.3",
	})
	result, err := handleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := result.(Response)
	if !ok {
		t.Fatalf("expected Response, got %T", result)
	}
	raw, ok := r.Result()
	if !ok || string(raw) != `"62888.3"` {
		t.Fatalf("result=%s ok=%v", raw, ok)
	}
}

func TestHandleResponseSuccessNilData(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": true,
		"data":    nil,
	})
	result, err := handleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := result.(Response)
	if !ok {
		t.Fatalf("expected Response, got %T", result)
	}
	raw, ok := r.Result()
	if !ok || string(raw) != "null" {
		t.Fatalf("result=%s ok=%v", raw, ok)
	}
}

func TestHandleResponseAPIErrorWithErrorsArray(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusBadRequest, map[string]any{
		"success": false,
		"errors":  []any{map[string]any{"code": 400, "text": "Bad request"}},
	})
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != 400 || apiErr.Message != "Bad request" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseAPIErrorWithoutErrorsArray(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusInternalServerError, map[string]any{"success": false})
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseSuccessFalse200(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": false,
		"errors":  []any{map[string]any{"code": 1001, "text": "Rate limited"}},
	})
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != 1001 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseInvalidJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("not json")),
	}
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || !strings.Contains(apiErr.Message, "Invalid JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseHTTPErrorNoSuccessField(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusForbidden, map[string]any{"error": "forbidden"})
	_, err := handleResponse(resp)
	if _, ok := err.(*MudrexAPIError); !ok {
		t.Fatalf("expected MudrexAPIError, got %v", err)
	}
}

func TestHandleResponseListWithNonDictItems(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusOK, map[string]any{
		"success": true,
		"data":    []any{"abc", 123, map[string]any{"id": "x"}},
	})
	result, err := handleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	items := result.([]any)
	if items[0] != "abc" || items[1] != float64(123) {
		t.Fatalf("unexpected items: %v", items)
	}
	if _, ok := items[2].(Response); !ok {
		t.Fatalf("expected Response at index 2, got %T", items[2])
	}
}

func TestHandleResponseEmptyBody204(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}
	result, err := handleResponse(resp)
	if err != nil || result != nil {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func TestHandleResponseEmptyBody429(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(""))}
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != 429 || apiErr.Message != "API rate limit exceeded" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseEmptyBody500(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || !strings.Contains(apiErr.Message, "Empty response body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponse429MalformedBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(strings.NewReader("not valid json")),
	}
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Message != "API rate limit exceeded" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleResponseDoubleEncodedErrorMessage(t *testing.T) {
	resp := makeJSONResponse(t, http.StatusTooManyRequests, map[string]any{
		"success": false,
		"message": `{"errors":[{"code":5002,"text":"API rate limit exceeded"}]}`,
	})
	_, err := handleResponse(resp)
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != 5002 || apiErr.Message != "API rate limit exceeded" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNetworkRetriesOnConnectionError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatal(err)
			}
			conn.Close()
			return
		}
		writeJSON(w, map[string]any{"success": true, "data": map[string]any{"ok": 1}})
	}))
	t.Cleanup(server.Close)

	client, err := newHTTPClient(clientConfig{
		apiSecret:  "s",
		baseURL:    server.URL,
		maxRetries: 2,
		skipPing:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.get("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	resp, ok := result.(Response)
	if !ok {
		t.Fatalf("expected Response, got %T", result)
	}
	val := 0
	if err := resp.Get("ok", &val); err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Fatalf("ok=%d", val)
	}
}

func TestPingSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/futures/ping" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeJSON(w, map[string]any{"code": 200, "text": "pong"})
	}))
	t.Cleanup(server.Close)

	client, err := newHTTPClient(clientConfig{apiSecret: "test_secret", baseURL: server.URL, skipPing: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ping(); err != nil {
		t.Fatal(err)
	}
}

func TestPingBadSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]any{
			"success": false,
			"errors":  []any{map[string]any{"text": "Invalid Authentication", "code": 3100}},
		})
	}))
	t.Cleanup(server.Close)

	client, err := newHTTPClient(clientConfig{apiSecret: "test_secret", baseURL: server.URL, skipPing: true})
	if err != nil {
		t.Fatal(err)
	}
	err = client.ping()
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != http.StatusUnauthorized || !strings.Contains(apiErr.Message, "Invalid Authentication") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPingUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, map[string]any{})
	}))
	t.Cleanup(server.Close)

	client, err := newHTTPClient(clientConfig{apiSecret: "test_secret", baseURL: server.URL, skipPing: true})
	if err != nil {
		t.Fatal(err)
	}
	err = client.ping()
	apiErr, ok := err.(*MudrexAPIError)
	if !ok || apiErr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResponseGetString(t *testing.T) {
	raw, err := responseFromMap(map[string]any{"order_id": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := raw.GetString("order_id")
	if err != nil || id != "abc" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func makeJSONResponse(t *testing.T, status int, body map[string]any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(data))),
	}
}

func responseDataForPath(path string) any {
	switch path {
	case "/fapi/v1/futures",
		"/fapi/v1/futures/orders",
		"/fapi/v1/futures/orders/history",
		"/fapi/v1/futures/positions",
		"/fapi/v1/futures/positions/history",
		"/fapi/v1/futures/fee/history":
		return []any{map[string]any{"id": "1"}}
	default:
		return map[string]any{"result": "ok"}
	}
}

func writeJSON(w http.ResponseWriter, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
