package validators

import (
	"fmt"
	"strings"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// Agregar las pruebas unitarias
func ValidateCreateOrderPayload(payload oModel.CreateOrderPayload) error {
	if strings.TrimSpace(payload.Name) == "" {
		return fmt.Errorf("El campo 'name' es obligatorio")
	}
	if strings.TrimSpace(payload.Email) == "" {
		return fmt.Errorf("El campo 'email' es obligatorio")
	}
	if !IsValidEmail(payload.Email) {
		return fmt.Errorf("El campo 'email' no tiene un formato válido")
	}
	if strings.TrimSpace(payload.Phone) == "" {
		return fmt.Errorf("El campo 'phone' es obligatorio")
	}
	if strings.TrimSpace(payload.DeliveryDate) == "" {
		return fmt.Errorf("El campo 'delivery_date' es obligatorio")
	}
	if len(payload.Items) == 0 {
		return fmt.Errorf("Debe incluir al menos un producto en 'items'")
	}
	for i, item := range payload.Items {
		if item.IdProduct == 0 {
			return fmt.Errorf("El producto en la posición %d tiene un 'id_product' inválido", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("El producto en la posición %d tiene una 'quantity' inválida", i)
		}
	}
	return nil
}
