package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"bekend/config"
	"bekend/database"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

var eventTemplates = []struct {
	Title           string
	ShortDesc       string
	FullDesc        string
	CategoryNames   []string
	Tags            []string
	Address         string
	Latitude        float64
	Longitude       float64
	PaymentInfo     string
	MaxParticipants *int
	ImageURL        string
}{
	{
		Title:         "Рок-концерт в парке Горького",
		ShortDesc:     "Живая музыка под открытым небом",
		FullDesc:      "Масштабный рок-концерт с участием популярных групп. Живая музыка, отличная атмосфера и незабываемые эмоции. Приходите всей семьей!",
		CategoryNames: []string{"Концерты", "Музыка"},
		Tags:          []string{"рок", "музыка", "концерт", "живая музыка"},
		Address:       "Москва, Парк Горького, Центральная аллея",
		Latitude:      55.7308,
		Longitude:     37.6014,
		PaymentInfo:   "Бесплатно",
		ImageURL:      "https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=800&h=600&fit=crop",
	},
	{
		Title:         "Футбольный матч: Спартак vs ЦСКА",
		ShortDesc:     "Дерби московских клубов",
		FullDesc:      "Легендарное московское дерби. Два сильнейших клуба столицы сойдутся в поединке за победу. Не пропустите это зрелищное событие!",
		CategoryNames: []string{"Спорт", "Футбол"},
		Tags:          []string{"футбол", "спорт", "дерби", "Спартак", "ЦСКА"},
		Address:       "Москва, Лужники, Большая спортивная арена",
		Latitude:      55.7158,
		Longitude:     37.5538,
		PaymentInfo:   "От 500 до 5000 рублей",
		MaxParticipants: intPtr(50000),
		ImageURL:      "https://images.unsplash.com/photo-1574629810360-7efbbe195018?w=800&h=600&fit=crop",
	},
	{
		Title:         "Выставка современного искусства",
		ShortDesc:     "Работы молодых художников",
		FullDesc:      "Экспозиция работ современных российских художников. Инсталляции, картины, скульптуры. Уникальная возможность познакомиться с актуальным искусством.",
		CategoryNames: []string{"Искусство", "Выставки"},
		Tags:          []string{"искусство", "выставка", "живопись", "современное искусство"},
		Address:       "Москва, Третьяковская галерея",
		Latitude:      55.7415,
		Longitude:     37.6208,
		PaymentInfo:   "500 рублей, льготы 250 рублей",
		MaxParticipants: intPtr(200),
		ImageURL:      "https://images.unsplash.com/photo-1541961017774-22349e4a1262?w=800&h=600&fit=crop",
	},
	{
		Title:         "Мастер-класс по программированию",
		ShortDesc:     "Изучение Go для начинающих",
		FullDesc:      "Практический мастер-класс по программированию на языке Go. Разберем основы, напишем несколько программ. Подходит для начинающих.",
		CategoryNames: []string{"Образование", "Технологии"},
		Tags:          []string{"программирование", "Go", "обучение", "IT"},
		Address:       "Москва, офис IT-компании",
		Latitude:      55.7558,
		Longitude:     37.6173,
		PaymentInfo:   "Бесплатно",
		MaxParticipants: intPtr(30),
		ImageURL:      "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&h=600&fit=crop",
	},
	{
		Title:         "Кулинарный фестиваль",
		ShortDesc:     "Дегустация блюд со всего мира",
		FullDesc:      "Фестиваль кухни разных стран. Мастер-классы от шеф-поваров, дегустации, конкурсы. Приходите попробовать что-то новое!",
		CategoryNames: []string{"Еда", "Фестивали"},
		Tags:          []string{"еда", "кулинария", "фестиваль", "дегустация"},
		Address:       "Москва, Парк Сокольники",
		Latitude:      55.7942,
		Longitude:     37.6794,
		PaymentInfo:   "Вход свободный, дегустации от 200 рублей",
		ImageURL:      "https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=800&h=600&fit=crop",
	},
	{
		Title:         "Беговой марафон",
		ShortDesc:     "Городской марафон 42 км",
		FullDesc:      "Ежегодный городской марафон. Дистанции: 5 км, 10 км, 21 км, 42 км. Регистрация обязательна. Награждение победителей.",
		CategoryNames: []string{"Спорт", "Бег"},
		Tags:          []string{"бег", "марафон", "спорт", "здоровье"},
		Address:       "Москва, старт на Воробьевых горах",
		Latitude:      55.7108,
		Longitude:     37.5533,
		PaymentInfo:   "Регистрация 1000 рублей",
		MaxParticipants: intPtr(5000),
		ImageURL:      "https://images.unsplash.com/photo-1571008887538-b36bb32f4571?w=800&h=600&fit=crop",
	},
	{
		Title:         "Джазовый вечер",
		ShortDesc:     "Живой джаз в уютной атмосфере",
		FullDesc:      "Вечер джазовой музыки. Выступление известных джазовых музыкантов. Уютная атмосфера, отличная музыка и напитки.",
		CategoryNames: []string{"Концерты", "Джаз"},
		Tags:          []string{"джаз", "музыка", "концерт", "вечер"},
		Address:       "Москва, джаз-клуб",
		Latitude:      55.7520,
		Longitude:     37.6175,
		PaymentInfo:   "1500 рублей",
		MaxParticipants: intPtr(100),
		ImageURL:      "https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800&h=600&fit=crop",
	},
	{
		Title:         "Йога в парке",
		ShortDesc:     "Утренняя практика на свежем воздухе",
		FullDesc:      "Групповое занятие йогой в парке. Подходит для всех уровней подготовки. Принесите коврик и хорошее настроение!",
		CategoryNames: []string{"Спорт", "Йога"},
		Tags:          []string{"йога", "здоровье", "спорт", "релаксация"},
		Address:       "Москва, Парк Сокольники",
		Latitude:      55.7942,
		Longitude:     37.6794,
		PaymentInfo:   "Бесплатно",
		MaxParticipants: intPtr(50),
		ImageURL:      "https://images.unsplash.com/photo-1506126613408-eca07ce68773?w=800&h=600&fit=crop",
	},
	{
		Title:         "Театральная премьера",
		ShortDesc:     "Новая постановка современной пьесы",
		FullDesc:      "Премьера спектакля по пьесе современного драматурга. Режиссер - лауреат театральных премий. Не пропустите!",
		CategoryNames: []string{"Театр", "Искусство"},
		Tags:          []string{"театр", "спектакль", "премьера", "культура"},
		Address:       "Москва, Театр на Таганке",
		Latitude:      55.7406,
		Longitude:     37.6542,
		PaymentInfo:   "От 800 до 3000 рублей",
		MaxParticipants: intPtr(500),
		ImageURL:      "https://images.unsplash.com/photo-1503095396549-807759245b35?w=800&h=600&fit=crop",
	},
	{
		Title:         "IT-конференция",
		ShortDesc:     "Конференция для разработчиков",
		FullDesc:      "Ежегодная конференция для IT-специалистов. Доклады о новых технологиях, нетворкинг, обмен опытом. Регистрация обязательна.",
		CategoryNames: []string{"Технологии", "Образование"},
		Tags:          []string{"IT", "конференция", "технологии", "разработка"},
		Address:       "Москва, конференц-центр",
		Latitude:      55.7558,
		Longitude:     37.6173,
		PaymentInfo:   "Ранняя регистрация 3000 рублей, стандартная 5000 рублей",
		MaxParticipants: intPtr(1000),
		ImageURL:      "https://images.unsplash.com/photo-1540575467063-178a55c61e40?w=800&h=600&fit=crop",
	},
}

var categoryTemplates = []struct {
	Name        string
	Description string
}{
	{"Концерты", "Музыкальные мероприятия и выступления"},
	{"Спорт", "Спортивные события и соревнования"},
	{"Искусство", "Выставки, перформансы, культурные мероприятия"},
	{"Образование", "Лекции, мастер-классы, курсы"},
	{"Технологии", "IT-события, конференции, хакатоны"},
	{"Еда", "Кулинарные мероприятия и фестивали"},
	{"Фестивали", "Многодневные культурные события"},
	{"Театр", "Театральные постановки и спектакли"},
	{"Музыка", "Музыкальные события различных жанров"},
	{"Футбол", "Футбольные матчи и турниры"},
	{"Джаз", "Джазовые концерты и выступления"},
	{"Бег", "Беговые события и марафоны"},
	{"Йога", "Йога-практики и занятия"},
	{"Выставки", "Художественные и тематические выставки"},
}

func intPtr(i int) *int {
	return &i
}

func main() {
	config.LoadConfig()
	database.Connect()

	fmt.Println("🌱 Начало заполнения базы данных тестовыми данными...")
	fmt.Println()

	var adminUser models.User
	err := database.DB.Where("role = ? AND status = ?", models.RoleAdmin, models.UserStatusActive).First(&adminUser).Error
	if err != nil {
		fmt.Println("⚠️  Активный администратор не найден. Попытка найти любого администратора...")
		
		err = database.DB.Where("role = ?", models.RoleAdmin).First(&adminUser).Error
		if err != nil {
			fmt.Println("⚠️  Администратор не найден. Создание администратора по умолчанию...")
			
			hashedPassword, hashErr := utils.HashPassword("Admin123!")
			if hashErr != nil {
				log.Fatal("Ошибка хеширования пароля: ", hashErr)
			}

			adminUser = models.User{
				ID:            uuid.New(),
				FullName:      "Администратор",
				Email:         "admin@admin.com",
				Password:      hashedPassword,
				Role:          models.RoleAdmin,
				Status:        models.UserStatusActive,
				EmailVerified: true,
				AuthProvider:  "email",
			}

			if createErr := database.DB.Create(&adminUser).Error; createErr != nil {
				log.Fatal("Ошибка создания администратора: ", createErr)
			}
			
			fmt.Println("✅ Администратор создан:", adminUser.Email)
		} else {
			if adminUser.Status == models.UserStatusDeleted {
				fmt.Println("⚠️  Найден удаленный администратор. Восстановление...")
				adminUser.Status = models.UserStatusActive
				if updateErr := database.DB.Save(&adminUser).Error; updateErr != nil {
					log.Fatal("Ошибка восстановления администратора: ", updateErr)
				}
				fmt.Println("✅ Администратор восстановлен:", adminUser.Email)
			} else {
				fmt.Println("✅ Найден администратор:", adminUser.Email, "(статус:", adminUser.Status, ")")
			}
		}
	} else {
		fmt.Println("✅ Найден администратор:", adminUser.Email)
	}
	fmt.Println()

	fmt.Println("📂 Создание категорий...")
	categoryMap := make(map[string]uuid.UUID)

	for _, catTemplate := range categoryTemplates {
		var category models.Category
		if err := database.DB.Where("name = ?", catTemplate.Name).First(&category).Error; err != nil {
			category = models.Category{
				ID:          uuid.New(),
				Name:        catTemplate.Name,
				Description: catTemplate.Description,
			}
			if err := database.DB.Create(&category).Error; err != nil {
				log.Printf("Ошибка создания категории %s: %v", catTemplate.Name, err)
				continue
			}
			fmt.Printf("  ✅ Создана категория: %s\n", catTemplate.Name)
		} else {
			fmt.Printf("  ℹ️  Категория уже существует: %s\n", catTemplate.Name)
		}
		categoryMap[catTemplate.Name] = category.ID
	}
	fmt.Println()

	fmt.Println("🎭 Создание событий...")
	rand.Seed(time.Now().UnixNano())

	now := time.Now()
	createdCount := 0
	skippedCount := 0

	for i, template := range eventTemplates {
		var existingEvent models.Event
		if err := database.DB.Where("title = ?", template.Title).First(&existingEvent).Error; err == nil {
			// Удаляем существующее событие, чтобы пересоздать с правильными датами
			if delErr := database.DB.Unscoped().Delete(&existingEvent).Error; delErr != nil {
				log.Printf("Ошибка удаления существующего события %s: %v", template.Title, delErr)
			} else {
				fmt.Printf("  🔄 Удалено существующее событие: %s (будет пересоздано)\n", template.Title)
			}
		}

		// Создаем события с датами в будущем для активных событий
		// Первые 7 событий - активные (в будущем)
		// Следующие 2 - прошедшие (в прошлом)
		// Последнее 1 - отклоненное (в прошлом)
		var daysOffset int
		var status models.EventStatus
		
		if i < 7 {
			// Активные события - в будущем
			daysOffset = i + 1 // 1, 2, 3, 4, 5, 6, 7 дней вперед
			status = models.EventStatusActive
		} else if i < 9 {
			// Прошедшие события - в прошлом
			daysOffset = -(i - 6) // -1, -2 дня назад
			status = models.EventStatusPast
		} else {
			// Отклоненное событие - в прошлом
			daysOffset = -3
			status = models.EventStatusRejected
		}
		
		hour := 10 + i*2
		if hour >= 24 {
			hour = hour % 24
		}
		startDate := now.AddDate(0, 0, daysOffset).Add(time.Hour * time.Duration(hour)).Add(time.Minute * time.Duration(rand.Intn(60)))
		durationHours := 2 + rand.Intn(4)
		endDate := startDate.Add(time.Hour * time.Duration(durationHours))

		imageURL := template.ImageURL
		if imageURL == "" {
			imageURL = fmt.Sprintf("/uploads/events/placeholder_%d.jpg", i+1)
		}

		event := models.Event{
			ID:              uuid.New(),
			Title:            template.Title,
			ShortDescription: template.ShortDesc,
			FullDescription:  template.FullDesc,
			StartDate:        startDate,
			EndDate:          endDate,
			ImageURL:         imageURL,
			PaymentInfo:      template.PaymentInfo,
			MaxParticipants:  template.MaxParticipants,
			Status:           status,
			OrganizerID:      adminUser.ID,
			Tags:             models.StringArray(template.Tags),
			Address:          template.Address,
			Latitude:         &template.Latitude,
			Longitude:        &template.Longitude,
			YandexMapLink:    fmt.Sprintf("https://yandex.ru/maps/?pt=%.6f,%.6f&z=16", template.Longitude, template.Latitude),
		}

		if err := database.DB.Create(&event).Error; err != nil {
			log.Printf("Ошибка создания события %s: %v", template.Title, err)
			skippedCount++
			continue
		}

		var categoriesToAdd []models.Category
		for _, catName := range template.CategoryNames {
			if catID, exists := categoryMap[catName]; exists {
				var category models.Category
				if err := database.DB.Where("id = ?", catID).First(&category).Error; err == nil {
					categoriesToAdd = append(categoriesToAdd, category)
				}
			}
		}

		if len(categoriesToAdd) > 0 {
			if err := database.DB.Model(&event).Association("Categories").Append(categoriesToAdd); err != nil {
				log.Printf("Ошибка добавления категорий к событию %s: %v", template.Title, err)
			}
		}

		statusEmoji := "🟢"
		if status == models.EventStatusPast {
			statusEmoji = "⚫"
		} else if status == models.EventStatusRejected {
			statusEmoji = "🔴"
		}

		fmt.Printf("  %s Создано событие: %s (%s)\n", statusEmoji, template.Title, status)
		createdCount++
	}

	fmt.Println()
	fmt.Println("👥 Создание тестовых пользователей...")
	testUsers := []struct {
		FullName string
		Email    string
	}{
		{"Иван Петров", "ivan.petrov@test.local"},
		{"Мария Сидорова", "maria.sidorova@test.local"},
		{"Алексей Иванов", "alexey.ivanov@test.local"},
		{"Елена Козлова", "elena.kozlova@test.local"},
		{"Дмитрий Смирнов", "dmitry.smirnov@test.local"},
		{"Анна Волкова", "anna.volkova@test.local"},
		{"Сергей Лебедев", "sergey.lebedev@test.local"},
		{"Ольга Новикова", "olga.novikova@test.local"},
	}

	createdUsers := make([]models.User, 0)
	for _, userData := range testUsers {
		var existingUser models.User
		if err := database.DB.Where("email = ?", userData.Email).First(&existingUser).Error; err != nil {
			hashedPassword, hashErr := utils.HashPassword("Test123!")
			if hashErr != nil {
				log.Printf("Ошибка хеширования пароля для %s: %v", userData.Email, hashErr)
				continue
			}

			newUser := models.User{
				ID:            uuid.New(),
				FullName:      userData.FullName,
				Email:         userData.Email,
				Password:      hashedPassword,
				Role:          models.RoleUser,
				Status:        models.UserStatusActive,
				EmailVerified: true,
				AuthProvider:  "email",
			}

			if createErr := database.DB.Create(&newUser).Error; createErr != nil {
				log.Printf("Ошибка создания пользователя %s: %v", userData.Email, createErr)
				continue
			}

			createdUsers = append(createdUsers, newUser)
			fmt.Printf("  ✅ Создан пользователь: %s (%s)\n", userData.FullName, userData.Email)
		} else {
			createdUsers = append(createdUsers, existingUser)
			fmt.Printf("  ℹ️  Пользователь уже существует: %s\n", userData.FullName)
		}
	}
	fmt.Println()

	if len(createdUsers) > 0 {
		fmt.Println("🎫 Добавление участников к событиям...")
		var allEvents []models.Event
		if err := database.DB.Find(&allEvents).Error; err != nil {
			log.Printf("Ошибка получения событий: %v", err)
		} else {
			for _, event := range allEvents {
				var participantsCount int
				if event.Status == models.EventStatusActive {
					participantsCount = rand.Intn(5) + 1
				} else if event.Status == models.EventStatusPast {
					participantsCount = rand.Intn(4) + 2
				} else {
					continue
				}

				if participantsCount > len(createdUsers) {
					participantsCount = len(createdUsers)
				}

				if participantsCount == 0 {
					continue
				}

				userIndices := rand.Perm(len(createdUsers))[:participantsCount]
				addedCount := 0

				for _, idx := range userIndices {
					user := createdUsers[idx]
					var existingParticipant models.EventParticipant
					if err := database.DB.Where("event_id = ? AND user_id = ?", event.ID, user.ID).First(&existingParticipant).Error; err != nil {
						participant := models.EventParticipant{
							ID:      uuid.New(),
							EventID: event.ID,
							UserID:  user.ID,
						}
						if err := database.DB.Create(&participant).Error; err != nil {
							log.Printf("Ошибка добавления участника %s к событию %s: %v", user.FullName, event.Title, err)
						} else {
							addedCount++
						}
					}
				}

				if addedCount > 0 {
					statusEmoji := "🟢"
					if event.Status == models.EventStatusPast {
						statusEmoji = "⚫"
					}
					fmt.Printf("  %s Добавлено %d участников к событию: %s\n", statusEmoji, addedCount, event.Title)
				}
			}
		}
		fmt.Println()

		fmt.Println("⭐ Создание отзывов для прошедших событий...")
		var pastEvents []models.Event
		if err := database.DB.Where("status = ?", models.EventStatusPast).Find(&pastEvents).Error; err != nil {
			log.Printf("Ошибка получения прошедших событий: %v", err)
		} else {
			reviewComments := []string{
				"Отличное событие! Очень понравилось, обязательно приду еще раз.",
				"Хорошая организация, интересная программа. Рекомендую!",
				"Впечатления отличные! Спасибо организаторам за такое мероприятие.",
				"Было здорово! Очень интересно и познавательно.",
				"Прекрасное событие, получил много положительных эмоций.",
				"Мероприятие прошло на высшем уровне. Очень доволен!",
				"Отличная атмосфера, замечательные люди. Все супер!",
				"Не ожидал, что будет так интересно. Восхищен!",
				"Очень понравилось, жду следующих подобных мероприятий.",
				"Отличная организация, все было на высоте.",
				"Было немного скучновато, но в целом неплохо.",
				"Неплохое мероприятие, но есть куда расти.",
				"Хорошее событие, но ожидал большего.",
				"Средненько, ничего особенного.",
				"Мероприятие прошло нормально, но не более того.",
			}

			reviewsCreated := 0
			for _, event := range pastEvents {
				var participants []models.EventParticipant
				if err := database.DB.Where("event_id = ?", event.ID).Find(&participants).Error; err != nil {
					continue
				}

				if len(participants) == 0 {
					continue
				}

				maxReviews := len(participants) / 2
				if maxReviews < 1 && len(participants) > 0 {
					maxReviews = 1
				}
				
				minReviews := maxReviews / 2
				if minReviews < 1 {
					minReviews = 1
				}
				
				reviewsCount := rand.Intn(maxReviews-minReviews+1) + minReviews
				if reviewsCount > len(participants) {
					reviewsCount = len(participants) / 2
					if reviewsCount < 1 {
						reviewsCount = 1
					}
				}

				participantIndices := rand.Perm(len(participants))[:reviewsCount]
				eventReviewsCreated := 0
				eventParticipantsWithoutReview := len(participants) - reviewsCount

				for _, idx := range participantIndices {
					participant := participants[idx]
					
					var existingReview models.EventReview
					if err := database.DB.Where("event_id = ? AND user_id = ?", event.ID, participant.UserID).First(&existingReview).Error; err == nil {
						continue
					}

					rating := rand.Intn(3) + 3
					if rand.Float32() < 0.2 {
						rating = rand.Intn(2) + 1
					}

					comment := reviewComments[rand.Intn(len(reviewComments))]
					if rating < 3 && rand.Float32() < 0.5 {
						comment = reviewComments[rand.Intn(5) + 10]
					}

					review := models.EventReview{
						ID:      uuid.New(),
						EventID: event.ID,
						UserID:  participant.UserID,
						Rating:  rating,
						Comment: comment,
					}

					if err := database.DB.Create(&review).Error; err != nil {
						log.Printf("Ошибка создания отзыва для события %s: %v", event.Title, err)
					} else {
						eventReviewsCreated++
						reviewsCreated++
					}
				}

				if eventReviewsCreated > 0 {
					fmt.Printf("  ⭐ Создано %d отзывов из %d участников для события: %s\n", eventReviewsCreated, len(participants), event.Title)
					if eventParticipantsWithoutReview > 0 {
						fmt.Printf("     (Осталось %d участников без отзывов для тестирования)\n", eventParticipantsWithoutReview)
					}
				}
			}

			if reviewsCreated > 0 {
				fmt.Printf("  ✅ Всего создано отзывов: %d\n", reviewsCreated)
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("✅ Готово! Создано событий: %d, пропущено: %d\n", createdCount, skippedCount)
	fmt.Println()
	fmt.Println("📊 Статистика:")
	
	var activeCount, pastCount, rejectedCount int64
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusActive).Count(&activeCount)
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusPast).Count(&pastCount)
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusRejected).Count(&rejectedCount)
	
	fmt.Printf("  🟢 Активных: %d\n", activeCount)
	fmt.Printf("  ⚫ Прошедших: %d\n", pastCount)
	fmt.Printf("  🔴 Отклоненных: %d\n", rejectedCount)
	
	var totalCategories int64
	database.DB.Model(&models.Category{}).Count(&totalCategories)
	fmt.Printf("  📂 Категорий: %d\n", totalCategories)
}

