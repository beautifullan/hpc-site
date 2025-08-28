package models

import (
	"time"
)

type Software struct {
	ID         int       `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	Abstract   string    `db:"abstract" json:"abstract"`
	Homepage   string    `db:"homepage" json:"homepage"`
	Github     string    `db:"github" json:"github"`
	Categories []string  `db:"categories" json:"categories"`
	Tags       []string  `db:"tags" json:"tags"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
