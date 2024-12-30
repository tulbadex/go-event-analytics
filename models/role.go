package models

import "github.com/google/uuid"

type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"unique;not null"`
}

// type UserRole struct {
// 	UserID uuid.UUID `gorm:"type:uuid;not null"`
// 	RoleID uint      `gorm:"not null"`
// }

type UserRole struct {
    UserID uuid.UUID `gorm:"type:uuid;not null;primaryKey;references:ID;constraint:OnDelete:CASCADE"`
    RoleID uint      `gorm:"not null;primaryKey;references:ID;constraint:OnDelete:CASCADE"`
}
