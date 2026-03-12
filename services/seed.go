package services

import (
	"fmt"
	"math/rand"
	"time"

	"bekend/config"
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

var promotionPackageTemplates = []struct {
	Name          models.PromotionPackageName
	Price         float64
	DurationDays  int
	Description   string
}{
	{models.PromotionPackageTop, 499, 7, "Событие отображается в топе списка на главной странице"},
	{models.PromotionPackageRecommended, 299, 14, "Событие попадает в блок «Рекомендовано для вас»"},
	{models.PromotionPackageHot, 199, 3, "Событие отображается в блоке «Горящие события»"},
}

func ensurePromotionPackages() {
	for _, p := range promotionPackageTemplates {
		var existing models.PromotionPackage
		if database.DB.Where("name = ?", p.Name).First(&existing).Error != nil {
			pack := models.PromotionPackage{
				ID:           uuid.New(),
				Name:         p.Name,
				Price:        p.Price,
				DurationDays: p.DurationDays,
				Description:  p.Description,
			}
			database.DB.Create(&pack)
		}
	}
}

func RunSeed() {
	log := logger.GetLogger()
	rand.Seed(time.Now().UnixNano())

	var adminUser models.User
	if err := database.DB.Where("role = ? AND status = ?", models.RoleAdmin, models.UserStatusActive).First(&adminUser).Error; err != nil {
		log.Warn("Seed: администратор не найден, пропуск сида")
		return
	}

	// Сид пакетов продвижения
	ensurePromotionPackages()

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

// RunSeedFull — полный сид: 100-150 пользователей, 100-120 афиш (при SEED_FULL=true)
func RunSeedFull() {
	log := logger.GetLogger()
	if !config.AppConfig.SeedFull {
		return
	}
	rand.Seed(time.Now().UnixNano())

	var adminUser models.User
	if database.DB.Where("role = ? AND status = ?", models.RoleAdmin, models.UserStatusActive).First(&adminUser).Error != nil {
		log.Warn("SeedFull: администратор не найден, пропуск")
		return
	}

	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count >= 100 {
		log.Info("SeedFull: достаточно пользователей, пропуск")
		return
	}

	firstNames := []string{"Александр", "Дмитрий", "Максим", "Сергей", "Андрей", "Алексей", "Анна", "Мария", "Елена", "Ольга"}
	lastNames := []string{"Иванов", "Петров", "Сидоров", "Козлов", "Смирнов", "Волков", "Лебедев", "Новиков", "Морозов", "Кузнецов"}
	eventTitlesFull := []string{"Рок-концерт", "Футбольный матч", "Выставка искусства", "Мастер-класс", "Кулинарный фестиваль", "Марафон", "Джазовый вечер", "Йога в парке", "Театральная премьера", "IT-конференция"}
	addressesFull := []string{"Москва, Парк Горького", "Москва, Лужники", "Москва, Парк Сокольники", "Москва, Воробьёвы горы", "Москва, центр"}
	latlngsFull := [][2]float64{{55.7308, 37.6014}, {55.7158, 37.5538}, {55.7415, 37.6208}, {55.7558, 37.6173}, {55.7942, 37.6794}}

	users := ensureTestUsers()
	seenEmails := make(map[string]bool)
	for _, u := range users {
		seenEmails[u.Email] = true
	}

	for len(users) < 130 {
		fn := firstNames[rand.Intn(len(firstNames))]
		ln := lastNames[rand.Intn(len(lastNames))]
		email := fmt.Sprintf("user%d@test.local", 1000+rand.Intn(9000))
		if seenEmails[email] {
			continue
		}
		seenEmails[email] = true
		var existing models.User
		if database.DB.Where("email = ?", email).First(&existing).Error == nil {
			users = append(users, existing)
			continue
		}
		hash, _ := utils.HashPassword("Test123!")
		u := models.User{
			ID: uuid.New(), FullName: fn + " " + ln, Email: email, Password: hash,
			Role: models.RoleUser, Status: models.UserStatusActive, EmailVerified: true, AuthProvider: "email",
		}
		database.DB.Create(&u)
		users = append(users, u)
	}

	var eventCount int64
	database.DB.Model(&models.Event{}).Count(&eventCount)
	for eventCount < 110 {
		title := eventTemplates[rand.Intn(len(eventTemplates))].Title
		if eventCount > 10 {
			title = fmt.Sprintf("%s #%d", eventTitlesFull[rand.Intn(len(eventTitlesFull))], rand.Intn(999))
		}
		shortDesc := eventTemplates[rand.Intn(len(eventTemplates))].ShortDesc
		t := eventTemplates[rand.Intn(len(eventTemplates))]
		addrIdx := rand.Intn(len(addressesFull))
		addr := addressesFull[addrIdx]
		lat, lng := latlngsFull[addrIdx%len(latlngsFull)][0], latlngsFull[addrIdx%len(latlngsFull)][1]

		daysOffset := rand.Intn(60) - 10
		now := time.Now()
		startDate := now.AddDate(0, 0, daysOffset).Add(time.Hour * time.Duration(10+rand.Intn(8)))
		endDate := startDate.Add(time.Hour * time.Duration(2+rand.Intn(4)))

		var status models.EventStatus
		if daysOffset < 0 {
			status = models.EventStatusPast
		} else {
			status = models.EventStatusActive
		}

		organizer := users[rand.Intn(len(users))]
		maxP := 20 + rand.Intn(80)
		imgURL := "/uploads/events/placeholder.jpg"
		if rand.Intn(3) == 0 && len(eventImageMap) > 0 {
			imgURL = eventImageMap[rand.Intn(len(eventImageMap))]
		}

		ev := models.Event{
			ID:               uuid.New(),
			Title:            title,
			ShortDescription: shortDesc,
			FullDescription:  t.FullDesc,
			StartDate:        startDate,
			EndDate:          endDate,
			ImageURL:         imgURL,
			PaymentInfo:      t.PaymentInfo,
			MaxParticipants:  &maxP,
			Status:           status,
			OrganizerID:      organizer.ID,
			Tags:             models.StringArray(t.Tags),
			Address:          addr,
			Latitude:         &lat,
			Longitude:        &lng,
			YandexMapLink:    fmt.Sprintf("https://yandex.ru/maps/?pt=%.6f,%.6f&z=16", lng, lat),
		}
		if database.DB.Create(&ev).Error != nil {
			continue
		}
		eventCount++

		for _, name := range t.CategoryNames {
			var cat models.Category
			if database.DB.Where("name = ?", name).First(&cat).Error == nil {
				database.DB.Model(&ev).Association("Categories").Append(&cat)
			}
		}

		for j := 0; j < rand.Intn(maxP/3)+1 && j < len(users); j++ {
			u := users[rand.Intn(len(users))]
			if u.ID != organizer.ID {
				var ep models.EventParticipant
				if database.DB.Where("event_id = ? AND user_id = ?", ev.ID, u.ID).First(&ep).Error != nil {
					database.DB.Create(&models.EventParticipant{ID: uuid.New(), EventID: ev.ID, UserID: u.ID})
				}
			}
		}
	}

	log.Info("SeedFull: создано пользователей и афиш", zap.Int("users", len(users)), zap.Int64("events", eventCount))
}
