// Package nakama is a Go client SDK for the Nakama server.
//
// Client is a port of Nakama/Client.cs from the .NET SDK. It wraps the
// low-level HTTP api client with retry support, automatic session refresh,
// and a more idiomatic Go API.
package nakama

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"
)

func int64ToString(i int64) string { return strconv.FormatInt(i, 10) }

const (
	// DefaultHost is the default host address of the Nakama server.
	DefaultHost = "127.0.0.1"
	// DefaultScheme is the default scheme for HTTP requests.
	DefaultScheme = "http"
	// DefaultPort is the default port the Nakama API listens on.
	DefaultPort = 7350
	// DefaultServerKey is the default unauthenticated server key.
	DefaultServerKey = "defaultkey"
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 15 * time.Second
	// DefaultExpiredWindow is how far in advance of expiry the SDK refreshes
	// the session token automatically.
	DefaultExpiredWindow = 5 * time.Minute
)

// SessionUpdateCallback is invoked whenever a new session is received as a
// result of session refresh. Mirrors IClient.ReceivedSessionUpdated.
type SessionUpdateCallback func(*Session)

// Client is the main entry point of the SDK.
type Client struct {
	mu sync.Mutex

	scheme     string
	host       string
	port       int
	serverKey  string
	httpKey    string
	autoRefresh bool

	logger Logger
	apiClient *apiClient
	retryInvoker *retryInvoker
	globalRetry  *RetryConfiguration

	timeout time.Duration

	sessionUpdated SessionUpdateCallback
}

// NewClient builds a Client connecting to the supplied host with sensible
// defaults. Equivalent to Client.cs's primary constructor.
func NewClient(serverKey string) *Client {
	return NewClientWithAdapter(DefaultScheme, DefaultHost, DefaultPort, serverKey, NewHttpAdapterWithGzip(), true)
}

// NewClientWithAdapter constructs a Client with a custom scheme/host/port/adapter.
func NewClientWithAdapter(scheme, host string, port int, serverKey string, adapter HttpAdapter, autoRefreshSession bool) *Client {
	if adapter == nil {
		adapter = NewHttpAdapterWithGzip()
	}
	if scheme == "" {
		scheme = DefaultScheme
	}
	if host == "" {
		host = DefaultHost
	}
	if port == 0 {
		port = DefaultPort
	}
	if serverKey == "" {
		serverKey = DefaultServerKey
	}

	baseURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", host, port),
	}

	c := &Client{
		scheme:       scheme,
		host:         host,
		port:         port,
		serverKey:    serverKey,
		autoRefresh:  autoRefreshSession,
		logger:       NullLogger{},
		apiClient:    newAPIClient(baseURL, adapter, DefaultTimeout),
		retryInvoker: newRetryInvoker(adapter.TransientError()),
		globalRetry:  NewRetryConfiguration(500, 4),
		timeout:      DefaultTimeout,
	}
	adapter.SetLogger(c.logger)
	return c
}

// NewClientWithURL builds a Client targeting the supplied URL.
func NewClientWithURL(u *url.URL, serverKey string, adapter HttpAdapter, autoRefreshSession bool) *Client {
	if u == nil {
		return NewClient(serverKey)
	}
	if adapter == nil {
		adapter = NewHttpAdapterWithGzip()
	}
	port := 0
	if u.Port() != "" {
		fmt.Sscan(u.Port(), &port)
	} else if u.Scheme == "https" {
		port = 443
	} else {
		port = 80
	}
	c := &Client{
		scheme:       u.Scheme,
		host:         u.Hostname(),
		port:         port,
		serverKey:    serverKey,
		autoRefresh:  autoRefreshSession,
		logger:       NullLogger{},
		apiClient:    newAPIClient(u, adapter, DefaultTimeout),
		retryInvoker: newRetryInvoker(adapter.TransientError()),
		globalRetry:  NewRetryConfiguration(500, 4),
		timeout:      DefaultTimeout,
	}
	adapter.SetLogger(c.logger)
	return c
}

// AutoRefreshSession reports whether expired sessions are refreshed automatically.
func (c *Client) AutoRefreshSession() bool { return c.autoRefresh }

// Host returns the host address of the server.
func (c *Client) Host() string { return c.host }

// Port returns the port number of the server.
func (c *Client) Port() int { return c.port }

// Scheme returns the protocol scheme.
func (c *Client) Scheme() string { return c.scheme }

// ServerKey returns the authentication key used for unauthenticated calls.
func (c *Client) ServerKey() string { return c.serverKey }

// SetTimeout overrides the request timeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = d
	c.apiClient.timeout = d
}

// Timeout returns the per-request timeout.
func (c *Client) Timeout() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.timeout
}

// SetLogger replaces the logger on the client and its HTTP adapter.
func (c *Client) SetLogger(l Logger) {
	if l == nil {
		l = NullLogger{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = l
	c.apiClient.adapter.SetLogger(l)
}

// Logger returns the configured logger.
func (c *Client) Logger() Logger {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.logger
}

// SetGlobalRetryConfiguration replaces the default retry configuration.
func (c *Client) SetGlobalRetryConfiguration(cfg *RetryConfiguration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.globalRetry = cfg
}

// GlobalRetryConfiguration returns the default retry configuration.
func (c *Client) GlobalRetryConfiguration() *RetryConfiguration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.globalRetry
}

// OnSessionUpdated registers a callback invoked when the session is refreshed.
func (c *Client) OnSessionUpdated(cb SessionUpdateCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionUpdated = cb
}

// HttpAdapter returns the underlying HTTP adapter.
func (c *Client) HttpAdapter() HttpAdapter {
	return c.apiClient.adapter
}

// resolveRetry returns the per-call retry config, falling back to the global.
func (c *Client) resolveRetry(cfg *RetryConfiguration) *RetryConfiguration {
	if cfg != nil {
		return cfg
	}
	return c.GlobalRetryConfiguration()
}

// ensureValidSession refreshes the session if it has expired or is about to expire.
func (c *Client) ensureValidSession(ctx context.Context, session *Session, retry *RetryConfiguration) error {
	if !c.autoRefresh || session == nil || session.RefreshToken() == "" {
		return nil
	}
	if !session.HasExpired(time.Now().UTC().Add(DefaultExpiredWindow)) {
		return nil
	}
	_, err := c.SessionRefreshAsync(ctx, session, nil, retry)
	return err
}

// invokeWithSession handles auto-refresh + retry for a session-bound call.
func (c *Client) invokeWithSession(ctx context.Context, session *Session, retry *RetryConfiguration, do func(context.Context) error) error {
	if err := c.ensureValidSession(ctx, session, retry); err != nil {
		return err
	}
	cfg := c.resolveRetry(retry)
	history := newRetryHistory(session.AuthToken(), cfg)
	return c.retryInvoker.invoke(ctx, history, do)
}

// invokeWithSessionT is invokeWithSession for typed return values.
func invokeWithSessionT[T any](ctx context.Context, c *Client, session *Session, retry *RetryConfiguration, do func(context.Context) (T, error)) (T, error) {
	var zero T
	if err := c.ensureValidSession(ctx, session, retry); err != nil {
		return zero, err
	}
	cfg := c.resolveRetry(retry)
	history := newRetryHistory(session.AuthToken(), cfg)
	return invokeT(ctx, c.retryInvoker, history, do)
}

// invokeUnauth runs an unauthenticated call (e.g. authentication itself) under
// the retry policy.
func invokeUnauth[T any](ctx context.Context, c *Client, retry *RetryConfiguration, jitterSeed string, do func(context.Context) (T, error)) (T, error) {
	cfg := c.resolveRetry(retry)
	history := newRetryHistory(jitterSeed, cfg)
	return invokeT(ctx, c.retryInvoker, history, do)
}

// ===== Authentication =====

// AuthenticateAppleAsync authenticates the user with an Apple Sign In token.
func (c *Client) AuthenticateAppleAsync(ctx context.Context, token, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, token, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateApple(ctx, c.serverKey, "", &ApiAccountApple{Token: token, Vars: vars}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateCustomAsync authenticates the user with a custom id.
func (c *Client) AuthenticateCustomAsync(ctx context.Context, id, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, id, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateCustom(ctx, c.serverKey, "", &ApiAccountCustom{Id: id, Vars: vars}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateDeviceAsync authenticates the user with a device id.
func (c *Client) AuthenticateDeviceAsync(ctx context.Context, id, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, id, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateDevice(ctx, c.serverKey, "", &ApiAccountDevice{Id: id, Vars: vars}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateEmailAsync authenticates the user with an email and password.
func (c *Client) AuthenticateEmailAsync(ctx context.Context, email, password, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, email, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateEmail(ctx, c.serverKey, "", &ApiAccountEmail{Email: email, Password: password, Vars: vars}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateFacebookAsync authenticates the user with a Facebook OAuth token.
func (c *Client) AuthenticateFacebookAsync(ctx context.Context, token, username string, create, importFriends bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	syncPtr := importFriends
	resp, err := invokeUnauth(ctx, c, retry, token, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateFacebook(ctx, c.serverKey, "", &ApiAccountFacebook{Token: token, Vars: vars}, &createPtr, username, &syncPtr)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateGameCenterAsync authenticates the user with Apple Game Center credentials.
func (c *Client) AuthenticateGameCenterAsync(ctx context.Context, bundleId, playerId, publicKeyUrl, salt, signature, timestamp, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, bundleId, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateGameCenter(ctx, c.serverKey, "", &ApiAccountGameCenter{
			BundleId:         bundleId,
			PlayerId:         playerId,
			PublicKeyUrl:     publicKeyUrl,
			Salt:             salt,
			Signature:        signature,
			TimestampSeconds: timestamp,
			Vars:             vars,
		}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateGoogleAsync authenticates the user with a Google OAuth token.
func (c *Client) AuthenticateGoogleAsync(ctx context.Context, token, username string, create bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	resp, err := invokeUnauth(ctx, c, retry, token, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateGoogle(ctx, c.serverKey, "", &ApiAccountGoogle{Token: token, Vars: vars}, &createPtr, username)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// AuthenticateSteamAsync authenticates the user with a Steam token.
func (c *Client) AuthenticateSteamAsync(ctx context.Context, token, username string, create, importFriends bool, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	createPtr := create
	syncPtr := importFriends
	resp, err := invokeUnauth(ctx, c, retry, token, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.AuthenticateSteam(ctx, c.serverKey, "", &ApiAccountSteam{Token: token, Vars: vars}, &createPtr, username, &syncPtr)
	})
	if err != nil {
		return nil, err
	}
	return NewSession(resp.Token, resp.RefreshToken, resp.Created)
}

// SessionRefreshAsync refreshes a session, optionally replacing its bundled vars.
func (c *Client) SessionRefreshAsync(ctx context.Context, session *Session, vars map[string]string, retry *RetryConfiguration) (*Session, error) {
	if session == nil {
		return nil, errors.New("nakama: session is nil")
	}
	cfg := c.resolveRetry(retry)
	history := newRetryHistory(session.RefreshToken(), cfg)
	resp, err := invokeT(ctx, c.retryInvoker, history, func(ctx context.Context) (*ApiSession, error) {
		return c.apiClient.SessionRefresh(ctx, c.serverKey, "", &ApiSessionRefreshRequest{Token: session.RefreshToken(), Vars: vars})
	})
	if err != nil {
		return nil, err
	}
	if err := session.Update(resp.Token, resp.RefreshToken); err != nil {
		return nil, err
	}
	if cb := c.sessionUpdated; cb != nil {
		cb(session)
	}
	return session, nil
}

// SessionLogoutAsync invalidates the session's auth and refresh tokens.
func (c *Client) SessionLogoutAsync(ctx context.Context, session *Session, retry *RetryConfiguration) error {
	if session == nil {
		return errors.New("nakama: session is nil")
	}
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.SessionLogout(ctx, session.AuthToken(), &ApiSessionLogoutRequest{
			Token:        session.AuthToken(),
			RefreshToken: session.RefreshToken(),
		})
	})
}

// ===== Account =====

// GetAccountAsync fetches the current user's account.
func (c *Client) GetAccountAsync(ctx context.Context, session *Session, retry *RetryConfiguration) (*ApiAccount, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiAccount, error) {
		return c.apiClient.GetAccount(ctx, session.AuthToken())
	})
}

// UpdateAccountAsync updates the current user's account.
func (c *Client) UpdateAccountAsync(ctx context.Context, session *Session, username, displayName, avatarUrl, langTag, location, timezone string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UpdateAccount(ctx, session.AuthToken(), &ApiUpdateAccountRequest{
			Username:    username,
			DisplayName: displayName,
			AvatarUrl:   avatarUrl,
			LangTag:     langTag,
			Location:    location,
			Timezone:    timezone,
		})
	})
}

// DeleteAccountAsync deletes the current user's account.
func (c *Client) DeleteAccountAsync(ctx context.Context, session *Session, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteAccount(ctx, session.AuthToken())
	})
}

// ===== Linking =====

// LinkAppleAsync links an Apple ID to the current account.
func (c *Client) LinkAppleAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkApple(ctx, session.AuthToken(), &ApiAccountApple{Token: token})
	})
}

// LinkCustomAsync links a custom id to the current account.
func (c *Client) LinkCustomAsync(ctx context.Context, session *Session, id string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkCustom(ctx, session.AuthToken(), &ApiAccountCustom{Id: id})
	})
}

// LinkDeviceAsync links a device id to the current account.
func (c *Client) LinkDeviceAsync(ctx context.Context, session *Session, id string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkDevice(ctx, session.AuthToken(), &ApiAccountDevice{Id: id})
	})
}

// LinkEmailAsync links an email and password to the current account.
func (c *Client) LinkEmailAsync(ctx context.Context, session *Session, email, password string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkEmail(ctx, session.AuthToken(), &ApiAccountEmail{Email: email, Password: password})
	})
}

// LinkFacebookAsync links a Facebook profile to the current account.
func (c *Client) LinkFacebookAsync(ctx context.Context, session *Session, token string, importFriends bool, retry *RetryConfiguration) error {
	syncPtr := importFriends
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkFacebook(ctx, session.AuthToken(), &ApiAccountFacebook{Token: token}, &syncPtr)
	})
}

// LinkGameCenterAsync links Game Center credentials.
func (c *Client) LinkGameCenterAsync(ctx context.Context, session *Session, bundleId, playerId, publicKeyUrl, salt, signature, timestamp string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkGameCenter(ctx, session.AuthToken(), &ApiAccountGameCenter{
			BundleId:         bundleId,
			PlayerId:         playerId,
			PublicKeyUrl:     publicKeyUrl,
			Salt:             salt,
			Signature:        signature,
			TimestampSeconds: timestamp,
		})
	})
}

// LinkGoogleAsync links a Google profile to the current account.
func (c *Client) LinkGoogleAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkGoogle(ctx, session.AuthToken(), &ApiAccountGoogle{Token: token})
	})
}

// LinkSteamAsync links a Steam profile to the current account.
func (c *Client) LinkSteamAsync(ctx context.Context, session *Session, token string, importFriends bool, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LinkSteam(ctx, session.AuthToken(), &ApiLinkSteamRequest{
			Account: &ApiAccountSteam{Token: token},
			Sync:    importFriends,
		})
	})
}

// UnlinkAppleAsync removes a linked Apple ID.
func (c *Client) UnlinkAppleAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkApple(ctx, session.AuthToken(), &ApiAccountApple{Token: token})
	})
}

// UnlinkCustomAsync removes a linked custom id.
func (c *Client) UnlinkCustomAsync(ctx context.Context, session *Session, id string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkCustom(ctx, session.AuthToken(), &ApiAccountCustom{Id: id})
	})
}

// UnlinkDeviceAsync removes a linked device id.
func (c *Client) UnlinkDeviceAsync(ctx context.Context, session *Session, id string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkDevice(ctx, session.AuthToken(), &ApiAccountDevice{Id: id})
	})
}

// UnlinkEmailAsync removes a linked email and password.
func (c *Client) UnlinkEmailAsync(ctx context.Context, session *Session, email, password string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkEmail(ctx, session.AuthToken(), &ApiAccountEmail{Email: email, Password: password})
	})
}

// UnlinkFacebookAsync removes a linked Facebook profile.
func (c *Client) UnlinkFacebookAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkFacebook(ctx, session.AuthToken(), &ApiAccountFacebook{Token: token})
	})
}

// UnlinkGameCenterAsync removes linked Game Center credentials.
func (c *Client) UnlinkGameCenterAsync(ctx context.Context, session *Session, bundleId, playerId, publicKeyUrl, salt, signature, timestamp string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkGameCenter(ctx, session.AuthToken(), &ApiAccountGameCenter{
			BundleId:         bundleId,
			PlayerId:         playerId,
			PublicKeyUrl:     publicKeyUrl,
			Salt:             salt,
			Signature:        signature,
			TimestampSeconds: timestamp,
		})
	})
}

// UnlinkGoogleAsync removes a linked Google profile.
func (c *Client) UnlinkGoogleAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkGoogle(ctx, session.AuthToken(), &ApiAccountGoogle{Token: token})
	})
}

// UnlinkSteamAsync removes a linked Steam profile.
func (c *Client) UnlinkSteamAsync(ctx context.Context, session *Session, token string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UnlinkSteam(ctx, session.AuthToken(), &ApiAccountSteam{Token: token})
	})
}

// ===== Friends =====

// AddFriendsAsync adds friends by id or username.
func (c *Client) AddFriendsAsync(ctx context.Context, session *Session, ids, usernames []string, metadata string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.AddFriends(ctx, session.AuthToken(), ids, usernames, metadata)
	})
}

// BlockFriendsAsync blocks one or more users.
func (c *Client) BlockFriendsAsync(ctx context.Context, session *Session, ids, usernames []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.BlockFriends(ctx, session.AuthToken(), ids, usernames)
	})
}

// DeleteFriendsAsync removes friends by id or username.
func (c *Client) DeleteFriendsAsync(ctx context.Context, session *Session, ids, usernames []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteFriends(ctx, session.AuthToken(), ids, usernames)
	})
}

// ListFriendsAsync lists the user's friends.
func (c *Client) ListFriendsAsync(ctx context.Context, session *Session, state *int, limit int, cursor string, retry *RetryConfiguration) (*ApiFriendList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiFriendList, error) {
		return c.apiClient.ListFriends(ctx, session.AuthToken(), &limit, state, cursor)
	})
}

// ListFriendsOfFriendsAsync lists friends of friends.
func (c *Client) ListFriendsOfFriendsAsync(ctx context.Context, session *Session, limit int, cursor string, retry *RetryConfiguration) (*ApiFriendsOfFriendsList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiFriendsOfFriendsList, error) {
		return c.apiClient.ListFriendsOfFriends(ctx, session.AuthToken(), &limit, cursor)
	})
}

// ImportFacebookFriendsAsync imports Facebook friends.
func (c *Client) ImportFacebookFriendsAsync(ctx context.Context, session *Session, token string, reset *bool, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.ImportFacebookFriends(ctx, session.AuthToken(), &ApiAccountFacebook{Token: token}, reset)
	})
}

// ImportSteamFriendsAsync imports Steam friends.
func (c *Client) ImportSteamFriendsAsync(ctx context.Context, session *Session, token string, reset *bool, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.ImportSteamFriends(ctx, session.AuthToken(), &ApiAccountSteam{Token: token}, reset)
	})
}

// ===== Groups =====

// CreateGroupAsync creates a new group.
func (c *Client) CreateGroupAsync(ctx context.Context, session *Session, name, description, avatarUrl, langTag string, open bool, maxCount int, retry *RetryConfiguration) (*ApiGroup, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiGroup, error) {
		return c.apiClient.CreateGroup(ctx, session.AuthToken(), &ApiCreateGroupRequest{
			Name:        name,
			Description: description,
			AvatarUrl:   avatarUrl,
			LangTag:     langTag,
			Open:        open,
			MaxCount:    maxCount,
		})
	})
}

// DeleteGroupAsync deletes a group by id.
func (c *Client) DeleteGroupAsync(ctx context.Context, session *Session, groupId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteGroup(ctx, session.AuthToken(), groupId)
	})
}

// UpdateGroupAsync updates fields of a group.
func (c *Client) UpdateGroupAsync(ctx context.Context, session *Session, groupId, name string, open bool, description, avatarUrl, langTag string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.UpdateGroup(ctx, session.AuthToken(), groupId, &ApiUpdateGroupRequest{
			Name:        name,
			Description: description,
			AvatarUrl:   avatarUrl,
			LangTag:     langTag,
			Open:        open,
		})
	})
}

// AddGroupUsersAsync adds users to a group.
func (c *Client) AddGroupUsersAsync(ctx context.Context, session *Session, groupId string, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.AddGroupUsers(ctx, session.AuthToken(), groupId, ids)
	})
}

// BanGroupUsersAsync bans users from a group.
func (c *Client) BanGroupUsersAsync(ctx context.Context, session *Session, groupId string, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.BanGroupUsers(ctx, session.AuthToken(), groupId, ids)
	})
}

// DemoteGroupUsersAsync demotes users in a group.
func (c *Client) DemoteGroupUsersAsync(ctx context.Context, session *Session, groupId string, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DemoteGroupUsers(ctx, session.AuthToken(), groupId, ids)
	})
}

// JoinGroupAsync requests to join a group.
func (c *Client) JoinGroupAsync(ctx context.Context, session *Session, groupId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.JoinGroup(ctx, session.AuthToken(), groupId)
	})
}

// KickGroupUsersAsync kicks users from a group.
func (c *Client) KickGroupUsersAsync(ctx context.Context, session *Session, groupId string, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.KickGroupUsers(ctx, session.AuthToken(), groupId, ids)
	})
}

// LeaveGroupAsync leaves a group.
func (c *Client) LeaveGroupAsync(ctx context.Context, session *Session, groupId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.LeaveGroup(ctx, session.AuthToken(), groupId)
	})
}

// PromoteGroupUsersAsync promotes users in a group.
func (c *Client) PromoteGroupUsersAsync(ctx context.Context, session *Session, groupId string, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.PromoteGroupUsers(ctx, session.AuthToken(), groupId, ids)
	})
}

// ListGroupsAsync lists groups based on filters.
func (c *Client) ListGroupsAsync(ctx context.Context, session *Session, name string, limit int, cursor, langTag string, members *int, open *bool, retry *RetryConfiguration) (*ApiGroupList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiGroupList, error) {
		return c.apiClient.ListGroups(ctx, session.AuthToken(), name, cursor, &limit, langTag, members, open)
	})
}

// ListGroupUsersAsync lists users belonging to a group.
func (c *Client) ListGroupUsersAsync(ctx context.Context, session *Session, groupId string, state *int, limit int, cursor string, retry *RetryConfiguration) (*ApiGroupUserList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiGroupUserList, error) {
		return c.apiClient.ListGroupUsers(ctx, session.AuthToken(), groupId, &limit, state, cursor)
	})
}

// ListUserGroupsAsync lists groups the user belongs to.
func (c *Client) ListUserGroupsAsync(ctx context.Context, session *Session, userId string, state *int, limit int, cursor string, retry *RetryConfiguration) (*ApiUserGroupList, error) {
	target := userId
	if target == "" {
		target = session.UserId()
	}
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiUserGroupList, error) {
		return c.apiClient.ListUserGroups(ctx, session.AuthToken(), target, &limit, state, cursor)
	})
}

// ===== Notifications =====

// DeleteNotificationsAsync deletes notifications by id.
func (c *Client) DeleteNotificationsAsync(ctx context.Context, session *Session, ids []string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteNotifications(ctx, session.AuthToken(), ids)
	})
}

// ListNotificationsAsync lists notifications for the user.
func (c *Client) ListNotificationsAsync(ctx context.Context, session *Session, limit int, cacheableCursor string, retry *RetryConfiguration) (*ApiNotificationList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiNotificationList, error) {
		return c.apiClient.ListNotifications(ctx, session.AuthToken(), &limit, cacheableCursor)
	})
}

// ===== Channel =====

// ListChannelMessagesAsync lists messages from a chat channel.
func (c *Client) ListChannelMessagesAsync(ctx context.Context, session *Session, channelId string, limit int, forward bool, cursor string, retry *RetryConfiguration) (*ApiChannelMessageList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiChannelMessageList, error) {
		return c.apiClient.ListChannelMessages(ctx, session.AuthToken(), channelId, &limit, &forward, cursor)
	})
}

// ===== Match =====

// ListMatchesAsync lists realtime matches.
func (c *Client) ListMatchesAsync(ctx context.Context, session *Session, min, max, limit int, authoritative bool, label, query string, retry *RetryConfiguration) (*ApiMatchList, error) {
	authPtr := authoritative
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiMatchList, error) {
		return c.apiClient.ListMatches(ctx, session.AuthToken(), &limit, &authPtr, label, &min, &max, query)
	})
}

// ===== Storage =====

// ReadStorageObjectsAsync reads storage objects.
func (c *Client) ReadStorageObjectsAsync(ctx context.Context, session *Session, ids []*ApiReadStorageObjectId, retry *RetryConfiguration) (*ApiStorageObjects, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiStorageObjects, error) {
		return c.apiClient.ReadStorageObjects(ctx, session.AuthToken(), &ApiReadStorageObjectsRequest{ObjectIds: ids})
	})
}

// WriteStorageObjectsAsync writes storage objects.
func (c *Client) WriteStorageObjectsAsync(ctx context.Context, session *Session, objects []*ApiWriteStorageObject, retry *RetryConfiguration) (*ApiStorageObjectAcks, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiStorageObjectAcks, error) {
		return c.apiClient.WriteStorageObjects(ctx, session.AuthToken(), &ApiWriteStorageObjectsRequest{Objects: objects})
	})
}

// DeleteStorageObjectsAsync deletes storage objects.
func (c *Client) DeleteStorageObjectsAsync(ctx context.Context, session *Session, ids []*StorageObjectId, retry *RetryConfiguration) error {
	delIds := make([]*ApiDeleteStorageObjectId, 0, len(ids))
	for _, id := range ids {
		delIds = append(delIds, &ApiDeleteStorageObjectId{
			Collection: id.Collection,
			Key:        id.Key,
			Version:    id.Version,
		})
	}
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteStorageObjects(ctx, session.AuthToken(), &ApiDeleteStorageObjectsRequest{ObjectIds: delIds})
	})
}

// ListStorageObjectsAsync lists public storage objects in a collection.
func (c *Client) ListStorageObjectsAsync(ctx context.Context, session *Session, collection string, limit int, cursor string, retry *RetryConfiguration) (*ApiStorageObjectList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiStorageObjectList, error) {
		return c.apiClient.ListStorageObjects(ctx, session.AuthToken(), collection, "", &limit, cursor)
	})
}

// ListUsersStorageObjectsAsync lists a user's storage objects in a collection.
func (c *Client) ListUsersStorageObjectsAsync(ctx context.Context, session *Session, collection, userId string, limit int, cursor string, retry *RetryConfiguration) (*ApiStorageObjectList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiStorageObjectList, error) {
		return c.apiClient.ListStorageObjects2(ctx, session.AuthToken(), collection, userId, &limit, cursor)
	})
}

// ===== Leaderboards =====

// DeleteLeaderboardRecordAsync deletes a leaderboard record.
func (c *Client) DeleteLeaderboardRecordAsync(ctx context.Context, session *Session, leaderboardId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteLeaderboardRecord(ctx, session.AuthToken(), leaderboardId)
	})
}

// ListLeaderboardRecordsAsync lists leaderboard records.
func (c *Client) ListLeaderboardRecordsAsync(ctx context.Context, session *Session, leaderboardId string, ownerIds []string, expiry *int64, limit int, cursor string, retry *RetryConfiguration) (*ApiLeaderboardRecordList, error) {
	expiryStr := ""
	if expiry != nil {
		expiryStr = int64ToString(*expiry)
	}
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiLeaderboardRecordList, error) {
		return c.apiClient.ListLeaderboardRecords(ctx, session.AuthToken(), leaderboardId, ownerIds, &limit, cursor, expiryStr)
	})
}

// ListLeaderboardRecordsAroundOwnerAsync lists records around a given owner.
func (c *Client) ListLeaderboardRecordsAroundOwnerAsync(ctx context.Context, session *Session, leaderboardId, ownerId string, expiry *int64, limit int, cursor string, retry *RetryConfiguration) (*ApiLeaderboardRecordList, error) {
	expiryStr := ""
	if expiry != nil {
		expiryStr = int64ToString(*expiry)
	}
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiLeaderboardRecordList, error) {
		return c.apiClient.ListLeaderboardRecordsAroundOwner(ctx, session.AuthToken(), leaderboardId, ownerId, &limit, expiryStr, cursor)
	})
}

// WriteLeaderboardRecordAsync writes a record to a leaderboard.
func (c *Client) WriteLeaderboardRecordAsync(ctx context.Context, session *Session, leaderboardId string, score, subScore int64, metadata string, op ApiOperator, retry *RetryConfiguration) (*ApiLeaderboardRecord, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiLeaderboardRecord, error) {
		return c.apiClient.WriteLeaderboardRecord(ctx, session.AuthToken(), leaderboardId, &WriteLeaderboardRecordRequestLeaderboardRecordWrite{
			Score:    int64ToString(score),
			Subscore: int64ToString(subScore),
			Metadata: metadata,
			Operator: op,
		})
	})
}

// ===== Tournaments =====

// DeleteTournamentRecordAsync deletes the user's tournament record.
func (c *Client) DeleteTournamentRecordAsync(ctx context.Context, session *Session, tournamentId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.DeleteTournamentRecord(ctx, session.AuthToken(), tournamentId)
	})
}

// ListTournamentsAsync lists tournaments on the server.
func (c *Client) ListTournamentsAsync(ctx context.Context, session *Session, categoryStart, categoryEnd int, startTime, endTime *int, limit int, cursor string, retry *RetryConfiguration) (*ApiTournamentList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiTournamentList, error) {
		return c.apiClient.ListTournaments(ctx, session.AuthToken(), &categoryStart, &categoryEnd, startTime, endTime, &limit, cursor)
	})
}

// ListTournamentRecordsAsync lists records in a tournament.
func (c *Client) ListTournamentRecordsAsync(ctx context.Context, session *Session, tournamentId string, ownerIds []string, expiry *int64, limit int, cursor string, retry *RetryConfiguration) (*ApiTournamentRecordList, error) {
	expiryStr := ""
	if expiry != nil {
		expiryStr = int64ToString(*expiry)
	}
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiTournamentRecordList, error) {
		return c.apiClient.ListTournamentRecords(ctx, session.AuthToken(), tournamentId, ownerIds, &limit, cursor, expiryStr)
	})
}

// ListTournamentRecordsAroundOwnerAsync lists tournament records around an owner.
func (c *Client) ListTournamentRecordsAroundOwnerAsync(ctx context.Context, session *Session, tournamentId, ownerId string, expiry *int64, limit int, cursor string, retry *RetryConfiguration) (*ApiTournamentRecordList, error) {
	expiryStr := ""
	if expiry != nil {
		expiryStr = int64ToString(*expiry)
	}
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiTournamentRecordList, error) {
		return c.apiClient.ListTournamentRecordsAroundOwner(ctx, session.AuthToken(), tournamentId, ownerId, &limit, expiryStr, cursor)
	})
}

// WriteTournamentRecordAsync writes a record to a tournament.
func (c *Client) WriteTournamentRecordAsync(ctx context.Context, session *Session, tournamentId string, score, subScore int64, metadata string, op ApiOperator, retry *RetryConfiguration) (*ApiLeaderboardRecord, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiLeaderboardRecord, error) {
		return c.apiClient.WriteTournamentRecord(ctx, session.AuthToken(), tournamentId, &WriteTournamentRecordRequestTournamentRecordWrite{
			Score:    int64ToString(score),
			Subscore: int64ToString(subScore),
			Metadata: metadata,
			Operator: op,
		})
	})
}

// JoinTournamentAsync joins a tournament.
func (c *Client) JoinTournamentAsync(ctx context.Context, session *Session, tournamentId string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.JoinTournament(ctx, session.AuthToken(), tournamentId)
	})
}

// ===== IAP =====

// ValidatePurchaseAppleAsync validates an Apple App Store purchase.
func (c *Client) ValidatePurchaseAppleAsync(ctx context.Context, session *Session, receipt string, persist bool, retry *RetryConfiguration) (*ApiValidatePurchaseResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidatePurchaseResponse, error) {
		return c.apiClient.ValidatePurchaseApple(ctx, session.AuthToken(), &ApiValidatePurchaseAppleRequest{Receipt: receipt, Persist: &persistPtr})
	})
}

// ValidatePurchaseFacebookInstantAsync validates a Facebook Instant purchase.
func (c *Client) ValidatePurchaseFacebookInstantAsync(ctx context.Context, session *Session, signedRequest string, persist bool, retry *RetryConfiguration) (*ApiValidatePurchaseResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidatePurchaseResponse, error) {
		return c.apiClient.ValidatePurchaseFacebookInstant(ctx, session.AuthToken(), &ApiValidatePurchaseFacebookInstantRequest{SignedRequest: signedRequest, Persist: &persistPtr})
	})
}

// ValidatePurchaseGoogleAsync validates a Google Play Store purchase.
func (c *Client) ValidatePurchaseGoogleAsync(ctx context.Context, session *Session, receipt string, persist bool, retry *RetryConfiguration) (*ApiValidatePurchaseResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidatePurchaseResponse, error) {
		return c.apiClient.ValidatePurchaseGoogle(ctx, session.AuthToken(), &ApiValidatePurchaseGoogleRequest{Purchase: receipt, Persist: &persistPtr})
	})
}

// ValidatePurchaseHuaweiAsync validates a Huawei AppGallery purchase.
func (c *Client) ValidatePurchaseHuaweiAsync(ctx context.Context, session *Session, receipt, signature string, persist bool, retry *RetryConfiguration) (*ApiValidatePurchaseResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidatePurchaseResponse, error) {
		return c.apiClient.ValidatePurchaseHuawei(ctx, session.AuthToken(), &ApiValidatePurchaseHuaweiRequest{Purchase: receipt, Signature: signature, Persist: &persistPtr})
	})
}

// ValidateSubscriptionAppleAsync validates an Apple subscription.
func (c *Client) ValidateSubscriptionAppleAsync(ctx context.Context, session *Session, receipt string, persist bool, retry *RetryConfiguration) (*ApiValidateSubscriptionResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidateSubscriptionResponse, error) {
		return c.apiClient.ValidateSubscriptionApple(ctx, session.AuthToken(), &ApiValidateSubscriptionAppleRequest{Receipt: receipt, Persist: &persistPtr})
	})
}

// ValidateSubscriptionGoogleAsync validates a Google subscription.
func (c *Client) ValidateSubscriptionGoogleAsync(ctx context.Context, session *Session, receipt string, persist bool, retry *RetryConfiguration) (*ApiValidateSubscriptionResponse, error) {
	persistPtr := persist
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidateSubscriptionResponse, error) {
		return c.apiClient.ValidateSubscriptionGoogle(ctx, session.AuthToken(), &ApiValidateSubscriptionGoogleRequest{Receipt: receipt, Persist: &persistPtr})
	})
}

// ListSubscriptionsAsync lists user subscriptions.
func (c *Client) ListSubscriptionsAsync(ctx context.Context, session *Session, limit int, cursor string, retry *RetryConfiguration) (*ApiSubscriptionList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiSubscriptionList, error) {
		return c.apiClient.ListSubscriptions(ctx, session.AuthToken(), &ApiListSubscriptionsRequest{Limit: &limit, Cursor: cursor})
	})
}

// GetSubscriptionAsync retrieves a subscription by product id.
func (c *Client) GetSubscriptionAsync(ctx context.Context, session *Session, productId string, retry *RetryConfiguration) (*ApiValidatedSubscription, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiValidatedSubscription, error) {
		return c.apiClient.GetSubscription(ctx, session.AuthToken(), productId)
	})
}

// ===== RPC =====

// RpcAsync executes a server RPC with the user's session.
func (c *Client) RpcAsync(ctx context.Context, session *Session, id, payload string, retry *RetryConfiguration) (*ApiRpc, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiRpc, error) {
		if payload == "" {
			return c.apiClient.RpcGet(ctx, session.AuthToken(), "", "", id, "", "")
		}
		return c.apiClient.RpcPost(ctx, session.AuthToken(), "", "", id, payload, "")
	})
}

// RpcWithHttpKeyAsync executes a server RPC using an HTTP key (no session).
func (c *Client) RpcWithHttpKeyAsync(ctx context.Context, httpKey, id, payload string, retry *RetryConfiguration) (*ApiRpc, error) {
	return invokeUnauth(ctx, c, retry, id, func(ctx context.Context) (*ApiRpc, error) {
		if payload == "" {
			return c.apiClient.RpcGet(ctx, "", "", "", id, "", httpKey)
		}
		return c.apiClient.RpcPost(ctx, "", "", "", id, payload, httpKey)
	})
}

// ===== Users =====

// GetUsersAsync fetches users by id, username, or facebook id.
func (c *Client) GetUsersAsync(ctx context.Context, session *Session, ids, usernames, facebookIds []string, retry *RetryConfiguration) (*ApiUsers, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiUsers, error) {
		return c.apiClient.GetUsers(ctx, session.AuthToken(), ids, usernames, facebookIds)
	})
}

// ===== Misc =====

// EventAsync submits a custom event for the runtime to handle.
func (c *Client) EventAsync(ctx context.Context, session *Session, name string, properties map[string]string, retry *RetryConfiguration) error {
	return c.invokeWithSession(ctx, session, retry, func(ctx context.Context) error {
		return c.apiClient.Event(ctx, session.AuthToken(), &ApiEvent{Name: name, Properties: properties})
	})
}

// ListPartiesAsync lists advertised parties.
func (c *Client) ListPartiesAsync(ctx context.Context, session *Session, limit int, open *bool, query, cursor string, retry *RetryConfiguration) (*ApiPartyList, error) {
	return invokeWithSessionT(ctx, c, session, retry, func(ctx context.Context) (*ApiPartyList, error) {
		return c.apiClient.ListParties(ctx, session.AuthToken(), &limit, open, query, cursor)
	})
}

// HealthcheckAsync invokes the server's healthcheck.
func (c *Client) HealthcheckAsync(ctx context.Context, retry *RetryConfiguration) error {
	cfg := c.resolveRetry(retry)
	history := newRetryHistory("healthcheck", cfg)
	return c.retryInvoker.invoke(ctx, history, func(ctx context.Context) error {
		return c.apiClient.Healthcheck(ctx, "")
	})
}
