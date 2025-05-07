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

// GetOrders fetches all orders along with their user and product details.
//
// It joins the orders, users, order_items, and products tables, grouping products under their respective orders.
// A map is used internally to avoid duplicating orders and to associate multiple products with the same order.
//
// Parameters:
// - ctx (context.Context): The context for query execution.
//
// Returns:
// - ([]oModel.OrderResponse): A list of orders with their products.
// - (error): If the query, scan, or row iteration fails.ppear exactly once in the returned slice,
// with all of its products grouped correctly.
func (r *OrderRepository) GetOrders(ctx context.Context) ([]oModel.OrderResponse, error) {
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
    `

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error executing query to fetch orders: %w", err)
	}
	defer rows.Close()

	ordersMap := make(map[uint64]*oModel.OrderResponse)

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
			return nil, fmt.Errorf("error scanning row for order: %w", err)
		}

		if _, exists := ordersMap[idOrder]; !exists {
			ordersMap[idOrder] = &oModel.OrderResponse{
				ID:           idOrder,
				Price:        totalPrice,
				Status:       oModel.OrderStatus(status),
				Note:         note,
				DeliveryDate: deliveryDate,
				CreatedOn:    createdOn,
				User:         userName,
				OrderItems:   []oModel.OrderItems{},
			}
		}

		ordersMap[idOrder].OrderItems = append(ordersMap[idOrder].OrderItems, oModel.OrderItems{
			ID:        idOrderItem,
			IdOrder:   idOrder,
			IdProduct: idProduct,
			Name:      productName,
			Quantity:  quantity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over order rows: %w", err)
	}

	orders := make([]oModel.OrderResponse, 0, len(ordersMap))
	for _, order := range ordersMap {
		orders = append(orders, *order)
	}

	return orders, nil
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
			return order, errors.NewInternalServerError(errors.ErrDatabaseOperation)
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
			return oModel.OrderResponse{}, fmt.Errorf("Error formating the order id: %v. Error: %w", id, err)
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
		return oModel.OrderResponse{}, fmt.Errorf("Error reading the rows for order id: %v, err: %w", id, err)
	}

	return order, nil
}
