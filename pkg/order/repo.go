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

func (or *OrderRepo) AddOrder(orderId, userId string) error {
	_, err := or.db.Exec("INSERT INTO orders(id, user_id) VALUES($1, $2)", orderId, userId)
	if err != nil {
		return fmt.Errorf("order/repo: failed inserting order, %w", err)
	}
	return nil
}
