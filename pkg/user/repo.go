package user

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
)

type repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *repo {
	return &repo{
		db: db,
	}
}

func (r *repo) Add(ctx context.Context, u *User) (string, error) {
	userID := 0
	err := r.db.QueryRowContext(ctx, "INSERT INTO users(login, password) VALUES($1, $2) RETURNING id",
		u.Login, u.Password).Scan(&userID)
	if err != nil {
		return ``, fmt.Errorf("user/repo: failed insert user, %w", err)
	}
	return strconv.Itoa(userID), nil
}

func (r *repo) GetByLoginAndPass(ctx context.Context, uname string, pass string) (*User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, login, password FROM users where login=$1", uname)
	u := new(User)
	if err := row.Scan(&u.ID, &u.Login, &u.Password); err != nil {
		return nil, fmt.Errorf("user/repo: row scan failed: %w", err)
	}
	// User found by login, now check if passwords are the same
	salt := string(u.Password[0:8])
	if !bytes.Equal(common.HashPass(pass, salt), u.Password) {
		return nil, errors.New("user/repo: password is invalid")
	}
	return u, nil
}

func (r *repo) UserExists(ctx context.Context, login string) (bool, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id FROM users where login=$1", login)
	u := new(User)
	if err := row.Scan(&u.ID); err != nil {
		return false, fmt.Errorf("user/repo.UserExists, could not scan row, user `%s` doesn't exist: %w", login, err)
	}
	return true, nil
}

func (r *repo) GetByID(ctx context.Context, uid string) (*User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, login FROM users where id=$1", uid)
	u := new(User)
	if err := row.Scan(&u.ID, &u.Login); err != nil {
		return u, fmt.Errorf("user/repo: could not scan row: %w", err)
	}
	return u, nil
}
