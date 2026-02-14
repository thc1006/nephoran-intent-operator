from typing import Dict, Any
from langchain_community.vectorstores import Weaviate
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from langchain.chains import RetrievalQA
from langchain.prompts import PromptTemplate
import json
import time
import logging

# Import document processor for knowledge base loading
from document_processor import DocumentProcessor


class TelecomRAGPipeline:
    """Production-ready RAG pipeline for telecom domain."""

    def __init__(self, config: Dict[str, Any]):
        if not config.get("openai_api_key"):
            raise ValueError("OpenAI API key is required")

        self.config = config
        self.logger = self._setup_logging()

        try:
            self.embeddings = OpenAIEmbeddings(
                model="text-embedding-3-large",
                dimensions=3072,
                openai_api_key=config["openai_api_key"]
            )
            self.llm = ChatOpenAI(
                model=config.get("llm_model", "gpt-4o-mini"),
                temperature=0,
                max_tokens=2048,
                model_kwargs={"response_format": {"type": "json_object"}},
                openai_api_key=config["openai_api_key"]
            )
            self.vector_store = self._setup_vector_store(config)
            self.qa_chain = self._create_qa_chain()

            # Load telecom domain knowledge from knowledge_base/
            self._load_domain_knowledge()
        except Exception as e:
            raise RuntimeError(f"Failed to initialize RAG pipeline: {e}")

    def _setup_logging(self) -> logging.Logger:
        """Setup logging for RAG pipeline"""
        logger = logging.getLogger(f"{__name__}.TelecomRAGPipeline")
        logger.setLevel(logging.INFO)
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
        return logger

    def _load_domain_knowledge(self):
        """Load and index telecom domain knowledge from knowledge_base/"""
        try:
            self.logger.info("Loading telecom domain knowledge...")

            # Initialize document processor
            doc_processor = DocumentProcessor(self.config)

            # Load knowledge base documents
            docs = doc_processor.load_telecom_knowledge()

            if docs:
                # Add documents to vector store
                self.vector_store.add_documents(docs)
                self.logger.info(
                    f"Successfully indexed {len(docs)} knowledge base documents"
                )
            else:
                self.logger.warning(
                    "No documents loaded from knowledge_base/. "
                    "RAG will work but without domain-specific knowledge."
                )
        except Exception as e:
            self.logger.error(f"Failed to load domain knowledge: {e}")
            # Don't fail initialization if knowledge loading fails
            self.logger.warning("RAG pipeline initialized without domain knowledge")

    def _setup_vector_store(self, config: Dict) -> Weaviate:
        """Initialize Weaviate vector database."""
        try:
            return Weaviate.from_existing_index(
                embedding=self.embeddings,
                index_name="telecom_knowledge",
                text_key="content",
                weaviate_url=config["weaviate_url"],
                additional_headers={"X-OpenAI-Api-Key": config["openai_api_key"]}
            )
        except Exception as e:
            raise ConnectionError(f"Failed to connect to Weaviate at {config.get('weaviate_url', 'unknown')}: {e}")

    def _create_qa_chain(self) -> RetrievalQA:
        """Create telecom-optimized QA chain."""
        template = """
You are an expert telecom network engineer responsible for translating
natural language commands into structured JSON objects.
Your task is to analyze the user's command and determine if it is a
"NetworkFunctionDeployment" or a "NetworkFunctionScale" intent.

The user command is: "{question}"

Use the following context to inform your decision on default values if they
are not specified in the command.
Context: {context}

The output MUST be a single, valid JSON object. Do not add any explanation
or introductory text.

If the intent is to **deploy a new function**, use this JSON schema:
{{
  "type": "NetworkFunctionDeployment",
  "name": "string",
  "namespace": "string",
  "spec": {{ "replicas": "integer", "image": "string", "resources": {{...}} }},
  "o1_config": "string (XML)",
  "a1_policy": {{ "policy_type_id": "string", "policy_data": "object" }}
}}

If the intent is to **scale an existing function**, use this JSON schema:
{{
  "type": "NetworkFunctionScale",
  "name": "string (The name of the function to scale)",
  "namespace": "string",
  "replicas": "integer (The target number of replicas)"
}}

Based on the user command, generate the single, corresponding JSON object.
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
                search_kwargs={"k": 3}
            ),
            chain_type_kwargs={"prompt": prompt},
            return_source_documents=False
        )

    def process_intent(self, intent: str) -> Dict[str, Any]:
        """Process user intent and return a structured JSON object."""
        if not intent or not intent.strip():
            return {
                "error": "Empty intent provided",
                "original_intent": intent
            }
        
        try:
            result = self.qa_chain.invoke({"query": intent})
            
            # Extract the result text
            result_text = result.get('result', '')
            if not result_text:
                return {
                    "error": "Empty response from LLM",
                    "original_intent": intent
                }
            
            # Parse the JSON response
            structured_output = json.loads(result_text)
            
            # Add metadata
            structured_output['original_intent'] = intent
            structured_output['timestamp'] = json.dumps(time.time())
            
            # Validate the response structure
            if not self._validate_response_structure(structured_output):
                return {
                    "error": "Invalid response structure from LLM",
                    "raw_output": result_text,
                    "original_intent": intent
                }
            
            return structured_output
            
        except (json.JSONDecodeError, TypeError) as e:
            return {
                "error": "Failed to parse LLM output as JSON",
                "raw_output": result.get('result', ''),
                "original_intent": intent,
                "exception": str(e)
            }
        except Exception as e:
            return {
                "error": "Unexpected error during intent processing",
                "original_intent": intent,
                "exception": str(e)
            }
    
    def _validate_response_structure(self, response: Dict[str, Any]) -> bool:
        """Validate that the response has the expected structure."""
        if not isinstance(response, dict):
            return False
        
        # Check for required 'type' field
        if 'type' not in response:
            return False
        
        response_type = response['type']
        
        # Validate NetworkFunctionDeployment structure
        if response_type == "NetworkFunctionDeployment":
            required_fields = ['name', 'namespace', 'spec']
            return all(field in response for field in required_fields)
        
        # Validate NetworkFunctionScale structure
        elif response_type == "NetworkFunctionScale":
            required_fields = ['name', 'namespace', 'replicas']
            return all(field in response for field in required_fields)
        
        return False
