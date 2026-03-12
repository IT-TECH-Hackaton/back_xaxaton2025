@echo off
chcp 65001 >nul
echo === Запуск сервера MyAfisha ===
echo.
echo При запуске будет создан/обновлён рекламодатель:
echo   Email: advertiser@test.local
echo   Пароль: Advertiser123!
echo.
cd /d "%~dp0"

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ОШИБКА] Go не найден в PATH.
    echo Установите Go: https://go.dev/dl/
    echo Или добавьте путь к go.exe в переменную PATH.
    pause
    exit /b 1
)

go run .
pause
