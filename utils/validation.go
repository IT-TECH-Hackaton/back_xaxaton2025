package utils

import (
	"regexp"
	"unicode"
)

func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func ValidateFullName(fullName string) bool {
	russianRegex := regexp.MustCompile(`^[а-яА-ЯёЁ\s]+$`)
	return russianRegex.MatchString(fullName)
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasLetter := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		if unicode.IsLetter(char) && (char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if unicode.IsDigit(char) {
			hasDigit = true
		}
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			hasSpecial = true
		}
	}

	return hasLetter && hasDigit && hasSpecial
}

func ValidateUUID(uuidStr string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(uuidStr)
}

func ValidateStringLength(str string, min, max int) bool {
	length := len([]rune(str))
	return length >= min && length <= max
}

func ValidateVerificationCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	codeRegex := regexp.MustCompile(`^[0-9]{6}$`)
	return codeRegex.MatchString(code)
}

func ValidateRole(role string) bool {
	return role == "Пользователь" || role == "Администратор"
}

func ValidateUserStatus(status string) bool {
	return status == "Активен" || status == "Удален"
}

func ValidateEventStatus(status string) bool {
	return status == "Активное" || status == "Прошедшее" || status == "Отклоненное"
}

func ValidateTelegramUsername(username string) bool {
	if username == "" {
		return true
	}
	telegramRegex := regexp.MustCompile(`^[a-zA-Z0-9_]{5,32}$`)
	return telegramRegex.MatchString(username)
}

func ValidateURL(url string) bool {
	if url == "" {
		return true
	}
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	localPathRegex := regexp.MustCompile(`^/uploads/[^\s]*$`)
	return urlRegex.MatchString(url) || localPathRegex.MatchString(url)
}

