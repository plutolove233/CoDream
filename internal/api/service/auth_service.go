package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/plutolove233/co-dream/internal/database"
	tokenutil "github.com/plutolove233/co-dream/internal/utils/token"
)

const (
	RefreshTokenCookieName     = "refresh_token"
	RefreshedAccessTokenHeader = "X-New-Access-Token"
	TokenRefreshedHeader       = "X-Token-Refreshed"
)

var (
	ErrSessionNotFound = errors.New("refresh session not found")
)

type RefreshSession struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"sid"`
	TokenID   string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

type IssuedSession struct {
	SessionID       string
	AccessToken     string
	RefreshToken    string
	AccessExpiresIn int
	AccessClaims    *tokenutil.JWTClaims
	RefreshClaims   *tokenutil.JWTClaims
}

type AuthCookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

type RefreshSessionStore interface {
	Save(ctx context.Context, session RefreshSession) error
	Get(ctx context.Context, sessionID string) (RefreshSession, error)
	Delete(ctx context.Context, sessionID string) error
}

type AuthService struct {
	store  RefreshSessionStore
	cookie AuthCookieConfig
}

func NewAuthService(store RefreshSessionStore, cookie AuthCookieConfig) *AuthService {
	if cookie.Name == "" {
		cookie.Name = RefreshTokenCookieName
	}
	if cookie.Path == "" {
		cookie.Path = "/"
	}
	if !cookie.HTTPOnly {
		cookie.HTTPOnly = true
	}
	if !cookie.Secure {
		cookie.Secure = true
	}
	if cookie.SameSite == 0 {
		cookie.SameSite = http.SameSiteLaxMode
	}
	return &AuthService{store: store, cookie: cookie}
}

func NewDefaultAuthService() *AuthService {
	return NewAuthService(newRedisRefreshSessionStore(), AuthCookieConfig{})
}

func (s *AuthService) IssueSession(ctx context.Context, userID string) (IssuedSession, error) {
	sessionID := uuid.NewString()
	refreshTokenID := uuid.NewString()

	accessToken, err := tokenutil.GenerateAccessToken(tokenutil.TokenClaimsInput{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   uuid.NewString(),
	})
	if err != nil {
		return IssuedSession{}, err
	}
	refreshToken, err := tokenutil.GenerateRefreshToken(tokenutil.TokenClaimsInput{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   refreshTokenID,
	})
	if err != nil {
		return IssuedSession{}, err
	}
	accessClaims, err := tokenutil.ParseAccessToken(accessToken)
	if err != nil {
		return IssuedSession{}, err
	}
	refreshClaims, err := tokenutil.ParseRefreshToken(refreshToken)
	if err != nil {
		return IssuedSession{}, err
	}

	session := RefreshSession{
		UserID:    userID,
		SessionID: sessionID,
		TokenID:   refreshTokenID,
		ExpiresAt: refreshClaims.ExpiresAt.Time,
	}
	if err := s.store.Save(ctx, session); err != nil {
		return IssuedSession{}, err
	}

	return IssuedSession{
		SessionID:       sessionID,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		AccessExpiresIn: int(tokenutil.AccessTokenTTL().Seconds()),
		AccessClaims:    accessClaims,
		RefreshClaims:   refreshClaims,
	}, nil
}

func (s *AuthService) RefreshSession(ctx context.Context, refreshToken string) (IssuedSession, error) {
	refreshClaims, err := tokenutil.ParseRefreshToken(refreshToken)
	if err != nil {
		return IssuedSession{}, err
	}

	session, err := s.store.Get(ctx, refreshClaims.SessionID)
	if err != nil {
		return IssuedSession{}, err
	}
	if session.UserID != refreshClaims.UserID || session.TokenID != refreshClaims.ID {
		return IssuedSession{}, ErrSessionNotFound
	}

	accessToken, err := tokenutil.GenerateAccessToken(tokenutil.TokenClaimsInput{
		UserID:    refreshClaims.UserID,
		SessionID: refreshClaims.SessionID,
		TokenID:   uuid.NewString(),
	})
	if err != nil {
		return IssuedSession{}, err
	}

	newRefreshTokenID := uuid.NewString()
	newRefreshToken, err := tokenutil.GenerateRefreshToken(tokenutil.TokenClaimsInput{
		UserID:    refreshClaims.UserID,
		SessionID: refreshClaims.SessionID,
		TokenID:   newRefreshTokenID,
	})
	if err != nil {
		return IssuedSession{}, err
	}
	accessClaims, err := tokenutil.ParseAccessToken(accessToken)
	if err != nil {
		return IssuedSession{}, err
	}
	newRefreshClaims, err := tokenutil.ParseRefreshToken(newRefreshToken)
	if err != nil {
		return IssuedSession{}, err
	}

	session.TokenID = newRefreshTokenID
	session.ExpiresAt = newRefreshClaims.ExpiresAt.Time
	if err := s.store.Save(ctx, session); err != nil {
		return IssuedSession{}, err
	}

	return IssuedSession{
		SessionID:       session.SessionID,
		AccessToken:     accessToken,
		RefreshToken:    newRefreshToken,
		AccessExpiresIn: int(tokenutil.AccessTokenTTL().Seconds()),
		AccessClaims:    accessClaims,
		RefreshClaims:   newRefreshClaims,
	}, nil
}

func (s *AuthService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.store.Delete(ctx, sessionID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	refreshClaims, err := tokenutil.ParseRefreshToken(refreshToken)
	if err != nil {
		return err
	}
	return s.DeleteSession(ctx, refreshClaims.SessionID)
}

func (s *AuthService) BuildRefreshCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     s.cookie.Name,
		Value:    token,
		Path:     s.cookie.Path,
		Domain:   s.cookie.Domain,
		HttpOnly: s.cookie.HTTPOnly,
		Secure:   s.cookie.Secure,
		SameSite: s.cookie.SameSite,
		MaxAge:   int(tokenutil.RefreshTokenTTL().Seconds()),
	}
}

func (s *AuthService) ClearRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.cookie.Name,
		Value:    "",
		Path:     s.cookie.Path,
		Domain:   s.cookie.Domain,
		HttpOnly: s.cookie.HTTPOnly,
		Secure:   s.cookie.Secure,
		SameSite: s.cookie.SameSite,
		MaxAge:   -1,
	}
}

type redisRefreshSessionStore struct{}

func newRedisRefreshSessionStore() RefreshSessionStore {
	return redisRefreshSessionStore{}
}

func (redisRefreshSessionStore) Save(ctx context.Context, session RefreshSession) error {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return errors.New("redis is not initialized")
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}

	return redisDB.Client().Set(ctx, refreshSessionKey(session.SessionID), payload, ttl).Err()
}

func (redisRefreshSessionStore) Get(ctx context.Context, sessionID string) (RefreshSession, error) {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return RefreshSession{}, errors.New("redis is not initialized")
	}

	payload, err := redisDB.Client().Get(ctx, refreshSessionKey(sessionID)).Bytes()
	if err != nil {
		return RefreshSession{}, ErrSessionNotFound
	}

	var session RefreshSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return RefreshSession{}, err
	}
	return session, nil
}

func (redisRefreshSessionStore) Delete(ctx context.Context, sessionID string) error {
	redisDB := database.GetRedisDatabase()
	if redisDB == nil {
		return errors.New("redis is not initialized")
	}
	return redisDB.Client().Del(ctx, refreshSessionKey(sessionID)).Err()
}

func refreshSessionKey(sessionID string) string {
	return fmt.Sprintf("auth:refresh:%s", sessionID)
}
