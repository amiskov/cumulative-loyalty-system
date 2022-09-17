package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type IRepo interface {
	UserExists(string) (bool, error)
	GetByLoginAndPass(string, string) (*User, error)
	Add(*User) (string, error)
}

type ISessionManager interface {
	CreateToken(*User) (string, error)
}

type service struct {
	repo IRepo
	sm   ISessionManager
}

var (
	errUserAlreadyExists = errors.New("user already exists")
	errUserNotFound      = errors.New("user not found")
)

func NewService(r IRepo, sm ISessionManager) *service {
	return &service{
		repo: r,
		sm:   sm,
	}
}

func (s *service) LoginUser(ctx context.Context, login, password string) (token string, err error) {
	usr, err := s.repo.GetByLoginAndPass(login, password)
	if err != nil {
		logger.Log(ctx).Errorf("can't get the user by login `%s` and password, %v", login, err)
		return ``, fmt.Errorf("can't get the user by login `%s`, %w", login, errUserNotFound)
	}

	token, err = s.sm.CreateToken(usr)
	if err != nil {
		logger.Log(ctx).Errorf("can't create JWT token from user: %v", err)
		return
	}

	return
}

func (s *service) RegUser(ctx context.Context, login, password string) (token string, err error) {
	userExists, _ := s.repo.UserExists(login)
	if userExists {
		logger.Log(ctx).Error(`user "%s" already exists`, login)
		return ``, fmt.Errorf("can't add `%s`, %w", login, errUserAlreadyExists)
	}

	salt := common.RandStringRunes(8)
	pass := common.HashPass(password, salt)
	user := &User{
		Login:    login,
		Password: pass,
		// Id is handled below
	}
	id, err := s.repo.Add(user)
	if err != nil {
		logger.Log(ctx).Errorf("user: can't add user to DB: %v", err)
		return
	}
	user.ID = id

	token, err = s.sm.CreateToken(user)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		return ``, err
	}

	return
}
