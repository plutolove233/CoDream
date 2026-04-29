package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var (
	ErrInvalidBearerToken = errors.New("invalid bearer token")
)

type TokenClaimsInput struct {
	UserID    string
	SessionID string
	TokenID   string
}

type JWTClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func accessSecret() []byte {
	return []byte(viper.GetString("system.AccessSecret"))
}

func refreshSecret() []byte {
	return []byte(viper.GetString("system.RefreshSecret"))
}

func accessTokenTTL() time.Duration {
	return time.Duration(viper.GetInt("system.AccessTokenExpireTime")) * time.Second
}

func refreshTokenTTL() time.Duration {
	return time.Duration(viper.GetInt("system.RefreshTokenExpireTime")) * time.Second
}

func GenerateAccessToken(input TokenClaimsInput) (string, error) {
	return generateToken(input, accessTokenTTL(), accessSecret())
}

func GenerateRefreshToken(input TokenClaimsInput) (string, error) {
	return generateToken(input, refreshTokenTTL(), refreshSecret())
}

func ParseAccessToken(tokenString string) (*JWTClaims, error) {
	return parseToken(tokenString, accessSecret())
}

func ParseRefreshToken(tokenString string) (*JWTClaims, error) {
	return parseToken(tokenString, refreshSecret())
}

func ParseBearerToken(header string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", ErrInvalidBearerToken
	}
	return strings.TrimSpace(parts[1]), nil
}

func IsTokenExpired(err error) bool {
	return err != nil && errors.Is(err, jwt.ErrTokenExpired)
}

func AccessTokenTTL() time.Duration {
	return accessTokenTTL()
}

func RefreshTokenTTL() time.Duration {
	return refreshTokenTTL()
}

func generateToken(input TokenClaimsInput, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    input.UserID,
		SessionID: input.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        input.TokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func parseToken(tokenString string, secret []byte) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
