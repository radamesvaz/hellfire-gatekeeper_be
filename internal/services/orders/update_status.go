package orders

import (
	"context"
	"database/sql"
	"fmt"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// OrderStatusRepository defines the interface for order status operations
type OrderStatusRepository interface {
	GetOrderByID(ctx context.Context, id uint64) (oModel.OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error
	CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error
	GetOrderItemsByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderItems, error)
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

// UpdateOrderStatusWithStockReversion updates order status and reverts stock if admin cancels order
func (s *StatusUpdaterWithStock) UpdateOrderStatusWithStockReversion(ctx context.Context, orderID uint64, newStatus oModel.OrderStatus, userID uint64, isAdmin bool) error {
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

	// If admin is cancelling the order, revert the stock
	if isAdmin && newStatus == oModel.StatusCancelled {
		err = s.revertOrderStock(ctx, orderID)
		if err != nil {
			// If stock reversion fails, we should return the error
			// This is because the order status has already been updated
			fmt.Printf("Warning: failed to revert stock for cancelled order %d: %v", orderID, err)
			return fmt.Errorf("error reverting stock for cancelled order: %w", err)
		}
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

// revertOrderStock reverts the stock for all items in an order
func (s *StatusUpdaterWithStock) revertOrderStock(ctx context.Context, orderID uint64) error {
	// Get all items for the order
	items, err := s.OrderRepo.GetOrderItemsByOrderID(ctx, orderID)
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
