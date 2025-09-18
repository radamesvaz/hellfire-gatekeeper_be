package order

import (
	"context"
	"regexp"
	"testing"
	"time"

	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
)

func TestOrderRepository_CreateOrderHistory(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &OrderRepository{DB: db}

	deliveryDate := time.Now().AddDate(0, 0, 7)

	tests := []struct {
		name          string
		payload       oModel.OrderHistory
		mockError     error
		expectedError bool
	}{
		{
			name: "HAPPY PATH: Creating an order history",
			payload: oModel.OrderHistory{
				IDOrder:      1,
				IdUser:       1,
				Status:       oModel.StatusPending,
				Price:        50.0,
				Note:         "Test order",
				DeliveryDate: sql.NullTime{Time: deliveryDate, Valid: true},
				Paid:         false,
				ModifiedBy:   1,
				Action:       oModel.ActionCreate,
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "HAPPY PATH: Updating an order history",
			payload: oModel.OrderHistory{
				IDOrder:      1,
				IdUser:       1,
				Status:       oModel.StatusPreparing,
				Price:        50.0,
				Note:         "Updated order",
				DeliveryDate: sql.NullTime{Time: deliveryDate, Valid: true},
				Paid:         true,
				ModifiedBy:   1,
				Action:       oModel.ActionUpdate,
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "HAPPY PATH: Deleting an order history",
			payload: oModel.OrderHistory{
				IDOrder:      1,
				IdUser:       1,
				Status:       oModel.StatusCancelled,
				Price:        50.0,
				Note:         "Cancelled order",
				DeliveryDate: sql.NullTime{Time: deliveryDate, Valid: true},
				Paid:         false,
				ModifiedBy:   1,
				Action:       oModel.ActionDelete,
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "ERROR PATH: Database error",
			payload: oModel.OrderHistory{
				IDOrder:      1,
				IdUser:       1,
				Status:       oModel.StatusPending,
				Price:        50.0,
				Note:         "Test order",
				DeliveryDate: sql.NullTime{Time: deliveryDate, Valid: true},
				Paid:         false,
				ModifiedBy:   1,
				Action:       oModel.ActionCreate,
			},
			mockError:     assert.AnError,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta(
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
							$1,
							$2, 
							$3, 
							$4, 
							$5,
							$6,
							$7,
							$8, 
							$9)`,
					),
				).WillReturnError(tt.mockError)
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta(
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
							$1,
							$2, 
							$3, 
							$4, 
							$5,
							$6,
							$7,
							$8, 
							$9)`,
					),
				).WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err := repo.CreateOrderHistory(context.Background(), tt.payload)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
