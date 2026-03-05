package repository

import (
	"event-analytics/models"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(event *models.Event) error {
	return r.db.Create(event).Error
}

func (r *EventRepository) FindByID(id uint) (*models.Event, error) {
	var event models.Event
	err := r.db.First(&event, id).Error
	return &event, err
}

func (r *EventRepository) FindAll(limit, offset int) ([]models.Event, error) {
	var events []models.Event
	err := r.db.Limit(limit).Offset(offset).Find(&events).Error
	return events, err
}

func (r *EventRepository) FindByUserID(userID uint, limit, offset int) ([]models.Event, error) {
	var events []models.Event
	err := r.db.Where("created_by = ?", userID).Limit(limit).Offset(offset).Find(&events).Error
	return events, err
}

func (r *EventRepository) Update(event *models.Event) error {
	return r.db.Save(event).Error
}

func (r *EventRepository) Delete(id uint) error {
	return r.db.Delete(&models.Event{}, id).Error
}

func (r *EventRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Event{}).Count(&count).Error
	return count, err
}
