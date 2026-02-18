// Package main implements the RAG Pipeline CLI tool.
// It accepts natural language network intent, calls the RAG service,
// and creates a NetworkIntent CRD in Kubernetes.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// RAGRequest is the payload sent to the RAG service /process_intent endpoint.
type RAGRequest struct {
	Intent string `json:"intent"`
}

// RAGResponse is the structured response from the RAG service.
type RAGResponse struct {
	IntentID         string           `json:"intent_id"`
	StructuredOutput StructuredOutput `json:"structured_output"`
	Status           string           `json:"status"`
	Metrics          RAGMetrics       `json:"metrics"`
}

// StructuredOutput is the NetworkFunctionScale result from the RAG service.
type StructuredOutput struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
}

// RAGMetrics contains performance metrics from the RAG service.
type RAGMetrics struct {
	ProcessingTimeMS float64 `json:"processing_time_ms"`
	ModelVersion     string  `json:"model_version"`
	RetrievalScore   float64 `json:"retrieval_score"`
}

var niGVR = schema.GroupVersionResource{
	Group:    "nephoran.com",
	Version:  "v1",
	Resource: "networkintents",
}

func main() {
	var (
		ragURL    = flag.String("rag-url", getEnvOrDefault("RAG_URL", "http://10.110.166.224:8000"), "RAG service URL")
		namespace = flag.String("namespace", "default", "Kubernetes namespace for NetworkIntent")
		watchFlag = flag.Bool("watch", true, "Watch for reconciliation status after creation")
		timeout   = flag.Duration("timeout", 60*time.Second, "Timeout for watching reconciliation")
	)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: rag-pipeline [flags] <natural language intent>")
		fmt.Println("Example: rag-pipeline \"scale up web-app to 5 replicas\"")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	intent := strings.Join(args, " ")
	fmt.Printf("\n[1] Natural Language Intent: %q\n\n", intent)

	// Step 1: Call RAG service
	fmt.Printf("[Step 1/3] Calling RAG service at %s...\n", *ragURL)
	ragResp, err := callRAGService(*ragURL, intent)
	if err != nil {
		log.Fatalf("RAG service error: %v", err)
	}

	fmt.Printf("[OK] RAG Response:\n")
	fmt.Printf("     Intent ID  : %s\n", ragResp.IntentID)
	fmt.Printf("     Type       : %s\n", ragResp.StructuredOutput.Type)
	fmt.Printf("     Target     : %s\n", ragResp.StructuredOutput.Name)
	fmt.Printf("     Namespace  : %s\n", ragResp.StructuredOutput.Namespace)
	fmt.Printf("     Replicas   : %d\n", ragResp.StructuredOutput.Replicas)
	fmt.Printf("     Model      : %s\n", ragResp.Metrics.ModelVersion)
	fmt.Printf("     RAG Score  : %.3f\n", ragResp.Metrics.RetrievalScore)
	fmt.Printf("     Time       : %.0fms\n\n", ragResp.Metrics.ProcessingTimeMS)

	// Step 2: Create NetworkIntent CRD
	fmt.Printf("[Step 2/3] Creating NetworkIntent CRD in K8s...\n")

	intentName := fmt.Sprintf("pipeline-%s-%d",
		sanitizeName(ragResp.StructuredOutput.Name),
		time.Now().Unix(),
	)

	targetNamespace := ragResp.StructuredOutput.Namespace
	if targetNamespace == "" {
		targetNamespace = *namespace
	}

	niObj := buildUnstructuredNetworkIntent(intentName, targetNamespace, ragResp, intent)

	dynamicClient, err := buildDynamicClient()
	if err != nil {
		log.Fatalf("Kubernetes client error: %v", err)
	}

	ctx := context.Background()

	created, err := dynamicClient.Resource(niGVR).Namespace(targetNamespace).Create(
		ctx, niObj, metav1.CreateOptions{},
	)
	if err != nil {
		log.Fatalf("Failed to create NetworkIntent: %v", err)
	}

	fmt.Printf("[OK] NetworkIntent created:\n")
	fmt.Printf("     Name      : %s\n", created.GetName())
	fmt.Printf("     Namespace : %s\n", created.GetNamespace())
	fmt.Printf("     UID       : %s\n\n", string(created.GetUID()))

	// Step 3: Watch for reconciliation
	if *watchFlag {
		fmt.Printf("[Step 3/3] Watching reconciliation (timeout: %v)...\n", *timeout)
		watchReconciliation(ctx, dynamicClient, targetNamespace, intentName, *timeout)
	}
}

func callRAGService(baseURL, intent string) (*RAGResponse, error) {
	reqBody, err := json.Marshal(RAGRequest{Intent: intent})
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(baseURL+"/process_intent", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RAG service returned status %d", resp.StatusCode)
	}

	var ragResp RAGResponse
	if err := json.NewDecoder(resp.Body).Decode(&ragResp); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return &ragResp, nil
}

func buildUnstructuredNetworkIntent(name, namespace string, ragResp *RAGResponse, originalIntent string) *unstructured.Unstructured {
	intentType := "scaling"
	switch ragResp.StructuredOutput.Type {
	case "NetworkFunctionScale":
		intentType = "scaling"
	default:
		if ragResp.StructuredOutput.Type != "" {
			intentType = strings.ToLower(ragResp.StructuredOutput.Type)
		}
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nephoran.com/v1",
			"kind":       "NetworkIntent",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"annotations": map[string]interface{}{
					"intent.nephoran.com/rag-intent-id":       ragResp.IntentID,
					"intent.nephoran.com/rag-model":           ragResp.Metrics.ModelVersion,
					"intent.nephoran.com/rag-retrieval-score": fmt.Sprintf("%.3f", ragResp.Metrics.RetrievalScore),
					"intent.nephoran.com/rag-target":          ragResp.StructuredOutput.Name,
					"intent.nephoran.com/source":              "nlp-rag-pipeline",
				},
			},
			"spec": map[string]interface{}{
				"intent":     originalIntent,
				"intentType": intentType,
			},
		},
	}
}

func buildDynamicClient() (dynamic.Interface, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = home + "/.kube/config"
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig error: %w", err)
	}

	return dynamic.NewForConfig(cfg)
}

func watchReconciliation(ctx context.Context, client dynamic.Interface, namespace, name string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	watcher, err := client.Resource(niGVR).Namespace(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		fmt.Printf("[WARN] Cannot watch: %v\n", err)
		return
	}
	defer watcher.Stop()

	fmt.Printf("     Watching %s/%s...\n", namespace, name)

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				fmt.Println("     Watch channel closed")
				return
			}

			if event.Type == watch.Modified || event.Type == watch.Added {
				obj, ok := event.Object.(*unstructured.Unstructured)
				if !ok {
					continue
				}

				phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
				message, _, _ := unstructured.NestedString(obj.Object, "status", "message")

				if phase != "" {
					fmt.Printf("     [Phase] %s", phase)
					if message != "" {
						fmt.Printf(" | %s", message)
					}
					fmt.Println()

					terminalPhases := map[string]bool{
						string(nephoranv1.NetworkIntentPhaseDeployed): true,
						string(nephoranv1.NetworkIntentPhaseFailed):   true,
						"Processed": true,
						"Error":     true,
					}
					if terminalPhases[phase] {
						if phase == string(nephoranv1.NetworkIntentPhaseDeployed) || phase == "Processed" {
							fmt.Println("\n[SUCCESS] Pipeline complete! NetworkIntent successfully reconciled.")
						} else {
							fmt.Println("\n[FAILED] Reconciliation failed.")
						}
						return
					}
				}
			}

		case <-ctx.Done():
			fmt.Printf("\n[TIMEOUT] Check NetworkIntent status manually:\n")
			fmt.Printf("     kubectl get networkintent %s -n %s -o yaml\n", name, namespace)
			return
		}
	}
}

func sanitizeName(name string) string {
	result := strings.ToLower(name)
	var sb strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	s := strings.Trim(sb.String(), "-")
	if len(s) > 30 {
		s = s[:30]
	}
	return s
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
