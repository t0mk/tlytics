package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"tlytics"
)

func main() {
	// Initialize tlytics client to connect to remote server
	config := tlytics.Config{
		ServerURL:   "http://192.168.1.100:8081", // Remote analytics server
		FlushPeriod: 3 * time.Second,
	}
	
	analytics, err := tlytics.NewClient(config)
	if err != nil {
		log.Fatal("Failed to initialize tlytics client:", err)
	}
	defer analytics.Close()
	
	// Create Gin router
	r := gin.Default()
	
	// Add analytics middleware
	r.Use(tlytics.GinMiddleware(analytics))
	
	// Example routes
	r.GET("/", func(c *gin.Context) {
		// Emit custom event
		analytics.Emit(tlytics.Event{
			Key: "page_view",
			Data: map[string]interface{}{
				"page": "home",
				"user_id": "123",
			},
		})
		
		c.JSON(200, gin.H{"message": "Hello World"})
	})
	
	r.GET("/api/users", tlytics.TrackEvent(analytics, "api_access", map[string]interface{}{
		"endpoint": "users",
		"action": "list",
	}), func(c *gin.Context) {
		c.JSON(200, gin.H{"users": []string{"alice", "bob"}})
	})
	
	r.GET("/api/slow", tlytics.TrackEvent(analytics, "api_access", map[string]interface{}{
		"endpoint": "slow",
		"action": "test",
	}), func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(200, gin.H{"message": "slow response"})
	})
	
	log.Printf("Analytics client configured to send to: %s", config.ServerURL)
	log.Println("Starting web server on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start web server:", err)
	}
}