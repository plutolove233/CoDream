package user

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/service"
	"github.com/plutolove233/co-dream/internal/globals"
	"github.com/plutolove233/co-dream/internal/utils/captcha"
	"github.com/plutolove233/co-dream/internal/utils/email"
	"github.com/plutolove233/co-dream/internal/utils/rsa"
	"github.com/spf13/viper"
)

type UserAPI struct {
}

type RegisterParser struct {
	UserName string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
}

func (u *UserAPI) RegisterUser(c *gin.Context) {
	var parser RegisterParser
	if err := c.ShouldBindJSON(&parser); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}

	ctx := c.Request.Context()

	// 1. 验证邮箱是否已存在
	var userService service.UserService
	userService.Email = parser.Email
	err := userService.Get(ctx)
	if err == nil && userService.ID != nil {
		globals.JsonParameterIllegal(c, "邮箱已存在", globals.ErrEmailAlreadyExists)
		return
	}

	// 2. 验证验证码是否正确
	valid, err := captcha.VerifyCaptcha(ctx, parser.Email, parser.Code)
	if err != nil {
		globals.JsonInternalError(c, "验证码验证失败", err)
		return
	}
	if !valid {
		globals.JsonParameterIllegal(c, "验证码错误或已过期", globals.ErrInvalidCaptcha)
		return
	}

	// 3. 删除已使用的验证码
	err = captcha.DeleteCaptcha(ctx, parser.Email)
	if err != nil {
		globals.JsonInternalError(c, "删除验证码失败", err)
		return
	}

	// 4. RSA加密密码
	rsaUtil := rsa.RSA{
		PublicKeyPath:  viper.GetString("system.RSAPublic"),
		PrivateKeyPath: viper.GetString("system.RSAPrivate"),
	}
	encryptedPassword, err := rsaUtil.Encrypt([]byte(parser.Password))
	if err != nil {
		globals.JsonInternalError(c, "密码加密失败", err)
		return
	}

	// 5. 创建用户记录
	var newUserService service.UserService
	newUserService.Username = parser.UserName
	newUserService.Email = parser.Email
	newUserService.Password = encryptedPassword
	err = newUserService.Add(ctx)
	if err != nil {
		globals.JsonInternalError(c, "创建用户失败", err)
		return
	}
	// 6. 返回成功响应
	globals.JsonOK(c, "注册成功", nil)
}

func (u *UserAPI) GetUserByID(c *gin.Context) {
}

func (u *UserAPI) UpdateUser(c *gin.Context) {

}

type SendCaptchaParser struct {
	Email string `json:"email" binding:"required,email"`
}

// SendCaptcha 发送邮箱验证码
func (u *UserAPI) SendCaptcha(c *gin.Context) {
	var parser SendCaptchaParser
	if err := c.ShouldBindJSON(&parser); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}

	ctx := c.Request.Context()

	// 1. 生成6位验证码
	code, err := captcha.GenerateEmailCode()
	if err != nil {
		globals.JsonInternalError(c, "生成验证码失败", err)
		return
	}

	// 2. 存储验证码到Redis（5分钟过期）
	err = captcha.StoreCaptcha(ctx, parser.Email, code)
	if err != nil {
		globals.JsonInternalError(c, "存储验证码失败", err)
		return
	}

	// 3. 发送邮件
	smtpClient := email.GetSMTPClient()
	subject := "CoDream 注册验证码"
	body := "您的验证码是: " + code + "\n\n验证码有效期为5分钟，请尽快使用。"
	err = smtpClient.SMTPSendEmail("CoDream", parser.Email, subject, "plain", body)
	if err != nil {
		globals.JsonInternalError(c, "发送邮件失败", err)
		return
	}

	globals.JsonOK(c, "验证码已发送", nil)
}
