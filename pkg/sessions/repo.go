package sessions

import (
	"database/sql"
	"fmt"
	"time"
)

type SessionRepo struct {
	DB *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{
		DB: db,
	}
}

func (sr *SessionRepo) Add(userId, sessionId string, exp int64) error {
	expTime := time.Unix(exp, 0)
	_, err := sr.DB.Exec("INSERT INTO sessions(session_id, user_id, expiration_date) VALUES($1, $2, $3::timestamptz)",
		sessionId, userId, expTime)
	if err != nil {
		return fmt.Errorf("sessions/repo: failed insert into session %w", err)
	}
	return nil
}

func (sr *SessionRepo) GetUserSession(sessionId, userId string) (*Session, error) {
	row := sr.DB.QueryRow(`SELECT session_id, user_id, expiration_date FROM sessions WHERE session_id = $1 and user_id = $2`,
		sessionId, userId)
	s := new(Session)
	err := row.Scan(&s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (sr *SessionRepo) Destroy(sessionId string) error {
	_, err := sr.DB.Exec("DELETE FROM sessions WHERE session_id = $1", sessionId)
	if err != nil {
		return err
	}
	return nil
}

func (sr *SessionRepo) DestroyAll(userId string) error {
	_, err := sr.DB.Exec("DELETE FROM sessions WHERE user_id = $1", userId)
	if err != nil {
		return err
	}
	return nil
}
