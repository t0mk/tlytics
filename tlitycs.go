package tlytics

import (
	"fmt"
	"time"
)

// Tlytics represents either a client or server instance
type Tlytics struct {
	db     *DB
	logger *Logger
	server *Server
	client *Client
	isServer bool
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
func NewClient(config Config) (*Tlytics, error) {
	if config.ServerURL == "" {
		return nil, fmt.Errorf("ServerURL is required")
	}

	client := newHTTPClient(config.ServerURL, config.FlushPeriod)
	
	return &Tlytics{
		client:   client,
		isServer: false,
	}, nil
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
		db:       db,
		logger:   logger,
		server:   server,
		isServer: true,
	}, nil
}

// New creates a client (for backwards compatibility, but NewClient is preferred)
func New(config Config) (*Tlytics, error) {
	return NewClient(config)
}

// Emit sends an event (works for both client and server)
func (t *Tlytics) Emit(event Event) error {
	if t.isServer {
		return t.logger.Emit(event)
	}
	return t.client.Emit(event)
}

// GetLogger returns the logger (only for server instances)
func (t *Tlytics) GetLogger() *Logger {
	if !t.isServer {
		return nil
	}
	return t.logger
}

// GetClient returns the client (only for client instances)
func (t *Tlytics) GetClient() *Client {
	if t.isServer {
		return nil
	}
	return t.client
}

// StartServer starts the analytics server (only for server instances)
func (t *Tlytics) StartServer() error {
	if !t.isServer {
		return fmt.Errorf("cannot start server on client instance")
	}
	return t.server.Start()
}

// Flush manually flushes queued events
func (t *Tlytics) Flush() {
	if t.isServer {
		t.logger.Flush()
	} else {
		t.client.Flush()
	}
}

// Close properly closes the tlytics instance
func (t *Tlytics) Close() error {
	if t.isServer {
		t.logger.Stop()
		return t.db.Close()
	} else {
		t.client.Stop()
		return nil
	}
}