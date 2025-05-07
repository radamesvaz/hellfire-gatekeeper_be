package model

type OrderItems struct {
	ID        uint64 `json:"id_order_item" gorm:"primaryKey"`
	IdOrder   uint64 `json:"id_order" gorm:"not null;unique"`
	IdProduct uint64 `json:"id_product" gorm:"not null;unique"`
	Name      string `json:"name"`
	Quantity  uint64 `json:"quantity"`
}
