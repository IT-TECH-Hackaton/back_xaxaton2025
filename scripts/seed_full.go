// Скрипт для заполнения БД тестовыми данными: 100-150 пользователей, 100-120 афиш
// Запуск: go run ./scripts/seed_full.go
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"bekend/config"
	"bekend/database"
	"bekend/logger"
	"bekend/models"
	"bekend/utils"

	"github.com/google/uuid"
)

const (
	targetUsers  = 130
	targetEvents = 110
)

var firstNames = []string{
	"Александр", "Дмитрий", "Максим", "Сергей", "Андрей", "Алексей", "Артём", "Илья", "Кирилл", "Михаил",
	"Евгений", "Даниил", "Денис", "Николай", "Егор", "Иван", "Павел", "Роман", "Владимир", "Никита",
	"Анна", "Мария", "Елена", "Ольга", "Наталья", "Татьяна", "Ирина", "Светлана", "Екатерина", "Юлия",
	"Виктория", "Анастасия", "Полина", "Дарья", "Кристина", "Алина", "Валерия", "София", "Вероника", "Александра",
}

var lastNames = []string{
	"Иванов", "Петров", "Сидоров", "Козлов", "Смирнов", "Волков", "Лебедев", "Новиков", "Морозов", "Кузнецов",
	"Попов", "Васильев", "Соколов", "Михайлов", "Федоров", "Фролов", "Алексеев", "Зайцев", "Степанов", "Николаев",
	"Орлов", "Андреев", "Макаров", "Павлов", "Егоров", "Семенов", "Голубев", "Виноградов", "Богданов", "Воробьев",
}

var eventTitles = []string{
	"Рок-концерт в парке", "Футбольный матч", "Выставка искусства", "Мастер-класс по программированию",
	"Кулинарный фестиваль", "Беговой марафон", "Джазовый вечер", "Йога в парке", "Театральная премьера",
	"IT-конференция", "Концерт живой музыки", "Кинофестиваль", "Фестиваль уличной еды", "Воркшоп по дизайну",
	"Научная лекция", "Танцевальный батл", "Спортивный турнир", "Встреча книжного клуба", "Фотосъёмка в парке",
	"Дегустация вин", "Квест по городу", "Выставка фотографии", "Показ мод", "Конференция стартапов",
	"Пикник с музыкой", "Вечер настольных игр", "Пленэр для художников", "Турнир по шахматам", "Презентация книги",
}

var shortDescs = []string{
	"Живая музыка под открытым небом", "Дерби московских клубов", "Работы молодых художников",
	"Изучение Go для начинающих", "Дегустация блюд со всего мира", "Городской марафон 42 км",
	"Живой джаз в уютной атмосфере", "Утренняя практика на свежем воздухе", "Новая постановка пьесы",
	"Конференция для разработчиков", "Концерт под звёздами", "Показ независимого кино",
	"Фуд-маркет с лучшими шеф-поварами", "Практика UI/UX дизайна", "Лекция о космосе",
	"Батл между танцевальными командами", "Соревнования по волейболу", "Обсуждение новой книги",
	"Совместная фотосессия", "Винный тур по регионам", "Приключенческий квест",
}

var addresses = []string{
	"Москва, Парк Горького", "Москва, Лужники", "Москва, Третьяковская галерея", "Москва, офис IT-компании",
	"Москва, Парк Сокольники", "Москва, Воробьёвы горы", "Москва, джаз-клуб", "Москва, Парк Зарядье",
	"Москва, Театр на Таганке", "Москва, конференц-центр", "Москва, Летний сад", "Москва, кинотеатр Октябрь",
	"Москва, фуд-маркет", "Москва, дизайн-студия", "Москва, планетарий", "Москва, танц-класс",
	"Москва, спортивный зал", "Москва, библиотека", "Москва, Нескучный сад", "Москва, винный бар",
	"Москва, центр города", "Москва, галерея", "Москва, шоурум", "Москва, коворкинг",
}

var latlngs = [][2]float64{
	{55.7308, 37.6014}, {55.7158, 37.5538}, {55.7415, 37.6208}, {55.7558, 37.6173}, {55.7942, 37.6794},
	{55.7108, 37.5533}, {55.7520, 37.6175}, {55.7514, 37.6188}, {55.7406, 37.6542}, {55.7558, 37.6173},
	{55.7489, 37.6194}, {55.7542, 37.6188}, {55.7520, 37.6175}, {55.7510, 37.6170}, {55.7558, 37.6173},
	{55.7520, 37.6175}, {55.7510, 37.6180}, {55.7515, 37.6190}, {55.7490, 37.6175}, {55.7525, 37.6180},
	{55.7530, 37.6178}, {55.7415, 37.6208}, {55.7520, 37.6175}, {55.7510, 37.6175},
}

var categoryNames = []string{
	"Концерты", "Спорт", "Искусство", "Образование", "Технологии", "Еда", "Фестивали", "Театр", "Музыка",
	"Футбол", "Джаз", "Бег", "Йога", "Выставки",
}

func intPtr(i int) *int { return &i }

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.AppEnv)
	defer logger.Sync()
	database.Connect()
	rand.Seed(time.Now().UnixNano())

	log.Println("=== Запуск полного сида: пользователи и афиши ===")

	// 1. Админ и категории
	var adminUser models.User
	if database.DB.Where("role = ? AND status = ?", models.RoleAdmin, models.UserStatusActive).First(&adminUser).Error != nil {
		log.Fatal("Администратор не найден. Сначала запустите приложение для создания админа.")
	}

	categoryMap := make(map[string]uuid.UUID)
	for _, name := range categoryNames {
		var cat models.Category
		if database.DB.Where("name = ?", name).First(&cat).Error != nil {
			cat = models.Category{ID: uuid.New(), Name: name, Description: name}
			database.DB.Create(&cat)
		}
		categoryMap[name] = cat.ID
	}

	// 2. Создаём 100-150 пользователей
	users := make([]models.User, 0, targetUsers)
	seenEmails := make(map[string]bool)
	for len(users) < targetUsers {
		fn := firstNames[rand.Intn(len(firstNames))]
		ln := lastNames[rand.Intn(len(lastNames))]
		email := fmt.Sprintf("%s.%s.%d@test.local", toLatin(fn), toLatin(ln), rand.Intn(9999))
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
	log.Printf("Создано пользователей: %d", len(users))

	// 3. Создаём 100-120 афиш
	now := time.Now()
	eventsCreated := 0
	for i := 0; i < targetEvents; i++ {
		titleIdx := rand.Intn(len(eventTitles))
		title := eventTitles[titleIdx]
		if i > 20 {
			title = fmt.Sprintf("%s #%d", title, i+1)
		}
		shortDesc := shortDescs[rand.Intn(len(shortDescs))]
		addrIdx := rand.Intn(len(addresses))
		addr := addresses[addrIdx]
		lat, lng := latlngs[addrIdx%len(latlngs)][0], latlngs[addrIdx%len(latlngs)][1]

		daysOffset := rand.Intn(60) - 10
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
		if titleIdx < 10 {
			imgs := []string{"event-rock.jpg", "event-football.jpg", "event-art.jpg", "event-coding.jpg", "event-food.jpg"}
			imgURL = "/uploads/events/" + imgs[titleIdx%len(imgs)]
		}

		ev := models.Event{
			ID:               uuid.New(),
			Title:            title,
			ShortDescription: shortDesc,
			FullDescription:  shortDesc + ". Подробная информация о мероприятии. Приходите, будет интересно!",
			StartDate:        startDate,
			EndDate:          endDate,
			ImageURL:         imgURL,
			PaymentInfo:      []string{"Бесплатно", "От 200 рублей", "500-1000 рублей"}[rand.Intn(3)],
			MaxParticipants:  &maxP,
			Status:           status,
			OrganizerID:      organizer.ID,
			Tags:             models.StringArray([]string{"тег1", "тег2"}),
			Address:          addr,
			Latitude:         &lat,
			Longitude:        &lng,
			YandexMapLink:    fmt.Sprintf("https://yandex.ru/maps/?pt=%.6f,%.6f&z=16", lng, lat),
		}
		if err := database.DB.Create(&ev).Error; err != nil {
			log.Printf("Ошибка создания события: %v", err)
			continue
		}

		catName := categoryNames[rand.Intn(len(categoryNames))]
		if catID, ok := categoryMap[catName]; ok {
			var cat models.Category
			if database.DB.Where("id = ?", catID).First(&cat).Error == nil {
				database.DB.Model(&ev).Association("Categories").Append(&cat)
			}
		}

		participantsToAdd := rand.Intn(maxP/2) + 1
		if participantsToAdd > len(users) {
			participantsToAdd = len(users)
		}
		for j := 0; j < participantsToAdd; j++ {
			u := users[rand.Intn(len(users))]
			if u.ID != organizer.ID {
				var ep models.EventParticipant
				if database.DB.Where("event_id = ? AND user_id = ?", ev.ID, u.ID).First(&ep).Error != nil {
					database.DB.Create(&models.EventParticipant{ID: uuid.New(), EventID: ev.ID, UserID: u.ID})
				}
			}
		}
		eventsCreated++
	}

	log.Printf("Создано событий: %d", eventsCreated)
	log.Println("=== Сид завершён ===")
}

func toLatin(s string) string {
	trans := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z",
		'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r",
		'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
		'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "a", 'Б': "b", 'В': "v", 'Г': "g", 'Д': "d", 'Е': "e", 'Ё': "e", 'Ж': "zh", 'З': "z",
		'И': "i", 'Й': "y", 'К': "k", 'Л': "l", 'М': "m", 'Н': "n", 'О': "o", 'П': "p", 'Р': "r",
		'С': "s", 'Т': "t", 'У': "u", 'Ф': "f", 'Х': "h", 'Ц': "ts", 'Ч': "ch", 'Ш': "sh", 'Щ': "sch",
		'Э': "e", 'Ю': "yu", 'Я': "ya",
	}
	result := ""
	for _, r := range s {
		if low, ok := trans[r]; ok {
			result += low
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if r >= 'a' && r <= 'z' {
			result += string(r)
		}
	}
	return result
}
