# Исправление ошибки распознавания речи

## Проблема
Браузер записывает аудио в формате WebM, а библиотека `speech_recognition` требует WAV формат.

## Решение

### 1. Установите FFmpeg (обязательно!)

**Через winget:**
```powershell
winget install ffmpeg
```

**Через Chocolatey:**
```powershell
choco install ffmpeg
```

**Вручную:**
1. Скачайте с https://ffmpeg.org/download.html
2. Распакуйте и добавьте в PATH

### 2. Установите зависимости Python

```powershell
cd audio-service
py -m pip install pydub
```

### 3. Перезапустите Audio Service

После установки FFmpeg перезапустите Audio Service:

```powershell
cd audio-service
py -m uvicorn app.main:app --reload --port 8000
```

## Что было исправлено:

1. ✅ Добавлена поддержка конвертации WebM → WAV через pydub
2. ✅ Обновлен фронтенд для правильной отправки MIME-типа
3. ✅ Добавлена обработка различных аудио форматов
4. ✅ Улучшена обработка ошибок

## Проверка работы:

После установки FFmpeg и перезапуска Audio Service попробуйте снова записать голосовое сообщение через веб-интерфейс.







