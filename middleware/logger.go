package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
)

func SetUpLogger(server *gin.Engine) {
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if shouldSkipAccessLogInLowMemory(param) {
			return ""
		}
		var requestID string
		if param.Keys != nil {
			requestID = param.Keys[helper.RequestIdKey].(string)
		}
		return fmt.Sprintf("[GIN] %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))
}

func shouldSkipAccessLogInLowMemory(param gin.LogFormatterParams) bool {
	if !config.ReduceWebLogInLowMemory {
		return false
	}
	if param.StatusCode >= 400 || param.Method != "GET" {
		return false
	}
	if strings.HasPrefix(param.Path, "/api") || strings.HasPrefix(param.Path, "/v1") {
		return false
	}
	return true
}
