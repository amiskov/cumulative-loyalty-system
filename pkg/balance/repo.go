package balance

import (
	"database/sql"
	"fmt"
)

type repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *repo {
	return &repo{
		db: db,
	}
}

func (r *repo) GetBalance(userID string) (*Balance, error) {
	bal := &Balance{}
	row := r.db.QueryRow("SELECT balance, withdrawn FROM users where id=$1", userID)
	if err := row.Scan(&bal.Current, &bal.Withdrawn); err != nil {
		return bal, fmt.Errorf("balance/repo: row scan failed: %w", err)
	}
	return bal, nil
}

func (r *repo) WithdrawFromUserBalance(userID, orderID string, sumToWithdraw float32) (float32, error) {
	// TODO: this should be transaction

	q := `UPDATE users SET balance=balance-$1, withdrawn=withdrawn+$1
		    WHERE id = $2 RETURNING balance`
	var newBalance float32
	err := r.db.QueryRow(q, sumToWithdraw, userID).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("order/repo: failed withdraw from user balance, %w", err)
	}

	// Add record to withdrawals table
	_, err = r.db.Exec(`INSERT INTO withdrawals(user_id, order_id, sum) VALUES($1, $2, $3)`,
		userID, orderID, sumToWithdraw)
	if err != nil {
		return 0, fmt.Errorf("order/repo: failed inserting to `withdrawals` table, %w", err)
	}

	return newBalance, nil
}

func (r *repo) GetWithdrawals(userID string) ([]*Withdraw, error) {
	q := `SELECT order_id, sum, processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC`
	rows, err := r.db.Query(q, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	withdrawals := []*Withdraw{}
	for rows.Next() {
		w := new(Withdraw)
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan withdraw row failed: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, nil
}
