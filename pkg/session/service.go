package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type iSessionRepo interface {
	Destroy(sessionID string) error
	GetUserSession(sessionID, userID string) (*Session, error)
	Add(userID, sessionID string, exp int64) error
}

type sessionKey string

type service struct {
	secret []byte
	repo   iSessionRepo
}

type jwtClaims struct {
	User user.User `json:"user"`
	jwt.StandardClaims
}

func NewSessionService(secret string, sr *repo) *service {
	return &service{
		secret: []byte(secret),
		repo:   sr,
	}
}

func (s *service) GetUserSession(token string) (*Session, error) {
	jwtToken, err := jwt.ParseWithClaims(token, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return s.secret, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("session: can't cast token to claim")
	}
	if !jwtToken.Valid {
		return nil, errors.New("session: token is not valid")
	}

	return s.repo.GetUserSession(claims.Id, claims.User.ID)
}

func (s *service) DestroySession(ctx context.Context) error {
	sessionID, err := GetAuthUserSessionID(ctx)
	if err != nil {
		return err
	}
	return s.repo.Destroy(sessionID)
}

func (s *service) CreateToken(user *user.User) (string, error) {
	sessionID := common.RandStringRunes(10)
	data := jwtClaims{
		User: *user,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour).Unix(), // 90 days
			IssuedAt:  time.Now().Unix(),
			Id:        sessionID,
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, data).SignedString(s.secret)
	if err != nil {
		return ``, err
	}

	err = s.repo.Add(user.ID, sessionID, data.ExpiresAt)
	if err != nil {
		return ``, fmt.Errorf("session: can't add session to repo, %w", err)
	}

	return token, nil
}

func GetAuthUserID(ctx context.Context) (string, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return ``, ErrNoAuth
	}
	return sess.UserID, nil
}

func GetAuthUserSessionID(ctx context.Context) (string, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return ``, ErrNoAuth
	}
	return sess.ID, nil
}
