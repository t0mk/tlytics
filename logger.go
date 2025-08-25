package tlytics

import (
	"sync"
	"time"
)

type Logger struct {
	db          *DB
	queue       []Event
	flushPeriod time.Duration
	mutex       sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewLogger(db *DB, flushPeriod time.Duration) *Logger {
	logger := &Logger{
		db:          db,
		queue:       make([]Event, 0),
		flushPeriod: flushPeriod,
		stopCh:      make(chan struct{}),
	}
	
	logger.wg.Add(1)
	go logger.flushWorker()
	
	return logger
}

func (l *Logger) Emit(e Event) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	
	l.mutex.Lock()
	l.queue = append(l.queue, e)
	l.mutex.Unlock()
	
	return nil
}

func (l *Logger) flushWorker() {
	defer l.wg.Done()
	
	ticker := time.NewTicker(l.flushPeriod)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			l.flush()
		case <-l.stopCh:
			l.flush() // Final flush before shutdown
			return
		}
	}
}

func (l *Logger) flush() {
	l.mutex.Lock()
	if len(l.queue) == 0 {
		l.mutex.Unlock()
		return
	}
	
	events := make([]Event, len(l.queue))
	copy(events, l.queue)
	l.queue = l.queue[:0] // Clear the queue
	l.mutex.Unlock()
	
	// Insert events to database
	if err := l.db.InsertEvents(events); err != nil {
		// In a production system, you might want to log this error
		// or implement a retry mechanism
		_ = err
	}
}

func (l *Logger) Stop() {
	close(l.stopCh)
	l.wg.Wait()
}

func (l *Logger) Flush() {
	l.flush()
}