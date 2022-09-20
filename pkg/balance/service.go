package balance

import (
	"context"
	"fmt"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
)

type iBalanceRepo interface {
	GetBalance(userID string) (*Balance, error)
	WithdrawFromUserBalance(userID, orderID string, sum float32) (float32, error)
	GetWithdrawals(userID string) ([]*Withdraw, error)
}

type service struct {
	repo iBalanceRepo
}

func NewService(r iBalanceRepo) *service {
	return &service{
		repo: r,
	}
}

func (s *service) Withdrawals(ctx context.Context) ([]*Withdraw, error) {
	userID, err := session.GetAuthUserID(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get authorized user, %v", err)
		return nil, err
	}

	withdrawals, err := s.repo.GetWithdrawals(userID)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get user withdrawals, %v", err)
		return nil, err
	}

	return withdrawals, nil
}

func (s *service) Withdraw(ctx context.Context, w *Withdraw) (float32, error) {
	userID, err := session.GetAuthUserID(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get authorized user, %v", err)
		return 0, err
	}

	bal, err := s.repo.GetBalance(userID)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get user balance, %v", err)
		return 0, err
	}

	if w.Sum > bal.Current {
		msg := fmt.Sprintf("balance: can't withdraw sum `%f` from balance `%f`", w.Sum, bal.Current)
		logger.Log(ctx).Error(msg)
		return bal.Current, fmt.Errorf(msg)
	}

	newBalance, err := s.repo.WithdrawFromUserBalance(userID, w.Order, w.Sum)
	if err != nil {
		logger.Log(ctx).Errorf("balance: withdraw failed, %v", err)
		return bal.Current, err
	}

	return newBalance, nil
}

func (s *service) GetUserBalance(ctx context.Context) (*Balance, error) {
	userID, err := session.GetAuthUserID(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get authorized user, %v", err)
		return nil, err
	}

	bal, err := s.repo.GetBalance(userID)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get user balance, %v", err)
		return nil, err
	}
	return bal, nil
}
