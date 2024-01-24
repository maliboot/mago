package mago

import (
	"gorm.io/gorm"
)

type DataObject interface {
	DatabaseName() string
	PrimaryKey() string
	PrimaryValue() uint
}

type AbstractDataObject struct {
	DataObject `gorm:"-:all" json:"-"`
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  DateTime       `json:"createdAt"`
	UpdatedAt  DateTime       `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (do AbstractDataObject) PrimaryKey() string {
	return "id"
}

func (do AbstractDataObject) PrimaryValue() uint {
	return do.ID
}
