package globals

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func JsonOK(c *gin.Context, msg string, data interface{}) {
	if msg == "" {
		msg = "成功!"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    OK,
		"message": msg,
		"data":    data,
	})
}

func JsonParameterIllegal(c *gin.Context, msg string, err error) {
	if msg == "" {
		msg = "参数非法!"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    ParameterIllegal,
		"message": msg,
		"data":    err.Error(),
	})
}

func JsonDataError(c *gin.Context, msg string, err error) {
	if msg == "" {
		msg = "数据错误!"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    DataError,
		"message": msg,
		"data":    err.Error(),
	})
}

func JsonNotData(c *gin.Context, msg string, err error) {
	if msg == "" {
		msg = "无数据!"
	}
	if err == nil {
		err = errors.New(msg)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    DataError,
		"message": msg,
		"data":    err.Error(),
	})
}

func JsonInternalError(c *gin.Context, msg string, err error) {
	if msg == "" {
		msg = "系统错误!"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    InternalError,
		"message": msg,
		"data":    err.Error(),
	})
}

func JsonDBError(c *gin.Context, msg string, err error) {
	if err.Error() == "record not found" {
		if msg == "" {
			msg = "无数据!"
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    NotData,
			"message": msg,
			"data":    err.Error(),
		})
		return
	}
	if msg == "" {
		msg = "数据库错误!"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    DBError,
		"message": msg,
		"data":    err.Error(),
	})
}

func JsonDataExist(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    DataExist,
		"message": msg,
	})
}

func JsonAccessDenied(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    AccessDenied,
		"message": msg,
	})
}

func JsonLoginError(c *gin.Context, msg string, err error) {
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    LoginError,
			"message": msg,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    LoginError,
			"message": msg,
			"data":    err,
		})
	}
}

func JsonUnauthorizedUserId(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    UnauthorizedUserId,
		"message": msg,
	})
}

func JsonIncompleteRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    ParameterIllegal,
		"message": msg,
	})
}
