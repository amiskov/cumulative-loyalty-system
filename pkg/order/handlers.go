package order

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/theplant/luhn"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type iOrderService interface {
	GetUserOrders(ctx context.Context) (orders []*Order, err error)
	AddOrder(ctx context.Context, orderNum string) (*Order, error)
}

type handler struct {
	service iOrderService
}

func NewOrderHandler(s iOrderService) *handler {
	return &handler{
		service: s,
	}
}

func (h handler) GetOrdersList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	orders, err := h.service.GetUserOrders(r.Context())
	if err != nil {
		common.WriteMsg(w, "user orders not found", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	common.WriteRespJSON(w, orders)
}

// Add order to the loyalty system.
func (h handler) AddOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse order number
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed parsing order number from post body")
		common.WriteMsg(w, "order number is not valid", http.StatusBadRequest)
		return
	}

	// Validate order number
	orderNum, err := strconv.Atoi(string(body))
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed converting order number from string")
		common.WriteMsg(w, "order number must be valid", http.StatusBadRequest)
		return
	}
	if !luhn.Valid(orderNum) {
		logger.Log(r.Context()).Errorf("order number `%d` validation failed", orderNum)
		common.WriteMsg(w, "order number is not valid", http.StatusUnprocessableEntity)
		return
	}

	// Add order number to system
	_, err = h.service.AddOrder(r.Context(), strconv.Itoa(orderNum))
	if errors.Is(err, errOrderAlreadyAdded) {
		common.WriteMsg(w, "order is already added", http.StatusOK)
		return
	}
	if errors.Is(err, errOrderExistsForOther) {
		common.WriteMsg(w, "order is already added for another user", http.StatusConflict)
		return
	}
	if err != nil {
		logger.Log(r.Context()).Errorf("failed adding order `%d`, %v", orderNum, err)
		common.WriteMsg(w, "can't add order", http.StatusInternalServerError)
		return
	}

	common.WriteMsg(w, "order has been added", http.StatusAccepted)
}
