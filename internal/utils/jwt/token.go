package jwt

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

type JWTClaims struct {
	jwt.StandardClaims
	UserID string `json:"user_id"`
}

func genToken(claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(viper.GetString("system.Secret")))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func MakeToken(userID string) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
	}
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Second * time.Duration(viper.GetInt("system.TokenExpireTime"))).Unix()
	return genToken(claims)
}

func VerifyToken(strToken string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(strToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("system.Secret")), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, err
	}
	if err := token.Claims.Valid(); err != nil {
		return nil, err
	}
	return claims, nil
}

func RefreshToken(strToken string) (string, error) {
	claims, err := VerifyToken(strToken)
	if err != nil {
		return "", err
	}
	claims.ExpiresAt = time.Now().Unix() + (claims.ExpiresAt - claims.IssuedAt)
	signedToken, err := genToken(claims)
	if err != nil {
		return "", err
	}
	return signedToken, err
}
