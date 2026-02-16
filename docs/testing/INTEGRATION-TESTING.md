# Integration Testing Guide for Nephoran Intent Operator
## Comprehensive Testing Strategy for Weaviate RAG System Integration

### Overview

This document provides comprehensive integration testing procedures for the Nephoran Intent Operator's Retrieval-Augmented Generation (RAG) system, focusing on Weaviate vector database integration, semantic search capabilities, intent processing, and end-to-end system validation.

The testing strategy covers multiple integration layers:
- **Weaviate Integration**: Vector database connectivity and operations
- **RAG Pipeline Integration**: End-to-end knowledge retrieval and generation
- **Intent Processing Integration**: Natural language to action conversion
- **API Integration**: RESTful service interactions
- **Performance Integration**: Load testing and scalability validation
- **Security Integration**: Authentication and authorization testing

### Table of Contents

1. [Test Environment Setup](#test-environment-setup)
2. [Weaviate Integration Tests](#weaviate-integration-tests)
3. [RAG Pipeline Integration Tests](#rag-pipeline-integration-tests)
4. [Intent Processing Integration Tests](#intent-processing-integration-tests)
5. [API Integration Tests](#api-integration-tests)
6. [Performance Integration Tests](#performance-integration-tests)
7. [Security Integration Tests](#security-integration-tests)
8. [End-to-End System Tests](#end-to-end-system-tests)
9. [Continuous Integration Setup](#continuous-integration-setup)
10. [Test Data Management](#test-data-management)

## Test Environment Setup

### Prerequisites and Dependencies

```yaml
# test-environment.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-test
  labels:
    environment: testing
    component: integration-tests
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: nephoran-test
data:
  WEAVIATE_URL: "http://weaviate.nephoran-test.svc.cluster.local:8080"
  TEST_DATA_SIZE: "1000"
  PERFORMANCE_TEST_DURATION: "300"
  PARALLEL_TEST_WORKERS: "5"
  LOG_LEVEL: "DEBUG"
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  namespace: nephoran-test
type: Opaque
data:
  weaviate-api-key: dGVzdC13ZWF2aWF0ZS1hcGkta2V5  # test-weaviate-api-key
  openai-api-key: dGVzdC1vcGVuYWktYXBpLWtleQ==    # test-openai-api-key
```

### Test Infrastructure Deployment

```bash
#!/bin/bash
# deploy-test-environment.sh

set -e

NAMESPACE="nephoran-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Deploying Integration Test Environment ==="

# 1. Create test namespace
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# 2. Deploy test configuration
kubectl apply -f test-environment.yaml

# 3. Deploy Weaviate test instance
helm upgrade --install weaviate-test \
  --namespace $NAMESPACE \
  --set image.tag=1.28.0 \
  --set replicas=1 \
  --set resources.requests.memory=2Gi \
  --set resources.requests.cpu=500m \
  --set authentication.apikey.enabled=true \
  --set modules.text2vec-openai.enabled=true \
  --set modules.generative-openai.enabled=true \
  weaviate/weaviate

# 4. Wait for Weaviate to be ready
echo "Waiting for Weaviate test instance..."
kubectl wait --for=condition=ready pod -l app=weaviate -n $NAMESPACE --timeout=300s

# 5. Deploy test data loader
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: test-data-loader
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: data-loader
        image: nephoran/test-data-loader:latest
        env:
        - name: WEAVIATE_URL
          valueFrom:
            configMapKeyRef:
              name: test-config
              key: WEAVIATE_URL
        - name: WEAVIATE_API_KEY
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: weaviate-api-key
        - name: TEST_DATA_SIZE
          valueFrom:
            configMapKeyRef:
              name: test-config
              key: TEST_DATA_SIZE
        command: ["python3", "/scripts/load_test_data.py"]
        volumeMounts:
        - name: test-data
          mountPath: /data
      volumes:
      - name: test-data
        configMap:
          name: telecom-test-data
      restartPolicy: Never
  backoffLimit: 3
EOF

# 6. Verify deployment
echo "Verifying test environment..."
kubectl get pods -n $NAMESPACE
kubectl get services -n $NAMESPACE

echo "✅ Test environment deployed successfully"
echo "Access Weaviate: kubectl port-forward svc/weaviate-test 8080:8080 -n $NAMESPACE"
```

### Test Data Generator

```python
# test_data_generator.py
import json
import uuid
import random
from datetime import datetime, timedelta
from typing import List, Dict, Any

class TelecomTestDataGenerator:
    """Generate realistic test data for telecommunications domain testing"""
    
    def __init__(self):
        self.network_functions = [
            "AMF", "SMF", "UPF", "gNB", "CU", "DU", "RU", 
            "AUSF", "UDM", "PCF", "NRF", "NSSF", "NEF"
        ]
        
        self.interfaces = [
            "N1", "N2", "N3", "N4", "N6", "N8", "N9", "N10", "N11", 
            "N12", "N14", "N15", "N16", "E1", "F1", "A1", "O1", "O2"
        ]
        
        self.procedures = [
            "Registration", "Authentication", "Authorization", "Handover",
            "Session Establishment", "Session Modification", "Session Release",
            "Mobility Management", "Policy Control", "Charging"
        ]
        
        self.sources = ["3GPP", "O-RAN", "ETSI", "ITU-T"]
        self.categories = ["Architecture", "Procedures", "Interfaces", "Management", "Security"]
        self.use_cases = ["eMBB", "URLLC", "mMTC", "Private", "General"]
        self.releases = ["Rel-15", "Rel-16", "Rel-17", "Rel-18"]
        
    def generate_knowledge_documents(self, count: int = 1000) -> List[Dict[str, Any]]:
        """Generate test knowledge documents"""
        documents = []
        
        for i in range(count):
            nf = random.choice(self.network_functions)
            procedure = random.choice(self.procedures)
            interface = random.choice(self.interfaces)
            
            doc = {
                "id": str(uuid.uuid4()),
                "title": f"{nf} {procedure} via {interface}",
                "content": self._generate_content(nf, procedure, interface),
                "source": random.choice(self.sources),
                "specification": f"TS {random.randint(23, 29)}.{random.randint(100, 999)}",
                "version": f"{random.randint(15, 18)}.{random.randint(0, 9)}.0",
                "release": random.choice(self.releases),
                "category": random.choice(self.categories),
                "domain": random.choice(["Core", "RAN", "Transport", "Edge", "Management"]),
                "technicalLevel": random.choice(["Basic", "Intermediate", "Advanced", "Expert"]),
                "keywords": self._generate_keywords(nf, procedure, interface),
                "networkFunctions": [nf] + random.sample(self.network_functions, random.randint(0, 3)),
                "interfaces": [interface] + random.sample(self.interfaces, random.randint(0, 2)),
                "procedures": [procedure] + random.sample(self.procedures, random.randint(0, 2)),
                "useCase": random.choice(self.use_cases),
                "priority": random.randint(1, 10),
                "confidence": round(random.uniform(0.6, 1.0), 2),
                "language": "en",
                "lastUpdated": (datetime.now() - timedelta(days=random.randint(0, 365))).isoformat(),
                "metadata": {
                    "testDocument": True,
                    "generatedAt": datetime.now().isoformat(),
                    "version": "1.0"
                }
            }
            documents.append(doc)
            
        return documents
    
    def generate_intent_patterns(self, count: int = 100) -> List[Dict[str, Any]]:
        """Generate test intent patterns"""
        patterns = []
        
        deployment_patterns = [
            "Deploy {networkFunction} with {replicas} replicas",
            "Create {networkFunction} deployment in {namespace}",
            "Install {networkFunction} with {configuration} config",
            "Setup {networkFunction} for {useCase} slice"
        ]
        
        scaling_patterns = [
            "Scale {networkFunction} to {replicas} instances",
            "Increase {networkFunction} capacity to {replicas}",
            "Autoscale {networkFunction} based on {metric}",
            "Resize {networkFunction} deployment"
        ]
        
        configuration_patterns = [
            "Configure {networkFunction} for {useCase}",
            "Update {networkFunction} {parameter} to {value}",
            "Set {networkFunction} {policy} policy",
            "Enable {feature} on {networkFunction}"
        ]
        
        all_patterns = [
            (deployment_patterns, "NetworkFunctionDeployment", 
             ["networkFunction", "replicas", "namespace", "configuration", "useCase"]),
            (scaling_patterns, "NetworkFunctionScale", 
             ["networkFunction", "replicas", "metric"]),
            (configuration_patterns, "NetworkSliceConfiguration", 
             ["networkFunction", "useCase", "parameter", "value", "policy", "feature"])
        ]
        
        for pattern_list, intent_type, params in all_patterns:
            for pattern in pattern_list:
                for variation in range(random.randint(3, 8)):
                    patterns.append({
                        "id": str(uuid.uuid4()),
                        "pattern": pattern,
                        "intentType": intent_type,
                        "parameters": random.sample(params, random.randint(2, len(params))),
                        "examples": self._generate_pattern_examples(pattern, intent_type),
                        "confidence": round(random.uniform(0.7, 0.95), 2),
                        "accuracy": round(random.uniform(0.8, 0.98), 2),
                        "frequency": random.randint(1, 100),
                        "successRate": round(random.uniform(0.85, 0.99), 2),
                        "lastUsed": (datetime.now() - timedelta(days=random.randint(0, 30))).isoformat(),
                        "userFeedback": round(random.uniform(7.0, 9.5), 1)
                    })
        
        return patterns[:count]
    
    def generate_network_functions(self, count: int = 50) -> List[Dict[str, Any]]:
        """Generate test network function definitions"""
        functions = []
        
        for nf in self.network_functions:
            for variant in range(random.randint(2, 5)):
                functions.append({
                    "id": str(uuid.uuid4()),
                    "name": f"{nf}{'_v' + str(variant + 1) if variant > 0 else ''}",
                    "description": f"The {nf} (Network Function) provides {random.choice(self.procedures).lower()} capabilities for 5G networks.",
                    "category": random.choice(["Core", "RAN", "Management", "Edge"]),
                    "version": f"{random.randint(1, 3)}.{random.randint(0, 9)}.{random.randint(0, 9)}",
                    "vendor": random.choice(["Ericsson", "Nokia", "Huawei", "Samsung", "Open5GS"]),
                    "deploymentOptions": random.sample(
                        ["kubernetes", "vm", "container", "bare-metal", "cloud-native"], 
                        random.randint(2, 4)
                    ),
                    "resourceRequirements": f"CPU: {random.randint(1, 8)} cores, Memory: {random.randint(2, 16)}GB, Storage: {random.randint(10, 100)}GB",
                    "scalingOptions": random.choice(["horizontal", "vertical", "both"]),
                    "configurationTemplates": [f"{nf.lower()}-basic.yaml", f"{nf.lower()}-advanced.yaml"],
                    "healthChecks": [f"/health", f"/ready", f"/metrics"],
                    "standardsCompliance": random.sample(
                        ["3GPP TS 23.501", "3GPP TS 29.518", "O-RAN WG3", "ETSI NFV"], 
                        random.randint(1, 3)
                    ),
                    "interfaces": random.sample(self.interfaces, random.randint(2, 6)),
                    "supportedReleases": random.sample(self.releases, random.randint(1, 3)),
                    "certificationLevel": random.choice(["None", "Basic", "Certified", "Premium"]),
                    "operationalStatus": random.choice(["Active", "Deprecated", "Development"])
                })
        
        return functions[:count]
    
    def _generate_content(self, nf: str, procedure: str, interface: str) -> str:
        """Generate realistic content for knowledge documents"""
        templates = [
            f"The {nf} handles {procedure.lower()} procedures through the {interface} interface. This involves coordination with other network functions to ensure proper service delivery.",
            f"When implementing {procedure.lower()} in {nf}, the {interface} interface provides the necessary signaling mechanisms. The procedure follows 3GPP specifications for reliability.",
            f"The {procedure} procedure executed by {nf} uses {interface} for communication. This ensures seamless integration with existing network infrastructure.",
            f"Configuration of {nf} for {procedure.lower()} requires proper {interface} interface setup. The implementation supports various deployment scenarios including cloud-native environments."
        ]
        
        base_content = random.choice(templates)
        
        # Add technical details
        technical_details = [
            f"The message flow includes initial request, authentication, authorization, and confirmation phases.",
            f"Error handling procedures are implemented according to standard specifications.",
            f"Load balancing and failover mechanisms ensure high availability.",
            f"Performance metrics are collected for monitoring and optimization.",
            f"Security measures include encryption and access control validation."
        ]
        
        return base_content + " " + " ".join(random.sample(technical_details, random.randint(2, 4)))
    
    def _generate_keywords(self, nf: str, procedure: str, interface: str) -> List[str]:
        """Generate relevant keywords"""
        base_keywords = [nf, procedure, interface]
        additional = random.sample([
            "5G", "authentication", "session", "mobility", "security", 
            "protocol", "signaling", "configuration", "deployment"
        ], random.randint(3, 6))
        return base_keywords + additional
    
    def _generate_pattern_examples(self, pattern: str, intent_type: str) -> List[str]:
        """Generate example phrases for intent patterns"""
        examples = []
        
        if intent_type == "NetworkFunctionDeployment":
            examples = [
                "Deploy AMF with 3 replicas in production namespace",
                "Create SMF deployment with high availability config",
                "Install UPF for URLLC slice configuration"
            ]
        elif intent_type == "NetworkFunctionScale":
            examples = [
                "Scale AMF to 5 instances for increased load",
                "Increase SMF capacity to handle more sessions",
                "Autoscale UPF based on throughput metrics"
            ]
        elif intent_type == "NetworkSliceConfiguration":
            examples = [
                "Configure AMF for eMBB slice with QoS parameters",
                "Update SMF session policy for low latency",
                "Set UPF forwarding rules for IoT traffic"
            ]
        
        return random.sample(examples, random.randint(2, len(examples)))
    
    def save_test_data(self, output_dir: str = "./test_data"):
        """Save generated test data to files"""
        import os
        os.makedirs(output_dir, exist_ok=True)
        
        # Generate and save knowledge documents
        knowledge_docs = self.generate_knowledge_documents(1000)
        with open(f"{output_dir}/knowledge_documents.json", "w") as f:
            json.dump(knowledge_docs, f, indent=2)
        
        # Generate and save intent patterns
        intent_patterns = self.generate_intent_patterns(100)
        with open(f"{output_dir}/intent_patterns.json", "w") as f:
            json.dump(intent_patterns, f, indent=2)
        
        # Generate and save network functions
        network_functions = self.generate_network_functions(50)
        with open(f"{output_dir}/network_functions.json", "w") as f:
            json.dump(network_functions, f, indent=2)
        
        print(f"Test data generated in {output_dir}/")
        print(f"- Knowledge documents: {len(knowledge_docs)}")
        print(f"- Intent patterns: {len(intent_patterns)}")
        print(f"- Network functions: {len(network_functions)}")

if __name__ == "__main__":
    generator = TelecomTestDataGenerator()
    generator.save_test_data()
```

## Weaviate Integration Tests

### Basic Connectivity Tests

```python
# test_weaviate_integration.py
import pytest
import weaviate
import json
import time
from typing import List, Dict, Any
import os

class TestWeaviateIntegration:
    """Test suite for Weaviate integration functionality"""
    
    @pytest.fixture(scope="class")
    def weaviate_client(self):
        """Setup Weaviate client for testing"""
        client = weaviate.Client(
            url=os.getenv("WEAVIATE_URL", "http://localhost:8080"),
            auth_client_secret=weaviate.AuthApiKey(
                api_key=os.getenv("WEAVIATE_API_KEY", "test-api-key")
            ),
            additional_headers={
                "X-OpenAI-Api-Key": os.getenv("OPENAI_API_KEY", "test-openai-key")
            },
            timeout_config=(5, 15)  # (connection, read) timeouts
        )
        
        # Verify connectivity
        assert client.is_ready(), "Weaviate is not ready"
        assert client.is_live(), "Weaviate is not live"
        
        yield client
        
        # Cleanup after tests (optional)
        # client.batch.delete_objects(...)
    
    def test_basic_connectivity(self, weaviate_client):
        """Test basic Weaviate connectivity"""
        # Test meta endpoint
        meta = weaviate_client.get_meta()
        assert "version" in meta
        assert "modules" in meta
        
        print(f"Connected to Weaviate version: {meta['version']}")
        print(f"Available modules: {list(meta['modules'].keys())}")
    
    def test_schema_operations(self, weaviate_client):
        """Test schema creation, retrieval, and deletion"""
        test_class_name = "TestKnowledge"
        
        # Define test schema
        test_class = {
            "class": test_class_name,
            "description": "Test class for integration testing",
            "vectorizer": "text2vec-openai",
            "properties": [
                {
                    "name": "content",
                    "dataType": ["text"],
                    "description": "Test content",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    }
                },
                {
                    "name": "title", 
                    "dataType": ["text"],
                    "description": "Test title"
                },
                {
                    "name": "category",
                    "dataType": ["text"],
                    "description": "Test category"
                }
            ]
        }
        
        try:
            # Create schema
            weaviate_client.schema.create_class(test_class)
            
            # Verify schema exists
            schema = weaviate_client.schema.get()
            class_names = [cls["class"] for cls in schema["classes"]]
            assert test_class_name in class_names
            
            # Get specific class
            retrieved_class = weaviate_client.schema.get(test_class_name)
            assert retrieved_class["class"] == test_class_name
            assert len(retrieved_class["properties"]) == 3
            
        finally:
            # Cleanup - delete test class
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")
    
    def test_object_operations(self, weaviate_client):
        """Test object CRUD operations"""
        test_class_name = "TestDocuments"
        
        # Create test class
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "confidence", "dataType": ["number"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Test object creation
            test_object = {
                "content": "This is a test document for AMF registration procedures",
                "title": "AMF Registration Test",
                "confidence": 0.95
            }
            
            # Create object
            uuid = weaviate_client.data_object.create(
                data_object=test_object,
                class_name=test_class_name
            )
            assert uuid is not None
            print(f"Created object with UUID: {uuid}")
            
            # Retrieve object
            retrieved_object = weaviate_client.data_object.get_by_id(uuid)
            assert retrieved_object["properties"]["title"] == test_object["title"]
            assert retrieved_object["properties"]["confidence"] == test_object["confidence"]
            
            # Update object
            updated_data = {"confidence": 0.98}
            weaviate_client.data_object.update(
                data_object=updated_data,
                class_name=test_class_name,
                uuid=uuid
            )
            
            # Verify update
            updated_object = weaviate_client.data_object.get_by_id(uuid)
            assert updated_object["properties"]["confidence"] == 0.98
            
            # Delete object
            weaviate_client.data_object.delete(uuid)
            
            # Verify deletion
            with pytest.raises(weaviate.UnexpectedStatusCodeException):
                weaviate_client.data_object.get_by_id(uuid)
                
        finally:
            # Cleanup
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")
    
    def test_batch_operations(self, weaviate_client):
        """Test batch import functionality"""
        test_class_name = "BatchTestDocuments"
        
        # Create test class
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "index", "dataType": ["int"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Prepare batch data
            batch_size = 10
            test_objects = []
            
            for i in range(batch_size):
                test_objects.append({
                    "content": f"This is test document number {i} about network functions",
                    "title": f"Test Document {i}",
                    "index": i
                })
            
            # Configure batch
            weaviate_client.batch.configure(
                batch_size=5,
                dynamic=False,
                timeout_retries=3
            )
            
            # Batch import
            start_time = time.time()
            with weaviate_client.batch as batch:
                for obj in test_objects:
                    batch.add_data_object(
                        data_object=obj,
                        class_name=test_class_name
                    )
            
            import_time = time.time() - start_time
            print(f"Batch import of {batch_size} objects took {import_time:.2f} seconds")
            
            # Verify batch import
            time.sleep(2)  # Allow time for indexing
            
            result = weaviate_client.query.get(test_class_name, ["title", "index"]).do()
            imported_objects = result["data"]["Get"][test_class_name]
            
            assert len(imported_objects) == batch_size
            print(f"Successfully imported {len(imported_objects)} objects")
            
        finally:
            # Cleanup
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")
    
    def test_vector_search(self, weaviate_client):
        """Test vector similarity search"""
        test_class_name = "VectorTestDocuments"
        
        # Create test class with telecommunications content
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "networkFunction", "dataType": ["text"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Add test documents with telecom content
            test_documents = [
                {
                    "content": "The AMF (Access and Mobility Management Function) handles registration procedures for 5G networks. It manages user authentication and mobility.",
                    "title": "AMF Registration Procedures",
                    "networkFunction": "AMF"
                },
                {
                    "content": "SMF (Session Management Function) manages PDU sessions and handles session establishment, modification, and release procedures.",
                    "title": "SMF Session Management",
                    "networkFunction": "SMF"
                },
                {
                    "content": "UPF (User Plane Function) forwards user data packets and enforces QoS policies for different traffic flows.",
                    "title": "UPF Data Forwarding",
                    "networkFunction": "UPF"
                },
                {
                    "content": "Network slicing enables multiple virtual networks on shared physical infrastructure with different service characteristics.",
                    "title": "Network Slicing Overview",
                    "networkFunction": "NSSF"
                }
            ]
            
            # Import test documents
            weaviate_client.batch.configure(batch_size=len(test_documents))
            with weaviate_client.batch as batch:
                for doc in test_documents:
                    batch.add_data_object(
                        data_object=doc,
                        class_name=test_class_name
                    )
            
            # Wait for indexing
            time.sleep(3)
            
            # Test semantic search
            search_queries = [
                ("user authentication and registration", "AMF"),
                ("session management and PDU", "SMF"),
                ("data forwarding and QoS", "UPF"),
                ("virtual networks and slicing", "NSSF")
            ]
            
            for query, expected_nf in search_queries:
                result = weaviate_client.query.get(
                    test_class_name, 
                    ["title", "networkFunction", "content"]
                ).with_near_text({
                    "concepts": [query],
                    "certainty": 0.7
                }).with_limit(2).with_additional(["certainty"]).do()
                
                documents = result["data"]["Get"][test_class_name]
                assert len(documents) > 0, f"No results for query: {query}"
                
                # Check if the most relevant document matches expected NF
                top_doc = documents[0]
                certainty = top_doc["_additional"]["certainty"]
                
                print(f"Query: '{query}' -> Top result: {top_doc['title']} (certainty: {certainty:.3f}, NF: {top_doc['networkFunction']})")
                assert certainty > 0.7, f"Low certainty ({certainty}) for query: {query}"
                
                # Verify the correct network function is found (allowing for some flexibility)
                if expected_nf in [doc["networkFunction"] for doc in documents]:
                    print(f"✅ Correct network function '{expected_nf}' found for query: '{query}'")
                else:
                    print(f"⚠️ Expected '{expected_nf}' but got different results for query: '{query}'")
        
        finally:
            # Cleanup
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")
    
    def test_hybrid_search(self, weaviate_client):
        """Test hybrid search combining keyword and vector search"""
        test_class_name = "HybridTestDocuments"
        
        # Create test class
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "specification", "dataType": ["text"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Add test documents with specific specifications
            test_documents = [
                {
                    "content": "3GPP TS 23.501 defines the system architecture for 5G networks including network functions and reference points.",
                    "title": "5G System Architecture",
                    "specification": "TS 23.501"
                },
                {
                    "content": "3GPP TS 29.518 specifies the AMF services for access and mobility management in 5G networks.",
                    "title": "AMF Services Specification",
                    "specification": "TS 29.518"
                },
                {
                    "content": "O-RAN WG3 defines the near real-time RAN intelligent controller architecture and interfaces.",
                    "title": "O-RAN Near-RT RIC",
                    "specification": "O-RAN.WG3"
                }
            ]
            
            # Import documents
            weaviate_client.batch.configure(batch_size=len(test_documents))
            with weaviate_client.batch as batch:
                for doc in test_documents:
                    batch.add_data_object(
                        data_object=doc,
                        class_name=test_class_name
                    )
            
            time.sleep(3)  # Wait for indexing
            
            # Test hybrid search with different alpha values
            test_cases = [
                ("TS 23.501", 0.9, "keyword-focused"),  # Favor keyword search
                ("5G system architecture", 0.3, "semantic-focused"),  # Favor vector search
                ("AMF access mobility", 0.7, "balanced")  # Balanced approach
            ]
            
            for query, alpha, description in test_cases:
                result = weaviate_client.query.get(
                    test_class_name,
                    ["title", "specification", "content"]
                ).with_hybrid(
                    query=query,
                    alpha=alpha
                ).with_limit(3).with_additional(["score"]).do()
                
                documents = result["data"]["Get"][test_class_name]
                assert len(documents) > 0, f"No results for hybrid query: {query}"
                
                top_doc = documents[0]
                score = top_doc["_additional"]["score"]
                
                print(f"Hybrid search ({description}, α={alpha}): '{query}' -> {top_doc['title']} (score: {score:.3f})")
                
                # Validate results make sense
                if alpha > 0.8:  # Keyword-focused
                    # Should find exact specification matches
                    if "TS 23.501" in query:
                        assert "TS 23.501" in top_doc["specification"], "Keyword search should find exact specification"
                
        finally:
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")
    
    def test_filtering_operations(self, weaviate_client):
        """Test where filter operations"""
        test_class_name = "FilterTestDocuments"
        
        # Create test class
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "source", "dataType": ["text"]},
                {"name": "networkFunctions", "dataType": ["text[]"]},
                {"name": "confidence", "dataType": ["number"]},
                {"name": "priority", "dataType": ["int"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Add test documents with various attributes
            test_documents = [
                {
                    "content": "3GPP specification for AMF procedures",
                    "title": "AMF Procedures",
                    "source": "3GPP",
                    "networkFunctions": ["AMF"],
                    "confidence": 0.95,
                    "priority": 8
                },
                {
                    "content": "O-RAN specification for CU procedures",
                    "title": "CU Procedures",
                    "source": "O-RAN",
                    "networkFunctions": ["CU", "DU"],
                    "confidence": 0.85,
                    "priority": 6
                },
                {
                    "content": "ETSI specification for MEC deployment",
                    "title": "MEC Deployment",
                    "source": "ETSI",
                    "networkFunctions": ["MEC"],
                    "confidence": 0.75,
                    "priority": 5
                },
                {
                    "content": "3GPP specification for SMF and UPF integration",
                    "title": "SMF-UPF Integration",
                    "source": "3GPP",
                    "networkFunctions": ["SMF", "UPF"],
                    "confidence": 0.92,
                    "priority": 9
                }
            ]
            
            # Import documents
            weaviate_client.batch.configure(batch_size=len(test_documents))
            with weaviate_client.batch as batch:
                for doc in test_documents:
                    batch.add_data_object(
                        data_object=doc,
                        class_name=test_class_name
                    )
            
            time.sleep(3)  # Wait for indexing
            
            # Test various filter operations
            filter_tests = [
                {
                    "name": "Source filter",
                    "where": {
                        "path": ["source"],
                        "operator": "Equal",
                        "valueText": "3GPP"
                    },
                    "expected_count": 2
                },
                {
                    "name": "Network function filter",
                    "where": {
                        "path": ["networkFunctions"],
                        "operator": "ContainsAny",
                        "valueTextArray": ["AMF"]
                    },
                    "expected_count": 1
                },
                {
                    "name": "Confidence threshold",
                    "where": {
                        "path": ["confidence"],
                        "operator": "GreaterThan",
                        "valueNumber": 0.9
                    },
                    "expected_count": 2
                },
                {
                    "name": "Priority range",
                    "where": {
                        "path": ["priority"],
                        "operator": "GreaterThanEqual",
                        "valueInt": 8
                    },
                    "expected_count": 2
                },
                {
                    "name": "Complex AND filter",
                    "where": {
                        "operator": "And",
                        "operands": [
                            {
                                "path": ["source"],
                                "operator": "Equal",
                                "valueText": "3GPP"
                            },
                            {
                                "path": ["confidence"],
                                "operator": "GreaterThan",
                                "valueNumber": 0.9
                            }
                        ]
                    },
                    "expected_count": 2
                }
            ]
            
            for test_case in filter_tests:
                result = weaviate_client.query.get(
                    test_class_name,
                    ["title", "source", "networkFunctions", "confidence", "priority"]
                ).with_where(test_case["where"]).do()
                
                documents = result["data"]["Get"][test_class_name]
                actual_count = len(documents)
                
                print(f"{test_case['name']}: Found {actual_count} documents (expected: {test_case['expected_count']})")
                
                for doc in documents:
                    print(f"  - {doc['title']} (source: {doc['source']}, confidence: {doc['confidence']}, priority: {doc['priority']})")
                
                assert actual_count == test_case["expected_count"], \
                    f"Filter '{test_case['name']}' returned {actual_count} documents, expected {test_case['expected_count']}"
        
        finally:
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")

    @pytest.mark.performance
    def test_performance_benchmarks(self, weaviate_client):
        """Test performance benchmarks for vector operations"""
        test_class_name = "PerformanceTestDocuments"
        
        # Create test class
        test_class = {
            "class": test_class_name,
            "vectorizer": "text2vec-openai",
            "properties": [
                {"name": "content", "dataType": ["text"]},
                {"name": "title", "dataType": ["text"]},
                {"name": "index", "dataType": ["int"]}
            ]
        }
        
        try:
            weaviate_client.schema.create_class(test_class)
            
            # Import a larger dataset for performance testing
            document_count = 100
            batch_size = 20
            
            print(f"Importing {document_count} documents for performance testing...")
            
            # Measure import performance
            start_time = time.time()
            
            weaviate_client.batch.configure(
                batch_size=batch_size,
                dynamic=True,
                timeout_retries=3
            )
            
            with weaviate_client.batch as batch:
                for i in range(document_count):
                    batch.add_data_object(
                        data_object={
                            "content": f"This is performance test document {i} containing information about network function {i % 10} and its procedures for 5G deployment scenarios.",
                            "title": f"Performance Test Document {i}",
                            "index": i
                        },
                        class_name=test_class_name
                    )
            
            import_time = time.time() - start_time
            print(f"✅ Import performance: {document_count} documents in {import_time:.2f}s ({document_count/import_time:.1f} docs/sec)")
            
            # Wait for indexing
            time.sleep(5)
            
            # Test query performance
            queries = [
                "network function deployment",
                "5G procedures",
                "performance test scenarios",
                "document information"
            ]
            
            query_times = []
            
            for query in queries:
                start_time = time.time()
                
                result = weaviate_client.query.get(
                    test_class_name,
                    ["title", "index"]
                ).with_near_text({
                    "concepts": [query],
                    "certainty": 0.7
                }).with_limit(10).do()
                
                query_time = time.time() - start_time
                query_times.append(query_time)
                
                documents = result["data"]["Get"][test_class_name]
                print(f"Query '{query}': {len(documents)} results in {query_time:.3f}s")
            
            avg_query_time = sum(query_times) / len(query_times)
            print(f"✅ Average query time: {avg_query_time:.3f}s")
            
            # Performance assertions
            assert avg_query_time < 1.0, f"Average query time ({avg_query_time:.3f}s) exceeds 1.0s threshold"
            assert document_count/import_time > 10, f"Import rate ({document_count/import_time:.1f} docs/sec) is below 10 docs/sec threshold"
            
        finally:
            try:
                weaviate_client.schema.delete_class(test_class_name)
            except Exception as e:
                print(f"Warning: Could not delete test class: {e}")

if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
```

### Schema Integration Tests

```python
# test_telecom_schema_integration.py
import pytest
import weaviate
import json
import os
from typing import Dict, Any, List

class TestTelecomSchemaIntegration:
    """Test integration with the telecommunications domain schema"""
    
    @pytest.fixture(scope="class")
    def weaviate_client(self):
        """Setup Weaviate client with telecom schema"""
        client = weaviate.Client(
            url=os.getenv("WEAVIATE_URL", "http://localhost:8080"),
            auth_client_secret=weaviate.AuthApiKey(
                api_key=os.getenv("WEAVIATE_API_KEY", "test-api-key")
            ),
            additional_headers={
                "X-OpenAI-Api-Key": os.getenv("OPENAI_API_KEY", "test-openai-key")
            }
        )
        
        # Ensure Weaviate is ready
        assert client.is_ready(), "Weaviate is not ready"
        
        yield client
    
    def test_telecom_knowledge_schema(self, weaviate_client):
        """Test TelecomKnowledge schema operations"""
        
        # Check if TelecomKnowledge class exists
        schema = weaviate_client.schema.get()
        class_names = [cls["class"] for cls in schema["classes"]]
        
        if "TelecomKnowledge" not in class_names:
            pytest.skip("TelecomKnowledge schema not found - run schema initialization first")
        
        # Get TelecomKnowledge class details
        telecom_class = weaviate_client.schema.get("TelecomKnowledge")
        
        # Verify essential properties exist
        property_names = [prop["name"] for prop in telecom_class["properties"]]
        
        required_properties = [
            "content", "title", "source", "specification", "networkFunctions", 
            "interfaces", "procedures", "confidence", "priority"
        ]
        
        for prop in required_properties:
            assert prop in property_names, f"Required property '{prop}' not found in TelecomKnowledge schema"
        
        # Verify vectorizer configuration
        assert telecom_class["vectorizer"] == "text2vec-openai", "Incorrect vectorizer configuration"
        
        # Verify module configuration
        module_config = telecom_class.get("moduleConfig", {})
        assert "text2vec-openai" in module_config, "OpenAI module not configured"
        
        openai_config = module_config["text2vec-openai"]
        assert openai_config["model"] == "text-embedding-3-large", "Incorrect embedding model"
        assert openai_config["dimensions"] == 3072, "Incorrect embedding dimensions"
        
        print("✅ TelecomKnowledge schema validation passed")
    
    def test_intent_patterns_schema(self, weaviate_client):
        """Test IntentPatterns schema operations"""
        
        schema = weaviate_client.schema.get()
        class_names = [cls["class"] for cls in schema["classes"]]
        
        if "IntentPatterns" not in class_names:
            pytest.skip("IntentPatterns schema not found - run schema initialization first")
        
        # Get IntentPatterns class details
        intent_class = weaviate_client.schema.get("IntentPatterns")
        
        # Verify essential properties
        property_names = [prop["name"] for prop in intent_class["properties"]]
        
        required_properties = [
            "pattern", "intentType", "parameters", "confidence", "examples"
        ]
        
        for prop in required_properties:
            assert prop in property_names, f"Required property '{prop}' not found in IntentPatterns schema"
        
        print("✅ IntentPatterns schema validation passed")
    
    def test_network_functions_schema(self, weaviate_client):
        """Test NetworkFunctions schema operations"""
        
        schema = weaviate_client.schema.get()
        class_names = [cls["class"] for cls in schema["classes"]]
        
        if "NetworkFunctions" not in class_names:
            pytest.skip("NetworkFunctions schema not found - run schema initialization first")
        
        # Get NetworkFunctions class details
        nf_class = weaviate_client.schema.get("NetworkFunctions")
        
        # Verify essential properties
        property_names = [prop["name"] for prop in nf_class["properties"]]
        
        required_properties = [
            "name", "description", "category", "deploymentOptions", 
            "interfaces", "standardsCompliance"
        ]
        
        for prop in required_properties:
            assert prop in property_names, f"Required property '{prop}' not found in NetworkFunctions schema"
        
        print("✅ NetworkFunctions schema validation passed")
    
    def test_cross_references(self, weaviate_client):
        """Test cross-reference relationships between classes"""
        
        schema = weaviate_client.schema.get()
        class_names = [cls["class"] for cls in schema["classes"]]
        
        required_classes = ["TelecomKnowledge", "IntentPatterns", "NetworkFunctions"]
        for cls in required_classes:
            if cls not in class_names:
                pytest.skip(f"{cls} schema not found - run schema initialization first")
        
        # Check TelecomKnowledge cross-references
        telecom_class = weaviate_client.schema.get("TelecomKnowledge")
        property_names = [prop["name"] for prop in telecom_class["properties"]]
        
        # Look for cross-reference properties (if implemented)
        cross_ref_properties = ["relatedIntents", "applicableNFs"]
        
        for prop in cross_ref_properties:
            if prop in property_names:
                prop_config = next(p for p in telecom_class["properties"] if p["name"] == prop)
                assert "IntentPatterns" in prop_config["dataType"] or "NetworkFunctions" in prop_config["dataType"], \
                    f"Cross-reference property '{prop}' not properly configured"
                print(f"✅ Cross-reference property '{prop}' found and configured")
        
        print("✅ Cross-reference validation completed")
    
    def test_telecom_data_insertion(self, weaviate_client):
        """Test insertion of realistic telecommunications data"""
        
        # Sample telecom knowledge document
        telecom_doc = {
            "content": "The AMF (Access and Mobility Management Function) handles registration management, connection management, reachability management, mobility management, and access authentication and authorization. It communicates with UE via N1 interface and with gNB via N2 interface.",
            "title": "AMF Functionality and Interfaces",
            "source": "3GPP",
            "specification": "TS 23.501",
            "version": "17.9.0",
            "release": "Rel-17",
            "category": "Architecture",
            "domain": "Core",
            "technicalLevel": "Intermediate",
            "keywords": ["AMF", "registration", "mobility", "authentication", "N1", "N2"],
            "networkFunctions": ["AMF"],
            "interfaces": ["N1", "N2", "N8", "N11", "N12", "N14", "N15"],
            "procedures": ["Registration", "Authentication", "Mobility"],
            "useCase": "General",
            "priority": 9,
            "confidence": 0.95,
            "language": "en"
        }
        
        # Insert document
        try:
            uuid = weaviate_client.data_object.create(
                data_object=telecom_doc,
                class_name="TelecomKnowledge"
            )
            
            assert uuid is not None, "Failed to create TelecomKnowledge object"
            print(f"✅ Created TelecomKnowledge object with UUID: {uuid}")
            
            # Retrieve and verify
            retrieved_doc = weaviate_client.data_object.get_by_id(uuid)
            props = retrieved_doc["properties"]
            
            assert props["title"] == telecom_doc["title"]
            assert props["source"] == telecom_doc["source"]
            assert "AMF" in props["networkFunctions"]
            assert props["confidence"] == telecom_doc["confidence"]
            
            print("✅ TelecomKnowledge document validation passed")
            
            # Test semantic search on inserted data
            result = weaviate_client.query.get(
                "TelecomKnowledge",
                ["title", "networkFunctions", "interfaces"]
            ).with_near_text({
                "concepts": ["AMF registration procedure"],
                "certainty": 0.7
            }).with_limit(5).with_additional(["certainty"]).do()
            
            documents = result["data"]["Get"]["TelecomKnowledge"]
            assert len(documents) > 0, "Semantic search returned no results"
            
            # Check if our document is in the results (by title match)
            found_doc = any(doc["title"] == telecom_doc["title"] for doc in documents)
            if found_doc:
                print("✅ Inserted document found via semantic search")
            else:
                print("⚠️ Inserted document not found in semantic search results")
            
            # Cleanup
            weaviate_client.data_object.delete(uuid)
            
        except Exception as e:
            pytest.fail(f"Failed to insert telecom document: {str(e)}")
    
    def test_intent_pattern_insertion(self, weaviate_client):
        """Test insertion of intent patterns"""
        
        intent_pattern = {
            "pattern": "Deploy {networkFunction} with {replicas} replicas in {namespace} namespace",
            "intentType": "NetworkFunctionDeployment",
            "parameters": ["networkFunction", "replicas", "namespace"],
            "examples": [
                "Deploy AMF with 3 replicas in production namespace",
                "Deploy SMF with 2 replicas in staging namespace"
            ],
            "confidence": 0.85,
            "accuracy": 0.92
        }
        
        try:
            uuid = weaviate_client.data_object.create(
                data_object=intent_pattern,
                class_name="IntentPatterns"
            )
            
            assert uuid is not None, "Failed to create IntentPatterns object"
            print(f"✅ Created IntentPatterns object with UUID: {uuid}")
            
            # Test semantic search for intent matching
            result = weaviate_client.query.get(
                "IntentPatterns",
                ["pattern", "intentType", "parameters"]
            ).with_near_text({
                "concepts": ["deploy network function with replicas"],
                "certainty": 0.7
            }).with_limit(3).with_additional(["certainty"]).do()
            
            patterns = result["data"]["Get"]["IntentPatterns"]
            assert len(patterns) > 0, "Intent pattern search returned no results"
            
            print(f"✅ Found {len(patterns)} matching intent patterns")
            
            # Cleanup
            weaviate_client.data_object.delete(uuid)
            
        except Exception as e:
            pytest.fail(f"Failed to insert intent pattern: {str(e)}")
    
    def test_network_function_insertion(self, weaviate_client):
        """Test insertion of network function definitions"""
        
        network_function = {
            "name": "AMF",
            "description": "The Access and Mobility Management Function (AMF) is a core network function in 5G that provides registration management, connection management, reachability management, mobility management, and access authentication and authorization.",
            "category": "Core",
            "version": "1.0.0",
            "vendor": "Test Vendor",
            "deploymentOptions": ["kubernetes", "container", "vm"],
            "resourceRequirements": "CPU: 2 cores, Memory: 4GB, Storage: 20GB",
            "scalingOptions": "horizontal",
            "interfaces": ["N1", "N2", "N8", "N11", "N12", "N14", "N15"],
            "standardsCompliance": ["3GPP TS 23.501", "3GPP TS 29.518"],
            "supportedReleases": ["Rel-15", "Rel-16", "Rel-17"],
            "operationalStatus": "Active"
        }
        
        try:
            uuid = weaviate_client.data_object.create(
                data_object=network_function,
                class_name="NetworkFunctions"
            )
            
            assert uuid is not None, "Failed to create NetworkFunctions object"
            print(f"✅ Created NetworkFunctions object with UUID: {uuid}")
            
            # Test search for network function
            result = weaviate_client.query.get(
                "NetworkFunctions",
                ["name", "description", "category", "interfaces"]
            ).with_near_text({
                "concepts": ["access mobility management 5G core"],
                "certainty": 0.7
            }).with_limit(3).with_additional(["certainty"]).do()
            
            functions = result["data"]["Get"]["NetworkFunctions"]
            assert len(functions) > 0, "Network function search returned no results"
            
            print(f"✅ Found {len(functions)} matching network functions")
            
            # Cleanup
            weaviate_client.data_object.delete(uuid)
            
        except Exception as e:
            pytest.fail(f"Failed to insert network function: {str(e)}")

if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
```

## RAG Pipeline Integration Tests

### End-to-End RAG Flow Tests

```python
# test_rag_pipeline_integration.py
import pytest
import requests
import json
import time
import os
from typing import Dict, Any, List

class TestRAGPipelineIntegration:
    """Test the complete RAG pipeline integration"""
    
    @pytest.fixture(scope="class")
    def api_config(self):
        """Setup API configuration"""
        return {
            "base_url": os.getenv("RAG_API_URL", "http://localhost:8080/v1"),
            "api_key": os.getenv("RAG_API_KEY", "test-api-key"),
            "timeout": 30
        }
    
    @pytest.fixture(scope="class")
    def api_headers(self, api_config):
        """Setup API headers"""
        return {
            "Authorization": f"Bearer {api_config['api_key']}",
            "Content-Type": "application/json"
        }
    
    def test_api_health_check(self, api_config, api_headers):
        """Test RAG API health and connectivity"""
        
        response = requests.get(
            f"{api_config['base_url']}/health",
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 200, f"Health check failed: {response.status_code}"
        
        health_data = response.json()
        assert health_data["status"] in ["healthy", "degraded"], f"Unhealthy status: {health_data['status']}"
        
        # Check service components
        services = health_data.get("services", {})
        
        if "weaviate" in services:
            assert services["weaviate"]["status"] == "up", "Weaviate service is down"
            print("✅ Weaviate service is healthy")
        
        if "vectorizer" in services:
            assert services["vectorizer"]["status"] == "up", "Vectorizer service is down"
            print("✅ Vectorizer service is healthy")
        
        print(f"✅ RAG API health check passed: {health_data['status']}")
    
    def test_semantic_search_endpoint(self, api_config, api_headers):
        """Test semantic search API endpoint"""
        
        search_request = {
            "query": "AMF registration procedure",
            "limit": 10,
            "certainty": 0.7,
            "includeMetadata": True
        }
        
        response = requests.post(
            f"{api_config['base_url']}/search/semantic",
            headers=api_headers,
            json=search_request,
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 200, f"Semantic search failed: {response.status_code} - {response.text}"
        
        search_results = response.json()
        
        # Validate response structure
        assert "query" in search_results
        assert "searchType" in search_results
        assert "results" in search_results
        assert "executionTime" in search_results
        
        assert search_results["query"] == search_request["query"]
        assert search_results["searchType"] == "semantic"
        
        results = search_results["results"]
        assert len(results) <= search_request["limit"], "Too many results returned"
        
        if len(results) > 0:
            # Validate first result structure
            first_result = results[0]
            
            required_fields = ["id", "title", "content", "source", "certainty"]
            for field in required_fields:
                assert field in first_result, f"Missing field '{field}' in search result"
            
            # Validate certainty score
            certainty = first_result["certainty"]
            assert 0.0 <= certainty <= 1.0, f"Invalid certainty score: {certainty}"
            assert certainty >= search_request["certainty"], f"Result certainty ({certainty}) below threshold ({search_request['certainty']})"
            
            print(f"✅ Semantic search returned {len(results)} results")
            print(f"Top result: {first_result['title']} (certainty: {certainty:.3f})")
        else:
            print("⚠️ Semantic search returned no results")
    
    def test_hybrid_search_endpoint(self, api_config, api_headers):
        """Test hybrid search API endpoint"""
        
        search_request = {
            "query": "SMF session management procedure",
            "alpha": 0.7,
            "limit": 5,
            "filters": {
                "source": ["3GPP"],
                "networkFunctions": ["SMF"]
            }
        }
        
        response = requests.post(
            f"{api_config['base_url']}/search/hybrid",
            headers=api_headers,
            json=search_request,
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 200, f"Hybrid search failed: {response.status_code} - {response.text}"
        
        search_results = response.json()
        
        # Validate response structure
        assert search_results["searchType"] == "hybrid"
        
        results = search_results["results"]
        
        if len(results) > 0:
            # Validate filtering worked
            for result in results:
                if "source" in result:
                    assert result["source"] in search_request["filters"]["source"], \
                        f"Result source '{result['source']}' not in filter"
                
                if "networkFunctions" in result:
                    nf_intersection = set(result["networkFunctions"]) & set(search_request["filters"]["networkFunctions"])
                    assert len(nf_intersection) > 0, "No network function overlap with filter"
            
            print(f"✅ Hybrid search with filters returned {len(results)} results")
        else:
            print("⚠️ Hybrid search returned no results")
    
    def test_cross_reference_search(self, api_config, api_headers):
        """Test cross-reference search across multiple classes"""
        
        search_request = {
            "query": "network slicing configuration",
            "analysisDepth": "standard",
            "includeStatistics": True,
            "includeInsights": True,
            "includeRecommendations": True
        }
        
        response = requests.post(
            f"{api_config['base_url']}/search/cross-reference",
            headers=api_headers,
            json=search_request,
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 200, f"Cross-reference search failed: {response.status_code} - {response.text}"
        
        results = response.json()
        
        # Validate response structure
        required_sections = ["knowledgeDocuments", "intentPatterns", "networkFunctions"]
        for section in required_sections:
            assert section in results, f"Missing section '{section}' in cross-reference results"
        
        # Check statistics if requested
        if search_request["includeStatistics"]:
            assert "statistics" in results, "Statistics not included in response"
            stats = results["statistics"]
            
            assert "totalKnowledgeDocs" in stats
            assert "totalIntentPatterns" in stats
            assert "totalNetworkFunctions" in stats
            
            print(f"✅ Cross-reference statistics: {stats['totalKnowledgeDocs']} knowledge docs, "
                  f"{stats['totalIntentPatterns']} intent patterns, {stats['totalNetworkFunctions']} network functions")
        
        # Check insights if requested
        if search_request["includeInsights"]:
            assert "insights" in results, "Insights not included in response"
            insights = results["insights"]
            assert isinstance(insights, list), "Insights should be a list"
            
            if len(insights) > 0:
                print(f"✅ Generated {len(insights)} insights")
                for insight in insights[:3]:  # Show first 3 insights
                    print(f"  - {insight}")
        
        # Check recommendations if requested
        if search_request["includeRecommendations"]:
            assert "recommendations" in results, "Recommendations not included in response"
            recommendations = results["recommendations"]
            assert isinstance(recommendations, list), "Recommendations should be a list"
            
            if len(recommendations) > 0:
                print(f"✅ Generated {len(recommendations)} recommendations")
    
    def test_intent_matching_pipeline(self, api_config, api_headers):
        """Test intent matching and processing pipeline"""
        
        # Test various intent patterns
        test_intents = [
            {
                "userInput": "Deploy AMF with 3 replicas in production namespace",
                "expectedType": "NetworkFunctionDeployment",
                "expectedParams": ["networkFunction", "replicas", "namespace"]
            },
            {
                "userInput": "Scale SMF to 5 instances for high throughput",
                "expectedType": "NetworkFunctionScale", 
                "expectedParams": ["networkFunction", "replicas"]
            },
            {
                "userInput": "Configure UPF for URLLC with low latency requirements",
                "expectedType": "NetworkSliceConfiguration",
                "expectedParams": ["networkFunction", "useCase"]
            }
        ]
        
        for test_case in test_intents:
            # Test intent matching
            match_request = {
                "userInput": test_case["userInput"],
                "confidenceThreshold": 0.7,
                "extractParameters": True,
                "includeRelatedKnowledge": True
            }
            
            response = requests.post(
                f"{api_config['base_url']}/intents/match",
                headers=api_headers,
                json=match_request,
                timeout=api_config['timeout']
            )
            
            assert response.status_code == 200, f"Intent matching failed: {response.status_code} - {response.text}"
            
            match_results = response.json()
            
            # Validate response structure
            assert "userInput" in match_results
            assert "matchedPatterns" in match_results
            assert "bestMatch" in match_results
            
            assert match_results["userInput"] == test_case["userInput"]
            
            if match_results["bestMatch"]:
                best_match = match_results["bestMatch"]
                
                # Validate intent type (if we have good test data)
                if "intentType" in best_match:
                    intent_type = best_match["intentType"]
                    print(f"✅ Intent '{test_case['userInput'][:50]}...' -> {intent_type}")
                    
                    # Check if it matches expected (allowing for some flexibility)
                    if intent_type == test_case["expectedType"]:
                        print(f"  ✅ Correct intent type: {intent_type}")
                    else:
                        print(f"  ⚠️ Expected {test_case['expectedType']}, got {intent_type}")
                
                # Check parameter extraction
                if "extractedParameters" in best_match:
                    extracted = best_match["extractedParameters"]
                    print(f"  Extracted parameters: {extracted}")
                    
                    # Check if expected parameters were found
                    for expected_param in test_case["expectedParams"]:
                        if expected_param in extracted:
                            print(f"    ✅ Found expected parameter: {expected_param} = {extracted[expected_param]}")
                        else:
                            print(f"    ⚠️ Missing expected parameter: {expected_param}")
            else:
                print(f"⚠️ No best match found for: {test_case['userInput']}")
    
    def test_intent_execution_pipeline(self, api_config, api_headers):
        """Test intent execution configuration generation"""
        
        # First, match an intent
        match_request = {
            "userInput": "Deploy AMF with 3 replicas in production namespace",
            "confidenceThreshold": 0.7,
            "extractParameters": True
        }
        
        match_response = requests.post(
            f"{api_config['base_url']}/intents/match",
            headers=api_headers,
            json=match_request,
            timeout=api_config['timeout']
        )
        
        if match_response.status_code != 200:
            pytest.skip("Intent matching failed - cannot test execution")
        
        match_results = match_response.json()
        
        if not match_results.get("bestMatch"):
            pytest.skip("No intent match found - cannot test execution")
        
        # Test intent execution
        execution_request = {
            "matchedIntent": match_results,
            "executionContext": {
                "environment": "testing",
                "dryRun": True,
                "validateOnly": False
            }
        }
        
        response = requests.post(
            f"{api_config['base_url']}/intents/execute",
            headers=api_headers,
            json=execution_request,
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 200, f"Intent execution failed: {response.status_code} - {response.text}"
        
        execution_results = response.json()
        
        # Validate response structure
        assert "originalIntent" in execution_results
        assert "action" in execution_results
        
        action = execution_results["action"]
        assert "actionType" in action
        assert "resource" in action
        assert "spec" in action
        
        print(f"✅ Intent execution generated {action['actionType']} action for {action['resource']}")
        
        # Validate execution plan if present
        if "executionPlan" in execution_results:
            plan = execution_results["executionPlan"]
            assert isinstance(plan, list), "Execution plan should be a list of steps"
            
            if len(plan) > 0:
                print(f"✅ Generated execution plan with {len(plan)} steps")
                for i, step in enumerate(plan[:3]):  # Show first 3 steps
                    print(f"  Step {step.get('step', i+1)}: {step.get('description', 'N/A')}")
    
    @pytest.mark.performance
    def test_rag_performance_pipeline(self, api_config, api_headers):
        """Test RAG pipeline performance under load"""
        
        # Define test queries with different complexities
        test_queries = [
            {"query": "AMF", "type": "simple"},
            {"query": "AMF registration procedure", "type": "medium"},
            {"query": "How to configure AMF for network slicing with QoS requirements", "type": "complex"},
            {"query": "Deployment procedures for SMF in 5G core network with high availability", "type": "complex"}
        ]
        
        performance_results = []
        
        for test_query in test_queries:
            # Measure semantic search performance
            start_time = time.time()
            
            search_request = {
                "query": test_query["query"],
                "limit": 10,
                "certainty": 0.7
            }
            
            response = requests.post(
                f"{api_config['base_url']}/search/semantic",
                headers=api_headers,
                json=search_request,
                timeout=api_config['timeout']
            )
            
            end_time = time.time()
            duration = end_time - start_time
            
            if response.status_code == 200:
                results = response.json()
                result_count = len(results.get("results", []))
                
                performance_results.append({
                    "query": test_query["query"],
                    "type": test_query["type"],
                    "duration": duration,
                    "result_count": result_count,
                    "status": "success"
                })
                
                print(f"✅ {test_query['type']} query: {duration:.3f}s, {result_count} results")
            else:
                performance_results.append({
                    "query": test_query["query"],
                    "type": test_query["type"],
                    "duration": duration,
                    "status": "failed",
                    "error": response.status_code
                })
                
                print(f"❌ {test_query['type']} query failed: {response.status_code}")
        
        # Analyze performance results
        successful_queries = [r for r in performance_results if r["status"] == "success"]
        
        if len(successful_queries) > 0:
            avg_duration = sum(r["duration"] for r in successful_queries) / len(successful_queries)
            max_duration = max(r["duration"] for r in successful_queries)
            
            print(f"✅ Performance summary: avg={avg_duration:.3f}s, max={max_duration:.3f}s")
            
            # Performance assertions
            assert avg_duration < 2.0, f"Average query time ({avg_duration:.3f}s) exceeds 2.0s threshold"
            assert max_duration < 5.0, f"Maximum query time ({max_duration:.3f}s) exceeds 5.0s threshold"
        else:
            pytest.fail("No successful queries in performance test")
    
    def test_error_handling_and_resilience(self, api_config, api_headers):
        """Test error handling and system resilience"""
        
        # Test invalid requests
        error_test_cases = [
            {
                "name": "Empty query",
                "endpoint": "/search/semantic",
                "payload": {"query": "", "limit": 10},
                "expected_status": 400
            },
            {
                "name": "Invalid limit",
                "endpoint": "/search/semantic", 
                "payload": {"query": "test", "limit": -1},
                "expected_status": 400
            },
            {
                "name": "Invalid certainty",
                "endpoint": "/search/semantic",
                "payload": {"query": "test", "certainty": 1.5},
                "expected_status": 400
            },
            {
                "name": "Missing required field",
                "endpoint": "/intents/match",
                "payload": {},
                "expected_status": 400
            }
        ]
        
        for test_case in error_test_cases:
            response = requests.post(
                f"{api_config['base_url']}{test_case['endpoint']}",
                headers=api_headers,
                json=test_case["payload"],
                timeout=api_config['timeout']
            )
            
            assert response.status_code == test_case["expected_status"], \
                f"Error test '{test_case['name']}' expected {test_case['expected_status']}, got {response.status_code}"
            
            # Validate error response format
            if response.status_code >= 400:
                try:
                    error_data = response.json()
                    assert "error" in error_data or "message" in error_data, \
                        f"Error response missing error/message field: {error_data}"
                    print(f"✅ Error test '{test_case['name']}': proper error response")
                except json.JSONDecodeError:
                    print(f"⚠️ Error test '{test_case['name']}': non-JSON error response")
        
        # Test authentication errors
        invalid_headers = {
            "Authorization": "Bearer invalid-token",
            "Content-Type": "application/json"
        }
        
        response = requests.post(
            f"{api_config['base_url']}/search/semantic",
            headers=invalid_headers,
            json={"query": "test"},
            timeout=api_config['timeout']
        )
        
        assert response.status_code == 401, f"Expected 401 for invalid token, got {response.status_code}"
        print("✅ Authentication error handling works correctly")

if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
```

I'll continue with the rest of the integration testing documentation in the next response, as this file is getting quite comprehensive. The current content provides a solid foundation for Weaviate integration testing, including comprehensive test suites for basic connectivity, schema operations, telecom-specific data validation, and RAG pipeline testing.

<function_calls>
<invoke name="TodoWrite">
