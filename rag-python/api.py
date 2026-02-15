"""
Nephoran RAG Service - FastAPI Application
Production-ready REST API for telecom intent processing with RAG pipeline
"""

from fastapi import FastAPI, HTTPException, Request, UploadFile, File
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel, Field
from typing import Optional, List, Dict, Any
import os
import logging
import time
import asyncio
from pathlib import Path

from telecom_pipeline import TelecomRAGPipeline
from enhanced_pipeline import EnhancedTelecomRAGPipeline
from document_processor import DocumentProcessor

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Nephoran RAG Service",
    description="Intent processing with RAG pipeline for O-RAN network functions",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc"
)

# CORS configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Enhanced configuration with production-ready settings
config = {
    "weaviate_url": os.environ.get("WEAVIATE_URL", "http://weaviate.weaviate.svc.cluster.local:80"),
    "weaviate_api_key": os.environ.get("WEAVIATE_API_KEY"),
    "openai_api_key": os.environ.get("OPENAI_API_KEY"),
    "llm_provider": os.environ.get("LLM_PROVIDER", "ollama"),  # "openai" or "ollama"
    "llm_model": os.environ.get("LLM_MODEL", "llama3.1:8b-instruct-q5_K_M"),
    "embedding_model": os.environ.get("EMBEDDING_MODEL", ""),  # defaults to llm_model if empty
    "ollama_base_url": os.environ.get("OLLAMA_BASE_URL", "http://localhost:11434"),
    "cache_max_size": int(os.environ.get("CACHE_MAX_SIZE", "1000")),
    "cache_ttl_seconds": int(os.environ.get("CACHE_TTL_SECONDS", "3600")),
    "chunk_size": int(os.environ.get("CHUNK_SIZE", "1000")),
    "chunk_overlap": int(os.environ.get("CHUNK_OVERLAP", "200")),
    "max_concurrent_files": int(os.environ.get("MAX_CONCURRENT_FILES", "5")),
    "knowledge_base_path": os.environ.get("KNOWLEDGE_BASE_PATH", "/app/knowledge_base")
}

# If embedding_model is empty, default to llm_model
if not config["embedding_model"]:
    config["embedding_model"] = config["llm_model"]

# Initialize the RAG pipeline (global state)
rag_pipeline: Optional[EnhancedTelecomRAGPipeline] = None
document_processor: Optional[DocumentProcessor] = None


# Pydantic models for request/response validation
class IntentRequest(BaseModel):
    """Request model for intent processing"""
    intent: str = Field(..., description="Natural language intent to process", min_length=1)
    intent_id: Optional[str] = Field(None, description="Optional intent ID for tracking")

    class Config:
        json_schema_extra = {
            "example": {
                "intent": "Deploy AMF with 5 replicas in namespace 5g-core",
                "intent_id": "intent-12345"
            }
        }


class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    timestamp: Optional[float] = None
    health: Optional[Dict[str, Any]] = None
    error: Optional[str] = None


class StatsResponse(BaseModel):
    """Statistics response"""
    timestamp: float
    config: Dict[str, Any]
    health: Optional[Dict[str, Any]] = None
    document_processor: Optional[Dict[str, Any]] = None


@app.on_event("startup")
async def startup_event():
    """Initialize services on startup"""
    global rag_pipeline, document_processor

    provider = config.get("llm_provider", "ollama")

    # Check required keys based on provider
    if provider == "openai" and not config.get("openai_api_key"):
        logger.warning("OPENAI_API_KEY not set. RAG pipeline will not be initialized.")
        return
    elif provider == "ollama":
        logger.info(f"Using Ollama provider with model: {config.get('llm_model', 'llama3.1:8b-instruct-q5_K_M')}")
        logger.info(f"Ollama base URL: {config.get('ollama_base_url', 'http://localhost:11434')}")
        logger.info(f"Embedding model: {config.get('embedding_model', config.get('llm_model'))}")

    try:
        # Initialize enhanced pipeline
        rag_pipeline = EnhancedTelecomRAGPipeline(config)
        document_processor = DocumentProcessor(config)
        logger.info("Enhanced TelecomRAGPipeline initialized successfully.")

        # Auto-populate knowledge base if directory exists
        kb_path = Path(config["knowledge_base_path"])
        if kb_path.exists() and kb_path.is_dir():
            logger.info(f"Found knowledge base directory: {kb_path}")
            try:
                documents, metrics = document_processor.process_directory(kb_path)
                if documents:
                    success = rag_pipeline.add_knowledge_documents(documents)
                    if success:
                        logger.info(
                            f"Populated knowledge base with {metrics.total_chunks} chunks "
                            f"from {metrics.processed_documents} documents"
                        )
                    else:
                        logger.warning("Failed to populate knowledge base")
            except Exception as e:
                logger.error(f"Error populating knowledge base: {e}")

    except Exception as e:
        logger.error(f"Failed to initialize Enhanced RAG Pipeline: {e}")
        # Fallback to basic pipeline
        try:
            rag_pipeline = TelecomRAGPipeline(config)
            logger.info("Fallback: Basic TelecomRAGPipeline initialized.")
        except Exception as fallback_error:
            logger.error(f"Failed to initialize fallback pipeline: {fallback_error}")


@app.get("/healthz", response_model=HealthResponse, tags=["Health"])
async def healthz():
    """Basic health check endpoint"""
    return HealthResponse(status="ok", timestamp=time.time())


@app.get("/readyz", response_model=HealthResponse, tags=["Health"])
async def readyz():
    """Readiness probe endpoint"""
    if rag_pipeline:
        try:
            # Check if enhanced pipeline is available
            if hasattr(rag_pipeline, 'get_health_status'):
                health = rag_pipeline.get_health_status()
                is_healthy = health["status"] == "healthy"
                return HealthResponse(
                    status="ready" if is_healthy else "not_ready",
                    health=health,
                    timestamp=time.time()
                )
            else:
                return HealthResponse(status="ready", timestamp=time.time())
        except Exception as e:
            raise HTTPException(status_code=503, detail={"status": "not_ready", "error": str(e)})
    else:
        raise HTTPException(status_code=503, detail={"status": "not_ready", "error": "Pipeline not initialized"})


@app.get("/health", response_model=HealthResponse, tags=["Health"])
async def health():
    """Health endpoint alias for Go code compatibility"""
    logger.info("Health check requested via /health endpoint")
    return await healthz()


@app.post("/process", tags=["Intent Processing"])
async def process(request: IntentRequest) -> Dict[str, Any]:
    """
    Primary intent processing endpoint.

    Processes natural language intents and returns structured NetworkIntent JSON.
    """
    if not rag_pipeline:
        raise HTTPException(status_code=503, detail={"error": "RAG pipeline not initialized"})

    logger.info(f"Processing intent via /process endpoint. Intent ID: {request.intent_id}")

    try:
        # Use enhanced pipeline if available
        if hasattr(rag_pipeline, 'process_intent_async'):
            result = await rag_pipeline.process_intent_async(request.intent, request.intent_id)

            # Convert dataclass to dict for JSON serialization
            if hasattr(result, '__dict__'):
                result_dict = result.__dict__.copy()
                if 'metrics' in result_dict and hasattr(result_dict['metrics'], '__dict__'):
                    result_dict['metrics'] = result_dict['metrics'].__dict__
                if 'status' in result_dict and hasattr(result_dict['status'], 'value'):
                    result_dict['status'] = result_dict['status'].value
                return result_dict
            else:
                return result
        else:
            # Fallback to basic pipeline (synchronous)
            loop = asyncio.get_event_loop()
            result = await loop.run_in_executor(
                None,
                lambda: rag_pipeline.process_intent(request.intent)
            )
            return result

    except Exception as e:
        logger.error(f"Error processing intent via /process: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "error": "Failed to process intent",
                "intent_id": request.intent_id,
                "original_intent": request.intent,
                "exception": str(e)
            }
        )


@app.post("/process_intent", tags=["Intent Processing"])
async def process_intent(request: IntentRequest) -> Dict[str, Any]:
    """
    Legacy intent processing endpoint.

    Maintained for backward compatibility. New implementations should use /process.
    """
    return await process(request)


@app.post("/stream", tags=["Intent Processing"])
async def stream(request: IntentRequest):
    """
    Server-Sent Events streaming endpoint.

    Provides real-time streaming of intent processing results.
    """
    if not rag_pipeline:
        raise HTTPException(status_code=503, detail={"error": "RAG pipeline not initialized"})

    logger.info(f"Streaming intent processing via /stream endpoint. Intent ID: {request.intent_id}")

    async def generate_stream():
        """Generator function for Server-Sent Events"""
        try:
            # Send initial status
            yield f"data: {{'status': 'started', 'intent_id': '{request.intent_id}', 'timestamp': {time.time()}}}\n\n"

            # Check if streaming is available in the enhanced pipeline
            if hasattr(rag_pipeline, 'process_intent_streaming'):
                # If streaming is implemented, use it
                async for chunk in rag_pipeline.process_intent_streaming(request.intent, request.intent_id):
                    yield f"data: {chunk}\n\n"
            else:
                # Fallback: process normally and send result as single chunk
                yield f"data: {{'status': 'processing', 'message': 'Using fallback processing'}}\n\n"

                # Use the same processing logic as the /process endpoint
                if hasattr(rag_pipeline, 'process_intent_async'):
                    result = await rag_pipeline.process_intent_async(request.intent, request.intent_id)

                    # Convert dataclass to dict
                    if hasattr(result, '__dict__'):
                        result_dict = result.__dict__.copy()
                        if 'metrics' in result_dict and hasattr(result_dict['metrics'], '__dict__'):
                            result_dict['metrics'] = result_dict['metrics'].__dict__
                        if 'status' in result_dict and hasattr(result_dict['status'], 'value'):
                            result_dict['status'] = result_dict['status'].value
                        yield f"data: {{'status': 'completed', 'result': {result_dict}}}\n\n"
                    else:
                        yield f"data: {{'status': 'completed', 'result': {result}}}\n\n"
                else:
                    # Fallback to basic pipeline
                    loop = asyncio.get_event_loop()
                    result = await loop.run_in_executor(
                        None,
                        lambda: rag_pipeline.process_intent(request.intent)
                    )
                    yield f"data: {{'status': 'completed', 'result': {result}}}\n\n"

        except Exception as e:
            logger.error(f"Error in streaming processing: {e}")
            error_data = {
                'status': 'error',
                'error': 'Failed to process intent',
                'intent_id': request.intent_id,
                'exception': str(e)
            }
            yield f"data: {error_data}\n\n"

    return StreamingResponse(
        generate_stream(),
        media_type='text/event-stream',
        headers={
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive',
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Headers': 'Cache-Control'
        }
    )


@app.post("/knowledge/upload", tags=["Knowledge Management"])
async def upload_knowledge(
    files: List[UploadFile] = File(...),
    page_size: Optional[int] = None
) -> Dict[str, Any]:
    """Upload and process knowledge documents"""
    if not document_processor or not rag_pipeline:
        raise HTTPException(status_code=503, detail={"error": "Document processor not available"})

    try:
        if not files:
            raise HTTPException(status_code=400, detail={"error": "No files provided"})

        # Create a custom document processor if page_size is specified
        processor = document_processor
        if page_size:
            custom_config = config.copy()
            custom_config['chunk_size'] = page_size
            processor = DocumentProcessor(custom_config)

        # Process uploaded files
        documents = []
        for file in files:
            if file.filename:
                # Save temporary file securely
                import tempfile
                with tempfile.NamedTemporaryFile(
                    delete=False,
                    suffix=Path(file.filename).suffix
                ) as tmp_file:
                    temp_path = Path(tmp_file.name)

                # Read and save file content
                content = await file.read()
                temp_path.write_bytes(content)

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
            return {
                "status": "success" if success else "failed",
                "documents_processed": len(documents),
                "chunks_created": len(documents),
                "chunk_size_used": page_size if page_size else config.get('chunk_size', 1000)
            }
        else:
            raise HTTPException(status_code=400, detail={"error": "No valid documents processed"})

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error uploading knowledge: {e}")
        raise HTTPException(status_code=500, detail={"error": f"Failed to upload knowledge: {e}"})


@app.get("/stats", response_model=StatsResponse, tags=["Monitoring"])
async def get_stats():
    """Get system statistics and metrics"""
    try:
        stats = {
            "timestamp": time.time(),
            "config": {
                "llm_provider": config.get("llm_provider"),
                "llm_model": config.get("llm_model"),
                "ollama_base_url": config.get("ollama_base_url") if config.get("llm_provider") == "ollama" else None,
                "weaviate_url": config.get("weaviate_url"),
                "cache_enabled": config.get("cache_max_size", 0) > 0,
                "knowledge_base_path": config.get("knowledge_base_path")
            }
        }

        # Add enhanced pipeline stats
        if rag_pipeline and hasattr(rag_pipeline, 'get_health_status'):
            stats["health"] = rag_pipeline.get_health_status()

        # Add document processor stats
        if document_processor:
            stats["document_processor"] = document_processor.get_processing_stats()

        return stats

    except Exception as e:
        logger.error(f"Error getting stats: {e}")
        raise HTTPException(status_code=500, detail={"error": f"Failed to get stats: {e}"})


@app.post("/knowledge/populate", tags=["Knowledge Management"])
async def populate_knowledge(path: Optional[str] = None) -> Dict[str, Any]:
    """Manually trigger knowledge base population"""
    if not document_processor or not rag_pipeline:
        raise HTTPException(status_code=503, detail={"error": "Services not available"})

    try:
        kb_path = Path(path if path else config["knowledge_base_path"])

        if not kb_path.exists():
            raise HTTPException(
                status_code=400,
                detail={"error": f"Knowledge base path does not exist: {kb_path}"}
            )

        # Process directory
        documents, metrics = document_processor.process_directory(kb_path)

        if documents:
            success = rag_pipeline.add_knowledge_documents(documents)
            return {
                "status": "success" if success else "failed",
                "metrics": {
                    "total_documents": metrics.total_documents,
                    "processed_documents": metrics.processed_documents,
                    "failed_documents": metrics.failed_documents,
                    "total_chunks": metrics.total_chunks,
                    "processing_time_seconds": metrics.processing_time_seconds,
                    "keywords_extracted": metrics.keywords_extracted
                }
            }
        else:
            raise HTTPException(status_code=400, detail={"error": "No documents found to process"})

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error populating knowledge: {e}")
        raise HTTPException(status_code=500, detail={"error": f"Failed to populate knowledge: {e}"})


if __name__ == '__main__':
    import uvicorn

    # Run with uvicorn
    debug_mode = os.environ.get('FASTAPI_DEBUG', '').lower() in ('1', 'true', 'yes')
    uvicorn.run(
        "api:app",
        host="0.0.0.0",
        port=8000,
        workers=1 if debug_mode else 4,
        reload=debug_mode,
        log_level="info"
    )
