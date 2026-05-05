package nakama

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TransientErrorFunc decides whether an error returned by the HTTP adapter
// should be retried. Mirrors Nakama/TransientExceptionDelegate.cs.
type TransientErrorFunc func(err error) bool

// HttpAdapter is the abstraction over HTTP transport used by the SDK. It is a
// port of Nakama/IHttpAdapter.cs.
type HttpAdapter interface {
	// Send executes an HTTP request and returns the response body as a string.
	// uri must be the fully qualified target URL.
	Send(ctx context.Context, method string, uri *url.URL, headers map[string]string, body []byte, timeout time.Duration) (string, error)

	// TransientError reports whether the supplied error should be considered
	// retriable.
	TransientError() TransientErrorFunc

	// Logger returns the logger configured on this adapter.
	Logger() Logger

	// SetLogger replaces the logger.
	SetLogger(Logger)
}

// DefaultHttpAdapter is an HttpAdapter backed by net/http with optional gzip
// compression of request bodies.
type DefaultHttpAdapter struct {
	Client     *http.Client
	UseGzip    bool
	logger     Logger
	transient  TransientErrorFunc
}

// NewHttpAdapter returns an HttpAdapter with sensible defaults.
func NewHttpAdapter() *DefaultHttpAdapter {
	return &DefaultHttpAdapter{
		Client:    http.DefaultClient,
		UseGzip:   false,
		logger:    NullLogger{},
		transient: defaultTransientErrorFunc,
	}
}

// NewHttpAdapterWithGzip returns an HttpAdapter that gzips its request bodies.
func NewHttpAdapterWithGzip() *DefaultHttpAdapter {
	a := NewHttpAdapter()
	a.UseGzip = true
	return a
}

// Logger returns the configured logger.
func (a *DefaultHttpAdapter) Logger() Logger {
	if a.logger == nil {
		return NullLogger{}
	}
	return a.logger
}

// SetLogger replaces the logger.
func (a *DefaultHttpAdapter) SetLogger(l Logger) {
	if l == nil {
		l = NullLogger{}
	}
	a.logger = l
}

// TransientError returns the transient-error predicate.
func (a *DefaultHttpAdapter) TransientError() TransientErrorFunc {
	if a.transient == nil {
		return defaultTransientErrorFunc
	}
	return a.transient
}

// SetTransientErrorFunc replaces the transient-error predicate.
func (a *DefaultHttpAdapter) SetTransientErrorFunc(f TransientErrorFunc) {
	a.transient = f
}

// Send sends an HTTP request and returns the response body as a string.
func (a *DefaultHttpAdapter) Send(ctx context.Context, method string, uri *url.URL, headers map[string]string, body []byte, timeout time.Duration) (string, error) {
	if uri == nil {
		return "", errors.New("nakama: nil request uri")
	}

	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	encoded := body
	gzippedBody := false
	if len(body) > 0 && a.UseGzip {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(body); err != nil {
			return "", err
		}
		if err := gz.Close(); err != nil {
			return "", err
		}
		encoded = buf.Bytes()
		gzippedBody = true
	}
	if len(encoded) > 0 {
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(cctx, method, uri.String(), bodyReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if gzippedBody {
		req.Header.Set("Content-Encoding", "gzip")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	a.Logger().DebugFormat("Sending %s %s", method, uri.String())

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respText := string(respBytes)

	a.Logger().DebugFormat("Received %d %s", resp.StatusCode, respText)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &ApiResponseError{StatusCode: resp.StatusCode, Message: respText}
		// Attempt to parse a JSON error payload.
		var parsed struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if jerr := json.Unmarshal(respBytes, &parsed); jerr == nil && parsed.Message != "" {
			apiErr.GrpcStatusCode = parsed.Code
			apiErr.Message = parsed.Message
		}
		return "", apiErr
	}

	return respText, nil
}

// defaultTransientErrorFunc treats network-level errors and 5xx responses as
// transient. 408 and 429 are also retried.
func defaultTransientErrorFunc(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var apiErr *ApiResponseError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == 408 || apiErr.StatusCode == 429
	}
	// network-level errors are transient.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection") || strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "eof") || strings.Contains(msg, "reset") {
		return true
	}
	return false
}

// urlEscape returns a URI-safe version of the input string.
func urlEscape(s string) string {
	return url.QueryEscape(s)
}

// joinURLValues concatenates name=value pairs separated by '&'.
// values that are nil are skipped.
func joinURLValues(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "&")
}

// formatURL builds a URL by combining a base URL, a path, and a query string.
func formatURL(base *url.URL, path string, rawQuery string) *url.URL {
	u := *base
	prefix := strings.TrimRight(base.Path, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = prefix + path
	u.RawQuery = rawQuery
	return &u
}

// urlMustParse panics if u cannot be parsed; intended for static URLs.
func urlMustParse(u string) *url.URL {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(fmt.Sprintf("nakama: invalid URL %q: %v", u, err))
	}
	return parsed
}
