package balance

import (
	"database/sql"
	"fmt"
)

type Repo struct {
	db *sql.DB
}

func NewBalanceRepo(db *sql.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) GetBalance(userId string) (float32, error) {
	row := r.db.QueryRow("SELECT balance FROM users where id=$1", userId)
	var balance float32
	if err := row.Scan(&balance); err != nil {
		return 0, fmt.Errorf("balance/repo: row scan failed: %w", err)
	}
	return balance, nil
}
