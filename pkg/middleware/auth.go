package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
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
	}
)

func NewAuthMiddleware(sm ISessionManager, ur IUserRepo) *Auth {
	return &Auth{
		UserRepo:       ur,
		SessionManager: sm,
	}
}

func (auth Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		userFromToken, err := auth.SessionManager.UserFromToken(authHeader)
		if err != nil {
			logger.Log(r.Context()).Errorf("can't get username from token: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		repoCtx, repoCtxCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer repoCtxCancel()
		user, err := auth.UserRepo.GetByID(repoCtx, userFromToken.ID)
		if err != nil {
			logger.Log(r.Context()).Errorf("auth: can't get the user form repo: %v", err)
			common.WriteMsg(w, "user not found", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), sessions.SessionKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
