package sessions

import (
	"context"
	"errors"

	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type (
	sessionKey string

	SessionManager struct {
		secret []byte
		repo   *SessionRepo
	}
)

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("sessions: no session found")

func NewSessionManager(secret string, sr *SessionRepo) *SessionManager {
	return &SessionManager{
		secret: []byte(secret),
		repo:   sr,
	}
}

func (sm *SessionManager) Create(userId string) (string, error) {
	sessionId, err := sm.repo.Add(userId)
	if err != nil {
		return ``, err
	}
	return sessionId, nil
}

func (sm *SessionManager) Check(sessionId, userId string) error {
	return sm.repo.Check(sessionId, userId)
}

func GetAuthUser(ctx context.Context) (*user.User, error) {
	user, ok := ctx.Value(SessionKey).(*user.User)
	if !ok || user == nil {
		return nil, ErrNoAuth
	}
	return user, nil
}
