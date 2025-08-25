package tlytics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Server struct {
	logger *Logger
	port   int
}

func newHTTPServer(logger *Logger, port int) *Server {
	return &Server{
		logger: logger,
		port:   port,
	}
}

func (s *Server) Start() error {
	r := gin.Default()
	
	r.POST("/events", s.handleEvents)
	r.POST("/batch", s.handleBatch)
	r.GET("/health", s.handleHealth)
	r.GET("/view", s.handleView)
	
	return r.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) handleEvents(c *gin.Context) {
	var events []Event
	
	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	for _, event := range events {
		if event.Key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Event key is required"})
			return
		}
		
		if err := s.logger.Emit(event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to emit event"})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully queued %d events", len(events)),
		"count":   len(events),
	})
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"port":   s.port,
	})
}

type BatchRequest struct {
	Events []json.RawMessage `json:"events"`
}

func (s *Server) handleBatch(c *gin.Context) {
	var batch BatchRequest
	
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	events := make([]Event, 0, len(batch.Events))
	
	for _, rawEvent := range batch.Events {
		var event Event
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event format"})
			return
		}
		
		if event.Key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Event key is required"})
			return
		}
		
		events = append(events, event)
	}
	
	for _, event := range events {
		if err := s.logger.Emit(event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to emit event"})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully queued %d events", len(events)),
		"count":   len(events),
	})
}

type ViewResponse struct {
	Events     []Event `json:"events"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

func (s *Server) handleView(c *gin.Context) {
	// Parse query parameters
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 1000 {
		pageSize = 10
	}
	
	// Calculate offset
	offset := (page - 1) * pageSize
	
	// Get events from database
	events, total, err := s.logger.db.GetEvents(pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}
	
	// Calculate total pages
	totalPages := (total + pageSize - 1) / pageSize
	
	response := ViewResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	
	c.JSON(http.StatusOK, response)
}