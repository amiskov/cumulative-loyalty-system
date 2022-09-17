package order

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

func (or *repo) GetOrders(userId string) ([]*Order, error) {
	q := `SELECT id, user_id, accrual, status, uploaded_at FROM orders
	      WHERE user_id=$1 ORDER BY uploaded_at DESC`
	rows, err := or.db.Query(q,
		userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*Order{}
	for rows.Next() {
		o := new(Order)
		if err := rows.Scan(&o.Number, &o.UserId, &o.Accrual, &o.Status, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order row failed: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (or *repo) AddOrder(order *Order) error {
	_, err := or.db.Exec("INSERT INTO orders(id, user_id, accrual, status) VALUES($1, $2, $3, $4)",
		order.Number, order.UserId, order.Accrual, order.Status)
	if err != nil {
		return fmt.Errorf("order/repo: failed inserting order, %w", err)
	}

	// TODO: Probably, this should be a separated process.
	if order.Status == "PROCESSED" {
		q := `UPDATE users SET balance = balance + $1 WHERE id = $2 RETURNING balance`
		var newBalance float32
		err := or.db.QueryRow(q, order.Accrual, order.UserId).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("order/repo: failed updating balance, %w", err)
		}
	}

	return nil
}

func (or *repo) GetOrder(orderId string) (*Order, error) {
	o := &Order{}
	q := `SELECT id, user_id, uploaded_at FROM orders WHERE id = $1`
	row := or.db.QueryRow(q, orderId)
	err := row.Scan(&o.Number, &o.UserId, &o.UploadedAt)
	if err != nil {
		return nil, fmt.Errorf("order/repo: can't get order with id `%s`, %w", orderId, err)
	}
	return o, nil
}
