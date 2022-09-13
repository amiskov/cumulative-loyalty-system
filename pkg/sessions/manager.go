package sessions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	. "github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

const redisNS = "redditSessions"

type (
	sessionKey string

	SessionManager struct {
		secret []byte
		repo   *SessionRepo
	}

	jwtClaims struct {
		User user.User `json:"user"`
		jwt.StandardClaims
	}
)

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("sessions: no session found")

func NewSessionManager(secret string, sr *SessionRepo) *SessionManager {
	return &SessionManager{
		secret: []byte(secret),
		repo:   sr,
	}
}

// Returns logged in user if the user from JWT token is valid
// and the session is valid.
func (sm *SessionManager) UserFromToken(authHeader string) (*user.User, error) {
	if authHeader == "" {
		return nil, errors.New("sessions: auth header not found")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(sm.secret), nil
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

	_, redisErr := sm.Check(claims.User.Id, claims.Id)
	if redisErr != nil {
		return nil, fmt.Errorf("sesssion/manager: session is not valid: %v", redisErr)
	}

	return &claims.User, nil
}

// Goes through all user sessions and removes expired ones.
func (sm *SessionManager) CleanupUserSessions(userId string) error {
	err := sm.repo.DestroyAll(userId)
	if err != nil {
		return fmt.Errorf("sessions/manager: failed destroying user sessions, %w", err)
	}
	return nil
}

func (sm *SessionManager) Check(userId, sessionId string) (bool, error) {
	session, err := sm.repo.GetUserSession(sessionId, userId)
	if err != nil {
		return false, fmt.Errorf("session/manager: failed get user session, %w", err)
	}

	// Check user session for expiration
	expiredTs := session.Expiration.Unix()
	nowTs := time.Now().Unix()
	if nowTs > expiredTs {
		return false, errors.New("session has beed expired")
	}

	// Prolongate session expiration time if it expires in less than 24 hours
	// because we don't want to kick off the active user.
	if expiredTs-nowTs < int64(time.Duration(24*time.Hour).Seconds()) {
		newExpDate := time.Now().Add(90 * 24 * time.Hour).Unix()
		err := sm.repo.Add(userId, sessionId, newExpDate)
		if err != nil {
			log.Println("session/manager: can't save session to repo", err)
			return false, err
		}
	}

	return true, nil
}

func (sm *SessionManager) CreateToken(user *user.User) (string, error) {
	sessionID := RandStringRunes(10)
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

	err = sm.repo.Add(user.Id, sessionID, data.ExpiresAt)
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
