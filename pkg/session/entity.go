package session

import (
	"errors"
	"time"
)

type Session struct {
	ID         string
	UserID     string
	Expiration time.Time
}

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("session: no session found")
