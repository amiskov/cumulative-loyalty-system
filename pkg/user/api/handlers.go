package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	SessionManager interface {
		CreateToken(*user.User) (string, error)
		CleanupUserSessions(userId string) error
	}

	UserHandler struct {
		Repo           UserRepo
		SessionManager SessionManager
	}

	HttpUser struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
)

func NewUserHandler(r UserRepo, sm SessionManager) *UserHandler {
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

	sessID := common.RandStringRunes(32)
	// TODO: save session to db, check it on login
	// _, err = uh.Repo.Exec("INSERT INTO sessions(id, user_id) VALUES(?, ?)", sessID, user.Id)
	// if err != nil {
	// 	return err
	// }

	cookie := &http.Cookie{
		Name:    "session_id",
		Value:   sessID,
		Expires: time.Now().Add(90 * 24 * time.Hour),
		Path:    "/",
	}
	http.SetCookie(w, cookie)

	w.Header().Set("Authorization", "With Cookie")
	w.WriteHeader(http.StatusOK)
	// uh.sendToken(w, user)
	resp := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    httpUser.Login,
		Password: httpUser.Password,
	}
	common.WriteRespJSON(w, resp)
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

	user, err := uh.Repo.GetByLoginAndPass(httpUser.Login, httpUser.Password)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get the user by login `%s` and password: %v",
			httpUser.Login, err)
		common.WriteMsg(w, "user not found", http.StatusNotFound)
		return
	}

	// Remove expired user session if there are any
	if err := uh.SessionManager.CleanupUserSessions(user.Id); err != nil {
		logger.Log(r.Context()).Errorf("user/handlers: can't cleanup sessions for user `%s`, %v",
			httpUser.Login, err)
		common.WriteMsg(w, "failed managing user sessions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	uh.sendToken(w, user)
}

func (uh *UserHandler) sendToken(w http.ResponseWriter, user *user.User) {
	token, err := uh.SessionManager.CreateToken(user)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}

	tk := struct {
		Token string `json:"token"`
	}{token}
	common.WriteRespJSON(w, tk)
}
