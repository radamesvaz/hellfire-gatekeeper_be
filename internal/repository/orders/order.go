package order

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/lib/pq"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/pagination"
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
func (r *OrderRepository) GetOrders(ctx context.Context, tenantID uint64) ([]oModel.OrderResponse, error) {
	return r.GetOrdersWithFilters(ctx, tenantID, false, nil)
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
func (r *OrderRepository) GetOrdersWithFilters(ctx context.Context, tenantID uint64, ignoreStatus bool, statusFilter *string) ([]oModel.OrderResponse, error) {
	query := `
        SELECT 
            o.id_order, 
            o.tenant_id,
            o.id_user,
            o.total_price, 
            o.status, 
            o.note, 
            o.delivery_date, 
            o.delivery_direction,
            o.paid,
            o.created_on,
            o.expires_at,
            o.cancellation_reason,
            u.name AS user_name, 
            u.phone,
            oi.id_order_item, 
            oi.id_product, 
            oi.product_name_snapshot,
            oi.unit_price_snapshot,
            oi.quantity
        FROM orders o
        LEFT JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        WHERE o.tenant_id = $1
    `

	if statusFilter != nil {
		query += " AND o.status = $2"
	} else if !ignoreStatus {
		query += " AND o.status != 'deleted'"
	}

	query += " ORDER BY o.id_order"

	var rows *sql.Rows
	var err error

	if statusFilter != nil {
		rows, err = r.DB.QueryContext(ctx, query, tenantID, *statusFilter)
	} else {
		rows, err = r.DB.QueryContext(ctx, query, tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("error executing query to fetch orders: %w", err)
	}
	defer rows.Close()

	return ordersFromJoinRows(rows, orderJoinSortIDAsc)
}

// ListOrdersPageResult is one page of orders (cursor on created_on ASC, id_order ASC).
type ListOrdersPageResult struct {
	Items      []oModel.OrderResponse
	NextCursor *string
}

// ListOrdersWithFiltersPage returns orders for the tenant with the same status semantics as GetOrdersWithFilters,
// ordered by creation time ascending (then id_order), paginated with an opaque cursor (see pagination.OrderKeyset).
func (r *OrderRepository) ListOrdersWithFiltersPage(
	ctx context.Context,
	tenantID uint64,
	ignoreStatus bool,
	statusFilter *string,
	limit int,
	after *pagination.OrderKeyset,
	filterUserID *uint64,
) (ListOrdersPageResult, error) {
	if limit < 1 {
		return ListOrdersPageResult{}, fmt.Errorf("limit must be at least 1")
	}

	idQuery, idArgs := buildOrderIDPageQuery(tenantID, ignoreStatus, statusFilter, after, filterUserID, limit+1)
	idRows, err := r.DB.QueryContext(ctx, idQuery, idArgs...)
	if err != nil {
		return ListOrdersPageResult{}, fmt.Errorf("listing order ids: %w", err)
	}
	var ids []uint64
	for idRows.Next() {
		var id uint64
		if err := idRows.Scan(&id); err != nil {
			idRows.Close()
			return ListOrdersPageResult{}, fmt.Errorf("scan order id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := idRows.Err(); err != nil {
		idRows.Close()
		return ListOrdersPageResult{}, err
	}
	idRows.Close()

	hasNext := len(ids) > limit
	if hasNext {
		ids = ids[:limit]
	}
	if len(ids) == 0 {
		return ListOrdersPageResult{Items: []oModel.OrderResponse{}, NextCursor: nil}, nil
	}

	detailQuery := `
        SELECT 
            o.id_order, 
            o.tenant_id,
            o.id_user,
            o.total_price, 
            o.status, 
            o.note, 
            o.delivery_date, 
            o.delivery_direction,
            o.paid,
            o.created_on,
            o.expires_at,
            o.cancellation_reason,
            u.name AS user_name, 
            u.phone,
            oi.id_order_item, 
            oi.id_product, 
            oi.product_name_snapshot,
            oi.unit_price_snapshot,
            oi.quantity
        FROM orders o
        LEFT JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        WHERE o.tenant_id = $1 AND o.id_order = ANY($2::bigint[])
        ORDER BY o.created_on ASC, o.id_order ASC, oi.id_order_item ASC
	`
	rows, err := r.DB.QueryContext(ctx, detailQuery, tenantID, pq.Array(ids))
	if err != nil {
		return ListOrdersPageResult{}, fmt.Errorf("fetching order details: %w", err)
	}
	defer rows.Close()

	orders, err := ordersFromJoinRows(rows, orderJoinSortCreatedOnAscIDAsc)
	if err != nil {
		return ListOrdersPageResult{}, err
	}

	var next *string
	if hasNext && len(orders) > 0 {
		last := orders[len(orders)-1]
		s, err := pagination.EncodeOrderCursor(last.CreatedOn, last.ID)
		if err != nil {
			return ListOrdersPageResult{}, fmt.Errorf("encoding next cursor: %w", err)
		}
		next = &s
	}

	return ListOrdersPageResult{Items: orders, NextCursor: next}, nil
}

func buildOrderIDPageQuery(tenantID uint64, ignoreStatus bool, statusFilter *string, after *pagination.OrderKeyset, filterUserID *uint64, limit int) (string, []interface{}) {
	q := `SELECT o.id_order FROM orders o WHERE o.tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2
	if statusFilter != nil {
		q += fmt.Sprintf(" AND o.status = $%d", idx)
		args = append(args, *statusFilter)
		idx++
	} else if !ignoreStatus {
		q += " AND o.status != 'deleted'"
	}
	if filterUserID != nil {
		q += fmt.Sprintf(" AND o.id_user = $%d", idx)
		args = append(args, *filterUserID)
		idx++
	}
	if after != nil {
		tArg := idx
		idArg := idx + 1
		q += fmt.Sprintf(" AND (o.created_on > $%d OR (o.created_on = $%d AND o.id_order > $%d))", tArg, tArg, idArg)
		args = append(args, after.CreatedOn, after.ID)
		idx += 2
	}
	q += fmt.Sprintf(" ORDER BY o.created_on ASC, o.id_order ASC LIMIT $%d", idx)
	args = append(args, limit)
	return q, args
}

type orderJoinSort int

const (
	orderJoinSortIDAsc orderJoinSort = iota
	orderJoinSortIDDesc
	orderJoinSortCreatedOnAscIDAsc
)

func ordersFromJoinRows(rows *sql.Rows, sortMode orderJoinSort) ([]oModel.OrderResponse, error) {
	ordersMap := make(map[uint64]*oModel.OrderResponse)

	for rows.Next() {
		var (
			idOrder            uint64
			tenantIDRow        uint64
			idUser             sql.NullInt64
			totalPrice         float64
			status             string
			note               string
			deliveryDate       time.Time
			deliveryDirection  string
			paid               bool
			createdOn          time.Time
			expiresAt          sql.NullTime
			cancellationReason sql.NullString
			userName           sql.NullString
			phone              sql.NullString
			idOrderItem        uint64
			idProduct          uint64
			productName        string
			unitPrice          float64
			quantity           uint64
		)

		err := rows.Scan(
			&idOrder,
			&tenantIDRow,
			&idUser,
			&totalPrice,
			&status,
			&note,
			&deliveryDate,
			&deliveryDirection,
			&paid,
			&createdOn,
			&expiresAt,
			&cancellationReason,
			&userName,
			&phone,
			&idOrderItem,
			&idProduct,
			&productName,
			&unitPrice,
			&quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row for order: %w", err)
		}

		if _, exists := ordersMap[idOrder]; !exists {
			resp := &oModel.OrderResponse{
				ID:                idOrder,
				TenantID:          tenantIDRow,
				Price:             totalPrice,
				Status:            oModel.OrderStatus(status),
				Note:              note,
				DeliveryDate:      deliveryDate,
				DeliveryDirection: deliveryDirection,
				Paid:              paid,
				CreatedOn:         createdOn,
				OrderItems:        []oModel.OrderItems{},
			}
			if idUser.Valid {
				resp.IdUser = uint64(idUser.Int64)
			}
			if expiresAt.Valid {
				resp.ExpiresAt = expiresAt.Time
			}
			if cancellationReason.Valid {
				resp.CancellationReason = &cancellationReason.String
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
			UnitPrice: unitPrice,
			Quantity:  quantity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over order rows: %w", err)
	}

	orders := make([]oModel.OrderResponse, 0, len(ordersMap))
	for _, resp := range ordersMap {
		orders = append(orders, *resp)
	}

	switch sortMode {
	case orderJoinSortIDAsc:
		sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	case orderJoinSortIDDesc:
		sort.Slice(orders, func(i, j int) bool { return orders[i].ID > orders[j].ID })
	case orderJoinSortCreatedOnAscIDAsc:
		sort.SliceStable(orders, func(i, j int) bool {
			a, b := orders[i], orders[j]
			if a.CreatedOn.Before(b.CreatedOn) {
				return true
			}
			if b.CreatedOn.Before(a.CreatedOn) {
				return false
			}
			return a.ID < b.ID
		})
	default:
		sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	}

	return orders, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, tenantID, id uint64) (oModel.OrderResponse, error) {
	query := `
        SELECT 
            o.id_order, 
            o.tenant_id,
            o.id_user,
            o.total_price, 
            o.status, 
            o.note, 
            o.delivery_date, 
            o.delivery_direction,
            o.paid,
            o.created_on,
            o.expires_at,
            o.cancellation_reason,
            u.name AS user_name, 
            u.phone,
            oi.id_order_item, 
            oi.id_product, 
            oi.product_name_snapshot,
            oi.unit_price_snapshot,
            oi.quantity
        FROM orders o
        LEFT JOIN users u ON o.id_user = u.id_user
        INNER JOIN order_items oi ON o.id_order = oi.id_order
        WHERE o.id_order = $1 AND o.tenant_id = $2
    `
	order := oModel.OrderResponse{}
	order.OrderItems = []oModel.OrderItems{}

	rows, err := r.DB.QueryContext(ctx, query, id, tenantID)
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
			idOrder            uint64
			tenantIDRow        uint64
			idUser             sql.NullInt64
			totalPrice         float64
			status             string
			note               string
			deliveryDate       time.Time
			deliveryDirection  string
			paid               bool
			createdOn          time.Time
			expiresAt          sql.NullTime
			cancellationReason sql.NullString
			userName           sql.NullString
			phone              sql.NullString
			idOrderItem        uint64
			idProduct          uint64
			productName        string
			unitPrice          float64
			quantity           uint64
		)

		err := rows.Scan(
			&idOrder,
			&tenantIDRow,
			&idUser,
			&totalPrice,
			&status,
			&note,
			&deliveryDate,
			&deliveryDirection,
			&paid,
			&createdOn,
			&expiresAt,
			&cancellationReason,
			&userName,
			&phone,
			&idOrderItem,
			&idProduct,
			&productName,
			&unitPrice,
			&quantity,
		)
		if err != nil {
			return oModel.OrderResponse{}, fmt.Errorf("Error formating the order id: %v. Error: %w", id, err)
		}

		if firstRow {
			order.ID = idOrder
			order.TenantID = tenantIDRow
			if idUser.Valid {
				order.IdUser = uint64(idUser.Int64)
			}
			order.Price = totalPrice
			order.Status = oModel.OrderStatus(status)
			order.Note = note
			order.DeliveryDate = deliveryDate
			order.DeliveryDirection = deliveryDirection
			order.Paid = paid
			order.CreatedOn = createdOn
			if expiresAt.Valid {
				order.ExpiresAt = expiresAt.Time
			}
			if cancellationReason.Valid {
				order.CancellationReason = &cancellationReason.String
			}
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
			UnitPrice: unitPrice,
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
func (r *OrderRepository) GetOrderItemsByOrderID(ctx context.Context, tenantID, orderID uint64) ([]oModel.OrderItems, error) {
	return r.getOrderItemsByOrderIDQuery(ctx, r.DB, tenantID, orderID)
}

// GetOrderItemsByOrderIDTx gets all items for a specific order within a transaction (for use in CancelExpiredOrders).
func (r *OrderRepository) GetOrderItemsByOrderIDTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64) ([]oModel.OrderItems, error) {
	return r.getOrderItemsByOrderIDQuery(ctx, tx, tenantID, orderID)
}

func (r *OrderRepository) getOrderItemsByOrderIDQuery(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, tenantID, orderID uint64) ([]oModel.OrderItems, error) {
	query := `
		SELECT 
			oi.id_order_item,
			oi.id_order,
			oi.id_product,
			COALESCE(oi.product_name_snapshot, ''),
			COALESCE(oi.unit_price_snapshot, 0),
			oi.quantity
		FROM order_items oi
		WHERE oi.id_order = $1 AND oi.tenant_id = $2
		ORDER BY oi.id_order_item
	`

	rows, err := q.QueryContext(ctx, query, orderID, tenantID)
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
			&item.UnitPrice,
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

func (r *OrderRepository) CreateOrderItems(ctx context.Context, tx *sql.Tx, tenantID uint64, items []oModel.OrderItemRequest) error {
	return r.createOrderItemTx(ctx, tx, tenantID, items)
}

// BeginTx starts a new transaction (for use by services that orchestrate order + product operations).
func (r *OrderRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.DB.BeginTx(ctx, nil)
}

func (r *OrderRepository) createOrderTx(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (id uint64, err error) {
	logger.Debug().
		Uint64("tenant_id", order.TenantID).
		Uint64("user_id", order.IdUser).
		Float64("total_price", order.Price).
		Str("status", string(order.Status)).
		Msg("Creating order for user")

	query := `INSERT INTO orders (tenant_id, id_user, total_price, status, note, delivery_date, delivery_direction, paid, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id_order`

	var insertedID uint64
	err = tx.QueryRowContext(
		ctx,
		query,
		order.TenantID,
		order.IdUser,
		order.Price,
		order.Status,
		order.Note,
		order.DeliveryDate,
		order.DeliveryDirection,
		order.Paid,
		order.ExpiresAt,
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

func (r *OrderRepository) createOrderItemTx(ctx context.Context, tx *sql.Tx, tenantID uint64, items []oModel.OrderItemRequest) error {
	if len(items) > 0 {
		logger.Debug().
			Uint64("order_id", items[0].IdOrder).
			Int("item_count", len(items)).
			Msg("Creating items for order")
	}
	exec := execerFrom(tx, r.DB)
	query := `INSERT INTO order_items (tenant_id, id_order, id_product, product_name_snapshot, unit_price_snapshot, quantity) VALUES ($1, $2, $3, $4, $5, $6)`

	for _, item := range items {
		_, err := exec.ExecContext(ctx, query, tenantID, item.IdOrder, item.IdProduct, item.ProductNameSnapshot, item.UnitPriceSnapshot, item.Quantity)
		if err != nil {
			logger.Err(err).
				Uint64("order_id", item.IdOrder).
				Uint64("product_id", item.IdProduct).
				Str("product_name_snapshot", item.ProductNameSnapshot).
				Float64("unit_price_snapshot", item.UnitPriceSnapshot).
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
		tenant_id,
		id_order, 
		id_user, 
		status, 
		total_price, 
		note,
		delivery_date,
		delivery_direction,
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
		$10, 
		$11,
		$12)`,
		order.TenantID,
		order.IDOrder,
		idUserVal,
		order.Status,
		order.Price,
		order.Note,
		order.DeliveryDate,
		order.DeliveryDirection,
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
func (r *OrderRepository) GetOrderHistoryByOrderID(ctx context.Context, tenantID, orderID uint64) ([]oModel.OrderHistory, error) {
	query := `
		SELECT id_order_history, tenant_id, id_order, id_user, status, total_price, note, 
			delivery_date, delivery_direction, paid, cancellation_reason, modified_on, modified_by, action
		FROM orders_history 
		WHERE id_order = $1 AND tenant_id = $2
		ORDER BY modified_on DESC
	`

	rows, err := r.DB.QueryContext(ctx, query, orderID, tenantID)
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
			&history.TenantID,
			&history.IDOrder,
			&idUser,
			&history.Status,
			&history.Price,
			&history.Note,
			&history.DeliveryDate,
			&history.DeliveryDirection,
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

// GetExpiredPendingOrders returns orders that are pending, unpaid, and created before the given expiration time (ghost orders) for a tenant.
func (r *OrderRepository) GetExpiredPendingOrders(ctx context.Context, tenantID uint64, expirationTime time.Time) ([]oModel.Order, error) {
	query := `
		SELECT id_order, tenant_id, id_user, total_price, status, note, created_on, delivery_date, delivery_direction, paid, cancellation_reason
		FROM orders
		WHERE tenant_id = $1 AND status = 'pending' AND paid = false AND created_on < $2
		ORDER BY created_on ASC
	`
	rows, err := r.DB.QueryContext(ctx, query, tenantID, expirationTime)
	if err != nil {
		return nil, fmt.Errorf("error querying expired pending orders: %w", err)
	}
	defer rows.Close()

	var orders []oModel.Order
	for rows.Next() {
		var o oModel.Order
		var note, deliveryDirection, cancellationReason sql.NullString
		err := rows.Scan(
			&o.ID,
			&o.TenantID,
			&o.IdUser,
			&o.Price,
			&o.Status,
			&note,
			&o.CreatedOn,
			&o.DeliveryDate,
			&deliveryDirection,
			&o.Paid,
			&cancellationReason,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning expired order row: %w", err)
		}
		if note.Valid {
			o.Note = note.String
		}
		if deliveryDirection.Valid {
			o.DeliveryDirection = deliveryDirection.String
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

// ClaimExpiredPendingOrdersTx atomically updates orders to the given status (e.g. cancelled) and returns
// the claimed rows. Used by the ghost-order cron so that overlapping runs or multiple workers
// never process the same order twice. Must be called within an existing transaction; the caller
// then performs stock reversion and history insert for each claimed order before committing.
func (r *OrderRepository) ClaimExpiredPendingOrdersTx(
	ctx context.Context,
	tx *sql.Tx,
	tenantID uint64,
	currentTime time.Time,
	status oModel.OrderStatus,
	cancellationReason *string,
) ([]oModel.Order, error) {
	query := `
		UPDATE orders
		SET status = $1, cancellation_reason = $2
		WHERE tenant_id = $3 AND status = 'pending' AND paid = false AND expires_at < $4
		RETURNING id_order, tenant_id, id_user, total_price, status, note, created_on, delivery_date, delivery_direction, paid, cancellation_reason
	`
	rows, err := tx.QueryContext(ctx, query, status, nullStringFromPtr(cancellationReason), tenantID, currentTime)
	if err != nil {
		return nil, fmt.Errorf("error claiming expired pending orders: %w", err)
	}
	defer rows.Close()

	var orders []oModel.Order
	for rows.Next() {
		var o oModel.Order
		var note, deliveryDirection, reason sql.NullString
		err := rows.Scan(
			&o.ID,
			&o.TenantID,
			&o.IdUser,
			&o.Price,
			&o.Status,
			&note,
			&o.CreatedOn,
			&o.DeliveryDate,
			&deliveryDirection,
			&o.Paid,
			&reason,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning claimed order row: %w", err)
		}
		if note.Valid {
			o.Note = note.String
		}
		if deliveryDirection.Valid {
			o.DeliveryDirection = deliveryDirection.String
		}
		if reason.Valid {
			o.CancellationReason = &reason.String
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claimed orders: %w", err)
	}
	return orders, nil
}

// UpdateOrderStatus updates the status of an order and optionally sets the cancellation reason.
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	return r.updateOrderStatusExec(ctx, r.DB, tenantID, orderID, status, cancellationReason)
}

// UpdateOrderStatusTx updates the order status within a transaction (for use in CancelExpiredOrders).
func (r *OrderRepository) UpdateOrderStatusTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	return r.updateOrderStatusExec(ctx, tx, tenantID, orderID, status, cancellationReason)
}

func (r *OrderRepository) updateOrderStatusExec(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	query := `UPDATE orders SET status = $1, cancellation_reason = $2 WHERE id_order = $3 AND tenant_id = $4`
	result, err := exec.ExecContext(ctx, query, status, nullStringFromPtr(cancellationReason), orderID, tenantID)
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
func (r *OrderRepository) UpdateOrderPaidStatus(ctx context.Context, tenantID, orderID uint64, paid bool) error {
	query := `UPDATE orders SET paid = $1 WHERE id_order = $2 AND tenant_id = $3`

	result, err := r.DB.ExecContext(ctx, query, paid, orderID, tenantID)
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
