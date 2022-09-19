package order

import (
	"context"
	"errors"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/accrual"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
)

type IOrderRepo interface {
	GetOrders(userID string) ([]*Order, error)
	GetOrder(string) (*Order, error)
	AddOrder(*Order) error
	UpdateOrderStatus(userID, orderID, newStatus string, accrual float32) error
}

type IAccrualSystem interface {
	GetOrderAccrual(ctx context.Context, orderNum string) (*accrual.OrderAccrual, error)
}

type service struct {
	repo          IOrderRepo
	accrualSystem IAccrualSystem
}

func NewService(r IOrderRepo, accSys IAccrualSystem) *service {
	return &service{
		repo:          r,
		accrualSystem: accSys,
	}
}

var (
	errOrderAlreadyAdded   = errors.New("order already added")
	errOrderExistsForOther = errors.New("order already exists for the other user")
)

func (s *service) AddOrder(ctx context.Context, orderNum string) (*Order, error) {
	usr, err := session.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return nil, err
	}

	ord, orderErr := s.repo.GetOrder(orderNum)

	// Order is already added, just sent OK status
	if ord != nil && orderErr == nil && ord.UserID == usr.ID {
		return nil, errOrderAlreadyAdded
	}

	// Order exists but for the other user
	if ord != nil && orderErr == nil && ord.UserID != usr.ID {
		logger.Log(ctx).Errorf("order: user `%s` tries to get the order of user `%s`, %v",
			usr.ID, ord.UserID, orderErr)
		return nil, errOrderExistsForOther
	}

	// Something unknown happened
	if ord != nil && orderErr != nil {
		logger.Log(ctx).Errorf("order/handlers: failed getting order`, %v", orderErr)
		return nil, orderErr
	}

	newOrder := &Order{
		Number:  orderNum,
		UserID:  usr.ID,
		Accrual: 0,
		Status:  NEW,
	}
	if err := s.repo.AddOrder(newOrder); err != nil {
		logger.Log(ctx).Errorf("order: failed add order, %w", err)
		return nil, err
	}

	// TODO: add limitation (if tried N times with no success then stop)
	go func(ctx context.Context, orderNum string) {
		// Every 3 seconds check and accrual system and update the order status.
		// If order status is `INVALID` or `PROCESSED`, then stop the ticker.
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			// TODO: use either channel or cancel via context
			s.updateOrderStatus(ctx, ticker, orderNum)
		}
	}(ctx, orderNum)

	return newOrder, nil
}

func (s *service) GetUserOrders(ctx context.Context) (orders []*Order, err error) {
	usr, err := session.GetAuthUser(ctx)
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

func (s *service) updateOrderStatus(ctx context.Context, ticker *time.Ticker, orderNum string) {
	usr, err := session.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("order: can't get authorized user, %v", err)
		return
	}

	orderAccrual, err := s.accrualSystem.GetOrderAccrual(ctx, orderNum)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed getting order accrual, %v", err)
		return
	}

	if err := s.repo.UpdateOrderStatus(usr.ID, orderNum, orderAccrual.Status, orderAccrual.Accrual); err != nil {
		logger.Log(ctx).Errorf("order: failed updating order, %w", err)
		return
	}

	if orderAccrual.Status == INVALID || orderAccrual.Status == PROCESSED {
		ticker.Stop()
		return
	}
}
