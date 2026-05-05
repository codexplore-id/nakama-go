// Realtime socket support, ported from Nakama/Socket.cs and ISocket.cs.
package nakama

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

const (
	// DefaultConnectTimeout is the default time allowed for the socket to connect.
	DefaultConnectTimeout = 30 * time.Second
	// DefaultSendTimeout is the maximum time allowed for a send to complete.
	DefaultSendTimeout = 10 * time.Second
)

// SocketHandlers holds the callbacks invoked when realtime events arrive on
// the socket. Mirrors the events on Nakama/ISocket.cs.
//
// All callbacks may be nil. They are invoked from a goroutine owned by the
// socket reader and must not block.
type SocketHandlers struct {
	OnConnected            func()
	OnClosed               func(reason string)
	OnError                func(err error)
	OnChannelMessage       func(*ApiChannelMessage)
	OnChannelPresence      func(*ChannelPresenceEvent)
	OnMatchmakerMatched    func(*MatchmakerMatched)
	OnMatchState           func(*MatchState)
	OnMatchPresence        func(*MatchPresenceEvent)
	OnNotification         func(*ApiNotification)
	OnStatusPresence       func(*StatusPresenceEvent)
	OnStreamPresence       func(*StreamPresenceEvent)
	OnStreamState          func(*StreamState)
	OnParty                func(*Party)
	OnPartyClose           func(*PartyClose)
	OnPartyData            func(*PartyData)
	OnPartyJoinRequest     func(*PartyJoinRequest)
	OnPartyLeader          func(*PartyLeader)
	OnPartyMatchmakerTicket func(*PartyMatchmakerTicket)
	OnPartyPresence        func(*PartyPresenceEvent)
	OnPartyUpdate          func(*PartyUpdate)
}

// Socket is the realtime client. It mirrors Nakama/Socket.cs.
type Socket struct {
	scheme   string
	host     string
	port     int
	logger   Logger
	handlers SocketHandlers

	cid           atomic.Int64
	sendTimeout   time.Duration
	mu            sync.Mutex
	conn          *websocket.Conn
	connected     atomic.Bool
	connecting    atomic.Bool
	pendingMu     sync.Mutex
	pending       map[string]chan *webSocketMessageEnvelope
	closeOnce     sync.Once
	closeReason   string
	readerCtx     context.Context
	readerCancel  context.CancelFunc
}

// NewSocket creates an unconnected Socket using the supplied scheme/host/port.
func NewSocket(scheme, host string, port int, logger Logger, handlers SocketHandlers) *Socket {
	if scheme == "" {
		scheme = "ws"
	}
	if logger == nil {
		logger = NullLogger{}
	}
	return &Socket{
		scheme:      scheme,
		host:        host,
		port:        port,
		logger:      logger,
		handlers:    handlers,
		pending:     map[string]chan *webSocketMessageEnvelope{},
		sendTimeout: DefaultSendTimeout,
	}
}

// SocketFromClient creates a Socket sharing the connection settings of c.
// "http" → "ws", "https" → "wss".
func SocketFromClient(c *Client, handlers SocketHandlers) *Socket {
	scheme := "ws"
	if c.Scheme() == "https" {
		scheme = "wss"
	}
	return NewSocket(scheme, c.Host(), c.Port(), c.Logger(), handlers)
}

// IsConnected reports whether the socket is currently connected.
func (s *Socket) IsConnected() bool { return s.connected.Load() }

// IsConnecting reports whether the socket is in the process of connecting.
func (s *Socket) IsConnecting() bool { return s.connecting.Load() }

// SetSendTimeout overrides the default per-message send timeout.
func (s *Socket) SetSendTimeout(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendTimeout = d
}

// Connect connects to the server using the supplied session.
func (s *Socket) Connect(ctx context.Context, session *Session, appearOnline bool, langTag string) error {
	if s.IsConnected() || s.IsConnecting() {
		return errors.New("nakama: socket already connecting or connected")
	}
	if session == nil {
		return errors.New("nakama: session is required")
	}
	if langTag == "" {
		langTag = "en"
	}
	s.connecting.Store(true)
	defer s.connecting.Store(false)

	u := &url.URL{
		Scheme: s.scheme,
		Host:   fmt.Sprintf("%s:%d", s.host, s.port),
		Path:   "/ws",
	}
	q := u.Query()
	q.Set("lang", langTag)
	q.Set("status", strconv.FormatBool(appearOnline))
	q.Set("token", session.AuthToken())
	u.RawQuery = q.Encode()

	dialCtx := ctx
	if dialCtx == nil {
		dialCtx = context.Background()
	}
	dialCtx, cancel := context.WithTimeout(dialCtx, DefaultConnectTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("nakama: cannot dial socket: %w", err)
	}
	conn.SetReadLimit(-1)

	s.mu.Lock()
	s.conn = conn
	s.readerCtx, s.readerCancel = context.WithCancel(context.Background())
	s.mu.Unlock()
	s.connected.Store(true)
	s.closeReason = ""
	s.closeOnce = sync.Once{}

	if h := s.handlers.OnConnected; h != nil {
		go h()
	}

	go s.readLoop()
	return nil
}

// Close closes the socket connection.
func (s *Socket) Close() error {
	s.mu.Lock()
	conn := s.conn
	cancel := s.readerCancel
	s.mu.Unlock()
	if conn == nil {
		return nil
	}
	err := conn.Close(websocket.StatusNormalClosure, "client closing")
	if cancel != nil {
		cancel()
	}
	s.handleClosed("client closing")
	return err
}

func (s *Socket) handleClosed(reason string) {
	s.closeOnce.Do(func() {
		s.connected.Store(false)
		s.pendingMu.Lock()
		for _, ch := range s.pending {
			close(ch)
		}
		s.pending = map[string]chan *webSocketMessageEnvelope{}
		s.pendingMu.Unlock()

		if h := s.handlers.OnClosed; h != nil {
			go h(reason)
		}
	})
}

func (s *Socket) readLoop() {
	for {
		s.mu.Lock()
		conn := s.conn
		ctx := s.readerCtx
		s.mu.Unlock()
		if conn == nil || ctx == nil {
			return
		}
		typ, data, err := conn.Read(ctx)
		if err != nil {
			if h := s.handlers.OnError; h != nil {
				go h(err)
			}
			s.handleClosed(err.Error())
			return
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		s.processMessage(data)
	}
}

func (s *Socket) processMessage(data []byte) {
	s.logger.DebugFormat("Received over socket: %s", string(data))
	envelope := &webSocketMessageEnvelope{}
	if err := json.Unmarshal(data, envelope); err != nil {
		if h := s.handlers.OnError; h != nil {
			h(fmt.Errorf("nakama: cannot decode envelope: %w", err))
		}
		return
	}

	if envelope.Cid != "" {
		s.pendingMu.Lock()
		ch, ok := s.pending[envelope.Cid]
		if ok {
			delete(s.pending, envelope.Cid)
		}
		s.pendingMu.Unlock()
		if ok {
			ch <- envelope
			close(ch)
			return
		}
		s.logger.WarnFormat("No pending request for cid: %s", envelope.Cid)
		return
	}

	switch {
	case envelope.Error != nil:
		if h := s.handlers.OnError; h != nil {
			h(fmt.Errorf("nakama: socket error %d: %s", envelope.Error.Code, envelope.Error.Message))
		}
	case envelope.ChannelMessage != nil:
		if h := s.handlers.OnChannelMessage; h != nil {
			h(envelope.ChannelMessage)
		}
	case envelope.ChannelPresenceEvent != nil:
		if h := s.handlers.OnChannelPresence; h != nil {
			h(envelope.ChannelPresenceEvent)
		}
	case envelope.MatchmakerMatched != nil:
		if h := s.handlers.OnMatchmakerMatched; h != nil {
			h(envelope.MatchmakerMatched)
		}
	case envelope.MatchPresenceEvent != nil:
		if h := s.handlers.OnMatchPresence; h != nil {
			h(envelope.MatchPresenceEvent)
		}
	case envelope.MatchState != nil:
		if h := s.handlers.OnMatchState; h != nil {
			h(envelope.MatchState)
		}
	case envelope.NotificationList != nil:
		if h := s.handlers.OnNotification; h != nil {
			for _, n := range envelope.NotificationList.Notifications {
				h(n)
			}
		}
	case envelope.StatusPresenceEvent != nil:
		if h := s.handlers.OnStatusPresence; h != nil {
			h(envelope.StatusPresenceEvent)
		}
	case envelope.StreamPresenceEvent != nil:
		if h := s.handlers.OnStreamPresence; h != nil {
			h(envelope.StreamPresenceEvent)
		}
	case envelope.StreamState != nil:
		if h := s.handlers.OnStreamState; h != nil {
			h(envelope.StreamState)
		}
	case envelope.Party != nil:
		if h := s.handlers.OnParty; h != nil {
			h(envelope.Party)
		}
	case envelope.PartyClose != nil:
		if h := s.handlers.OnPartyClose; h != nil {
			h(envelope.PartyClose)
		}
	case envelope.PartyData != nil:
		if h := s.handlers.OnPartyData; h != nil {
			h(envelope.PartyData)
		}
	case envelope.PartyJoinRequest != nil:
		if h := s.handlers.OnPartyJoinRequest; h != nil {
			h(envelope.PartyJoinRequest)
		}
	case envelope.PartyLeader != nil:
		if h := s.handlers.OnPartyLeader; h != nil {
			h(envelope.PartyLeader)
		}
	case envelope.PartyMatchmakerTicket != nil:
		if h := s.handlers.OnPartyMatchmakerTicket; h != nil {
			h(envelope.PartyMatchmakerTicket)
		}
	case envelope.PartyPresenceEvent != nil:
		if h := s.handlers.OnPartyPresence; h != nil {
			h(envelope.PartyPresenceEvent)
		}
	case envelope.PartyUpdate != nil:
		if h := s.handlers.OnPartyUpdate; h != nil {
			h(envelope.PartyUpdate)
		}
	default:
		s.logger.WarnFormat("Unrecognised socket message: %s", string(data))
	}
}

// send sends an envelope without waiting for a response.
func (s *Socket) send(ctx context.Context, envelope *webSocketMessageEnvelope) error {
	if !s.IsConnected() {
		return errors.New("nakama: socket is not connected")
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	s.logger.DebugFormat("Sending over socket: %s", string(data))

	s.mu.Lock()
	conn := s.conn
	timeout := s.sendTimeout
	s.mu.Unlock()
	if conn == nil {
		return errors.New("nakama: socket has no connection")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

// sendRequest sends an envelope with a fresh cid and waits for the matching
// response.
func (s *Socket) sendRequest(ctx context.Context, envelope *webSocketMessageEnvelope) (*webSocketMessageEnvelope, error) {
	cid := strconv.FormatInt(s.cid.Add(1), 10)
	envelope.Cid = cid

	ch := make(chan *webSocketMessageEnvelope, 1)
	s.pendingMu.Lock()
	s.pending[cid] = ch
	s.pendingMu.Unlock()

	if err := s.send(ctx, envelope); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, cid)
		s.pendingMu.Unlock()
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timeout := s.sendTimeout
	if timeout <= 0 {
		timeout = DefaultSendTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case env, ok := <-ch:
		if !ok {
			return nil, errors.New("nakama: socket closed before response")
		}
		if env.Error != nil {
			return env, fmt.Errorf("nakama: socket error %d: %s", env.Error.Code, env.Error.Message)
		}
		return env, nil
	case <-ctx.Done():
		s.pendingMu.Lock()
		delete(s.pending, cid)
		s.pendingMu.Unlock()
		return nil, ctx.Err()
	case <-timer.C:
		s.pendingMu.Lock()
		delete(s.pending, cid)
		s.pendingMu.Unlock()
		return nil, errors.New("nakama: socket request timed out")
	}
}

// ===== Match =====

// CreateMatch creates a new authoritative or relayed match.
func (s *Socket) CreateMatch(ctx context.Context, name string) (*Match, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchCreate: &MatchCreateMessage{Name: name}})
	if err != nil {
		return nil, err
	}
	return resp.Match, nil
}

// JoinMatch joins a match by id.
func (s *Socket) JoinMatch(ctx context.Context, matchId string, metadata map[string]string) (*Match, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchJoin: &MatchJoinMessage{Id: matchId, Metadata: metadata}})
	if err != nil {
		return nil, err
	}
	return resp.Match, nil
}

// JoinMatchByToken joins a match using a matchmaker token.
func (s *Socket) JoinMatchByToken(ctx context.Context, token string) (*Match, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchJoin: &MatchJoinMessage{Token: token}})
	if err != nil {
		return nil, err
	}
	return resp.Match, nil
}

// LeaveMatch leaves a match.
func (s *Socket) LeaveMatch(ctx context.Context, matchId string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchLeave: &MatchLeaveMessage{MatchId: matchId}})
	return err
}

// SendMatchState sends realtime state to a match. The data is base64-encoded
// when marshalled to JSON.
func (s *Socket) SendMatchState(ctx context.Context, matchId string, opCode int64, data []byte, presences []*UserPresence) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	return s.send(ctx, &webSocketMessageEnvelope{MatchStateSend: &MatchSendMessage{
		MatchId:   matchId,
		OpCode:    strconv.FormatInt(opCode, 10),
		Data:      encoded,
		Presences: presences,
	}})
}

// ===== Channels =====

// JoinChat joins a chat channel.
func (s *Socket) JoinChat(ctx context.Context, target string, channelType ChannelType, persistence, hidden bool) (*Channel, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{ChannelJoin: &ChannelJoinMessage{
		Target:      target,
		Type:        int(channelType),
		Persistence: persistence,
		Hidden:      hidden,
	}})
	if err != nil {
		return nil, err
	}
	return resp.Channel, nil
}

// LeaveChat leaves a chat channel.
func (s *Socket) LeaveChat(ctx context.Context, channelId string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{ChannelLeave: &ChannelLeaveMessage{ChannelId: channelId}})
	return err
}

// WriteChatMessage writes a chat message to a channel.
func (s *Socket) WriteChatMessage(ctx context.Context, channelId, content string) (*ChannelMessageAck, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{ChannelMessageSend: &ChannelSendMessage{ChannelId: channelId, Content: content}})
	if err != nil {
		return nil, err
	}
	return resp.ChannelMessageAck, nil
}

// UpdateChatMessage updates a previously sent chat message.
func (s *Socket) UpdateChatMessage(ctx context.Context, channelId, messageId, content string) (*ChannelMessageAck, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{ChannelMessageUpdate: &ChannelUpdateMessage{ChannelId: channelId, MessageId: messageId, Content: content}})
	if err != nil {
		return nil, err
	}
	return resp.ChannelMessageAck, nil
}

// RemoveChatMessage removes a chat message.
func (s *Socket) RemoveChatMessage(ctx context.Context, channelId, messageId string) (*ChannelMessageAck, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{ChannelMessageRemove: &ChannelRemoveMessage{ChannelId: channelId, MessageId: messageId}})
	if err != nil {
		return nil, err
	}
	return resp.ChannelMessageAck, nil
}

// ===== Matchmaker =====

// AddMatchmaker joins the matchmaker pool.
func (s *Socket) AddMatchmaker(ctx context.Context, query string, minCount, maxCount int, stringProps map[string]string, numericProps map[string]float64, countMultiple *int) (*MatchmakerTicket, error) {
	if query == "" {
		query = "*"
	}
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchmakerAdd: &MatchmakerAddMessage{
		Query:             query,
		MinCount:          minCount,
		MaxCount:          maxCount,
		StringProperties:  stringProps,
		NumericProperties: numericProps,
		CountMultiple:     countMultiple,
	}})
	if err != nil {
		return nil, err
	}
	return resp.MatchmakerTicket, nil
}

// RemoveMatchmaker leaves the matchmaker pool.
func (s *Socket) RemoveMatchmaker(ctx context.Context, ticket string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{MatchmakerRemove: &MatchmakerRemoveMessage{Ticket: ticket}})
	return err
}

// ===== Status =====

// FollowUsers subscribes to status changes for the given users.
func (s *Socket) FollowUsers(ctx context.Context, userIds, usernames []string) (*Status, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{StatusFollow: &StatusFollowMessage{UserIds: userIds, Usernames: usernames}})
	if err != nil {
		return nil, err
	}
	return resp.Status, nil
}

// UnfollowUsers stops following user statuses.
func (s *Socket) UnfollowUsers(ctx context.Context, userIds []string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{StatusUnfollow: &StatusUnfollowMessage{UserIds: userIds}})
	return err
}

// UpdateStatus changes the user's online status string.
func (s *Socket) UpdateStatus(ctx context.Context, status string) error {
	statusPtr := status
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{StatusUpdate: &StatusUpdateMessage{Status: &statusPtr}})
	return err
}

// ===== RPC =====

// Rpc invokes a server-side RPC.
func (s *Socket) Rpc(ctx context.Context, id, payload string) (*ApiRpc, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{Rpc: &ApiRpc{Id: id, Payload: payload}})
	if err != nil {
		return nil, err
	}
	return resp.Rpc, nil
}

// ===== Parties =====

// CreateParty creates a new party.
func (s *Socket) CreateParty(ctx context.Context, open, hidden bool, maxSize int, label string) (*Party, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyCreate: &PartyCreate{Open: open, Hidden: hidden, MaxSize: maxSize, Label: label}})
	if err != nil {
		return nil, err
	}
	return resp.Party, nil
}

// JoinParty joins an existing party.
func (s *Socket) JoinParty(ctx context.Context, partyId string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyJoin: &PartyJoin{PartyId: partyId}})
	return err
}

// LeaveParty leaves a party.
func (s *Socket) LeaveParty(ctx context.Context, partyId string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyLeave: &PartyLeave{PartyId: partyId}})
	return err
}

// ClosePartyAsync ends a party.
func (s *Socket) ClosePartyAsync(ctx context.Context, partyId string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyClose: &PartyClose{PartyId: partyId}})
	return err
}

// AcceptPartyMember accepts a join request.
func (s *Socket) AcceptPartyMember(ctx context.Context, partyId string, presence *UserPresence) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyAccept: &PartyAccept{PartyId: partyId, Presence: presence}})
	return err
}

// RemovePartyMember removes a party member or rejects a join request.
func (s *Socket) RemovePartyMember(ctx context.Context, partyId string, presence *UserPresence) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyMemberRemove: &PartyMemberRemove{PartyId: partyId, Presence: presence}})
	return err
}

// PromotePartyMember promotes a party member to leader.
func (s *Socket) PromotePartyMember(ctx context.Context, partyId string, presence *UserPresence) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyPromote: &PartyPromote{PartyId: partyId, Presence: presence}})
	return err
}

// ListPartyJoinRequests lists pending join requests for a party.
func (s *Socket) ListPartyJoinRequests(ctx context.Context, partyId string) (*PartyJoinRequest, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyJoinRequestList: &PartyJoinRequestList{PartyId: partyId}})
	if err != nil {
		return nil, err
	}
	return resp.PartyJoinRequest, nil
}

// AddMatchmakerParty starts party matchmaking.
func (s *Socket) AddMatchmakerParty(ctx context.Context, partyId, query string, minCount, maxCount int, stringProps map[string]string, numericProps map[string]float64, countMultiple *int) (*PartyMatchmakerTicket, error) {
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyMatchmakerAdd: &PartyMatchmakerAdd{
		PartyId:           partyId,
		Query:             query,
		MinCount:          minCount,
		MaxCount:          maxCount,
		StringProperties:  stringProps,
		NumericProperties: numericProps,
		CountMultiple:     countMultiple,
	}})
	if err != nil {
		return nil, err
	}
	return resp.PartyMatchmakerTicket, nil
}

// RemoveMatchmakerParty cancels a party matchmaker ticket.
func (s *Socket) RemoveMatchmakerParty(ctx context.Context, partyId, ticket string) error {
	_, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyMatchmakerRemove: &PartyMatchmakerRemove{PartyId: partyId, Ticket: ticket}})
	return err
}

// SendPartyData sends realtime data to a party.
func (s *Socket) SendPartyData(ctx context.Context, partyId string, opCode int64, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	return s.send(ctx, &webSocketMessageEnvelope{PartyDataSend: &PartyDataSend{PartyId: partyId, OpCode: opCode, Data: encoded}})
}

// UpdateParty updates the label/openness of a party.
func (s *Socket) UpdateParty(ctx context.Context, partyId string, open, hidden bool, label string) (*PartyUpdate, error) {
	openPtr := open
	hiddenPtr := hidden
	resp, err := s.sendRequest(ctx, &webSocketMessageEnvelope{PartyUpdate: &PartyUpdate{PartyId: partyId, Open: &openPtr, Hidden: &hiddenPtr, Label: label}})
	if err != nil {
		return nil, err
	}
	return resp.PartyUpdate, nil
}
