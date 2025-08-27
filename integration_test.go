package tlytics

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestClientServerIntegration(t *testing.T) {
	// Create a test database file
	dbPath := "./test_integration.duckdb"
	defer os.Remove(dbPath)

	// Initialize server
	serverConfig := ServerConfig{
		DBPath:      dbPath,
		FlushPeriod: 100 * time.Millisecond,
		ServerPort:  8082, // Use different port to avoid conflicts
	}

	server, err := NewServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Start server in background
	go func() {
		if err := server.StartServer(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Verify server is running by checking health endpoint
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		t.Fatalf("Server not responding: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Server health check failed: status %d", resp.StatusCode)
	}

	// Initialize client
	clientConfig := Config{
		ServerURL:   "http://localhost:8082",
		FlushPeriod: 50 * time.Millisecond,
	}

	client, err := NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test EmitAndSend (immediate send)
	testEvent1 := Event{
		Key: "test_immediate",
		Data: map[string]interface{}{
			"action": "immediate_test",
			"value":  "direct_send",
		},
	}

	err = client.EmitAndSend(testEvent1)
	if err != nil {
		t.Fatalf("EmitAndSend failed: %v", err)
	}

	// Test regular Emit (queued)
	testEvent2 := Event{
		Key: "test_queued",
		Data: map[string]interface{}{
			"action": "queued_test",
			"value":  "batched_send",
		},
	}

	err = client.Emit(testEvent2)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	// Wait for queued events to be flushed
	time.Sleep(150 * time.Millisecond)

	// Wait for server to process and store events
	time.Sleep(200 * time.Millisecond)

	// Verify events were stored in the server database
	events, total, err := server.db.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("Failed to retrieve events from server: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 events stored, got %d", total)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 retrieved events, got %d", len(events))
	}

	// Check that both test events are present
	eventKeys := make(map[string]Event)
	for _, event := range events {
		eventKeys[event.Key] = event
	}

	// Verify immediate event
	if immediateEvent, found := eventKeys["test_immediate"]; found {
		if immediateEvent.Data["action"] != "immediate_test" {
			t.Errorf("Immediate event action mismatch: expected 'immediate_test', got %v", immediateEvent.Data["action"])
		}
		if immediateEvent.Data["value"] != "direct_send" {
			t.Errorf("Immediate event value mismatch: expected 'direct_send', got %v", immediateEvent.Data["value"])
		}
	} else {
		t.Error("Immediate event not found in stored events")
	}

	// Verify queued event
	if queuedEvent, found := eventKeys["test_queued"]; found {
		if queuedEvent.Data["action"] != "queued_test" {
			t.Errorf("Queued event action mismatch: expected 'queued_test', got %v", queuedEvent.Data["action"])
		}
		if queuedEvent.Data["value"] != "batched_send" {
			t.Errorf("Queued event value mismatch: expected 'batched_send', got %v", queuedEvent.Data["value"])
		}
	} else {
		t.Error("Queued event not found in stored events")
	}
}

func TestEmitAndSendConnectionError(t *testing.T) {
	// Test EmitAndSend with non-existent server
	clientConfig := Config{
		ServerURL:   "http://localhost:9999", // Non-existent server
		FlushPeriod: 50 * time.Millisecond,
	}

	client, err := NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	testEvent := Event{
		Key: "test_connection_error",
		Data: map[string]interface{}{
			"test": "value",
		},
	}

	// This should return an error since server is not available
	err = client.EmitAndSend(testEvent)
	if err == nil {
		t.Error("Expected EmitAndSend to return error when server is not available")
	}
}

func TestMultipleEventsIntegration(t *testing.T) {
	// Create a test database file
	dbPath := "./test_multiple_integration.duckdb"
	defer os.Remove(dbPath)

	// Initialize server
	serverConfig := ServerConfig{
		DBPath:      dbPath,
		FlushPeriod: 50 * time.Millisecond,
		ServerPort:  8083, // Different port
	}

	server, err := NewServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Start server in background
	go func() {
		if err := server.StartServer(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Initialize client
	clientConfig := Config{
		ServerURL:   "http://localhost:8083",
		FlushPeriod: 30 * time.Millisecond,
	}

	client, err := NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send multiple events using both methods
	numEvents := 5

	// Mix of immediate and queued events
	for i := 0; i < numEvents; i++ {
		event := Event{
			Key: fmt.Sprintf("test_event_%d", i),
			Data: map[string]interface{}{
				"index":  i,
				"method": "mixed",
			},
		}

		if i%2 == 0 {
			// Even indices: immediate send
			err = client.EmitAndSend(event)
			if err != nil {
				t.Fatalf("EmitAndSend failed for event %d: %v", i, err)
			}
		} else {
			// Odd indices: queued send
			err = client.Emit(event)
			if err != nil {
				t.Fatalf("Emit failed for event %d: %v", i, err)
			}
		}

		// Small delay between events
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all queued events to be flushed
	time.Sleep(100 * time.Millisecond)

	// Wait for server to process all events
	time.Sleep(100 * time.Millisecond)

	// Verify all events were stored
	events, total, err := server.db.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("Failed to retrieve events from server: %v", err)
	}

	if total != numEvents {
		t.Errorf("Expected %d events stored, got %d", numEvents, total)
	}

	if len(events) != numEvents {
		t.Errorf("Expected %d retrieved events, got %d", numEvents, len(events))
	}

	// Verify all events are present and have correct data
	eventMap := make(map[string]Event)
	for _, event := range events {
		eventMap[event.Key] = event
	}

	for i := 0; i < numEvents; i++ {
		expectedKey := fmt.Sprintf("test_event_%d", i)
		if event, found := eventMap[expectedKey]; found {
			if event.Data["index"] != float64(i) { // JSON unmarshals numbers as float64
				t.Errorf("Event %s index mismatch: expected %d, got %v", expectedKey, i, event.Data["index"])
			}
			if event.Data["method"] != "mixed" {
				t.Errorf("Event %s method mismatch: expected 'mixed', got %v", expectedKey, event.Data["method"])
			}
		} else {
			t.Errorf("Event %s not found in stored events", expectedKey)
		}
	}
}