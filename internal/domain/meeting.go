package domain

import "time"

// Meeting stores a transcribed meeting and its summary.
type Meeting struct {
	ID         int64
	UserID     int64
	Transcript string
	Summary    string
	CreatedAt  time.Time
}
