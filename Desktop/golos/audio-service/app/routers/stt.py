from fastapi import APIRouter, UploadFile, File, HTTPException
from app.services.stt_service import STTService

router = APIRouter()
stt_service = STTService()

@router.post("/stt")
async def speech_to_text(audio: UploadFile = File(...)):
    try:
        audio_data = await audio.read()
        text = await stt_service.transcribe(audio_data, audio.filename)
        return {"text": text}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Ошибка распознавания речи: {str(e)}")






