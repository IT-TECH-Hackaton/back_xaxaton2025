package services

import (
	"time"

	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type CronService struct {
	emailService *EmailService
	logger       *zap.Logger
}

func NewCronService() *CronService {
	return &CronService{
		emailService: NewEmailService(),
		logger:       utils.GetLogger(),
	}
}

func (cs *CronService) Start() {
	c := cron.New()

	c.AddFunc("@hourly", cs.UpdateEventStatuses)
	c.AddFunc("@daily", cs.SendEventReminders)

	c.Start()
	cs.logger.Info("Cron jobs started")
}

func (cs *CronService) UpdateEventStatuses() {
	now := time.Now()

	var events []models.Event
	if err := database.DB.Where("status = ? AND end_date < ?", models.EventStatusActive, now).Find(&events).Error; err != nil {
		cs.logger.Error("Ошибка обновления статусов событий", zap.Error(err))
		return
	}

	for _, event := range events {
		event.Status = models.EventStatusPast
		if err := database.DB.Save(&event).Error; err != nil {
			cs.logger.Error("Ошибка обновления статуса события",
				zap.String("eventID", event.ID.String()),
				zap.Error(err),
			)
		}
	}

	var activeEvents []models.Event
	if err := database.DB.Where("status = ? AND start_date <= ? AND end_date >= ?", models.EventStatusActive, now, now).Find(&activeEvents).Error; err == nil {
		for _, event := range activeEvents {
			if event.Status != models.EventStatusActive {
				event.Status = models.EventStatusActive
				database.DB.Save(&event)
			}
		}
	}

	cs.logger.Info("Обновлены статусы событий", zap.Int("count", len(events)))
}

func (cs *CronService) SendEventReminders() {
	now := time.Now()
	reminderTime := now.Add(24 * time.Hour)

	var events []models.Event
	if err := database.DB.Preload("Participants.User").Where("status = ? AND start_date BETWEEN ? AND ?", models.EventStatusActive, now, reminderTime).Find(&events).Error; err != nil {
		cs.logger.Error("Ошибка поиска событий для напоминаний", zap.Error(err))
		return
	}

	for _, event := range events {
		for _, participant := range event.Participants {
			if err := cs.emailService.SendEventNotification(
				participant.User.Email,
				event.Title,
				"Напоминание: событие начнется через 24 часа",
			); err != nil {
				cs.logger.Error("Ошибка отправки напоминания",
					zap.String("email", participant.User.Email),
					zap.String("eventID", event.ID.String()),
					zap.Error(err),
				)
			}
		}
	}

	cs.logger.Info("Отправлены напоминания о событиях", zap.Int("count", len(events)))
}

