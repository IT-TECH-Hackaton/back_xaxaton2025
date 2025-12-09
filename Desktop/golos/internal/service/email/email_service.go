package email

import (
	"fmt"
	"os"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewEmailService(host string, port int, user, password, from string) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *EmailService) sendEmail(to, subject, body string) error {
	if s.host == "" || s.user == "" || s.password == "" {
		return fmt.Errorf("настройки email не сконфигурированы")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.host, s.port, s.user, s.password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("ошибка отправки email: %w", err)
	}

	return nil
}

func (s *EmailService) SendVerificationCode(email, code string) error {
	subject := "Подтверждение регистрации"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Подтверждение регистрации</h2>
			<p>Ваш код подтверждения: <strong style="font-size: 24px; color: #007bff;">%s</strong></p>
			<p>Код действителен в течение 15 минут.</p>
		</body>
		</html>
	`, code)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendWelcomeEmail(email, fullName string) error {
	subject := "Добро пожаловать!"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Добро пожаловать, %s!</h2>
			<p>Ваша регистрация успешно завершена.</p>
			<p>Теперь вы можете использовать все возможности системы электронной афиши.</p>
		</body>
		</html>
	`, fullName)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendPasswordResetLink(email, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("FRONTEND_URL"), token)
	subject := "Восстановление пароля"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Восстановление пароля</h2>
			<p>Для восстановления пароля перейдите по ссылке:</p>
			<p><a href="%s" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Восстановить пароль</a></p>
			<p>Ссылка действительна в течение 24 часов.</p>
			<p>Если вы не запрашивали восстановление пароля, проигнорируйте это письмо.</p>
		</body>
		</html>
	`, resetURL)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendPasswordChangedNotification(email string) error {
	subject := "Пароль изменен"
	body := `
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Пароль успешно изменен</h2>
			<p>Ваш пароль был успешно изменен.</p>
			<p>Если это были не вы, немедленно свяжитесь с поддержкой.</p>
		</body>
		</html>
	`

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendEventNotification(email, subject, body string) error {
	return s.sendEmail(email, subject, body)
}
