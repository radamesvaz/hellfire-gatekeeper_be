package order

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

type OrderRepository struct {
	DB *sql.DB
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id uint64) (oModel.OrderResponse, error) {
	query := `
        SELECT 
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
        WHERE o.id_order = ?
    `
	order := oModel.OrderResponse{}
	order.OrderItems = []oModel.OrderItems{}

	rows, err := r.DB.QueryContext(ctx, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return order, errors.NewNotFound(errors.ErrOrderNotFound)
		} else {
			return order, errors.NewInternalServerError(errors.ErrOrderNotFound)
		}
	}
	defer rows.Close()

	firstRow := true

	for rows.Next() {
		var (
			idOrder      uint64
			totalPrice   float64
			status       string
			note         string
			deliveryDate time.Time
			createdOn    time.Time
			userName     string
			idOrderItem  uint64
			idProduct    uint64
			productName  string
			quantity     uint64
		)

		err := rows.Scan(
			&idOrder,
			&totalPrice,
			&status,
			&note,
			&deliveryDate,
			&createdOn,
			&userName,
			&idOrderItem,
			&idProduct,
			&productName,
			&quantity,
		)
		if err != nil {
			fmt.Errorf("Error formating the order id: %v. Error: %v", id, err)
			return oModel.OrderResponse{}, err
		}

		if firstRow {
			order.ID = idOrder
			order.Price = totalPrice
			order.Status = oModel.OrderStatus(status)
			order.Note = note
			order.DeliveryDate = deliveryDate
			order.CreatedOn = createdOn
			order.User = userName
			firstRow = false
		}

		order.OrderItems = append(order.OrderItems, oModel.OrderItems{
			ID:        idOrderItem,
			IdOrder:   idOrder,
			IdProduct: idProduct,
			Name:      productName,
			Quantity:  quantity,
		})
	}

	if err = rows.Err(); err != nil {
		return oModel.OrderResponse{}, err
	}

	return order, nil
}
