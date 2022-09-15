package order

import (
	"database/sql"
	"fmt"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{
		db: db,
	}
}

func (or *OrderRepo) GetOrders(userId string) ([]*Order, error) {
	rows, err := or.db.Query("SELECT id, uploaded_at FROM orders WHERE user_id=$1 ORDER BY uploaded_at DESC", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbUserOrders := []*Order{}
	for rows.Next() {
		o := new(Order)
		if err := rows.Scan(&o.Number, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order row failed: %w", err)
		}
		dbUserOrders = append(dbUserOrders, o)
	}
	return dbUserOrders, nil
}

func (or *OrderRepo) AddOrder(order *Order) error {
	_, err := or.db.Exec("INSERT INTO orders(id, user_id, accrual, status) VALUES($1, $2, $3, $4)",
		order.Number, order.UserId, order.Accrual, order.Status)
	if err != nil {
		return fmt.Errorf("order/repo: failed inserting order, %w", err)
	}
	return nil
}

func (or *OrderRepo) GetOrder(orderId string) (*Order, error) {
	o := &Order{}
	q := `SELECT id, user_id, uploaded_at FROM orders WHERE id = $1`
	row := or.db.QueryRow(q, orderId)
	err := row.Scan(&o.Number, &o.UserId, &o.UploadedAt)
	if err != nil {
		return nil, fmt.Errorf("order/repo: can't get order with id `%s`, %w", orderId, err)
	}
	return o, nil
}
