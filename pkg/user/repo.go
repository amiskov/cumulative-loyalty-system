package user

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) Add(u *User) (string, error) {
	userID := 0
	err := r.db.QueryRow("INSERT INTO users(login, password) VALUES($1, $2) RETURNING id",
		u.Login, u.Password).Scan(&userID)
	if err != nil {
		return ``, fmt.Errorf("user/repo: failed insert user, %w", err)
	}
	return strconv.Itoa(userID), nil
}

func (r *UserRepo) GetByLoginAndPass(uname string, pass string) (*User, error) {
	row := r.db.QueryRow("SELECT id, login, password FROM users where login=$1", uname)
	u := new(User)
	if err := row.Scan(&u.Id, &u.Login, &u.Password); err != nil {
		return nil, fmt.Errorf("user/repo: row scan failed: %w", err)
	}
	// User found by login, now check if passwords are the same
	salt := string(u.Password[0:8])
	if !bytes.Equal(common.HashPass(pass, salt), u.Password) {
		return nil, errors.New("user/repo: password is invalid")
	}
	return u, nil
}

func (r *UserRepo) UserExists(login string) bool {
	row := r.db.QueryRow("SELECT id FROM users where login=$1", login)
	u := new(User)
	if err := row.Scan(&u.Id); err != nil {
		log.Printf("user/repo.UserExists, could not scan row, user `%s` doesn't exist: %v", login, err)
		return false
	}
	return true
}

func (r *UserRepo) GetById(ctx context.Context, uid string) (*User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, login FROM users where id=$1", uid)
	u := new(User)
	if err := row.Scan(&u.Id, &u.Login); err != nil {
		return u, fmt.Errorf("user/repo: could not scan row: %w", err)
	}
	return u, nil
}
