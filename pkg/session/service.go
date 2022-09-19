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

	manager struct {
		secret []byte
		repo   *repo
	}

	jwtClaims struct {
		User user.User `json:"user"`
		jwt.StandardClaims
	}
)

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("sessions: no session found")

func NewSessionManager(secret string, sr *repo) *manager {
	return &manager{
		secret: []byte(secret),
		repo:   sr,
	}
}

// Returns logged in user if the user from JWT token is valid
// and the session is valid.
func (sm *manager) UserFromToken(authHeader string) (*user.User, error) {
	if authHeader == "" {
		return nil, errors.New("sessions: auth header not found")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return sm.secret, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("sessions: can't cast token to claim")
	}
	if !token.Valid {
		return nil, errors.New("sessions: token is not valid")
	}

	_, err = sm.Check(claims.User.ID, claims.Id)
	if err != nil {
		return nil, fmt.Errorf("sesssion/manager: session is not valid: %w", err)
	}

	return &claims.User, nil
}

// Goes through all user sessions and removes expired ones.
func (sm *manager) CleanupUserSessions(userID string) error {
	err := sm.repo.DestroyAll(userID)
	if err != nil {
		return fmt.Errorf("sessions/manager: failed destroying user sessions, %w", err)
	}
	return nil
}

func (sm *manager) Check(userID, sessionID string) (bool, error) {
	session, err := sm.repo.GetUserSession(sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("session/manager: failed get user session, %w", err)
	}

	// Check user session for expiration
	expiredTS := session.Expiration.Unix()
	nowTS := time.Now().Unix()
	if nowTS > expiredTS {
		return false, errors.New("session has beed expired")
	}

	// Prolongate session expiration time if it expires in less than 24 hours
	// because we don't want to kick off the active user.
	if expiredTS-nowTS < int64((24 * time.Hour).Seconds()) {
		newExpDate := time.Now().Add(90 * 24 * time.Hour).Unix()
		err := sm.repo.Add(userID, sessionID, newExpDate)
		if err != nil {
			log.Println("session/manager: can't save session to repo", err)
			return false, err
		}
	}

	return true, nil
}

func (sm *manager) CreateToken(user *user.User) (string, error) {
	sessionID := common.RandStringRunes(10)
	data := jwtClaims{
		User: *user,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour).Unix(), // 90 days
			IssuedAt:  time.Now().Unix(),
			Id:        sessionID,
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, data).SignedString(sm.secret)
	if err != nil {
		return ``, err
	}

	err = sm.repo.Add(user.ID, sessionID, data.ExpiresAt)
	if err != nil {
		return ``, fmt.Errorf("session/manager: can't add session to repo, %w", err)
	}

	return token, nil
}

func GetAuthUser(ctx context.Context) (*user.User, error) {
	user, ok := ctx.Value(SessionKey).(*user.User)
	if !ok || user == nil {
		return nil, ErrNoAuth
	}
	return user, nil
}