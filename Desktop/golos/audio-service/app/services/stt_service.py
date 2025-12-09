import speech_recognition as sr
import io
import tempfile
import os
try:
    from pydub import AudioSegment
    PYDUB_AVAILABLE = True
except ImportError:
    PYDUB_AVAILABLE = False

class STTService:
    def __init__(self):
        self.recognizer = sr.Recognizer()
    
    async def transcribe(self, audio_data: bytes, filename: str) -> str:
        tmp_file_path = None
        wav_file_path = None
        
        try:
            file_ext = os.path.splitext(filename)[1].lower() if filename else '.webm'
            
            with tempfile.NamedTemporaryFile(delete=False, suffix=file_ext) as tmp_file:
                tmp_file.write(audio_data)
                tmp_file_path = tmp_file.name
            
            wav_file_path = tmp_file_path.replace(file_ext, '.wav')
            
            try:
                if file_ext in ['.wav', '.wave']:
                    wav_file_path = tmp_file_path
                elif PYDUB_AVAILABLE:
                    if file_ext in ['.mp3', '.mpeg']:
                        audio_segment = AudioSegment.from_mp3(tmp_file_path)
                    elif file_ext in ['.ogg', '.oga']:
                        audio_segment = AudioSegment.from_ogg(tmp_file_path)
                    elif file_ext in ['.webm']:
                        try:
                            audio_segment = AudioSegment.from_file(tmp_file_path, format="webm")
                        except Exception as e:
                            raise Exception(f"Для конвертации WebM нужен FFmpeg. Ошибка: {str(e)}. Установите: winget install ffmpeg")
                    elif file_ext in ['.m4a', '.aac']:
                        audio_segment = AudioSegment.from_file(tmp_file_path, format="m4a")
                    else:
                        try:
                            audio_segment = AudioSegment.from_file(tmp_file_path)
                        except:
                            raise Exception(f"Неподдерживаемый формат: {file_ext}")
                    
                    audio_segment = audio_segment.set_frame_rate(16000)
                    audio_segment = audio_segment.set_channels(1)
                    audio_segment = audio_segment.set_sample_width(2)
                    
                    audio_segment.export(wav_file_path, format="wav")
                else:
                    raise Exception(f"Формат {file_ext} требует конвертации. Установите: pip install pydub")
                
                with sr.AudioFile(wav_file_path) as source:
                    audio = self.recognizer.record(source)
                
                text = self.recognizer.recognize_google(audio, language="ru-RU")
                return text
            finally:
                if tmp_file_path and os.path.exists(tmp_file_path):
                    os.unlink(tmp_file_path)
                if wav_file_path and os.path.exists(wav_file_path):
                    os.unlink(wav_file_path)
        except sr.UnknownValueError:
            return "Не удалось распознать речь"
        except sr.RequestError as e:
            raise Exception(f"Ошибка сервиса распознавания: {e}")
        except Exception as e:
            raise Exception(f"Ошибка обработки аудио: {str(e)}")
