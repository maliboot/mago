package mago

import (
	"gorm.io/gorm"
)

type DataObject interface {
	DatabaseName() string
	PrimaryKey() string
	PrimaryValue() uint
	TableName() string
}

type AbstractDataObject struct {
	DataObject `gorm:"-:all" json:"-"`

	ID        int64          `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt DateTime       `json:"createdAt" gorm:"autoCreateTime;type:timestamp null;comment:创建时间"`
	UpdatedAt DateTime       `json:"updatedAt" gorm:"autoUpdateTime;type:timestamp null;comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index;type:timestamp null;comment:删除时间"`
}

func (do AbstractDataObject) PrimaryKey() string {
	return "id"
}

func (do AbstractDataObject) PrimaryValue() uint {
	return uint(do.ID)
}

func (do AbstractDataObject) TableName() string {
	panic("impl me")
}
