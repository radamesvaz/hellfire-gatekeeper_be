package order

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
)

func TestOrderRepository_GetOrderByID(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

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
			mockRows: sqlmock.NewRows([]string{ // change when ready 4 readability
				"id_order", "total_price", "status", "note", "delivery_date", "created_on", "user_name",
				"id_order_item", "id_product", "product_name", "quantity",
			}).
				AddRow(1, 50.0, "pending", "note testing", "2025-04-30 10:00:00", "2025-04-25 10:00:00", "Client Example",
					1, 2, "Product A", 2).
				AddRow(1, 50.0, "pending", "note testing", "2025-04-30 10:00:00", "2025-04-25 10:00:00", "Client Example",
					2, 1, "Product B", 3),
			expected: oModel.OrderResponse{
				ID:           1,
				Price:        50.0,
				Status:       oModel.StatusPending,
				Note:         "note testing",
				DeliveryDate: time.Date(2025, 4, 30, 10, 0, 0, 0, time.UTC),
				CreatedOn:    time.Date(2025, 4, 25, 10, 0, 0, 0, time.UTC),
				User:         "Client Example",
				OrderItems: []oModel.OrderItems{
					{
						ID:        1,
						IdOrder:   1,
						IdProduct: 2,
						Name:      "Product B",
						Quantity:  2,
					},
					{
						ID:        1,
						IdOrder:   2,
						IdProduct: 1,
						Name:      "Product A",
						Quantity:  3,
					},
				},
			},
			expectedError:    false,
			errorStatus:      0,
			idOrderForLookup: 1,
			mockError:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta(
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
					),
				).
					WithArgs(tt.idOrderForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta(
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

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
