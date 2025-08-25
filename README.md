# Tlytics

A lightweight, real-time analytics microservice written in Go. Tlytics is designed as a centralized analytics collection service that applications connect to via HTTP client, providing event tracking and analytics capabilities with SQLite storage, batch processing, and HTTP API endpoints.

## Architecture

Tlytics follows a **microservice architecture** where:

- **Analytics Server**: A standalone microservice that collects, stores, and serves analytics data
- **Client Libraries**: Applications integrate using HTTP clients that send events to the remote server
- **Centralized Storage**: All analytics data is stored in a single SQLite database on the server
- **Network Communication**: All event data is sent over HTTP to the remote server

## Features

- **Remote Event Collection**: HTTP client automatically sends events to remote analytics server
- **Microservice Design**: Standalone analytics server that multiple applications can connect to
- **Batch Processing**: Efficient client-side batching with configurable flush periods  
- **SQLite Storage**: Persistent storage using SQLite database on the server
- **HTTP API**: REST endpoints for event submission and data retrieval
- **Gin Middleware**: Ready-to-use middleware for automatic request tracking
- **Docker Support**: Containerized deployment with Docker and Docker Compose
- **Pagination**: Built-in pagination support for event retrieval
- **Network Resilience**: Client handles network errors gracefully without blocking requests

## Quick Start

### Running the Analytics Server

```bash
# Clone the repository
git clone <repository-url>
cd tlytics

# Install dependencies
go mod download

# Run the analytics server
go run cmd/tlytics/main.go --port 8081
```

### Connecting from Your Application

```go
import "tlytics"

// Configure client to connect to remote analytics server
config := tlytics.Config{
    ServerURL:   "http://192.168.1.100:8081", // Your analytics server
    FlushPeriod: 5 * time.Second,
}

analytics, err := tlytics.NewClient(config)
if err != nil {
    log.Fatal("Failed to initialize analytics client:", err)
}
defer analytics.Close()

// Send events
analytics.Emit(tlytics.Event{
    Key: "user_signup",
    Data: map[string]interface{}{
        "user_id": "123",
        "source": "web",
    },
})
```

### Using Docker

```bash
# Build and run with Docker Compose
docker-compose up --build

# Or build and run manually
docker build -t tlytics .
docker run -p 8081:8081 -v $(pwd)/data:/data tlytics
```

## Configuration

The server can be configured using command-line flags:

- `--db`: Path to SQLite database file (default: `./analytics.db`)
- `--port`: Port for analytics collection server (default: `8081`)
- `--flush`: Flush period for batching events (default: `5s`)

Example:
```bash
./tlytics --db /data/analytics.sqlite --port 8080 --flush 10s
```

## API Endpoints

### POST /events
Submit analytics events as JSON array.

```bash
curl -X POST http://localhost:8081/events \
  -H "Content-Type: application/json" \
  -d '[{
    "key": "page_view",
    "timestamp": "2025-08-25T10:00:00Z",
    "data": {
      "page": "home",
      "user_id": "123"
    }
  }]'
```

### POST /batch
Submit batch of events with flexible JSON structure.

```bash
curl -X POST http://localhost:8081/batch \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {"key": "click", "data": {"button": "signup"}},
      {"key": "view", "data": {"page": "about"}}
    ]
  }'
```

### GET /health
Health check endpoint.

```bash
curl http://localhost:8081/health
```

Response:
```json
{
  "status": "healthy",
  "port": 8081
}
```

### GET /view
Retrieve stored events with pagination.

```bash
# Get first page (10 events)
curl http://localhost:8081/view

# Get specific page and page size
curl "http://localhost:8081/view?page=2&page_size=20"
```

Response:
```json
{
  "events": [...],
  "total": 150,
  "page": 1,
  "page_size": 10,
  "total_pages": 15
}
```

## Usage with Gin Framework

### Client Integration

Connect your Gin application to a remote Tlytics analytics server:

```go
package main

import (
    "log"
    "time"
    "github.com/gin-gonic/gin"
    "tlytics"
)

func main() {
    // Configure client to connect to remote analytics server
    config := tlytics.Config{
        ServerURL:   "http://192.168.1.100:8081", // Remote analytics server
        FlushPeriod: 3 * time.Second,
    }
    
    analytics, err := tlytics.NewClient(config)
    if err != nil {
        log.Fatal("Failed to initialize tlytics client:", err)
    }
    defer analytics.Close()
    
    // Create Gin router with analytics middleware
    r := gin.Default()
    r.Use(tlytics.GinMiddleware(analytics))
    
    // Track specific events on routes
    r.GET("/api/users", 
        tlytics.TrackEvent(analytics, "api_access", map[string]interface{}{
            "endpoint": "users",
            "action": "list",
        }), 
        func(c *gin.Context) {
            c.JSON(200, gin.H{"users": []string{"alice", "bob"}})
        })
    
    // Manual event tracking
    r.GET("/", func(c *gin.Context) {
        analytics.Emit(tlytics.Event{
            Key: "page_view",
            Data: map[string]interface{}{
                "page": "home",
                "user_id": "123",
            },
        })
        c.JSON(200, gin.H{"message": "Hello World"})
    })
    
    log.Printf("Analytics client configured to send to: %s", config.ServerURL)
    r.Run(":8080")
}
```

### Manual HTTP Client Usage

If you prefer to use standard HTTP client without the Tlytics library:

```bash
# Send events directly via HTTP
curl -X POST http://192.168.1.100:8081/events \
  -H "Content-Type: application/json" \
  -d '[{
    "key": "user_action",
    "timestamp": "2025-08-25T10:00:00Z",
    "data": {
      "action": "click",
      "button": "signup"
    }
  }]'
```

## Event Structure

Events have the following structure:

```go
type Event struct {
    Key       string                 `json:"key"`       // Event identifier
    Timestamp time.Time              `json:"timestamp"` // When the event occurred
    Data      map[string]interface{} `json:"data"`      // Event payload
}
```

## Architecture

### Components

- **Logger**: Handles event queuing and batch processing
- **DB**: SQLite database interface with connection management  
- **Server**: HTTP API server with endpoints for event collection and retrieval
- **Middleware**: Gin middleware for automatic request tracking

### Data Flow

1. Events are submitted via HTTP API or middleware
2. Events are queued in memory by the Logger
3. Events are automatically flushed to SQLite database at configurable intervals
4. Events can be retrieved via pagination API

## Testing

Run the test suite:

```bash
go test -v
```

Tests cover:
- Event emission and retrieval
- Manual and automatic flushing
- Database pagination
- Event ordering (newest first)

## Deployment

### Docker Deployment

Deploy the analytics server using Docker:

```bash
# Build and run the analytics server
docker-compose up --build

# Or run manually
docker build -t tlytics .
docker run -p 8081:8081 -v $(pwd)/data:/data tlytics
```

The included `docker-compose.yml` provides a production-ready deployment:

```yaml
version: '3.8'

services:
  tlytics:
    image: t0mk/tlytics:latest
    ports:
      - "8081:8081"
    volumes:
      - ./data:/data
    environment:
      - TLYTICS_DB=/data/analytics.sqlite
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8081/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Architecture Deployment

**Recommended Architecture:**
1. **Single Analytics Server**: Deploy one Tlytics server (e.g., at `192.168.1.100:8081`)
2. **Multiple Client Applications**: Each application connects using `tlytics.NewClient()`
3. **Centralized Data**: All analytics stored in one SQLite database
4. **Horizontal Scaling**: Add more client applications without affecting the server

```
┌─────────────────┐    HTTP     ┌─────────────────┐
│   Web App       │────────────▶│                 │
│   :8080         │             │   Tlytics       │
└─────────────────┘             │   Server        │
                                │   :8081         │
┌─────────────────┐    HTTP     │                 │
│   API Service   │────────────▶│  ┌─────────────┐│
│   :9000         │             │  │   SQLite    ││
└─────────────────┘             │  │   Database  ││
                                │  └─────────────┘│
┌─────────────────┐    HTTP     │                 │
│   Background    │────────────▶│                 │
│   Worker        │             │                 │
└─────────────────┘             └─────────────────┘
```

## Performance Considerations

- Events are batched in memory before writing to database
- Configurable flush periods balance between data safety and performance
- SQLite provides good performance for most analytics workloads
- Pagination prevents large result sets from overwhelming clients
- Automatic timestamps reduce client-side complexity

## License

[Add your license information here]

## Contributing

[Add contributing guidelines here]