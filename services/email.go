package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"

	"bekend/config"
	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
}

func NewEmailService() *EmailService {
	return &EmailService{
		dialer: gomail.NewDialer(
			config.AppConfig.EmailHost,
			config.AppConfig.EmailPort,
			config.AppConfig.EmailUser,
			config.AppConfig.EmailPassword,
		),
	}
}

func (es *EmailService) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.AppConfig.EmailFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := es.dialer.DialAndSend(m); err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	return nil
}

func (es *EmailService) SendVerificationCode(email, code string) error {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.code { font-size: 24px; font-weight: bold; text-align: center; padding: 20px; background: #f4f4f4; border-radius: 5px; margin: 20px 0; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Подтверждение электронной почты</h2>
			<p>Для завершения регистрации введите следующий код подтверждения:</p>
			<div class="code">{{.Code}}</div>
			<p>Код действителен в течение 10 минут.</p>
		</div>
	</body>
	</html>
	`

	t, err := template.New("verification").Parse(tmpl)
	if err != nil {
		return err
	}

	var bodyBuffer bytes.Buffer
	if err := t.Execute(&bodyBuffer, map[string]string{"Code": code}); err != nil {
		return err
	}

	return es.SendEmail(email, "Подтверждение электронной почты", bodyBuffer.String())
}

func (es *EmailService) SendWelcomeEmail(email, fullName string) error {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Добро пожаловать, {{.FullName}}!</h2>
			<p>Ваша регистрация успешно завершена. Теперь вы можете использовать все возможности нашей системы электронной афиши.</p>
			<p>Приятного использования!</p>
		</div>
	</body>
	</html>
	`

	t, err := template.New("welcome").Parse(tmpl)
	if err != nil {
		return err
	}

	var bodyBuffer bytes.Buffer
	if err := t.Execute(&bodyBuffer, map[string]string{"FullName": fullName}); err != nil {
		return err
	}

	return es.SendEmail(email, "Добро пожаловать!", bodyBuffer.String())
}

func (es *EmailService) SendPasswordResetLink(email, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", config.AppConfig.FrontendURL, token)
	
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.button { display: inline-block; padding: 12px 24px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Восстановление пароля</h2>
			<p>Вы запросили восстановление пароля. Для сброса пароля перейдите по ссылке ниже:</p>
			<a href="{{.ResetURL}}" class="button">Сбросить пароль</a>
			<p>Ссылка действительна в течение 24 часов.</p>
			<p>Если вы не запрашивали восстановление пароля, проигнорируйте это письмо.</p>
		</div>
	</body>
	</html>
	`

	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		return err
	}

	var bodyBuffer bytes.Buffer
	if err := t.Execute(&bodyBuffer, map[string]string{"ResetURL": resetURL}); err != nil {
		return err
	}

	return es.SendEmail(email, "Восстановление пароля", bodyBuffer.String())
}

func (es *EmailService) SendPasswordChangedNotification(email string) error {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Пароль изменен</h2>
			<p>Ваш пароль был успешно изменен.</p>
			<p>Если это были не вы, немедленно свяжитесь с поддержкой.</p>
		</div>
	</body>
	</html>
	`

	return es.SendEmail(email, "Пароль изменен", tmpl)
}

func (es *EmailService) SendEventNotification(email, eventTitle, message string) error {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Уведомление о событии: {{.EventTitle}}</h2>
			<p>{{.Message}}</p>
		</div>
	</body>
	</html>
	`

	t, err := template.New("event").Parse(tmpl)
	if err != nil {
		return err
	}

	var bodyBuffer bytes.Buffer
	if err := t.Execute(&bodyBuffer, map[string]string{
		"EventTitle": eventTitle,
		"Message":    message,
	}); err != nil {
		return err
	}

	return es.SendEmail(email, fmt.Sprintf("Уведомление: %s", eventTitle), bodyBuffer.String())
}

