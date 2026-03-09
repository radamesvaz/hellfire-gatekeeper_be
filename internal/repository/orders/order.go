package order

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
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
	return r.GetOrdersWithFilters(ctx, false, nil)
}

// GetOrdersWithFilters fetches orders with optional status filtering.
//
// Parameters:
// - ctx (context.Context): The context for query execution.
// - ignoreStatus (bool): If true, returns all orders including deleted ones. If false, excludes deleted orders.
// - statusFilter (*string): Optional specific status to filter by. If nil, uses ignoreStatus logic.
//
// Returns:
// - ([]oModel.OrderResponse): A list of orders with their products.
// - (error): If the query, scan, or row iteration fails.
func (r *OrderRepository) GetOrdersWithFilters(ctx context.Context, ignoreStatus bool, statusFilter *string) ([]oModel.OrderResponse, error) {
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
        LEFT JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        INNER JOIN products p ON oi.id_product = p.id_product
    `

	// Add WHERE clause based on filters
	whereClause := ""
	if statusFilter != nil {
		// Filter by specific status
		whereClause = " WHERE o.status = $1"
	} else if !ignoreStatus {
		// Exclude deleted orders by default
		whereClause = " WHERE o.status != 'deleted'"
	}

	query += whereClause + " ORDER BY o.id_order"

	var rows *sql.Rows
	var err error

	if statusFilter != nil {
		rows, err = r.DB.QueryContext(ctx, query, *statusFilter)
	} else {
		rows, err = r.DB.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("error executing query to fetch orders: %w", err)
	}
	defer rows.Close()

	ordersMap := make(map[uint64]*oModel.OrderResponse)

	for rows.Next() {
		var (
			idOrder      uint64
			idUser       sql.NullInt64
			totalPrice   float64
			status       string
			note         string
			deliveryDate time.Time
			paid         bool
			createdOn    time.Time
			userName     sql.NullString
			phone        sql.NullString
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
			resp := &oModel.OrderResponse{
				ID:           idOrder,
				Price:        totalPrice,
				Status:       oModel.OrderStatus(status),
				Note:         note,
				DeliveryDate: deliveryDate,
				Paid:         paid,
				CreatedOn:    createdOn,
				OrderItems:   []oModel.OrderItems{},
			}
			if idUser.Valid {
				resp.IdUser = uint64(idUser.Int64)
			}
			if userName.Valid {
				resp.User = userName.String
			}
			if phone.Valid {
				resp.Phone = phone.String
			}
			ordersMap[idOrder] = resp
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
	keys := make([]uint64, 0, len(ordersMap))
	for id := range ordersMap {
		keys = append(keys, id)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, id := range keys {
		orders = append(orders, *ordersMap[id])
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
        LEFT JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        INNER JOIN products p ON oi.id_product = p.id_product
        WHERE o.id_order = $1
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
			idUser       sql.NullInt64
			totalPrice   float64
			status       string
			note         string
			deliveryDate time.Time
			paid         bool
			createdOn    time.Time
			userName     sql.NullString
			phone        sql.NullString
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
			if idUser.Valid {
				order.IdUser = uint64(idUser.Int64)
			}
			order.Price = totalPrice
			order.Status = oModel.OrderStatus(status)
			order.Note = note
			order.DeliveryDate = deliveryDate
			order.Paid = paid
			order.CreatedOn = createdOn
			if userName.Valid {
				order.User = userName.String
			}
			if phone.Valid {
				order.Phone = phone.String
			}
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

// GetOrderItemsByOrderID gets all items for a specific order
func (r *OrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderItems, error) {
	return r.getOrderItemsByOrderIDQuery(ctx, r.DB, orderID)
}

// GetOrderItemsByOrderIDTx gets all items for a specific order within a transaction (for use in CancelExpiredOrders).
func (r *OrderRepository) GetOrderItemsByOrderIDTx(ctx context.Context, tx *sql.Tx, orderID uint64) ([]oModel.OrderItems, error) {
	return r.getOrderItemsByOrderIDQuery(ctx, tx, orderID)
}

func (r *OrderRepository) getOrderItemsByOrderIDQuery(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, orderID uint64) ([]oModel.OrderItems, error) {
	query := `
		SELECT 
			oi.id_order_item,
			oi.id_order,
			oi.id_product,
			p.name AS product_name,
			oi.quantity
		FROM order_items oi
		INNER JOIN products p ON oi.id_product = p.id_product
		WHERE oi.id_order = $1
		ORDER BY oi.id_order_item
	`

	rows, err := q.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error querying order items: %w", err)
	}
	defer rows.Close()

	var items []oModel.OrderItems
	for rows.Next() {
		var item oModel.OrderItems
		err := rows.Scan(
			&item.ID,
			&item.IdOrder,
			&item.IdProduct,
			&item.Name,
			&item.Quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over order item rows: %w", err)
	}

	return items, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (id uint64, err error) {
	return r.createOrderTx(ctx, tx, order)
}

func (r *OrderRepository) CreateOrderItems(ctx context.Context, tx *sql.Tx, items []oModel.OrderItemRequest) error {
	return r.createOrderItemTx(ctx, tx, items)
}

// BeginTx starts a new transaction (for use by services that orchestrate order + product operations).
func (r *OrderRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.DB.BeginTx(ctx, nil)
}

func (r *OrderRepository) createOrderTx(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (id uint64, err error) {
	logger.Debug().
		Uint64("user_id", order.IdUser).
		Float64("total_price", order.Price).
		Str("status", string(order.Status)).
		Msg("Creating order for user")

	query := `INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_order`

	var insertedID uint64
	err = tx.QueryRowContext(
		ctx,
		query,
		order.IdUser,
		order.Price,
		order.Status,
		order.Note,
		order.DeliveryDate,
		order.Paid,
	).Scan(&insertedID)

	if err != nil {
		logger.Err(err).
			Uint64("user_id", order.IdUser).
			Msg("Error creating the order")
		return 0, errors.NewInternalServerError(errors.ErrCreatingOrder)
	}

	logger.Debug().
		Uint64("order_id", insertedID).
		Uint64("user_id", order.IdUser).
		Msg("Order created successfully")
	return insertedID, nil
}

func (r *OrderRepository) createOrderItemTx(ctx context.Context, tx *sql.Tx, items []oModel.OrderItemRequest) error {
	if len(items) > 0 {
		logger.Debug().
			Uint64("order_id", items[0].IdOrder).
			Int("item_count", len(items)).
			Msg("Creating items for order")
	}
	exec := execerFrom(tx, r.DB)
	query := `INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)`

	for _, item := range items {
		_, err := exec.ExecContext(ctx, query, item.IdOrder, item.IdProduct, item.Quantity)
		if err != nil {
			logger.Err(err).
				Uint64("order_id", item.IdOrder).
				Uint64("product_id", item.IdProduct).
				Uint64("quantity", item.Quantity).
				Msg("Error inserting order item")
			return errors.NewInternalServerError(errors.ErrCreatingOrderItem)
		}
	}

	logger.Debug().
		Int("item_count", len(items)).
		Msg("Order items created successfully")
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

// nullStringFromPtr converts a *string to sql.NullString for use in DB Exec/Query.
// A nil pointer becomes a NullString with Valid: false (stored as NULL); otherwise
// the string value is copied and Valid is set to true.
func nullStringFromPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// CreateOrderHistory creates a new order history record
func (r *OrderRepository) CreateOrderHistory(_ context.Context, order oModel.OrderHistory) error {
	return r.createOrderHistoryExec(nil, r.DB, order)
}

// CreateOrderHistoryTx creates a new order history record within a transaction (for use in CancelExpiredOrders).
func (r *OrderRepository) CreateOrderHistoryTx(_ context.Context, tx *sql.Tx, order oModel.OrderHistory) error {
	return r.createOrderHistoryExec(tx, r.DB, order)
}

func (r *OrderRepository) createOrderHistoryExec(tx *sql.Tx, fallback *sql.DB, order oModel.OrderHistory) error {
	logger.Debug().
		Uint64("order_id", order.IDOrder).
		Str("action", string(order.Action)).
		Str("status", string(order.Status)).
		Msg("Creating order history")

	var idUserVal sql.NullInt64
	if order.IdUser != nil {
		idUserVal = sql.NullInt64{Int64: int64(*order.IdUser), Valid: true}
	}

	exec := execerFrom(tx, fallback)
	result, err := exec.ExecContext(context.Background(),
		`INSERT INTO orders_history (
		id_order, 
		id_user, 
		status, 
		total_price, 
		note,
		delivery_date,
		paid,
		cancellation_reason,
		modified_by, 
		action
		) 
		VALUES (
		$1,
		$2, 
		$3, 
		$4, 
		$5,
		$6,
		$7,
		$8,
		$9, 
		$10)`,
		order.IDOrder,
		idUserVal,
		order.Status,
		order.Price,
		order.Note,
		order.DeliveryDate,
		order.Paid,
		nullStringFromPtr(order.CancellationReason),
		order.ModifiedBy,
		order.Action,
	)

	if err != nil {
		logger.Err(err).
			Uint64("order_id", order.IDOrder).
			Msg("Error creating the order in the history table")
		return errors.NewInternalServerError(errors.ErrCreatingOrderHistory)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("order_id", order.IDOrder).
			Msg("Could not get the rows affected")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("order_id", order.IDOrder).
			Msg("No rows affected when creating order history")
		return errors.NewNotFound(errors.ErrOrderNotFound)
	}

	logger.Debug().
		Uint64("order_id", order.IDOrder).
		Int64("rows_affected", rows).
		Msg("Order history created successfully")

	return nil
}

// GetOrderHistoryByOrderID retrieves order history by order ID
func (r *OrderRepository) GetOrderHistoryByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderHistory, error) {
	query := `
		SELECT id_order_history, id_order, id_user, status, total_price, note, 
			delivery_date, paid, cancellation_reason, modified_on, modified_by, action
		FROM orders_history 
		WHERE id_order = $1
		ORDER BY modified_on DESC
	`

	rows, err := r.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error querying order history: %w", err)
	}
	defer rows.Close()

	var histories []oModel.OrderHistory
	for rows.Next() {
		var (
			history            oModel.OrderHistory
			idUser             sql.NullInt64
			cancellationReason sql.NullString
		)
		err := rows.Scan(
			&history.ID,
			&history.IDOrder,
			&idUser,
			&history.Status,
			&history.Price,
			&history.Note,
			&history.DeliveryDate,
			&history.Paid,
			&cancellationReason,
			&history.ModifiedOn,
			&history.ModifiedBy,
			&history.Action,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order history row: %w", err)
		}
		if idUser.Valid {
			u := uint64(idUser.Int64)
			history.IdUser = &u
		}
		if cancellationReason.Valid {
			history.CancellationReason = &cancellationReason.String
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order history rows: %w", err)
	}

	return histories, nil
}

// GetExpiredPendingOrders returns orders that are pending, unpaid, and created before the given expiration time (ghost orders).
func (r *OrderRepository) GetExpiredPendingOrders(ctx context.Context, expirationTime time.Time) ([]oModel.Order, error) {
	query := `
		SELECT id_order, id_user, total_price, status, note, created_on, delivery_date, paid, cancellation_reason
		FROM orders
		WHERE status = 'pending' AND paid = false AND created_on < $1
		ORDER BY created_on ASC
	`
	rows, err := r.DB.QueryContext(ctx, query, expirationTime)
	if err != nil {
		return nil, fmt.Errorf("error querying expired pending orders: %w", err)
	}
	defer rows.Close()

	var orders []oModel.Order
	for rows.Next() {
		var o oModel.Order
		var note, cancellationReason sql.NullString
		err := rows.Scan(
			&o.ID,
			&o.IdUser,
			&o.Price,
			&o.Status,
			&note,
			&o.CreatedOn,
			&o.DeliveryDate,
			&o.Paid,
			&cancellationReason,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning expired order row: %w", err)
		}
		if note.Valid {
			o.Note = note.String
		}
		if cancellationReason.Valid {
			o.CancellationReason = &cancellationReason.String
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired orders: %w", err)
	}
	return orders, nil
}

// UpdateOrderStatus updates the status of an order and optionally sets the cancellation reason.
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	return r.updateOrderStatusExec(ctx, r.DB, orderID, status, cancellationReason)
}

// UpdateOrderStatusTx updates the order status within a transaction (for use in CancelExpiredOrders).
func (r *OrderRepository) UpdateOrderStatusTx(ctx context.Context, tx *sql.Tx, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	return r.updateOrderStatusExec(ctx, tx, orderID, status, cancellationReason)
}

func (r *OrderRepository) updateOrderStatusExec(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	query := `UPDATE orders SET status = $1, cancellation_reason = $2 WHERE id_order = $3`
	result, err := exec.ExecContext(ctx, query, status, nullStringFromPtr(cancellationReason), orderID)
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
	query := `UPDATE orders SET paid = $1 WHERE id_order = $2`

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
