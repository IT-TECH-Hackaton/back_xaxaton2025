package auth

import (
	"errors"
	"unicode"

	"golos/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct{}

func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

func (s *PasswordService) HashPassword(password string) (string, error) {
	if !utils.ValidatePassword(password) {
		return "", errors.New("пароль не соответствует требованиям безопасности")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func (s *PasswordService) ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (s *PasswordService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("пароль должен содержать минимум 8 символов")
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

	if !hasLetter {
		return errors.New("пароль должен содержать латинские буквы")
	}
	if !hasDigit {
		return errors.New("пароль должен содержать цифры")
	}
	if !hasSpecial {
		return errors.New("пароль должен содержать специальные символы")
	}

	return nil
}
