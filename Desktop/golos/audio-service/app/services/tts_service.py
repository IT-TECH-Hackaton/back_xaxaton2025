import pyttsx3
import io
import asyncio
import concurrent.futures
import tempfile
import os
import logging

logger = logging.getLogger(__name__)

class TTSService:
    def __init__(self):
        self.engine = None
        self.executor = concurrent.futures.ThreadPoolExecutor(max_workers=2)
        self._init_engine()
    
    def _init_engine(self):
        self.engine = pyttsx3.init()
        self.engine.setProperty('rate', 150)
        self.engine.setProperty('volume', 0.9)
        
        voices = self.engine.getProperty('voices')
        if voices:
            for voice in voices:
                if 'russian' in voice.name.lower() or 'ru' in voice.id.lower():
                    self.engine.setProperty('voice', voice.id)
                    break
    
    def _synthesize_sync(self, text: str, temp_path: str):
        try:
            logger.info(f"Инициализация pyttsx3 для текста длиной {len(text)} символов")
            engine = pyttsx3.init(driverName='espeak')
            engine.setProperty('rate', 150)
            engine.setProperty('volume', 0.9)
            
            voices = engine.getProperty('voices')
            logger.info(f"Доступно голосов: {len(voices) if voices else 0}")
            if voices:
                for voice in voices:
                    logger.info(f"Проверка голоса: {voice.name}, id: {voice.id}")
                    if 'russian' in voice.name.lower() or 'ru' in voice.id.lower():
                        engine.setProperty('voice', voice.id)
                        logger.info(f"Выбран голос: {voice.name} ({voice.id})")
                        break
            
            logger.info(f"Сохранение аудио в файл: {temp_path}")
            logger.info(f"Первые 100 символов текста: {text[:100]}")
            
            engine.save_to_file(text, temp_path)
            logger.info("Запуск синтеза речи (runAndWait)...")
            engine.runAndWait()
            
            import time
            time.sleep(0.5)
            
            if not os.path.exists(temp_path):
                raise Exception(f"Файл {temp_path} не был создан")
            
            file_size = os.path.getsize(temp_path)
            logger.info(f"Синтез речи завершен, размер файла: {file_size} байт")
            
            if file_size == 0:
                logger.error(f"Файл {temp_path} пустой! Попытка повторного синтеза...")
                try:
                    engine2 = pyttsx3.init(driverName='espeak')
                    engine2.setProperty('rate', 150)
                    engine2.setProperty('volume', 0.9)
                    engine2.save_to_file(text[:500], temp_path)
                    engine2.runAndWait()
                    time.sleep(0.5)
                    file_size = os.path.getsize(temp_path)
                    logger.info(f"После повторного синтеза размер файла: {file_size} байт")
                    if file_size == 0:
                        raise Exception(f"Файл все еще пустой после повторного синтеза")
                except Exception as e2:
                    logger.error(f"Ошибка при повторном синтезе: {e2}")
                    raise Exception(f"Не удалось создать аудио файл: {e2}")
        except Exception as e:
            logger.error(f"Ошибка в _synthesize_sync: {e}", exc_info=True)
            raise
    
    async def synthesize(self, text: str, voice: str = "default") -> bytes:
        original_length = len(text)
        if len(text) > 1500:
            sentences = text.split('. ')
            if len(sentences) > 1:
                text = '. '.join(sentences[:5]) + '.'
                if len(text) > 1500:
                    text = text[:1500] + "..."
            else:
                text = text[:1500] + "..."
            logger.warning(f"Текст обрезан с {original_length} до {len(text)} символов для ускорения синтеза")
        
        loop = asyncio.get_event_loop()
        temp_path = None
        try:
            logger.info(f"Начало синтеза речи, длина текста: {len(text)} символов")
            with tempfile.NamedTemporaryFile(delete=False, suffix='.wav') as tmp_file:
                temp_path = tmp_file.name
            
            logger.info(f"Временный файл создан: {temp_path}")
            await loop.run_in_executor(
                self.executor,
                self._synthesize_sync,
                text,
                temp_path
            )
            
            logger.info(f"Синтез завершен, проверка файла: {temp_path}")
            if not os.path.exists(temp_path):
                raise Exception(f"Файл {temp_path} не существует после синтеза")
            
            file_size = os.path.getsize(temp_path)
            logger.info(f"Размер файла перед чтением: {file_size} байт")
            
            if file_size == 0:
                raise Exception(f"Файл {temp_path} пустой после синтеза")
            
            with open(temp_path, 'rb') as f:
                audio_data = f.read()
            
            logger.info(f"Аудио данные прочитаны, размер: {len(audio_data)} байт")
            if os.path.exists(temp_path):
                os.remove(temp_path)
            
            if len(audio_data) == 0:
                raise Exception("Аудио данные пустые после чтения файла")
            
            return audio_data
        except Exception as e:
            logger.error(f"Ошибка синтеза речи: {e}")
            if temp_path and os.path.exists(temp_path):
                try:
                    os.remove(temp_path)
                except:
                    pass
            raise Exception(f"Ошибка синтеза речи: {e}")
