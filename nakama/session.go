package nakama

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Session represents an authenticated user session.
// It is a port of Nakama/Session.cs.
type Session struct {
	authToken         string
	refreshToken      string
	created           bool
	createTime        int64
	expireTime        int64
	refreshExpireTime int64
	username          string
	userId            string
	vars              map[string]string
}

// AuthToken returns the authorization JWT.
func (s *Session) AuthToken() string { return s.authToken }

// RefreshToken returns the refresh JWT used to obtain a new authorization
// token without re-authenticating.
func (s *Session) RefreshToken() string { return s.refreshToken }

// Created reports whether the session represents a newly-created user.
func (s *Session) Created() bool { return s.created }

// CreateTime returns the session creation time in unix seconds.
func (s *Session) CreateTime() int64 { return s.createTime }

// ExpireTime returns the auth token expiry time in unix seconds.
func (s *Session) ExpireTime() int64 { return s.expireTime }

// RefreshExpireTime returns the refresh token expiry time in unix seconds.
func (s *Session) RefreshExpireTime() int64 { return s.refreshExpireTime }

// Username returns the username associated with this session.
func (s *Session) Username() string { return s.username }

// UserId returns the user id associated with this session.
func (s *Session) UserId() string { return s.userId }

// Vars returns the extra variables bundled inside the session token.
func (s *Session) Vars() map[string]string { return s.vars }

// IsExpired reports whether the auth token has already expired.
func (s *Session) IsExpired() bool { return s.HasExpired(time.Now().UTC()) }

// IsRefreshExpired reports whether the refresh token has already expired.
func (s *Session) IsRefreshExpired() bool { return s.HasRefreshExpired(time.Now().UTC()) }

// HasExpired reports whether the auth token has expired by the given time.
func (s *Session) HasExpired(at time.Time) bool {
	return at.After(time.Unix(s.expireTime, 0).UTC())
}

// HasRefreshExpired reports whether the refresh token has expired by the given time.
func (s *Session) HasRefreshExpired(at time.Time) bool {
	return at.After(time.Unix(s.refreshExpireTime, 0).UTC())
}

// Update replaces the session tokens with fresh values, parsing new claims.
func (s *Session) Update(authToken, refreshToken string) error {
	s.authToken = authToken
	s.refreshToken = refreshToken
	if s.vars == nil {
		s.vars = map[string]string{}
	}

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
	if v, ok := claims["usn"]; ok {
		s.username = fmt.Sprint(v)
	}
	if v, ok := claims["uid"]; ok {
		s.userId = fmt.Sprint(v)
	}
	if v, ok := claims["vrs"].(map[string]any); ok {
		for k, val := range v {
			s.vars[k] = fmt.Sprint(val)
		}
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

// String returns a debug representation of the session.
func (s *Session) String() string {
	return fmt.Sprintf("Session(AuthToken='%s', Created=%t, CreateTime=%d, ExpireTime=%d, RefreshToken=%s, RefreshExpireTime=%d, Username='%s', UserId='%s')",
		s.authToken, s.created, s.createTime, s.expireTime, s.refreshToken, s.refreshExpireTime, s.username, s.userId)
}

// NewSession constructs a session from a fresh authentication response.
func NewSession(authToken, refreshToken string, created bool) (*Session, error) {
	s := &Session{
		created:    created,
		createTime: time.Now().UTC().Unix(),
		vars:       map[string]string{},
	}
	if err := s.Update(authToken, refreshToken); err != nil {
		return nil, err
	}
	return s, nil
}

// RestoreSession restores a Session from a previously-issued auth token. The
// returned Session is marked as "not newly created".
func RestoreSession(authToken, refreshToken string) (*Session, error) {
	if authToken == "" {
		return nil, errors.New("nakama: cannot restore session from empty auth token")
	}
	return NewSession(authToken, refreshToken, false)
}

func jwtUnpack(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, errors.New("nakama: invalid JWT")
	}
	payload := parts[1]
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("nakama: cannot decode JWT payload: %w", err)
	}

	claims := map[string]any{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("nakama: cannot parse JWT payload: %w", err)
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
		var i int64
		fmt.Sscan(x, &i)
		return i
	default:
		return 0
	}
}
