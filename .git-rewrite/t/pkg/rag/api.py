from flask import Flask, request, jsonify
from telecom_pipeline import TelecomRAGPipeline
import os
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)

# In a production environment, this should be loaded from a secure source
# like a Kubernetes secret or a vault.
config = {
    "weaviate_url": os.environ.get("WEAVIATE_URL", "http://localhost:8080"),
    "openai_api_key": os.environ.get("OPENAI_API_KEY")
}

# Initialize the RAG pipeline
rag_pipeline = None
if not config.get("openai_api_key"):
    msg = "OPENAI_API_KEY not set. RAG will not be initialized."
    app.logger.warning(msg)
else:
    try:
        rag_pipeline = TelecomRAGPipeline(config)
        app.logger.info("TelecomRAGPipeline initialized successfully.")
    except Exception as e:
        app.logger.error(f"Failed to initialize TelecomRAGPipeline: {e}")


@app.route('/healthz', methods=['GET'])
def healthz():
    return jsonify({"status": "ok"})


@app.route('/readyz', methods=['GET'])
def readyz():
    if rag_pipeline:
        return jsonify({"status": "ready"})
    else:
        return jsonify({"status": "not_ready"}), 503


@app.route('/process_intent', methods=['POST'])
def process_intent():
    if not rag_pipeline:
        return jsonify({"error": "RAG pipeline not initialized"}), 503

    data = request.get_json()
    if not data or 'intent' not in data:
        return jsonify({"error": "Missing 'intent' in request body"}), 400

    intent = data['intent']
    try:
        result = rag_pipeline.process_intent(intent)
        return jsonify(result)
    except Exception as e:
        app.logger.error(f"Error processing intent: {e}")
        return jsonify({"error": "Failed to process intent"}), 500


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
