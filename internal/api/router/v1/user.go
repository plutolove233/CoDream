package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler/user"
)

func RegisterUserRoutes(engine *gin.Engine) {
	u := &user.UserAPI{}
	userGroup := engine.Group("/user")
	{
		userGroup.POST("/register", u.RegisterUser)
		userGroup.GET("/info", u.GetUserByID)
		userGroup.PUT("/update", u.UpdateUser)
	}
}
