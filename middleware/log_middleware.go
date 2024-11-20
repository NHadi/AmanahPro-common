package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func GinLoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read and log the request body
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}

		logger.Info("Request",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("request_body", string(body)),
		)

		// Continue to the next middleware/handler
		c.Next()

		// Log the response
		latency := time.Since(start)
		logger.Info("Response",
			zap.String("trace_id", traceID),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
		)
	}
}
