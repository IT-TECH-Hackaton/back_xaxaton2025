package utils

import (
	"regexp"
	"unicode"
)

func ValidateEmail(email string) bool {
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

