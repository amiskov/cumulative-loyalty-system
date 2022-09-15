package api

import (
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/go-resty/resty/v2"
)

type Repo interface {
	GetBalance(userId string) (float32, error)
}

type BalanceHandler struct {
	repo   Repo
	client *resty.Client
}

func NewBalanceHandler(r Repo, c *resty.Client) *BalanceHandler {
	return &BalanceHandler{
		repo:   r,
		client: c,
	}
}

func (bh *BalanceHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get current user
	usr, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get authorized user, %v", err)
		common.WriteMsg(w, "user not found", http.StatusUnauthorized)
		return
	}

	bal, err := bh.repo.GetBalance(usr.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get user balance, %v", err)
		common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return
	}

	resp := struct {
		Current   float32 `json:"current"`
		Withdrawn float32 `json:"withdrawn"`
	}{
		Current:   bal,
		Withdrawn: 0,
	}
	common.WriteRespJSON(w, resp)
}

func (bh *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Withdraw": "test"}`))
}

func (b *BalanceHandler) Withdrawalls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Withdrawals": "test"}`))
}
