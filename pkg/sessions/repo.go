package sessions

import (
	"database/sql"
	"fmt"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
)

type SessionRepo struct {
	DB *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{
		DB: db,
	}
}

func (sr *SessionRepo) Add(userId string) (string, error) {
	sessID := common.RandStringRunes(32)
	_, err := sr.DB.Exec("INSERT INTO sessions(session_id, user_id) VALUES($1, $2)", sessID, userId)
	if err != nil {
		return ``, fmt.Errorf("sessions/repo: failed insert into session %w", err)
	}
	return sessID, nil
}

func (sr *SessionRepo) Check(sessionId, userId string) error {
	row := sr.DB.QueryRow(`SELECT user_id FROM sessions WHERE session_id = $1 and user_id = $2`,
		sessionId, userId)
	var uid string
	err := row.Scan(&uid)
	if err == sql.ErrNoRows {
		return ErrNoAuth
	} else if err != nil {
		return err
	}
	return nil
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
