package api

import (
	"log"
	"net/http"

	"golos/internal/service/gigachat"
	"golos/internal/storage"

	"github.com/gin-gonic/gin"
)

func (s *Server) indexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func (s *Server) voiceProcessHandler(c *gin.Context) {
	sessionID := c.Query("session_id")

	file, err := c.FormFile("audio")
	if err != nil {
		log.Printf("Ошибка получения файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось получить аудио файл"})
		return
	}

	if err := validateAudioFile(file); err != nil {
		log.Printf("Ошибка валидации файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	audioFile, err := file.Open()
	if err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось открыть файл"})
		return
	}
	defer audioFile.Close()

	log.Printf("Начало распознавания речи для сессии: %s", sessionID)
	text, err := s.audioClient.SpeechToText(audioFile, file.Filename)
	if err != nil {
		log.Printf("Ошибка STT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка распознавания речи: " + err.Error()})
		return
	}
	log.Printf("Распознанный текст: %s", text)

	storage := s.sessionManager.GetStorage()
	session := storage.GetOrCreate(sessionID)
	log.Printf("Сессия получена/создана: %s", session.ID)

	storage.AddMessage(session.ID, "user", text)
	log.Printf("Сообщение добавлено в сессию")

	allMessages := storage.GetMessages(session.ID)
	log.Printf("Получено сообщений из хранилища: %d", len(allMessages))
	if len(allMessages) == 0 {
		log.Printf("ОШИБКА: Массив сообщений пустой после добавления!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить сообщения для отправки"})
		return
	}

	messages := make([]gigachat.Message, 0, len(allMessages))
	for _, msg := range allMessages {
		messages = append(messages, gigachat.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	log.Printf("Построено сообщений для контекста: %d", len(messages))
	log.Printf("Отправка сообщения в GigaChat для сессии: %s", session.ID)
	response, err := s.gigaChat.SendMessageWithContext(messages)
	if err != nil {
		log.Printf("Ошибка GigaChat: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка GigaChat: " + err.Error()})
		return
	}
	log.Printf("Получен ответ от GigaChat: %s", response)

	storage.AddMessage(session.ID, "assistant", response)

	log.Printf("Начало синтеза речи, длина текста: %d символов", len(response))
	if len(response) > 1000 {
		log.Printf("ВНИМАНИЕ: Текст длинный (%d символов), синтез может занять больше времени", len(response))
	}
	audioData, err := s.audioClient.TextToSpeech(response)
	if err != nil {
		log.Printf("Ошибка TTS: %v", err)
		log.Printf("Отправка ответа БЕЗ аудио из-за ошибки TTS")
		c.JSON(http.StatusOK, gin.H{
			"text":       text,
			"response":   response,
			"audio":      "",
			"session_id": session.ID,
		})
		return
	}
	log.Printf("Синтез речи завершен, размер аудио: %d байт", len(audioData))
	log.Printf("Отправка ответа клиенту для сессии: %s", session.ID)

	c.JSON(http.StatusOK, gin.H{
		"text":       text,
		"response":   response,
		"audio":      audioData,
		"session_id": session.ID,
	})

	log.Printf("Ответ отправлен клиенту, обработка завершена успешно для сессии: %s", session.ID)
}

func (s *Server) chatMessageHandler(c *gin.Context) {
	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	if err := validateMessage(req.Message); err != nil {
		log.Printf("Ошибка валидации сообщения: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storage := s.sessionManager.GetStorage()
	session := storage.GetOrCreate(req.SessionID)
	storage.AddMessage(session.ID, "user", req.Message)

	messages := s.buildMessages(session)
	log.Printf("Отправка текстового сообщения в GigaChat для сессии: %s", session.ID)
	response, err := s.gigaChat.SendMessageWithContext(messages)
	if err != nil {
		log.Printf("Ошибка GigaChat: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка GigaChat: " + err.Error()})
		return
	}

	storage.AddMessage(session.ID, "assistant", response)

	c.JSON(http.StatusOK, gin.H{
		"response":   response,
		"session_id": session.ID,
	})
}

func (s *Server) clearSessionHandler(c *gin.Context) {
	sessionID := c.Param("id")
	storage := s.sessionManager.GetStorage()
	storage.Clear(sessionID)
	c.JSON(http.StatusOK, gin.H{"message": "Сессия очищена"})
}

func (s *Server) buildMessages(session *storage.Session) []gigachat.Message {
	messages := make([]gigachat.Message, 0, len(session.Messages))
	for i := range session.Messages {
		msg := session.Messages[i]
		messages = append(messages, gigachat.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return messages
}
