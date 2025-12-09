package api

import (
	"fmt"
	"mime/multipart"
	"strings"
)

const (
	maxFileSize      = 10 * 1024 * 1024
	maxMessageLength = 5000
)

var allowedMimeTypes = []string{
	"audio/wav",
	"audio/wave",
	"audio/mpeg",
	"audio/mp3",
	"audio/ogg",
	"audio/webm",
	"audio/x-wav",
}

func validateAudioFile(file *multipart.FileHeader) error {
	if file.Size == 0 {
		return fmt.Errorf("файл пустой")
	}

	if file.Size > maxFileSize {
		return fmt.Errorf("файл слишком большой (максимум %d МБ)", maxFileSize/(1024*1024))
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, ".")+1:])
		if ext != "wav" && ext != "mp3" && ext != "ogg" && ext != "webm" {
			return fmt.Errorf("неподдерживаемый формат файла")
		}
	} else {
		contentTypeBase := strings.Split(contentType, ";")[0]
		contentTypeBase = strings.TrimSpace(contentTypeBase)
		allowed := false
		for _, allowedType := range allowedMimeTypes {
			if contentTypeBase == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("неподдерживаемый тип файла: %s", contentType)
		}
	}

	return nil
}

func validateMessage(message string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("сообщение не может быть пустым")
	}

	if len(message) > maxMessageLength {
		return fmt.Errorf("сообщение слишком длинное (максимум %d символов)", maxMessageLength)
	}

	return nil
}
