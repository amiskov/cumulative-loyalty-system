package sessions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gomodule/redigo/redis"

	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

const redisNS = "redditSessions"

type (
	sessionKey string

	SessionManager struct {
		secret []byte
		redis  redis.Conn
	}

	jwtClaims struct {
		User user.User `json:"user"`
		jwt.StandardClaims
	}
)

const SessionKey sessionKey = "authenticatedUser"

var ErrNoAuth = errors.New("sessions: no session found")

func NewSessionManager(secret string, conn redis.Conn) *SessionManager {
	return &SessionManager{
		secret: []byte(secret),
		redis:  conn,
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

	_, redisErr := sm.CheckRedis(claims.User.Id, claims.Id)
	if redisErr != nil {
		return nil, fmt.Errorf("sesssion/manager: Redis session is not valid: %v", redisErr)
	}

	return &claims.User, nil
}

// Goes through all user sessions and removes expired ones.
func (sm *SessionManager) CleanupUserSessions(userId string) error {
	sessions, err := redis.StringMap(sm.redis.Do("HGETALL", userId))
	if err != nil {
		log.Println("session/manager: can't HGETALL user sessions from Redis:", err)
		return err
	}

	nowTs := time.Now().Unix()
	for sessId, exp := range sessions {
		expTs, _ := strconv.ParseInt(exp, 10, 64)
		if nowTs > expTs {
			sm.redis.Do("HDEL", userId, sessId)
			log.Printf("session/manager: sessions %s removed (expired at %s)\n", sessId, exp)
		}
	}

	return nil
}

func (sm *SessionManager) CheckRedis(userId, sessionId string) (bool, error) {
	expirationData, err := redis.Bytes(sm.redis.Do("HGET", userId, sessionId))
	if err != nil {
		log.Println("session/manager: can't HGET from Redis:", err)
		return false, err
	}

	// Check user session for expiration
	expiredTs, _ := strconv.ParseInt(string(expirationData), 10, 64)
	nowTs := time.Now().Unix()
	if nowTs > expiredTs {
		return false, errors.New("session has beed expired")
	}

	// Prolongate session expiration time if it expires in less than 24 hours
	// because we don't want to kick off the active user.
	if expiredTs-nowTs < int64(time.Duration(24*time.Hour).Seconds()) {
		newExpDate := time.Now().Add(90 * 24 * time.Hour).Unix()
		err := sm.AddToRedis(userId, sessionId, newExpDate)
		if err != nil {
			log.Println("session/manager: failed add to Redis", err)
			return false, err
		}
	}

	return true, nil
}

func (sm *SessionManager) AddToRedis(userId, sessionId string, exp int64) error {
	_, err := sm.redis.Do("HSET", userId, sessionId, exp)
	if err != nil {
		return fmt.Errorf("session/manager: failed HSET to Redis: %v", err)
	}
	return nil
}

func (sm *SessionManager) CreateToken(user *user.User) (string, error) {
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
		return "", err
	}

	redisErr := sm.AddToRedis(user.Id, sessionID, data.ExpiresAt)
	if redisErr != nil {
		log.Println("session/manager: failed add to redis", redisErr)
		return ``, redisErr
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
