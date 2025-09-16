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
	Paid         bool        `json:"paid" gorm:"default:false"`
}

type OrderResponse struct {
	ID           uint64      `json:"id_order" gorm:"primaryKey"`
	IdUser       uint64      `json:"id_user" gorm:"not null"`
	User         string      `json:"user_name" gorm:"not null;unique"`
	Status       OrderStatus `json:"status"`
	Price        float64     `json:"total_price" gorm:"not null;check:price >= 0"`
	Note         string      `json:"note"`
	OrderItems   []OrderItems
	CreatedOn    time.Time `json:"created_on"`
	DeliveryDate time.Time `json:"delivery_date"`
	Paid         bool      `json:"paid"`
}

type CreateOrderPayload struct {
	Name         string                 `json:"name"`
	Email        string                 `json:"email"`
	Phone        string                 `json:"phone"`
	DeliveryDate string                 `json:"delivery_date"`
	Note         string                 `json:"note"`
	Items        []CreateOrderItemInput `json:"items"`
}

type CreateOrderRequest struct {
	IdUser       uint64      `json:"id_user" gorm:"not null;unique"`
	DeliveryDate time.Time   `json:"delivery_date" validate:"required,datetime=2006-01-02"`
	Note         string      `json:"note"`
	Price        float64     `json:"total_price" gorm:"not null;check:price >= 0"`
	Status       OrderStatus `json:"status"`
	Paid         bool        `json:"paid" gorm:"default:false"`
}

type CreateFullOrder struct {
	IdUser       uint64      `json:"id_user" gorm:"not null;unique"`
	DeliveryDate time.Time   `json:"delivery_date" validate:"required,datetime=2006-01-02"`
	Note         string      `json:"note"`
	Price        float64     `json:"total_price" gorm:"not null;check:price >= 0"`
	Status       OrderStatus `json:"status"`
	Paid         bool        `json:"paid" gorm:"default:false"`
	OrderItems   []OrderItemRequest
}
