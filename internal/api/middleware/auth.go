package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/service"
	"github.com/plutolove233/co-dream/internal/globals"
	tokenutil "github.com/plutolove233/co-dream/internal/utils/token"
)

type refreshSessionService interface {
	RefreshSession(ctx context.Context, refreshToken string) (service.IssuedSession, error)
	BuildRefreshCookie(token string) *http.Cookie
}

func TokenRequired() gin.HandlerFunc {
	return TokenRequiredWithService(service.NewDefaultAuthService())
}

func TokenRequiredWithService(auth refreshSessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := tokenutil.ParseBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			globals.JsonAccessDenied(c, "Token is required")
			c.Abort()
			return
		}

		claims, err := tokenutil.ParseAccessToken(tokenString)
		if err == nil {
			c.Set("id", claims.UserID)
			c.Next()
			return
		}

		if !tokenutil.IsTokenExpired(err) {
			globals.JsonAccessDenied(c, "Invalid token")
			c.Abort()
			return
		}

		refreshCookie, cookieErr := c.Cookie(service.RefreshTokenCookieName)
		if cookieErr != nil {
			globals.JsonAccessDenied(c, "登录状态已失效")
			c.Abort()
			return
		}

		issued, refreshErr := auth.RefreshSession(c.Request.Context(), refreshCookie)
		if refreshErr != nil {
			globals.JsonAccessDenied(c, "登录状态已失效")
			c.Abort()
			return
		}

		http.SetCookie(c.Writer, auth.BuildRefreshCookie(issued.RefreshToken))
		c.Header(service.RefreshedAccessTokenHeader, issued.AccessToken)
		c.Header(service.TokenRefreshedHeader, "true")
		c.Set("id", issued.AccessClaims.UserID)
		c.Next()
	}
}
