Write-Host "=== Запуск голосового помощника ===" -ForegroundColor Green

$env:GIGACHAT_CLIENT_ID = '019a81d2-9f7c-7429-a7eb-f240038d4d22'
$env:GIGACHAT_CLIENT_SECRET = '9fc30b5d-f451-4963-8495-7da27ef39ef1'
$env:GIGACHAT_AUTHORIZATION_KEY = 'MDE5YTgxZDItOWY3Yy03NDI5LWE3ZWItZjI0MDAzOGQ0ZDIyOjlmYzMwYjVkLWY0NTEtNDk2My04NDk1LTdkYTI3ZWYzOWVmMQ=='
$env:GIGACHAT_SCOPE = 'GIGACHAT_API_PERS'
$env:API_PORT = '8080'
$env:API_HOST = '0.0.0.0'
$env:AUDIO_SERVICE_URL = 'http://localhost:8000'

Write-Host "`n✅ Переменные окружения установлены" -ForegroundColor Green

Write-Host "`n📡 Проверка Audio Service..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8000/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
    Write-Host "✅ Audio Service работает" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Audio Service не запущен" -ForegroundColor Yellow
    Write-Host "   Запускаю Audio Service в отдельном окне..." -ForegroundColor Cyan
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\audio-service'; python -m uvicorn app.main:app --reload --port 8000" -WindowStyle Normal
    Start-Sleep -Seconds 3
}

Write-Host "`n🔄 Освобождение порта 8080..." -ForegroundColor Yellow
$port = (Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue).OwningProcess
if ($port) {
    Stop-Process -Id $port -Force -ErrorAction SilentlyContinue
    Write-Host "   Порт освобожден" -ForegroundColor Green
    Start-Sleep -Seconds 2
}

Write-Host "`n🚀 Запуск Go API сервера..." -ForegroundColor Green
Write-Host "   Сервер будет доступен на: http://localhost:8080" -ForegroundColor Cyan
Write-Host "   Health check: http://localhost:8080/api/v1/health" -ForegroundColor Cyan
Write-Host "   Веб-интерфейс: http://localhost:8080" -ForegroundColor Cyan
Write-Host "`n" -ForegroundColor White

go run cmd/api/main.go






