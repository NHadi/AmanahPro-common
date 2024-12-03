package middleware

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// CustomResponseWriter captures the response body for logging purposes.
type CustomResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write overrides the default Write method to capture the response body.
func (w *CustomResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)                  // Write to buffer for logging
	return w.ResponseWriter.Write(data) // Write to the actual response
}

// RequestLoggingMiddleware logs incoming requests and outgoing responses.
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique request ID (optional, if needed for tracing)
		requestID := time.Now().Format("20060102150405") // Example: YYYYMMDDHHMMSS
		c.Set("RequestID", requestID)

		// Log request details
		log.Printf("[RequestID=%s] Incoming request: Method=%s, Path=%s", requestID, c.Request.Method, c.Request.URL.Path)

		// Log query parameters if present
		if len(c.Request.URL.RawQuery) > 0 {
			log.Printf("[RequestID=%s] Query parameters: %s", requestID, c.Request.URL.RawQuery)
		}

		// Log URL parameters if present
		if len(c.Params) > 0 {
			log.Printf("[RequestID=%s] URL parameters: %v", requestID, c.Params)
		}

		// Log request body if applicable
		if c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				log.Printf("[RequestID=%s] Request body: %s", requestID, string(bodyBytes))
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore the body for further processing
			} else {
				log.Printf("[RequestID=%s] Failed to read request body: %v", requestID, err)
			}
		}

		// Replace the default response writer with our custom one to capture response body
		responseWriter := &CustomResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseWriter

		// Record start time for performance monitoring
		startTime := time.Now()

		// Process the request
		c.Next()

		// Record the response details
		duration := time.Since(startTime)
		statusCode := responseWriter.Status()
		responseBody := responseWriter.body.String()

		// Log response details
		log.Printf("[RequestID=%s] Response status: %d", requestID, statusCode)
		if len(responseBody) > 0 {
			log.Printf("[RequestID=%s] Response body: %s", requestID, responseBody)
		}
		log.Printf("[RequestID=%s] Request processed in: %v", requestID, duration)

		// Log errors if any occurred during request processing
		if len(c.Errors) > 0 {
			log.Printf("[RequestID=%s] Errors: %v", requestID, c.Errors.String())
		}
	}
}
