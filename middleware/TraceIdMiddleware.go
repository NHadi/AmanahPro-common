package middleware

import (
	"github.com/gin-gonic/gin"
)

const TraceIDHeader = "X-Trace-Id"

func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			// If Trace-ID is missing, you can log it or handle it appropriately
			traceID = "unknown" // Use "unknown" or a generated default
		}
		// Set the Trace-ID in the context
		c.Set(TraceIDHeader, traceID)

		// Continue to the next middleware
		c.Next()
	}
}
