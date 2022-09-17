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

func (sr *SessionRepo) Add(userID, sessionID string, exp int64) error {
	expTime := time.Unix(exp, 0)
	_, err := sr.DB.Exec("INSERT INTO sessions(session_id, user_id, expiration_date) VALUES($1, $2, $3::timestamptz)",
		sessionID, userID, expTime)
	if err != nil {
		return fmt.Errorf("sessions/repo: failed insert into session %w", err)
	}
	return nil
}

func (sr *SessionRepo) GetUserSession(sessionID, userID string) (*Session, error) {
	q := `SELECT session_id, user_id, expiration_date FROM sessions WHERE session_id = $1 and user_id = $2`
	row := sr.DB.QueryRow(q, sessionID, userID)
	s := new(Session)
	err := row.Scan(&s.ID, &s.UserID, &s.Expiration)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (sr *SessionRepo) Destroy(sessionID string) error {
	_, err := sr.DB.Exec("DELETE FROM sessions WHERE session_id = $1", sessionID)
	if err != nil {
		return err
	}
	return nil
}

func (sr *SessionRepo) DestroyAll(userID string) error {
	_, err := sr.DB.Exec("DELETE FROM sessions WHERE user_id = $1", userID)
	if err != nil {
		return err
	}
	return nil
}
