Write-Host "Запуск голосового помощника..." -ForegroundColor Green

Write-Host "`n1. Запуск Audio Service (FastAPI)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\audio-service'; if (Test-Path venv) { .\venv\Scripts\Activate.ps1 }; python -m uvicorn app.main:app --reload --port 8000" -WindowStyle Normal

Start-Sleep -Seconds 3

Write-Host "`n2. Запуск Go API сервера..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; `$env:GIGACHAT_CLIENT_ID='019a81d2-9f7c-7429-a7eb-f240038d4d22'; `$env:GIGACHAT_CLIENT_SECRET='9fc30b5d-f451-4963-8495-7da27ef39ef1'; `$env:GIGACHAT_AUTHORIZATION_KEY='MDE5YTgxZDItOWY3Yy03NDI5LWE3ZWItZjI0MDAzOGQ0ZDIyOjlmYzMwYjVkLWY0NTEtNDk2My04NDk1LTdkYTI3ZWYzOWVmMQ=='; `$env:GIGACHAT_SCOPE='GIGACHAT_API_PERS'; `$env:API_PORT='8080'; `$env:API_HOST='0.0.0.0'; `$env:AUDIO_SERVICE_URL='http://localhost:8000'; go run cmd/api/main.go" -WindowStyle Normal

Start-Sleep -Seconds 2

Write-Host "`n✅ Сервисы запускаются!" -ForegroundColor Green
Write-Host "   - Audio Service: http://localhost:8000" -ForegroundColor Cyan
Write-Host "   - API Server: http://localhost:8080" -ForegroundColor Cyan
Write-Host "   - Web Interface: http://localhost:8080" -ForegroundColor Cyan
Write-Host "`nПроверка health check через 5 секунд..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/health" -Method Get -TimeoutSec 5
    Write-Host "`n✅ Сервер работает! Статус: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "`n⚠️  Сервер еще запускается, попробуйте открыть http://localhost:8080 через несколько секунд" -ForegroundColor Yellow
}






