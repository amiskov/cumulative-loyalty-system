package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type (
	UserRepo interface {
		UserExists(string) bool
		GetByLoginAndPass(string, string) (*user.User, error)
		Add(*user.User) (string, error)
	}

	ISessionManager interface {
		CreateToken(*user.User) (string, error)
		// CleanupUserSessions(userId string) error
	}

	UserHandler struct {
		Repo           UserRepo
		SessionManager ISessionManager
	}

	HttpUser struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
)

func NewUserHandler(r UserRepo, sm ISessionManager) *UserHandler {
	return &UserHandler{
		Repo:           r,
		SessionManager: sm,
	}
}

func (uh UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	httpUser := new(HttpUser)
	err := common.ParseReqBody(r.Body, httpUser)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	if uh.Repo.UserExists(httpUser.Login) {
		msg := fmt.Sprintf(`user "%s" already exists`, httpUser.Login)
		logger.Log(r.Context()).Error(msg)
		common.WriteMsg(w, msg, http.StatusConflict)
		return
	}

	salt := common.RandStringRunes(8)
	pass := common.HashPass(httpUser.Password, salt)
	user := &user.User{
		Login:    httpUser.Login,
		Password: pass,
		// Id is handled below
	}
	id, err := uh.Repo.Add(user)
	if err != nil {
		common.WriteMsg(w, "can't add user", http.StatusInternalServerError)
		return
	}
	user.Id = id

	token, err := uh.SessionManager.CreateToken(user)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", `Bearer `+token)
	w.WriteHeader(http.StatusOK)
}

func (uh UserHandler) LogIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	httpUser := new(HttpUser)
	err := common.ParseReqBody(r.Body, httpUser)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	// Check if user exists

	usr, err := uh.Repo.GetByLoginAndPass(httpUser.Login, httpUser.Password)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get the user by login `%s` and password: %v",
			httpUser.Login, err)
		common.WriteMsg(w, "user not found", http.StatusNotFound)
		return
	}

	token, err := uh.SessionManager.CreateToken(usr)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", `Bearer `+token)
	w.WriteHeader(http.StatusOK)
}

func (uh *UserHandler) LogOut(w http.ResponseWriter, user *user.User) {
}

func (uh *UserHandler) createSessionAndSendToken(w http.ResponseWriter, user *user.User, pass string) {
	token, err := uh.SessionManager.CreateToken(user)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", `Bearer `+token)

	tk := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    user.Login,
		Password: pass,
	}
	common.WriteRespJSON(w, tk)
}
