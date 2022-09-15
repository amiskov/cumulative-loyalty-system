package api

import (
	"fmt"
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/balance"
	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/go-resty/resty/v2"
)

type Repo interface {
	GetBalance(userId string) (*balance.Balance, error)
	WithdrawFromUserBalance(userId, orderId string, sum float32) (float32, error)
	GetWithdrawals(userId string) ([]*balance.Withdraw, error)
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

	// Get user balance
	bal, err := bh.repo.GetBalance(usr.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get user balance, %v", err)
		common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return
	}
	common.WriteRespJSON(w, bal)
}

func (bh *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	withdraw := new(balance.Withdraw)
	err := common.ParseReqBody(r.Body, withdraw)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as withdraw: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	// Get current user
	usr, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get authorized user, %v", err)
		common.WriteMsg(w, "user not found", http.StatusUnauthorized)
		return
	}

	// TODO: make checking balance and updating balance in one query

	// Get user balance
	bal, err := bh.repo.GetBalance(usr.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get user balance, %v", err)
		common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return
	}

	if withdraw.Sum > bal.Current {
		common.WriteMsg(w, "not enough balance", http.StatusPaymentRequired)
		return
	}

	newBalance, err := bh.repo.WithdrawFromUserBalance(usr.Id, withdraw.Order, withdraw.Sum)
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: withdraw failed, %v", err)
		common.WriteMsg(w, "failed to withdraw from user balance", http.StatusInternalServerError)
		return
	}
	// TODO: user pgx extension for numeric type: https://github.com/jackc/pgx/wiki/Numeric-and-decimal-support
	msg := fmt.Sprintf(`successfully withdrawn %f from %s; current balance: %f`,
		withdraw.Sum, withdraw.Order, newBalance)
	common.WriteMsg(w, msg, http.StatusOK)
}

func (b *BalanceHandler) Withdrawalls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get current user
	usr, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get authorized user, %v", err)
		common.WriteMsg(w, "user not found", http.StatusUnauthorized)
		return
	}

	withdrawals, err := b.repo.GetWithdrawals(usr.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("balance/handlers: can't get user withdrawals, %v", err)
		common.WriteMsg(w, "can't get user withdrawals", http.StatusInternalServerError)
		return
	}

	common.WriteRespJSON(w, withdrawals)
}
