// Socket message types ported from Nakama/*Message*.cs and friends.
package nakama

import "encoding/json"

// UserPresence represents a connected user identified by a tuple of
// (node_id, user_id, session_id). Mirrors Nakama/IUserPresence.cs.
type UserPresence struct {
	Persistence bool   `json:"persistence,omitempty"`
	SessionId   string `json:"session_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Username    string `json:"username,omitempty"`
	UserId      string `json:"user_id,omitempty"`
}

// Stream is a realtime socket stream on the server.
type Stream struct {
	Descriptor string `json:"descriptor,omitempty"`
	Label      string `json:"label,omitempty"`
	Mode       int    `json:"mode,omitempty"`
	Subject    string `json:"subject,omitempty"`
}

// Channel is a chat channel on the server. Mirrors Nakama/IChannel.cs.
type Channel struct {
	Id        string         `json:"id,omitempty"`
	Presences []*UserPresence `json:"presences,omitempty"`
	Self      *UserPresence  `json:"self,omitempty"`
	RoomName  string         `json:"room_name,omitempty"`
	GroupId   string         `json:"group_id,omitempty"`
	UserIdOne string         `json:"user_id_one,omitempty"`
	UserIdTwo string         `json:"user_id_two,omitempty"`
}

// ChannelType describes which kind of channel to join.
type ChannelType int

const (
	ChannelTypeRoom        ChannelType = 1
	ChannelTypeDirectMessage ChannelType = 2
	ChannelTypeGroup       ChannelType = 3
)

// ChannelJoinMessage is sent to join a chat channel.
type ChannelJoinMessage struct {
	Hidden      bool   `json:"hidden,omitempty"`
	Persistence bool   `json:"persistence,omitempty"`
	Target      string `json:"target,omitempty"`
	Type        int    `json:"type,omitempty"`
}

// ChannelLeaveMessage is sent to leave a chat channel.
type ChannelLeaveMessage struct {
	ChannelId string `json:"channel_id,omitempty"`
}

// ChannelRemoveMessage removes a previously-sent message from a channel.
type ChannelRemoveMessage struct {
	ChannelId string `json:"channel_id,omitempty"`
	MessageId string `json:"message_id,omitempty"`
}

// ChannelSendMessage is a chat message sent to a channel.
type ChannelSendMessage struct {
	ChannelId string `json:"channel_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// ChannelUpdateMessage updates a previously-sent chat message.
type ChannelUpdateMessage struct {
	ChannelId string `json:"channel_id,omitempty"`
	Content   string `json:"content,omitempty"`
	MessageId string `json:"message_id,omitempty"`
}

// ChannelMessageAck is the server's acknowledgement of a chat message.
type ChannelMessageAck struct {
	ChannelId  string `json:"channel_id,omitempty"`
	Code       int    `json:"code,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	MessageId  string `json:"message_id,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`
	Username   string `json:"username,omitempty"`
	RoomName   string `json:"room_name,omitempty"`
	GroupId    string `json:"group_id,omitempty"`
	UserIdOne  string `json:"user_id_one,omitempty"`
	UserIdTwo  string `json:"user_id_two,omitempty"`
}

// ChannelPresenceEvent is a presence change for a chat channel.
type ChannelPresenceEvent struct {
	ChannelId string         `json:"channel_id,omitempty"`
	Joins     []*UserPresence `json:"joins,omitempty"`
	Leaves    []*UserPresence `json:"leaves,omitempty"`
	RoomName  string         `json:"room_name,omitempty"`
	GroupId   string         `json:"group_id,omitempty"`
	UserIdOne string         `json:"user_id_one,omitempty"`
	UserIdTwo string         `json:"user_id_two,omitempty"`
}

// Match is a multiplayer match. Mirrors Nakama/IMatch.cs.
type Match struct {
	Authoritative bool           `json:"authoritative,omitempty"`
	Id            string         `json:"match_id,omitempty"`
	Label         string         `json:"label,omitempty"`
	Presences     []*UserPresence `json:"presences,omitempty"`
	Size          int            `json:"size,omitempty"`
	Self          *UserPresence  `json:"self,omitempty"`
}

// MatchCreateMessage is sent to create a new match.
type MatchCreateMessage struct {
	Name string `json:"name,omitempty"`
}

// MatchJoinMessage joins an existing match by id or matchmaker token.
type MatchJoinMessage struct {
	Id       string            `json:"match_id,omitempty"`
	Token    string            `json:"token,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MatchLeaveMessage leaves a multiplayer match.
type MatchLeaveMessage struct {
	MatchId string `json:"match_id,omitempty"`
}

// MatchSendMessage sends realtime data to a match.
type MatchSendMessage struct {
	MatchId   string         `json:"match_id,omitempty"`
	OpCode    string         `json:"op_code,omitempty"`
	Data      string         `json:"data,omitempty"`
	Presences []*UserPresence `json:"presences,omitempty"`
}

// MatchPresenceEvent describes joins and leaves for a match.
type MatchPresenceEvent struct {
	MatchId string         `json:"match_id,omitempty"`
	Joins   []*UserPresence `json:"joins,omitempty"`
	Leaves  []*UserPresence `json:"leaves,omitempty"`
}

// MatchState is realtime data received from a match.
type MatchState struct {
	MatchId      string        `json:"match_id,omitempty"`
	OpCode       string        `json:"op_code,omitempty"`
	Data         string        `json:"data,omitempty"`
	UserPresence *UserPresence `json:"presence,omitempty"`
}

// OpCodeInt parses the OpCode string field as an int64. Returns 0 on error.
func (m *MatchState) OpCodeInt() int64 {
	if m == nil || m.OpCode == "" {
		return 0
	}
	var n json.Number = json.Number(m.OpCode)
	v, _ := n.Int64()
	return v
}

// MatchmakerAddMessage joins the matchmaker pool.
type MatchmakerAddMessage struct {
	MinCount          int                `json:"min_count,omitempty"`
	MaxCount          int                `json:"max_count,omitempty"`
	Query             string             `json:"query,omitempty"`
	StringProperties  map[string]string  `json:"string_properties,omitempty"`
	NumericProperties map[string]float64 `json:"numeric_properties,omitempty"`
	CountMultiple     *int               `json:"count_multiple,omitempty"`
}

// MatchmakerRemoveMessage cancels a matchmaker ticket.
type MatchmakerRemoveMessage struct {
	Ticket string `json:"ticket,omitempty"`
}

// MatchmakerTicket is returned when the user has joined the matchmaker.
type MatchmakerTicket struct {
	Ticket string `json:"ticket,omitempty"`
}

// MatchmakerUser is a peer in a matchmaker match.
type MatchmakerUser struct {
	NumericProperties map[string]float64 `json:"numeric_properties,omitempty"`
	Presence          *UserPresence      `json:"presence,omitempty"`
	StringProperties  map[string]string  `json:"string_properties,omitempty"`
}

// MatchmakerMatched is delivered when the matchmaker has found a match.
type MatchmakerMatched struct {
	MatchId string            `json:"match_id,omitempty"`
	Ticket  string            `json:"ticket,omitempty"`
	Token   string            `json:"token,omitempty"`
	Users   []*MatchmakerUser `json:"users,omitempty"`
	Self    *MatchmakerUser   `json:"self,omitempty"`
}

// Status describes the online statuses of users you follow.
type Status struct {
	Presences []*UserPresence `json:"presences,omitempty"`
}

// StatusFollowMessage subscribes to status changes for users.
type StatusFollowMessage struct {
	UserIds   []string `json:"user_ids,omitempty"`
	Usernames []string `json:"usernames,omitempty"`
}

// StatusUnfollowMessage unsubscribes from status changes for users.
type StatusUnfollowMessage struct {
	UserIds []string `json:"user_ids,omitempty"`
}

// StatusUpdateMessage updates the current user's status string.
type StatusUpdateMessage struct {
	Status *string `json:"status,omitempty"`
}

// StatusPresenceEvent describes joins and leaves of statuses from users you follow.
type StatusPresenceEvent struct {
	Joins  []*UserPresence `json:"joins,omitempty"`
	Leaves []*UserPresence `json:"leaves,omitempty"`
}

// StreamPresenceEvent describes joins and leaves of low-level streams.
type StreamPresenceEvent struct {
	Stream *Stream         `json:"stream,omitempty"`
	Joins  []*UserPresence `json:"joins,omitempty"`
	Leaves []*UserPresence `json:"leaves,omitempty"`
}

// StreamState is data received from a low-level stream.
type StreamState struct {
	Stream *Stream       `json:"stream,omitempty"`
	Sender *UserPresence `json:"sender,omitempty"`
	Data   string        `json:"data,omitempty"`
}

// WebSocketErrorMessage is delivered when an error occurs server-side.
type WebSocketErrorMessage struct {
	Code    int               `json:"code,omitempty"`
	Message string            `json:"message,omitempty"`
	Context map[string]string `json:"context,omitempty"`
}

// Party is the realtime view of a party. Mirrors Nakama/Party.cs.
type Party struct {
	PartyId   string         `json:"party_id,omitempty"`
	Open      bool           `json:"open,omitempty"`
	MaxSize   int            `json:"max_size,omitempty"`
	Self      *UserPresence  `json:"self,omitempty"`
	Leader    *UserPresence  `json:"leader,omitempty"`
	Presences []*UserPresence `json:"presences,omitempty"`
}

// PartyAccept accepts a join request.
type PartyAccept struct {
	PartyId  string        `json:"party_id,omitempty"`
	Presence *UserPresence `json:"presence,omitempty"`
}

// PartyClose closes a party.
type PartyClose struct {
	PartyId string `json:"party_id,omitempty"`
}

// PartyCreate creates a new party.
type PartyCreate struct {
	Open    bool   `json:"open,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
	MaxSize int    `json:"max_size,omitempty"`
	Label   string `json:"label,omitempty"`
}

// PartyData is realtime data sent to a party.
type PartyData struct {
	PartyId  string        `json:"party_id,omitempty"`
	Presence *UserPresence `json:"presence,omitempty"`
	OpCode   int64         `json:"op_code,omitempty"`
	Data     string        `json:"data,omitempty"`
}

// PartyDataSend sends realtime data to a party.
type PartyDataSend struct {
	PartyId string `json:"party_id,omitempty"`
	OpCode  int64  `json:"op_code,omitempty"`
	Data    string `json:"data,omitempty"`
}

// PartyJoin joins a party.
type PartyJoin struct {
	PartyId string `json:"party_id,omitempty"`
}

// PartyJoinRequest is a request to join a party.
type PartyJoinRequest struct {
	PartyId   string         `json:"party_id,omitempty"`
	Presences []*UserPresence `json:"presences,omitempty"`
}

// PartyJoinRequestList lists pending join requests.
type PartyJoinRequestList struct {
	PartyId string `json:"party_id,omitempty"`
}

// PartyLeader announces a new party leader.
type PartyLeader struct {
	PartyId  string        `json:"party_id,omitempty"`
	Presence *UserPresence `json:"presence,omitempty"`
}

// PartyLeave leaves a party.
type PartyLeave struct {
	PartyId string `json:"party_id,omitempty"`
}

// PartyMatchmakerAdd starts party matchmaking.
type PartyMatchmakerAdd struct {
	PartyId           string             `json:"party_id,omitempty"`
	MinCount          int                `json:"min_count,omitempty"`
	MaxCount          int                `json:"max_count,omitempty"`
	Query             string             `json:"query,omitempty"`
	StringProperties  map[string]string  `json:"string_properties,omitempty"`
	NumericProperties map[string]float64 `json:"numeric_properties,omitempty"`
	CountMultiple     *int               `json:"count_multiple,omitempty"`
}

// PartyMatchmakerRemove cancels party matchmaking.
type PartyMatchmakerRemove struct {
	PartyId string `json:"party_id,omitempty"`
	Ticket  string `json:"ticket,omitempty"`
}

// PartyMatchmakerTicket is the ticket returned when party matchmaking has begun.
type PartyMatchmakerTicket struct {
	PartyId string `json:"party_id,omitempty"`
	Ticket  string `json:"ticket,omitempty"`
}

// PartyMemberRemove removes a member from the party.
type PartyMemberRemove struct {
	PartyId  string        `json:"party_id,omitempty"`
	Presence *UserPresence `json:"presence,omitempty"`
}

// PartyPresenceEvent describes joins and leaves of a party.
type PartyPresenceEvent struct {
	PartyId string         `json:"party_id,omitempty"`
	Joins   []*UserPresence `json:"joins,omitempty"`
	Leaves  []*UserPresence `json:"leaves,omitempty"`
}

// PartyPromote promotes a party member to leader.
type PartyPromote struct {
	PartyId  string        `json:"party_id,omitempty"`
	Presence *UserPresence `json:"presence,omitempty"`
}

// PartyUpdate updates a party label and openness.
type PartyUpdate struct {
	PartyId string `json:"party_id,omitempty"`
	Open    *bool  `json:"open,omitempty"`
	Hidden  *bool  `json:"hidden,omitempty"`
	Label   string `json:"label,omitempty"`
}

// Rpc on the realtime socket.
type Rpc struct {
	Id      string `json:"id,omitempty"`
	Payload string `json:"payload,omitempty"`
	HttpKey string `json:"http_key,omitempty"`
}

// webSocketMessageEnvelope is a port of Nakama/WebSocketMessageEnvelope.cs.
// It serialises a single field at a time on the wire.
type webSocketMessageEnvelope struct {
	Cid                  string                  `json:"cid,omitempty"`
	Channel              *Channel                `json:"channel,omitempty"`
	ChannelJoin          *ChannelJoinMessage     `json:"channel_join,omitempty"`
	ChannelLeave         *ChannelLeaveMessage    `json:"channel_leave,omitempty"`
	ChannelMessage       *ApiChannelMessage      `json:"channel_message,omitempty"`
	ChannelMessageAck    *ChannelMessageAck      `json:"channel_message_ack,omitempty"`
	ChannelMessageRemove *ChannelRemoveMessage   `json:"channel_message_remove,omitempty"`
	ChannelMessageSend   *ChannelSendMessage     `json:"channel_message_send,omitempty"`
	ChannelMessageUpdate *ChannelUpdateMessage   `json:"channel_message_update,omitempty"`
	ChannelPresenceEvent *ChannelPresenceEvent   `json:"channel_presence_event,omitempty"`
	Error                *WebSocketErrorMessage  `json:"error,omitempty"`
	Match                *Match                  `json:"match,omitempty"`
	MatchCreate          *MatchCreateMessage     `json:"match_create,omitempty"`
	MatchJoin            *MatchJoinMessage       `json:"match_join,omitempty"`
	MatchLeave           *MatchLeaveMessage      `json:"match_leave,omitempty"`
	MatchPresenceEvent   *MatchPresenceEvent     `json:"match_presence_event,omitempty"`
	MatchState           *MatchState             `json:"match_data,omitempty"`
	MatchStateSend       *MatchSendMessage       `json:"match_data_send,omitempty"`
	MatchmakerAdd        *MatchmakerAddMessage   `json:"matchmaker_add,omitempty"`
	MatchmakerMatched    *MatchmakerMatched      `json:"matchmaker_matched,omitempty"`
	MatchmakerRemove     *MatchmakerRemoveMessage `json:"matchmaker_remove,omitempty"`
	MatchmakerTicket     *MatchmakerTicket       `json:"matchmaker_ticket,omitempty"`
	NotificationList     *ApiNotificationList    `json:"notifications,omitempty"`
	Rpc                  *ApiRpc                 `json:"rpc,omitempty"`
	Status               *Status                 `json:"status,omitempty"`
	StatusFollow         *StatusFollowMessage    `json:"status_follow,omitempty"`
	StatusPresenceEvent  *StatusPresenceEvent    `json:"status_presence_event,omitempty"`
	StatusUnfollow       *StatusUnfollowMessage  `json:"status_unfollow,omitempty"`
	StatusUpdate         *StatusUpdateMessage    `json:"status_update,omitempty"`
	StreamPresenceEvent  *StreamPresenceEvent    `json:"stream_presence_event,omitempty"`
	StreamState          *StreamState            `json:"stream_data,omitempty"`
	Party                *Party                  `json:"party,omitempty"`
	PartyCreate          *PartyCreate            `json:"party_create,omitempty"`
	PartyUpdate          *PartyUpdate            `json:"party_update,omitempty"`
	PartyJoin            *PartyJoin              `json:"party_join,omitempty"`
	PartyLeave           *PartyLeave             `json:"party_leave,omitempty"`
	PartyPromote         *PartyPromote           `json:"party_promote,omitempty"`
	PartyLeader          *PartyLeader            `json:"party_leader,omitempty"`
	PartyAccept          *PartyAccept            `json:"party_accept,omitempty"`
	PartyMemberRemove    *PartyMemberRemove      `json:"party_remove,omitempty"`
	PartyClose           *PartyClose             `json:"party_close,omitempty"`
	PartyJoinRequestList *PartyJoinRequestList   `json:"party_join_request_list,omitempty"`
	PartyJoinRequest     *PartyJoinRequest       `json:"party_join_request,omitempty"`
	PartyMatchmakerAdd   *PartyMatchmakerAdd     `json:"party_matchmaker_add,omitempty"`
	PartyMatchmakerRemove *PartyMatchmakerRemove `json:"party_matchmaker_remove,omitempty"`
	PartyMatchmakerTicket *PartyMatchmakerTicket `json:"party_matchmaker_ticket,omitempty"`
	PartyData            *PartyData              `json:"party_data,omitempty"`
	PartyDataSend        *PartyDataSend          `json:"party_data_send,omitempty"`
	PartyPresenceEvent   *PartyPresenceEvent     `json:"party_presence_event,omitempty"`
}
