// Package satori is a Go client SDK for the Satori service.
//
// It is a port of the Satori .NET SDK (Satori/ApiClient.gen.cs and
// Satori/Client.cs).
package satori

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default per-request timeout.
	DefaultTimeout = 10 * time.Second
	// DefaultExpiredWindow is the window before token expiry within which a
	// refresh is automatically attempted.
	DefaultExpiredWindow = 5 * time.Minute
)

// FlagValueChangeReasonType identifies the kind of configuration that
// produced an override.
type FlagValueChangeReasonType int

const (
	FlagReasonUnknown      FlagValueChangeReasonType = 0
	FlagReasonFlagVariant  FlagValueChangeReasonType = 1
	FlagReasonLiveEvent    FlagValueChangeReasonType = 2
	FlagReasonExperiment   FlagValueChangeReasonType = 3
)

// ApiFlagOverrideType identifies the kind of override that affects a flag.
type ApiFlagOverrideType int

const (
	OverrideFlag                       ApiFlagOverrideType = 0
	OverrideFlagVariant                ApiFlagOverrideType = 1
	OverrideLiveEventFlag              ApiFlagOverrideType = 2
	OverrideLiveEventFlagVariant       ApiFlagOverrideType = 3
	OverrideExperimentPhaseVariantFlag ApiFlagOverrideType = 4
)

// ApiLiveEventStatus describes a live event's current state.
type ApiLiveEventStatus int

const (
	LiveEventUnknown    ApiLiveEventStatus = 0
	LiveEventActive     ApiLiveEventStatus = 1
	LiveEventUpcoming   ApiLiveEventStatus = 2
	LiveEventTerminated ApiLiveEventStatus = 3
)

// ApiResponseError is returned for non-2xx HTTP responses.
type ApiResponseError struct {
	StatusCode     int    `json:"-"`
	GrpcStatusCode int    `json:"code"`
	Message        string `json:"message"`
}

func (e *ApiResponseError) Error() string {
	return fmt.Sprintf("ApiResponseError(StatusCode=%d, Message='%s', GrpcStatusCode=%d)",
		e.StatusCode, e.Message, e.GrpcStatusCode)
}

// ===== DTOs =====

type FlagValueChangeReason struct {
	Name        string                    `json:"name,omitempty"`
	Type        FlagValueChangeReasonType `json:"type,omitempty"`
	VariantName string                    `json:"variant_name,omitempty"`
}

type ApiAuthenticateLogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	Token        string `json:"token,omitempty"`
}

type ApiAuthenticateRefreshRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type ApiAuthenticateRequest struct {
	Custom    map[string]string `json:"custom,omitempty"`
	Default   map[string]string `json:"default,omitempty"`
	Id        string            `json:"id,omitempty"`
	NoSession bool              `json:"no_session,omitempty"`
}

type ApiEvent struct {
	Id               string            `json:"id,omitempty"`
	IdentityId       string            `json:"identity_id,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Name             string            `json:"name,omitempty"`
	SessionExpiresAt string            `json:"session_expires_at,omitempty"`
	SessionId        string            `json:"session_id,omitempty"`
	SessionIssuedAt  string            `json:"session_issued_at,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
	Value            string            `json:"value,omitempty"`
}

type ApiEventRequest struct {
	Events []*ApiEvent `json:"events,omitempty"`
}

type ApiExperiment struct {
	Labels           []string `json:"labels,omitempty"`
	Name             string   `json:"name,omitempty"`
	PhaseName        string   `json:"phase_name,omitempty"`
	PhaseVariantName string   `json:"phase_variant_name,omitempty"`
	Value            string   `json:"value,omitempty"`
}

type ApiExperimentList struct {
	Experiments []*ApiExperiment `json:"experiments,omitempty"`
}

type ApiFlag struct {
	ChangeReason     *FlagValueChangeReason `json:"change_reason,omitempty"`
	ConditionChanged bool                   `json:"condition_changed,omitempty"`
	Labels           []string               `json:"labels,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Value            string                 `json:"value,omitempty"`
}

type ApiFlagList struct {
	Flags []*ApiFlag `json:"flags,omitempty"`
}

type ApiFlagOverrideValue struct {
	CreateTimeSec string              `json:"create_time_sec,omitempty"`
	Name          string              `json:"name,omitempty"`
	Type          ApiFlagOverrideType `json:"type,omitempty"`
	Value         string              `json:"value,omitempty"`
	VariantName   string              `json:"variant_name,omitempty"`
}

type ApiFlagOverride struct {
	FlagName  string                  `json:"flag_name,omitempty"`
	Labels    []string                `json:"labels,omitempty"`
	Overrides []*ApiFlagOverrideValue `json:"overrides,omitempty"`
}

type ApiFlagOverrideList struct {
	Flags []*ApiFlagOverride `json:"flags,omitempty"`
}

type ApiMessage struct {
	ConsumeTime string            `json:"consume_time,omitempty"`
	CreateTime  string            `json:"create_time,omitempty"`
	Id          string            `json:"id,omitempty"`
	ImageUrl    string            `json:"image_url,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ReadTime    string            `json:"read_time,omitempty"`
	ScheduleId  string            `json:"schedule_id,omitempty"`
	SendTime    string            `json:"send_time,omitempty"`
	Text        string            `json:"text,omitempty"`
	Title       string            `json:"title,omitempty"`
	UpdateTime  string            `json:"update_time,omitempty"`
}

type ApiGetMessageListResponse struct {
	CacheableCursor string        `json:"cacheable_cursor,omitempty"`
	Messages        []*ApiMessage `json:"messages,omitempty"`
	NextCursor      string        `json:"next_cursor,omitempty"`
	PrevCursor      string        `json:"prev_cursor,omitempty"`
}

type ApiIdentifyRequest struct {
	Custom  map[string]string `json:"custom,omitempty"`
	Default map[string]string `json:"default,omitempty"`
	Id      string            `json:"id,omitempty"`
}

type ApiLiveEvent struct {
	ActiveEndTimeSec   string             `json:"active_end_time_sec,omitempty"`
	ActiveStartTimeSec string             `json:"active_start_time_sec,omitempty"`
	Description        string             `json:"description,omitempty"`
	DurationSec        string             `json:"duration_sec,omitempty"`
	EndTimeSec         string             `json:"end_time_sec,omitempty"`
	Id                 string             `json:"id,omitempty"`
	Labels             []string           `json:"labels,omitempty"`
	Name               string             `json:"name,omitempty"`
	ResetCron          string             `json:"reset_cron,omitempty"`
	StartTimeSec       string             `json:"start_time_sec,omitempty"`
	Status             ApiLiveEventStatus `json:"status,omitempty"`
}

type ApiLiveEventList struct {
	ExplicitJoinLiveEvents []*ApiLiveEvent `json:"explicit_join_live_events,omitempty"`
	LiveEvents             []*ApiLiveEvent `json:"live_events,omitempty"`
}

type ApiProperties struct {
	Computed map[string]string `json:"computed,omitempty"`
	Custom   map[string]string `json:"custom,omitempty"`
	Default  map[string]string `json:"default,omitempty"`
}

type ApiSession struct {
	Properties   *ApiProperties `json:"properties,omitempty"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	Token        string         `json:"token,omitempty"`
}

type ApiUpdateMessageRequest struct {
	ConsumeTime string `json:"consume_time,omitempty"`
	ReadTime    string `json:"read_time,omitempty"`
}

type ApiUpdatePropertiesRequest struct {
	Custom    map[string]string `json:"custom,omitempty"`
	Default   map[string]string `json:"default,omitempty"`
	Recompute bool              `json:"recompute,omitempty"`
}

// ===== Session =====

// Session represents an authenticated Satori session.
type Session struct {
	authToken         string
	refreshToken      string
	createTime        int64
	expireTime        int64
	refreshExpireTime int64
	identityId        string
	properties        *ApiProperties
}

func (s *Session) AuthToken() string             { return s.authToken }
func (s *Session) RefreshToken() string          { return s.refreshToken }
func (s *Session) CreateTime() int64             { return s.createTime }
func (s *Session) ExpireTime() int64             { return s.expireTime }
func (s *Session) RefreshExpireTime() int64      { return s.refreshExpireTime }
func (s *Session) IdentityId() string            { return s.identityId }
func (s *Session) Properties() *ApiProperties    { return s.properties }
func (s *Session) IsExpired() bool               { return s.HasExpired(time.Now().UTC()) }
func (s *Session) IsRefreshExpired() bool        { return s.HasRefreshExpired(time.Now().UTC()) }
func (s *Session) HasExpired(at time.Time) bool {
	return at.After(time.Unix(s.expireTime, 0).UTC())
}
func (s *Session) HasRefreshExpired(at time.Time) bool {
	return at.After(time.Unix(s.refreshExpireTime, 0).UTC())
}

// Update applies a fresh authentication response to the session.
func (s *Session) Update(authToken, refreshToken string, properties *ApiProperties) error {
	s.authToken = authToken
	s.refreshToken = refreshToken
	s.properties = properties
	claims, err := jwtUnpack(authToken)
	if err != nil {
		return err
	}
	if v, ok := claims["exp"]; ok {
		s.expireTime = toInt64(v)
	}
	if v, ok := claims["iat"]; ok {
		s.createTime = toInt64(v)
	}
	if v, ok := claims["iid"]; ok {
		s.identityId = fmt.Sprint(v)
	}
	if refreshToken != "" {
		rclaims, err := jwtUnpack(refreshToken)
		if err == nil {
			if v, ok := rclaims["exp"]; ok {
				s.refreshExpireTime = toInt64(v)
			}
		}
	}
	return nil
}

// NewSession constructs a session from a fresh ApiSession response.
func NewSession(api *ApiSession) (*Session, error) {
	if api == nil {
		return nil, errors.New("satori: nil session")
	}
	s := &Session{createTime: time.Now().UTC().Unix()}
	if err := s.Update(api.Token, api.RefreshToken, api.Properties); err != nil {
		return nil, err
	}
	return s, nil
}

// RestoreSession recreates a session from previously-issued tokens.
func RestoreSession(authToken, refreshToken string) (*Session, error) {
	if authToken == "" {
		return nil, errors.New("satori: empty auth token")
	}
	return NewSession(&ApiSession{Token: authToken, RefreshToken: refreshToken})
}

func jwtUnpack(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, errors.New("satori: invalid JWT")
	}
	payload := parts[1]
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	claims := map[string]any{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		i, _ := x.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(x, 10, 64)
		return i
	default:
		return 0
	}
}

// ===== Logger =====

// Logger is the interface used by the Satori SDK to emit log messages.
type Logger interface {
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}

// NullLogger discards all log messages.
type NullLogger struct{}

func (NullLogger) Debugf(string, ...any) {}
func (NullLogger) Errorf(string, ...any) {}
func (NullLogger) Infof(string, ...any)  {}
func (NullLogger) Warnf(string, ...any)  {}

// ===== Client =====

// Client is the high-level Satori client, mirroring Satori/Client.cs.
type Client struct {
	baseURL  *url.URL
	apiKey   string
	httpClient *http.Client
	timeout  time.Duration
	logger   Logger
	autoRefresh bool
}

// ClientOption customises the behaviour of NewClient.
type ClientOption func(*Client)

// WithHttpClient overrides the http.Client used for requests.
func WithHttpClient(h *http.Client) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithLogger overrides the logger.
func WithLogger(l Logger) ClientOption {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithTimeout overrides the request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithAutoRefresh disables automatic session refresh.
func WithAutoRefresh(enabled bool) ClientOption {
	return func(c *Client) {
		c.autoRefresh = enabled
	}
}

// NewClient builds a Client targeting the supplied URL.
//
// scheme should be "http" or "https"; host is the host name or IP; port is
// the TCP port. apiKey is the Satori API key issued by the dashboard.
func NewClient(scheme, host string, port int, apiKey string, opts ...ClientOption) *Client {
	if scheme == "" {
		scheme = "http"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		if scheme == "https" {
			port = 443
		} else {
			port = 7450
		}
	}
	c := &Client{
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", host, port),
		},
		apiKey:    apiKey,
		httpClient: http.DefaultClient,
		timeout:   DefaultTimeout,
		logger:    NullLogger{},
		autoRefresh: true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewClientWithURL builds a Client from a base URL.
func NewClientWithURL(u *url.URL, apiKey string, opts ...ClientOption) *Client {
	if u == nil {
		return NewClient("", "", 0, apiKey, opts...)
	}
	c := &Client{
		baseURL:    u,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
		timeout:    DefaultTimeout,
		logger:     NullLogger{},
		autoRefresh: true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetLogger replaces the logger.
func (c *Client) SetLogger(l Logger) {
	if l == nil {
		l = NullLogger{}
	}
	c.logger = l
}

// SetTimeout overrides the request timeout.
func (c *Client) SetTimeout(d time.Duration) { c.timeout = d }

// internal HTTP plumbing
type queryBuilder struct{ parts []string }

func (q *queryBuilder) addString(name, value string) {
	q.parts = append(q.parts, name+"="+url.QueryEscape(value))
}
func (q *queryBuilder) addStrings(name string, values []string) {
	for _, v := range values {
		q.addString(name, v)
	}
}
func (q *queryBuilder) addInt(name string, value int) {
	q.parts = append(q.parts, name+"="+strconv.Itoa(value))
}
func (q *queryBuilder) addIntPtr(name string, v *int) {
	if v != nil {
		q.addInt(name, *v)
	}
}
func (q *queryBuilder) addBool(name string, v bool) {
	q.parts = append(q.parts, name+"="+strconv.FormatBool(v))
}
func (q *queryBuilder) addBoolPtr(name string, v *bool) {
	if v != nil {
		q.addBool(name, *v)
	}
}
func (q *queryBuilder) String() string { return strings.Join(q.parts, "&") }

func (c *Client) buildURL(path, query string) *url.URL {
	u := *c.baseURL
	prefix := strings.TrimRight(c.baseURL.Path, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = prefix + path
	u.RawQuery = query
	return &u
}

func (c *Client) do(ctx context.Context, method, path, query string, headers map[string]string, body any, target any) error {
	uri := c.buildURL(path, query)
	var bodyReader io.Reader
	if body != nil {
		var bodyBytes []byte
		switch b := body.(type) {
		case []byte:
			bodyBytes = b
		case string:
			bodyBytes = []byte(b)
		default:
			marshalled, err := json.Marshal(b)
			if err != nil {
				return fmt.Errorf("satori: cannot marshal body: %w", err)
			}
			bodyBytes = marshalled
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, method, uri.String(), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.logger.Debugf("Sending %s %s", method, uri.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	c.logger.Debugf("Received %d %s", resp.StatusCode, string(respBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &ApiResponseError{StatusCode: resp.StatusCode, Message: string(respBytes)}
		var parsed struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if jerr := json.Unmarshal(respBytes, &parsed); jerr == nil && parsed.Message != "" {
			apiErr.GrpcStatusCode = parsed.Code
			apiErr.Message = parsed.Message
		}
		return apiErr
	}
	if target == nil || len(respBytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBytes, target); err != nil {
		return fmt.Errorf("satori: cannot decode response: %w", err)
	}
	return nil
}

func (c *Client) basicHeaders() map[string]string {
	creds := base64.StdEncoding.EncodeToString([]byte(c.apiKey + ":"))
	return map[string]string{"Authorization": "Basic " + creds}
}

func (c *Client) bearerHeaders(token string) map[string]string {
	if token == "" {
		return map[string]string{}
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

func (c *Client) ensureValidSession(ctx context.Context, session *Session) error {
	if !c.autoRefresh || session == nil || session.RefreshToken() == "" {
		return nil
	}
	if !session.HasExpired(time.Now().UTC().Add(DefaultExpiredWindow)) {
		return nil
	}
	_, err := c.SessionRefresh(ctx, session)
	return err
}

// ===== Operations =====

// Authenticate authenticates an identity against the server. NoSession=true
// will perform a property update without returning a session.
func (c *Client) Authenticate(ctx context.Context, id string, defaultProps, customProps map[string]string) (*Session, error) {
	body := &ApiAuthenticateRequest{Id: id, Default: defaultProps, Custom: customProps}
	api := &ApiSession{}
	if err := c.do(ctx, "POST", "/v1/authenticate", "", c.basicHeaders(), body, api); err != nil {
		return nil, err
	}
	return NewSession(api)
}

// AuthenticateLogout invalidates the supplied session and refresh tokens.
func (c *Client) AuthenticateLogout(ctx context.Context, session *Session) error {
	if session == nil {
		return errors.New("satori: session is nil")
	}
	body := &ApiAuthenticateLogoutRequest{Token: session.AuthToken(), RefreshToken: session.RefreshToken()}
	return c.do(ctx, "POST", "/v1/authenticate/logout", "", c.bearerHeaders(session.AuthToken()), body, nil)
}

// SessionRefresh refreshes a session using its refresh token.
func (c *Client) SessionRefresh(ctx context.Context, session *Session) (*Session, error) {
	if session == nil || session.RefreshToken() == "" {
		return nil, errors.New("satori: refresh token required")
	}
	body := &ApiAuthenticateRefreshRequest{RefreshToken: session.RefreshToken()}
	api := &ApiSession{}
	if err := c.do(ctx, "POST", "/v1/authenticate/refresh", "", c.basicHeaders(), body, api); err != nil {
		return nil, err
	}
	if err := session.Update(api.Token, api.RefreshToken, api.Properties); err != nil {
		return nil, err
	}
	return session, nil
}

// Identify enriches the session with a new identity id.
func (c *Client) Identify(ctx context.Context, session *Session, id string, defaultProps, customProps map[string]string) (*Session, error) {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return nil, err
	}
	body := &ApiIdentifyRequest{Id: id, Default: defaultProps, Custom: customProps}
	api := &ApiSession{}
	if err := c.do(ctx, "PUT", "/v1/identify", "", c.bearerHeaders(session.AuthToken()), body, api); err != nil {
		return nil, err
	}
	return NewSession(api)
}

// DeleteIdentity deletes the caller's identity and associated data.
func (c *Client) DeleteIdentity(ctx context.Context, session *Session) error {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "DELETE", "/v1/identity", "", c.bearerHeaders(session.AuthToken()), nil, nil)
}

// Event publishes events on behalf of the session's identity.
func (c *Client) Event(ctx context.Context, session *Session, events []*ApiEvent) error {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "POST", "/v1/event", "", c.bearerHeaders(session.AuthToken()), &ApiEventRequest{Events: events}, nil)
}

// ServerEvent publishes events for multiple identities at once. Requires the
// API key (no session).
func (c *Client) ServerEvent(ctx context.Context, events []*ApiEvent) error {
	return c.do(ctx, "POST", "/v1/server-event", "", c.basicHeaders(), &ApiEventRequest{Events: events}, nil)
}

// GetExperiments lists or queries experiments visible to the identity.
func (c *Client) GetExperiments(ctx context.Context, session *Session, names, labels []string) (*ApiExperimentList, error) {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return nil, err
	}
	q := queryBuilder{}
	q.addStrings("names", names)
	q.addStrings("labels", labels)
	out := &ApiExperimentList{}
	if err := c.do(ctx, "GET", "/v1/experiment", q.String(), c.bearerHeaders(session.AuthToken()), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetFlags returns flags for the identity (or default flags via API key).
func (c *Client) GetFlags(ctx context.Context, session *Session, names, labels []string) (*ApiFlagList, error) {
	q := queryBuilder{}
	q.addStrings("names", names)
	q.addStrings("labels", labels)
	headers := c.basicHeaders()
	if session != nil {
		if err := c.ensureValidSession(ctx, session); err != nil {
			return nil, err
		}
		headers = c.bearerHeaders(session.AuthToken())
	}
	out := &ApiFlagList{}
	if err := c.do(ctx, "GET", "/v1/flag", q.String(), headers, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetFlagOverrides returns flag overrides visible to the identity (or
// defaults via API key).
func (c *Client) GetFlagOverrides(ctx context.Context, session *Session, names, labels []string) (*ApiFlagOverrideList, error) {
	q := queryBuilder{}
	q.addStrings("names", names)
	q.addStrings("labels", labels)
	headers := c.basicHeaders()
	if session != nil {
		if err := c.ensureValidSession(ctx, session); err != nil {
			return nil, err
		}
		headers = c.bearerHeaders(session.AuthToken())
	}
	out := &ApiFlagOverrideList{}
	if err := c.do(ctx, "GET", "/v1/flag/override", q.String(), headers, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetLiveEvents lists live events visible to the identity. limits and
// time-window filters are optional.
func (c *Client) GetLiveEvents(ctx context.Context, session *Session, names, labels []string, pastRunCount, futureRunCount *int, startTimeSec, endTimeSec string) (*ApiLiveEventList, error) {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return nil, err
	}
	q := queryBuilder{}
	q.addStrings("names", names)
	q.addStrings("labels", labels)
	q.addIntPtr("past_run_count", pastRunCount)
	q.addIntPtr("future_run_count", futureRunCount)
	if startTimeSec != "" {
		q.addString("start_time_sec", startTimeSec)
	}
	if endTimeSec != "" {
		q.addString("end_time_sec", endTimeSec)
	}
	out := &ApiLiveEventList{}
	if err := c.do(ctx, "GET", "/v1/live-event", q.String(), c.bearerHeaders(session.AuthToken()), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// JoinLiveEvent joins an explicit-join live event.
func (c *Client) JoinLiveEvent(ctx context.Context, session *Session, id string) error {
	if id == "" {
		return errors.New("satori: 'id' is required")
	}
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "POST", "/v1/live-event/"+url.PathEscape(id)+"/participation", "", c.bearerHeaders(session.AuthToken()), nil, nil)
}

// GetMessageList lists messages for the identity.
func (c *Client) GetMessageList(ctx context.Context, session *Session, limit *int, forward *bool, cursor string, messageIds []string) (*ApiGetMessageListResponse, error) {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return nil, err
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addBoolPtr("forward", forward)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	q.addStrings("message_ids", messageIds)
	out := &ApiGetMessageListResponse{}
	if err := c.do(ctx, "GET", "/v1/message", q.String(), c.bearerHeaders(session.AuthToken()), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteMessage deletes a message owned by the identity.
func (c *Client) DeleteMessage(ctx context.Context, session *Session, id string) error {
	if id == "" {
		return errors.New("satori: 'id' is required")
	}
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "DELETE", "/v1/message/"+url.PathEscape(id), "", c.bearerHeaders(session.AuthToken()), nil, nil)
}

// UpdateMessage updates the read or consume time of a message.
func (c *Client) UpdateMessage(ctx context.Context, session *Session, id string, body *ApiUpdateMessageRequest) error {
	if id == "" {
		return errors.New("satori: 'id' is required")
	}
	if body == nil {
		return errors.New("satori: 'body' is required")
	}
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "PUT", "/v1/message/"+url.PathEscape(id), "", c.bearerHeaders(session.AuthToken()), body, nil)
}

// ListProperties returns the identity's properties.
func (c *Client) ListProperties(ctx context.Context, session *Session) (*ApiProperties, error) {
	if err := c.ensureValidSession(ctx, session); err != nil {
		return nil, err
	}
	out := &ApiProperties{}
	if err := c.do(ctx, "GET", "/v1/properties", "", c.bearerHeaders(session.AuthToken()), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateProperties replaces or merges properties on the identity.
func (c *Client) UpdateProperties(ctx context.Context, session *Session, body *ApiUpdatePropertiesRequest) error {
	if body == nil {
		return errors.New("satori: 'body' is required")
	}
	if err := c.ensureValidSession(ctx, session); err != nil {
		return err
	}
	return c.do(ctx, "PUT", "/v1/properties", "", c.bearerHeaders(session.AuthToken()), body, nil)
}

// Healthcheck pings the server.
func (c *Client) Healthcheck(ctx context.Context, session *Session) error {
	headers := c.basicHeaders()
	if session != nil {
		headers = c.bearerHeaders(session.AuthToken())
	}
	return c.do(ctx, "GET", "/healthcheck", "", headers, nil, nil)
}

// Readycheck pings the server's readiness endpoint.
func (c *Client) Readycheck(ctx context.Context, session *Session) error {
	headers := c.basicHeaders()
	if session != nil {
		headers = c.bearerHeaders(session.AuthToken())
	}
	return c.do(ctx, "GET", "/readycheck", "", headers, nil, nil)
}
