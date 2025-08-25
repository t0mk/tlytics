package tlytics

import (
	"os"
	"testing"
	"time"
)

func TestEmitAndGetEvents(t *testing.T) {
	// Create a test database file
	dbPath := "./test_emit.duckdb"
	defer os.Remove(dbPath) // Clean up after test
	
	// Initialize database
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Create logger with short flush period
	logger := NewLogger(db, 100*time.Millisecond)
	defer logger.Stop()
	
	// Test data
	testEvents := []Event{
		{
			Key: "test_event_1",
			Data: map[string]interface{}{
				"user":   "alice",
				"action": "login",
			},
		},
		{
			Key: "test_event_2", 
			Data: map[string]interface{}{
				"user":   "bob",
				"action": "logout",
			},
		},
	}
	
	// Emit events
	for _, event := range testEvents {
		err := logger.Emit(event)
		if err != nil {
			t.Fatalf("Failed to emit event: %v", err)
		}
	}
	
	// Wait for flush to occur
	time.Sleep(200 * time.Millisecond)
	
	// Retrieve events from database
	retrievedEvents, total, err := db.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	
	// Verify total count
	if total != len(testEvents) {
		t.Errorf("Expected %d events, got %d", len(testEvents), total)
	}
	
	// Verify we got the right number of events
	if len(retrievedEvents) != len(testEvents) {
		t.Errorf("Expected %d retrieved events, got %d", len(testEvents), len(retrievedEvents))
	}
	
	// Verify event data (check that both expected events are present)
	expectedEventKeys := make(map[string]Event)
	for _, event := range testEvents {
		expectedEventKeys[event.Key] = event
	}
	
	retrievedEventKeys := make(map[string]Event)
	for _, event := range retrievedEvents {
		retrievedEventKeys[event.Key] = event
	}
	
	// Check that we got all expected events
	for key, expected := range expectedEventKeys {
		retrieved, found := retrievedEventKeys[key]
		if !found {
			t.Errorf("Expected event with key %s not found in retrieved events", key)
			continue
		}
		
		// Check data fields
		if retrieved.Data["user"] != expected.Data["user"] {
			t.Errorf("Event %s: expected user %s, got %s", key, expected.Data["user"], retrieved.Data["user"])
		}
		
		if retrieved.Data["action"] != expected.Data["action"] {
			t.Errorf("Event %s: expected action %s, got %s", key, expected.Data["action"], retrieved.Data["action"])
		}
		
		// Verify timestamp is set
		if retrieved.Timestamp.IsZero() {
			t.Errorf("Event %s: timestamp should not be zero", key)
		}
	}
}

func TestEmitWithManualFlush(t *testing.T) {
	// Create a test database file
	dbPath := "./test_manual_flush.duckdb"
	defer os.Remove(dbPath) // Clean up after test
	
	// Initialize database  
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Create logger with very long flush period so we control flushing
	logger := NewLogger(db, 1*time.Hour)
	defer logger.Stop()
	
	// Emit an event
	testEvent := Event{
		Key: "manual_test",
		Data: map[string]interface{}{
			"test": "value",
		},
	}
	
	err = logger.Emit(testEvent)
	if err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}
	
	// Before manual flush - should have 0 events in DB
	events, total, err := db.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("Failed to get events before flush: %v", err)
	}
	
	if total != 0 {
		t.Errorf("Expected 0 events before flush, got %d", total)
	}
	
	if len(events) != 0 {
		t.Errorf("Expected 0 retrieved events before flush, got %d", len(events))
	}
	
	// Manually trigger flush
	logger.Flush()
	
	// After manual flush - should have 1 event in DB
	events, total, err = db.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("Failed to get events after flush: %v", err)
	}
	
	if total != 1 {
		t.Errorf("Expected 1 event after flush, got %d", total)
	}
	
	if len(events) != 1 {
		t.Errorf("Expected 1 retrieved event after flush, got %d", len(events))
	}
	
	// Verify the event data
	if len(events) > 0 {
		retrieved := events[0]
		if retrieved.Key != testEvent.Key {
			t.Errorf("Expected key %s, got %s", testEvent.Key, retrieved.Key)
		}
		
		if retrieved.Data["test"] != testEvent.Data["test"] {
			t.Errorf("Expected test value %s, got %s", testEvent.Data["test"], retrieved.Data["test"])
		}
	}
}

func TestPagination(t *testing.T) {
	// Create a test database file
	dbPath := "./test_pagination.duckdb"
	defer os.Remove(dbPath) // Clean up after test
	
	// Initialize database
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Create logger
	logger := NewLogger(db, 50*time.Millisecond)
	defer logger.Stop()
	
	// Emit 5 events
	for i := 0; i < 5; i++ {
		event := Event{
			Key: "paginate_test",
			Data: map[string]interface{}{
				"index": i,
			},
		}
		
		err := logger.Emit(event)
		if err != nil {
			t.Fatalf("Failed to emit event %d: %v", i, err)
		}
		
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}
	
	// Wait for flush
	time.Sleep(100 * time.Millisecond)
	
	// Test pagination: get 2 events per page
	events, total, err := db.GetEvents(2, 0) // Page 1
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}
	
	if total != 5 {
		t.Errorf("Expected total 5 events, got %d", total)
	}
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events on page 1, got %d", len(events))
	}
	
	// Get page 2
	events2, total2, err := db.GetEvents(2, 2) // Page 2 (offset 2)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}
	
	if total2 != 5 {
		t.Errorf("Expected total 5 events on page 2, got %d", total2)
	}
	
	if len(events2) != 2 {
		t.Errorf("Expected 2 events on page 2, got %d", len(events2))
	}
	
	// Get page 3 (last page)
	events3, total3, err := db.GetEvents(2, 4) // Page 3 (offset 4)
	if err != nil {
		t.Fatalf("Failed to get page 3: %v", err)
	}
	
	if total3 != 5 {
		t.Errorf("Expected total 5 events on page 3, got %d", total3)
	}
	
	if len(events3) != 1 {
		t.Errorf("Expected 1 event on page 3, got %d", len(events3))
	}
	
	// Verify events are in descending timestamp order (newest first)
	// The newest event should have the highest index
	if len(events) > 0 {
		// First event on page 1 should have index 4 (newest)
		if events[0].Data["index"] != float64(4) {
			t.Errorf("Expected first event to have index 4, got %v", events[0].Data["index"])
		}
	}
	
	if len(events3) > 0 {
		// Last event on page 3 should have index 0 (oldest)
		if events3[0].Data["index"] != float64(0) {
			t.Errorf("Expected last event to have index 0, got %v", events3[0].Data["index"])
		}
	}
}