// Package nakama is a Go client SDK for Nakama server.
//
// This file contains the data transfer objects (DTOs) used by the Nakama API.
// It is a port of Nakama/ApiClient.gen.cs from the .NET SDK.
package nakama

import "fmt"

// ApiOperator is the operator that can be used to override the one set in the leaderboard.
type ApiOperator int

const (
	OperatorNoOverride ApiOperator = 0
	OperatorBest       ApiOperator = 1
	OperatorSet        ApiOperator = 2
	OperatorIncrement  ApiOperator = 3
	OperatorDecrement  ApiOperator = 4
)

// ApiStoreEnvironment - Environment where a purchase/subscription took place.
type ApiStoreEnvironment int

const (
	StoreEnvUnknown    ApiStoreEnvironment = 0
	StoreEnvSandbox    ApiStoreEnvironment = 1
	StoreEnvProduction ApiStoreEnvironment = 2
)

// ApiStoreProvider - Validation provider.
type ApiStoreProvider int

const (
	StoreProviderAppleAppStore       ApiStoreProvider = 0
	StoreProviderGooglePlayStore     ApiStoreProvider = 1
	StoreProviderHuaweiAppGallery    ApiStoreProvider = 2
	StoreProviderFacebookInstantStore ApiStoreProvider = 3
)

// ApiResponseError is returned when an HTTP response is not successful.
type ApiResponseError struct {
	StatusCode     int    `json:"-"`
	GrpcStatusCode int    `json:"code"`
	Message        string `json:"message"`
}

func (e *ApiResponseError) Error() string {
	return fmt.Sprintf("ApiResponseError(StatusCode=%d, Message='%s', GrpcStatusCode=%d)",
		e.StatusCode, e.Message, e.GrpcStatusCode)
}

// ApiAccount is a user with additional account details. Always the current user.
type ApiAccount struct {
	CustomId    string             `json:"custom_id,omitempty"`
	Devices     []*ApiAccountDevice `json:"devices,omitempty"`
	DisableTime string             `json:"disable_time,omitempty"`
	Email       string             `json:"email,omitempty"`
	User        *ApiUser           `json:"user,omitempty"`
	VerifyTime  string             `json:"verify_time,omitempty"`
	Wallet      string             `json:"wallet,omitempty"`
}

// ApiAccountApple - Send an Apple Sign In token to the server.
type ApiAccountApple struct {
	Token string            `json:"token,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// ApiAccountCustom - Send a custom ID to the server.
type ApiAccountCustom struct {
	Id   string            `json:"id,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
}

// ApiAccountDevice - Send a device to the server.
type ApiAccountDevice struct {
	Id   string            `json:"id,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
}

// ApiAccountEmail - Send an email and password to the server.
type ApiAccountEmail struct {
	Email    string            `json:"email,omitempty"`
	Password string            `json:"password,omitempty"`
	Vars     map[string]string `json:"vars,omitempty"`
}

// ApiAccountFacebook - Send a Facebook token to the server.
type ApiAccountFacebook struct {
	Token string            `json:"token,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// ApiAccountFacebookInstantGame - Send a Facebook Instant Game token to the server.
type ApiAccountFacebookInstantGame struct {
	SignedPlayerInfo string            `json:"signed_player_info,omitempty"`
	Vars             map[string]string `json:"vars,omitempty"`
}

// ApiAccountGameCenter - Send Apple's Game Center credentials to the server.
type ApiAccountGameCenter struct {
	BundleId         string            `json:"bundle_id,omitempty"`
	PlayerId         string            `json:"player_id,omitempty"`
	PublicKeyUrl     string            `json:"public_key_url,omitempty"`
	Salt             string            `json:"salt,omitempty"`
	Signature        string            `json:"signature,omitempty"`
	TimestampSeconds string            `json:"timestamp_seconds,omitempty"`
	Vars             map[string]string `json:"vars,omitempty"`
}

// ApiAccountGoogle - Send a Google token to the server.
type ApiAccountGoogle struct {
	Token string            `json:"token,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// ApiAccountSteam - Send a Steam token to the server.
type ApiAccountSteam struct {
	Token string            `json:"token,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// ApiChannelMessage - A message sent on a channel.
type ApiChannelMessage struct {
	ChannelId  string `json:"channel_id,omitempty"`
	Code       int    `json:"code,omitempty"`
	Content    string `json:"content,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	GroupId    string `json:"group_id,omitempty"`
	MessageId  string `json:"message_id,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	RoomName   string `json:"room_name,omitempty"`
	SenderId   string `json:"sender_id,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`
	UserIdOne  string `json:"user_id_one,omitempty"`
	UserIdTwo  string `json:"user_id_two,omitempty"`
	Username   string `json:"username,omitempty"`
}

// ApiChannelMessageList - A list of channel messages, usually a result of a list operation.
type ApiChannelMessageList struct {
	CacheableCursor string               `json:"cacheable_cursor,omitempty"`
	Messages        []*ApiChannelMessage `json:"messages,omitempty"`
	NextCursor      string               `json:"next_cursor,omitempty"`
	PrevCursor      string               `json:"prev_cursor,omitempty"`
}

// ApiCreateGroupRequest - Create a group with the current user as owner.
type ApiCreateGroupRequest struct {
	AvatarUrl   string `json:"avatar_url,omitempty"`
	Description string `json:"description,omitempty"`
	LangTag     string `json:"lang_tag,omitempty"`
	MaxCount    int    `json:"max_count,omitempty"`
	Name        string `json:"name,omitempty"`
	Open        bool   `json:"open,omitempty"`
}

// ApiDeleteStorageObjectId - Storage objects to delete.
type ApiDeleteStorageObjectId struct {
	Collection string `json:"collection,omitempty"`
	Key        string `json:"key,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ApiDeleteStorageObjectsRequest - Batch delete storage objects.
type ApiDeleteStorageObjectsRequest struct {
	ObjectIds []*ApiDeleteStorageObjectId `json:"object_ids,omitempty"`
}

// ApiEvent - Represents an event to be passed through the server's event handling system.
type ApiEvent struct {
	External   bool              `json:"external,omitempty"`
	Name       string            `json:"name,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
}

// ApiFriend - A friend of a user.
type ApiFriend struct {
	State      *int     `json:"state,omitempty"`
	UpdateTime string   `json:"update_time,omitempty"`
	User       *ApiUser `json:"user,omitempty"`
}

// ApiFriendList - A collection of zero or more friends of the user.
type ApiFriendList struct {
	Cursor  string       `json:"cursor,omitempty"`
	Friends []*ApiFriend `json:"friends,omitempty"`
}

// ApiFriendsOfFriendsList - A collection of zero or more friends of friends.
type ApiFriendsOfFriendsList struct {
	Cursor           string                                  `json:"cursor,omitempty"`
	FriendsOfFriends []*FriendsOfFriendsListFriendOfFriend `json:"friends_of_friends,omitempty"`
}

// FriendsOfFriendsListFriendOfFriend - A friend of a friend.
type FriendsOfFriendsListFriendOfFriend struct {
	Referrer string   `json:"referrer,omitempty"`
	User     *ApiUser `json:"user,omitempty"`
}

// ApiGroup - A group on the server.
type ApiGroup struct {
	AvatarUrl   string `json:"avatar_url,omitempty"`
	CreateTime  string `json:"create_time,omitempty"`
	CreatorId   string `json:"creator_id,omitempty"`
	Description string `json:"description,omitempty"`
	EdgeCount   int    `json:"edge_count,omitempty"`
	Id          string `json:"id,omitempty"`
	LangTag     string `json:"lang_tag,omitempty"`
	MaxCount    int    `json:"max_count,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
	Name        string `json:"name,omitempty"`
	Open        bool   `json:"open,omitempty"`
	UpdateTime  string `json:"update_time,omitempty"`
}

// ApiGroupList - One or more groups returned from a listing operation.
type ApiGroupList struct {
	Cursor string      `json:"cursor,omitempty"`
	Groups []*ApiGroup `json:"groups,omitempty"`
}

// ApiGroupUserList - A list of users belonging to a group, along with their role.
type ApiGroupUserList struct {
	Cursor     string                    `json:"cursor,omitempty"`
	GroupUsers []*GroupUserListGroupUser `json:"group_users,omitempty"`
}

// GroupUserListGroupUser - A single user-role pair.
type GroupUserListGroupUser struct {
	State *int     `json:"state,omitempty"`
	User  *ApiUser `json:"user,omitempty"`
}

// ApiLeaderboardRecord - Represents a record from a leaderboard or tournament.
type ApiLeaderboardRecord struct {
	CreateTime    string            `json:"create_time,omitempty"`
	ExpiryTime    string            `json:"expiry_time,omitempty"`
	LeaderboardId string            `json:"leaderboard_id,omitempty"`
	MaxNumScore   int               `json:"max_num_score,omitempty"`
	Metadata      string            `json:"metadata,omitempty"`
	NumScore      int               `json:"num_score,omitempty"`
	OwnerId       string            `json:"owner_id,omitempty"`
	Rank          string            `json:"rank,omitempty"`
	Score         string            `json:"score,omitempty"`
	Subscore      string            `json:"subscore,omitempty"`
	UpdateTime    string            `json:"update_time,omitempty"`
	Username      string            `json:"username,omitempty"`
}

// ApiLeaderboardRecordList - A set of leaderboard records, fetched via list operations.
type ApiLeaderboardRecordList struct {
	NextCursor   string                  `json:"next_cursor,omitempty"`
	OwnerRecords []*ApiLeaderboardRecord `json:"owner_records,omitempty"`
	PrevCursor   string                  `json:"prev_cursor,omitempty"`
	RankCount    string                  `json:"rank_count,omitempty"`
	Records      []*ApiLeaderboardRecord `json:"records,omitempty"`
}

// ApiLinkSteamRequest - Link a Steam profile to a user account.
type ApiLinkSteamRequest struct {
	Account *ApiAccountSteam `json:"account,omitempty"`
	Sync    bool             `json:"sync,omitempty"`
}

// ApiListSubscriptionsRequest - List user's subscriptions.
type ApiListSubscriptionsRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  *int   `json:"limit,omitempty"`
}

// ApiMatch - Represents a realtime match.
type ApiMatch struct {
	Authoritative bool   `json:"authoritative,omitempty"`
	HandlerName   string `json:"handler_name,omitempty"`
	Label         string `json:"label,omitempty"`
	MatchId       string `json:"match_id,omitempty"`
	Size          int    `json:"size,omitempty"`
	TickRate      int    `json:"tick_rate,omitempty"`
}

// ApiMatchList - A list of realtime matches.
type ApiMatchList struct {
	Matches []*ApiMatch `json:"matches,omitempty"`
}

// ApiMatchmakerCompletionStats - Matchmaker completion statistics.
type ApiMatchmakerCompletionStats struct {
	EmptySec  int `json:"empty_sec,omitempty"`
	FilledSec int `json:"filled_sec,omitempty"`
}

// ApiMatchmakerStats - Matchmaker stats.
type ApiMatchmakerStats struct {
	ActiveCount     int                            `json:"active_count,omitempty"`
	CompletionStats []*ApiMatchmakerCompletionStats `json:"completion_stats,omitempty"`
	OldestTicketCreateTime int                     `json:"oldest_ticket_create_time,omitempty"`
}

// ApiNotification - A notification.
type ApiNotification struct {
	Code       int    `json:"code,omitempty"`
	Content    string `json:"content,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	Id         string `json:"id,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	SenderId   string `json:"sender_id,omitempty"`
	Subject    string `json:"subject,omitempty"`
}

// ApiNotificationList - A list of notifications.
type ApiNotificationList struct {
	CacheableCursor string             `json:"cacheable_cursor,omitempty"`
	Notifications   []*ApiNotification `json:"notifications,omitempty"`
}

// ApiParty - Incoming information about a party.
type ApiParty struct {
	Hidden    bool            `json:"hidden,omitempty"`
	Id        string          `json:"id,omitempty"`
	Label     string          `json:"label,omitempty"`
	Leader    *UserPresence   `json:"leader,omitempty"`
	MaxSize   int             `json:"max_size,omitempty"`
	Open      bool            `json:"open,omitempty"`
	Presences []*UserPresence `json:"presences,omitempty"`
	Self      *UserPresence   `json:"self,omitempty"`
}

// ApiPartyList - List of parties for listing.
type ApiPartyList struct {
	Cursor  string      `json:"cursor,omitempty"`
	Parties []*ApiParty `json:"parties,omitempty"`
}

// ApiReadStorageObjectId - Storage object identifier to read.
type ApiReadStorageObjectId struct {
	Collection string `json:"collection,omitempty"`
	Key        string `json:"key,omitempty"`
	UserId     string `json:"user_id,omitempty"`
}

// ApiReadStorageObjectsRequest - Batch read multiple storage objects.
type ApiReadStorageObjectsRequest struct {
	ObjectIds []*ApiReadStorageObjectId `json:"object_ids,omitempty"`
}

// ApiRpc - Execute an RPC function on the server.
type ApiRpc struct {
	HttpKey string `json:"http_key,omitempty"`
	Id      string `json:"id,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// ApiSession - A user's session, used to authenticate API calls.
type ApiSession struct {
	Created      bool   `json:"created,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Token        string `json:"token,omitempty"`
}

// ApiSessionLogoutRequest - Log out a session, invalidating tokens.
type ApiSessionLogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
	Token        string `json:"token,omitempty"`
}

// ApiSessionRefreshRequest - Refresh a user's session.
type ApiSessionRefreshRequest struct {
	Token string            `json:"token,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// ApiStorageObject - An object stored on the server.
type ApiStorageObject struct {
	Collection      string `json:"collection,omitempty"`
	CreateTime      string `json:"create_time,omitempty"`
	Key             string `json:"key,omitempty"`
	PermissionRead  int    `json:"permission_read,omitempty"`
	PermissionWrite int    `json:"permission_write,omitempty"`
	UpdateTime      string `json:"update_time,omitempty"`
	UserId          string `json:"user_id,omitempty"`
	Value           string `json:"value,omitempty"`
	Version         string `json:"version,omitempty"`
}

// ApiStorageObjectAck - A storage acknowledgement.
type ApiStorageObjectAck struct {
	Collection string `json:"collection,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	Key        string `json:"key,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`
	UserId     string `json:"user_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ApiStorageObjectAcks - A batch of storage write acknowledgements.
type ApiStorageObjectAcks struct {
	Acks []*ApiStorageObjectAck `json:"acks,omitempty"`
}

// ApiStorageObjectList - List of storage objects.
type ApiStorageObjectList struct {
	Cursor  string              `json:"cursor,omitempty"`
	Objects []*ApiStorageObject `json:"objects,omitempty"`
}

// ApiStorageObjects - Batch of storage objects.
type ApiStorageObjects struct {
	Objects []*ApiStorageObject `json:"objects,omitempty"`
}

// ApiSubscriptionList - A list of validated subscriptions stored by Nakama.
type ApiSubscriptionList struct {
	Cursor                string                     `json:"cursor,omitempty"`
	PrevCursor            string                     `json:"prev_cursor,omitempty"`
	ValidatedSubscriptions []*ApiValidatedSubscription `json:"validated_subscriptions,omitempty"`
}

// ApiTournament - A tournament on the server.
type ApiTournament struct {
	Authoritative bool   `json:"authoritative,omitempty"`
	CanEnter      bool   `json:"can_enter,omitempty"`
	Category      int    `json:"category,omitempty"`
	CreateTime    string `json:"create_time,omitempty"`
	Description   string `json:"description,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	EndActive     int    `json:"end_active,omitempty"`
	EndTime       string `json:"end_time,omitempty"`
	Id            string `json:"id,omitempty"`
	MaxNumScore   int    `json:"max_num_score,omitempty"`
	MaxSize       int    `json:"max_size,omitempty"`
	Metadata      string `json:"metadata,omitempty"`
	NextReset     int    `json:"next_reset,omitempty"`
	Operator      *ApiOperator `json:"operator,omitempty"`
	PrevReset     int    `json:"prev_reset,omitempty"`
	Size          int    `json:"size,omitempty"`
	SortOrder     int    `json:"sort_order,omitempty"`
	StartActive   int    `json:"start_active,omitempty"`
	StartTime     string `json:"start_time,omitempty"`
	Title         string `json:"title,omitempty"`
}

// ApiTournamentList - A list of tournaments.
type ApiTournamentList struct {
	Cursor      string           `json:"cursor,omitempty"`
	Tournaments []*ApiTournament `json:"tournaments,omitempty"`
}

// ApiTournamentRecordList - A list of tournament records.
type ApiTournamentRecordList struct {
	NextCursor   string                  `json:"next_cursor,omitempty"`
	OwnerRecords []*ApiLeaderboardRecord `json:"owner_records,omitempty"`
	PrevCursor   string                  `json:"prev_cursor,omitempty"`
	RankCount    string                  `json:"rank_count,omitempty"`
	Records      []*ApiLeaderboardRecord `json:"records,omitempty"`
}

// ApiUpdateAccountRequest - Update fields in the current user's account.
type ApiUpdateAccountRequest struct {
	AvatarUrl   string `json:"avatar_url,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	LangTag     string `json:"lang_tag,omitempty"`
	Location    string `json:"location,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Username    string `json:"username,omitempty"`
}

// ApiUpdateGroupRequest - Update fields in a given group.
type ApiUpdateGroupRequest struct {
	AvatarUrl   string `json:"avatar_url,omitempty"`
	Description string `json:"description,omitempty"`
	LangTag     string `json:"lang_tag,omitempty"`
	Name        string `json:"name,omitempty"`
	Open        bool   `json:"open,omitempty"`
}

// ApiUser - A user in the server.
type ApiUser struct {
	AppleId               string `json:"apple_id,omitempty"`
	AvatarUrl             string `json:"avatar_url,omitempty"`
	CreateTime            string `json:"create_time,omitempty"`
	DisplayName           string `json:"display_name,omitempty"`
	EdgeCount             int    `json:"edge_count,omitempty"`
	FacebookId            string `json:"facebook_id,omitempty"`
	FacebookInstantGameId string `json:"facebook_instant_game_id,omitempty"`
	GamecenterId          string `json:"gamecenter_id,omitempty"`
	GoogleId              string `json:"google_id,omitempty"`
	Id                    string `json:"id,omitempty"`
	LangTag               string `json:"lang_tag,omitempty"`
	Location              string `json:"location,omitempty"`
	Metadata              string `json:"metadata,omitempty"`
	Online                bool   `json:"online,omitempty"`
	SteamId               string `json:"steam_id,omitempty"`
	Timezone              string `json:"timezone,omitempty"`
	UpdateTime            string `json:"update_time,omitempty"`
	Username              string `json:"username,omitempty"`
}

// ApiUserGroupList - A list of groups belonging to a user, along with the user's role.
type ApiUserGroupList struct {
	Cursor     string                       `json:"cursor,omitempty"`
	UserGroups []*UserGroupListUserGroup    `json:"user_groups,omitempty"`
}

// UserGroupListUserGroup - A single group-role pair.
type UserGroupListUserGroup struct {
	Group *ApiGroup `json:"group,omitempty"`
	State *int      `json:"state,omitempty"`
}

// ApiUsers - A collection of zero or more users.
type ApiUsers struct {
	Users []*ApiUser `json:"users,omitempty"`
}

// ApiValidatePurchaseAppleRequest - Validate an Apple purchase receipt.
type ApiValidatePurchaseAppleRequest struct {
	Persist *bool  `json:"persist,omitempty"`
	Receipt string `json:"receipt,omitempty"`
}

// ApiValidatePurchaseFacebookInstantRequest - Validate a Facebook Instant purchase.
type ApiValidatePurchaseFacebookInstantRequest struct {
	Persist       *bool  `json:"persist,omitempty"`
	SignedRequest string `json:"signed_request,omitempty"`
}

// ApiValidatePurchaseGoogleRequest - Validate a Google purchase.
type ApiValidatePurchaseGoogleRequest struct {
	Persist *bool  `json:"persist,omitempty"`
	Purchase string `json:"purchase,omitempty"`
}

// ApiValidatePurchaseHuaweiRequest - Validate a Huawei purchase.
type ApiValidatePurchaseHuaweiRequest struct {
	Persist   *bool  `json:"persist,omitempty"`
	Purchase  string `json:"purchase,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// ApiValidatePurchaseResponse - Response for validating a purchase.
type ApiValidatePurchaseResponse struct {
	ValidatedPurchases []*ApiValidatedPurchase `json:"validated_purchases,omitempty"`
}

// ApiValidateSubscriptionAppleRequest - Validate an Apple subscription.
type ApiValidateSubscriptionAppleRequest struct {
	Persist *bool  `json:"persist,omitempty"`
	Receipt string `json:"receipt,omitempty"`
}

// ApiValidateSubscriptionGoogleRequest - Validate a Google subscription.
type ApiValidateSubscriptionGoogleRequest struct {
	Persist *bool  `json:"persist,omitempty"`
	Receipt string `json:"receipt,omitempty"`
}

// ApiValidateSubscriptionResponse - Response for validating a subscription.
type ApiValidateSubscriptionResponse struct {
	ValidatedSubscription *ApiValidatedSubscription `json:"validated_subscription,omitempty"`
}

// ApiValidatedPurchase - A validated purchase stored by Nakama.
type ApiValidatedPurchase struct {
	CreateTime       string              `json:"create_time,omitempty"`
	Environment      ApiStoreEnvironment `json:"environment,omitempty"`
	ProductId        string              `json:"product_id,omitempty"`
	ProviderResponse string              `json:"provider_response,omitempty"`
	PurchaseTime     string              `json:"purchase_time,omitempty"`
	RefundTime       string              `json:"refund_time,omitempty"`
	SeenBefore       bool                `json:"seen_before,omitempty"`
	Store            ApiStoreProvider    `json:"store,omitempty"`
	TransactionId    string              `json:"transaction_id,omitempty"`
	UpdateTime       string              `json:"update_time,omitempty"`
	UserId           string              `json:"user_id,omitempty"`
}

// ApiValidatedSubscription - A validated subscription stored by Nakama.
type ApiValidatedSubscription struct {
	Active                bool                `json:"active,omitempty"`
	CreateTime            string              `json:"create_time,omitempty"`
	Environment           ApiStoreEnvironment `json:"environment,omitempty"`
	ExpiryTime            string              `json:"expiry_time,omitempty"`
	OriginalTransactionId string              `json:"original_transaction_id,omitempty"`
	ProductId             string              `json:"product_id,omitempty"`
	ProviderNotification  string              `json:"provider_notification,omitempty"`
	ProviderResponse      string              `json:"provider_response,omitempty"`
	PurchaseTime          string              `json:"purchase_time,omitempty"`
	RefundTime            string              `json:"refund_time,omitempty"`
	Store                 ApiStoreProvider    `json:"store,omitempty"`
	UpdateTime            string              `json:"update_time,omitempty"`
	UserId                string              `json:"user_id,omitempty"`
}

// ApiWriteStorageObject - The object to write into the storage engine.
type ApiWriteStorageObject struct {
	Collection      string `json:"collection,omitempty"`
	Key             string `json:"key,omitempty"`
	PermissionRead  *int   `json:"permission_read,omitempty"`
	PermissionWrite *int   `json:"permission_write,omitempty"`
	Value           string `json:"value,omitempty"`
	Version         string `json:"version,omitempty"`
}

// ApiWriteStorageObjectsRequest - Batch write objects to the storage engine.
type ApiWriteStorageObjectsRequest struct {
	Objects []*ApiWriteStorageObject `json:"objects,omitempty"`
}

// WriteLeaderboardRecordRequestLeaderboardRecordWrite - Record values to write.
type WriteLeaderboardRecordRequestLeaderboardRecordWrite struct {
	Metadata string      `json:"metadata,omitempty"`
	Operator ApiOperator `json:"operator,omitempty"`
	Score    string      `json:"score,omitempty"`
	Subscore string      `json:"subscore,omitempty"`
}

// WriteTournamentRecordRequestTournamentRecordWrite - Record values to write to a tournament.
type WriteTournamentRecordRequestTournamentRecordWrite struct {
	Metadata string      `json:"metadata,omitempty"`
	Operator ApiOperator `json:"operator,omitempty"`
	Score    string      `json:"score,omitempty"`
	Subscore string      `json:"subscore,omitempty"`
}

// ProtobufAny - Used by RpcStatus details.
type ProtobufAny struct {
	TypeUrl string `json:"type_url,omitempty"`
	Value   string `json:"value,omitempty"`
}

// RpcStatus - Standard gRPC status.
type RpcStatus struct {
	Code    int            `json:"code,omitempty"`
	Details []*ProtobufAny `json:"details,omitempty"`
	Message string         `json:"message,omitempty"`
}

// StorageObjectId is a tuple identifying a storage object.
type StorageObjectId struct {
	Collection string `json:"collection,omitempty"`
	Key        string `json:"key,omitempty"`
	UserId     string `json:"user_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

