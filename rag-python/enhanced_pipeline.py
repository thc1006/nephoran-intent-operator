"""
Enhanced RAG Pipeline for Nephoran Intent Operator
Production-ready telecom domain-specific RAG system with advanced features
"""

import asyncio
import logging
import time
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, asdict
from enum import Enum
import json
import hashlib
import uuid

from langchain_community.vectorstores import Weaviate
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from langchain.chains import RetrievalQA
from langchain.prompts import PromptTemplate
from langchain.schema import Document
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain.callbacks import AsyncCallbackHandler
import weaviate as wv
import openai

# Import Ollama support
try:
    from langchain_ollama import ChatOllama
    OLLAMA_AVAILABLE = True
except ImportError:
    OLLAMA_AVAILABLE = False
    logging.warning("langchain-ollama not installed. Ollama support disabled.")


class ProcessingStatus(Enum):
    """Processing status for intent handling"""
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"
    CACHED = "cached"


@dataclass
class IntentMetrics:
    """Metrics for intent processing performance"""
    processing_time_ms: float
    tokens_used: int
    retrieval_score: float
    confidence_score: float
    cache_hit: bool
    model_version: str


@dataclass
class ProcessedIntent:
    """Structured result from intent processing"""
    intent_id: str
    original_intent: str
    structured_output: Dict[str, Any]
    status: ProcessingStatus
    metrics: IntentMetrics
    timestamp: float
    error_message: Optional[str] = None


class TelecomKnowledgeManager:
    """Manages telecom-specific knowledge base operations"""
    
    def __init__(self, weaviate_client: wv.Client, embeddings: OpenAIEmbeddings):
        self.client = weaviate_client
        self.embeddings = embeddings
        self.logger = logging.getLogger(__name__)
        
    def initialize_schema(self) -> bool:
        """Initialize Weaviate schema for telecom knowledge"""
        try:
            schema = {
                "classes": [
                    {
                        "class": "TelecomKnowledge",
                        "description": "Telecommunications domain knowledge base",
                        "vectorizer": "text2vec-openai",
                        "moduleConfig": {
                            "text2vec-openai": {
                                "model": "text-embedding-3-large",
                                "dimensions": 3072,
                                "type": "text"
                            }
                        },
                        "properties": [
                            {
                                "name": "content",
                                "dataType": ["text"],
                                "description": "Document content",
                                "moduleConfig": {
                                    "text2vec-openai": {
                                        "skip": False,
                                        "vectorizePropertyName": False
                                    }
                                }
                            },
                            {
                                "name": "source",
                                "dataType": ["text"],
                                "description": "Document source (3GPP, O-RAN, etc.)"
                            },
                            {
                                "name": "category",
                                "dataType": ["text"],
                                "description": "Knowledge category"
                            },
                            {
                                "name": "version",
                                "dataType": ["text"],
                                "description": "Specification version"
                            },
                            {
                                "name": "keywords",
                                "dataType": ["text[]"],
                                "description": "Extracted keywords"
                            },
                            {
                                "name": "confidence",
                                "dataType": ["number"],
                                "description": "Content confidence score"
                            }
                        ]
                    }
                ]
            }
            
            # Check if schema already exists
            existing_schema = self.client.schema.get()
            class_names = [c["class"] for c in existing_schema.get("classes", [])]
            
            if "TelecomKnowledge" not in class_names:
                self.client.schema.create(schema)
                self.logger.info("Created TelecomKnowledge schema")
            else:
                self.logger.info("TelecomKnowledge schema already exists")
            
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to initialize schema: {e}")
            return False
    
    def add_documents(self, documents: List[Document], batch_size: int = 50) -> bool:
        """Add documents to knowledge base with batch processing"""
        try:
            text_splitter = RecursiveCharacterTextSplitter(
                chunk_size=1000,
                chunk_overlap=200,
                length_function=len,
                separators=["\n\n", "\n", ". ", " ", ""]
            )
            
            # Split documents into chunks
            chunks = []
            for doc in documents:
                doc_chunks = text_splitter.split_text(doc.page_content)
                for i, chunk in enumerate(doc_chunks):
                    chunk_doc = Document(
                        page_content=chunk,
                        metadata={
                            **doc.metadata,
                            "chunk_id": f"{doc.metadata.get('id', 'unknown')}_{i}",
                            "chunk_index": i,
                            "total_chunks": len(doc_chunks)
                        }
                    )
                    chunks.append(chunk_doc)
            
            # Process in batches
            for i in range(0, len(chunks), batch_size):
                batch = chunks[i:i + batch_size]
                self._add_batch(batch)
                self.logger.info(f"Processed batch {i//batch_size + 1}/{(len(chunks)-1)//batch_size + 1}")
            
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to add documents: {e}")
            return False
    
    def _add_batch(self, documents: List[Document]) -> None:
        """Add a batch of documents to Weaviate"""
        with self.client.batch as batch:
            batch.batch_size = len(documents)
            
            for doc in documents:
                properties = {
                    "content": doc.page_content,
                    "source": doc.metadata.get("source", "unknown"),
                    "category": doc.metadata.get("category", "general"),
                    "version": doc.metadata.get("version", "1.0"),
                    "keywords": doc.metadata.get("keywords", []),
                    "confidence": doc.metadata.get("confidence", 1.0)
                }
                
                batch.add_data_object(
                    data_object=properties,
                    class_name="TelecomKnowledge",
                    uuid=doc.metadata.get("uuid", str(uuid.uuid4()))
                )


class CacheManager:
    """Manages caching for processed intents"""
    
    def __init__(self, max_size: int = 1000, ttl_seconds: int = 3600):
        self.cache: Dict[str, Tuple[ProcessedIntent, float]] = {}
        self.max_size = max_size
        self.ttl_seconds = ttl_seconds
        self.logger = logging.getLogger(__name__)
    
    def _generate_cache_key(self, intent: str) -> str:
        """Generate cache key from intent"""
        return hashlib.sha256(intent.encode()).hexdigest()
    
    def get(self, intent: str) -> Optional[ProcessedIntent]:
        """Get cached result if available and not expired"""
        cache_key = self._generate_cache_key(intent)
        
        if cache_key in self.cache:
            result, timestamp = self.cache[cache_key]
            
            # Check if expired
            if time.time() - timestamp > self.ttl_seconds:
                del self.cache[cache_key]
                return None
            
            # Update metrics to reflect cache hit
            result.metrics.cache_hit = True
            result.status = ProcessingStatus.CACHED
            
            return result
        
        return None
    
    def set(self, intent: str, result: ProcessedIntent) -> None:
        """Cache processing result"""
        cache_key = self._generate_cache_key(intent)
        
        # Implement LRU eviction if cache is full
        if len(self.cache) >= self.max_size:
            # Remove oldest entry
            oldest_key = min(self.cache.keys(), key=lambda k: self.cache[k][1])
            del self.cache[oldest_key]
        
        self.cache[cache_key] = (result, time.time())
        self.logger.debug(f"Cached result for intent: {intent[:50]}...")


class EnhancedTelecomRAGPipeline:
    """Enhanced production-ready RAG pipeline for telecom domain"""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.logger = self._setup_logging()
        self.cache = CacheManager(
            max_size=config.get("cache_max_size", 1000),
            ttl_seconds=config.get("cache_ttl_seconds", 3600)
        )
        
        # Initialize components
        self._initialize_components()
        
        # Initialize knowledge manager
        self.knowledge_manager = TelecomKnowledgeManager(
            self.weaviate_client, self.embeddings
        )
        
        # Initialize schema
        if not self.knowledge_manager.initialize_schema():
            raise RuntimeError("Failed to initialize knowledge base schema")
    
    def _setup_logging(self) -> logging.Logger:
        """Setup structured logging"""
        logger = logging.getLogger(__name__)
        logger.setLevel(logging.INFO)

        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)

        return logger

    def _initialize_llm(self):
        """Initialize LLM based on provider configuration"""
        provider = self.config.get("llm_provider", "openai").lower()

        if provider == "ollama":
            if not OLLAMA_AVAILABLE:
                raise ImportError("langchain-ollama is not installed. Install with: pip install langchain-ollama")

            self.logger.info(f"Initializing Ollama LLM: {self.config.get('llm_model', 'llama2')}")

            return ChatOllama(
                model=self.config.get("llm_model", "llama2"),
                base_url=self.config.get("ollama_base_url", "http://localhost:11434"),
                temperature=0,
                num_predict=2048,
                format="json",  # Request JSON output
            )

        elif provider == "openai":
            if not self.config.get("openai_api_key"):
                raise ValueError("OpenAI API key is required for OpenAI provider")

            self.logger.info(f"Initializing OpenAI LLM: {self.config.get('llm_model', 'gpt-4o-mini')}")

            return ChatOpenAI(
                model=self.config.get("llm_model", "gpt-4o-mini"),
                temperature=0,
                max_tokens=2048,
                model_kwargs={"response_format": {"type": "json_object"}},
                openai_api_key=self.config["openai_api_key"]
            )

        else:
            raise ValueError(f"Unsupported LLM provider: {provider}. Choose 'openai' or 'ollama'")
    
    def _initialize_components(self) -> None:
        """Initialize LLM and vector store components"""
        try:
            # Initialize embeddings (always use OpenAI for consistency)
            self.embeddings = OpenAIEmbeddings(
                model="text-embedding-3-large",
                dimensions=3072,
                openai_api_key=self.config.get("openai_api_key", "not-needed")
            )

            # Initialize LLM based on provider
            self.llm = self._initialize_llm()
            
            # Initialize Weaviate client with authentication
            weaviate_url = self.config.get("weaviate_url", "http://weaviate.nephoran-system.svc.cluster.local:8080")
            weaviate_api_key = self.config.get("weaviate_api_key", "nephoran-rag-key-production")
            
            self.weaviate_client = wv.Client(
                url=weaviate_url,
                additional_headers={
                    "X-OpenAI-Api-Key": self.config["openai_api_key"]
                },
                auth_client_secret=wv.AuthApiKey(api_key=weaviate_api_key)
            )
            
            # Initialize vector store
            self.vector_store = Weaviate(
                client=self.weaviate_client,
                index_name="TelecomKnowledge",
                text_key="content",
                embedding=self.embeddings,
                by_text=False
            )
            
            # Create QA chain with enhanced prompt
            self.qa_chain = self._create_enhanced_qa_chain()
            
            self.logger.info("Enhanced RAG pipeline initialized successfully")
            
        except Exception as e:
            self.logger.error(f"Failed to initialize components: {e}")
            raise RuntimeError(f"Failed to initialize RAG pipeline: {e}")
    
    def _create_enhanced_qa_chain(self) -> RetrievalQA:
        """Create enhanced QA chain with telecom-specific optimizations"""
        template = """
You are an expert telecommunications network engineer responsible for translating 
natural language network operations commands into structured JSON objects for 
5G and O-RAN network functions.

Your task is to analyze the user's command and determine the appropriate action type:
1. "NetworkFunctionDeployment" - for deploying new network functions
2. "NetworkFunctionScale" - for scaling existing network functions
3. "NetworkSliceConfiguration" - for network slice management
4. "PolicyConfiguration" - for A1 policy management

User Command: "{question}"

Context from Knowledge Base:
{context}

Guidelines:
- Use the context to fill in default values and validate parameters
- Ensure all resource specifications align with 3GPP and O-RAN standards
- Include appropriate O1 configuration (FCAPS) and A1 policies when relevant
- For scaling operations, preserve existing configuration parameters

Output MUST be a single, valid JSON object without explanation.

For NetworkFunctionDeployment:
{{
  "type": "NetworkFunctionDeployment",
  "name": "string",
  "namespace": "string",
  "spec": {{
    "replicas": "integer",
    "image": "string",
    "resources": {{
      "requests": {{"cpu": "string", "memory": "string"}},
      "limits": {{"cpu": "string", "memory": "string"}}
    }},
    "ports": [{{"name": "string", "port": "integer", "protocol": "string"}}],
    "env": [{{"name": "string", "value": "string"}}]
  }},
  "o1_config": {{
    "management_endpoint": "string",
    "fcaps_config": "object"
  }},
  "a1_policy": {{
    "policy_type_id": "string",
    "policy_data": "object"
  }},
  "network_slice": {{
    "slice_id": "string",
    "slice_type": "string",
    "sla_parameters": "object"
  }}
}}

For NetworkFunctionScale:
{{
  "type": "NetworkFunctionScale",
  "name": "string",
  "namespace": "string", 
  "replicas": "integer",
  "resource_adjustments": {{
    "cpu_scale_factor": "number",
    "memory_scale_factor": "number"
  }}
}}

For NetworkSliceConfiguration:
{{
  "type": "NetworkSliceConfiguration",
  "slice_id": "string",
  "name": "string",
  "slice_type": "eMBB|URLLC|mMTC",
  "sla_parameters": {{
    "latency_ms": "integer",
    "throughput_mbps": "integer",
    "reliability": "number"
  }},
  "network_functions": ["string"],
  "policies": ["string"]
}}

For PolicyConfiguration:
{{
  "type": "PolicyConfiguration",
  "policy_name": "string",
  "policy_type_id": "string",
  "scope": {{
    "ue_id": "string",
    "cell_id": "string",
    "slice_id": "string"
  }},
  "policy_data": "object"
}}

Generate the appropriate JSON object based on the user command.
"""
        
        prompt = PromptTemplate(
            template=template,
            input_variables=["context", "question"]
        )
        
        return RetrievalQA.from_chain_type(
            llm=self.llm,
            chain_type="stuff",
            retriever=self.vector_store.as_retriever(
                search_type="mmr",
                search_kwargs={
                    "k": 5,
                    "fetch_k": 20,
                    "lambda_mult": 0.7
                }
            ),
            chain_type_kwargs={"prompt": prompt},
            return_source_documents=True
        )
    
    async def process_intent_async(self, intent: str, intent_id: Optional[str] = None) -> ProcessedIntent:
        """Asynchronously process user intent with enhanced features"""
        if not intent_id:
            intent_id = str(uuid.uuid4())
        
        start_time = time.time()
        
        # Check cache first
        cached_result = self.cache.get(intent)
        if cached_result:
            self.logger.info(f"Cache hit for intent: {intent_id}")
            return cached_result
        
        try:
            # Validate input
            if not intent or not intent.strip():
                return self._create_error_result(
                    intent_id, intent, "Empty intent provided", start_time
                )
            
            # Process with QA chain
            result = await asyncio.get_event_loop().run_in_executor(
                None, lambda: self.qa_chain.invoke({"query": intent})
            )
            
            # Extract and parse result
            result_text = result.get('result', '')
            source_documents = result.get('source_documents', [])
            
            if not result_text:
                return self._create_error_result(
                    intent_id, intent, "Empty response from LLM", start_time
                )
            
            # Parse JSON response
            try:
                structured_output = json.loads(result_text)
            except json.JSONDecodeError as e:
                return self._create_error_result(
                    intent_id, intent, f"Invalid JSON from LLM: {e}", start_time, result_text
                )
            
            # Calculate metrics
            processing_time = (time.time() - start_time) * 1000
            retrieval_score = self._calculate_retrieval_score(source_documents)
            confidence_score = self._calculate_confidence_score(structured_output, source_documents)
            
            metrics = IntentMetrics(
                processing_time_ms=processing_time,
                tokens_used=self._estimate_tokens(intent + result_text),
                retrieval_score=retrieval_score,
                confidence_score=confidence_score,
                cache_hit=False,
                model_version=self.config.get("llm_model", "gpt-4o-mini")
            )
            
            # Validate response structure
            if not self._validate_enhanced_response(structured_output):
                return self._create_error_result(
                    intent_id, intent, "Invalid response structure", start_time, result_text
                )
            
            # Add metadata
            structured_output['original_intent'] = intent
            structured_output['intent_id'] = intent_id
            structured_output['timestamp'] = time.time()
            structured_output['source_documents'] = len(source_documents)
            
            # Create successful result
            processed_result = ProcessedIntent(
                intent_id=intent_id,
                original_intent=intent,
                structured_output=structured_output,
                status=ProcessingStatus.COMPLETED,
                metrics=metrics,
                timestamp=time.time()
            )
            
            # Cache the result
            self.cache.set(intent, processed_result)
            
            self.logger.info(f"Successfully processed intent {intent_id} in {processing_time:.2f}ms")
            return processed_result
            
        except Exception as e:
            self.logger.error(f"Unexpected error processing intent {intent_id}: {e}")
            return self._create_error_result(
                intent_id, intent, f"Unexpected error: {e}", start_time
            )
    
    def process_intent(self, intent: str, intent_id: Optional[str] = None) -> Dict[str, Any]:
        """Synchronous wrapper for intent processing"""
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # If running in an existing event loop, create a new one
            import concurrent.futures
            with concurrent.futures.ThreadPoolExecutor() as executor:
                future = executor.submit(
                    lambda: asyncio.run(self.process_intent_async(intent, intent_id))
                )
                result = future.result()
        else:
            result = loop.run_until_complete(self.process_intent_async(intent, intent_id))
        
        return asdict(result)
    
    def _create_error_result(self, intent_id: str, intent: str, error: str, 
                           start_time: float, raw_output: str = "") -> ProcessedIntent:
        """Create error result with metrics"""
        processing_time = (time.time() - start_time) * 1000
        
        metrics = IntentMetrics(
            processing_time_ms=processing_time,
            tokens_used=0,
            retrieval_score=0.0,
            confidence_score=0.0,
            cache_hit=False,
            model_version=self.config.get("model", "gpt-4o-mini")
        )
        
        return ProcessedIntent(
            intent_id=intent_id,
            original_intent=intent,
            structured_output={"error": error, "raw_output": raw_output},
            status=ProcessingStatus.FAILED,
            metrics=metrics,
            timestamp=time.time(),
            error_message=error
        )
    
    def _validate_enhanced_response(self, response: Dict[str, Any]) -> bool:
        """Enhanced validation for response structure"""
        if not isinstance(response, dict) or 'type' not in response:
            return False
        
        response_type = response['type']
        
        # Define required fields for each type
        required_fields = {
            "NetworkFunctionDeployment": ['name', 'namespace', 'spec'],
            "NetworkFunctionScale": ['name', 'namespace', 'replicas'],
            "NetworkSliceConfiguration": ['slice_id', 'name', 'slice_type'],
            "PolicyConfiguration": ['policy_name', 'policy_type_id', 'policy_data']
        }
        
        if response_type not in required_fields:
            return False
        
        return all(field in response for field in required_fields[response_type])
    
    def _calculate_retrieval_score(self, source_documents: List[Document]) -> float:
        """Calculate retrieval quality score"""
        if not source_documents:
            return 0.0
        
        # Simple scoring based on document count and content length
        total_content_length = sum(len(doc.page_content) for doc in source_documents)
        avg_content_length = total_content_length / len(source_documents)
        
        # Normalize score between 0 and 1
        return min(1.0, (len(source_documents) * avg_content_length) / 5000)
    
    def _calculate_confidence_score(self, output: Dict[str, Any], 
                                   source_documents: List[Document]) -> float:
        """Calculate confidence score for the response"""
        score = 0.0
        
        # Base score from having required fields
        if self._validate_enhanced_response(output):
            score += 0.4
        
        # Score from source document relevance
        if source_documents:
            score += min(0.3, len(source_documents) * 0.1)
        
        # Score from response completeness
        if 'spec' in output and isinstance(output['spec'], dict):
            score += 0.2 * (len(output['spec']) / 5)  # Normalize by expected field count
        
        # Score from having metadata fields
        metadata_fields = ['o1_config', 'a1_policy', 'network_slice']
        existing_metadata = sum(1 for field in metadata_fields if field in output)
        score += 0.1 * (existing_metadata / len(metadata_fields))
        
        return min(1.0, score)
    
    def _estimate_tokens(self, text: str) -> int:
        """Estimate token count for text"""
        return len(text.split()) * 1.3  # Rough estimation
    
    def add_knowledge_documents(self, documents: List[Document]) -> bool:
        """Add documents to the knowledge base"""
        return self.knowledge_manager.add_documents(documents)
    
    def get_health_status(self) -> Dict[str, Any]:
        """Get system health status"""
        try:
            # Check Weaviate connection
            weaviate_ready = self.weaviate_client.is_ready()
            
            # Check object count
            result = self.weaviate_client.query.aggregate("TelecomKnowledge").with_meta_count().do()
            object_count = result.get("data", {}).get("Aggregate", {}).get("TelecomKnowledge", [{}])[0].get("meta", {}).get("count", 0)
            
            return {
                "status": "healthy" if weaviate_ready else "unhealthy",
                "weaviate_ready": weaviate_ready,
                "knowledge_objects": object_count,
                "cache_size": len(self.cache.cache),
                "model": self.config.get("llm_model", "gpt-4o-mini"),
                "timestamp": time.time()
            }
        except Exception as e:
            return {
                "status": "unhealthy",
                "error": str(e),
                "timestamp": time.time()
            }