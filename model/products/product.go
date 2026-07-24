package model

import (
	"database/sql"
)

type ProductStatus string

const (
	StatusActive   ProductStatus = "active"
	StatusInactive ProductStatus = "inactive"
	StatusDeleted  ProductStatus = "deleted"
)

type Product struct {
	ID             uint64        `json:"id_product" gorm:"primaryKey"`
	TenantID       uint64        `json:"tenant_id"`
	Name           string        `json:"name" gorm:"not null;unique"`
	Description    string        `json:"description"`
	Price          float64       `json:"price" gorm:"not null;check:price >= 0"`
	TrackInventory bool          `json:"track_inventory"`
	Stock          uint64        `json:"stock"`
	Status         ProductStatus `json:"status"`
	ImageURLs      []string      `json:"image_urls"`
	ThumbnailURL   string        `json:"thumbnail_url"`
	CreatedOn      sql.NullTime  `json:"created_on"`
}

type CreateProductRequest struct {
	Name           string        `form:"name" json:"name" gorm:"not null;unique"`
	Description    string        `form:"description" json:"description"`
	Price          float64       `form:"price" json:"price" gorm:"not null;check:price >= 0"`
	TrackInventory *bool         `form:"track_inventory" json:"track_inventory"`
	Stock          uint64        `form:"stock" json:"stock"`
	Status         ProductStatus `form:"status" json:"status"`
}

type UpdateProductRequest struct {
	Name           string        `form:"name" json:"name"`
	Description    string        `form:"description" json:"description"`
	Price          float64       `form:"price" json:"price"`
	TrackInventory *bool         `form:"track_inventory" json:"track_inventory"`
	Stock          uint64        `form:"stock" json:"stock"`
	Status         ProductStatus `form:"status" json:"status"`
}

type UpdateProductStatusRequest struct {
	Status ProductStatus `json:"status"`
}
