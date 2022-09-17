package order

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http/cookiejar"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/go-resty/resty/v2"
)

type IOrderRepo interface {
	GetOrders(userID string) ([]*Order, error)
	GetOrder(string) (*Order, error)
	AddOrder(*Order) error
}

type service struct {
	repo   IOrderRepo
	client *resty.Client
}

func NewService(r IOrderRepo, accrualAddr string) *service {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// TODO: Is it fine to use `log.Fatal` here?
		log.Fatalln("can't create cookie jar")
	}
	httpClient := resty.New().SetBaseURL(accrualAddr).SetCookieJar(jar)

	return &service{
		repo:   r,
		client: httpClient,
	}
}

var (
	errOrderAlreadyAdded   = errors.New("order already added")
	errOrderExistsForOther = errors.New("order already exists for the other user")
)

func (s *service) AddOrder(ctx context.Context, orderNum string) (*Order, error) {
	// Get current user
	usr, err := sessions.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return nil, errOrderAlreadyAdded
	}

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

	// Order not found by the given number, check the accrual system.
	resp, err := s.client.R().Get("/api/orders/" + orderNum)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed sending request to accrual, %v", err)
		return nil, err
	}

	// {"order":"2060100522","status":"PROCESSED","accrual":729.98}
	httpOrder := struct {
		Order   string
		Status  string
		Accrual float32
	}{}
	jsonErr := json.Unmarshal(resp.Body(), &httpOrder)
	if jsonErr != nil {
		logger.Log(ctx).Errorf("order: failed parsing response from accrual, %w", jsonErr)
		return nil, jsonErr
	}

	newOrder := &Order{
		Number:  orderNum,
		UserID:  usr.ID,
		Accrual: httpOrder.Accrual,
		// TODO: Probably just add as 'NEW' without even checking accrual in this method
		// and later check for PROCESSED/INVALID separately?
		Status: httpOrder.Status,
	}
	err = s.repo.AddOrder(newOrder)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed add order, %w", err)
		return nil, err
	}

	return newOrder, nil
}

func (s *service) GetUserOrders(ctx context.Context) (orders []*Order, err error) {
	usr, err := sessions.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return
	}
	orders, err = s.repo.GetOrders(usr.ID)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get user orders, %v", err)
		return
	}
	return
}
