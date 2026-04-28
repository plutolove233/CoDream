package user

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/globals"
)

type UserAPI struct {
}

type RegisterParser struct {
	UserName string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (u *UserAPI) RegisterUser(c *gin.Context) {
	var parser RegisterParser
	if err := c.ShouldBindJSON(&parser); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}

	// 实现注册逻辑
	// 1. 验证用户名和邮箱是否已存在
	// 2. 通过邮箱发送验证码并验证验证码是否正确
	// 3. RSA加密密码
	// 3. 创建用户记录
	// 4. 返回成功响应
	
}

func (u *UserAPI) GetUserByID(c *gin.Context) {
}

func (u *UserAPI) UpdateUser(c *gin.Context) {

}
