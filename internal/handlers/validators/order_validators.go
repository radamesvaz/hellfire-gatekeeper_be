package validators

import (
	"fmt"
	"strings"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// Agregar las pruebas unitarias
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
	if len(payload.Items) == 0 {
		return fmt.Errorf("An item must be sent for the order")
	}
	for i, item := range payload.Items {
		if item.IdProduct == 0 {
			return fmt.Errorf("The product at position %d has an invalid quantity", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("The product at position %d has an invalid quantity", i)
		}
	}
	return nil
}
