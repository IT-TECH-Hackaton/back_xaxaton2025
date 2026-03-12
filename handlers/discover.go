package handlers

import (
	"math"
	"net/http"
	"time"

	"bekend/database"
	"bekend/dto"
	"bekend/models"
	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DiscoverHandler struct {
	logger *zap.Logger
}

func NewDiscoverHandler() *DiscoverHandler {
	return &DiscoverHandler{logger: utils.GetLogger()}
}

// HotEventResponse — событие с мета-данными для блока «Горящие»
type HotEventResponse struct {
	dto.EventResponse
	FillPercent float64 `json:"fillPercent"` // % заполненности мест
	HoursLeft   float64 `json:"hoursLeft"`   // часов до начала
}

// GetHotEvents godoc
// @Summary Горящие события
// @Description Активные события, которые начнутся в ближайшие 72 часа и заполнены на 50%+ или имеют платное продвижение «Горящее»
// @Tags Discover
// @Produce json
// @Success 200 {array} HotEventResponse
// @Router /events/hot [get]
func (h *DiscoverHandler) GetHotEvents(c *gin.Context) {
	now := time.Now().UTC()
	deadline := now.Add(72 * time.Hour)

	// События с продвижением «Горящее» (приоритет, показываем даже при <50% заполненности)
	var promotedIDs []uuid.UUID
	database.DB.Model(&models.EventPromotion{}).
		Joins("JOIN promotion_packages ON promotion_packages.id = event_promotions.package_id").
		Joins("JOIN events ON events.id = event_promotions.event_id").
		Where("event_promotions.status = ? AND event_promotions.end_date > ? AND event_promotions.start_date <= ?",
			models.EventPromotionStatusActive, now, now).
		Where("promotion_packages.name = ?", models.PromotionPackageHot).
		Where("events.status = ? AND events.start_date > ? AND events.start_date <= ?",
			models.EventStatusActive, now, deadline).
		Pluck("event_promotions.event_id", &promotedIDs)

	promotedSet := make(map[uuid.UUID]bool)
	for _, id := range promotedIDs {
		promotedSet[id] = true
	}

	// Загружаем продвинутые события (могут быть без max_participants)
	var promotedEvents []models.Event
	if len(promotedIDs) > 0 {
		database.DB.
			Preload("Participants").
			Preload("Categories").
			Preload("Organizer").
			Where("id IN ?", promotedIDs).
			Order("start_date ASC").
			Find(&promotedEvents)
	}

	var events []models.Event
	if err := database.DB.
		Preload("Participants").
		Preload("Categories").
		Preload("Organizer").
		Where("status = ? AND start_date > ? AND start_date <= ?",
			models.EventStatusActive, now, deadline).
		Where("max_participants IS NOT NULL AND max_participants > 0").
		Order("start_date ASC").
		Limit(20).
		Find(&events).Error; err != nil {
		h.logger.Error("GetHotEvents: ошибка БД", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных"})
		return
	}

	// Объединяем: сначала продвинутые, потом обычные (без дублей)
	seen := make(map[uuid.UUID]bool)
	allEvents := make([]models.Event, 0, len(promotedEvents)+len(events))
	for _, ev := range promotedEvents {
		if !seen[ev.ID] {
			seen[ev.ID] = true
			allEvents = append(allEvents, ev)
		}
	}
	for _, ev := range events {
		if !seen[ev.ID] {
			seen[ev.ID] = true
			allEvents = append(allEvents, ev)
		}
	}

	result := make([]HotEventResponse, 0, 10)
	for _, ev := range allEvents {
		count := len(ev.Participants)
		fillPct := 0.0
		if ev.MaxParticipants != nil && *ev.MaxParticipants > 0 {
			fillPct = float64(count) / float64(*ev.MaxParticipants) * 100.0
		}
		// Пропускаем если не продвинуто и заполненность < 50%
		if !promotedSet[ev.ID] && fillPct < 50.0 {
			continue
		}
		if len(result) >= 10 {
			break
		}

		hoursLeft := math.Max(0, ev.StartDate.Sub(now).Hours())

		cats := make([]dto.CategoryInfo, len(ev.Categories))
		for i, cat := range ev.Categories {
			cats[i] = dto.CategoryInfo{ID: cat.ID.String(), Name: cat.Name}
		}

		tags := []string(ev.Tags)
		if tags == nil {
			tags = []string{}
		}

		er := dto.EventResponse{
			ID:               ev.ID.String(),
			Title:            ev.Title,
			ShortDescription: ev.ShortDescription,
			FullDescription:  ev.FullDescription,
			StartDate:        ev.StartDate,
			EndDate:          ev.EndDate,
			ImageURL:         ev.ImageURL,
			PaymentInfo:      ev.PaymentInfo,
			MaxParticipants:  ev.MaxParticipants,
			Status:           string(ev.Status),
			ParticipantsCount: count,
			Categories:       cats,
			Tags:             tags,
			Address:          ev.Address,
			Latitude:         ev.Latitude,
			Longitude:        ev.Longitude,
			YandexMapLink:    ev.YandexMapLink,
		}

		result = append(result, HotEventResponse{
			EventResponse: er,
			FillPercent:   math.Round(fillPct*10) / 10,
			HoursLeft:     math.Round(hoursLeft*10) / 10,
		})
	}

	c.JSON(http.StatusOK, result)
}

// RecommendedEventResponse — событие с оценкой релевантности
type RecommendedEventResponse struct {
	dto.EventResponse
	RelevanceScore   float64  `json:"relevanceScore"`   // 0..100
	MatchedInterests []string `json:"matchedInterests"` // совпавшие интересы
}

// GetRecommendedEvents godoc
// @Summary Рекомендованные события
// @Description Персональные рекомендации на основе интересов пользователя
// @Tags Discover
// @Security BearerAuth
// @Produce json
// @Success 200 {array} RecommendedEventResponse
// @Failure 401 {object} map[string]string
// @Router /events/recommended [get]
func (h *DiscoverHandler) GetRecommendedEvents(c *gin.Context) {
	rawID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}
	userID := rawID.(uuid.UUID)

	// 1. Загрузить интересы пользователя
	var userInterests []models.UserInterest
	database.DB.Preload("Interest").Where("user_id = ?", userID).Find(&userInterests)

	// 2. Собрать имена и веса интересов, категорий
	interestNames := make(map[string]int)
	for _, ui := range userInterests {
		if ui.Interest.Name != "" {
			interestNames[ui.Interest.Name] = ui.Weight
		}
	}

	// 3. Получить активные будущие события (включая с продвижением «Рекомендуемое»)
	now := time.Now().UTC()
	var promotedRecIDs []uuid.UUID
	database.DB.Model(&models.EventPromotion{}).
		Joins("JOIN promotion_packages ON promotion_packages.id = event_promotions.package_id").
		Where("event_promotions.status = ? AND event_promotions.end_date > ?",
			models.EventPromotionStatusActive, now).
		Where("promotion_packages.name = ?", models.PromotionPackageRecommended).
		Pluck("event_promotions.event_id", &promotedRecIDs)

	var events []models.Event
	database.DB.
		Preload("Participants").
		Preload("Categories").
		Preload("Organizer").
		Where("status = ? AND start_date > ?", models.EventStatusActive, now).
		Order("start_date ASC").
		Limit(50).
		Find(&events)

	promotedRecSet := make(map[uuid.UUID]bool)
	for _, id := range promotedRecIDs {
		promotedRecSet[id] = true
	}
	// Дозагружаем продвинутые события, которых нет в основном списке
	eventsSet := make(map[uuid.UUID]bool)
	for _, e := range events {
		eventsSet[e.ID] = true
	}
	for _, pid := range promotedRecIDs {
		if !eventsSet[pid] {
			var addEv models.Event
			if database.DB.Preload("Participants").Preload("Categories").Preload("Organizer").
				First(&addEv, "id = ? AND status = ? AND start_date > ?", pid, models.EventStatusActive, now).Error == nil {
				events = append(events, addEv)
			}
		}
	}

	// 4. Исключить события, в которых пользователь уже участвует
	var participating []models.EventParticipant
	database.DB.Where("user_id = ?", userID).Find(&participating)
	participatingSet := make(map[uuid.UUID]bool)
	for _, p := range participating {
		participatingSet[p.EventID] = true
	}

	// 5. Рассчитать релевантность
	type scored struct {
		ev    models.Event
		score float64
		names []string
	}
	var candidates []scored

	for _, ev := range events {
		if participatingSet[ev.ID] {
			continue
		}
		if len(interestNames) == 0 {
			// Нет интересов — рекомендуем самые популярные
			score := math.Min(float64(len(ev.Participants))*2, 50.0)
			candidates = append(candidates, scored{ev: ev, score: score, names: []string{}})
			continue
		}

		var matched []string
		totalScore := 0.0

		// Совпадение по тегам
		for _, tag := range ev.Tags {
			if w, ok := interestNames[tag]; ok {
				matched = append(matched, tag)
				totalScore += float64(w) * 2.0
			}
		}

		// Совпадение по категориям
		for _, cat := range ev.Categories {
			if w, ok := interestNames[cat.Name]; ok {
				matched = append(matched, cat.Name)
				totalScore += float64(w) * 3.0
			}
		}

		if totalScore == 0 && len(interestNames) > 0 {
			continue
		}

		// Нормализация: max возможный вклад = 5 (интересов) * max_weight(10) * 3 = 150
		normalized := math.Min(totalScore/150.0*100.0, 100.0)
		// Поп-бонус: +5 за каждые 10 участников, но не более +20
		popBonus := math.Min(float64(len(ev.Participants))/10.0*5.0, 20.0)
		finalScore := math.Min(normalized+popBonus, 100.0)
		// Бонус за платное продвижение «Рекомендуемое»
		if promotedRecSet[ev.ID] {
			finalScore = math.Min(finalScore+30, 100.0)
		}

		candidates = append(candidates, scored{ev: ev, score: math.Round(finalScore*10) / 10, names: matched})
	}

	// 6. Сортировка по убыванию score
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].score > candidates[j-1].score; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
	if len(candidates) > 6 {
		candidates = candidates[:6]
	}

	result := make([]RecommendedEventResponse, 0, len(candidates))
	for _, c2 := range candidates {
		ev := c2.ev
		cats := make([]dto.CategoryInfo, len(ev.Categories))
		for i, cat := range ev.Categories {
			cats[i] = dto.CategoryInfo{ID: cat.ID.String(), Name: cat.Name}
		}
		tags := []string(ev.Tags)
		if tags == nil {
			tags = []string{}
		}

		names := c2.names
		if names == nil {
			names = []string{}
		}

		er := dto.EventResponse{
			ID:               ev.ID.String(),
			Title:            ev.Title,
			ShortDescription: ev.ShortDescription,
			FullDescription:  ev.FullDescription,
			StartDate:        ev.StartDate,
			EndDate:          ev.EndDate,
			ImageURL:         ev.ImageURL,
			PaymentInfo:      ev.PaymentInfo,
			MaxParticipants:  ev.MaxParticipants,
			Status:           string(ev.Status),
			ParticipantsCount: len(ev.Participants),
			Categories:       cats,
			Tags:             tags,
			Address:          ev.Address,
			Latitude:         ev.Latitude,
			Longitude:        ev.Longitude,
			YandexMapLink:    ev.YandexMapLink,
		}

		result = append(result, RecommendedEventResponse{
			EventResponse:    er,
			RelevanceScore:   c2.score,
			MatchedInterests: names,
		})
	}

	c.JSON(http.StatusOK, result)
}
