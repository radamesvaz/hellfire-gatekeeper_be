package model

type Roles struct {
	ID   int    `json:"id_role" gorm:"primaryKey"`
	Name string `json:"name" gorm:"not null;unique"`
}
