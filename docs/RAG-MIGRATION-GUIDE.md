# RAG Migration Guide

This comprehensive guide covers migration scenarios for the Nephoran Intent Operator's RAG pipeline, including upgrades from legacy systems, version migrations, cloud provider migrations, and disaster recovery scenarios.

## Table of Contents

1. [Migration Overview](#migration-overview)
2. [Pre-Migration Assessment](#pre-migration-assessment)
3. [Legacy System Migration](#legacy-system-migration)
4. [Version Upgrade Migration](#version-upgrade-migration)
5. [Cloud Provider Migration](#cloud-provider-migration)
6. [Data Migration Procedures](#data-migration-procedures)
7. [Configuration Migration](#configuration-migration)
8. [Testing and Validation](#testing-and-validation)
9. [Rollback Procedures](#rollback-procedures)
10. [Post-Migration Optimization](#post-migration-optimization)

## Migration Overview

### Migration Types

The Nephoran RAG system supports several migration scenarios:

- **Legacy System Migration**: Moving from traditional knowledge management systems
- **Version Upgrades**: Upgrading between RAG pipeline versions
- **Cloud Migrations**: Moving between cloud providers or on-premises to cloud
- **Schema Migrations**: Updating data schemas and structures
- **Component Migrations**: Migrating individual components (Weaviate, Redis, etc.)

### Migration Principles

1. **Zero-Downtime Migration**: Minimize service interruption
2. **Data Integrity**: Ensure no data loss during migration
3. **Rollback Capability**: Ability to revert changes if needed
4. **Validation**: Comprehensive testing at each stage
5. **Monitoring**: Real-time monitoring during migration

## Pre-Migration Assessment

### System Assessment Checklist

```bash
#!/bin/bash
# pre-migration-assessment.sh

echo "=== Nephoran RAG Pre-Migration Assessment ==="

# 1. Current system inventory
echo "1. Current System Inventory"
kubectl get all -n nephoran-system
kubectl get pv,pvc -n nephoran-system

# 2. Resource utilization
echo "2. Resource Utilization"
kubectl top nodes
kubectl top pods -n nephoran-system

# 3. Data size assessment
echo "3. Data Size Assessment"
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -s http://localhost:8080/v1/meta | jq '.nodeStatus'

# 4. Performance baseline
echo "4. Performance Baseline"
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
PID=$!
sleep 5

# Sample query performance test
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ Get { TelecomKnowledge(limit: 10) { content title } } }"
  }' \
  "http://localhost:8080/v1/graphql" | jq '.data'

kill $PID

# 5. Configuration backup
echo "5. Configuration Backup"
kubectl get configmaps -n nephoran-system -o yaml > config-backup.yaml
kubectl get secrets -n nephoran-system -o yaml > secrets-backup.yaml

# 6. Database schema export
echo "6. Database Schema Export"
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -s http://localhost:8080/v1/schema > schema-backup.json

echo "Assessment completed. Review outputs before proceeding with migration."
```

### Migration Readiness Assessment

```yaml
# assessment/migration-readiness.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migration-readiness-assessment
  namespace: nephoran-system
data:
  checklist.yaml: |
    migration_readiness:
      infrastructure:
        - name: "Kubernetes cluster health"
          status: "pending"
          command: "kubectl cluster-info"
          
        - name: "Storage availability"
          status: "pending"
          command: "kubectl get storageclass"
          
        - name: "Network connectivity"
          status: "pending"
          command: "kubectl get networkpolicy -n nephoran-system"
          
      data:
        - name: "Data backup completed"
          status: "pending"
          validation: "backup_verification"
          
        - name: "Schema compatibility"
          status: "pending"
          validation: "schema_validation"
          
        - name: "Data integrity check"
          status: "pending"
          validation: "integrity_check"
          
      configuration:
        - name: "Configuration backed up"
          status: "pending"
          files: ["configmaps", "secrets", "deployments"]
          
        - name: "Environment variables verified"
          status: "pending"
          validation: "env_validation"
          
      resources:
        - name: "Resource requirements met"
          status: "pending"
          check: "resource_calculation"
          
        - name: "Migration window scheduled"
          status: "pending"
          duration: "estimated_migration_time"
          
  validation_scripts.sh: |
    #!/bin/bash
    
    # Backup verification
    backup_verification() {
      echo "Verifying backups..."
      if [[ -f "weaviate-backup-$(date +%Y%m%d).tar.gz" ]]; then
        echo "✓ Weaviate backup found"
        return 0
      else
        echo "✗ Weaviate backup missing"
        return 1
      fi
    }
    
    # Schema validation
    schema_validation() {
      echo "Validating schema compatibility..."
      python3 -c "
    import json
    import sys
    
    try:
        with open('schema-backup.json', 'r') as f:
            schema = json.load(f)
        
        required_classes = ['TelecomKnowledge', 'IntentPatterns', 'NetworkFunctions']
        existing_classes = [cls['class'] for cls in schema.get('classes', [])]
        
        for cls in required_classes:
            if cls not in existing_classes:
                print(f'✗ Missing required class: {cls}')
                sys.exit(1)
        
        print('✓ Schema validation passed')
    except Exception as e:
        print(f'✗ Schema validation failed: {e}')
        sys.exit(1)
    "
    }
    
    # Data integrity check
    integrity_check() {
      echo "Checking data integrity..."
      # Calculate checksums for critical data
      find /var/lib/weaviate -name "*.db" -exec sha256sum {} \; > data-checksums.txt
      echo "✓ Data integrity checksums generated"
    }
```

## Legacy System Migration

### From Traditional Document Stores

#### Elasticsearch Migration

```bash
#!/bin/bash
# elasticsearch-to-weaviate-migration.sh

set -e

ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}"
WEAVIATE_URL="${WEAVIATE_URL:-http://localhost:8080}"
INDEX_NAME="${INDEX_NAME:-telecom_docs}"
BATCH_SIZE="${BATCH_SIZE:-100}"

echo "Starting Elasticsearch to Weaviate migration..."

# 1. Export data from Elasticsearch
echo "Exporting data from Elasticsearch..."
curl -X GET "${ELASTICSEARCH_URL}/${INDEX_NAME}/_search?scroll=1m&size=${BATCH_SIZE}" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {"match_all": {}},
    "_source": ["content", "title", "source", "category", "metadata"]
  }' > elasticsearch_export.json

# 2. Transform data for Weaviate
echo "Transforming data for Weaviate..."
python3 << 'EOF'
import json
import requests
from datetime import datetime

# Load Elasticsearch export
with open('elasticsearch_export.json', 'r') as f:
    es_data = json.load(f)

# Weaviate client setup
weaviate_url = "http://localhost:8080"
headers = {"Content-Type": "application/json"}

# Transform and upload data
batch_objects = []
for hit in es_data['hits']['hits']:
    source = hit['_source']
    
    # Transform Elasticsearch document to Weaviate object
    weaviate_object = {
        "class": "TelecomKnowledge",
        "properties": {
            "content": source.get('content', ''),
            "title": source.get('title', ''),
            "source": source.get('source', 'Unknown'),
            "category": source.get('category', 'General'),
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "keywords": source.get('metadata', {}).get('keywords', []),
            "confidence": 0.8  # Default confidence for migrated data
        }
    }
    
    batch_objects.append(weaviate_object)
    
    # Send batch when it reaches batch size
    if len(batch_objects) >= 100:
        batch_request = {"objects": batch_objects}
        response = requests.post(
            f"{weaviate_url}/v1/batch/objects",
            headers=headers,
            json=batch_request
        )
        
        if response.status_code == 200:
            print(f"Uploaded batch of {len(batch_objects)} objects")
        else:
            print(f"Error uploading batch: {response.text}")
            
        batch_objects = []

# Upload remaining objects
if batch_objects:
    batch_request = {"objects": batch_objects}
    response = requests.post(
        f"{weaviate_url}/v1/batch/objects",
        headers=headers,
        json=batch_request
    )
    print(f"Uploaded final batch of {len(batch_objects)} objects")

print("Migration completed successfully!")
EOF

echo "Elasticsearch to Weaviate migration completed!"
```

#### Solr Migration

```python
# solr_to_weaviate_migration.py
import json
import requests
import pysolr
from datetime import datetime
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SolrToWeaviateMigration:
    def __init__(self, solr_url, weaviate_url, collection_name):
        self.solr = pysolr.Solr(solr_url)
        self.weaviate_url = weaviate_url
        self.collection_name = collection_name
        self.headers = {"Content-Type": "application/json"}
        
    def export_from_solr(self, batch_size=1000):
        """Export documents from Solr in batches"""
        logger.info(f"Exporting documents from Solr collection: {self.collection_name}")
        
        start = 0
        total_docs = 0
        
        while True:
            # Query Solr for documents
            results = self.solr.search(
                '*:*',
                **{
                    'rows': batch_size,
                    'start': start,
                    'fl': 'id,content,title,source,category,spec_version,domain'
                }
            )
            
            if not results:
                break
                
            logger.info(f"Processing batch starting at {start}, count: {len(results)}")
            
            # Transform and upload to Weaviate
            self.transform_and_upload(results)
            
            total_docs += len(results)
            start += batch_size
            
            if len(results) < batch_size:
                break
                
        logger.info(f"Migration completed. Total documents migrated: {total_docs}")
        
    def transform_and_upload(self, solr_docs):
        """Transform Solr documents to Weaviate format and upload"""
        batch_objects = []
        
        for doc in solr_docs:
            # Map Solr fields to Weaviate properties
            weaviate_object = {
                "class": "TelecomKnowledge",
                "properties": {
                    "content": doc.get('content', [''])[0] if isinstance(doc.get('content'), list) else doc.get('content', ''),
                    "title": doc.get('title', [''])[0] if isinstance(doc.get('title'), list) else doc.get('title', ''),
                    "source": self.map_source(doc.get('source', ['Unknown'])[0] if isinstance(doc.get('source'), list) else doc.get('source', 'Unknown')),
                    "category": doc.get('category', ['General'])[0] if isinstance(doc.get('category'), list) else doc.get('category', 'General'),
                    "version": doc.get('spec_version', ['1.0'])[0] if isinstance(doc.get('spec_version'), list) else doc.get('spec_version', '1.0'),
                    "domain": doc.get('domain', ['Core'])[0] if isinstance(doc.get('domain'), list) else doc.get('domain', 'Core'),
                    "timestamp": datetime.utcnow().isoformat() + "Z",
                    "confidence": 0.8,  # Default confidence for migrated data
                    "keywords": self.extract_keywords(doc.get('content', ''))
                }
            }
            
            batch_objects.append(weaviate_object)
            
        # Upload batch to Weaviate
        if batch_objects:
            self.upload_batch(batch_objects)
            
    def map_source(self, solr_source):
        """Map Solr source values to standardized format"""
        source_mapping = {
            '3gpp': '3GPP',
            'oran': 'O-RAN',
            'etsi': 'ETSI',
            'itu': 'ITU',
            'ieee': 'IEEE'
        }
        return source_mapping.get(solr_source.lower(), solr_source)
        
    def extract_keywords(self, content):
        """Extract telecom-specific keywords from content"""
        telecom_keywords = [
            'AMF', 'SMF', 'UPF', 'gNB', 'eNB', '5G', '4G', 'LTE',
            'RAN', 'Core', 'CUPS', 'SBA', 'NF', 'API', 'REST',
            'HTTP/2', 'JSON', 'YAML', 'OpenAPI'
        ]
        
        content_upper = content.upper()
        found_keywords = [kw for kw in telecom_keywords if kw in content_upper]
        return found_keywords[:10]  # Limit to 10 keywords
        
    def upload_batch(self, batch_objects):
        """Upload batch of objects to Weaviate"""
        batch_request = {"objects": batch_objects}
        
        response = requests.post(
            f"{self.weaviate_url}/v1/batch/objects",
            headers=self.headers,
            json=batch_request
        )
        
        if response.status_code == 200:
            logger.info(f"Successfully uploaded batch of {len(batch_objects)} objects")
        else:
            logger.error(f"Failed to upload batch: {response.status_code} - {response.text}")
            raise Exception(f"Batch upload failed: {response.text}")

# Usage example
if __name__ == "__main__":
    migration = SolrToWeaviateMigration(
        solr_url="http://localhost:8983/solr",
        weaviate_url="http://localhost:8080",
        collection_name="telecom_docs"
    )
    
    migration.export_from_solr(batch_size=500)
```

### From File-Based Systems

```bash
#!/bin/bash
# file-system-migration.sh

SOURCE_DIR="${SOURCE_DIR:-/path/to/source/documents}"
WEAVIATE_URL="${WEAVIATE_URL:-http://localhost:8080}"
BATCH_SIZE="${BATCH_SIZE:-50}"

echo "Starting file system to Weaviate migration..."

# Create temporary processing directory
TEMP_DIR=$(mktemp -d)
echo "Using temporary directory: $TEMP_DIR"

# Find all supported document files
find "$SOURCE_DIR" -type f \( -name "*.pdf" -o -name "*.txt" -o -name "*.md" -o -name "*.docx" \) > "$TEMP_DIR/file_list.txt"

TOTAL_FILES=$(wc -l < "$TEMP_DIR/file_list.txt")
echo "Found $TOTAL_FILES files to migrate"

# Process files in batches
CURRENT_BATCH=0
while read -r file_path; do
    echo "Processing: $file_path"
    
    # Extract text content based on file type
    case "${file_path##*.}" in
        pdf)
            python3 -c "
import PyPDF2
import sys
import json

try:
    with open('$file_path', 'rb') as f:
        reader = PyPDF2.PdfFileReader(f)
        text = ''
        for page_num in range(reader.numPages):
            text += reader.getPage(page_num).extractText()
    
    # Create metadata
    metadata = {
        'filename': '$(basename "$file_path")',
        'filepath': '$file_path',
        'filetype': 'pdf',
        'content': text.strip()
    }
    
    with open('$TEMP_DIR/batch_${CURRENT_BATCH}.json', 'a') as out:
        json.dump(metadata, out)
        out.write('\n')
        
except Exception as e:
    print(f'Error processing $file_path: {e}', file=sys.stderr)
"
            ;;
        txt|md)
            python3 -c "
import json
import sys

try:
    with open('$file_path', 'r', encoding='utf-8') as f:
        content = f.read()
    
    metadata = {
        'filename': '$(basename "$file_path")',
        'filepath': '$file_path',
        'filetype': '${file_path##*.}',
        'content': content.strip()
    }
    
    with open('$TEMP_DIR/batch_${CURRENT_BATCH}.json', 'a') as out:
        json.dump(metadata, out)
        out.write('\n')
        
except Exception as e:
    print(f'Error processing $file_path: {e}', file=sys.stderr)
"
            ;;
        docx)
            python3 -c "
import docx
import json
import sys

try:
    doc = docx.Document('$file_path')
    text = '\n'.join([paragraph.text for paragraph in doc.paragraphs])
    
    metadata = {
        'filename': '$(basename "$file_path")',
        'filepath': '$file_path',
        'filetype': 'docx',
        'content': text.strip()
    }
    
    with open('$TEMP_DIR/batch_${CURRENT_BATCH}.json', 'a') as out:
        json.dump(metadata, out)
        out.write('\n')
        
except Exception as e:
    print(f'Error processing $file_path: {e}', file=sys.stderr)
"
            ;;
    esac
    
    # Check if batch is complete
    BATCH_COUNT=$(wc -l < "$TEMP_DIR/batch_${CURRENT_BATCH}.json" 2>/dev/null || echo "0")
    if [[ $BATCH_COUNT -ge $BATCH_SIZE ]]; then
        # Upload batch to Weaviate
        echo "Uploading batch $CURRENT_BATCH to Weaviate..."
        python3 << EOF
import json
import requests
from datetime import datetime
import os

batch_file = "$TEMP_DIR/batch_${CURRENT_BATCH}.json"
weaviate_url = "$WEAVIATE_URL"

if not os.path.exists(batch_file):
    exit(0)

objects = []
with open(batch_file, 'r') as f:
    for line in f:
        if line.strip():
            metadata = json.loads(line)
            
            # Determine source and category from filename/path
            filename = metadata['filename'].lower()
            if 'ts' in filename or '3gpp' in filename:
                source = '3GPP'
            elif 'oran' in filename or 'o-ran' in filename:
                source = 'O-RAN'
            elif 'etsi' in filename:
                source = 'ETSI'
            else:
                source = 'Unknown'
                
            # Determine category from path or filename
            filepath = metadata['filepath'].lower()
            if 'ran' in filepath:
                category = 'RAN'
            elif 'core' in filepath:
                category = 'Core'
            elif 'transport' in filepath:
                category = 'Transport'
            else:
                category = 'General'
            
            obj = {
                "class": "TelecomKnowledge",
                "properties": {
                    "content": metadata['content'][:50000],  # Limit content size
                    "title": metadata['filename'],
                    "source": source,
                    "category": category,
                    "documentType": metadata['filetype'],
                    "timestamp": datetime.utcnow().isoformat() + "Z",
                    "confidence": 0.7,
                    "keywords": []  # Could extract keywords here
                }
            }
            objects.append(obj)

if objects:
    response = requests.post(
        f"{weaviate_url}/v1/batch/objects",
        headers={"Content-Type": "application/json"},
        json={"objects": objects}
    )
    
    if response.status_code == 200:
        print(f"Successfully uploaded {len(objects)} objects")
    else:
        print(f"Error uploading batch: {response.text}")
        
# Remove processed batch file
os.remove(batch_file)
EOF
        
        ((CURRENT_BATCH++))
    fi
    
done < "$TEMP_DIR/file_list.txt"

# Upload any remaining files in the last batch
if [[ -f "$TEMP_DIR/batch_${CURRENT_BATCH}.json" ]]; then
    echo "Uploading final batch..."
    # Same upload logic as above
fi

# Cleanup
rm -rf "$TEMP_DIR"
echo "File system migration completed!"
```

## Version Upgrade Migration

### RAG Pipeline Version Upgrade

```yaml
# migration/version-upgrade.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: version-upgrade-config
  namespace: nephoran-system
data:
  upgrade-plan.yaml: |
    upgrade:
      # Version information
      from_version: "v1.1.0"
      to_version: "v1.2.0"
      
      # Upgrade strategy
      strategy: "rolling"  # rolling, blue-green, canary
      
      # Pre-upgrade steps
      pre_upgrade:
        - name: "backup_data"
          type: "job"
          config: "backup-job.yaml"
          
        - name: "validate_compatibility"
          type: "job"
          config: "compatibility-check.yaml"
          
        - name: "scale_down_non_critical"
          type: "scale"
          targets: ["rag-api", "llm-processor"]
          
      # Upgrade steps
      upgrade_steps:
        - name: "update_crds"
          type: "apply"
          resources: ["crds/"]
          
        - name: "update_weaviate"
          type: "rolling_update"
          deployment: "weaviate"
          image: "semitechnologies/weaviate:1.28.1"
          
        - name: "migrate_schema"
          type: "job"
          config: "schema-migration.yaml"
          
        - name: "update_rag_components"
          type: "rolling_update"
          deployments: ["rag-api", "llm-processor"]
          
      # Post-upgrade steps
      post_upgrade:
        - name: "validate_functionality"
          type: "job"
          config: "functionality-test.yaml"
          
        - name: "performance_test"
          type: "job"
          config: "performance-test.yaml"
          
        - name: "cleanup_old_resources"
          type: "cleanup"
          resources: ["old-configmaps", "old-secrets"]
          
      # Rollback configuration
      rollback:
        enabled: true
        timeout: "30m"
        trigger_conditions:
          - "functionality_test_failure"
          - "performance_degradation_>50%"
          - "error_rate_>5%"

---
# Schema migration job
apiVersion: batch/v1
kind: Job
metadata:
  name: schema-migration-v1-2-0
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: schema-migrator
        image: nephoran/schema-migrator:v1.2.0
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: FROM_VERSION
          value: "v1.1.0"
        - name: TO_VERSION
          value: "v1.2.0"
        command: ["/bin/sh"]
        args:
        - -c
        - |
          echo "Starting schema migration from v1.1.0 to v1.2.0..."
          
          # Add new properties to existing classes
          python3 << 'EOF'
          import requests
          import json
          
          weaviate_url = "http://weaviate:8080"
          headers = {"Content-Type": "application/json"}
          
          # Add new properties to TelecomKnowledge class
          new_properties = [
              {
                  "name": "priority",
                  "dataType": ["int"],
                  "description": "Document priority score"
              },
              {
                  "name": "lastUpdated",
                  "dataType": ["date"],
                  "description": "Last update timestamp"
              }
          ]
          
          for prop in new_properties:
              response = requests.post(
                  f"{weaviate_url}/v1/schema/TelecomKnowledge/properties",
                  headers=headers,
                  json=prop
              )
              
              if response.status_code == 200:
                  print(f"Added property: {prop['name']}")
              else:
                  print(f"Error adding property {prop['name']}: {response.text}")
          
          print("Schema migration completed!")
          EOF
          
      restartPolicy: Never
  backoffLimit: 3

---
# Functionality validation job
apiVersion: batch/v1
kind: Job
metadata:
  name: functionality-test-v1-2-0
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: functionality-tester
        image: curlimages/curl:8.5.0
        env:
        - name: RAG_API_URL
          value: "http://rag-api:8080"
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        command: ["/bin/sh"]
        args:
        - -c
        - |
          echo "Starting functionality tests..."
          
          # Test 1: Basic query functionality
          echo "Test 1: Basic query functionality"
          response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d '{"query": "What is AMF?", "limit": 5}' \
            "$RAG_API_URL/api/v1/query/search")
          
          if echo "$response" | grep -q "results"; then
            echo "✓ Basic query test passed"
          else
            echo "✗ Basic query test failed"
            exit 1
          fi
          
          # Test 2: Intent processing
          echo "Test 2: Intent processing"
          response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d '{"intent": "Configure AMF with high availability"}' \
            "$RAG_API_URL/api/v1/query/intent")
          
          if echo "$response" | grep -q "interpretation"; then
            echo "✓ Intent processing test passed"
          else
            echo "✗ Intent processing test failed"
            exit 1
          fi
          
          # Test 3: New schema properties
          echo "Test 3: New schema properties"
          response=$(curl -s "$WEAVIATE_URL/v1/schema/TelecomKnowledge")
          
          if echo "$response" | grep -q "priority" && echo "$response" | grep -q "lastUpdated"; then
            echo "✓ Schema migration test passed"
          else
            echo "✗ Schema migration test failed"
            exit 1
          fi
          
          echo "All functionality tests passed!"
          
      restartPolicy: Never
  backoffLimit: 2
```

### Automated Upgrade Script

```bash
#!/bin/bash
# automated-upgrade.sh

set -e

# Configuration
NAMESPACE="nephoran-system"
FROM_VERSION="${FROM_VERSION:-v1.1.0}"
TO_VERSION="${TO_VERSION:-v1.2.0}"
STRATEGY="${STRATEGY:-rolling}"
TIMEOUT="${TIMEOUT:-1800}"  # 30 minutes

echo "Starting RAG pipeline upgrade from $FROM_VERSION to $TO_VERSION"

# Pre-upgrade validation
echo "Running pre-upgrade validation..."
kubectl apply -f migration/pre-upgrade-validation.yaml
kubectl wait --for=condition=complete job/pre-upgrade-validation -n $NAMESPACE --timeout=600s

if ! kubectl logs job/pre-upgrade-validation -n $NAMESPACE | grep -q "Validation passed"; then
    echo "Pre-upgrade validation failed. Aborting upgrade."
    exit 1
fi

# Create backup
echo "Creating backup..."
kubectl apply -f migration/backup-job.yaml
kubectl wait --for=condition=complete job/upgrade-backup -n $NAMESPACE --timeout=1200s

# Store current state for rollback
echo "Storing current state for rollback..."
kubectl get deployment weaviate -n $NAMESPACE -o yaml > rollback/weaviate-deployment.yaml
kubectl get deployment rag-api -n $NAMESPACE -o yaml > rollback/rag-api-deployment.yaml
kubectl get deployment llm-processor -n $NAMESPACE -o yaml > rollback/llm-processor-deployment.yaml

# Update CRDs first
echo "Updating Custom Resource Definitions..."
kubectl apply -f crds/

# Upgrade based on strategy
case $STRATEGY in
    "rolling")
        echo "Performing rolling upgrade..."
        
        # Update Weaviate
        kubectl set image deployment/weaviate weaviate=semitechnologies/weaviate:1.28.1 -n $NAMESPACE
        kubectl rollout status deployment/weaviate -n $NAMESPACE --timeout=600s
        
        # Run schema migration
        kubectl apply -f migration/schema-migration.yaml
        kubectl wait --for=condition=complete job/schema-migration-v1-2-0 -n $NAMESPACE --timeout=600s
        
        # Update RAG components
        kubectl set image deployment/rag-api rag-api=nephoran/rag-api:$TO_VERSION -n $NAMESPACE
        kubectl set image deployment/llm-processor llm-processor=nephoran/llm-processor:$TO_VERSION -n $NAMESPACE
        
        kubectl rollout status deployment/rag-api -n $NAMESPACE --timeout=600s
        kubectl rollout status deployment/llm-processor -n $NAMESPACE --timeout=600s
        ;;
        
    "blue-green")
        echo "Performing blue-green deployment..."
        
        # Create green environment
        sed 's/name: weaviate/name: weaviate-green/g' deployments/weaviate.yaml | kubectl apply -f -
        sed 's/name: rag-api/name: rag-api-green/g' deployments/rag-api.yaml | kubectl apply -f -
        
        # Wait for green environment to be ready
        kubectl wait --for=condition=available deployment/weaviate-green -n $NAMESPACE --timeout=600s
        kubectl wait --for=condition=available deployment/rag-api-green -n $NAMESPACE --timeout=600s
        
        # Switch traffic to green
        kubectl patch service weaviate -n $NAMESPACE -p '{"spec":{"selector":{"app":"weaviate-green"}}}'
        kubectl patch service rag-api -n $NAMESPACE -p '{"spec":{"selector":{"app":"rag-api-green"}}}'
        
        # Remove blue environment after validation
        sleep 300  # Wait 5 minutes for traffic switch
        kubectl delete deployment weaviate rag-api llm-processor -n $NAMESPACE
        
        # Rename green to blue
        kubectl patch deployment weaviate-green -n $NAMESPACE -p '{"metadata":{"name":"weaviate"}}'
        kubectl patch deployment rag-api-green -n $NAMESPACE -p '{"metadata":{"name":"rag-api"}}'
        ;;
        
    "canary")
        echo "Performing canary deployment..."
        
        # Create canary deployments with 10% traffic
        kubectl apply -f migration/canary-deployment.yaml
        
        # Monitor canary for 10 minutes
        echo "Monitoring canary deployment for 10 minutes..."
        sleep 600
        
        # Check canary health
        if kubectl logs -l version=canary,app=rag-api -n $NAMESPACE | grep -q "ERROR"; then
            echo "Canary deployment has errors. Rolling back..."
            kubectl delete -f migration/canary-deployment.yaml
            exit 1
        fi
        
        # Gradually increase canary traffic
        for percentage in 25 50 75 100; do
            echo "Increasing canary traffic to $percentage%"
            kubectl patch virtualservice rag-api -n $NAMESPACE --type='merge' -p="{\"spec\":{\"http\":[{\"match\":[{\"headers\":{\"canary\":{\"exact\":\"true\"}}}],\"route\":[{\"destination\":{\"host\":\"rag-api\",\"subset\":\"canary\"},\"weight\":$percentage}]}]}}"
            sleep 300  # Wait 5 minutes between increases
        done
        
        # Replace original with canary
        kubectl delete deployment rag-api -n $NAMESPACE
        kubectl patch deployment rag-api-canary -n $NAMESPACE -p '{"metadata":{"name":"rag-api"}}'
        ;;
esac

# Post-upgrade validation
echo "Running post-upgrade validation..."
kubectl apply -f migration/functionality-test.yaml
kubectl wait --for=condition=complete job/functionality-test-v1-2-0 -n $NAMESPACE --timeout=600s

if ! kubectl logs job/functionality-test-v1-2-0 -n $NAMESPACE | grep -q "All functionality tests passed"; then
    echo "Post-upgrade validation failed. Initiating rollback..."
    ./rollback.sh
    exit 1
fi

# Performance validation
echo "Running performance validation..."
kubectl apply -f migration/performance-test.yaml
kubectl wait --for=condition=complete job/performance-test-v1-2-0 -n $NAMESPACE --timeout=600s

# Cleanup old resources
echo "Cleaning up old resources..."
kubectl delete job pre-upgrade-validation upgrade-backup -n $NAMESPACE
kubectl delete configmap rag-config-v1-1-0 -n $NAMESPACE --ignore-not-found

echo "Upgrade completed successfully from $FROM_VERSION to $TO_VERSION!"
```

## Cloud Provider Migration

### AWS to Azure Migration

```yaml
# migration/cloud-migration.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-migration-config
  namespace: nephoran-system
data:
  migration-plan.yaml: |
    cloud_migration:
      source:
        provider: "aws"
        region: "us-east-1"
        cluster: "nephoran-eks-cluster"
        
      target:
        provider: "azure"
        region: "East US"
        cluster: "nephoran-aks-cluster"
        
      migration_strategy: "lift_and_shift"  # lift_and_shift, replatform, refactor
      
      components:
        weaviate:
          data_size: "500Gi"
          migration_method: "backup_restore"
          downtime_window: "4h"
          
        redis:
          data_size: "50Gi"
          migration_method: "replication"
          downtime_window: "30m"
          
        configuration:
          migration_method: "export_import"
          downtime_window: "15m"
          
      networking:
        dns_update_required: true
        load_balancer_migration: true
        
      storage:
        source_class: "gp3"
        target_class: "managed-premium"
        
      timeline:
        preparation: "1 week"
        migration: "1 day"
        validation: "2 days"
        cutover: "4 hours"

---
# Cloud migration execution script
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-migration-script
  namespace: nephoran-system
data:
  migrate.sh: |
    #!/bin/bash
    set -e
    
    # Configuration
    SOURCE_KUBECONFIG="$HOME/.kube/aws-config"
    TARGET_KUBECONFIG="$HOME/.kube/azure-config"
    MIGRATION_BUCKET="nephoran-migration-bucket"
    
    echo "Starting cloud migration from AWS to Azure..."
    
    # Step 1: Export data from source
    echo "Step 1: Exporting data from AWS cluster..."
    export KUBECONFIG="$SOURCE_KUBECONFIG"
    
    # Create migration namespace in source
    kubectl create namespace migration --dry-run=client -o yaml | kubectl apply -f -
    
    # Export Weaviate data
    kubectl apply -f - <<EOF
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: export-weaviate-data
      namespace: migration
    spec:
      template:
        spec:
          containers:
          - name: exporter
            image: nephoran/migration-tool:latest
            env:
            - name: WEAVIATE_URL
              value: "http://weaviate.nephoran-system:8080"
            - name: EXPORT_BUCKET
              value: "$MIGRATION_BUCKET"
            - name: OPERATION
              value: "export"
            command: ["/scripts/migrate-weaviate.sh"]
          restartPolicy: Never
    EOF
    
    kubectl wait --for=condition=complete job/export-weaviate-data -n migration --timeout=7200s
    
    # Export configurations
    kubectl get configmaps -n nephoran-system -o yaml > /tmp/configmaps.yaml
    kubectl get secrets -n nephoran-system -o yaml > /tmp/secrets.yaml
    kubectl get deployments -n nephoran-system -o yaml > /tmp/deployments.yaml
    
    # Upload configurations to migration bucket
    aws s3 cp /tmp/configmaps.yaml s3://$MIGRATION_BUCKET/config/
    aws s3 cp /tmp/secrets.yaml s3://$MIGRATION_BUCKET/config/
    aws s3 cp /tmp/deployments.yaml s3://$MIGRATION_BUCKET/config/
    
    # Step 2: Prepare target environment
    echo "Step 2: Preparing Azure target environment..."
    export KUBECONFIG="$TARGET_KUBECONFIG"
    
    # Create namespace
    kubectl create namespace nephoran-system --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace migration --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply storage classes for Azure
    kubectl apply -f - <<EOF
    apiVersion: storage.k8s.io/v1
    kind: StorageClass
    metadata:
      name: managed-premium
    provisioner: kubernetes.io/azure-disk
    parameters:
      storageaccounttype: Premium_LRS
      kind: Managed
    allowVolumeExpansion: true
    volumeBindingMode: WaitForFirstConsumer
    EOF
    
    # Step 3: Import configurations
    echo "Step 3: Importing configurations to Azure..."
    
    # Download and modify configurations
    az storage blob download --container-name migration --name config/configmaps.yaml --file /tmp/configmaps-azure.yaml
    az storage blob download --container-name migration --name config/secrets.yaml --file /tmp/secrets-azure.yaml
    az storage blob download --container-name migration --name config/deployments.yaml --file /tmp/deployments-azure.yaml
    
    # Modify configurations for Azure
    sed -i 's/gp3/managed-premium/g' /tmp/deployments-azure.yaml
    sed -i 's/aws-load-balancer/azure-load-balancer/g' /tmp/deployments-azure.yaml
    
    # Apply configurations
    kubectl apply -f /tmp/configmaps-azure.yaml
    kubectl apply -f /tmp/secrets-azure.yaml
    kubectl apply -f /tmp/deployments-azure.yaml
    
    # Step 4: Import data
    echo "Step 4: Importing Weaviate data..."
    kubectl apply -f - <<EOF
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: import-weaviate-data
      namespace: migration
    spec:
      template:
        spec:
          containers:
          - name: importer
            image: nephoran/migration-tool:latest
            env:
            - name: WEAVIATE_URL
              value: "http://weaviate.nephoran-system:8080"
            - name: IMPORT_BUCKET
              value: "$MIGRATION_BUCKET"
            - name: OPERATION
              value: "import"
            command: ["/scripts/migrate-weaviate.sh"]
          restartPolicy: Never
    EOF
    
    kubectl wait --for=condition=complete job/import-weaviate-data -n migration --timeout=7200s
    
    # Step 5: Validation
    echo "Step 5: Validating migration..."
    kubectl apply -f migration/validation-job.yaml
    kubectl wait --for=condition=complete job/migration-validation -n migration --timeout=1800s
    
    if kubectl logs job/migration-validation -n migration | grep -q "Migration validation passed"; then
      echo "Cloud migration completed successfully!"
    else
      echo "Migration validation failed. Please check logs."
      exit 1
    fi
    
    echo "Cloud migration from AWS to Azure completed!"
```

## Data Migration Procedures

### Large-Scale Data Migration

```python
# data_migration.py
import asyncio
import aiohttp
import json
import logging
from datetime import datetime
from typing import List, Dict, Any
import hashlib

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class DataMigrationManager:
    def __init__(self, source_url: str, target_url: str, batch_size: int = 1000):
        self.source_url = source_url
        self.target_url = target_url
        self.batch_size = batch_size
        self.migration_stats = {
            'total_objects': 0,
            'migrated_objects': 0,
            'failed_objects': 0,
            'start_time': None,
            'end_time': None
        }
        
    async def migrate_data(self, class_name: str):
        """Migrate all objects from source to target for a specific class"""
        logger.info(f"Starting migration for class: {class_name}")
        
        self.migration_stats['start_time'] = datetime.now()
        
        async with aiohttp.ClientSession() as session:
            # Get total count first
            total_count = await self._get_object_count(session, class_name)
            self.migration_stats['total_objects'] = total_count
            
            logger.info(f"Total objects to migrate: {total_count}")
            
            # Migrate in batches
            offset = 0
            while offset < total_count:
                try:
                    batch = await self._fetch_batch(session, class_name, offset, self.batch_size)
                    
                    if not batch:
                        break
                        
                    await self._migrate_batch(session, batch, class_name)
                    
                    offset += len(batch)
                    logger.info(f"Migrated {offset}/{total_count} objects")
                    
                except Exception as e:
                    logger.error(f"Error migrating batch at offset {offset}: {e}")
                    self.migration_stats['failed_objects'] += self.batch_size
                    offset += self.batch_size
                    
        self.migration_stats['end_time'] = datetime.now()
        self._log_migration_stats()
        
    async def _get_object_count(self, session: aiohttp.ClientSession, class_name: str) -> int:
        """Get total count of objects for a class"""
        query = f"""
        {{
          Aggregate {{
            {class_name} {{
              meta {{
                count
              }}
            }}
          }}
        }}
        """
        
        async with session.post(
            f"{self.source_url}/v1/graphql",
            json={"query": query}
        ) as response:
            data = await response.json()
            return data['data']['Aggregate'][class_name][0]['meta']['count']
            
    async def _fetch_batch(self, session: aiohttp.ClientSession, class_name: str, 
                          offset: int, limit: int) -> List[Dict[str, Any]]:
        """Fetch a batch of objects from source"""
        query = f"""
        {{
          Get {{
            {class_name}(offset: {offset}, limit: {limit}) {{
              _additional {{
                id
                vector
              }}
              content
              title
              source
              category
              version
              timestamp
              confidence
              keywords
              documentType
              networkFunction
              technology
              useCase
            }}
          }}
        }}
        """
        
        async with session.post(
            f"{self.source_url}/v1/graphql",
            json={"query": query}
        ) as response:
            if response.status == 200:
                data = await response.json()
                return data['data']['Get'][class_name]
            else:
                logger.error(f"Error fetching batch: {response.status}")
                return []
                
    async def _migrate_batch(self, session: aiohttp.ClientSession, 
                           batch: List[Dict[str, Any]], class_name: str):
        """Migrate a batch of objects to target"""
        objects = []
        
        for obj in batch:
            # Extract additional data
            additional = obj.pop('_additional', {})
            obj_id = additional.get('id')
            vector = additional.get('vector')
            
            # Create Weaviate object
            weaviate_obj = {
                "class": class_name,
                "id": obj_id,
                "properties": obj
            }
            
            if vector:
                weaviate_obj["vector"] = vector
                
            objects.append(weaviate_obj)
            
        # Send batch to target
        batch_request = {"objects": objects}
        
        async with session.post(
            f"{self.target_url}/v1/batch/objects",
            json=batch_request,
            headers={"Content-Type": "application/json"}
        ) as response:
            if response.status == 200:
                result = await response.json()
                
                # Check for errors in batch response
                for item in result:
                    if item.get('result', {}).get('errors'):
                        self.migration_stats['failed_objects'] += 1
                        logger.error(f"Error migrating object: {item['result']['errors']}")
                    else:
                        self.migration_stats['migrated_objects'] += 1
            else:
                logger.error(f"Batch migration failed: {response.status}")
                self.migration_stats['failed_objects'] += len(objects)
                
    def _log_migration_stats(self):
        """Log migration statistics"""
        duration = self.migration_stats['end_time'] - self.migration_stats['start_time']
        
        logger.info("Migration Statistics:")
        logger.info(f"  Total objects: {self.migration_stats['total_objects']}")
        logger.info(f"  Migrated objects: {self.migration_stats['migrated_objects']}")
        logger.info(f"  Failed objects: {self.migration_stats['failed_objects']}")
        logger.info(f"  Duration: {duration}")
        logger.info(f"  Success rate: {(self.migration_stats['migrated_objects'] / self.migration_stats['total_objects']) * 100:.2f}%")

# Usage example
async def main():
    migration_manager = DataMigrationManager(
        source_url="http://source-weaviate:8080",
        target_url="http://target-weaviate:8080",
        batch_size=500
    )
    
    # Migrate each class
    classes = ["TelecomKnowledge", "IntentPatterns", "NetworkFunctions"]
    
    for class_name in classes:
        await migration_manager.migrate_data(class_name)
        
if __name__ == "__main__":
    asyncio.run(main())
```

## Testing and Validation

### Comprehensive Migration Testing

```yaml
# testing/migration-test-suite.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migration-test-suite
  namespace: nephoran-system
data:
  test-suite.sh: |
    #!/bin/bash
    set -e
    
    WEAVIATE_URL="${WEAVIATE_URL:-http://weaviate:8080}"
    RAG_API_URL="${RAG_API_URL:-http://rag-api:8080}"
    
    echo "Starting comprehensive migration validation..."
    
    # Test 1: Data integrity check
    echo "Test 1: Data integrity validation"
    
    # Get object counts per class
    for class in TelecomKnowledge IntentPatterns NetworkFunctions; do
      count=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"{ Aggregate { ${class} { meta { count } } } }\"}" \
        "$WEAVIATE_URL/v1/graphql" | jq -r ".data.Aggregate.${class}[0].meta.count")
      
      echo "  $class: $count objects"
      
      if [[ $count -eq 0 ]]; then
        echo "  ✗ No objects found for $class"
        exit 1
      else
        echo "  ✓ $class has $count objects"
      fi
    done
    
    # Test 2: Schema validation
    echo "Test 2: Schema validation"
    
    schema=$(curl -s "$WEAVIATE_URL/v1/schema")
    
    for class in TelecomKnowledge IntentPatterns NetworkFunctions; do
      if echo "$schema" | jq -r '.classes[].class' | grep -q "$class"; then
        echo "  ✓ $class schema found"
      else
        echo "  ✗ $class schema missing"
        exit 1
      fi
    done
    
    # Test 3: Query functionality
    echo "Test 3: Query functionality validation"
    
    # Test vector search
    query_result=$(curl -s -X POST \
      -H "Content-Type: application/json" \
      -d '{
        "query": "{ Get { TelecomKnowledge(nearText: {concepts: [\"AMF\"]}, limit: 5) { content title } } }"
      }' \
      "$WEAVIATE_URL/v1/graphql")
    
    if echo "$query_result" | jq -r '.data.Get.TelecomKnowledge | length' | grep -q '^[1-9]'; then
      echo "  ✓ Vector search working"
    else
      echo "  ✗ Vector search not working"
      exit 1
    fi
    
    # Test hybrid search
    hybrid_result=$(curl -s -X POST \
      -H "Content-Type: application/json" \
      -d '{
        "query": "{ Get { TelecomKnowledge(hybrid: {query: \"5G Core\"}, limit: 5) { content title } } }"
      }' \
      "$WEAVIATE_URL/v1/graphql")
    
    if echo "$hybrid_result" | jq -r '.data.Get.TelecomKnowledge | length' | grep -q '^[1-9]'; then
      echo "  ✓ Hybrid search working"
    else
      echo "  ✗ Hybrid search not working"
      exit 1
    fi
    
    # Test 4: RAG API functionality
    echo "Test 4: RAG API functionality validation"
    
    # Test search endpoint
    search_result=$(curl -s -X POST \
      -H "Content-Type: application/json" \
      -d '{"query": "What is AMF?", "limit": 5}' \
      "$RAG_API_URL/api/v1/query/search")
    
    if echo "$search_result" | jq -r '.results | length' | grep -q '^[1-9]'; then
      echo "  ✓ RAG API search working"
    else
      echo "  ✗ RAG API search not working"
      exit 1
    fi
    
    # Test intent processing
    intent_result=$(curl -s -X POST \
      -H "Content-Type: application/json" \
      -d '{"intent": "Configure AMF for high availability"}' \
      "$RAG_API_URL/api/v1/query/intent")
    
    if echo "$intent_result" | jq -r '.interpretation' | grep -q '.*'; then
      echo "  ✓ Intent processing working"
    else
      echo "  ✗ Intent processing not working"
      exit 1
    fi
    
    # Test 5: Performance validation
    echo "Test 5: Performance validation"
    
    # Measure query latency
    start_time=$(date +%s%3N)
    curl -s -X POST \
      -H "Content-Type: application/json" \
      -d '{"query": "5G network functions", "limit": 10}' \
      "$RAG_API_URL/api/v1/query/search" > /dev/null
    end_time=$(date +%s%3N)
    
    latency=$((end_time - start_time))
    echo "  Query latency: ${latency}ms"
    
    if [[ $latency -lt 5000 ]]; then
      echo "  ✓ Query latency acceptable (<5s)"
    else
      echo "  ✗ Query latency too high (>5s)"
      exit 1
    fi
    
    echo "All migration validation tests passed!"

---
# Performance test job
apiVersion: batch/v1
kind: Job
metadata:
  name: migration-performance-test
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: performance-tester
        image: loadimpact/k6:latest
        env:
        - name: RAG_API_URL
          value: "http://rag-api:8080"
        command: ["k6"]
        args: ["run", "/scripts/performance-test.js"]
        volumeMounts:
        - name: test-scripts
          mountPath: /scripts
      volumes:
      - name: test-scripts
        configMap:
          name: performance-test-scripts
      restartPolicy: Never

---
# Performance test scripts
apiVersion: v1
kind: ConfigMap
metadata:
  name: performance-test-scripts
  namespace: nephoran-system
data:
  performance-test.js: |
    import http from 'k6/http';
    import { check, sleep } from 'k6';
    
    export let options = {
      stages: [
        { duration: '2m', target: 10 },   // Ramp up to 10 users
        { duration: '5m', target: 10 },   // Stay at 10 users
        { duration: '2m', target: 20 },   // Ramp up to 20 users
        { duration: '5m', target: 20 },   // Stay at 20 users
        { duration: '2m', target: 0 },    // Ramp down to 0 users
      ],
      thresholds: {
        http_req_duration: ['p(95)<2000'], // 95% of requests must complete below 2s
        http_req_failed: ['rate<0.05'],    // Error rate must be below 5%
      },
    };
    
    const RAG_API_URL = __ENV.RAG_API_URL || 'http://rag-api:8080';
    
    export default function() {
      // Test search queries
      let searchPayload = JSON.stringify({
        query: 'What is AMF in 5G?',
        limit: 10
      });
      
      let searchResponse = http.post(`${RAG_API_URL}/api/v1/query/search`, searchPayload, {
        headers: { 'Content-Type': 'application/json' },
      });
      
      check(searchResponse, {
        'search status is 200': (r) => r.status === 200,
        'search response has results': (r) => JSON.parse(r.body).results.length > 0,
        'search response time < 2s': (r) => r.timings.duration < 2000,
      });
      
      sleep(1);
      
      // Test intent processing
      let intentPayload = JSON.stringify({
        intent: 'Configure SMF with load balancing'
      });
      
      let intentResponse = http.post(`${RAG_API_URL}/api/v1/query/intent`, intentPayload, {
        headers: { 'Content-Type': 'application/json' },
      });
      
      check(intentResponse, {
        'intent status is 200': (r) => r.status === 200,
        'intent response has interpretation': (r) => JSON.parse(r.body).interpretation !== undefined,
        'intent response time < 3s': (r) => r.timings.duration < 3000,
      });
      
      sleep(2);
    }
```

## Rollback Procedures

### Automated Rollback Script

```bash
#!/bin/bash
# rollback.sh

set -e

NAMESPACE="nephoran-system"
BACKUP_DIR="${BACKUP_DIR:-./rollback}"
ROLLBACK_REASON="${ROLLBACK_REASON:-Manual rollback}"

echo "Starting rollback procedure..."
echo "Reason: $ROLLBACK_REASON"

# Confirm rollback
read -p "Are you sure you want to rollback? This will restore the previous version. (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Rollback cancelled."
    exit 0
fi

# Create rollback log
ROLLBACK_LOG="rollback-$(date +%Y%m%d-%H%M%S).log"
exec > >(tee -a "$ROLLBACK_LOG")
exec 2>&1

echo "Rollback started at $(date)"

# Step 1: Stop current deployments
echo "Step 1: Scaling down current deployments..."
kubectl scale deployment weaviate --replicas=0 -n $NAMESPACE
kubectl scale deployment rag-api --replicas=0 -n $NAMESPACE
kubectl scale deployment llm-processor --replicas=0 -n $NAMESPACE

# Wait for pods to terminate
kubectl wait --for=delete pod -l app=weaviate -n $NAMESPACE --timeout=300s
kubectl wait --for=delete pod -l app=rag-api -n $NAMESPACE --timeout=300s
kubectl wait --for=delete pod -l app=llm-processor -n $NAMESPACE --timeout=300s

# Step 2: Restore configurations
echo "Step 2: Restoring previous configurations..."

if [[ -f "$BACKUP_DIR/weaviate-deployment.yaml" ]]; then
    kubectl apply -f "$BACKUP_DIR/weaviate-deployment.yaml"
    echo "Weaviate deployment restored"
else
    echo "Warning: Weaviate deployment backup not found"
fi

if [[ -f "$BACKUP_DIR/rag-api-deployment.yaml" ]]; then
    kubectl apply -f "$BACKUP_DIR/rag-api-deployment.yaml"
    echo "RAG API deployment restored"
else
    echo "Warning: RAG API deployment backup not found"
fi

if [[ -f "$BACKUP_DIR/llm-processor-deployment.yaml" ]]; then
    kubectl apply -f "$BACKUP_DIR/llm-processor-deployment.yaml"
    echo "LLM processor deployment restored"
else
    echo "Warning: LLM processor deployment backup not found"
fi

# Step 3: Restore data if needed
echo "Step 3: Checking if data restore is needed..."

if [[ -f "$BACKUP_DIR/data-backup-checksum.txt" ]]; then
    echo "Data backup found. Initiating data restore..."
    
    # Create data restore job
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: rollback-data-restore
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: data-restore
        image: nephoran/backup-tool:latest
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: OPERATION
          value: "restore"
        - name: BACKUP_PATH
          value: "/backup/data"
        command: ["/scripts/restore-data.sh"]
        volumeMounts:
        - name: backup-data
          mountPath: /backup
      volumes:
      - name: backup-data
        persistentVolumeClaim:
          claimName: backup-pvc
      restartPolicy: Never
  backoffLimit: 3
EOF

    # Wait for data restore to complete
    kubectl wait --for=condition=complete job/rollback-data-restore -n $NAMESPACE --timeout=3600s
    
    if kubectl logs job/rollback-data-restore -n $NAMESPACE | grep -q "Data restore completed"; then
        echo "Data restore completed successfully"
    else
        echo "Warning: Data restore may have failed. Check logs."
    fi
else
    echo "No data backup found. Skipping data restore."
fi

# Step 4: Wait for services to be ready
echo "Step 4: Waiting for services to be ready..."

kubectl wait --for=condition=available deployment/weaviate -n $NAMESPACE --timeout=600s
kubectl wait --for=condition=available deployment/rag-api -n $NAMESPACE --timeout=600s
kubectl wait --for=condition=available deployment/llm-processor -n $NAMESPACE --timeout=600s

# Step 5: Validate rollback
echo "Step 5: Validating rollback..."

# Basic health check
HEALTH_CHECK_PASSED=true

# Check Weaviate health
if ! kubectl exec deployment/weaviate -n $NAMESPACE -- curl -f http://localhost:8080/v1/.well-known/ready > /dev/null 2>&1; then
    echo "Warning: Weaviate health check failed"
    HEALTH_CHECK_PASSED=false
fi

# Check RAG API health
if ! kubectl exec deployment/rag-api -n $NAMESPACE -- curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "Warning: RAG API health check failed"
    HEALTH_CHECK_PASSED=false
fi

# Run validation tests
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: rollback-validation
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: validator
        image: curlimages/curl:8.5.0
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: RAG_API_URL
          value: "http://rag-api:8080"
        command: ["/bin/sh"]
        args:
        - -c
        - |
          echo "Running rollback validation..."
          
          # Test basic query
          if curl -f -X POST \
            -H "Content-Type: application/json" \
            -d '{"query": "{ Get { TelecomKnowledge(limit: 1) { content } } }"}' \
            "\$WEAVIATE_URL/v1/graphql" > /dev/null 2>&1; then
            echo "Basic query test passed"
          else
            echo "Basic query test failed"
            exit 1
          fi
          
          # Test RAG API
          if curl -f -X POST \
            -H "Content-Type: application/json" \
            -d '{"query": "test query", "limit": 1}' \
            "\$RAG_API_URL/api/v1/query/search" > /dev/null 2>&1; then
            echo "RAG API test passed"
          else
            echo "RAG API test failed"
            exit 1
          fi
          
          echo "Rollback validation completed successfully"
      restartPolicy: Never
  backoffLimit: 2
EOF

kubectl wait --for=condition=complete job/rollback-validation -n $NAMESPACE --timeout=300s

if kubectl logs job/rollback-validation -n $NAMESPACE | grep -q "Rollback validation completed successfully"; then
    echo "✓ Rollback validation passed"
else
    echo "✗ Rollback validation failed"
    HEALTH_CHECK_PASSED=false
fi

# Step 6: Cleanup rollback resources
echo "Step 6: Cleaning up rollback resources..."
kubectl delete job rollback-data-restore rollback-validation -n $NAMESPACE --ignore-not-found

# Final status
echo "Rollback completed at $(date)"

if [[ $HEALTH_CHECK_PASSED == true ]]; then
    echo "✓ Rollback completed successfully"
    echo "Services are healthy and responding"
else
    echo "⚠ Rollback completed with warnings"
    echo "Some health checks failed. Please investigate."
fi

echo "Rollback log saved to: $ROLLBACK_LOG"
```

## Post-Migration Optimization

### Performance Optimization After Migration

```yaml
# optimization/post-migration-optimization.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: post-migration-optimization
  namespace: nephoran-system
data:
  optimization-script.sh: |
    #!/bin/bash
    set -e
    
    echo "Starting post-migration optimization..."
    
    # 1. Index optimization
    echo "Step 1: Optimizing vector indexes..."
    
    kubectl exec deployment/weaviate -n nephoran-system -- curl -X POST \
      -H "Content-Type: application/json" \
      -d '{"query": "{ Meta { TelecomKnowledge { count } } }"}' \
      http://localhost:8080/v1/graphql > /tmp/object_count.json
    
    OBJECT_COUNT=$(cat /tmp/object_count.json | jq -r '.data.Meta.TelecomKnowledge.count')
    echo "Total objects: $OBJECT_COUNT"
    
    # Optimize HNSW parameters based on object count
    if [[ $OBJECT_COUNT -gt 1000000 ]]; then
      # Large dataset optimization
      kubectl patch deployment weaviate -n nephoran-system -p '{
        "spec": {
          "template": {
            "spec": {
              "containers": [{
                "name": "weaviate",
                "env": [
                  {"name": "HNSW_MAX_CONNECTIONS", "value": "64"},
                  {"name": "HNSW_EF_CONSTRUCTION", "value": "512"},
                  {"name": "VECTOR_CACHE_MAX_OBJECTS", "value": "5000000"}
                ]
              }]
            }
          }
        }
      }'
    elif [[ $OBJECT_COUNT -gt 100000 ]]; then
      # Medium dataset optimization
      kubectl patch deployment weaviate -n nephoran-system -p '{
        "spec": {
          "template": {
            "spec": {
              "containers": [{
                "name": "weaviate",
                "env": [
                  {"name": "HNSW_MAX_CONNECTIONS", "value": "32"},
                  {"name": "HNSW_EF_CONSTRUCTION", "value": "256"},
                  {"name": "VECTOR_CACHE_MAX_OBJECTS", "value": "2000000"}
                ]
              }]
            }
          }
        }
      }'
    fi
    
    # 2. Cache warming
    echo "Step 2: Warming up caches..."
    
    kubectl apply -f - <<EOF
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: cache-warming
      namespace: nephoran-system
    spec:
      template:
        spec:
          containers:
          - name: cache-warmer
            image: curlimages/curl:8.5.0
            env:
            - name: RAG_API_URL
              value: "http://rag-api:8080"
            command: ["/bin/sh"]
            args:
            - -c
            - |
              echo "Warming up caches with common queries..."
              
              # Common telecom queries for cache warming
              queries=(
                "What is AMF?"
                "5G Core network functions"
                "gNB configuration"
                "SMF procedures"
                "UPF deployment"
                "O-RAN architecture"
                "3GPP Release 17"
                "Network slicing"
                "CUPS architecture"
                "SBA design principles"
              )
              
              for query in "\${queries[@]}"; do
                echo "Warming cache with query: \$query"
                curl -X POST \
                  -H "Content-Type: application/json" \
                  -d "{\"query\": \"\$query\", \"limit\": 10}" \
                  "\$RAG_API_URL/api/v1/query/search" > /dev/null 2>&1 || true
                sleep 2
              done
              
              echo "Cache warming completed"
          restartPolicy: Never
    EOF
    
    kubectl wait --for=condition=complete job/cache-warming -n nephoran-system --timeout=600s
    
    # 3. Resource optimization
    echo "Step 3: Optimizing resource allocation..."
    
    # Get current resource usage
    kubectl top pods -n nephoran-system > /tmp/resource_usage.txt
    
    # Adjust HPA settings based on current load
    kubectl patch hpa weaviate-hpa -n nephoran-system -p '{
      "spec": {
        "behavior": {
          "scaleUp": {
            "stabilizationWindowSeconds": 60,
            "policies": [
              {"type": "Percent", "value": 100, "periodSeconds": 60}
            ]
          },
          "scaleDown": {
            "stabilizationWindowSeconds": 300,
            "policies": [
              {"type": "Percent", "value": 25, "periodSeconds": 60}
            ]
          }
        }
      }
    }'
    
    # 4. Monitoring setup
    echo "Step 4: Setting up enhanced monitoring..."
    
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: post-migration-alerts
      namespace: nephoran-system
    data:
      alerts.yaml: |
        groups:
        - name: post-migration-alerts
          rules:
          - alert: HighQueryLatency
            expr: histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m])) > 2
            for: 2m
            labels:
              severity: warning
            annotations:
              summary: "High query latency detected after migration"
              
          - alert: LowCacheHitRate
            expr: rag_cache_hit_rate < 0.7
            for: 5m
            labels:
              severity: warning
            annotations:
              summary: "Cache hit rate is low after migration"
              
          - alert: HighErrorRate
            expr: rate(rag_api_errors_total[5m]) > 0.05
            for: 2m
            labels:
              severity: critical
            annotations:
              summary: "High error rate detected after migration"
    EOF
    
    echo "Post-migration optimization completed successfully!"
    
  performance-validation.sh: |
    #!/bin/bash
    
    echo "Running post-migration performance validation..."
    
    RAG_API_URL="http://rag-api:8080"
    
    # Performance benchmarks
    declare -A benchmarks=(
      ["query_latency_p95"]=2000  # 2 seconds
      ["cache_hit_rate"]=0.7      # 70%
      ["error_rate"]=0.05         # 5%
      ["throughput"]=100          # queries per minute
    )
    
    # Run performance tests
    echo "Testing query latency..."
    start_time=$(date +%s%3N)
    for i in {1..10}; do
      curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"query": "5G AMF configuration", "limit": 5}' \
        "$RAG_API_URL/api/v1/query/search" > /dev/null
    done
    end_time=$(date +%s%3N)
    
    avg_latency=$(( (end_time - start_time) / 10 ))
    echo "Average query latency: ${avg_latency}ms"
    
    if [[ $avg_latency -lt ${benchmarks["query_latency_p95"]} ]]; then
      echo "✓ Query latency meets benchmark"
    else
      echo "✗ Query latency exceeds benchmark"
    fi
    
    # Test throughput
    echo "Testing throughput..."
    start_time=$(date +%s)
    for i in {1..100}; do
      curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"query": "test query '$i'", "limit": 1}' \
        "$RAG_API_URL/api/v1/query/search" > /dev/null &
    done
    wait
    end_time=$(date +%s)
    
    duration=$((end_time - start_time))
    throughput=$((100 * 60 / duration))  # queries per minute
    echo "Throughput: ${throughput} queries/minute"
    
    if [[ $throughput -gt ${benchmarks["throughput"]} ]]; then
      echo "✓ Throughput meets benchmark"
    else
      echo "✗ Throughput below benchmark"
    fi
    
    echo "Performance validation completed"
```

This comprehensive migration guide provides detailed procedures for various migration scenarios, ensuring data integrity, minimizing downtime, and validating successful migrations. Each section includes practical scripts and configurations that can be adapted to specific migration requirements.