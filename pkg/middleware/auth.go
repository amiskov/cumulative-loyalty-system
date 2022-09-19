package middleware

import (
	"context"
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type IUserRepo interface {
	GetByID(context.Context, string) (*user.User, error)
}

type ISessionService interface {
	UserFromToken(string) (*user.User, error)
	SessionFromToken(authHeader string) (*session.Session, error)
}

type Auth struct {
	repo           IUserRepo
	sessionService ISessionService
	noAuthUrls     map[string]struct{}
}

func NewAuthMiddleware(sess ISessionService, r IUserRepo, noAuthUrls map[string]struct{}) *Auth {
	return &Auth{
		repo:           r,
		sessionService: sess,
		noAuthUrls:     noAuthUrls,
	}
}

func (auth Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.noAuthUrls[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		currentSession, err := auth.sessionService.SessionFromToken(r.Header.Get("Authorization"))
		if err != nil {
			logger.Log(r.Context()).Errorf("auth: can't get the session form token: %v", err)
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		// Pass user session further
		ctxWithAuth := context.WithValue(r.Context(), session.SessionKey, currentSession)
		next.ServeHTTP(w, r.WithContext(ctxWithAuth))
	})
}
