package orders

import (
	"context"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

type OrderCreator interface {
	Create(ctx context.Context, payload oModel.CreateOrderPayload) (uint64, error)
}
