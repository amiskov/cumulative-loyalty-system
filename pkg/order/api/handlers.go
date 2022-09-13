package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/theplant/luhn"

	"github.com/amiskov/cumulative-loyalty-system/pkg/common"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/order"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
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

	currentUser, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers.GetOrderList: can't get authorized user, %v", err)
		common.WriteMsg(w, "auth user not found", http.StatusBadRequest)
		return
	}

	currentUserOrders, err := oh.repo.GetOrders(currentUser.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers.GetOrderList: can't get user orders, %v", err)
		common.WriteMsg(w, "user orders not found", http.StatusBadRequest)
		return
	}

	// Orders with information from accrual (not only DB)
	respOrders := []*order.Order{}

	var wg sync.WaitGroup
	for _, o := range currentUserOrders { // orders from DB
		wg.Add(1)
		go func(ord *order.Order) {
			defer wg.Done()

			accrualOrder := &order.Order{} // orders with full info (inc accrual)
			req := oh.client.R().
				SetContext(ctx).
				SetResult(&accrualOrder)
			resp, err := req.Get("/api/orders/" + ord.Number)
			if err != nil {
				logger.Log(r.Context()).Errorf("order/handlers.GetOrdersList: failed sending request to accrual, %v", err)
				common.WriteMsg(w, "failed sending request to accrual", http.StatusInternalServerError)
				return
			}
			respOrders = append(respOrders, accrualOrder)
			fmt.Println("order", ord)
			// respStatus := resp.StatusCode()
			fmt.Printf("GetOrdersList resp: %#v\n", string(resp.Body()))
		}(o)
	}
	wg.Wait()

	w.WriteHeader(http.StatusOK)
	common.WriteRespJSON(w, []*order.Order{})
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
	usr, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: can't get authorized user, %v", err)
		common.WriteMsg(w, "user not found", http.StatusBadRequest)
		return
	}

	err = oh.repo.AddOrder(orderSNum, usr.Id)
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers: failed add order, %w", err)
		common.WriteMsg(w, "order number is not valid", http.StatusInternalServerError)
		return
	}

	resp, err := oh.client.R().
		Get("/api/orders/" + strconv.Itoa(orderNum))
	if err != nil {
		logger.Log(r.Context()).Errorf("order/handlers.SendToAccrual: failed sending request to accrual, %v", err)
		common.WriteMsg(w, "failed sending request to accrual", http.StatusInternalServerError)
		return
	}
	respStatus := resp.StatusCode()
	fmt.Printf("resp: %#v\n", resp.RawResponse)
	w.WriteHeader(respStatus)
	w.Write([]byte(`{"hello": "world"}`))
	// common.WriteRespJSON(w, {})
}
