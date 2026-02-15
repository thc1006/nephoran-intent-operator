#!/usr/bin/env python3
"""
Weaviate Performance Benchmark
Tests: batch insert, similarity search, query latency
"""

import json
import time
import random
import statistics
import sys
import urllib.request
import urllib.error

WEAVIATE_URL = "http://localhost:18080"
WEAVIATE_API_KEY = "nephoran-rag-dev-key"
RESULTS_FILE = "/home/thc1006/dev/nephoran-intent-operator/benchmarks/weaviate-raw-results.json"

def api_call(method, path, data=None):
    """Make API call to Weaviate"""
    url = f"{WEAVIATE_URL}{path}"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {WEAVIATE_API_KEY}"
    }
    body = json.dumps(data).encode() if data else None
    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        error_body = e.read().decode() if e.fp else ""
        return {"error": str(e), "body": error_body}
    except Exception as e:
        return {"error": str(e)}

def delete_class(class_name):
    """Delete a class if it exists"""
    try:
        req = urllib.request.Request(
            f"{WEAVIATE_URL}/v1/schema/{class_name}",
            method="DELETE",
            headers={"Authorization": f"Bearer {WEAVIATE_API_KEY}"}
        )
        urllib.request.urlopen(req, timeout=10)
    except:
        pass

def generate_vector(dim):
    """Generate a random unit vector"""
    vec = [random.gauss(0, 1) for _ in range(dim)]
    norm = sum(v*v for v in vec) ** 0.5
    return [v/norm for v in vec]

def benchmark_dimension(dim, num_objects=1000):
    """Run benchmark for a specific vector dimension"""
    class_name = f"BenchmarkDim{dim}"
    print(f"\n--- Benchmarking with {dim}-dimensional vectors ---")

    # Clean up
    delete_class(class_name)
    time.sleep(0.5)

    # Create class
    schema = {
        "class": class_name,
        "vectorizer": "none",
        "properties": [
            {"name": "text", "dataType": ["text"]},
            {"name": "idx", "dataType": ["int"]}
        ]
    }
    result = api_call("POST", "/v1/schema", schema)
    if "error" in result:
        print(f"  Error creating class: {result}")
        return None
    print(f"  Created class: {class_name}")

    # Batch insert
    print(f"  Inserting {num_objects} objects in batches of 100...")
    batch_size = 100
    total_insert_time = 0
    objects_inserted = 0

    for batch_start in range(0, num_objects, batch_size):
        batch_end = min(batch_start + batch_size, num_objects)
        objects = []
        for i in range(batch_start, batch_end):
            objects.append({
                "class": class_name,
                "properties": {
                    "text": f"Test document number {i} with some content for benchmarking purposes",
                    "idx": i
                },
                "vector": generate_vector(dim)
            })

        start = time.time()
        result = api_call("POST", "/v1/batch/objects", {"objects": objects})
        elapsed = time.time() - start
        total_insert_time += elapsed

        if isinstance(result, list):
            objects_inserted += len(result)
        elif isinstance(result, dict) and "error" not in result:
            objects_inserted += batch_end - batch_start

    insert_rate = objects_inserted / total_insert_time if total_insert_time > 0 else 0
    print(f"  Inserted {objects_inserted} objects in {total_insert_time:.3f}s ({insert_rate:.1f} objects/sec)")

    time.sleep(1)  # Let indexing settle

    # Similarity search benchmark
    print(f"  Running similarity search queries...")
    query_latencies = []
    num_queries = 50

    for q in range(num_queries):
        query_vector = generate_vector(dim)
        graphql = {
            "query": f"""
            {{
                Get {{
                    {class_name}(
                        nearVector: {{
                            vector: {json.dumps(query_vector[:10])}
                            distance: 2.0
                        }}
                        limit: 10
                    ) {{
                        text
                        idx
                        _additional {{
                            distance
                        }}
                    }}
                }}
            }}
            """
        }
        # Use full vector for actual query
        graphql["query"] = f"""
        {{
            Get {{
                {class_name}(
                    nearVector: {{
                        vector: {json.dumps(query_vector)}
                        distance: 2.0
                    }}
                    limit: 10
                ) {{
                    text
                    idx
                    _additional {{
                        distance
                    }}
                }}
            }}
        }}
        """

        start = time.time()
        result = api_call("POST", "/v1/graphql", graphql)
        elapsed = time.time() - start
        query_latencies.append(elapsed * 1000)  # Convert to ms

    query_latencies.sort()
    p50 = query_latencies[len(query_latencies) // 2]
    p95 = query_latencies[int(len(query_latencies) * 0.95)]
    p99 = query_latencies[int(len(query_latencies) * 0.99)]
    avg = statistics.mean(query_latencies)

    print(f"  Query latencies (ms): avg={avg:.2f}, p50={p50:.2f}, p95={p95:.2f}, p99={p99:.2f}")

    # Clean up
    delete_class(class_name)

    return {
        "dimension": dim,
        "num_objects": objects_inserted,
        "insert_total_sec": round(total_insert_time, 3),
        "insert_rate_per_sec": round(insert_rate, 1),
        "query_count": num_queries,
        "query_latency_ms": {
            "avg": round(avg, 2),
            "p50": round(p50, 2),
            "p95": round(p95, 2),
            "p99": round(p99, 2),
            "min": round(min(query_latencies), 2),
            "max": round(max(query_latencies), 2)
        }
    }

def main():
    print("=== Weaviate Performance Benchmark ===")
    print(f"Timestamp: {time.strftime('%Y-%m-%dT%H:%M:%S%z')}")
    print(f"Weaviate URL: {WEAVIATE_URL}")

    # Check connectivity
    meta = api_call("GET", "/v1/meta")
    if "error" in meta:
        print(f"Error connecting to Weaviate: {meta}")
        sys.exit(1)
    print(f"Weaviate version: {meta.get('version', 'unknown')}")

    results = {
        "timestamp": time.strftime('%Y-%m-%dT%H:%M:%S%z'),
        "weaviate_version": meta.get("version", "unknown"),
        "benchmarks": []
    }

    # Test with different vector dimensions
    for dim in [128, 384, 768, 1536]:
        result = benchmark_dimension(dim, num_objects=1000)
        if result:
            results["benchmarks"].append(result)

    # Save results
    with open(RESULTS_FILE, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\nResults saved to {RESULTS_FILE}")
    print("=== Benchmark complete ===")

if __name__ == "__main__":
    main()
