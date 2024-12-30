package services

import (
    "event-analytics/models"
    "event-analytics/config"
)

// CheckEventEditPermission checks if a user can edit an event
func CheckEventEditPermission(event *models.Event, user *models.User) bool {
    if user == nil {
        return false
    }

    // Check if user is the creator
    if event.CreatedBy == user.ID {
        return true
    }

    // Check if user is admin
    var count int64
    config.DB.Model(&models.Role{}).
        Joins("JOIN user_roles ON user_roles.role_id = roles.id").
        Where("user_roles.user_id = ? AND roles.name = ?", user.ID, "admin").
        Count(&count)

    return count > 0
}