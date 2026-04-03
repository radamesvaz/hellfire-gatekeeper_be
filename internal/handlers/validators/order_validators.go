package validators

import (
	"fmt"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

func ValidateCreateOrderPayload(payload oModel.CreateOrderPayload) error {
	if strings.TrimSpace(payload.Name) == "" {
		return fmt.Errorf("The 'name' field is mandatory")
	}
	if strings.TrimSpace(payload.Email) == "" {
		return fmt.Errorf("The 'email' field is mandatory")
	}
	if !IsValidEmail(payload.Email) {
		return fmt.Errorf("The 'email' field has no valid format")
	}
	if strings.TrimSpace(payload.Phone) == "" {
		return fmt.Errorf("The 'phone' field is mandatory")
	}
	if strings.TrimSpace(payload.DeliveryDate) == "" {
		return fmt.Errorf("The 'delivery_date' field is mandatory")
	}
	if strings.TrimSpace(payload.DeliveryDirection) == "" {
		return errors.ErrMissingDeliveryDirection
	}
	if len(payload.Items) == 0 {
		return fmt.Errorf("An item must be sent for the order")
	}
	for i, item := range payload.Items {
		if item.IdProduct == 0 {
			return fmt.Errorf("The product at position %d has an invalid ID", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("The product at position %d has an invalid quantity", i)
		}
	}
	return nil
}

// ValidateOrderListStatusFilter returns an error if s is not a known order status (for GET list filters).
func ValidateOrderListStatusFilter(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	switch oModel.OrderStatus(s) {
	case oModel.StatusPending,
		oModel.StatusPreparing,
		oModel.StatusReady,
		oModel.StatusDelivered,
		oModel.StatusCancelled,
		oModel.StatusExpired,
		oModel.StatusDeleted:
		return nil
	default:
		return errors.NewBadRequest(fmt.Errorf("invalid status filter"))
	}
}
