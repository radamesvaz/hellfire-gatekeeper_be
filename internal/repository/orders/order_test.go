package order

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	stdErrors "errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderRepository_GetOrders(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

	deliveryDate := time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC)
	createdOn := time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		mockRows      *sqlmock.Rows
		expected      []oModel.OrderResponse
		expectedError bool
		errorStatus   int
		mockError     error
	}{
		{
			name: "HAPPY PATH: getting all orders",
			mockRows: sqlmock.NewRows([]string{
				"id_order",
				"id_user",
				"total_price",
				"status",
				"note",
				"delivery_date",
				"paid",
				"created_on",
				"user_name",
				"phone",
				"id_order_item",
				"id_product",
				"product_name",
				"quantity",
			}).
				AddRow(
					1,
					2,
					50.0,
					"pending",
					"note testing",
					time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC),
					false,
					time.Date(2025, 4, 10, 10, 0, 0, 0, time.UTC),
					"Client Example",
					"66-6666",
					1,
					2,
					"Product A",
					2,
				).
				AddRow(
					1,
					2,
					50.0,
					"pending",
					"note testing",
					deliveryDate,
					false,
					createdOn,
					"Client Example",
					"66-6666",
					2,
					1,
					"Product B",
					3,
				).AddRow(
				2,
				2,
				25,
				"delivered",
				"note testing",
				time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC),
				true,
				time.Date(2025, 4, 10, 10, 0, 0, 0, time.UTC),
				"Client Example",
				"66-6666",
				3,
				1,
				"Product B",
				1,
			),
			expected: []oModel.OrderResponse{
				{
					ID:           1,
					IdUser:       2,
					Price:        50.0,
					Status:       oModel.StatusPending,
					Note:         "note testing",
					DeliveryDate: time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC),
					Paid:         false,
					CreatedOn:    time.Date(2025, 4, 10, 10, 0, 0, 0, time.UTC),
					User:         "Client Example",
					Phone:        "66-6666",
					OrderItems: []oModel.OrderItems{
						{
							ID:        1,
							IdOrder:   1,
							IdProduct: 2,
							Name:      "Product A",
							Quantity:  2,
						},
						{
							ID:        2,
							IdOrder:   1,
							IdProduct: 1,
							Name:      "Product B",
							Quantity:  3,
						},
					},
				},
				{
					ID:           2,
					IdUser:       2,
					Price:        25,
					Status:       oModel.StatusDelivered,
					Note:         "note testing",
					DeliveryDate: time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC),
					Paid:         true,
					CreatedOn:    time.Date(2025, 4, 10, 10, 0, 0, 0, time.UTC),
					User:         "Client Example",
					Phone:        "66-6666",
					OrderItems: []oModel.OrderItems{
						{
							ID:        3,
							IdOrder:   2,
							IdProduct: 1,
							Name:      "Product B",
							Quantity:  1,
						},
					},
				},
			},
			expectedError: false,
			errorStatus:   0,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`SELECT
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
					ORDER BY o.id_order`,
					),
				).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`SELECT
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
					ORDER BY o.id_order`,
					),
				).
					WillReturnRows(tt.mockRows)
			}

			order, err := repo.GetOrders(context.Background())
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, order)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOrderRepository_GetOrderByID(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

	deliveryDate := time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC)
	createdOn := time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		mockRows         *sqlmock.Rows
		expected         oModel.OrderResponse
		expectedError    bool
		idOrderForLookup uint64
		errorStatus      int
		mockError        error
	}{
		{
			name: "HAPPY PATH: getting an order with multiple products",
			mockRows: sqlmock.NewRows([]string{
				"id_order",
				"id_user",
				"total_price",
				"status",
				"note",
				"delivery_date",
				"paid",
				"created_on",
				"user_name",
				"phone",
				"id_order_item",
				"id_product",
				"product_name",
				"quantity",
			}).
				AddRow(
					1,
					2,
					50.0,
					"pending",
					"note testing",
					deliveryDate,
					false,
					createdOn,
					"Client Example",
					"66-6666",
					1,
					2,
					"Product A",
					2,
				).
				AddRow(
					1,
					2,
					50.0,
					"pending",
					"note testing",
					deliveryDate,
					false,
					createdOn,
					"Client Example",
					"66-6666",
					2,
					1,
					"Product B",
					3,
				),
			expected: oModel.OrderResponse{
				ID:           1,
				IdUser:       2,
				Price:        50.0,
				Status:       oModel.StatusPending,
				Note:         "note testing",
				DeliveryDate: time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC),
				Paid:         false,
				CreatedOn:    time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC),
				User:         "Client Example",
				Phone:        "66-6666",
				OrderItems: []oModel.OrderItems{
					{
						ID:        1,
						IdOrder:   1,
						IdProduct: 2,
						Name:      "Product A",
						Quantity:  2,
					},
					{
						ID:        2,
						IdOrder:   1,
						IdProduct: 1,
						Name:      "Product B",
						Quantity:  3,
					},
				},
			},
			expectedError:    false,
			errorStatus:      0,
			idOrderForLookup: 1,
			mockError:        nil,
		},
		{
			name: "SAD PATH: Order not found",
			mockRows: sqlmock.NewRows([]string{
				"id_order",
				"id_user",
				"total_price",
				"status", "note",
				"delivery_date",
				"paid",
				"created_on",
				"user_name",
				"phone",
				"id_order_item",
				"id_product",
				"product_name",
				"quantity",
			}).
				AddRow(1, 2, 50.0, "pending", "note testing", deliveryDate, false, createdOn, "Client Example", "66-6666",
					1, 2, "Product A", 2).
				AddRow(1, 2, 50.0, "pending", "note testing", deliveryDate, false, createdOn, "Client Example", "66-6666",
					2, 1, "Product B", 3),
			expected: oModel.OrderResponse{
				ID:           1,
				IdUser:       2,
				Price:        50.0,
				Status:       oModel.StatusPending,
				Note:         "note testing",
				DeliveryDate: time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC),
				Paid:         false,
				CreatedOn:    time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC),
				User:         "Client Example",
				Phone:        "66-6666",
				OrderItems: []oModel.OrderItems{
					{
						ID:        1,
						IdOrder:   1,
						IdProduct: 2,
						Name:      "Product A",
						Quantity:  2,
					},
					{
						ID:        2,
						IdOrder:   1,
						IdProduct: 1,
						Name:      "Product B",
						Quantity:  3,
					},
				},
			},
			expectedError:    true,
			errorStatus:      404,
			idOrderForLookup: 9999999,
			mockError:        errors.ErrOrderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`SELECT
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
					WHERE o.id_order = $1`,
					),
				).
					WithArgs(tt.idOrderForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`SELECT
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
					WHERE o.id_order = $1`,
					),
				).
					WithArgs(tt.idOrderForLookup).
					WillReturnRows(tt.mockRows)
			}

			order, err := repo.GetOrderByID(context.Background(), tt.idOrderForLookup)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, order)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOrderRepository_CreateOrderOrchestrator(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &OrderRepository{DB: db}

	deliveryDate := time.Date(2025, 5, 20, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		order        oModel.CreateFullOrder
		mockBehavior func()
		expectError  bool
	}{
		{
			name: "HAPPY PATH: everything succeeds",
			order: oModel.CreateFullOrder{
				IdUser:       1,
				DeliveryDate: deliveryDate,
				Note:         "Éxito total",
				Price:        100.0,
				Status:       oModel.StatusPending,
				Paid:         false,
				OrderItems: []oModel.OrderItemRequest{
					{IdProduct: 2, Quantity: 3},
					{IdProduct: 5, Quantity: 1},
				},
			},
			mockBehavior: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(
					"INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_order",
				)).WithArgs(1, 100.0, oModel.StatusPending, "Éxito total", deliveryDate, false).
					WillReturnRows(sqlmock.NewRows([]string{"id_order"}).AddRow(1))

				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)",
				)).WithArgs(1, 2, 3).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)",
				)).WithArgs(1, 5, 1).
					WillReturnResult(sqlmock.NewResult(2, 1))

				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "SAD PATH: item insert fails, triggers rollback",
			order: oModel.CreateFullOrder{
				IdUser:       1,
				DeliveryDate: deliveryDate,
				Note:         "Este fallo es intencional",
				Price:        100.0,
				Status:       oModel.StatusPending,
				Paid:         false,
				OrderItems: []oModel.OrderItemRequest{
					{IdProduct: 2, Quantity: 3},
					{IdProduct: 99, Quantity: 1},
				},
			},
			mockBehavior: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(
					"INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_order",
				)).WithArgs(1, 100.0, oModel.StatusPending, "Este fallo es intencional", deliveryDate, false).
					WillReturnRows(sqlmock.NewRows([]string{"id_order"}).AddRow(1))

				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)",
				)).WithArgs(1, 2, 3).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)",
				)).WithArgs(1, 99, 1).
					WillReturnError(stdErrors.New("simulated item failure"))

				mock.ExpectRollback()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			_, err := repo.CreateOrderOrchestrator(context.Background(), tt.order)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOrderRepository_CreateOrder(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

	deliveryDate := time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		orderRequest  oModel.CreateOrderRequest
		expected      uint64
		expectedError bool
		errorStatus   int
		mockError     error
	}{
		{
			name:     "HAPPY PATH: creating an order",
			expected: 666,
			orderRequest: oModel.CreateOrderRequest{
				IdUser:       2,
				DeliveryDate: deliveryDate,
				Note:         "entregar a la tarde",
				Price:        20,
				Status:       oModel.StatusPending,
				Paid:         false,
			},
			expectedError: false,
			errorStatus:   0,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock transaction
			mock.ExpectBegin()
			tx, _ := db.Begin()

			if tt.expectedError {
				mock.ExpectQuery(regexp.QuoteMeta(
					"INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_order",
				)).WithArgs(
					tt.orderRequest.IdUser,
					tt.orderRequest.Price,
					tt.orderRequest.Status,
					tt.orderRequest.Note,
					tt.orderRequest.DeliveryDate,
					tt.orderRequest.Paid,
				).WillReturnError(tt.mockError)
			} else {
				mock.ExpectQuery(regexp.QuoteMeta(
					"INSERT INTO orders (id_user, total_price, status, note, delivery_date, paid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_order",
				)).WithArgs(
					tt.orderRequest.IdUser,
					tt.orderRequest.Price,
					oModel.StatusPending,
					tt.orderRequest.Note,
					tt.orderRequest.DeliveryDate,
					tt.orderRequest.Paid,
				).WillReturnRows(sqlmock.NewRows([]string{"id_order"}).AddRow(tt.expected))
			}

			orderID, err := repo.CreateOrder(context.Background(), tx, tt.orderRequest)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, orderID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOrderRepository_CreateOrderItems(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

	tests := []struct {
		name              string
		orderItemsRequest []oModel.OrderItemRequest
		expectedError     bool
		errorStatus       int
		mockError         error
	}{
		{
			name: "HAPPY PATH: creating order items",
			orderItemsRequest: []oModel.OrderItemRequest{
				oModel.OrderItemRequest{
					IdProduct: 1,
					IdOrder:   1,
					Quantity:  5,
				},
				oModel.OrderItemRequest{
					IdProduct: 2,
					IdOrder:   1,
					Quantity:  2,
				},
			},
			expectedError: false,
			errorStatus:   0,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, item := range tt.orderItemsRequest {
				exec := mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO order_items (id_order, id_product, quantity) VALUES ($1, $2, $3)",
				)).WithArgs(item.IdOrder, item.IdProduct, item.Quantity)

				if tt.expectedError && i == 1 {
					exec.WillReturnError(tt.mockError)
					break
				} else {
					exec.WillReturnResult(sqlmock.NewResult(1, 1))
				}
			}

			err := repo.CreateOrderItems(context.Background(), nil, tt.orderItemsRequest)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
