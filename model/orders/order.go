package model

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPreparing OrderStatus = "preparing"
	StatusReady     OrderStatus = "ready"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID           uint64      `json:"id_order" gorm:"primaryKey"`
	IdUser       uint64      `json:"id_user" gorm:"not null;unique"`
	Status       OrderStatus `json:"status"`
	Price        float64     `json:"total_price" gorm:"not null;check:price >= 0"`
	Note         string      `json:"note"`
	CreatedOn    time.Time   `json:"created_on"`
	DeliveryDate time.Time   `json:"delivery_date"`
}

type OrderResponse struct {
	ID           uint64      `json:"id_order" gorm:"primaryKey"`
	User         string      `json:"id_user" gorm:"not null;unique"` // doble check
	Status       OrderStatus `json:"status"`
	Price        float64     `json:"total_price" gorm:"not null;check:price >= 0"`
	Note         string      `json:"note"`
	OrderItems   []OrderItems
	CreatedOn    time.Time `json:"created_on"`
	DeliveryDate time.Time `json:"delivery_date"`
}
