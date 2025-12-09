Write-Host "=== Запуск голосового помощника ===" -ForegroundColor Green

$env:GIGACHAT_CLIENT_ID = '019a81d2-9f7c-7429-a7eb-f240038d4d22'
$env:GIGACHAT_CLIENT_SECRET = '9fc30b5d-f451-4963-8495-7da27ef39ef1'
$env:GIGACHAT_AUTHORIZATION_KEY = 'MDE5YTgxZDItOWY3Yy03NDI5LWE3ZWItZjI0MDAzOGQ0ZDIyOjlmYzMwYjVkLWY0NTEtNDk2My04NDk1LTdkYTI3ZWYzOWVmMQ=='
$env:GIGACHAT_SCOPE = 'GIGACHAT_API_PERS'
$env:API_PORT = '8080'
$env:API_HOST = '0.0.0.0'
$env:AUDIO_SERVICE_URL = 'http://localhost:8000'

Write-Host "`n✅ Переменные окружения установлены" -ForegroundColor Green

Write-Host "`n📡 Запуск Audio Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\audio-service'; Write-Host 'Audio Service запускается...' -ForegroundColor Green; python -m uvicorn app.main:app --reload --port 8000" -WindowStyle Normal

Write-Host "   Audio Service запущен в отдельном окне" -ForegroundColor Cyan
Start-Sleep -Seconds 4

Write-Host "`n🔄 Освобождение порта 8080..." -ForegroundColor Yellow
$ports = Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue
foreach ($port in $ports) {
    Stop-Process -Id $port.OwningProcess -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

Write-Host "`n🚀 Запуск Go API сервера..." -ForegroundColor Green
Write-Host "   =========================================" -ForegroundColor Cyan
Write-Host "   Веб-интерфейс: http://localhost:8080" -ForegroundColor White
Write-Host "   Health check:  http://localhost:8080/api/v1/health" -ForegroundColor White
Write-Host "   Metrics:       http://localhost:8080/api/v1/metrics" -ForegroundColor White
Write-Host "   =========================================" -ForegroundColor Cyan
Write-Host "`nСервер запускается...`n" -ForegroundColor Yellow

go run cmd/api/main.go






