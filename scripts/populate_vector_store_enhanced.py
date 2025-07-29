#!/usr/bin/env python3
"""
Enhanced Vector Store Population Script for Nephoran Intent Operator
Processes and indexes telecom domain documentation using the enhanced RAG pipeline
"""

import os
import sys
import argparse
import logging
import asyncio
from pathlib import Path
from typing import Dict, Any, List, Optional
import json
import time

# Add the parent pkg directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "pkg" / "rag"))

from document_processor import DocumentProcessor, ProcessingMetrics
from enhanced_pipeline import EnhancedTelecomRAGPipeline
import weaviate as wv


def setup_logging(level: str = "INFO") -> logging.Logger:
    """Setup structured logging"""
    logging.basicConfig(
        level=getattr(logging, level.upper()),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler('vector_store_population.log')
        ]
    )
    return logging.getLogger(__name__)


def load_config() -> Dict[str, Any]:
    """Load configuration from environment variables"""
    return {
        "weaviate_url": os.environ.get("WEAVIATE_URL", "http://localhost:8080"),
        "openai_api_key": os.environ.get("OPENAI_API_KEY"),
        "model": os.environ.get("OPENAI_MODEL", "gpt-4o-mini"),
        "cache_max_size": int(os.environ.get("CACHE_MAX_SIZE", "1000")),
        "cache_ttl_seconds": int(os.environ.get("CACHE_TTL_SECONDS", "3600")),
        "chunk_size": int(os.environ.get("CHUNK_SIZE", "1000")),
        "chunk_overlap": int(os.environ.get("CHUNK_OVERLAP", "200")),
        "max_concurrent_files": int(os.environ.get("MAX_CONCURRENT_FILES", "5")),
        "batch_size": int(os.environ.get("BATCH_SIZE", "50"))
    }


def check_weaviate_connection(config: Dict[str, Any], logger: logging.Logger) -> bool:
    """Check Weaviate connection and health"""
    try:
        client = wv.Client(
            url=config["weaviate_url"],
            additional_headers={
                "X-OpenAI-Api-Key": config["openai_api_key"]
            } if config.get("openai_api_key") else {}
        )
        
        if client.is_ready():
            logger.info(f"Successfully connected to Weaviate at {config['weaviate_url']}")
            
            # Check schema
            schema = client.schema.get()
            class_names = [c["class"] for c in schema.get("classes", [])]
            logger.info(f"Existing schema classes: {class_names}")
            
            return True
        else:
            logger.error("Weaviate is not ready")
            return False
            
    except Exception as e:
        logger.error(f"Failed to connect to Weaviate: {e}")
        return False


def get_existing_object_count(config: Dict[str, Any], logger: logging.Logger) -> int:
    """Get count of existing objects in Weaviate"""
    try:
        client = wv.Client(
            url=config["weaviate_url"],
            additional_headers={
                "X-OpenAI-Api-Key": config["openai_api_key"]
            } if config.get("openai_api_key") else {}
        )
        
        result = client.query.aggregate("TelecomKnowledge").with_meta_count().do()
        count = result.get("data", {}).get("Aggregate", {}).get("TelecomKnowledge", [{}])[0].get("meta", {}).get("count", 0)
        
        logger.info(f"Existing objects in TelecomKnowledge class: {count}")
        return count
        
    except Exception as e:
        logger.warning(f"Could not get existing object count: {e}")
        return 0


def clear_existing_data(config: Dict[str, Any], logger: logging.Logger) -> bool:
    """Clear existing data from Weaviate"""
    try:
        client = wv.Client(
            url=config["weaviate_url"],
            additional_headers={
                "X-OpenAI-Api-Key": config["openai_api_key"]
            } if config.get("openai_api_key") else {}
        )
        
        # Delete all objects from TelecomKnowledge class
        logger.info("Clearing existing TelecomKnowledge objects...")
        client.batch.delete_objects(
            class_name="TelecomKnowledge",
            where={
                "operator": "Like",
                "path": ["source"],
                "valueText": "*"
            }
        )
        
        logger.info("Existing data cleared successfully")
        return True
        
    except Exception as e:
        logger.error(f"Failed to clear existing data: {e}")
        return False


async def populate_vector_store(
    knowledge_base_path: Path,
    config: Dict[str, Any],
    logger: logging.Logger,
    file_patterns: Optional[List[str]] = None,
    clear_existing: bool = False
) -> Dict[str, Any]:
    """Main function to populate vector store with enhanced processing"""
    
    start_time = time.time()
    
    # Validate inputs
    if not knowledge_base_path.exists():
        logger.error(f"Knowledge base path does not exist: {knowledge_base_path}")
        return {"success": False, "error": "Knowledge base path not found"}
    
    if not config.get("openai_api_key"):
        logger.error("OpenAI API key not provided")
        return {"success": False, "error": "OpenAI API key required"}
    
    # Check Weaviate connection
    if not check_weaviate_connection(config, logger):
        return {"success": False, "error": "Cannot connect to Weaviate"}
    
    # Get initial count
    initial_count = get_existing_object_count(config, logger)
    
    # Clear existing data if requested
    if clear_existing:
        if not clear_existing_data(config, logger):
            return {"success": False, "error": "Failed to clear existing data"}
    
    try:
        # Initialize components
        logger.info("Initializing enhanced RAG pipeline...")
        rag_pipeline = EnhancedTelecomRAGPipeline(config)
        document_processor = DocumentProcessor(config)
        
        # Process documents
        logger.info(f"Processing documents from: {knowledge_base_path}")
        documents, metrics = await document_processor.process_directory_async(
            knowledge_base_path, file_patterns
        )
        
        if not documents:
            logger.warning("No documents were processed")
            return {
                "success": True,
                "warning": "No documents found to process",
                "metrics": {
                    "total_documents": 0,
                    "processed_documents": 0,
                    "total_chunks": 0,
                    "processing_time_seconds": 0
                }
            }
        
        # Add documents to vector store
        logger.info(f"Adding {len(documents)} document chunks to vector store...")
        success = rag_pipeline.add_knowledge_documents(documents)
        
        if not success:
            return {"success": False, "error": "Failed to add documents to vector store"}
        
        # Get final count
        final_count = get_existing_object_count(config, logger)
        new_objects = final_count - (initial_count if not clear_existing else 0)
        
        # Calculate total time
        total_time = time.time() - start_time
        
        # Prepare results
        results = {
            "success": True,
            "metrics": {
                "total_documents": metrics.total_documents,
                "processed_documents": metrics.processed_documents,
                "failed_documents": metrics.failed_documents,
                "total_chunks": metrics.total_chunks,
                "processing_time_seconds": metrics.processing_time_seconds,
                "average_chunk_size": metrics.average_chunk_size,
                "keywords_extracted": metrics.keywords_extracted,
                "total_time_seconds": total_time,
                "initial_object_count": initial_count,
                "final_object_count": final_count,
                "new_objects_added": new_objects
            },
            "knowledge_base_path": str(knowledge_base_path),
            "configuration": {
                "chunk_size": config["chunk_size"],
                "chunk_overlap": config["chunk_overlap"],
                "model": config["model"],
                "weaviate_url": config["weaviate_url"]
            }
        }
        
        logger.info(f"Vector store population completed successfully!")
        logger.info(f"Added {new_objects} new objects in {total_time:.2f} seconds")
        
        return results
        
    except Exception as e:
        logger.error(f"Error during vector store population: {e}")
        return {"success": False, "error": str(e)}


def save_results(results: Dict[str, Any], output_path: Optional[Path] = None) -> None:
    """Save results to JSON file"""
    if output_path is None:
        timestamp = int(time.time())
        output_path = Path(f"vector_store_population_results_{timestamp}.json")
    
    with open(output_path, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"Results saved to: {output_path}")


def print_summary(results: Dict[str, Any]) -> None:
    """Print a summary of the population results"""
    if not results.get("success"):
        print(f"❌ Population failed: {results.get('error', 'Unknown error')}")
        return
    
    metrics = results.get("metrics", {})
    
    print("\n" + "="*60)
    print("📊 VECTOR STORE POPULATION SUMMARY")
    print("="*60)
    
    print(f"✅ Status: {'SUCCESS' if results['success'] else 'FAILED'}")
    print(f"📁 Knowledge Base: {results.get('knowledge_base_path', 'Unknown')}")
    print(f"⏱️  Total Time: {metrics.get('total_time_seconds', 0):.2f} seconds")
    
    print("\n📄 Document Processing:")
    print(f"   • Total Documents: {metrics.get('total_documents', 0)}")
    print(f"   • Processed Successfully: {metrics.get('processed_documents', 0)}")
    print(f"   • Failed to Process: {metrics.get('failed_documents', 0)}")
    print(f"   • Processing Time: {metrics.get('processing_time_seconds', 0):.2f}s")
    
    print("\n🔍 Text Chunking:")
    print(f"   • Total Chunks Created: {metrics.get('total_chunks', 0)}")
    print(f"   • Average Chunk Size: {metrics.get('average_chunk_size', 0)} characters")
    print(f"   • Keywords Extracted: {metrics.get('keywords_extracted', 0)}")
    
    print("\n🗃️  Vector Database:")
    print(f"   • Initial Object Count: {metrics.get('initial_object_count', 0)}")
    print(f"   • Final Object Count: {metrics.get('final_object_count', 0)}")
    print(f"   • New Objects Added: {metrics.get('new_objects_added', 0)}")
    
    config = results.get("configuration", {})
    print(f"\n⚙️  Configuration:")
    print(f"   • Chunk Size: {config.get('chunk_size', 'Unknown')}")
    print(f"   • Chunk Overlap: {config.get('chunk_overlap', 'Unknown')}")
    print(f"   • Model: {config.get('model', 'Unknown')}")
    print(f"   • Weaviate URL: {config.get('weaviate_url', 'Unknown')}")
    
    print("="*60)


async def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(
        description="Populate Weaviate vector store with telecom domain knowledge"
    )
    parser.add_argument(
        "knowledge_base_path",
        type=Path,
        help="Path to knowledge base directory"
    )
    parser.add_argument(
        "--clear-existing",
        action="store_true",
        help="Clear existing data before populating"
    )
    parser.add_argument(
        "--file-patterns",
        nargs="+",
        default=["*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.pdf"],
        help="File patterns to process (default: *.md *.txt *.json *.yaml *.yml *.pdf)"
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="Output path for results JSON file"
    )
    parser.add_argument(
        "--log-level",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        default="INFO",
        help="Logging level"
    )
    
    args = parser.parse_args()
    
    # Setup logging
    logger = setup_logging(args.log_level)
    
    # Load configuration
    config = load_config()
    
    # Validate configuration
    if not config.get("openai_api_key"):
        logger.error("OPENAI_API_KEY environment variable is required")
        print("❌ Error: Set OPENAI_API_KEY environment variable")
        sys.exit(1)
    
    # Run population
    logger.info("Starting vector store population...")
    results = await populate_vector_store(
        args.knowledge_base_path,
        config,
        logger,
        args.file_patterns,
        args.clear_existing
    )
    
    # Save and display results
    if args.output:
        save_results(results, args.output)
    
    print_summary(results)
    
    # Exit with appropriate code
    sys.exit(0 if results.get("success") else 1)


if __name__ == "__main__":
    asyncio.run(main())