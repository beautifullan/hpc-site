package models

import (
	"database/sql"
	"time"
)

type Benchmark struct {
	ID          int            `json:"id"`
	SoftwareID  int            `json:"software_id"`
	Name        string         `json:"name"`
	DatasetName sql.NullString `json:"dataset"`
	DatasetUrl  sql.NullString `json:"dataset_url"`
	Version     sql.NullString `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
}

type BenchmarkResults struct {
	ID          int          `json:"id"`
	BenchmarkID int          `json:"benchmark_id"`
	Hardware    []byte       `json:"hardware"`
	Metrics     []byte       `json:"metrics"`
	RunDate     sql.NullTime `json:"run_date"`
}

type BenchmarkResultResponse struct {
	ID          int64                  `json:"id"`
	BenchmarkID int                    `json:"benchmark_id"`
	Hardware    map[string]interface{} `json:"hardware"` // 接收非固定结构的 JSON
	Metrics     map[string]interface{} `json:"metrics"`  // 接收非固定结构的 JSON
	RunDate     *time.Time             `json:"run_date,omitempty"`
}
