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
// - (error): If the query, scan, or row iteration fails.
func (r *OrderRepository) GetOrders(ctx context.Context) ([]oModel.OrderResponse, error) {
	query := `
        SELECT 
            o.id_order, 
            o.id_user,
            o.total_price, 
            o.status, 
            o.note, 
            o.delivery_date, 
            o.paid,
            o.created_on, 
            u.name AS user_name, 
            u.phone,
            oi.id_order_item, 
            oi.id_product, 
            p.name AS product_name, 
            oi.quantity
        FROM orders o
        INNER JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        INNER JOIN products p ON oi.id_product = p.id_product
        ORDER BY o.id_order
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
			idUser       uint64
			totalPrice   float64
			status       string
			note         string
			deliveryDate time.Time
			paid         bool
			createdOn    time.Time
			userName     string
			phone        string
			idOrderItem  uint64
			idProduct    uint64
			productName  string
			quantity     uint64
		)

		err := rows.Scan(
			&idOrder,
			&idUser,
			&totalPrice,
			&status,
			&note,
			&deliveryDate,
			&paid,
			&createdOn,
			&userName,
			&phone,
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
				IdUser:       idUser,
				Price:        totalPrice,
				Status:       oModel.OrderStatus(status),
				Note:         note,
				DeliveryDate: deliveryDate,
				Paid:         paid,
				CreatedOn:    createdOn,
				User:         userName,
				Phone:        phone,
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
            o.id_user,
            o.total_price, 
            o.status, 
            o.note, 
            o.delivery_date, 
            o.paid,
            o.created_on, 
            u.name AS user_name, 
            u.phone,
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
		// Check if the error is specifically "no rows found"
		if err == sql.ErrNoRows {
			return order, errors.NewNotFound(errors.ErrOrderNotFound)
		}
		return order, errors.NewInternalServerError(errors.ErrDatabaseOperation)
	}
	defer rows.Close()

	firstRow := true
	hasRows := false

	for rows.Next() {
		hasRows = true
		var (
			idOrder      uint64
			idUser       uint64
			totalPrice   float64
			status       string
			note         string
			deliveryDate time.Time
			paid         bool
			createdOn    time.Time
			userName     string
			phone        string
			idOrderItem  uint64
			idProduct    uint64
			productName  string
			quantity     uint64
		)

		err := rows.Scan(
			&idOrder,
			&idUser,
			&totalPrice,
			&status,
			&note,
			&deliveryDate,
			&paid,
			&createdOn,
			&userName,
			&phone,
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
			order.IdUser = idUser
			order.Price = totalPrice
			order.Status = oModel.OrderStatus(status)
			order.Note = note
			order.DeliveryDate = deliveryDate
			order.Paid = paid
			order.CreatedOn = createdOn
			order.User = userName
			order.Phone = phone
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

	// Check if no rows were found
	if !hasRows {
		return order, errors.NewNotFound(errors.ErrOrderNotFound)
	}

	return order, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (id uint64, err error) {
	return r.createOrderTx(ctx, tx, order)
}

func (r *OrderRepository) CreateOrderItems(ctx context.Context, tx *sql.Tx, items []oModel.OrderItemRequest) error {
	return r.createOrderItemTx(ctx, tx, items)
}

func (r *OrderRepository) CreateOrderOrchestrator(ctx context.Context, order oModel.CreateFullOrder) (uint64, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("OrderOrchestrator: error starting transaction: %w", err)
	}

	orderRequest := oModel.CreateOrderRequest{
		IdUser:       order.IdUser,
		DeliveryDate: order.DeliveryDate,
		Note:         order.Note,
		Price:        order.Price,
		Status:       order.Status,
		Paid:         order.Paid,
	}
	orderID, err := r.CreateOrder(ctx, tx, orderRequest)

	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("OrderOrchestrator: Error creating the order: %w", err)
	}

	orderItems := []oModel.OrderItemRequest{}

	for _, items := range order.OrderItems {
		item := oModel.OrderItemRequest{
			IdOrder:   orderID,
			IdProduct: items.IdProduct,
			Quantity:  items.Quantity,
		}
		orderItems = append(orderItems, item)
	}

	err = r.CreateOrderItems(ctx, tx, orderItems)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("OrderOrchestrator: error inserting item: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("OrderOrchestrator: error committing transaction: %w", err)
	}

	return orderID, nil
}

func (r *OrderRepository) createOrderTx(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (id uint64, err error) {
	fmt.Printf("Creating order for user: %v", order.IdUser)
	exec := execerFrom(tx, r.DB)
	query := `INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES (?, ?, ?, ?, ?, ?)`

	result, err := exec.ExecContext(
		ctx,
		query,
		order.IdUser,
		order.Price,
		order.Status,
		order.Note,
		order.DeliveryDate,
		order.Paid,
	)

	if err != nil {
		fmt.Printf("Error creating the order: %v", err)
		return 0, errors.NewInternalServerError(errors.ErrCreatingOrder)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Error getting the last insert ID: %v", err)
		return 0, errors.NewInternalServerError(errors.ErrGettingTheOrderID)
	}

	return uint64(insertedID), nil
}

func (r *OrderRepository) createOrderItemTx(ctx context.Context, tx *sql.Tx, items []oModel.OrderItemRequest) error {
	fmt.Printf("creating items for order: %v", items[0].IdOrder)
	exec := execerFrom(tx, r.DB)
	query := `INSERT INTO order_items (id_order, id_product, quantity) VALUES (?, ?, ?)`

	for _, item := range items {
		_, err := exec.ExecContext(ctx, query, item.IdOrder, item.IdProduct, item.Quantity)
		if err != nil {
			fmt.Printf("error inserting item (orderID: %d, productID: %d): %v", item.IdOrder, item.IdProduct, err)
			return errors.NewInternalServerError(errors.ErrCreatingOrderItem)
		}
	}

	return nil
}

func execerFrom(tx *sql.Tx, db *sql.DB) interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
} {
	if tx != nil {
		return tx
	}
	return db
}

// CreateOrderHistory creates a new order history record
func (r *OrderRepository) CreateOrderHistory(_ context.Context, order oModel.OrderHistory) error {
	fmt.Printf("Creating order history: %v", order.IDOrder)

	result, err := r.DB.Exec(
		`INSERT INTO orders_history (
		id_order, 
		id_user, 
		status, 
		total_price, 
		note,
		delivery_date,
		paid,
		modified_by, 
		action
		) 
		VALUES (
		?,
		?, 
		?, 
		?, 
		?,
		?,
		?,
		?, 
		?)`,
		order.IDOrder,
		order.IdUser,
		order.Status,
		order.Price,
		order.Note,
		order.DeliveryDate,
		order.Paid,
		order.ModifiedBy,
		order.Action,
	)

	if err != nil {
		fmt.Printf("Error creating the order in the history table: %v", err)
		return errors.NewInternalServerError(errors.ErrCreatingOrderHistory)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Could not get the rows affected: %v", err)
	}

	if rows == 0 {
		return errors.NewNotFound(errors.ErrOrderNotFound)
	}

	fmt.Printf("RESULT: %v", rows)

	return nil
}

// GetOrderHistoryByOrderID retrieves order history by order ID
func (r *OrderRepository) GetOrderHistoryByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderHistory, error) {
	query := `
		SELECT id_order_history, id_order, id_user, status, total_price, note, 
			delivery_date, paid, modified_on, modified_by, action
		FROM orders_history 
		WHERE id_order = ?
		ORDER BY modified_on DESC
	`

	rows, err := r.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error querying order history: %w", err)
	}
	defer rows.Close()

	var histories []oModel.OrderHistory
	for rows.Next() {
		var history oModel.OrderHistory
		err := rows.Scan(
			&history.ID,
			&history.IDOrder,
			&history.IdUser,
			&history.Status,
			&history.Price,
			&history.Note,
			&history.DeliveryDate,
			&history.Paid,
			&history.ModifiedOn,
			&history.ModifiedBy,
			&history.Action,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order history row: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order history rows: %w", err)
	}

	return histories, nil
}

// UpdateOrderStatus updates the status of an order
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error {
	query := `UPDATE orders SET status = ? WHERE id_order = ?`

	result, err := r.DB.ExecContext(ctx, query, status, orderID)
	if err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFound(errors.ErrOrderNotFound)
	}

	return nil
}

// UpdateOrderPaidStatus updates the paid status of an order
func (r *OrderRepository) UpdateOrderPaidStatus(ctx context.Context, orderID uint64, paid bool) error {
	query := `UPDATE orders SET paid = ? WHERE id_order = ?`

	result, err := r.DB.ExecContext(ctx, query, paid, orderID)
	if err != nil {
		return fmt.Errorf("error updating order paid status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFound(errors.ErrOrderNotFound)
	}

	return nil
}
