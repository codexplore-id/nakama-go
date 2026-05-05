// Code adapted from Nakama/ApiClient.gen.cs in the .NET SDK.
package nakama

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// apiClient is the low-level HTTP client used internally by Client. It maps
// 1:1 to the operations exposed by the Nakama REST API.
type apiClient struct {
	baseURL *url.URL
	adapter HttpAdapter
	timeout time.Duration
}

func newAPIClient(baseURL *url.URL, adapter HttpAdapter, timeout time.Duration) *apiClient {
	return &apiClient{baseURL: baseURL, adapter: adapter, timeout: timeout}
}

// queryBuilder accumulates query parameters into a single string.
type queryBuilder struct {
	parts []string
}

func (q *queryBuilder) addString(name, value string) {
	q.parts = append(q.parts, name+"="+url.QueryEscape(value))
}

func (q *queryBuilder) addStringPtr(name string, value *string) {
	if value != nil {
		q.addString(name, *value)
	}
}

func (q *queryBuilder) addInt(name string, value int) {
	q.parts = append(q.parts, name+"="+strconv.Itoa(value))
}

func (q *queryBuilder) addIntPtr(name string, value *int) {
	if value != nil {
		q.addInt(name, *value)
	}
}

func (q *queryBuilder) addInt64Ptr(name string, value *int64) {
	if value != nil {
		q.parts = append(q.parts, name+"="+strconv.FormatInt(*value, 10))
	}
}

func (q *queryBuilder) addBool(name string, value bool) {
	q.parts = append(q.parts, name+"="+strconv.FormatBool(value))
}

func (q *queryBuilder) addBoolPtr(name string, value *bool) {
	if value != nil {
		q.addBool(name, *value)
	}
}

func (q *queryBuilder) addStrings(name string, values []string) {
	for _, v := range values {
		q.addString(name, v)
	}
}

func (q *queryBuilder) String() string {
	return strings.Join(q.parts, "&")
}

// bearerHeaders returns headers carrying a Bearer authorization.
func bearerHeaders(token string) map[string]string {
	if token == "" {
		return map[string]string{}
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

// basicHeaders returns headers carrying a Basic authorization.
func basicHeaders(user, pass string) map[string]string {
	if user == "" {
		return map[string]string{}
	}
	creds := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return map[string]string{"Authorization": "Basic " + creds}
}

// send is a small wrapper that sends a request and parses the response into
// the supplied target value. If target is nil, the response body is discarded.
func (c *apiClient) send(ctx context.Context, method, path, query string, headers map[string]string, body any, target any) error {
	uri := formatURL(c.baseURL, path, query)
	var bodyBytes []byte
	if body != nil {
		switch b := body.(type) {
		case string:
			bodyBytes = []byte(b)
		case []byte:
			bodyBytes = b
		default:
			marshalled, err := json.Marshal(b)
			if err != nil {
				return fmt.Errorf("nakama: cannot marshal request body: %w", err)
			}
			bodyBytes = marshalled
		}
	}
	resp, err := c.adapter.Send(ctx, method, uri, headers, bodyBytes, c.timeout)
	if err != nil {
		return err
	}
	if target == nil || resp == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(resp), target); err != nil {
		return fmt.Errorf("nakama: cannot decode response: %w", err)
	}
	return nil
}

// pathReplace URL-escapes value and replaces {placeholder} in template.
func pathReplace(template, placeholder, value string) string {
	return strings.Replace(template, "{"+placeholder+"}", url.PathEscape(value), 1)
}

// ===== Account =====

// Healthcheck pings the server's healthcheck endpoint.
func (c *apiClient) Healthcheck(ctx context.Context, bearerToken string) error {
	return c.send(ctx, "GET", "/healthcheck", "", bearerHeaders(bearerToken), nil, nil)
}

// DeleteAccount deletes the current user's account.
func (c *apiClient) DeleteAccount(ctx context.Context, bearerToken string) error {
	return c.send(ctx, "DELETE", "/v2/account", "", bearerHeaders(bearerToken), nil, nil)
}

// GetAccount fetches the current user's account.
func (c *apiClient) GetAccount(ctx context.Context, bearerToken string) (*ApiAccount, error) {
	out := &ApiAccount{}
	if err := c.send(ctx, "GET", "/v2/account", "", bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateAccount updates fields in the current user's account.
func (c *apiClient) UpdateAccount(ctx context.Context, bearerToken string, body *ApiUpdateAccountRequest) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "PUT", "/v2/account", "", bearerHeaders(bearerToken), body, nil)
}

// AuthenticateApple authenticates a user with an Apple ID.
func (c *apiClient) AuthenticateApple(ctx context.Context, basicUser, basicPass string, account *ApiAccountApple, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/apple", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateCustom authenticates a user with a custom id.
func (c *apiClient) AuthenticateCustom(ctx context.Context, basicUser, basicPass string, account *ApiAccountCustom, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/custom", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateDevice authenticates a user with a device id.
func (c *apiClient) AuthenticateDevice(ctx context.Context, basicUser, basicPass string, account *ApiAccountDevice, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/device", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateEmail authenticates a user with an email and password.
func (c *apiClient) AuthenticateEmail(ctx context.Context, basicUser, basicPass string, account *ApiAccountEmail, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/email", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateFacebook authenticates a user with a Facebook OAuth token.
func (c *apiClient) AuthenticateFacebook(ctx context.Context, basicUser, basicPass string, account *ApiAccountFacebook, create *bool, username string, sync *bool) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	q.addBoolPtr("sync", sync)
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/facebook", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateFacebookInstantGame authenticates a user with a Facebook Instant Games token.
func (c *apiClient) AuthenticateFacebookInstantGame(ctx context.Context, basicUser, basicPass string, account *ApiAccountFacebookInstantGame, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/facebookinstantgame", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateGameCenter authenticates a user with Apple Game Center credentials.
func (c *apiClient) AuthenticateGameCenter(ctx context.Context, basicUser, basicPass string, account *ApiAccountGameCenter, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/gamecenter", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateGoogle authenticates a user with a Google OAuth token.
func (c *apiClient) AuthenticateGoogle(ctx context.Context, basicUser, basicPass string, account *ApiAccountGoogle, create *bool, username string) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/google", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AuthenticateSteam authenticates a user with a Steam token.
func (c *apiClient) AuthenticateSteam(ctx context.Context, basicUser, basicPass string, account *ApiAccountSteam, create *bool, username string, sync *bool) (*ApiSession, error) {
	if account == nil {
		return nil, fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("create", create)
	if username != "" {
		q.addString("username", username)
	}
	q.addBoolPtr("sync", sync)
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/authenticate/steam", q.String(), basicHeaders(basicUser, basicPass), account, out); err != nil {
		return nil, err
	}
	return out, nil
}

// LinkApple links an Apple ID to the current account.
func (c *apiClient) LinkApple(ctx context.Context, bearerToken string, body *ApiAccountApple) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/apple", "", bearerHeaders(bearerToken), body, nil)
}

// LinkCustom links a custom id to the current account.
func (c *apiClient) LinkCustom(ctx context.Context, bearerToken string, body *ApiAccountCustom) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/custom", "", bearerHeaders(bearerToken), body, nil)
}

// LinkDevice links a device id to the current account.
func (c *apiClient) LinkDevice(ctx context.Context, bearerToken string, body *ApiAccountDevice) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/device", "", bearerHeaders(bearerToken), body, nil)
}

// LinkEmail links an email and password to the current account.
func (c *apiClient) LinkEmail(ctx context.Context, bearerToken string, body *ApiAccountEmail) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/email", "", bearerHeaders(bearerToken), body, nil)
}

// LinkFacebook links a Facebook profile to the current account.
func (c *apiClient) LinkFacebook(ctx context.Context, bearerToken string, body *ApiAccountFacebook, sync *bool) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("sync", sync)
	return c.send(ctx, "POST", "/v2/account/link/facebook", q.String(), bearerHeaders(bearerToken), body, nil)
}

// LinkFacebookInstantGame links a Facebook Instant Games profile.
func (c *apiClient) LinkFacebookInstantGame(ctx context.Context, bearerToken string, body *ApiAccountFacebookInstantGame) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/facebookinstantgame", "", bearerHeaders(bearerToken), body, nil)
}

// LinkGameCenter links Game Center credentials.
func (c *apiClient) LinkGameCenter(ctx context.Context, bearerToken string, body *ApiAccountGameCenter) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/gamecenter", "", bearerHeaders(bearerToken), body, nil)
}

// LinkGoogle links a Google profile to the current account.
func (c *apiClient) LinkGoogle(ctx context.Context, bearerToken string, body *ApiAccountGoogle) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/google", "", bearerHeaders(bearerToken), body, nil)
}

// LinkSteam links a Steam profile to the current account.
func (c *apiClient) LinkSteam(ctx context.Context, bearerToken string, body *ApiLinkSteamRequest) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/account/link/steam", "", bearerHeaders(bearerToken), body, nil)
}

// SessionRefresh refreshes the current session, returning a new session.
func (c *apiClient) SessionRefresh(ctx context.Context, basicUser, basicPass string, body *ApiSessionRefreshRequest) (*ApiSession, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiSession{}
	if err := c.send(ctx, "POST", "/v2/account/session/refresh", "", basicHeaders(basicUser, basicPass), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UnlinkApple removes a linked Apple ID.
func (c *apiClient) UnlinkApple(ctx context.Context, bearerToken string, body *ApiAccountApple) error {
	return c.send(ctx, "POST", "/v2/account/unlink/apple", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkCustom removes a linked custom id.
func (c *apiClient) UnlinkCustom(ctx context.Context, bearerToken string, body *ApiAccountCustom) error {
	return c.send(ctx, "POST", "/v2/account/unlink/custom", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkDevice removes a linked device id.
func (c *apiClient) UnlinkDevice(ctx context.Context, bearerToken string, body *ApiAccountDevice) error {
	return c.send(ctx, "POST", "/v2/account/unlink/device", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkEmail removes a linked email and password.
func (c *apiClient) UnlinkEmail(ctx context.Context, bearerToken string, body *ApiAccountEmail) error {
	return c.send(ctx, "POST", "/v2/account/unlink/email", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkFacebook removes a linked Facebook profile.
func (c *apiClient) UnlinkFacebook(ctx context.Context, bearerToken string, body *ApiAccountFacebook) error {
	return c.send(ctx, "POST", "/v2/account/unlink/facebook", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkFacebookInstantGame removes a linked Facebook Instant Game profile.
func (c *apiClient) UnlinkFacebookInstantGame(ctx context.Context, bearerToken string, body *ApiAccountFacebookInstantGame) error {
	return c.send(ctx, "POST", "/v2/account/unlink/facebookinstantgame", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkGameCenter removes linked Game Center credentials.
func (c *apiClient) UnlinkGameCenter(ctx context.Context, bearerToken string, body *ApiAccountGameCenter) error {
	return c.send(ctx, "POST", "/v2/account/unlink/gamecenter", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkGoogle removes a linked Google profile.
func (c *apiClient) UnlinkGoogle(ctx context.Context, bearerToken string, body *ApiAccountGoogle) error {
	return c.send(ctx, "POST", "/v2/account/unlink/google", "", bearerHeaders(bearerToken), body, nil)
}

// UnlinkSteam removes a linked Steam profile.
func (c *apiClient) UnlinkSteam(ctx context.Context, bearerToken string, body *ApiAccountSteam) error {
	return c.send(ctx, "POST", "/v2/account/unlink/steam", "", bearerHeaders(bearerToken), body, nil)
}

// ===== Channel =====

// ListChannelMessages lists messages from a chat channel.
func (c *apiClient) ListChannelMessages(ctx context.Context, bearerToken, channelId string, limit *int, forward *bool, cursor string) (*ApiChannelMessageList, error) {
	if channelId == "" {
		return nil, fmt.Errorf("nakama: 'channelId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addBoolPtr("forward", forward)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiChannelMessageList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/channel/{channelId}", "channelId", channelId), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Event =====

// Event submits an event for processing in the server.
func (c *apiClient) Event(ctx context.Context, bearerToken string, body *ApiEvent) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/event", "", bearerHeaders(bearerToken), body, nil)
}

// ===== Friends =====

// DeleteFriends removes friends by id or username.
func (c *apiClient) DeleteFriends(ctx context.Context, bearerToken string, ids, usernames []string) error {
	q := queryBuilder{}
	q.addStrings("ids", ids)
	q.addStrings("usernames", usernames)
	return c.send(ctx, "DELETE", "/v2/friend", q.String(), bearerHeaders(bearerToken), nil, nil)
}

// ListFriends lists the current user's friends.
func (c *apiClient) ListFriends(ctx context.Context, bearerToken string, limit, state *int, cursor string) (*ApiFriendList, error) {
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addIntPtr("state", state)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiFriendList{}
	if err := c.send(ctx, "GET", "/v2/friend", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddFriends adds friends by id or username.
func (c *apiClient) AddFriends(ctx context.Context, bearerToken string, ids, usernames []string, metadata string) error {
	q := queryBuilder{}
	q.addStrings("ids", ids)
	q.addStrings("usernames", usernames)
	if metadata != "" {
		q.addString("metadata", metadata)
	}
	return c.send(ctx, "POST", "/v2/friend", q.String(), bearerHeaders(bearerToken), nil, nil)
}

// BlockFriends blocks one or more users.
func (c *apiClient) BlockFriends(ctx context.Context, bearerToken string, ids, usernames []string) error {
	q := queryBuilder{}
	q.addStrings("ids", ids)
	q.addStrings("usernames", usernames)
	return c.send(ctx, "POST", "/v2/friend/block", q.String(), bearerHeaders(bearerToken), nil, nil)
}

// ImportFacebookFriends imports Facebook friends.
func (c *apiClient) ImportFacebookFriends(ctx context.Context, bearerToken string, account *ApiAccountFacebook, reset *bool) error {
	if account == nil {
		return fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("reset", reset)
	return c.send(ctx, "POST", "/v2/friend/facebook", q.String(), bearerHeaders(bearerToken), account, nil)
}

// ListFriendsOfFriends lists friends of the current user's friends.
func (c *apiClient) ListFriendsOfFriends(ctx context.Context, bearerToken string, limit *int, cursor string) (*ApiFriendsOfFriendsList, error) {
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiFriendsOfFriendsList{}
	if err := c.send(ctx, "GET", "/v2/friend/friends", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ImportSteamFriends imports Steam friends.
func (c *apiClient) ImportSteamFriends(ctx context.Context, bearerToken string, account *ApiAccountSteam, reset *bool) error {
	if account == nil {
		return fmt.Errorf("nakama: 'account' is required")
	}
	q := queryBuilder{}
	q.addBoolPtr("reset", reset)
	return c.send(ctx, "POST", "/v2/friend/steam", q.String(), bearerHeaders(bearerToken), account, nil)
}

// ===== Groups =====

// ListGroups lists groups based on filters.
func (c *apiClient) ListGroups(ctx context.Context, bearerToken, name, cursor string, limit *int, langTag string, members *int, open *bool) (*ApiGroupList, error) {
	q := queryBuilder{}
	if name != "" {
		q.addString("name", name)
	}
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	q.addIntPtr("limit", limit)
	if langTag != "" {
		q.addString("lang_tag", langTag)
	}
	q.addIntPtr("members", members)
	q.addBoolPtr("open", open)
	out := &ApiGroupList{}
	if err := c.send(ctx, "GET", "/v2/group", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateGroup creates a new group.
func (c *apiClient) CreateGroup(ctx context.Context, bearerToken string, body *ApiCreateGroupRequest) (*ApiGroup, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiGroup{}
	if err := c.send(ctx, "POST", "/v2/group", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteGroup deletes a group by id.
func (c *apiClient) DeleteGroup(ctx context.Context, bearerToken, groupId string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	return c.send(ctx, "DELETE", pathReplace("/v2/group/{groupId}", "groupId", groupId), "", bearerHeaders(bearerToken), nil, nil)
}

// UpdateGroup updates fields in a group.
func (c *apiClient) UpdateGroup(ctx context.Context, bearerToken, groupId string, body *ApiUpdateGroupRequest) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "PUT", pathReplace("/v2/group/{groupId}", "groupId", groupId), "", bearerHeaders(bearerToken), body, nil)
}

// AddGroupUsers adds users to a group.
func (c *apiClient) AddGroupUsers(ctx context.Context, bearerToken, groupId string, userIds []string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addStrings("user_ids", userIds)
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/add", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, nil)
}

// BanGroupUsers bans users from a group.
func (c *apiClient) BanGroupUsers(ctx context.Context, bearerToken, groupId string, userIds []string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addStrings("user_ids", userIds)
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/ban", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, nil)
}

// DemoteGroupUsers demotes users in a group.
func (c *apiClient) DemoteGroupUsers(ctx context.Context, bearerToken, groupId string, userIds []string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addStrings("user_ids", userIds)
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/demote", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, nil)
}

// JoinGroup joins (or requests to join) a group.
func (c *apiClient) JoinGroup(ctx context.Context, bearerToken, groupId string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/join", "groupId", groupId), "", bearerHeaders(bearerToken), nil, nil)
}

// KickGroupUsers kicks users from a group.
func (c *apiClient) KickGroupUsers(ctx context.Context, bearerToken, groupId string, userIds []string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addStrings("user_ids", userIds)
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/kick", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, nil)
}

// LeaveGroup leaves a group.
func (c *apiClient) LeaveGroup(ctx context.Context, bearerToken, groupId string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/leave", "groupId", groupId), "", bearerHeaders(bearerToken), nil, nil)
}

// PromoteGroupUsers promotes users in a group.
func (c *apiClient) PromoteGroupUsers(ctx context.Context, bearerToken, groupId string, userIds []string) error {
	if groupId == "" {
		return fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addStrings("user_ids", userIds)
	return c.send(ctx, "POST", pathReplace("/v2/group/{groupId}/promote", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, nil)
}

// ListGroupUsers lists users in a group.
func (c *apiClient) ListGroupUsers(ctx context.Context, bearerToken, groupId string, limit, state *int, cursor string) (*ApiGroupUserList, error) {
	if groupId == "" {
		return nil, fmt.Errorf("nakama: 'groupId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addIntPtr("state", state)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiGroupUserList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/group/{groupId}/user", "groupId", groupId), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== IAP =====

// ValidatePurchaseApple validates an Apple App Store purchase receipt.
func (c *apiClient) ValidatePurchaseApple(ctx context.Context, bearerToken string, body *ApiValidatePurchaseAppleRequest) (*ApiValidatePurchaseResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidatePurchaseResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/purchase/apple", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidatePurchaseFacebookInstant validates a Facebook Instant purchase.
func (c *apiClient) ValidatePurchaseFacebookInstant(ctx context.Context, bearerToken string, body *ApiValidatePurchaseFacebookInstantRequest) (*ApiValidatePurchaseResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidatePurchaseResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/purchase/facebookinstant", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidatePurchaseGoogle validates a Google Play Store purchase.
func (c *apiClient) ValidatePurchaseGoogle(ctx context.Context, bearerToken string, body *ApiValidatePurchaseGoogleRequest) (*ApiValidatePurchaseResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidatePurchaseResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/purchase/google", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidatePurchaseHuawei validates a Huawei AppGallery purchase.
func (c *apiClient) ValidatePurchaseHuawei(ctx context.Context, bearerToken string, body *ApiValidatePurchaseHuaweiRequest) (*ApiValidatePurchaseResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidatePurchaseResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/purchase/huawei", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListSubscriptions lists user subscriptions.
func (c *apiClient) ListSubscriptions(ctx context.Context, bearerToken string, body *ApiListSubscriptionsRequest) (*ApiSubscriptionList, error) {
	if body == nil {
		body = &ApiListSubscriptionsRequest{}
	}
	out := &ApiSubscriptionList{}
	if err := c.send(ctx, "POST", "/v2/iap/subscription", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateSubscriptionApple validates an Apple subscription receipt.
func (c *apiClient) ValidateSubscriptionApple(ctx context.Context, bearerToken string, body *ApiValidateSubscriptionAppleRequest) (*ApiValidateSubscriptionResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidateSubscriptionResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/subscription/apple", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateSubscriptionGoogle validates a Google subscription receipt.
func (c *apiClient) ValidateSubscriptionGoogle(ctx context.Context, bearerToken string, body *ApiValidateSubscriptionGoogleRequest) (*ApiValidateSubscriptionResponse, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiValidateSubscriptionResponse{}
	if err := c.send(ctx, "POST", "/v2/iap/subscription/google", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSubscription retrieves a subscription by product id.
func (c *apiClient) GetSubscription(ctx context.Context, bearerToken, productId string) (*ApiValidatedSubscription, error) {
	if productId == "" {
		return nil, fmt.Errorf("nakama: 'productId' is required")
	}
	out := &ApiValidatedSubscription{}
	if err := c.send(ctx, "GET", pathReplace("/v2/iap/subscription/{productId}", "productId", productId), "", bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Leaderboards =====

// DeleteLeaderboardRecord removes the user's record from a leaderboard.
func (c *apiClient) DeleteLeaderboardRecord(ctx context.Context, bearerToken, leaderboardId string) error {
	if leaderboardId == "" {
		return fmt.Errorf("nakama: 'leaderboardId' is required")
	}
	return c.send(ctx, "DELETE", pathReplace("/v2/leaderboard/{leaderboardId}", "leaderboardId", leaderboardId), "", bearerHeaders(bearerToken), nil, nil)
}

// ListLeaderboardRecords lists records from a leaderboard.
func (c *apiClient) ListLeaderboardRecords(ctx context.Context, bearerToken, leaderboardId string, ownerIds []string, limit *int, cursor, expiry string) (*ApiLeaderboardRecordList, error) {
	if leaderboardId == "" {
		return nil, fmt.Errorf("nakama: 'leaderboardId' is required")
	}
	q := queryBuilder{}
	q.addStrings("owner_ids", ownerIds)
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	if expiry != "" {
		q.addString("expiry", expiry)
	}
	out := &ApiLeaderboardRecordList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/leaderboard/{leaderboardId}", "leaderboardId", leaderboardId), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WriteLeaderboardRecord writes a record to a leaderboard.
func (c *apiClient) WriteLeaderboardRecord(ctx context.Context, bearerToken, leaderboardId string, body *WriteLeaderboardRecordRequestLeaderboardRecordWrite) (*ApiLeaderboardRecord, error) {
	if leaderboardId == "" {
		return nil, fmt.Errorf("nakama: 'leaderboardId' is required")
	}
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiLeaderboardRecord{}
	if err := c.send(ctx, "POST", pathReplace("/v2/leaderboard/{leaderboardId}", "leaderboardId", leaderboardId), "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListLeaderboardRecordsAroundOwner lists records around a given owner.
func (c *apiClient) ListLeaderboardRecordsAroundOwner(ctx context.Context, bearerToken, leaderboardId, ownerId string, limit *int, expiry, cursor string) (*ApiLeaderboardRecordList, error) {
	if leaderboardId == "" {
		return nil, fmt.Errorf("nakama: 'leaderboardId' is required")
	}
	if ownerId == "" {
		return nil, fmt.Errorf("nakama: 'ownerId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	if expiry != "" {
		q.addString("expiry", expiry)
	}
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiLeaderboardRecordList{}
	path := pathReplace(pathReplace("/v2/leaderboard/{leaderboardId}/owner/{ownerId}", "leaderboardId", leaderboardId), "ownerId", ownerId)
	if err := c.send(ctx, "GET", path, q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Match =====

// ListMatches lists realtime matches.
func (c *apiClient) ListMatches(ctx context.Context, bearerToken string, limit *int, authoritative *bool, label string, minSize, maxSize *int, query string) (*ApiMatchList, error) {
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addBoolPtr("authoritative", authoritative)
	if label != "" {
		q.addString("label", label)
	}
	q.addIntPtr("min_size", minSize)
	q.addIntPtr("max_size", maxSize)
	if query != "" {
		q.addString("query", query)
	}
	out := &ApiMatchList{}
	if err := c.send(ctx, "GET", "/v2/match", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetMatchmakerStats returns matchmaker statistics.
func (c *apiClient) GetMatchmakerStats(ctx context.Context, bearerToken string) (*ApiMatchmakerStats, error) {
	out := &ApiMatchmakerStats{}
	if err := c.send(ctx, "GET", "/v2/matchmaker/stats", "", bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Notifications =====

// DeleteNotifications deletes notifications by id.
func (c *apiClient) DeleteNotifications(ctx context.Context, bearerToken string, ids []string) error {
	q := queryBuilder{}
	q.addStrings("ids", ids)
	return c.send(ctx, "DELETE", "/v2/notification", q.String(), bearerHeaders(bearerToken), nil, nil)
}

// ListNotifications lists notifications for the current user.
func (c *apiClient) ListNotifications(ctx context.Context, bearerToken string, limit *int, cacheableCursor string) (*ApiNotificationList, error) {
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	if cacheableCursor != "" {
		q.addString("cacheable_cursor", cacheableCursor)
	}
	out := &ApiNotificationList{}
	if err := c.send(ctx, "GET", "/v2/notification", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Parties =====

// ListParties lists advertised parties.
func (c *apiClient) ListParties(ctx context.Context, bearerToken string, limit *int, open *bool, query, cursor string) (*ApiPartyList, error) {
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addBoolPtr("open", open)
	if query != "" {
		q.addString("query", query)
	}
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiPartyList{}
	if err := c.send(ctx, "GET", "/v2/party", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== RPC =====

// RpcGet executes a server-registered RPC via GET.
// Either bearerToken or basicUser/basicPass plus optional httpKey may be used.
func (c *apiClient) RpcGet(ctx context.Context, bearerToken, basicUser, basicPass, id, payload, httpKey string) (*ApiRpc, error) {
	if id == "" {
		return nil, fmt.Errorf("nakama: 'id' is required")
	}
	q := queryBuilder{}
	if payload != "" {
		q.addString("payload", payload)
	}
	if httpKey != "" {
		q.addString("http_key", httpKey)
	}

	headers := map[string]string{}
	if bearerToken != "" {
		headers["Authorization"] = "Bearer " + bearerToken
	} else if basicUser != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(basicUser + ":" + basicPass))
		headers["Authorization"] = "Basic " + creds
	}
	out := &ApiRpc{}
	if err := c.send(ctx, "GET", pathReplace("/v2/rpc/{id}", "id", id), q.String(), headers, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RpcPost executes a server-registered RPC via POST.
func (c *apiClient) RpcPost(ctx context.Context, bearerToken, basicUser, basicPass, id, payload, httpKey string) (*ApiRpc, error) {
	if id == "" {
		return nil, fmt.Errorf("nakama: 'id' is required")
	}
	q := queryBuilder{}
	if httpKey != "" {
		q.addString("http_key", httpKey)
	}
	headers := map[string]string{}
	if bearerToken != "" {
		headers["Authorization"] = "Bearer " + bearerToken
	} else if basicUser != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(basicUser + ":" + basicPass))
		headers["Authorization"] = "Basic " + creds
	}
	// The Nakama server expects the payload to be a JSON-encoded string.
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out := &ApiRpc{}
	if err := c.send(ctx, "POST", pathReplace("/v2/rpc/{id}", "id", id), q.String(), headers, bodyJSON, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Session =====

// SessionLogout invalidates a session and/or refresh token.
func (c *apiClient) SessionLogout(ctx context.Context, bearerToken string, body *ApiSessionLogoutRequest) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "POST", "/v2/session/logout", "", bearerHeaders(bearerToken), body, nil)
}

// ===== Storage =====

// ReadStorageObjects reads storage objects.
func (c *apiClient) ReadStorageObjects(ctx context.Context, bearerToken string, body *ApiReadStorageObjectsRequest) (*ApiStorageObjects, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiStorageObjects{}
	if err := c.send(ctx, "POST", "/v2/storage", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WriteStorageObjects writes storage objects.
func (c *apiClient) WriteStorageObjects(ctx context.Context, bearerToken string, body *ApiWriteStorageObjectsRequest) (*ApiStorageObjectAcks, error) {
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiStorageObjectAcks{}
	if err := c.send(ctx, "PUT", "/v2/storage", "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteStorageObjects deletes storage objects.
func (c *apiClient) DeleteStorageObjects(ctx context.Context, bearerToken string, body *ApiDeleteStorageObjectsRequest) error {
	if body == nil {
		return fmt.Errorf("nakama: 'body' is required")
	}
	return c.send(ctx, "PUT", "/v2/storage/delete", "", bearerHeaders(bearerToken), body, nil)
}

// ListStorageObjects lists storage objects in a collection.
func (c *apiClient) ListStorageObjects(ctx context.Context, bearerToken, collection, userId string, limit *int, cursor string) (*ApiStorageObjectList, error) {
	if collection == "" {
		return nil, fmt.Errorf("nakama: 'collection' is required")
	}
	q := queryBuilder{}
	if userId != "" {
		q.addString("user_id", userId)
	}
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiStorageObjectList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/storage/{collection}", "collection", collection), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListStorageObjects2 lists storage objects in a collection for a specific user.
func (c *apiClient) ListStorageObjects2(ctx context.Context, bearerToken, collection, userId string, limit *int, cursor string) (*ApiStorageObjectList, error) {
	if collection == "" {
		return nil, fmt.Errorf("nakama: 'collection' is required")
	}
	if userId == "" {
		return nil, fmt.Errorf("nakama: 'userId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	path := pathReplace(pathReplace("/v2/storage/{collection}/{userId}", "collection", collection), "userId", userId)
	out := &ApiStorageObjectList{}
	if err := c.send(ctx, "GET", path, q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Tournaments =====

// ListTournaments lists tournaments on the server.
func (c *apiClient) ListTournaments(ctx context.Context, bearerToken string, categoryStart, categoryEnd, startTime, endTime, limit *int, cursor string) (*ApiTournamentList, error) {
	q := queryBuilder{}
	q.addIntPtr("category_start", categoryStart)
	q.addIntPtr("category_end", categoryEnd)
	q.addIntPtr("start_time", startTime)
	q.addIntPtr("end_time", endTime)
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiTournamentList{}
	if err := c.send(ctx, "GET", "/v2/tournament", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteTournamentRecord deletes the user's tournament record.
func (c *apiClient) DeleteTournamentRecord(ctx context.Context, bearerToken, tournamentId string) error {
	if tournamentId == "" {
		return fmt.Errorf("nakama: 'tournamentId' is required")
	}
	return c.send(ctx, "DELETE", pathReplace("/v2/tournament/{tournamentId}", "tournamentId", tournamentId), "", bearerHeaders(bearerToken), nil, nil)
}

// ListTournamentRecords lists tournament records.
func (c *apiClient) ListTournamentRecords(ctx context.Context, bearerToken, tournamentId string, ownerIds []string, limit *int, cursor, expiry string) (*ApiTournamentRecordList, error) {
	if tournamentId == "" {
		return nil, fmt.Errorf("nakama: 'tournamentId' is required")
	}
	q := queryBuilder{}
	q.addStrings("owner_ids", ownerIds)
	q.addIntPtr("limit", limit)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	if expiry != "" {
		q.addString("expiry", expiry)
	}
	out := &ApiTournamentRecordList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/tournament/{tournamentId}", "tournamentId", tournamentId), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WriteTournamentRecord writes a tournament record (PUT).
func (c *apiClient) WriteTournamentRecord(ctx context.Context, bearerToken, tournamentId string, body *WriteTournamentRecordRequestTournamentRecordWrite) (*ApiLeaderboardRecord, error) {
	if tournamentId == "" {
		return nil, fmt.Errorf("nakama: 'tournamentId' is required")
	}
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiLeaderboardRecord{}
	if err := c.send(ctx, "PUT", pathReplace("/v2/tournament/{tournamentId}", "tournamentId", tournamentId), "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WriteTournamentRecord2 writes a tournament record (POST).
func (c *apiClient) WriteTournamentRecord2(ctx context.Context, bearerToken, tournamentId string, body *WriteTournamentRecordRequestTournamentRecordWrite) (*ApiLeaderboardRecord, error) {
	if tournamentId == "" {
		return nil, fmt.Errorf("nakama: 'tournamentId' is required")
	}
	if body == nil {
		return nil, fmt.Errorf("nakama: 'body' is required")
	}
	out := &ApiLeaderboardRecord{}
	if err := c.send(ctx, "POST", pathReplace("/v2/tournament/{tournamentId}", "tournamentId", tournamentId), "", bearerHeaders(bearerToken), body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// JoinTournament joins a tournament by id.
func (c *apiClient) JoinTournament(ctx context.Context, bearerToken, tournamentId string) error {
	if tournamentId == "" {
		return fmt.Errorf("nakama: 'tournamentId' is required")
	}
	return c.send(ctx, "POST", pathReplace("/v2/tournament/{tournamentId}/join", "tournamentId", tournamentId), "", bearerHeaders(bearerToken), nil, nil)
}

// ListTournamentRecordsAroundOwner lists records around a given owner.
func (c *apiClient) ListTournamentRecordsAroundOwner(ctx context.Context, bearerToken, tournamentId, ownerId string, limit *int, expiry, cursor string) (*ApiTournamentRecordList, error) {
	if tournamentId == "" {
		return nil, fmt.Errorf("nakama: 'tournamentId' is required")
	}
	if ownerId == "" {
		return nil, fmt.Errorf("nakama: 'ownerId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	if expiry != "" {
		q.addString("expiry", expiry)
	}
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	path := pathReplace(pathReplace("/v2/tournament/{tournamentId}/owner/{ownerId}", "tournamentId", tournamentId), "ownerId", ownerId)
	out := &ApiTournamentRecordList{}
	if err := c.send(ctx, "GET", path, q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===== Users =====

// GetUsers fetches users by id, username, and/or facebook id.
func (c *apiClient) GetUsers(ctx context.Context, bearerToken string, ids, usernames, facebookIds []string) (*ApiUsers, error) {
	q := queryBuilder{}
	q.addStrings("ids", ids)
	q.addStrings("usernames", usernames)
	q.addStrings("facebook_ids", facebookIds)
	out := &ApiUsers{}
	if err := c.send(ctx, "GET", "/v2/user", q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListUserGroups lists groups the user is a member of.
func (c *apiClient) ListUserGroups(ctx context.Context, bearerToken, userId string, limit, state *int, cursor string) (*ApiUserGroupList, error) {
	if userId == "" {
		return nil, fmt.Errorf("nakama: 'userId' is required")
	}
	q := queryBuilder{}
	q.addIntPtr("limit", limit)
	q.addIntPtr("state", state)
	if cursor != "" {
		q.addString("cursor", cursor)
	}
	out := &ApiUserGroupList{}
	if err := c.send(ctx, "GET", pathReplace("/v2/user/{userId}/group", "userId", userId), q.String(), bearerHeaders(bearerToken), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
