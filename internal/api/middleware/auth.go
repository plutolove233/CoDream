package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/globals"
	"github.com/plutolove233/co-dream/internal/utils/jwt"
)

func TokenRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the request header
		token := c.Request.Header.Get("token")

		// Check if the token is present
		if token == "" {
			globals.JsonAccessDenied(c, "Token is required")
			c.Abort()
			return
		}

		// Validate the token (this is a placeholder, implement your own validation logic)
		claim, err := jwt.VerifyToken(token)
		if err != nil {
			globals.JsonAccessDenied(c, "Invalid token")
			c.Abort()
			return
		}

		// If the token is valid, proceed to the next handler
		c.Set("id", claim.UserID)
		c.Next()
	}
}
