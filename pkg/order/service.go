package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
	"github.com/go-resty/resty/v2"
)

type IOrderRepo interface {
	GetOrders(userID string) ([]*Order, error)
	GetOrder(string) (*Order, error)
	AddOrder(*Order) error
	UpdateOrderStatus(userID, orderID, newStatus string, accrual float32) error
}

type service struct {
	repo   IOrderRepo
	client *resty.Client
}

func NewService(r IOrderRepo, accrualAddr string) (*service, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("can't create cookie jar, %w", err)
	}
	httpClient := resty.New().SetBaseURL(accrualAddr).SetCookieJar(jar)

	return &service{
		repo:   r,
		client: httpClient,
	}, nil
}

var (
	errOrderAlreadyAdded   = errors.New("order already added")
	errOrderExistsForOther = errors.New("order already exists for the other user")
)

func (s *service) AddOrder(ctx context.Context, usr *user.User, orderNum string) (*Order, error) {
	o, orderErr := s.repo.GetOrder(orderNum)

	// Order is already added, just sent OK status
	if o != nil && orderErr == nil && o.UserID == usr.ID {
		return nil, errOrderAlreadyAdded
	}

	// Order exists but for the other user
	if o != nil && orderErr == nil && o.UserID != usr.ID {
		logger.Log(ctx).Errorf("order: user `%s` tries to get the order of user `%s`, %v",
			usr.ID, o.UserID, orderErr)
		return nil, errOrderExistsForOther
	}

	// Something unknown happened
	if o != nil && orderErr != nil {
		logger.Log(ctx).Errorf("order/handlers: failed getting order`, %v", orderErr)
		return nil, orderErr
	}

	newOrder := &Order{
		Number:  orderNum,
		UserID:  usr.ID,
		Accrual: 0,
		Status:  NEW,
	}
	err := s.repo.AddOrder(newOrder)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed add order, %w", err)
		return nil, err
	}

	// TODO: add limitation (if tried N times with no success then stop)
	go func(ctx context.Context, usr *user.User, orderNum string) {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			// Run query each 3 seconds and update order status.
			// If order status is `INVALID` or `PROCESSED`, then stop the ticker.
			s.updateOrderStatus(ctx, ticker, usr, orderNum)
		}
	}(ctx, usr, orderNum)

	return newOrder, nil
}

func (s *service) GetUserOrders(ctx context.Context, usr *user.User) (orders []*Order, err error) {
	orders, err = s.repo.GetOrders(usr.ID)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get user orders, %v", err)
		return
	}
	return
}

func (s *service) updateOrderStatus(ctx context.Context, ticker *time.Ticker, usr *user.User, orderNum string) {
	log.Println("Start order status updating...")
	resp, err := s.client.R().Get("/api/orders/" + orderNum)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed sending request to accrual, %v", err)
		return
	}

	// Accrual response format: `{"order":"2060100522","status":"PROCESSED","accrual":729.98}`
	httpOrder := struct {
		Order   string
		Status  string
		Accrual float32
	}{}
	jsonErr := json.Unmarshal(resp.Body(), &httpOrder)
	if jsonErr != nil {
		logger.Log(ctx).Errorf("order: failed parsing response from accrual, %w", jsonErr)
		return
	}

	if err := s.repo.UpdateOrderStatus(usr.ID, orderNum, httpOrder.Status, httpOrder.Accrual); err != nil {
		logger.Log(ctx).Errorf("order: failed updating order, %w", err)
		return
	}

	if httpOrder.Status == INVALID || httpOrder.Status == PROCESSED {
		ticker.Stop()
		return
	}
}
