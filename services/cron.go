package services

import (
	"log"
	"time"

	"bekend/database"
	"bekend/models"

	"github.com/robfig/cron/v3"
)

type CronService struct {
	emailService *EmailService
}

func NewCronService() *CronService {
	return &CronService{
		emailService: NewEmailService(),
	}
}

func (cs *CronService) Start() {
	c := cron.New()

	c.AddFunc("@hourly", cs.UpdateEventStatuses)
	c.AddFunc("@daily", cs.SendEventReminders)

	c.Start()
	log.Println("Cron jobs started")
}

func (cs *CronService) UpdateEventStatuses() {
	now := time.Now()

	var events []models.Event
	if err := database.DB.Where("status = ? AND end_date < ?", models.EventStatusActive, now).Find(&events).Error; err != nil {
		log.Printf("Error updating event statuses: %v", err)
		return
	}

	for _, event := range events {
		event.Status = models.EventStatusPast
		if err := database.DB.Save(&event).Error; err != nil {
			log.Printf("Error updating event %s: %v", event.ID, err)
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

	log.Printf("Updated %d event statuses", len(events))
}

func (cs *CronService) SendEventReminders() {
	now := time.Now()
	reminderTime := now.Add(24 * time.Hour)

	var events []models.Event
	if err := database.DB.Preload("Participants.User").Where("status = ? AND start_date BETWEEN ? AND ?", models.EventStatusActive, now, reminderTime).Find(&events).Error; err != nil {
		log.Printf("Error finding events for reminders: %v", err)
		return
	}

	for _, event := range events {
		for _, participant := range event.Participants {
			if err := cs.emailService.SendEventNotification(
				participant.User.Email,
				event.Title,
				"Напоминание: событие начнется через 24 часа",
			); err != nil {
				log.Printf("Error sending reminder to %s: %v", participant.User.Email, err)
			}
		}
	}

	log.Printf("Sent reminders for %d events", len(events))
}

