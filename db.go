package tlytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn  *sql.DB
	path  string
	mutex sync.Mutex
}

func Init(dbPath string) (*DB, error) {
	db := &DB{path: dbPath}

	if err := db.createDBIfNotExists(); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.conn = conn

	if err := db.createTableIfNotExists(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) createDBIfNotExists() error {
	// For DuckDB, we don't need to pre-create the file
	// DuckDB will create it automatically when we connect
	return nil
}

func (db *DB) createTableIfNotExists() error {
	query := `
	CREATE TABLE IF NOT EXISTS tlytics (
		key TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		data TEXT
	);`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) InsertEvents(events []Event) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO tlytics (key, timestamp, data) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, event := range events {
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}

		_, err = stmt.Exec(event.Key, event.Timestamp, string(dataJSON))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) GetEvents(limit, offset int) ([]Event, int, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Get total count
	var totalCount int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM tlytics").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated events
	query := "SELECT key, timestamp, data FROM tlytics ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var dataJSON string

		err := rows.Scan(&event.Key, &event.Timestamp, &dataJSON)
		if err != nil {
			return nil, 0, err
		}

		// Parse JSON data
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, 0, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, totalCount, nil
}
