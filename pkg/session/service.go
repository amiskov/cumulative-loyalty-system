package session

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type (
	sessionKey string

	service struct {
		secret []byte
		repo   *repo
	}

	jwtClaims struct {
		User user.User `json:"user"`
		jwt.StandardClaims
	}
)

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("session: no session found")

func NewSessionService(secret string, sr *repo) *service {
	return &service{
		secret: []byte(secret),
		repo:   sr,
	}
}

func (s *service) SessionFromToken(authHeader string) (*Session, error) {
	if authHeader == "" {
		return nil, errors.New("session: auth header not found")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return s.secret, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	log.Printf("Claims! %#v\n", claims)
	if !ok {
		return nil, errors.New("session: can't cast token to claim")
	}
	if !token.Valid {
		return nil, errors.New("session: token is not valid")
	}

	_, err = s.Check(claims.User.ID, claims.Id)
	if err != nil {
		return nil, fmt.Errorf("session: session is not valid: %w", err)
	}

	session := &Session{
		ID:     claims.Id,
		UserID: claims.User.ID,
	}

	return session, nil
}

// Returns logged in user if the user from JWT token is valid
// and the session is valid.
func (s *service) UserFromToken(authHeader string) (*user.User, error) {
	if authHeader == "" {
		return nil, errors.New("session: auth header not found")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return s.secret, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("session: can't cast token to claim")
	}
	if !token.Valid {
		return nil, errors.New("session: token is not valid")
	}

	_, err = s.Check(claims.User.ID, claims.Id)
	if err != nil {
		return nil, fmt.Errorf("session: session is not valid: %w", err)
	}

	return &claims.User, nil
}

func (s *service) DestroySession(sessionID string) error {
	return s.repo.Destroy(sessionID)
}

// Goes through all user sessions and removes expired ones.
func (s *service) CleanupUserSessions(userID string) error {
	err := s.repo.DestroyAll(userID)
	if err != nil {
		return fmt.Errorf("session: failed destroying user sessions, %w", err)
	}
	return nil
}

func (s *service) Check(userID, sessionID string) (bool, error) {
	session, err := s.repo.GetUserSession(sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("session: failed get user session, %w", err)
	}

	// Check user session for expiration
	expiredTS := session.Expiration.Unix()
	nowTS := time.Now().Unix()
	if nowTS > expiredTS {
		return false, errors.New("session: session has beed expired")
	}

	// Prolongate session expiration time if it expires in less than 24 hours
	// because we don't want to kick off the active user.
	if expiredTS-nowTS < int64((24 * time.Hour).Seconds()) {
		newExpDate := time.Now().Add(90 * 24 * time.Hour).Unix()
		err := s.repo.Add(userID, sessionID, newExpDate)
		if err != nil {
			log.Println("session: can't save session to repo", err)
			return false, err
		}
	}

	return true, nil
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
	s, ok := ctx.Value(SessionKey).(*Session)
	if !ok || s == nil {
		return ``, ErrNoAuth
	}
	return s.UserID, nil
}

func GetAuthUserSessionID(ctx context.Context) (string, error) {
	s, ok := ctx.Value(SessionKey).(*Session)
	if !ok || s == nil {
		return ``, ErrNoAuth
	}
	return s.ID, nil
}
