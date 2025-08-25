package tlytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	serverURL   string
	httpClient  *http.Client
	queue       []Event
	flushPeriod time.Duration
	mutex       sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func newHTTPClient(serverURL string, flushPeriod time.Duration) *Client {
	if flushPeriod == 0 {
		flushPeriod = 5 * time.Second
	}

	client := &Client{
		serverURL:   serverURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		queue:       make([]Event, 0),
		flushPeriod: flushPeriod,
		stopCh:      make(chan struct{}),
	}

	client.wg.Add(1)
	go client.flushWorker()

	return client
}

func (c *Client) Emit(e Event) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	c.mutex.Lock()
	c.queue = append(c.queue, e)
	c.mutex.Unlock()

	return nil
}

func (c *Client) flushWorker() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.stopCh:
			c.flush() // Final flush before shutdown
			return
		}
	}
}

func (c *Client) flush() {
	c.mutex.Lock()
	if len(c.queue) == 0 {
		c.mutex.Unlock()
		return
	}

	events := make([]Event, len(c.queue))
	copy(events, c.queue)
	c.queue = c.queue[:0] // Clear the queue
	c.mutex.Unlock()

	// Send events to remote server
	if err := c.sendEvents(events); err != nil {
		// In a production system, you might want to implement retry logic
		// or log this error somewhere
		_ = err
	}
}

func (c *Client) sendEvents(events []Event) error {
	jsonData, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	resp, err := c.httpClient.Post(c.serverURL+"/events", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Flush() {
	c.flush()
}

func (c *Client) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}