package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

func main() {
	config.LoadConfig()
	database.Connect()

	fmt.Println("📝 Управление отзывами")
	fmt.Println("======================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("1. Список всех пользователей")
	fmt.Println("2. Список прошедших событий")
	fmt.Println("3. Создать отзыв от имени пользователя")
	fmt.Println("4. Обновить отзыв от имени пользователя")
	fmt.Println("5. Удалить отзыв от имени пользователя")
	fmt.Println("6. Список отзывов пользователя")
	fmt.Println()
	fmt.Print("Выберите действие (1-6): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		listUsers()
	case "2":
		listPastEvents()
	case "3":
		createReview(reader)
	case "4":
		updateReview(reader)
	case "5":
		deleteReview(reader)
	case "6":
		listUserReviews(reader)
	default:
		fmt.Println("Неверный выбор")
	}
}

func listUsers() {
	fmt.Println()
	fmt.Println("👥 Список пользователей:")
	fmt.Println()

	var users []models.User
	if err := database.DB.Where("status = ?", models.UserStatusActive).Find(&users).Error; err != nil {
		log.Fatal("Ошибка получения пользователей:", err)
	}

	if len(users) == 0 {
		fmt.Println("Пользователи не найдены")
		return
	}

	for i, user := range users {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, user.FullName, user.Email, user.Role)
	}
}

func listPastEvents() {
	fmt.Println()
	fmt.Println("⚫ Список прошедших событий:")
	fmt.Println()

	var events []models.Event
	if err := database.DB.Where("status = ?", models.EventStatusPast).Order("end_date DESC").Find(&events).Error; err != nil {
		log.Fatal("Ошибка получения событий:", err)
	}

	if len(events) == 0 {
		fmt.Println("Прошедшие события не найдены")
		return
	}

	for i, event := range events {
		var participantsCount int64
		database.DB.Model(&models.EventParticipant{}).Where("event_id = ?", event.ID).Count(&participantsCount)

		fmt.Printf("%d. %s\n", i+1, event.Title)
		fmt.Printf("   ID: %s\n", event.ID)
		fmt.Printf("   Дата: %s - %s\n", event.StartDate.Format("02.01.2006 15:04"), event.EndDate.Format("02.01.2006 15:04"))
		fmt.Printf("   Участников: %d\n", participantsCount)
		fmt.Println()
	}
}

func createReview(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("➕ Создание отзыва")

	userID := getUserIdInput(reader, "Введите email пользователя: ")
	eventID := getEventIdInput(reader, "Введите ID события (или название): ")

	var user models.User
	if err := database.DB.Where("id = ? AND status = ?", userID, models.UserStatusActive).First(&user).Error; err != nil {
		log.Fatal("Пользователь не найден или неактивен:", err)
	}

	var event models.Event
	if err := database.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		log.Fatal("Событие не найдено:", err)
	}

	if event.Status != models.EventStatusPast {
		log.Fatal("Отзыв можно оставить только для прошедших событий")
	}

	var participant models.EventParticipant
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		fmt.Printf("⚠️  Пользователь %s не участвовал в этом событии. Добавить как участника? (y/n): ", user.FullName)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "yes" || answer == "да" {
			participant = models.EventParticipant{
				ID:      uuid.New(),
				EventID: eventID,
				UserID:  userID,
			}
			if err := database.DB.Create(&participant).Error; err != nil {
				log.Fatal("Ошибка добавления участника:", err)
			}
			fmt.Println("✅ Пользователь добавлен как участник")
		} else {
			log.Fatal("Отзыв можно оставить только участникам события")
		}
	}

	var existingReview models.EventReview
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existingReview).Error; err == nil {
		log.Fatal("Пользователь уже оставил отзыв на это событие. Используйте обновление отзыва.")
	}

	fmt.Print("Введите рейтинг (1-5): ")
	ratingStr, _ := reader.ReadString('\n')
	rating, err := strconv.Atoi(strings.TrimSpace(ratingStr))
	if err != nil || rating < 1 || rating > 5 {
		log.Fatal("Рейтинг должен быть от 1 до 5")
	}

	fmt.Print("Введите комментарий (Enter для пропуска): ")
	comment, _ := reader.ReadString('\n')
	comment = strings.TrimSpace(comment)

	review := models.EventReview{
		ID:      uuid.New(),
		EventID: eventID,
		UserID:  userID,
		Rating:  rating,
		Comment: comment,
	}

	if err := database.DB.Create(&review).Error; err != nil {
		log.Fatal("Ошибка создания отзыва:", err)
	}

	fmt.Println()
	fmt.Println("✅ Отзыв успешно создан!")
	fmt.Printf("   Событие: %s\n", event.Title)
	fmt.Printf("   Пользователь: %s\n", user.FullName)
	fmt.Printf("   Рейтинг: %d ⭐\n", rating)
	if comment != "" {
		fmt.Printf("   Комментарий: %s\n", comment)
	}
}

func updateReview(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("✏️  Обновление отзыва")

	userID := getUserIdInput(reader, "Введите email пользователя: ")
	eventID := getEventIdInput(reader, "Введите ID события (или название): ")

	var review models.EventReview
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&review).Error; err != nil {
		log.Fatal("Отзыв не найден:", err)
	}

	database.DB.Preload("Event").Preload("User").First(&review, review.ID)

	fmt.Printf("Текущий отзыв:\n")
	fmt.Printf("  Событие: %s\n", review.Event.Title)
	fmt.Printf("  Пользователь: %s\n", review.User.FullName)
	fmt.Printf("  Рейтинг: %d ⭐\n", review.Rating)
	fmt.Printf("  Комментарий: %s\n", review.Comment)
	fmt.Println()

	fmt.Print("Новый рейтинг (1-5, Enter для пропуска): ")
	ratingStr, _ := reader.ReadString('\n')
	ratingStr = strings.TrimSpace(ratingStr)
	if ratingStr != "" {
		rating, err := strconv.Atoi(ratingStr)
		if err != nil || rating < 1 || rating > 5 {
			log.Fatal("Рейтинг должен быть от 1 до 5")
		}
		review.Rating = rating
	}

	fmt.Print("Новый комментарий (Enter для пропуска): ")
	comment, _ := reader.ReadString('\n')
	comment = strings.TrimSpace(comment)
	if comment != "" {
		review.Comment = comment
	}

	if err := database.DB.Save(&review).Error; err != nil {
		log.Fatal("Ошибка обновления отзыва:", err)
	}

	fmt.Println()
	fmt.Println("✅ Отзыв успешно обновлен!")
	fmt.Printf("   Рейтинг: %d ⭐\n", review.Rating)
	if review.Comment != "" {
		fmt.Printf("   Комментарий: %s\n", review.Comment)
	}
}

func deleteReview(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("🗑️  Удаление отзыва")

	userID := getUserIdInput(reader, "Введите email пользователя: ")
	eventID := getEventIdInput(reader, "Введите ID события (или название): ")

	var review models.EventReview
	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&review).Error; err != nil {
		log.Fatal("Отзыв не найден:", err)
	}

	database.DB.Preload("Event").Preload("User").First(&review, review.ID)

	fmt.Printf("Отзыв для удаления:\n")
	fmt.Printf("  Событие: %s\n", review.Event.Title)
	fmt.Printf("  Пользователь: %s\n", review.User.FullName)
	fmt.Printf("  Рейтинг: %d ⭐\n", review.Rating)
	fmt.Println()

	fmt.Print("Вы уверены? (y/n): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" && answer != "да" {
		fmt.Println("Отмена")
		return
	}

	if err := database.DB.Delete(&review).Error; err != nil {
		log.Fatal("Ошибка удаления отзыва:", err)
	}

	fmt.Println()
	fmt.Println("✅ Отзыв успешно удален!")
}

func listUserReviews(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("⭐ Список отзывов пользователя")

	userID := getUserIdInput(reader, "Введите email пользователя: ")

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		log.Fatal("Пользователь не найден:", err)
	}

	var reviews []models.EventReview
	if err := database.DB.Preload("Event").Where("user_id = ?", userID).Order("created_at DESC").Find(&reviews).Error; err != nil {
		log.Fatal("Ошибка получения отзывов:", err)
	}

	if len(reviews) == 0 {
		fmt.Printf("Пользователь %s еще не оставил отзывов\n", user.FullName)
		return
	}

	fmt.Printf("\nОтзывы пользователя %s (%s):\n\n", user.FullName, user.Email)

	for i, review := range reviews {
		fmt.Printf("%d. Событие: %s\n", i+1, review.Event.Title)
		fmt.Printf("   Рейтинг: %d ⭐\n", review.Rating)
		fmt.Printf("   Комментарий: %s\n", review.Comment)
		fmt.Printf("   Дата: %s\n", review.CreatedAt.Format("02.01.2006 15:04"))
		fmt.Println()
	}
}

func getUserIdInput(reader *bufio.Reader, prompt string) uuid.UUID {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var user models.User
	var err error

	if utils.ValidateUUID(input) {
		err = database.DB.Where("id = ?", input).First(&user).Error
	} else {
		err = database.DB.Where("email = ?", input).First(&user).Error
	}

	if err != nil {
		log.Fatal("Пользователь не найден:", err)
	}

	return user.ID
}

func getEventIdInput(reader *bufio.Reader, prompt string) uuid.UUID {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var event models.Event
	var err error

	if utils.ValidateUUID(input) {
		err = database.DB.Where("id = ?", input).First(&event).Error
	} else {
		err = database.DB.Where("title ILIKE ?", "%"+input+"%").First(&event).Error
		if err != nil {
			log.Fatal("Событие не найдено:", err)
		}
		if err == nil {
			var count int64
			database.DB.Model(&models.Event{}).Where("title ILIKE ?", "%"+input+"%").Count(&count)
			if count > 1 {
				fmt.Printf("Найдено несколько событий. Используется: %s (ID: %s)\n", event.Title, event.ID)
			}
		}
	}

	if err != nil {
		log.Fatal("Событие не найдено:", err)
	}

	return event.ID
}

