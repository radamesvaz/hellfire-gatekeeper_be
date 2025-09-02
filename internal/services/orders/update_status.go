package orders

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// OrderStatusRepository defines the interface for order status operations
type OrderStatusRepository interface {
	GetOrderByID(ctx context.Context, id uint64) (oModel.OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error
	CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error
}

type StatusUpdater struct {
	OrderRepo OrderStatusRepository
}

func NewStatusUpdater(orderRepo OrderStatusRepository) *StatusUpdater {
	return &StatusUpdater{
		OrderRepo: orderRepo,
	}
}

func (s *StatusUpdater) UpdateOrderStatus(ctx context.Context, orderID uint64, newStatus oModel.OrderStatus, userID uint64) error {
	// Get the current order
	order, err := s.OrderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate status transition
	if err := s.validateStatusTransition(order.Status, newStatus); err != nil {
		return err
	}

	// Update the order status
	err = s.OrderRepo.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	// Create order history record
	orderHistory := oModel.OrderHistory{
		IDOrder: orderID,
		IdUser:  order.IdUser,
		Status:  newStatus,
		Price:   order.Price,
		Note:    order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		ModifiedBy: userID,
		Action:     oModel.ActionUpdate,
	}

	err = s.OrderRepo.CreateOrderHistory(ctx, orderHistory)
	if err != nil {
		// Log the error but don't fail the status update
		fmt.Printf("Warning: failed to create order history: %v", err)
	}

	return nil
}

func (s *StatusUpdater) validateStatusTransition(currentStatus, newStatus oModel.OrderStatus) error {
	// Check if order is already in a final state
	if currentStatus == oModel.StatusCancelled {
		return errors.ErrOrderAlreadyCancelled
	}
	if currentStatus == oModel.StatusDelivered {
		return errors.ErrOrderAlreadyDelivered
	}

	// Define valid transitions based on actual model states
	validTransitions := map[oModel.OrderStatus][]oModel.OrderStatus{
		oModel.StatusPending: {
			oModel.StatusPreparing,
			oModel.StatusCancelled,
		},
		oModel.StatusPreparing: {
			oModel.StatusReady,
			oModel.StatusCancelled,
		},
		oModel.StatusReady: {
			oModel.StatusDelivered,
			oModel.StatusCancelled,
		},
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("invalid current status: %s", currentStatus)
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return nil
		}
	}

	return errors.ErrInvalidStatusTransition
}
