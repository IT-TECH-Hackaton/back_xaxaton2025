let mediaRecorder;
let audioChunks = [];
let isRecording = false;
let currentSessionId = null;

const recordBtn = document.getElementById('recordBtn');
const stopBtn = document.getElementById('stopBtn');
const statusDiv = document.getElementById('status');
const userMessageDiv = document.getElementById('userMessage');
const assistantMessageDiv = document.getElementById('assistantMessage');
const audioPlayer = document.getElementById('audioPlayer');

recordBtn.addEventListener('click', startRecording);
stopBtn.addEventListener('click', stopRecording);

const clearBtn = document.getElementById('clearBtn');
clearBtn.addEventListener('click', async () => {
    if (currentSessionId) {
        try {
            await fetch(`/api/v1/session/${currentSessionId}`, {
                method: 'DELETE'
            });
        } catch (error) {
            console.error('Ошибка очистки сессии:', error);
        }
    }
    currentSessionId = null;
    userMessageDiv.textContent = '';
    assistantMessageDiv.textContent = '';
    audioPlayer.style.display = 'none';
    statusDiv.textContent = 'Сессия очищена';
    statusDiv.className = 'status success';
    setTimeout(() => {
        statusDiv.textContent = '';
        statusDiv.className = 'status';
    }, 2000);
});

async function startRecording() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        mediaRecorder = new MediaRecorder(stream);
        audioChunks = [];

        mediaRecorder.ondataavailable = (event) => {
            audioChunks.push(event.data);
        };

        mediaRecorder.onstop = async () => {
            const mimeType = mediaRecorder.mimeType || 'audio/webm';
            const audioBlob = new Blob(audioChunks, { type: mimeType });
            await processAudio(audioBlob, mimeType);
            stream.getTracks().forEach(track => track.stop());
        };

        mediaRecorder.start();
        isRecording = true;
        
        recordBtn.disabled = true;
        stopBtn.disabled = false;
        statusDiv.textContent = '🎤 Идет запись...';
        statusDiv.className = 'status recording';
    } catch (error) {
        showError('Ошибка доступа к микрофону: ' + error.message);
    }
}

function stopRecording() {
    if (mediaRecorder && isRecording) {
        mediaRecorder.stop();
        isRecording = false;
        recordBtn.disabled = false;
        stopBtn.disabled = true;
        statusDiv.textContent = '⏳ Обработка...';
        statusDiv.className = 'status processing';
    }
}

async function processAudio(audioBlob, mimeType = 'audio/webm') {
    try {
        const formData = new FormData();
        const extension = mimeType.includes('webm') ? 'webm' : 
                         mimeType.includes('mp3') ? 'mp3' : 
                         mimeType.includes('ogg') ? 'ogg' : 'wav';
        formData.append('audio', audioBlob, `recording.${extension}`);

        let url = '/api/v1/voice/process';
        if (currentSessionId) {
            url += '?session_id=' + currentSessionId;
        }

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 180000);
        
        const response = await fetch(url, {
            method: 'POST',
            body: formData,
            signal: controller.signal
        });
        
        clearTimeout(timeoutId);

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Ошибка обработки');
        }

        const data = await response.json();
        console.log('Получен ответ от сервера:', data);
        
        if (data.session_id) {
            currentSessionId = data.session_id;
        }
        
        userMessageDiv.textContent = data.text || 'Текст не распознан';
        assistantMessageDiv.textContent = data.response || 'Ответ не получен';

        if (data.audio) {
            try {
                const audioData = 'data:audio/wav;base64,' + data.audio;
                audioPlayer.src = audioData;
                audioPlayer.style.display = 'block';
                
                audioPlayer.onloadeddata = () => {
                    audioPlayer.play();
                };
                audioPlayer.onerror = (e) => {
                    console.error('Ошибка загрузки аудио:', e);
                };
            } catch (audioError) {
                console.error('Ошибка обработки аудио:', audioError);
            }
        }

        statusDiv.textContent = '✅ Готово!';
        statusDiv.className = 'status success';
    } catch (error) {
        if (error.name === 'AbortError') {
            showError('Таймаут запроса. Обработка занимает слишком много времени.');
        } else {
            showError('Ошибка: ' + error.message);
        }
        recordBtn.disabled = false;
        stopBtn.disabled = true;
        isRecording = false;
    }
}

function showError(message) {
    statusDiv.textContent = '❌ ' + message;
    statusDiv.className = 'status error';
    recordBtn.disabled = false;
    stopBtn.disabled = true;
    isRecording = false;
}
