package main

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plutolove233/co-dream/internal/api/router"
	"github.com/plutolove233/co-dream/internal/database"
	"github.com/plutolove233/co-dream/internal/setting"
	"github.com/spf13/viper"
)

func main() {
	setting.InitViper()
	godotenv.Load()

	ctx := context.Background()
	database.InitPostgreSqlDatabase(ctx, database.NewPostgreSqlConfig())

	// 初始化Redis连接
	if viper.GetBool("system.UseRedis") {
		err := database.InitRedisDatabase(ctx, database.NewRedisConfig())
		if err != nil {
			panic(err)
		}
	}

	gin.SetMode(viper.GetString("system.Mode"))
	engine := gin.Default()
	// engine.Static("/static", "static")

	// 初始化Session
	store := cookie.NewStore([]byte(viper.GetString("system.Secret")))
	store.Options(sessions.Options{
		MaxAge: 3600, // 设置Session过期时间为1小时
	})
	engine.Use(sessions.Sessions("mySession", store))

	// 初始化路由
	router.InitRouter(engine)

	err := engine.Run(viper.GetString("system.SysIP") + ":" + viper.GetString("system.SysPort"))
	if err != nil {
		panic(err)
	}
}
