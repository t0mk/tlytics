package tlytics

import "time"

type Event struct {
	Key       string                 `json:"key" db:"key"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
	Data      map[string]interface{} `json:"data" db:"data"`
}