package mudrex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://trade.mudrex.com"
	defaultAPIPrefix = "/fapi/v1"
	defaultTimeout   = 10 * time.Second
	defaultMaxRetry  = 3
)

type httpClient struct {
	apiSecret   string
	baseURL     string
	apiPrefix   string
	timeout     time.Duration
	maxRetries  int
	logRequests bool
	http        *http.Client
}

type clientConfig struct {
	apiSecret   string
	tradeCurr   string
	timeout     time.Duration
	maxRetries  int
	logRequests bool
	baseURL     string
	httpClient  *http.Client
	skipPing    bool
}

func newHTTPClient(cfg clientConfig) (*httpClient, error) {
	apiSecret := cfg.apiSecret
	if apiSecret == "" {
		apiSecret = os.Getenv("MUDREX_API_SECRET")
	}
	if apiSecret == "" {
		return nil, fmt.Errorf("API secret is required. Pass APISecret or set the MUDREX_API_SECRET environment variable")
	}

	timeout := cfg.timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	maxRetries := cfg.maxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetry
	}

	baseURL := cfg.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpTransport := cfg.httpClient
	if httpTransport == nil {
		httpTransport = &http.Client{Timeout: timeout}
	}

	client := &httpClient{
		apiSecret:   apiSecret,
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiPrefix:   defaultAPIPrefix,
		timeout:     timeout,
		maxRetries:  maxRetries,
		logRequests: cfg.logRequests,
		http:        httpTransport,
	}

	if !cfg.skipPing {
		if err := client.ping(); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *httpClient) ping() error {
	endpoint := c.baseURL + c.apiPrefix + "/futures/ping"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	c.setDefaultHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return &MudrexRequestError{
			MudrexError:   MudrexError{Message: fmt.Sprintf("Cannot reach Mudrex API: %v", err)},
			OriginalError: err,
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		message := "Invalid API secret"
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if errorsArr, ok := payload["errors"].([]any); ok && len(errorsArr) > 0 {
				if first, ok := errorsArr[0].(map[string]any); ok {
					if text, ok := first["text"].(string); ok && text != "" {
						message = text
					}
				}
			}
		}
		return &MudrexAPIError{
			MudrexError: MudrexError{Message: message},
			Code:        http.StatusUnauthorized,
			StatusCode:  resp.StatusCode,
			Body:        string(body),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &MudrexAPIError{
			MudrexError: MudrexError{Message: fmt.Sprintf("Ping failed with status %d", resp.StatusCode)},
			Code:        resp.StatusCode,
			StatusCode:  resp.StatusCode,
			Body:        string(body),
		}
	}

	return nil
}

func (c *httpClient) setDefaultHeaders(req *http.Request) {
	req.Header.Set("X-Authentication", c.apiSecret)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Mudrex-SDK-Version", Version)
}

func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

func cleanParams(params map[string]any) url.Values {
	if len(params) == 0 {
		return nil
	}

	values := url.Values{}
	for key, value := range params {
		if isNilValue(value) {
			continue
		}
		switch v := value.(type) {
		case string:
			if v == "" && key != "is_symbol" {
				continue
			}
			values.Set(key, v)
		case *string:
			if *v == "" && key != "is_symbol" {
				continue
			}
			values.Set(key, *v)
		case bool:
			if v {
				values.Set(key, "true")
			} else {
				values.Set(key, "false")
			}
		case int:
			values.Set(key, fmt.Sprintf("%d", v))
		case int64:
			values.Set(key, fmt.Sprintf("%d", v))
		case float64:
			values.Set(key, fmt.Sprintf("%v", v))
		default:
			values.Set(key, fmt.Sprintf("%v", v))
		}
	}

	if len(values) == 0 {
		return nil
	}
	return values
}

func prepareBody(body map[string]any) map[string]any {
	if len(body) == 0 {
		return nil
	}

	prepared := make(map[string]any, len(body))
	for key, value := range body {
		if isNilValue(value) {
			continue
		}
		prepared[key] = value
	}

	if len(prepared) == 0 {
		return nil
	}
	return prepared
}

func (c *httpClient) request(method, path string, params map[string]any, body map[string]any) (any, error) {
	query := cleanParams(params)
	preparedBody := prepareBody(body)

	endpoint := c.baseURL + c.apiPrefix + path
	if query != nil {
		endpoint += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if preparedBody != nil {
		encoded, err := json.Marshal(preparedBody)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(encoded)
	}

	if c.logRequests {
		log.Printf("mudrex: %s %s params=%v body=%v", method, endpoint, params, preparedBody)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequest(method, endpoint, bodyReader)
		if err != nil {
			return nil, err
		}
		c.setDefaultHeaders(req)
		if bodyReader != nil && (method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete) {
			req.Header.Set("Content-Type", "application/json")
		}

		if bodyReader != nil {
			if seeker, ok := bodyReader.(*bytes.Reader); ok {
				seeker.Seek(0, io.SeekStart)
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				continue
			}
			return nil, &MudrexRequestError{
				MudrexError:   MudrexError{Message: err.Error()},
				OriginalError: err,
			}
		}

		result, handleErr := handleResponse(resp)
		resp.Body.Close()
		return result, handleErr
	}

	return nil, &MudrexRequestError{
		MudrexError:   MudrexError{Message: lastErr.Error()},
		OriginalError: lastErr,
	}
}

func parseErrorPayload(data map[string]any, statusCode int) (string, int) {
	if errorsArr, ok := data["errors"].([]any); ok && len(errorsArr) > 0 {
		if first, ok := errorsArr[0].(map[string]any); ok {
			message := "Unknown error"
			if text, ok := first["text"].(string); ok && text != "" {
				message = text
			}
			code := statusCode
			switch c := first["code"].(type) {
			case float64:
				code = int(c)
			case int:
				code = c
			}
			return message, code
		}
	}

	if msgRaw, ok := data["message"].(string); ok && strings.TrimSpace(msgRaw) != "" {
		var nested map[string]any
		if err := json.Unmarshal([]byte(msgRaw), &nested); err == nil {
			return parseErrorPayload(nested, statusCode)
		}
		return strings.TrimSpace(msgRaw), statusCode
	}

	return fmt.Sprintf("Request failed with status %d", statusCode), statusCode
}

func handleResponse(resp *http.Response) (any, error) {
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawText := strings.TrimSpace(string(rawBody))
	status := resp.StatusCode

	if rawText == "" {
		if status >= 200 && status < 300 {
			return nil, nil
		}
		if status == http.StatusTooManyRequests {
			return nil, &MudrexAPIError{
				MudrexError: MudrexError{Message: "API rate limit exceeded"},
				Code:        http.StatusTooManyRequests,
				StatusCode:  status,
			}
		}
		return nil, &MudrexAPIError{
			MudrexError: MudrexError{Message: fmt.Sprintf("Empty response body (HTTP %d)", status)},
			Code:        status,
			StatusCode:  status,
		}
	}

	var data map[string]any
	if err := json.Unmarshal(rawBody, &data); err != nil {
		if status == http.StatusTooManyRequests {
			return nil, &MudrexAPIError{
				MudrexError: MudrexError{Message: "API rate limit exceeded"},
				Code:        http.StatusTooManyRequests,
				StatusCode:  status,
			}
		}
		preview := rawText
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, &MudrexAPIError{
			MudrexError: MudrexError{Message: fmt.Sprintf("Invalid JSON response: %s", preview)},
			Code:        status,
			StatusCode:  status,
			Body:        rawText,
		}
	}

	success, _ := data["success"].(bool)
	if status >= 400 || !success {
		message, code := parseErrorPayload(data, status)
		return nil, &MudrexAPIError{
			MudrexError: MudrexError{Message: message},
			Code:        code,
			StatusCode:  status,
			Body:        rawText,
		}
	}

	result := data["data"]
	switch typed := result.(type) {
	case map[string]any:
		return responseFromMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			if obj, ok := item.(map[string]any); ok {
				respObj, err := responseFromMap(obj)
				if err != nil {
					return nil, err
				}
				out[i] = respObj
			} else {
				out[i] = item
			}
		}
		return out, nil
	default:
		wrapped, err := responseFromMap(map[string]any{"result": result})
		if err != nil {
			return nil, err
		}
		return wrapped, nil
	}
}

func (c *httpClient) get(path string, params map[string]any) (any, error) {
	return c.request(http.MethodGet, path, params, nil)
}

func (c *httpClient) post(path string, params map[string]any, body map[string]any) (any, error) {
	return c.request(http.MethodPost, path, params, body)
}

func (c *httpClient) patch(path string, body map[string]any) (any, error) {
	return c.request(http.MethodPatch, path, nil, body)
}

func (c *httpClient) delete(path string) (any, error) {
	return c.request(http.MethodDelete, path, nil, nil)
}
