package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type (
	IUserRepo interface {
		GetByID(context.Context, string) (*user.User, error)
	}
	ISessionManager interface {
		UserFromToken(string) (*user.User, error)
	}
	Auth struct {
		UserRepo       IUserRepo
		SessionManager ISessionManager
		noAuthUrls     map[string]struct{}
	}
)

func NewAuthMiddleware(sm ISessionManager, ur IUserRepo, noAuthUrls map[string]struct{}) *Auth {
	return &Auth{
		UserRepo:       ur,
		SessionManager: sm,
		noAuthUrls:     noAuthUrls,
	}
}

func (auth Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.noAuthUrls[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		userFromToken, err := auth.SessionManager.UserFromToken(r.Header.Get("Authorization"))
		if err != nil {
			logger.Log(r.Context()).Errorf("can't get username from token: %v", err)
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		repoCtx, repoCtxCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer repoCtxCancel()
		user, err := auth.UserRepo.GetByID(repoCtx, userFromToken.ID)
		if err != nil {
			logger.Log(r.Context()).Errorf("auth: can't get the user form repo: %v", err)
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), session.SessionKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
