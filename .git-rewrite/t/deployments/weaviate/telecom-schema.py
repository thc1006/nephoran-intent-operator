#!/usr/bin/env python3
"""
Telecom-specific Weaviate Schema Implementation for Nephoran Intent Operator
This script initializes comprehensive schemas optimized for 3GPP and O-RAN documents
"""

import weaviate
import json
import os
import sys
import logging
from typing import Dict, Any, List
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class TelecomSchemaManager:
    """Manages telecom-specific Weaviate schemas for 3GPP and O-RAN documentation"""
    
    def __init__(self, weaviate_url: str, api_key: str):
        """Initialize the schema manager with Weaviate connection"""
        self.client = weaviate.Client(
            url=weaviate_url,
            additional_headers={"X-OpenAI-Api-Key": os.getenv("OPENAI_API_KEY", "")},
            auth_client_secret=weaviate.AuthApiKey(api_key=api_key)
        )
        logger.info(f"Connected to Weaviate at {weaviate_url}")
    
    def get_telecom_knowledge_schema(self) -> Dict[str, Any]:
        """Define comprehensive schema for telecom knowledge base"""
        return {
            "class": "TelecomKnowledge",
            "description": "Comprehensive telecommunications domain knowledge base for 3GPP and O-RAN specifications",
            "vectorizer": "text2vec-openai",
            "moduleConfig": {
                "text2vec-openai": {
                    "model": "text-embedding-3-large",
                    "dimensions": 3072,
                    "type": "text",
                    "baseURL": "https://api.openai.com/v1"
                },
                "generative-openai": {
                    "model": "gpt-4o-mini"
                }
            },
            "vectorIndexConfig": {
                "distance": "cosine",
                "ef": 128,
                "efConstruction": 256,
                "maxConnections": 32,
                "dynamicEfMin": 64,
                "dynamicEfMax": 256,
                "dynamicEfFactor": 8,
                "vectorCacheMaxObjects": 1000000,
                "flatSearchCutoff": 40000,
                "skip": False,
                "cleanupIntervalSeconds": 300
            },
            "invertedIndexConfig": {
                "bm25": {
                    "k1": 1.2,
                    "b": 0.75
                },
                "cleanupIntervalSeconds": 60,
                "stopwords": {
                    "preset": "en",
                    "additions": ["3gpp", "oran", "ran", "core", "network"],
                    "removals": ["a", "an", "and", "are", "as", "at", "be", "by", "for", "from"]
                }
            },
            "properties": [
                {
                    "name": "content",
                    "dataType": ["text"],
                    "description": "Main document content with telecommunications specifications",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    },
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "title",
                    "dataType": ["text"],
                    "description": "Document or section title",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    },
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "source",
                    "dataType": ["text"],
                    "description": "Document source organization (3GPP, O-RAN, ETSI, ITU, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "specification",
                    "dataType": ["text"],
                    "description": "Specific specification identifier (e.g., TS 23.501, O-RAN.WG1.Use-Cases)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "version",
                    "dataType": ["text"],
                    "description": "Specification version number",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "release",
                    "dataType": ["text"],
                    "description": "3GPP Release number or O-RAN version",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "category",
                    "dataType": ["text"],
                    "description": "Document category (Architecture, Procedures, Interfaces, Management, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "domain",
                    "dataType": ["text"],
                    "description": "Technical domain (RAN, Core, Transport, Management, Security, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "keywords",
                    "dataType": ["text[]"],
                    "description": "Extracted technical keywords and acronyms",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "networkFunctions",
                    "dataType": ["text[]"],
                    "description": "Referenced network functions (AMF, SMF, UPF, gNB, CU, DU, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "interfaces",
                    "dataType": ["text[]"],
                    "description": "Referenced interfaces and reference points (N1, N2, N3, E1, F1, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "procedures",
                    "dataType": ["text[]"],
                    "description": "Referenced procedures and protocols",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "useCase",
                    "dataType": ["text"],
                    "description": "Primary use case (eMBB, URLLC, mMTC, Private Networks, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "priority",
                    "dataType": ["int"],
                    "description": "Content priority for retrieval (1-10, higher is more important)",
                    "indexFilterable": True
                },
                {
                    "name": "confidence",
                    "dataType": ["number"],
                    "description": "Content accuracy confidence score (0.0-1.0)",
                    "indexFilterable": True
                },
                {
                    "name": "lastUpdated",
                    "dataType": ["date"],
                    "description": "Last content update timestamp",
                    "indexFilterable": True
                },
                {
                    "name": "documentUrl",
                    "dataType": ["text"],
                    "description": "Original document URL or file path",
                    "indexFilterable": True
                },
                {
                    "name": "chunkIndex",
                    "dataType": ["int"],
                    "description": "Chunk sequence number within the document",
                    "indexFilterable": True
                },
                {
                    "name": "totalChunks",
                    "dataType": ["int"],
                    "description": "Total number of chunks in the document",
                    "indexFilterable": True
                },
                {
                    "name": "language",
                    "dataType": ["text"],
                    "description": "Document language (ISO 639-1 code)",
                    "indexFilterable": True
                },
                {
                    "name": "technicalLevel",
                    "dataType": ["text"],
                    "description": "Technical complexity level (Basic, Intermediate, Advanced, Expert)",
                    "indexFilterable": True
                }
            ]
        }
    
    def get_intent_patterns_schema(self) -> Dict[str, Any]:
        """Define schema for intent patterns and templates"""
        return {
            "class": "IntentPatterns",
            "description": "Natural language intent patterns for telecom operations",
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
                    "name": "pattern",
                    "dataType": ["text"],
                    "description": "Natural language pattern for intent recognition",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    }
                },
                {
                    "name": "intentType",
                    "dataType": ["text"],
                    "description": "Classification of intent (NetworkFunctionDeployment, NetworkSliceConfiguration, etc.)",
                    "indexFilterable": True
                },
                {
                    "name": "parameters",
                    "dataType": ["text[]"],
                    "description": "Expected parameters for this intent pattern",
                    "indexFilterable": True
                },
                {
                    "name": "examples",
                    "dataType": ["text[]"],
                    "description": "Example phrases matching this pattern"
                },
                {
                    "name": "confidence",
                    "dataType": ["number"],
                    "description": "Pattern matching confidence threshold",
                    "indexFilterable": True
                }
            ]
        }
    
    def get_network_functions_schema(self) -> Dict[str, Any]:
        """Define schema for network function definitions and configurations"""
        return {
            "class": "NetworkFunctions",
            "description": "5G and O-RAN network function definitions and configurations",
            "vectorizer": "text2vec-openai",
            "moduleConfig": {
                "text2vec-openai": {
                    "model": "text-embedding-3-large",
                    "dimensions": 3072
                }
            },
            "properties": [
                {
                    "name": "name",
                    "dataType": ["text"],
                    "description": "Network function name (AMF, SMF, UPF, etc.)",
                    "indexFilterable": True,
                    "indexSearchable": True
                },
                {
                    "name": "description",
                    "dataType": ["text"],
                    "description": "Detailed function description and purpose",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    }
                },
                {
                    "name": "category",
                    "dataType": ["text"],
                    "description": "Function category (Core, RAN, Management, etc.)",
                    "indexFilterable": True
                },
                {
                    "name": "interfaces",
                    "dataType": ["text[]"],
                    "description": "Supported interfaces and reference points",
                    "indexFilterable": True
                },
                {
                    "name": "deploymentOptions",
                    "dataType": ["text[]"],
                    "description": "Supported deployment configurations",
                    "indexFilterable": True
                },
                {
                    "name": "resourceRequirements",
                    "dataType": ["text"],
                    "description": "Typical resource requirements and constraints"
                },
                {
                    "name": "scalingOptions",
                    "dataType": ["text"],
                    "description": "Available scaling and dimensioning options"
                },
                {
                    "name": "standardsCompliance",
                    "dataType": ["text[]"],
                    "description": "Applicable standards and specifications",
                    "indexFilterable": True
                }
            ]
        }
    
    def initialize_schemas(self) -> bool:
        """Initialize all telecom-specific schemas in Weaviate"""
        try:
            # Check if schemas already exist
            existing_schema = self.client.schema.get()
            existing_classes = [cls["class"] for cls in existing_schema.get("classes", [])]
            
            schemas = [
                self.get_telecom_knowledge_schema(),
                self.get_intent_patterns_schema(),
                self.get_network_functions_schema()
            ]
            
            for schema in schemas:
                class_name = schema["class"]
                if class_name in existing_classes:
                    logger.info(f"Schema {class_name} already exists, skipping...")
                    continue
                
                logger.info(f"Creating schema: {class_name}")
                self.client.schema.create_class(schema)
                logger.info(f"Successfully created schema: {class_name}")
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize schemas: {e}")
            return False
    
    def populate_sample_data(self) -> bool:
        """Populate schemas with sample telecom data"""
        try:
            # Sample telecom knowledge data
            sample_knowledge = [
                {
                    "content": "The Access and Mobility Management Function (AMF) provides registration management, connection management, reachability management, mobility management, and access authentication and authorization.",
                    "title": "AMF Functionality Overview",
                    "source": "3GPP",
                    "specification": "TS 23.501",
                    "version": "17.9.0",
                    "release": "Rel-17",
                    "category": "Architecture",
                    "domain": "Core",
                    "keywords": ["AMF", "mobility", "authentication", "registration"],
                    "networkFunctions": ["AMF"],
                    "interfaces": ["N1", "N2", "N8", "N11", "N12", "N14", "N15"],
                    "useCase": "eMBB",
                    "priority": 9,
                    "confidence": 0.95,
                    "lastUpdated": datetime.utcnow().isoformat() + "Z",
                    "technicalLevel": "Intermediate"
                },
                {
                    "content": "The Session Management Function (SMF) is responsible for session management, UE IP address allocation and management, selection and control of UP function, policy enforcement and QoS, and downlink data notification.",
                    "title": "SMF Core Functions",
                    "source": "3GPP",
                    "specification": "TS 23.501",
                    "version": "17.9.0",
                    "release": "Rel-17",
                    "category": "Architecture",
                    "domain": "Core",
                    "keywords": ["SMF", "session", "QoS", "policy"],
                    "networkFunctions": ["SMF", "UPF"],
                    "interfaces": ["N4", "N7", "N10", "N11"],
                    "useCase": "eMBB",
                    "priority": 9,
                    "confidence": 0.95,
                    "lastUpdated": datetime.utcnow().isoformat() + "Z",
                    "technicalLevel": "Advanced"
                },
                {
                    "content": "O-RAN Alliance defines an open, intelligent, virtualized and fully interoperable RAN architecture. The O-RAN architecture introduces new interfaces including A1, E2, and O1 for enhanced network management and intelligence.",
                    "title": "O-RAN Architecture Overview",
                    "source": "O-RAN",
                    "specification": "O-RAN.WG1.O-RAN-Architecture",
                    "version": "07.00",
                    "category": "Architecture",
                    "domain": "RAN",
                    "keywords": ["O-RAN", "virtualization", "intelligence", "interoperability"],
                    "networkFunctions": ["Near-RT RIC", "Non-RT RIC", "CU", "DU", "RU"],
                    "interfaces": ["A1", "E2", "O1"],
                    "useCase": "Network Intelligence",
                    "priority": 8,
                    "confidence": 0.90,
                    "lastUpdated": datetime.utcnow().isoformat() + "Z",
                    "technicalLevel": "Expert"
                }
            ]
            
            # Insert sample knowledge data
            with self.client.batch as batch:
                batch.batch_size = 100
                for item in sample_knowledge:
                    batch.add_data_object(
                        data_object=item,
                        class_name="TelecomKnowledge"
                    )
            
            # Sample intent patterns
            sample_patterns = [
                {
                    "pattern": "Deploy {networkFunction} with {replicas} instances in {namespace}",
                    "intentType": "NetworkFunctionDeployment",
                    "parameters": ["networkFunction", "replicas", "namespace"],
                    "examples": [
                        "Deploy AMF with 3 instances in core-network",
                        "Deploy SMF with 2 replicas in 5g-core",
                        "Create UPF deployment with 5 instances"
                    ],
                    "confidence": 0.85
                },
                {
                    "pattern": "Scale {networkFunction} to {replicas} replicas",
                    "intentType": "NetworkFunctionScale",
                    "parameters": ["networkFunction", "replicas"],
                    "examples": [
                        "Scale AMF to 5 replicas",
                        "Increase UPF instances to 10",
                        "Scale down SMF to 2 instances"
                    ],
                    "confidence": 0.90
                },
                {
                    "pattern": "Create network slice for {sliceType} with {sla} requirements",
                    "intentType": "NetworkSliceConfiguration",
                    "parameters": ["sliceType", "sla"],
                    "examples": [
                        "Create network slice for URLLC with 1ms latency",
                        "Configure eMBB slice with high throughput",
                        "Set up mMTC slice for IoT devices"
                    ],
                    "confidence": 0.88
                }
            ]
            
            # Insert sample intent patterns
            with self.client.batch as batch:
                batch.batch_size = 100
                for pattern in sample_patterns:
                    batch.add_data_object(
                        data_object=pattern,
                        class_name="IntentPatterns"
                    )
            
            # Sample network functions
            sample_nfs = [
                {
                    "name": "AMF",
                    "description": "Access and Mobility Management Function handles UE registration, mobility, and access authentication in the 5G core network.",
                    "category": "Core",
                    "interfaces": ["N1", "N2", "N8", "N11", "N12", "N14", "N15", "N22"],
                    "deploymentOptions": ["Standalone", "Cloud-Native", "Containerized"],
                    "resourceRequirements": "CPU: 2-8 cores, Memory: 4-16GB, Storage: 10-50GB",
                    "scalingOptions": "Horizontal scaling supported, Session-aware load balancing",
                    "standardsCompliance": ["3GPP TS 23.501", "3GPP TS 29.518"]
                },
                {
                    "name": "SMF",
                    "description": "Session Management Function manages PDU sessions, QoS policies, and UE IP address allocation.",
                    "category": "Core",
                    "interfaces": ["N4", "N7", "N10", "N11", "N16"],
                    "deploymentOptions": ["Centralized", "Distributed", "Edge"],
                    "resourceRequirements": "CPU: 4-12 cores, Memory: 8-32GB, Storage: 20-100GB",
                    "scalingOptions": "Session-based scaling, Geographic distribution",
                    "standardsCompliance": ["3GPP TS 23.501", "3GPP TS 29.502"]
                }
            ]
            
            # Insert sample network functions
            with self.client.batch as batch:
                batch.batch_size = 100
                for nf in sample_nfs:
                    batch.add_data_object(
                        data_object=nf,
                        class_name="NetworkFunctions"
                    )
            
            logger.info("Successfully populated sample data")
            return True
            
        except Exception as e:
            logger.error(f"Failed to populate sample data: {e}")
            return False
    
    def validate_schemas(self) -> bool:
        """Validate that all schemas are properly created and accessible"""
        try:
            schema = self.client.schema.get()
            expected_classes = ["TelecomKnowledge", "IntentPatterns", "NetworkFunctions"]
            existing_classes = [cls["class"] for cls in schema.get("classes", [])]
            
            for expected_class in expected_classes:
                if expected_class not in existing_classes:
                    logger.error(f"Schema validation failed: {expected_class} not found")
                    return False
            
            # Test basic queries
            for class_name in expected_classes:
                try:
                    result = self.client.query.get(class_name).with_limit(1).do()
                    logger.info(f"Schema {class_name} is accessible and queryable")
                except Exception as e:
                    logger.warning(f"Query test failed for {class_name}: {e}")
            
            logger.info("Schema validation completed successfully")
            return True
            
        except Exception as e:
            logger.error(f"Schema validation failed: {e}")
            return False

def main():
    """Main function to initialize telecom schemas"""
    # Get configuration from environment variables
    weaviate_url = os.getenv("WEAVIATE_URL", "http://weaviate:8080")
    api_key = os.getenv("WEAVIATE_API_KEY", "nephoran-rag-key-production")
    
    if not os.getenv("OPENAI_API_KEY"):
        logger.error("OPENAI_API_KEY environment variable is required")
        sys.exit(1)
    
    try:
        # Initialize schema manager
        schema_manager = TelecomSchemaManager(weaviate_url, api_key)
        
        # Initialize schemas
        logger.info("Initializing telecom-specific schemas...")
        if not schema_manager.initialize_schemas():
            logger.error("Failed to initialize schemas")
            sys.exit(1)
        
        # Populate with sample data
        logger.info("Populating schemas with sample data...")
        if not schema_manager.populate_sample_data():
            logger.error("Failed to populate sample data")
            sys.exit(1)
        
        # Validate schemas
        logger.info("Validating schema installation...")
        if not schema_manager.validate_schemas():
            logger.error("Schema validation failed")
            sys.exit(1)
        
        logger.info("Telecom schema initialization completed successfully!")
        
    except Exception as e:
        logger.error(f"Schema initialization failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()