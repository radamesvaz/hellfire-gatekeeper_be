package order

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

type OrderRepository struct {
	DB *sql.DB
}

func (r *OrderRepository) GetOrderByID(_ context.Context, idOrder uint64) (oModel.Order, error) {
	fmt.Printf("Getting an order by ID: %v", idOrder)

	order := oModel.Order{}

	err := r.DB.QueryRow(
		`SELECT
			o.id_order,
			o.total_price,
			o.status,
			o.note,
			o.delivery_date,
			o.created_on,
			u.name AS user_name,
			oi.id_order_item,
			oi.id_product,
			p.name AS product_name,
			oi.quantity
		FROM orders o
		INNER JOIN users u ON o.id_user = u.id_user
		INNER JOIN order_items oi ON o.id_order = oi.id_order
		INNER JOIN products p ON oi.id_product = p.id_product
		WHERE o.id_order = ?`,
		idOrder,
	).Scan(
		&order.ID,
		&order.IDRole,
		&order.Name,
		&order.Email,
		&order.Password,
		&order.Phone,
		&order.CreatedOn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("order not found: %v", err)
			return order, errors.NewNotFound(errors.ErrUserNotFound)
		} else {
			fmt.Printf("Could not get the user: %v", err)
			return order, errors.NewNotFound(errors.ErrCouldNotGetTheUser)
		}
	}

	return order, nil
}
