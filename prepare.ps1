# Скрипт подготовки проекта для запуска

Write-Host "Подготовка проекта Bekend..." -ForegroundColor Green

# Проверка наличия .env файла
if (-not (Test-Path ".env")) {
    Write-Host "Создание .env файла из шаблона..." -ForegroundColor Yellow
    Copy-Item "env.template" ".env"
    Write-Host ".env файл создан! Отредактируйте его и заполните настройки EMAIL_*" -ForegroundColor Green
} else {
    Write-Host ".env файл уже существует" -ForegroundColor Green
}

# Установка зависимостей Go
Write-Host "Установка зависимостей Go..." -ForegroundColor Yellow
go mod download
go mod tidy

Write-Host "`nГотово! Следующие шаги:" -ForegroundColor Green
Write-Host "1. Отредактируйте .env файл и заполните настройки EMAIL_*" -ForegroundColor Cyan
Write-Host "2. Для Docker: docker-compose up -d" -ForegroundColor Cyan
Write-Host "3. Для локального запуска: go run main.go" -ForegroundColor Cyan
Write-Host "4. Создайте администратора: go run scripts/create_admin.go admin@example.com Admin123! `"Иванов Иван Иванович`"" -ForegroundColor Cyan

