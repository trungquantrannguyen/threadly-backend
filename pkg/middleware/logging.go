package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Logging(log zerolog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		latency := time.Since(start)
		requetsID := GetRequestID(ctx)

		event := log.Info()

		if len(ctx.Errors) > 0 {
			event = log.Error().Str("error", ctx.Errors.String())
		}

		event.
			Str("request_id", requetsID).
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Int("status", ctx.Writer.Status()).
			Dur("latency", latency).
			Str("client_ip", ctx.ClientIP()).
			Msg("request completed")
	}
}
