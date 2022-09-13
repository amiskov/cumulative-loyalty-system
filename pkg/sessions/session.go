package sessions

import "time"

type Session struct {
	Id         string
	UserId     string
	Expiration time.Time
}
