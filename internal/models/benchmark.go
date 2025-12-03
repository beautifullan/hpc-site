package models

import "time"

type Benchmark struct {
	ID          int       `json:"id"`
	SoftwareID  int       `json:"software_id"`
	Name        string    `json:"name"`
	DatasetName string    `json:"dataset"`
	DatasetUrl  string    `json:"dataset_url"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}
