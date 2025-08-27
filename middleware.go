package tlytics

import (
	"time"

	"github.com/gin-gonic/gin"
)

func GinMiddleware(analytics Emitter) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Log the request event
		event := Event{
			Key:       "http_request",
			Timestamp: start,
			Data: map[string]interface{}{
				"method":        c.Request.Method,
				"path":          c.Request.URL.Path,
				"status_code":   c.Writer.Status(),
				"duration_ms":   time.Since(start).Milliseconds(),
				"client_ip":     c.ClientIP(),
				"user_agent":    c.GetHeader("User-Agent"),
				"response_size": c.Writer.Size(),
			},
		}
		
		analytics.Emit(event)
	}
}

func TrackEvent(analytics Emitter, key string, data map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request first
		c.Next()
		
		// Create event with duration
		event := Event{
			Key:       key,
			Timestamp: start,
			Data:      data,
		}
		
		if event.Data == nil {
			event.Data = make(map[string]interface{})
		}
		
		// Add request context and duration to the event data
		event.Data["request_path"] = c.Request.URL.Path
		event.Data["client_ip"] = c.ClientIP()
		event.Data["duration_ms"] = time.Since(start).Milliseconds()
		
		analytics.Emit(event)
	}
}