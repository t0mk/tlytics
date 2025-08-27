package tlytics

import (
	"fmt"
	"time"
)

// Emitter interface for both Client and Server instances
type Emitter interface {
	Emit(event Event) error
}

// Tlytics represents a server instance
type Tlytics struct {
	db     *DB
	logger *Logger
	server *Server
}

// Config for client connecting to remote server
type Config struct {
	ServerURL   string        // Remote server URL (e.g., "http://192.168.1.100:8081")
	FlushPeriod time.Duration // How often to flush queued events
}

// ServerConfig for running local analytics server
type ServerConfig struct {
	DBPath      string
	FlushPeriod time.Duration
	ServerPort  int
}

// NewClient creates a client that connects to a remote analytics server
func NewClient(config Config) (*Client, error) {
	if config.ServerURL == "" {
		return nil, fmt.Errorf("ServerURL is required")
	}

	client := newHTTPClient(config.ServerURL, config.FlushPeriod)
	
	return client, nil
}

// NewServer creates a local analytics server
func NewServer(config ServerConfig) (*Tlytics, error) {
	if config.FlushPeriod == 0 {
		config.FlushPeriod = 5 * time.Second
	}
	
	if config.ServerPort == 0 {
		config.ServerPort = 8080
	}
	
	db, err := Init(config.DBPath)
	if err != nil {
		return nil, err
	}
	
	logger := NewLogger(db, config.FlushPeriod)
	server := newHTTPServer(logger, config.ServerPort)
	
	return &Tlytics{
		db:     db,
		logger: logger,
		server: server,
	}, nil
}

// New creates a client (for backwards compatibility, but NewClient is preferred)
func New(config Config) (*Client, error) {
	return NewClient(config)
}

// Emit sends an event to the server logger
func (t *Tlytics) Emit(event Event) error {
	return t.logger.Emit(event)
}

// GetLogger returns the server logger
func (t *Tlytics) GetLogger() *Logger {
	return t.logger
}


// StartServer starts the analytics server
func (t *Tlytics) StartServer() error {
	return t.server.Start()
}

// Flush manually flushes queued events in the server logger
func (t *Tlytics) Flush() {
	t.logger.Flush()
}

// Close properly closes the server instance
func (t *Tlytics) Close() error {
	t.logger.Stop()
	return t.db.Close()
}