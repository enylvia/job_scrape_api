package models

import "time"

type AboutPage struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
