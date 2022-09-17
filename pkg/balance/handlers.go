package balance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type IService interface {
	GetUserBalance(ctx context.Context) (*Balance, error)
	Withdraw(ctx context.Context, w *Withdraw) (float32, error)
	Withdrawals(ctx context.Context) ([]*Withdraw, error)
}

type Handler struct {
	service IService
}

func NewBalanceHandler(s IService) *Handler {
	return &Handler{
		service: s,
	}
}

func (bh *Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	bal, err := bh.service.GetUserBalance(r.Context())
	if err != nil {
		common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return
	}
	common.WriteRespJSON(w, bal)
}

func (bh *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	withdraw := new(Withdraw)
	err := json.NewDecoder(r.Body).Decode(withdraw)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as withdraw: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	newBalance, err := bh.service.Withdraw(r.Context(), withdraw)
	if err != nil {
		common.WriteMsg(w, "failed to withdraw from user balance", http.StatusInternalServerError)
		return
	}

	// TODO: user pgx extension for numeric type: https://github.com/jackc/pgx/wiki/Numeric-and-decimal-support
	msg := fmt.Sprintf(`successfully withdrawn %f from %s; current balance: %f`, withdraw.Sum, withdraw.Order, newBalance)
	common.WriteMsg(w, msg, http.StatusOK)
}

func (bh *Handler) Withdrawalls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	withdrawals, err := bh.service.Withdrawals(r.Context())
	if err != nil {
		common.WriteMsg(w, "can't get user withdrawals", http.StatusInternalServerError)
		return
	}

	common.WriteRespJSON(w, withdrawals)
}
