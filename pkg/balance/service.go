package balance

import (
	"context"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
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

func (s *service) Withdrawals(ctx context.Context) ([]*Withdraw, error) {
	// Get current user
	usr, err := sessions.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get authorized user, %v", err)
		return nil, err
	}

	withdrawals, err := s.repo.GetWithdrawals(usr.Id)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get user withdrawals, %v", err)
		return nil, err
	}

	return withdrawals, nil
}

func (s *service) Withdraw(ctx context.Context, w *Withdraw) (newBalance float32, err error) {
	// Get current user
	usr, err := sessions.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get authorized user, %v", err)
		return 0, err
	}

	// TODO: make checking balance and updating balance in one query

	// Get user balance
	bal, err := s.repo.GetBalance(usr.Id)
	if err != nil {
		logger.Log(ctx).Errorf("balance: can't get user balance, %v", err)
		// common.WriteMsg(w, "can't get user balance", http.StatusBadRequest)
		return 0, err
	}

	if w.Sum > bal.Current {
		// common.WriteMsg(w, "not enough balance", http.StatusPaymentRequired)
		return 0, err
	}

	newBalance, err = s.repo.WithdrawFromUserBalance(usr.Id, w.Order, w.Sum)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: withdraw failed, %v", err)
		return 0, err
	}

	return newBalance, nil
}

func (s *service) GetUserBalance(ctx context.Context) (*Balance, error) {
	usr, err := sessions.GetAuthUser(ctx)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get authorized user, %v", err)
		return nil, err
	}

	// Get user balance
	bal, err := s.repo.GetBalance(usr.Id)
	if err != nil {
		logger.Log(ctx).Errorf("balance/handlers: can't get user balance, %v", err)
		return nil, err
	}

	return bal, nil
}
