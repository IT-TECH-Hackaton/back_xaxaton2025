from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.routers import stt, tts

app = FastAPI(title="Audio Service", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(stt.router, prefix="/api/v1", tags=["STT"])
app.include_router(tts.router, prefix="/api/v1", tags=["TTS"])

@app.get("/health")
async def health():
    return {"status": "ok", "service": "audio-service"}






