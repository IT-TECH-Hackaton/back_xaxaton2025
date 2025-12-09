package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не найден"})
		return
	}

	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Размер файла не должен превышать 10MB"})
		return
	}

	if file.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл пустой"})
		return
	}

	ext := filepath.Ext(file.Filename)
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowed := false
	for _, e := range allowedExts {
		if strings.EqualFold(ext, e) {
			allowed = true
			break
		}
	}

	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый формат файла. Разрешены: jpg, jpeg, png, gif, webp"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при открытии файла"})
		return
	}
	defer src.Close()

	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка при чтении файла"})
		return
	}

	mimeType := http.DetectContentType(buffer)
	allowedMimeTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	mimeAllowed := false
	for _, mime := range allowedMimeTypes {
		if mimeType == mime {
			mimeAllowed = true
			break
		}
	}

	if !mimeAllowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый тип файла. Файл должен быть изображением"})
		return
	}

	if _, err := src.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла"})
		return
	}

	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании директории"})
		return
	}

	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при открытии файла"})
		return
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании файла"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении файла"})
		return
	}

	imageURL := fmt.Sprintf("/uploads/%s", filename)
	c.JSON(http.StatusOK, gin.H{"imageURL": imageURL})
}

