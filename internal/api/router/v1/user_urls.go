package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler/user"
	"github.com/plutolove233/co-dream/internal/api/middleware"
)

func RegisterUserRouters(engine *gin.RouterGroup) {
	u := user.NewUserAPI()
	userGroup := engine.Group("/user")
	{
		userGroup.POST("/send-captcha", u.SendCaptcha)
		userGroup.POST("/register", u.RegisterUser)
		userGroup.POST("/login", u.Login)
		userGroup.Use(middleware.TokenRequired())
		userGroup.POST("/logout", u.Logout)
		userGroup.GET("/info", u.GetUserByID)
		userGroup.PUT("/update", u.UpdateUser)
	}
}
