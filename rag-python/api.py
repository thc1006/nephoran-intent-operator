from flask import Flask, request, jsonify
from flask_cors import CORS
from telecom_pipeline import TelecomRAGPipeline
from enhanced_pipeline import EnhancedTelecomRAGPipeline
from document_processor import DocumentProcessor
import os
import logging
import time
from pathlib import Path

app = Flask(__name__)
CORS(app, resources={r"/*": {"origins": "*"}}, supports_credentials=True)
logging.basicConfig(level=logging.INFO)

# Enhanced configuration with production-ready settings
config = {
    "weaviate_url": os.environ.get("WEAVIATE_URL", "http://weaviate:8080"),
    "openai_api_key": os.environ.get("OPENAI_API_KEY"),
    "model": os.environ.get("OPENAI_MODEL", "gpt-4o-mini"),
    "cache_max_size": int(os.environ.get("CACHE_MAX_SIZE", "1000")),
    "cache_ttl_seconds": int(os.environ.get("CACHE_TTL_SECONDS", "3600")),
    "chunk_size": int(os.environ.get("CHUNK_SIZE", "1000")),
    "chunk_overlap": int(os.environ.get("CHUNK_OVERLAP", "200")),
    "max_concurrent_files": int(os.environ.get("MAX_CONCURRENT_FILES", "5")),
    "knowledge_base_path": os.environ.get("KNOWLEDGE_BASE_PATH", "/app/knowledge_base")
}

# Initialize the enhanced RAG pipeline
rag_pipeline = None
document_processor = None

if not config.get("openai_api_key"):
    msg = "OPENAI_API_KEY not set. RAG will not be initialized."
    app.logger.warning(msg)
else:
    try:
        # Initialize enhanced pipeline
        rag_pipeline = EnhancedTelecomRAGPipeline(config)
        document_processor = DocumentProcessor(config)
        app.logger.info("Enhanced TelecomRAGPipeline initialized successfully.")
        
        # Auto-populate knowledge base if directory exists
        kb_path = Path(config["knowledge_base_path"])
        if kb_path.exists() and kb_path.is_dir():
            app.logger.info(f"Found knowledge base directory: {kb_path}")
            try:
                documents, metrics = document_processor.process_directory(kb_path)
                if documents:
                    success = rag_pipeline.add_knowledge_documents(documents)
                    if success:
                        app.logger.info(f"Populated knowledge base with {metrics.total_chunks} chunks "
                                      f"from {metrics.processed_documents} documents")
                    else:
                        app.logger.warning("Failed to populate knowledge base")
            except Exception as e:
                app.logger.error(f"Error populating knowledge base: {e}")
                
    except Exception as e:
        app.logger.error(f"Failed to initialize Enhanced RAG Pipeline: {e}")
        # Fallback to basic pipeline
        try:
            rag_pipeline = TelecomRAGPipeline(config)
            app.logger.info("Fallback: Basic TelecomRAGPipeline initialized.")
        except Exception as fallback_error:
            app.logger.error(f"Failed to initialize fallback pipeline: {fallback_error}")


@app.route('/healthz', methods=['GET'])
def healthz():
    return jsonify({"status": "ok"})


@app.route('/readyz', methods=['GET'])
def readyz():
    if rag_pipeline:
        try:
            # Check if enhanced pipeline is available
            if hasattr(rag_pipeline, 'get_health_status'):
                health = rag_pipeline.get_health_status()
                return jsonify({
                    "status": "ready" if health["status"] == "healthy" else "not_ready",
                    "health": health
                }), 200 if health["status"] == "healthy" else 503
            else:
                return jsonify({"status": "ready"}), 200
        except Exception as e:
            return jsonify({"status": "not_ready", "error": str(e)}), 503
    else:
        return jsonify({"status": "not_ready", "error": "Pipeline not initialized"}), 503


@app.route('/health', methods=['GET'])
def health():
    """Health endpoint alias for Go code compatibility"""
    app.logger.info("Health check requested via /health endpoint")
    return healthz()


def _process_intent_core(endpoint_name):
    """Core intent processing logic shared by both endpoints"""
    if not rag_pipeline:
        return jsonify({"error": "RAG pipeline not initialized"}), 503

    data = request.get_json()
    if not data or 'intent' not in data:
        return jsonify({"error": "Missing 'intent' in request body"}), 400

    intent = data['intent']
    intent_id = data.get('intent_id')  # Optional intent ID for tracking
    
    app.logger.info(f"Processing intent via {endpoint_name} endpoint. Intent ID: {intent_id}")
    
    try:
        # Use enhanced pipeline if available
        if hasattr(rag_pipeline, 'process_intent_async'):
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                result = loop.run_until_complete(
                    rag_pipeline.process_intent_async(intent, intent_id)
                )
                # Convert dataclass to dict for JSON serialization
                if hasattr(result, '__dict__'):
                    result_dict = result.__dict__.copy()
                    if 'metrics' in result_dict and hasattr(result_dict['metrics'], '__dict__'):
                        result_dict['metrics'] = result_dict['metrics'].__dict__
                    return jsonify(result_dict)
                else:
                    return jsonify(result)
            finally:
                loop.close()
        else:
            # Fallback to basic pipeline
            result = rag_pipeline.process_intent(intent)
            return jsonify(result)
            
    except Exception as e:
        app.logger.error(f"Error processing intent via {endpoint_name}: {e}")
        return jsonify({
            "error": "Failed to process intent",
            "intent_id": intent_id,
            "original_intent": intent,
            "exception": str(e)
        }), 500


@app.route('/process', methods=['POST'])
def process():
    """
    Primary intent processing endpoint for Go code compatibility.
    This is the preferred endpoint - /process_intent is maintained for legacy support.
    """
    return _process_intent_core('/process')


@app.route('/process_intent', methods=['POST'])
def process_intent():
    """
    Legacy intent processing endpoint - maintained for backward compatibility.
    New implementations should use /process endpoint.
    """
    return _process_intent_core('/process_intent')


@app.route('/stream', methods=['POST'])
def stream():
    """
    Server-Sent Events streaming endpoint for Go code compatibility.
    Provides real-time streaming of intent processing results.
    """
    if not rag_pipeline:
        return jsonify({"error": "RAG pipeline not initialized"}), 503

    data = request.get_json()
    if not data or 'intent' not in data:
        return jsonify({"error": "Missing 'intent' in request body"}), 400

    intent = data['intent']
    intent_id = data.get('intent_id')
    
    app.logger.info(f"Streaming intent processing via /stream endpoint. Intent ID: {intent_id}")
    
    def generate_stream():
        """Generator function for Server-Sent Events"""
        try:
            # Send initial status
            yield f"data: {jsonify({'status': 'started', 'intent_id': intent_id, 'timestamp': time.time()}).get_data(as_text=True)}\n\n"
            
            # Check if streaming is available in the enhanced pipeline
            if hasattr(rag_pipeline, 'process_intent_streaming'):
                # If streaming is implemented, use it
                for chunk in rag_pipeline.process_intent_streaming(intent, intent_id):
                    yield f"data: {jsonify(chunk).get_data(as_text=True)}\n\n"
            else:
                # Fallback: process normally and send result as single chunk
                yield f"data: {jsonify({'status': 'processing', 'message': 'Using fallback processing'}).get_data(as_text=True)}\n\n"
                
                # Use the same processing logic as the /process endpoint
                if hasattr(rag_pipeline, 'process_intent_async'):
                    import asyncio
                    loop = asyncio.new_event_loop()
                    asyncio.set_event_loop(loop)
                    try:
                        result = loop.run_until_complete(
                            rag_pipeline.process_intent_async(intent, intent_id)
                        )
                        # Convert dataclass to dict for JSON serialization
                        if hasattr(result, '__dict__'):
                            result_dict = result.__dict__.copy()
                            if 'metrics' in result_dict and hasattr(result_dict['metrics'], '__dict__'):
                                result_dict['metrics'] = result_dict['metrics'].__dict__
                            yield f"data: {jsonify({'status': 'completed', 'result': result_dict}).get_data(as_text=True)}\n\n"
                        else:
                            yield f"data: {jsonify({'status': 'completed', 'result': result}).get_data(as_text=True)}\n\n"
                    finally:
                        loop.close()
                else:
                    # Fallback to basic pipeline
                    result = rag_pipeline.process_intent(intent)
                    yield f"data: {jsonify({'status': 'completed', 'result': result}).get_data(as_text=True)}\n\n"
                    
        except Exception as e:
            app.logger.error(f"Error in streaming processing: {e}")
            error_data = {
                'status': 'error',
                'error': 'Failed to process intent',
                'intent_id': intent_id,
                'exception': str(e)
            }
            yield f"data: {jsonify(error_data).get_data(as_text=True)}\n\n"

    # Return Server-Sent Events response
    from flask import Response
    return Response(
        generate_stream(),
        mimetype='text/event-stream',
        headers={
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive',
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Headers': 'Cache-Control'
        }
    )


@app.route('/knowledge/upload', methods=['POST'])
def upload_knowledge():
    """Upload and process knowledge documents"""
    if not document_processor or not rag_pipeline:
        return jsonify({"error": "Document processor not available"}), 503
    
    try:
        files = request.files.getlist('files')
        if not files:
            return jsonify({"error": "No files provided"}), 400
        
        # Get page_size parameter if provided
        page_size = request.args.get('page_size', type=int)
        
        # Create a custom document processor if page_size is specified
        processor = document_processor
        if page_size:
            # Create a temporary config with custom chunk_size
            custom_config = config.copy()
            custom_config['chunk_size'] = page_size
            processor = DocumentProcessor(custom_config)
        
        # Process uploaded files
        documents = []
        for file in files:
            if file.filename:
                # Save temporary file
                temp_path = Path(f"/tmp/{file.filename}")
                file.save(temp_path)
                
                try:
                    # Detect and process document
                    doc_type = processor.detect_document_type(temp_path)
                    document = processor.load_document(temp_path, doc_type)
                    
                    if document:
                        processed_docs = processor.process_document(document)
                        documents.extend(processed_docs)
                finally:
                    # Clean up temporary file
                    if temp_path.exists():
                        temp_path.unlink()
        
        # Add to knowledge base
        if documents:
            success = rag_pipeline.add_knowledge_documents(documents)
            return jsonify({
                "status": "success" if success else "failed",
                "documents_processed": len(documents),
                "chunks_created": len(documents),
                "chunk_size_used": page_size if page_size else config.get('chunk_size', 1000)
            })
        else:
            return jsonify({"error": "No valid documents processed"}), 400
            
    except Exception as e:
        app.logger.error(f"Error uploading knowledge: {e}")
        return jsonify({"error": f"Failed to upload knowledge: {e}"}), 500


@app.route('/stats', methods=['GET'])
def get_stats():
    """Get system statistics and metrics"""
    try:
        stats = {
            "timestamp": time.time(),
            "config": {
                "model": config.get("model"),
                "weaviate_url": config.get("weaviate_url"),
                "cache_enabled": config.get("cache_max_size", 0) > 0,
                "knowledge_base_path": config.get("knowledge_base_path")
            }
        }
        
        # Add enhanced pipeline stats
        if hasattr(rag_pipeline, 'get_health_status'):
            stats["health"] = rag_pipeline.get_health_status()
        
        # Add document processor stats
        if document_processor:
            stats["document_processor"] = document_processor.get_processing_stats()
        
        return jsonify(stats)
        
    except Exception as e:
        app.logger.error(f"Error getting stats: {e}")
        return jsonify({"error": f"Failed to get stats: {e}"}), 500


@app.route('/knowledge/populate', methods=['POST'])
def populate_knowledge():
    """Manually trigger knowledge base population"""
    if not document_processor or not rag_pipeline:
        return jsonify({"error": "Services not available"}), 503
    
    try:
        data = request.get_json() or {}
        kb_path = Path(data.get("path", config["knowledge_base_path"]))
        
        if not kb_path.exists():
            return jsonify({"error": f"Knowledge base path does not exist: {kb_path}"}), 400
        
        # Process directory
        documents, metrics = document_processor.process_directory(kb_path)
        
        if documents:
            success = rag_pipeline.add_knowledge_documents(documents)
            return jsonify({
                "status": "success" if success else "failed",
                "metrics": {
                    "total_documents": metrics.total_documents,
                    "processed_documents": metrics.processed_documents,
                    "failed_documents": metrics.failed_documents,
                    "total_chunks": metrics.total_chunks,
                    "processing_time_seconds": metrics.processing_time_seconds,
                    "keywords_extracted": metrics.keywords_extracted
                }
            })
        else:
            return jsonify({"error": "No documents found to process"}), 400
            
    except Exception as e:
        app.logger.error(f"Error populating knowledge: {e}")
        return jsonify({"error": f"Failed to populate knowledge: {e}"}), 500


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
