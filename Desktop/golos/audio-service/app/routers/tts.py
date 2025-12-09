from fastapi import APIRouter, HTTPException
from fastapi.responses import Response
from pydantic import BaseModel
from app.services.tts_service import TTSService
import asyncio

router = APIRouter()
tts_service = TTSService()

class TTSRequest(BaseModel):
    text: str
    voice: str = "default"

@router.post("/tts")
async def text_to_speech(request: TTSRequest):
    import logging
    logger = logging.getLogger(__name__)
    try:
        logger.info(f"Получен запрос TTS, длина текста: {len(request.text)} символов")
        audio_data = await asyncio.wait_for(
            tts_service.synthesize(request.text, request.voice),
            timeout=100.0
        )
        logger.info(f"TTS завершен успешно, размер аудио: {len(audio_data)} байт")
        return Response(content=audio_data, media_type="audio/wav")
    except asyncio.TimeoutError:
        logger.error("Таймаут синтеза речи")
        raise HTTPException(status_code=504, detail="Таймаут синтеза речи")
    except Exception as e:
        logger.error(f"Ошибка синтеза речи: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Ошибка синтеза речи: {str(e)}")






