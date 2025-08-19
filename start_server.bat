@echo off
set JWT_SECRET_KEY=test123
set LLM_API_KEY=test
set LLM_BACKEND_TYPE=openai
set LLM_ALLOWED_ORIGINS=*
set LLM_MODEL_NAME=gpt-4
set LLM_PROCESSOR_URL=http://localhost:8090
echo Starting LLM Processor Server on port 8090...
llm-processor.exe -listen :8090