package api

import (
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

	ISessionManager interface {
		Create(userId string) (string, error)
		Check(sessionId, userId string) error
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

	// Create user session

	sessID, err := uh.SessionManager.Create(user.Id)
	if err != nil {
		common.WriteMsg(w, "can't create session", http.StatusInternalServerError)
		return
	}
	cookie := &http.Cookie{
		Name:    "session_id",
		Value:   sessID,
		Expires: time.Now().Add(90 * 24 * time.Hour),
		Path:    "/",
	}
	http.SetCookie(w, cookie)

	// Set headers and send response

	w.Header().Set("Authorization", "With Cookie")
	w.WriteHeader(http.StatusOK)
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

	// Check if user exists

	user, err := uh.Repo.GetByLoginAndPass(httpUser.Login, httpUser.Password)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get the user by login `%s` and password: %v",
			httpUser.Login, err)
		common.WriteMsg(w, "user not found", http.StatusNotFound)
		return
	}

	// Check user session

	// TODO: use auth middleware and session from request

	sessionCookie, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		logger.Log(r.Context()).Errorf("bad cookie for user `%s`. %v", httpUser.Login, err)
		return
	}
	fmt.Printf("cookie: %#v\n", sessionCookie)
	uh.SessionManager.Check(sessionCookie.Value, user.Id)

	w.WriteHeader(http.StatusOK)
}

func (uh *UserHandler) LogOut(w http.ResponseWriter, user *user.User) {
	// TODO: remove from sessions table
	cookie := http.Cookie{
		Name:    "session_id",
		Expires: time.Now().AddDate(0, 0, -1),
		Path:    "/",
	}
	http.SetCookie(w, &cookie)
}
