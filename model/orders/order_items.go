package model

type OrderItems struct {
	ID        uint64 `json:"id_order_item" gorm:"primaryKey"`
	IdOrder   uint64 `json:"id_order" gorm:"not null;unique"`
	IdProduct uint64 `json:"id_product" gorm:"not null;unique"`
	Name      string `json:"name"`
	Quantity  uint64 `json:"quantity"`
}

type OrderItemRequest struct {
	IdProduct uint64 `json:"id_product" validate:"required"`
	IdOrder   uint64 `json:"id_order_item" gorm:"primaryKey"`
	Quantity  uint64 `json:"quantity" validate:"required,gt=0"`
}

type CreateOrderItemInput struct {
	IdProduct uint64 `json:"id_product"`
	Quantity  uint64 `json:"quantity"`
}
