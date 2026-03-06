"""LLM Gateway - FastAPI service for template generation, hydration, and analysis."""
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from llm_client import LLMClient

app = FastAPI(title="LLM Gateway", version="0.1.0")
llm = LLMClient()


class GenerateRequest(BaseModel):
    intent_plan: dict
    template_name: str = "crd_generation"


class HydrateRequest(BaseModel):
    template_yaml: str
    parameters: dict


class AnalyzeRequest(BaseModel):
    metrics: dict
    intent_sla: dict


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/generate")
async def generate(req: GenerateRequest):
    try:
        result = await llm.generate(req.intent_plan, req.template_name)
        return {"yaml": result, "status": "generated"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/hydrate")
async def hydrate(req: HydrateRequest):
    try:
        result = await llm.hydrate(req.template_yaml, req.parameters)
        return {"yaml": result, "status": "hydrated"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/analyze")
async def analyze(req: AnalyzeRequest):
    try:
        result = await llm.analyze(req.metrics, req.intent_sla)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
