FROM python:3.11-slim
WORKDIR /app
COPY requirements-rag.txt .
RUN pip install --no-cache-dir -r requirements-rag.txt
COPY pkg/rag /app
EXPOSE 5001
CMD ["gunicorn", "--bind", "0.0.0.0:5001", "api:app"]
