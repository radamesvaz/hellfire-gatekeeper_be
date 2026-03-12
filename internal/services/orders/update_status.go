package orders

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/logger"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// OrderStatusRepository defines the interface for order status operations
type OrderStatusRepository interface {
	GetOrderByID(ctx context.Context, tenantID, id uint64) (oModel.OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error
	CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error
	GetOrderItemsByOrderID(ctx context.Context, tenantID, orderID uint64) ([]oModel.OrderItems, error)
}

// ProductStockRepository defines the interface for product stock operations
type ProductStockRepository interface {
	RevertProductStock(ctx context.Context, idProduct uint64, quantityToRevert uint64) error
}

type StatusUpdaterWithStock struct {
	OrderRepo   OrderStatusRepository
	ProductRepo ProductStockRepository
}

func NewStatusUpdaterWithStock(orderRepo OrderStatusRepository, productRepo ProductStockRepository) *StatusUpdaterWithStock {
	return &StatusUpdaterWithStock{
		OrderRepo:   orderRepo,
		ProductRepo: productRepo,
	}
}

func (s *StatusUpdaterWithStock) validateStatusTransition(currentStatus, newStatus oModel.OrderStatus) error {
	// Allow all status transitions - no restrictions
	return nil
}

// UpdateOrderStatusWithStockReversion updates order status and reverts stock if admin cancels order.
// cancellationReason is optional; only used when newStatus is cancelled (e.g. user-provided reason or nil).
func (s *StatusUpdaterWithStock) UpdateOrderStatusWithStockReversion(ctx context.Context, tenantID, orderID uint64, newStatus oModel.OrderStatus, userID uint64, isAdmin bool, cancellationReason *string) error {
	// Get the current order
	order, err := s.OrderRepo.GetOrderByID(ctx, tenantID, orderID)
	if err != nil {
		return err
	}

	// Validate status transition
	if err := s.validateStatusTransition(order.Status, newStatus); err != nil {
		return err
	}

	var effectiveCancellationReason *string
	if newStatus == oModel.StatusCancelled {
		effectiveCancellationReason = cancellationReason
	}
	err = s.OrderRepo.UpdateOrderStatus(ctx, tenantID, orderID, newStatus, effectiveCancellationReason)
	if err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	// If admin is cancelling the order, revert the stock
	if isAdmin && newStatus == oModel.StatusCancelled {
		err = s.revertOrderStock(ctx, tenantID, orderID)
		if err != nil {
			// If stock reversion fails, we should return the error
			// This is because the order status has already been updated
			logger.Warn().Err(err).
				Uint64("order_id", orderID).
				Msg("Failed to revert stock for cancelled order")
			return fmt.Errorf("error reverting stock for cancelled order: %w", err)
		}
	}

	// Create order history record (IdUser nil when order's user was deleted)
	var orderHistoryIdUser *uint64
	if order.IdUser != 0 {
		orderHistoryIdUser = &order.IdUser
	}
	orderHistory := oModel.OrderHistory{
		IDOrder: orderID,
		IdUser:  orderHistoryIdUser,
		Status:  newStatus,
		Price:   order.Price,
		Note:    order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		Paid:               order.Paid,
		CancellationReason: effectiveCancellationReason,
		ModifiedBy:         userID,
		Action:             oModel.ActionUpdate,
	}

	err = s.OrderRepo.CreateOrderHistory(ctx, orderHistory)
	if err != nil {
		// Log the error but don't fail the status update
		logger.Warn().Err(err).
			Uint64("order_id", orderID).
			Str("new_status", string(newStatus)).
			Msg("Failed to create order history")
	}

	return nil
}

// revertOrderStock reverts the stock for all items in an order
func (s *StatusUpdaterWithStock) revertOrderStock(ctx context.Context, tenantID, orderID uint64) error {
	// Get all items for the order
	items, err := s.OrderRepo.GetOrderItemsByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return fmt.Errorf("error getting order items: %w", err)
	}

	// Revert stock for each item
	for _, item := range items {
		err = s.ProductRepo.RevertProductStock(ctx, item.IdProduct, item.Quantity)
		if err != nil {
			return fmt.Errorf("error reverting stock for product %d: %w", item.IdProduct, err)
		}
	}

	return nil
}
