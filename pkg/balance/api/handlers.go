package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/go-resty/resty/v2"
)

type Repo interface{}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	type order struct {
		Number     string    `json:"number"`
		Status     string    `json:"status"`
		Accrual    float32   `json:"accrual"`
		UploadedAt time.Time `json:"uploaded_at"`
	}
	var orders []order

	req := bh.client.R().
		SetContext(ctx).
		SetResult(&orders)

	resp, err := req.Get("/api/orders")
	cancel()

	fmt.Printf("Orders: %#v\n", orders)

	if err != nil {
		logger.Log(r.Context()).Errorf("failed sending request to accrual")
		common.WriteMsg(w, "failed sending request to accrual", http.StatusInternalServerError)
		return
	}
	fmt.Printf("GetOrdersList resp: %#v\n", string(resp.Body()))
	respStatus := resp.StatusCode()
	w.WriteHeader(respStatus)
	common.WriteRespJSON(w, orders)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"GetUserBalance": "test"}`))
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
