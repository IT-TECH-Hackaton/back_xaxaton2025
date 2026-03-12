package services

import (
	"fmt"
	"math/rand"
	"time"

	"bekend/database"
	"bekend/logger"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var eventImageMap = []string{
	"/uploads/events/event-rock.jpg",
	"/uploads/events/event-football.jpg",
	"/uploads/events/event-art.jpg",
	"/uploads/events/event-coding.jpg",
	"/uploads/events/event-food.jpg",
	"/uploads/events/event-marathon.jpg",
	"/uploads/events/event-jazz.jpg",
	"/uploads/events/event-yoga.jpg",
	"/uploads/events/event-theater.jpg",
	"/uploads/events/event-it.jpg",
}

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
}{
	{"Рок-концерт в парке Горького", "Живая музыка под открытым небом", "Масштабный рок-концерт с участием популярных групп.", []string{"Концерты", "Музыка"}, []string{"рок", "музыка", "концерт"}, "Москва, Парк Горького", 55.7308, 37.6014, "Бесплатно", intPtr(18)},
	{"Футбольный матч: Спартак vs ЦСКА", "Дерби московских клубов", "Легендарное московское дерби.", []string{"Спорт", "Футбол"}, []string{"футбол", "спорт", "дерби"}, "Москва, Лужники", 55.7158, 37.5538, "От 500 до 5000 рублей", intPtr(16)},
	{"Выставка современного искусства", "Работы молодых художников", "Экспозиция работ современных российских художников.", []string{"Искусство", "Выставки"}, []string{"искусство", "выставка"}, "Москва, Третьяковская галерея", 55.7415, 37.6208, "500 рублей", intPtr(18)},
	{"Мастер-класс по программированию", "Изучение Go для начинающих", "Практический мастер-класс по Go.", []string{"Образование", "Технологии"}, []string{"программирование", "Go", "IT"}, "Москва, офис IT-компании", 55.7558, 37.6173, "Бесплатно", intPtr(30)},
	{"Кулинарный фестиваль", "Дегустация блюд со всего мира", "Фестиваль кухни разных стран.", []string{"Еда", "Фестивали"}, []string{"еда", "кулинария", "фестиваль"}, "Москва, Парк Сокольники", 55.7942, 37.6794, "Вход свободный", intPtr(100)},
	{"Беговой марафон", "Городской марафон 42 км", "Ежегодный городской марафон.", []string{"Спорт", "Бег"}, []string{"бег", "марафон", "спорт"}, "Москва, Воробьёвы горы", 55.7108, 37.5533, "Регистрация 1000 рублей", intPtr(500)},
	{"Джазовый вечер", "Живой джаз в уютной атмосфере", "Вечер джазовой музыки.", []string{"Концерты", "Джаз"}, []string{"джаз", "музыка", "концерт"}, "Москва, джаз-клуб", 55.7520, 37.6175, "1500 рублей", intPtr(50)},
	{"Йога в парке", "Утренняя практика на свежем воздухе", "Групповое занятие йогой в парке.", []string{"Спорт", "Йога"}, []string{"йога", "здоровье", "спорт"}, "Москва, Парк Сокольники", 55.7942, 37.6794, "Бесплатно", intPtr(50)},
	{"Театральная премьера", "Новая постановка современной пьесы", "Премьера спектакля.", []string{"Театр", "Искусство"}, []string{"театр", "спектакль", "культура"}, "Москва, Театр на Таганке", 55.7406, 37.6542, "От 800 до 3000 рублей", intPtr(100)},
	{"IT-конференция", "Конференция для разработчиков", "Ежегодная конференция для IT-специалистов.", []string{"Технологии", "Образование"}, []string{"IT", "конференция", "технологии"}, "Москва, конференц-центр", 55.7558, 37.6173, "3000-5000 рублей", intPtr(200)},
}

var categoryTemplates = []struct {
	Name        string
	Description string
}{
	{"Концерты", "Музыкальные мероприятия"}, {"Спорт", "Спортивные события"}, {"Искусство", "Выставки и культура"},
	{"Образование", "Лекции и мастер-классы"}, {"Технологии", "IT-события"}, {"Еда", "Кулинарные мероприятия"},
	{"Фестивали", "Культурные события"}, {"Театр", "Театральные постановки"}, {"Музыка", "Музыкальные события"},
	{"Футбол", "Футбольные матчи"}, {"Джаз", "Джазовые концерты"}, {"Бег", "Беговые события"}, {"Йога", "Йога-практики"}, {"Выставки", "Художественные выставки"},
}

func intPtr(i int) *int { return &i }

func RunSeed() {
	log := logger.GetLogger()
	rand.Seed(time.Now().UnixNano())

	var adminUser models.User
	if err := database.DB.Where("role = ? AND status = ?", models.RoleAdmin, models.UserStatusActive).First(&adminUser).Error; err != nil {
		log.Warn("Seed: администратор не найден, пропуск сида")
		return
	}

	categoryMap := make(map[string]uuid.UUID)
	for _, c := range categoryTemplates {
		var cat models.Category
		if err := database.DB.Where("name = ?", c.Name).First(&cat).Error; err != nil {
			cat = models.Category{ID: uuid.New(), Name: c.Name, Description: c.Description}
			database.DB.Create(&cat)
		}
		categoryMap[c.Name] = cat.ID
	}

	var count int64
	database.DB.Model(&models.Event{}).Count(&count)
	if count >= 5 {
		log.Info("Seed: события уже есть в БД, пропуск")
		return
	}

	now := time.Now()
	createdUsers := ensureTestUsers()

	for i, t := range eventTemplates {
		var daysOffset int
		var status models.EventStatus
		if i < 7 {
			daysOffset = i + 1
			status = models.EventStatusActive
		} else if i < 9 {
			daysOffset = -(i - 6)
			status = models.EventStatusPast
		} else {
			daysOffset = -3
			status = models.EventStatusRejected
		}

		imgURL := "/uploads/events/placeholder.jpg"
		if i < len(eventImageMap) {
			imgURL = eventImageMap[i]
		}

		startDate := now.AddDate(0, 0, daysOffset).Add(time.Hour * time.Duration(10+i%12))
		endDate := startDate.Add(time.Hour * 3)

		ev := models.Event{
			ID:               uuid.New(),
			Title:            t.Title,
			ShortDescription: t.ShortDesc,
			FullDescription:  t.FullDesc,
			StartDate:        startDate,
			EndDate:          endDate,
			ImageURL:         imgURL,
			PaymentInfo:      t.PaymentInfo,
			MaxParticipants:  t.MaxParticipants,
			Status:           status,
			OrganizerID:      adminUser.ID,
			Tags:             models.StringArray(t.Tags),
			Address:          t.Address,
			Latitude:         &t.Latitude,
			Longitude:        &t.Longitude,
			YandexMapLink:    fmt.Sprintf("https://yandex.ru/maps/?pt=%.6f,%.6f&z=16", t.Longitude, t.Latitude),
		}
		if err := database.DB.Create(&ev).Error; err != nil {
			log.Error("Seed: ошибка создания события", zap.String("title", t.Title), zap.Error(err))
			continue
		}

		for _, name := range t.CategoryNames {
			if id, ok := categoryMap[name]; ok {
				var cat models.Category
				if database.DB.Where("id = ?", id).First(&cat).Error == nil {
					database.DB.Model(&ev).Association("Categories").Append(&cat)
				}
			}
		}

		participantsToAdd := 0
		if status == models.EventStatusActive && len(createdUsers) > 0 {
			if t.MaxParticipants != nil && *t.MaxParticipants > 0 {
				targetFill := 0.5
				if i < 3 {
					targetFill = 0.55
				}
				participantsToAdd = int(float64(*t.MaxParticipants) * targetFill)
				if participantsToAdd > len(createdUsers) {
					participantsToAdd = len(createdUsers)
				}
				if participantsToAdd < 1 {
					participantsToAdd = 1
				}
			}
		} else if status == models.EventStatusPast {
			participantsToAdd = 2 + rand.Intn(4)
			if participantsToAdd > len(createdUsers) {
				participantsToAdd = len(createdUsers)
			}
		}

		for j := 0; j < participantsToAdd && j < len(createdUsers); j++ {
			u := createdUsers[j]
			var ep models.EventParticipant
			if database.DB.Where("event_id = ? AND user_id = ?", ev.ID, u.ID).First(&ep).Error != nil {
				database.DB.Create(&models.EventParticipant{ID: uuid.New(), EventID: ev.ID, UserID: u.ID})
			}
		}
	}

	log.Info("Seed: созданы мероприятия и тестовые данные")
}

func ensureTestUsers() []models.User {
	users := []struct {
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
		{"Иван Тестов", "user1@test.local"},
		{"Мария Тестова", "user2@test.local"},
	}

	var result []models.User
	for _, u := range users {
		var existing models.User
		if database.DB.Where("email = ?", u.Email).First(&existing).Error == nil {
			result = append(result, existing)
			continue
		}
		hash, _ := utils.HashPassword("Test123!")
		newUser := models.User{
			ID: uuid.New(), FullName: u.FullName, Email: u.Email, Password: hash,
			Role: models.RoleUser, Status: models.UserStatusActive, EmailVerified: true, AuthProvider: "email",
		}
		database.DB.Create(&newUser)
		result = append(result, newUser)
	}
	return result
}
