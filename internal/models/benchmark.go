package models

import "time"

type Benchmark struct {
	ID         int            `json:"id"`
	SoftwareID int            `json:"software_id"`
	Name       string         `json:"name"`
	Dataset    string         `json:"dataset"`
	Hardware   map[string]any `json:"hardware"` // JSONB → Go map
	Metrics    map[string]any `json:"metrics"`  // JSONB → Go map
	Version    string         `json:"version"`
	CreatedAt  time.Time      `json:"created_at"`
}
