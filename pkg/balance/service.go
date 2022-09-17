package balance

import (
	"context"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

type IBalanceRepo interface {
	GetBalance(userID string) (*Balance, error)
	WithdrawFromUserBalance(userID, orderID string, sum float32) (float32, error)
	GetWithdrawals(userID string) ([]*Withdraw, error)
}

type service struct {
	repo IBalanceRepo
}

func NewService(r IBalanceRepo) *service {
	return &service{
		repo: r,
	}
}

func (s *service) Withdrawals(ctx context.Context, usr *user.User) ([]*Withdraw, error) {
	withdrawals, err := s.repo.GetWithdrawals(usr.ID)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get user withdrawals, %v", err)
		return nil, err
	}

	return withdrawals, nil
}

func (s *service) Withdraw(ctx context.Context, usr *user.User, w *Withdraw) (newBalance float32, err error) {
	// TODO: make checking balance and updating balance in one query

	// Get user balance
	bal, err := s.repo.GetBalance(usr.ID)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get user balance, %v", err)
		// common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return 0, err
	}

	if w.Sum > bal.Current {
		// common.WriteMsg(w, "not enough balance", http.StatusPaymentRequired)
		return 0, err
	}

	newBalance, err = s.repo.WithdrawFromUserBalance(usr.ID, w.Order, w.Sum)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: withdraw failed, %v", err)
		return 0, err
	}

	return newBalance, nil
}

func (s *service) GetUserBalance(ctx context.Context, usr *user.User) (*Balance, error) {
	bal, err := s.repo.GetBalance(usr.ID)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get user balance, %v", err)
		return nil, err
	}
	return bal, nil
}
