package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bekend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UploadHandler struct {
	logger *zap.Logger
}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{
		logger: utils.GetLogger(),
	}
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

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}

	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый формат файла. Разрешены: jpg, jpeg, png, gif, webp, svg"})
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
	
	validImage := h.isValidImageFile(buffer, mimeType, ext)
	if !validImage {
		h.logger.Warn("Попытка загрузки недопустимого файла",
			zap.String("mimeType", mimeType),
			zap.String("extension", ext),
			zap.String("filename", file.Filename),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый тип файла. Файл должен быть изображением (JPEG, PNG, GIF, WebP)"})
		return
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		h.logger.Error("Ошибка сброса указателя файла", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла"})
		return
	}

	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("Ошибка создания директории для загрузки", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании директории"})
		return
	}

	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

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

func (h *UploadHandler) isValidImageFile(buffer []byte, mimeType string, ext string) bool {
	extLower := strings.ToLower(ext)
	
	if extLower == ".svg" {
		svgMagic := []byte("<svg")
		svgMagicAlt := []byte("<?xml")
		svgContent := strings.ToLower(string(buffer[:min(len(buffer), 100)]))
		return bytes.HasPrefix(buffer, svgMagic) || bytes.HasPrefix(buffer, svgMagicAlt) || strings.Contains(svgContent, "<svg")
	}

	allowedMimeTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	mimeAllowed := false
	for _, mime := range allowedMimeTypes {
		if strings.HasPrefix(mimeType, mime) {
			mimeAllowed = true
			break
		}
	}

	if !mimeAllowed {
		return false
	}

	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	extAllowed := false
	for _, e := range allowedExts {
		if extLower == e {
			extAllowed = true
			break
		}
	}

	if !extAllowed {
		return false
	}

	if len(buffer) < 4 {
		return false
	}

	jpegMagic := []byte{0xFF, 0xD8, 0xFF}
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47}
	gifMagic := []byte{0x47, 0x49, 0x46, 0x38}
	webpMagic := []byte{0x52, 0x49, 0x46, 0x46}

	if bytes.HasPrefix(buffer, jpegMagic) {
		return extLower == ".jpg" || extLower == ".jpeg"
	}
	if bytes.HasPrefix(buffer, pngMagic) {
		return extLower == ".png"
	}
	if bytes.HasPrefix(buffer, gifMagic) {
		return extLower == ".gif"
	}
	if bytes.HasPrefix(buffer, webpMagic) && len(buffer) >= 12 {
		if bytes.Equal(buffer[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
			return extLower == ".webp"
		}
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

