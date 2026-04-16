package models

import "time"

type PublishLog struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	Response  string    `json:"response"`
	CreatedAt time.Time `json:"created_at"`
}
