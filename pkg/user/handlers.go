package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type iService interface {
	RegUser(ctx context.Context, login, pass string) (token string, err error)
	LoginUser(ctx context.Context, login, password string) (token string, err error)
	LogOutUser(ctx context.Context) error
}

type Handler struct {
	service iService
}

func NewHandler(s iService) *Handler {
	return &Handler{
		service: s,
	}
}

func (uh Handler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	login, pass, err := userFromRequest(r.Body)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	token, err := uh.service.RegUser(r.Context(), login, pass)
	if errors.Is(err, errUserAlreadyExists) {
		msg := fmt.Sprintf(`user "%s" already exists`, login)
		common.WriteMsg(w, msg, http.StatusConflict)
		return
	}
	if err != nil {
		common.WriteMsg(w, "can't add user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", `Bearer `+token)
	w.WriteHeader(http.StatusOK)
}

func (uh Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	login, pass, err := userFromRequest(r.Body)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	token, err := uh.service.LoginUser(r.Context(), login, pass)
	if errors.Is(err, errUserNotFound) {
		msg := fmt.Sprintf(`user "%s" not found`, login)
		common.WriteMsg(w, msg, http.StatusNotFound)
		return
	}
	if err != nil {
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", `Bearer `+token)
	w.WriteHeader(http.StatusOK)
}

func (uh Handler) LogOut(w http.ResponseWriter, r *http.Request) {
	err := uh.service.LogOutUser(r.Context())
	if err != nil {
		common.WriteMsg(w, "user logout failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func userFromRequest(reqBody io.ReadCloser) (login, password string, err error) {
	httpUser := &struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{}
	err = json.NewDecoder(reqBody).Decode(httpUser)
	if err != nil {
		return ``, ``, err
	}
	return httpUser.Login, httpUser.Password, nil
}
