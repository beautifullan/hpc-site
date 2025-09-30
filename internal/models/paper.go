package models

import "time"

type Paper struct {
	ID            string    `db:"id" json:"id"`
	Title         string    `db:"title" json:"title"`
	Authors       []string  `db:"authors" json:"authors"`
	Abstract      string    `db:"abstract" json:"abstract"`
	URL           string    `db:"url" json:"url"`
	SoftwareNames []string  `db:"software_names" json:"software_names"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}
