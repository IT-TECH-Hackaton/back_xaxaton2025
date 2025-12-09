#!/bin/bash

# Скрипт подготовки проекта для запуска

echo "Подготовка проекта Bekend..."

# Проверка наличия .env файла
if [ ! -f ".env" ]; then
    echo "Создание .env файла из шаблона..."
    cp env.template .env
    echo ".env файл создан! Отредактируйте его и заполните настройки EMAIL_*"
else
    echo ".env файл уже существует"
fi

# Установка зависимостей Go
echo "Установка зависимостей Go..."
go mod download
go mod tidy

echo ""
echo "Готово! Следующие шаги:"
echo "1. Отредактируйте .env файл и заполните настройки EMAIL_*"
echo "2. Для Docker: docker-compose up -d"
echo "3. Для локального запуска: go run main.go"
echo "4. Создайте администратора: go run scripts/create_admin.go admin@example.com Admin123! \"Иванов Иван Иванович\""

