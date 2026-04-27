package main

import (
	"context"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plutolove233/co-dream/internal/api/router"
	"github.com/plutolove233/co-dream/internal/database"
)

func main() {
	godotenv.Load()
	ctx := context.Background()
	database.InitPostgreSqlDatabase(ctx, database.NewPostgreSqlConfig())

	gin.SetMode(os.Getenv("CODREAM_MODE"))
	engine := gin.Default()
	// engine.Static("/static", "static")

	// 初始化Session
	store := cookie.NewStore([]byte(os.Getenv("CODREAM_SECRET")))
	store.Options(sessions.Options{
		MaxAge: 3600, // 设置Session过期时间为1小时
	})
	engine.Use(sessions.Sessions("mySession", store))

	// 初始化路由
	router.InitRouter(engine)

	err := engine.Run(os.Getenv("CODREAM_HOST") + ":" + os.Getenv("CODREAM_PORT"))
	if err != nil {
		panic(err)
	}
}
