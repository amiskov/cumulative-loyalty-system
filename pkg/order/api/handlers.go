package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/theplant/luhn"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/order"
)

type IOrderRepo interface {
	GetOrders(string) ([]*order.Order, error)
	AddOrder(orderId, userId string) error
}

type OrderHandler struct {
	repo   IOrderRepo
	client *resty.Client
}

func NewOrderHandler(r IOrderRepo, c *resty.Client) *OrderHandler {
	return &OrderHandler{
		repo:   r,
		client: c,
	}
}

func (oh OrderHandler) GetOrdersList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// TODO:
	// get all user orders
	// send queue to accrual service
	// return the right response

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	o := &order.Order{}
	req := oh.client.R().
		SetContext(ctx).
		SetResult(&o)
		// SetResult(&orders)

	resp, err := req.Get("/api/orders" + `/2060100522`)
	if err != nil {
		logger.Log(r.Context()).Errorf("failed sending request to accrual, %v", err)
		common.WriteMsg(w, "failed sending request to accrual", http.StatusInternalServerError)
		return
	}
	fmt.Println("order", o)
	fmt.Printf("GetOrdersList resp: %#v\n", string(resp.Body()))
	respStatus := resp.StatusCode()
	w.WriteHeader(respStatus)
	common.WriteRespJSON(w, nil)
}

func (oh OrderHandler) SendToAccrual(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed parsing order number from post body")
		common.WriteMsg(w, "order number is not valid", http.StatusBadRequest)
	}

	orderNum, err := strconv.Atoi(string(body))
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed converting order number from string")
		common.WriteMsg(w, "order number must be valid", http.StatusBadRequest)
	}
	if !luhn.Valid(orderNum) {
		logger.Log(r.Context()).Errorf("order number `%d` validation failed", orderNum)
		common.WriteMsg(w, "order number is not valid", http.StatusUnprocessableEntity)
		return
	}

	// 200 — номер заказа уже был загружен этим пользователем;
	// 202 — новый номер заказа принят в обработку;
	// 400 — неверный формат запроса;
	// 401 — пользователь не аутентифицирован;
	// 409 — номер заказа уже был загружен другим пользователем;
	// 422 — неверный формат номера заказа;
	// 500 — внутренняя ошибка сервера.

	orderSNum := strconv.Itoa(orderNum)
	userId := "someUser"
	err = oh.repo.AddOrder(orderSNum, userId)
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed add order, %w", err)
		common.WriteMsg(w, "order number is not valid", http.StatusInternalServerError)
		return
	}

	resp, err := oh.client.R().
		Get("/api/orders/" + strconv.Itoa(orderNum))
	if err != nil {
		logger.Log(r.Context()).Errorf("failed sending request to accrual, %v", err)
		common.WriteMsg(w, "failed sending request to accrual", http.StatusInternalServerError)
		return
	}
	respStatus := resp.StatusCode()
	fmt.Printf("resp: %#v\n", resp.RawResponse)
	w.WriteHeader(respStatus)
	w.Write([]byte(`{"hello": "world"}`))
	// common.WriteRespJSON(w, {})
}
