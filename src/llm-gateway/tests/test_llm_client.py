"""Tests for LLM client."""
from llm_client import LLMClient

def test_extract_yaml_strips_fences():
    raw = "```yaml\napiVersion: v1\nkind: Pod\n```"
    assert LLMClient._extract_yaml(raw) == "apiVersion: v1\nkind: Pod"

def test_extract_yaml_plain():
    raw = "apiVersion: v1\nkind: Pod"
    assert LLMClient._extract_yaml(raw) == raw
