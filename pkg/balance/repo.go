package balance

import "database/sql"

type Repo struct {
	db *sql.DB
}

func NewBalanceRepo(db *sql.DB) *Repo {
	return &Repo{
		db: db,
	}
}
