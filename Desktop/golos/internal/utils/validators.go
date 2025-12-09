package utils

import (
	"regexp"
	"unicode"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	russianRegex  = regexp.MustCompile(`^[А-Яа-яЁё\s]+$`)
	passwordRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+$`)
)

func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func ValidateFullName(name string) bool {
	return russianRegex.MatchString(name) && len(name) >= 2
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasLetter := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsLetter(char) && (char < 128):
			hasLetter = true
		case unicode.IsDigit(char):
			hasDigit = true
		case !unicode.IsLetter(char) && !unicode.IsDigit(char) && char < 128:
			hasSpecial = true
		}
	}

	return hasLetter && hasDigit && hasSpecial && passwordRegex.MatchString(password)
}
