package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := ctx.GetHeader("X-Request-ID")

		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx.Set(RequestIDKey, requestID)
		ctx.Header("X-Request-ID", requestID)

		ctx.Next()
	}
}

func GetRequestID(ctx *gin.Context) string {
	value, exists := ctx.Get(RequestIDKey)
	if !exists {
		return ""
	}

	requestID, ok := value.(string)

	if !ok {
		return ""
	}

	return requestID
}
