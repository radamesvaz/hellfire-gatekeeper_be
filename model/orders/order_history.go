package model

import (
	"database/sql"
)

type OrderAction string

const (
	ActionCreate OrderAction = "create"
	ActionUpdate OrderAction = "update"
	ActionDelete OrderAction = "delete"
)

type OrderHistory struct {
	ID                 uint64       `json:"id_order_history" gorm:"primaryKey"`
	TenantID           uint64       `json:"tenant_id"`
	IDOrder            uint64       `json:"id_order" gorm:"not null"`
	IdUser             *uint64      `json:"id_user"` // nil when order's user was deleted (ON DELETE SET NULL)
	Status             OrderStatus  `json:"status"`
	Price              float64      `json:"total_price" gorm:"not null;check:price >= 0"`
	Note               string       `json:"note"`
	DeliveryDirection  string       `json:"delivery_direction"`
	DeliveryDate       sql.NullTime `json:"delivery_date"`
	Paid               bool         `json:"paid"`
	CancellationReason *string      `json:"cancellation_reason,omitempty"`
	ModifiedOn         sql.NullTime `json:"modified_on"`
	ModifiedBy         uint64       `json:"modified_by"`
	Action             OrderAction  `json:"action"`
}
