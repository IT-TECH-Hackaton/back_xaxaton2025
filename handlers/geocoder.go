package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"bekend/config"
	"github.com/gin-gonic/gin"
)

type GeocoderHandler struct{}

func NewGeocoderHandler() *GeocoderHandler {
	return &GeocoderHandler{}
}


type YandexGeocoderResponse struct {
	Response struct {
		GeoObjectCollection struct {
			FeatureMember []struct {
				GeoObject struct {
					Point struct {
						Pos string `json:"pos"`
					} `json:"Point"`
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Text string `json:"text"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

func (h *GeocoderHandler) GeocodeAddress(c *gin.Context) {
	var req dto.GeocodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if config.AppConfig.YandexGeocoderAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Яндекс.Геокодер не настроен"})
		return
	}

	// Вызов API Яндекс.Геокодера
	geocodeURL := fmt.Sprintf(
		"https://geocode-maps.yandex.ru/1.x/?apikey=%s&geocode=%s&format=json",
		config.AppConfig.YandexGeocoderAPIKey,
		url.QueryEscape(req.Address),
	)

	resp, err := http.Get(geocodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обращении к геокодеру"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении ответа геокодера"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка геокодера: " + string(body)})
		return
	}

	var geocodeResp YandexGeocoderResponse
	if err := json.Unmarshal(body, &geocodeResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга ответа геокодера"})
		return
	}

	if len(geocodeResp.Response.GeoObjectCollection.FeatureMember) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Адрес не найден"})
		return
	}

	// Получаем первый результат
	geoObject := geocodeResp.Response.GeoObjectCollection.FeatureMember[0].GeoObject
	
	// Парсим координаты (формат: "долгота широта")
	var longitude, latitude float64
	pos := geoObject.Point.Pos
	if _, err := fmt.Sscanf(pos, "%f %f", &longitude, &latitude); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга координат"})
		return
	}

	// Формируем ссылку на Яндекс.Карты
	yandexMapLink := fmt.Sprintf("https://yandex.ru/maps/?pt=%f,%f&z=16", longitude, latitude)

	response := dto.GeocodeResponse{
		Latitude:      latitude,
		Longitude:     longitude,
		Address:       geoObject.MetaDataProperty.GeocoderMetaData.Text,
		YandexMapLink: yandexMapLink,
	}

	c.JSON(http.StatusOK, response)
}

type ReverseGeocodeRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

func (h *GeocoderHandler) ReverseGeocode(c *gin.Context) {
	var req dto.ReverseGeocodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Широта должна быть от -90 до 90"})
		return
	}

	if req.Longitude < -180 || req.Longitude > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Долгота должна быть от -180 до 180"})
		return
	}

	if config.AppConfig.YandexGeocoderAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Яндекс.Геокодер не настроен"})
		return
	}

	// Обратный геокодинг (координаты -> адрес)
	geocodeURL := fmt.Sprintf(
		"https://geocode-maps.yandex.ru/1.x/?apikey=%s&geocode=%f,%f&format=json",
		config.AppConfig.YandexGeocoderAPIKey,
		req.Longitude,
		req.Latitude,
	)

	resp, err := http.Get(geocodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обращении к геокодеру"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении ответа геокодера"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка геокодера: " + string(body)})
		return
	}

	var geocodeResp YandexGeocoderResponse
	if err := json.Unmarshal(body, &geocodeResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга ответа геокодера"})
		return
	}

	if len(geocodeResp.Response.GeoObjectCollection.FeatureMember) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Адрес не найден для данных координат"})
		return
	}

	geoObject := geocodeResp.Response.GeoObjectCollection.FeatureMember[0].GeoObject
	
	// Формируем ссылку на Яндекс.Карты
	yandexMapLink := fmt.Sprintf("https://yandex.ru/maps/?pt=%f,%f&z=16", req.Longitude, req.Latitude)

	response := dto.GeocodeResponse{
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Address:       geoObject.MetaDataProperty.GeocoderMetaData.Text,
		YandexMapLink: yandexMapLink,
	}

	c.JSON(http.StatusOK, response)
}

func (h *GeocoderHandler) GenerateMapLink(c *gin.Context) {
	var req dto.MapLinkRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	var mapLink string

	if req.Latitude != nil && req.Longitude != nil {
		// Если есть координаты, используем их
		mapLink = fmt.Sprintf("https://yandex.ru/maps/?pt=%f,%f&z=16", *req.Longitude, *req.Latitude)
	} else if req.Address != "" {
		// Если есть адрес, формируем ссылку с адресом
		mapLink = fmt.Sprintf("https://yandex.ru/maps/?text=%s", url.QueryEscape(req.Address))
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать адрес или координаты"})
		return
	}

	c.JSON(http.StatusOK, dto.MapLinkResponse{
		YandexMapLink: mapLink,
	})
}

