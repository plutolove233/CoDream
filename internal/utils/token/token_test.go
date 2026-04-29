package token

import (
	"testing"

	"github.com/spf13/viper"
)

func setTokenTestConfig() {
	viper.Set("system.AccessSecret", "access-secret")
	viper.Set("system.RefreshSecret", "refresh-secret")
	viper.Set("system.AccessTokenExpireTime", 900)
	viper.Set("system.RefreshTokenExpireTime", 3600)
}

func TestGenerateAccessTokenIncludesSessionClaims(t *testing.T) {
	setTokenTestConfig()

	tokenString, err := GenerateAccessToken(TokenClaimsInput{
		UserID:    "user-1",
		SessionID: "sid-1",
		TokenID:   "jti-1",
	})
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ParseAccessToken(tokenString)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("claims.UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.SessionID != "sid-1" {
		t.Fatalf("claims.SessionID = %q, want %q", claims.SessionID, "sid-1")
	}
	if claims.ID != "jti-1" {
		t.Fatalf("claims.ID = %q, want %q", claims.ID, "jti-1")
	}
}

func TestParseBearerTokenRejectsInvalidHeader(t *testing.T) {
	_, err := ParseBearerToken("Token abc")
	if err == nil {
		t.Fatal("expected error for invalid bearer header")
	}
}
