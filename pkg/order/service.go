package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/accrual"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
)

type iOrderRepo interface {
	GetOrders(ctx context.Context, userID string) ([]*Order, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	AddOrder(ctx context.Context, o *Order) error
	UpdateOrderStatus(userID, orderID, newStatus string, accrual float32) error
}

type iAccrualClient interface {
	GetOrderAccrual(ctx context.Context, orderNum string) (*accrual.OrderAccrual, error)
	MaxAttempts() int
	Interval() time.Duration
}

type service struct {
	repo          iOrderRepo
	accrualClient iAccrualClient
}

func NewService(r iOrderRepo, accSys iAccrualClient) *service {
	return &service{
		repo:          r,
		accrualClient: accSys,
	}
}

var (
	errOrderAlreadyAdded   = errors.New("order already added")
	errOrderExistsForOther = errors.New("order already exists for the other user")
)

func (s *service) AddOrder(ctx context.Context, orderNum string) (*Order, error) {
	userID, err := session.GetAuthUserID(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return nil, err
	}

	// We want `ord` to be `nil` and `ordErr` to be `sql.ErrNoRows` meaning order with `orderNum` not exists
	ord, ordErr := s.repo.GetOrder(ctx, orderNum)

	// Something bad happened
	if !errors.Is(ordErr, sql.ErrNoRows) && ord == nil {
		logger.Log(ctx).Errorf("order: failed getting order`, %v", ordErr)
		return nil, ordErr
	}

	// Order exists for the current user
	if !errors.Is(ordErr, sql.ErrNoRows) && ord.UserID == userID {
		return nil, errOrderAlreadyAdded
	}

	// Order exists for another user
	if !errors.Is(ordErr, sql.ErrNoRows) && ord.UserID != userID {
		logger.Log(ctx).Errorf("order: user `%s` tries to get the order of user `%s`, %v", userID, ord.UserID, ordErr)
		return nil, errOrderExistsForOther
	}

	newOrder := &Order{
		Number:  orderNum,
		UserID:  userID,
		Accrual: 0,
		Status:  NEW,
	}
	if err := s.repo.AddOrder(ctx, newOrder); err != nil {
		logger.Log(ctx).Errorf("order: failed add order, %w", err)
		return nil, err
	}

	go s.updateOrderStatus(ctx, userID, orderNum)

	return newOrder, nil
}

func (s *service) updateOrderStatus(ctx context.Context, userID, orderNum string) {
	done := make(chan struct{})
	attempts := 0
	maxAttempts := s.accrualClient.MaxAttempts()
	pause := s.accrualClient.Interval()

	for {
		select {
		case <-done:
			return
		default:
			attempts++
			if attempts >= maxAttempts {
				logger.Log(ctx).Errorf("order: can't get order accrual, max attempts exceeded")
				done <- struct{}{}
			}

			orderAccrual, err := s.accrualClient.GetOrderAccrual(ctx, orderNum)
			if err != nil {
				logger.Log(ctx).Errorf("order: failed getting order accrual, %v", err)
				time.Sleep(pause)
				continue // try once again
			}

			if err := s.repo.UpdateOrderStatus(userID, orderNum, orderAccrual.Status, orderAccrual.Accrual); err != nil {
				logger.Log(ctx).Errorf("order: failed updating order status, %w", err)
				time.Sleep(pause)
				continue // try once again
			}

			if orderAccrual.Status == INVALID || orderAccrual.Status == PROCESSED {
				done <- struct{}{}
			}

			time.Sleep(pause)
		}
	}
}

func (s *service) GetUserOrders(ctx context.Context) (orders []*Order, err error) {
	userID, err := session.GetAuthUserID(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return
	}

	orders, err = s.repo.GetOrders(ctx, userID)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get user orders, %v", err)
		return
	}
	return
}
