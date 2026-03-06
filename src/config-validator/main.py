"""Config Validator - Multi-layer YAML validation."""
import io, re
from fastapi import FastAPI
from pydantic import BaseModel
from ruamel.yaml import YAML

app = FastAPI(title="Config Validator", version="0.1.0")
yaml = YAML()

NAMING_RE = re.compile(r"^(odu|ocu-cp|ocu-up|amf|smf|upf|nrf|nssf)-(embb|urllc|mmtc|custom-\d+)-\d{3}(-scaled-\d{2})?$")


class ValidateRequest(BaseModel):
    yaml_content: str


class ValidateResponse(BaseModel):
    valid: bool
    errors: list[str] = []
    warnings: list[str] = []


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/validate", response_model=ValidateResponse)
async def validate(req: ValidateRequest):
    errors, warnings = [], []
    try:
        docs = list(yaml.load_all(io.StringIO(req.yaml_content)))
    except Exception as e:
        return ValidateResponse(valid=False, errors=[f"YAML syntax error: {e}"])

    for doc in docs:
        if not isinstance(doc, dict):
            continue
        name = doc.get("metadata", {}).get("name", "")
        if doc.get("kind") in ("Deployment", "StatefulSet", "Pod") and name:
            if not NAMING_RE.match(name):
                errors.append(f"Name \'{name}\' violates naming convention")
    return ValidateResponse(valid=len(errors) == 0, errors=errors, warnings=warnings)
