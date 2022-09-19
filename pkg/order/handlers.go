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
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type IOrderService interface {
	GetUserOrders(ctx context.Context, usr *user.User) (orders []*Order, err error)
	AddOrder(ctx context.Context, usr *user.User, orderNum string) (*Order, error)
}

type Handler struct {
	service IOrderService
}

func NewOrderHandler(s IOrderService) *Handler {
	return &Handler{
		service: s,
	}
}

func (oh Handler) GetOrdersList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	usr, err := session.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("order: can't get authorized user, %v", err)
		common.WriteMsg(w, "authorization required", http.StatusUnauthorized)
		return
	}

	orders, err := oh.service.GetUserOrders(r.Context(), usr)
	if err != nil {
		common.WriteMsg(w, "user orders not found", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	common.WriteRespJSON(w, orders)
}

// Add order to the loyalty system.
//
// Possible status code:
// - 200 — номер заказа уже был загружен этим пользователем;
// - 202 — новый номер заказа принят в обработку;
// - 400 — неверный формат запроса;
// - 401 — пользователь не аутентифицирован;
// - 409 — номер заказа уже был загружен другим пользователем;
// - 422 — неверный формат номера заказа;
// - 500 — внутренняя ошибка сервера.
func (oh Handler) AddOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	usr, err := session.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("order: can't get authorized user, %v", err)
		common.WriteMsg(w, "authorization required", http.StatusUnauthorized)
		return
	}

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
	validOrderNum := strconv.Itoa(orderNum)

	_, err = oh.service.AddOrder(r.Context(), usr, validOrderNum)
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
