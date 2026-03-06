"""Abstraction layer for vLLM / Ollama backends."""
import os
import httpx
from jinja2 import Environment, FileSystemLoader

LLM_BASE_URL = os.getenv("LLM_BASE_URL", "http://localhost:8000/v1")
LLM_MODEL = os.getenv("LLM_MODEL", "Qwen/Qwen2.5-Coder-32B-Instruct")

template_env = Environment(loader=FileSystemLoader("prompt_templates"))


class LLMClient:
    def __init__(self):
        self.client = httpx.AsyncClient(timeout=120.0)

    async def _call_llm(self, prompt: str) -> str:
        resp = await self.client.post(
            f"{LLM_BASE_URL}/chat/completions",
            json={"model": LLM_MODEL, "messages": [{"role": "user", "content": prompt}], "temperature": 0.1, "max_tokens": 4096},
        )
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]["content"]

    async def generate(self, intent_plan: dict, template_name: str) -> str:
        tmpl = template_env.get_template(f"{template_name}.j2")
        prompt = tmpl.render(intent_plan=intent_plan)
        return self._extract_yaml(await self._call_llm(prompt))

    async def hydrate(self, template_yaml: str, parameters: dict) -> str:
        tmpl = template_env.get_template("hydration.j2")
        prompt = tmpl.render(template_yaml=template_yaml, parameters=parameters)
        return self._extract_yaml(await self._call_llm(prompt))

    async def analyze(self, metrics: dict, intent_sla: dict) -> dict:
        tmpl = template_env.get_template("scaling_analysis.j2")
        prompt = tmpl.render(metrics=metrics, intent_sla=intent_sla)
        return {"recommendation": await self._call_llm(prompt)}

    @staticmethod
    def _extract_yaml(text: str) -> str:
        if "```yaml" in text:
            text = text.split("```yaml", 1)[1].split("```", 1)[0]
        elif "```" in text:
            text = text.split("```", 1)[1].split("```", 1)[0]
        return text.strip()
