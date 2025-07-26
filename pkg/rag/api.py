from flask import Flask, request, jsonify
from telecom_pipeline import TelecomRAGPipeline
import os

app = Flask(__name__)

# This is a placeholder for the actual configuration.
# In a production environment, this should be loaded from a secure source.
config = {
    "weaviate_url": os.environ.get("WEAVIATE_URL", "http://localhost:8080"),
    "openai_api_key": os.environ.get("OPENAI_API_KEY")
}

# Initialize the RAG pipeline
try:
    rag_pipeline = TelecomRAGPipeline(config)
except Exception as e:
    # If the pipeline fails to initialize (e.g., missing API key),
    # we'll log the error and exit.
    app.logger.error(f"Failed to initialize TelecomRAGPipeline: {e}")
    rag_pipeline = None

@app.route('/process_intent', methods=['POST'])
def process_intent():
    if not rag_pipeline:
        return jsonify({"error": "RAG pipeline not initialized"}), 500

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
