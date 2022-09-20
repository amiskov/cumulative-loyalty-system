package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type iUserRepo interface {
	GetByID(context.Context, string) (*user.User, error)
}

type iSessionService interface {
	GetUserSession(string) (*session.Session, error)
}

type authMiddleware struct {
	repo           iUserRepo
	sessionService iSessionService
	noAuthUrls     map[string]struct{}
}

func NewAuthMiddleware(sess iSessionService, r iUserRepo, noAuthUrls map[string]struct{}) *authMiddleware {
	return &authMiddleware{
		repo:           r,
		sessionService: sess,
		noAuthUrls:     noAuthUrls,
	}
}

func (a *authMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.noAuthUrls[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		currentSession, err := a.sessionService.GetUserSession(token)
		if err != nil {
			logger.Log(r.Context()).Errorf("auth: can't get user session form token: %v", err)
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		// Pass user session further
		ctxWithAuth := context.WithValue(r.Context(), session.SessionKey, currentSession)
		next.ServeHTTP(w, r.WithContext(ctxWithAuth))
	})
}
