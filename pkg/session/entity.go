package session

import "time"

type Session struct {
	ID         string
	UserID     string
	Expiration time.Time
}
