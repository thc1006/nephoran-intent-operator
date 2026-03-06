"""Intent Engine - TMF 921-aligned API."""
from fastapi import FastAPI, HTTPException
from models import IntentRequest, IntentResponse, IntentStatus

app = FastAPI(title="Intent Engine", version="0.1.0")
intents_store: dict[str, IntentResponse] = {}


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/intents", response_model=IntentResponse)
async def create_intent(req: IntentRequest):
    intent = IntentResponse(original_text=req.original_text)
    intents_store[intent.intent_id] = intent
    # TODO: trigger plan_executor pipeline
    return intent


@app.get("/intents/{intent_id}", response_model=IntentResponse)
async def get_intent(intent_id: str):
    if intent_id not in intents_store:
        raise HTTPException(status_code=404, detail="Intent not found")
    return intents_store[intent_id]


@app.delete("/intents/{intent_id}")
async def cancel_intent(intent_id: str):
    if intent_id not in intents_store:
        raise HTTPException(status_code=404, detail="Intent not found")
    intents_store[intent_id].status = IntentStatus.ROLLED_BACK
    return {"status": "cancelled"}
