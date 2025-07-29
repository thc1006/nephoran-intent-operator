"""
Document Processing Pipeline for Telecom Domain Knowledge
Handles ingestion, processing, and embedding of telecom-specific documents
"""

import asyncio
import logging
import re
import hashlib
from pathlib import Path
from typing import Dict, Any, List, Optional, Tuple, Set
from dataclasses import dataclass
from enum import Enum
import json
import yaml

from langchain.schema import Document
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain.document_loaders import (
    TextLoader, 
    UnstructuredMarkdownLoader,
    PyPDFLoader,
    JSONLoader
)
import openai


class DocumentType(Enum):
    """Supported document types for processing"""
    THREEGPP_SPEC = "3gpp_specification"
    ORAN_SPEC = "oran_specification"
    YAML_CONFIG = "yaml_configuration"
    JSON_CONFIG = "json_configuration"
    MARKDOWN_DOC = "markdown_documentation"
    TEXT_DOC = "text_documentation"
    PDF_SPEC = "pdf_specification"


@dataclass
class ProcessingMetrics:
    """Metrics for document processing"""
    total_documents: int
    processed_documents: int
    failed_documents: int
    total_chunks: int
    processing_time_seconds: float
    average_chunk_size: int
    keywords_extracted: int


class TelecomKeywordExtractor:
    """Extracts telecom-specific keywords and concepts"""
    
    # Telecom domain keywords and patterns
    TELECOM_KEYWORDS = {
        "5g_core": [
            "AMF", "SMF", "UPF", "NSSF", "AUSF", "UDM", "UDR", "PCF", "BSF",
            "NWDAF", "CHF", "SCP", "SEPP", "5GC", "Service Based Architecture"
        ],
        "ran": [
            "gNB", "ng-eNB", "CU", "DU", "RU", "RIC", "Near-RT RIC", "Non-RT RIC",
            "xApp", "rApp", "O-RAN", "E2", "F1", "Xn", "NG", "RRC", "PDCP", "RLC", "MAC", "PHY"
        ],
        "network_slice": [
            "S-NSSAI", "NSI", "NSSI", "eMBB", "URLLC", "mMTC", "Network Slice",
            "Slice Selection", "NSSF", "Network Slice Template"
        ],
        "interfaces": [
            "A1", "O1", "O2", "E2", "Open Fronthaul", "M-Plane", "C-Plane", "U-Plane",
            "NETCONF", "YANG", "RESTful", "gRPC"
        ],
        "mgmt": [
            "FCAPS", "OAM", "Policy", "Configuration Management", "Fault Management",
            "Performance Management", "Security Management", "Accounting Management"
        ],
        "protocols": [
            "HTTP/2", "gRPC", "SCTP", "TCP", "UDP", "TLS", "IPSec", "OAuth2", "JWT"
        ]
    }
    
    # Regex patterns for technical specifications
    SPEC_PATTERNS = {
        "3gpp_ref": re.compile(r'TS\s+\d{2}\.\d{3}', re.IGNORECASE),
        "oran_ref": re.compile(r'O-RAN\.[A-Z]{2,3}-\d+\.\d+', re.IGNORECASE),
        "rfc_ref": re.compile(r'RFC\s+\d{4,5}', re.IGNORECASE),
        "version": re.compile(r'[Vv](\d+)\.(\d+)\.(\d+)', re.IGNORECASE),
        "frequency": re.compile(r'(\d+(?:\.\d+)?)\s*(MHz|GHz|kHz)', re.IGNORECASE),
        "latency": re.compile(r'(\d+(?:\.\d+)?)\s*(ms|μs|ns)', re.IGNORECASE),
        "throughput": re.compile(r'(\d+(?:\.\d+)?)\s*(Mbps|Gbps|kbps)', re.IGNORECASE)
    }
    
    def extract_keywords(self, text: str) -> Tuple[List[str], Dict[str, List[str]]]:
        """Extract keywords and technical references from text"""
        keywords = []
        references = {
            "specifications": [],
            "technical_params": [],
            "domain_concepts": []
        }
        
        # Extract domain-specific keywords
        text_lower = text.lower()
        for category, terms in self.TELECOM_KEYWORDS.items():
            for term in terms:
                if term.lower() in text_lower:
                    keywords.append(term)
                    references["domain_concepts"].append(f"{category}:{term}")
        
        # Extract technical references
        for ref_type, pattern in self.SPEC_PATTERNS.items():
            matches = pattern.findall(text)
            for match in matches:
                if isinstance(match, tuple):
                    match = ' '.join(match)
                references["specifications"].append(f"{ref_type}:{match}")
                keywords.append(match)
        
        # Remove duplicates while preserving order
        keywords = list(dict.fromkeys(keywords))
        
        return keywords, references


class DocumentProcessor:
    """Processes telecom documents for knowledge base ingestion"""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.logger = self._setup_logging()
        self.keyword_extractor = TelecomKeywordExtractor()
        
        # Initialize text splitter
        self.text_splitter = RecursiveCharacterTextSplitter(
            chunk_size=config.get("chunk_size", 1000),
            chunk_overlap=config.get("chunk_overlap", 200),
            length_function=len,
            separators=["\n\n", "\n", ". ", " ", ""]
        )
    
    def _setup_logging(self) -> logging.Logger:
        """Setup logging for document processor"""
        logger = logging.getLogger(f"{__name__}.DocumentProcessor")
        logger.setLevel(logging.INFO)
        
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
        
        return logger
    
    def detect_document_type(self, file_path: Path) -> DocumentType:
        """Detect document type based on content and filename"""
        filename = file_path.name.lower()
        
        # Check file extension first
        if filename.endswith('.pdf'):
            return DocumentType.PDF_SPEC
        elif filename.endswith(('.yml', '.yaml')):
            return DocumentType.YAML_CONFIG
        elif filename.endswith('.json'):
            return DocumentType.JSON_CONFIG
        elif filename.endswith('.md'):
            return DocumentType.MARKDOWN_DOC
        elif filename.endswith('.txt'):
            return DocumentType.TEXT_DOC
        
        # Check content-based patterns
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read(1000)  # Read first 1000 chars
                
                if 'TS ' in content and re.search(r'\d{2}\.\d{3}', content):
                    return DocumentType.THREEGPP_SPEC
                elif 'O-RAN' in content:
                    return DocumentType.ORAN_SPEC
                    
        except (UnicodeDecodeError, IOError):
            pass
        
        return DocumentType.TEXT_DOC
    
    def load_document(self, file_path: Path, doc_type: DocumentType) -> Optional[Document]:
        """Load document based on its type"""
        try:
            if doc_type == DocumentType.PDF_SPEC:
                loader = PyPDFLoader(str(file_path))
                docs = loader.load()
                # Combine all pages into single document
                content = "\n\n".join([doc.page_content for doc in docs])
                
            elif doc_type == DocumentType.MARKDOWN_DOC:
                loader = UnstructuredMarkdownLoader(str(file_path))
                docs = loader.load()
                content = docs[0].page_content if docs else ""
                
            elif doc_type in [DocumentType.JSON_CONFIG, DocumentType.YAML_CONFIG]:
                with open(file_path, 'r', encoding='utf-8') as f:
                    if doc_type == DocumentType.JSON_CONFIG:
                        data = json.load(f)
                        content = json.dumps(data, indent=2)
                    else:
                        data = yaml.safe_load(f)
                        content = yaml.dump(data, default_flow_style=False, indent=2)
                        
            else:
                # Text-based documents
                loader = TextLoader(str(file_path))
                docs = loader.load()
                content = docs[0].page_content if docs else ""
            
            # Create document with metadata
            metadata = {
                "source": str(file_path),
                "filename": file_path.name,
                "doc_type": doc_type.value,
                "file_size": file_path.stat().st_size,
                "last_modified": file_path.stat().st_mtime
            }
            
            return Document(page_content=content, metadata=metadata)
            
        except Exception as e:
            self.logger.error(f"Failed to load document {file_path}: {e}")
            return None
    
    def process_document(self, document: Document) -> List[Document]:
        """Process document into chunks with enhanced metadata"""
        try:
            # Extract keywords and references
            keywords, references = self.keyword_extractor.extract_keywords(
                document.page_content
            )
            
            # Determine document category
            category = self._categorize_document(document, keywords)
            
            # Split document into chunks
            chunks = self.text_splitter.split_text(document.page_content)
            
            processed_docs = []
            for i, chunk in enumerate(chunks):
                # Extract chunk-specific keywords
                chunk_keywords, chunk_refs = self.keyword_extractor.extract_keywords(chunk)
                
                # Calculate confidence score
                confidence = self._calculate_confidence_score(chunk, chunk_keywords)
                
                # Create enhanced metadata
                chunk_metadata = {
                    **document.metadata,
                    "chunk_id": f"{document.metadata.get('filename', 'unknown')}_{i}",
                    "chunk_index": i,
                    "total_chunks": len(chunks),
                    "keywords": list(set(keywords + chunk_keywords)),
                    "category": category,
                    "confidence": confidence,
                    "references": {**references, **chunk_refs},
                    "chunk_size": len(chunk),
                    "uuid": self._generate_uuid(chunk, document.metadata.get("source", ""))
                }
                
                processed_docs.append(Document(
                    page_content=chunk,
                    metadata=chunk_metadata
                ))
            
            return processed_docs
            
        except Exception as e:
            self.logger.error(f"Failed to process document: {e}")
            return []
    
    def _categorize_document(self, document: Document, keywords: List[str]) -> str:
        """Categorize document based on content and keywords"""
        categories = {
            "5g_core": 0,
            "ran": 0,
            "network_slice": 0,
            "interfaces": 0,
            "mgmt": 0,
            "protocols": 0
        }
        
        # Count keyword matches per category
        for keyword in keywords:
            for category, terms in self.keyword_extractor.TELECOM_KEYWORDS.items():
                if keyword in terms:
                    categories[category] = categories.get(category, 0) + 1
        
        # Return category with highest score
        if max(categories.values()) > 0:
            return max(categories, key=categories.get)
        
        # Fallback categorization
        doc_type = document.metadata.get("doc_type", "")
        if "3gpp" in doc_type:
            return "3gpp_specification"
        elif "oran" in doc_type:
            return "oran_specification"
        elif "config" in doc_type:
            return "configuration"
        
        return "general"
    
    def _calculate_confidence_score(self, text: str, keywords: List[str]) -> float:
        """Calculate confidence score for document chunk"""
        score = 0.0
        
        # Base score from text length
        if len(text) > 100:
            score += 0.3
        
        # Score from keyword density
        if keywords:
            keyword_density = len(keywords) / len(text.split())
            score += min(0.4, keyword_density * 10)
        
        # Score from technical patterns
        tech_patterns = sum(1 for pattern in self.keyword_extractor.SPEC_PATTERNS.values() 
                           if pattern.search(text))
        score += min(0.3, tech_patterns * 0.1)
        
        return min(1.0, score)
    
    def _generate_uuid(self, content: str, source: str) -> str:
        """Generate deterministic UUID for document chunk"""
        hash_input = f"{source}:{content[:100]}"
        return hashlib.sha256(hash_input.encode()).hexdigest()[:32]
    
    async def process_directory_async(self, directory_path: Path, 
                                    file_patterns: List[str] = None) -> ProcessingMetrics:
        """Asynchronously process all documents in directory"""
        start_time = asyncio.get_event_loop().time()
        
        if file_patterns is None:
            file_patterns = ["*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.pdf"]
        
        # Find all matching files
        all_files = []
        for pattern in file_patterns:
            all_files.extend(directory_path.glob(pattern))
        
        processed_docs = []
        failed_count = 0
        total_keywords = 0
        
        # Process files concurrently
        semaphore = asyncio.Semaphore(self.config.get("max_concurrent_files", 5))
        
        async def process_file(file_path: Path) -> Tuple[List[Document], int]:
            async with semaphore:
                return await asyncio.get_event_loop().run_in_executor(
                    None, self._process_single_file, file_path
                )
        
        tasks = [process_file(file_path) for file_path in all_files]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Collect results
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                self.logger.error(f"Failed to process {all_files[i]}: {result}")
                failed_count += 1
            else:
                docs, keywords_count = result
                processed_docs.extend(docs)
                total_keywords += keywords_count
        
        # Calculate metrics
        end_time = asyncio.get_event_loop().time()
        total_chunks = len(processed_docs)
        avg_chunk_size = sum(len(doc.page_content) for doc in processed_docs) // max(1, total_chunks)
        
        metrics = ProcessingMetrics(
            total_documents=len(all_files),
            processed_documents=len(all_files) - failed_count,
            failed_documents=failed_count,
            total_chunks=total_chunks,
            processing_time_seconds=end_time - start_time,
            average_chunk_size=avg_chunk_size,
            keywords_extracted=total_keywords
        )
        
        self.logger.info(f"Processed {metrics.processed_documents}/{metrics.total_documents} "
                        f"documents in {metrics.processing_time_seconds:.2f}s")
        
        return processed_docs, metrics
    
    def _process_single_file(self, file_path: Path) -> Tuple[List[Document], int]:
        """Process a single file and return documents and keyword count"""
        try:
            doc_type = self.detect_document_type(file_path)
            document = self.load_document(file_path, doc_type)
            
            if document:
                processed_docs = self.process_document(document)
                keyword_count = sum(len(doc.metadata.get("keywords", [])) for doc in processed_docs)
                return processed_docs, keyword_count
            
            return [], 0
            
        except Exception as e:
            self.logger.error(f"Error processing {file_path}: {e}")
            return [], 0
    
    def process_directory(self, directory_path: Path, 
                         file_patterns: List[str] = None) -> Tuple[List[Document], ProcessingMetrics]:
        """Synchronous wrapper for directory processing"""
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # If running in an existing event loop, create a new one
            import concurrent.futures
            with concurrent.futures.ThreadPoolExecutor() as executor:
                future = executor.submit(
                    lambda: asyncio.run(self.process_directory_async(directory_path, file_patterns))
                )
                return future.result()
        else:
            return loop.run_until_complete(self.process_directory_async(directory_path, file_patterns))
    
    def get_processing_stats(self) -> Dict[str, Any]:
        """Get processing statistics"""
        return {
            "supported_formats": [dt.value for dt in DocumentType],
            "chunk_size": self.config.get("chunk_size", 1000),
            "chunk_overlap": self.config.get("chunk_overlap", 200),
            "max_concurrent_files": self.config.get("max_concurrent_files", 5),
            "keyword_categories": list(self.keyword_extractor.TELECOM_KEYWORDS.keys()),
            "pattern_matchers": list(self.keyword_extractor.SPEC_PATTERNS.keys())
        }