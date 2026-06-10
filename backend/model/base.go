package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 通用主键与时间戳
type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// SoftDeleteModel 带软删除
type SoftDeleteModel struct {
	BaseModel
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
