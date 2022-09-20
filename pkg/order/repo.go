package order

import (
	"context"
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

func (r *repo) GetOrders(ctx context.Context, userID string) ([]*Order, error) {
	q := `SELECT id, user_id, accrual, status, uploaded_at FROM orders
	      WHERE user_id=$1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()

	orders := []*Order{}
	for rows.Next() {
		o := new(Order)
		if err := rows.Scan(&o.Number, &o.UserID, &o.Accrual, &o.Status, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order row failed: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *repo) AddOrder(ctx context.Context, order *Order) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO orders(id, user_id, accrual, status) VALUES($1, $2, $3, $4)",
		order.Number, order.UserID, order.Accrual, order.Status)
	if err != nil {
		return fmt.Errorf("order/repo: failed inserting order, %w", err)
	}
	return nil
}

func (r *repo) UpdateOrderStatus(userID, orderID, newStatus string, accrual float32) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("order: failed init update order status transaction, %w", err)
	}
	defer tx.Rollback()

	q := `UPDATE orders SET status=$1, accrual=$2 WHERE id=$3`
	_, err = tx.Exec(q, newStatus, accrual, orderID)
	if err != nil {
		return fmt.Errorf("order: failed updating order status, %w", err)
	}

	if newStatus == PROCESSED {
		q := `UPDATE users SET balance = balance + $1 WHERE id = $2 RETURNING balance`
		var newBalance float32
		err = tx.QueryRow(q, accrual, userID).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("order: failed updating user balance, %w", err)
		}
	}

	return tx.Commit()
}

func (r *repo) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	o := &Order{}
	q := `SELECT id, user_id, uploaded_at FROM orders WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, orderID)
	err := row.Scan(&o.Number, &o.UserID, &o.UploadedAt)
	if err != nil {
		return nil, fmt.Errorf("order/repo: can't get order with id `%s`, %w", orderID, err)
	}
	return o, nil
}
