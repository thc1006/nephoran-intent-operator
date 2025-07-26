package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

var (
	kubeconfig    string
	ragServiceURL string
	namespace     string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig file")
	flag.StringVar(&ragServiceURL, "rag-service-url", "http://localhost:5001/process_intent", "URL of the RAG service")
	flag.StringVar(&namespace, "namespace", "default", "The namespace to create the NetworkIntent in")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	http.HandleFunc("/intent", handleIntentRequest)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	setupLog.Info("starting server", "addr", ":8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		setupLog.Error(err, "problem running server")
		os.Exit(1)
	}
}

func handleIntentRequest(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(err, "Error reading request body")
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var requestData struct {
		Intent string `json:"intent"`
	}
	if err := json.Unmarshal(body, &requestData); err != nil {
		logger.Error(err, "Error parsing JSON request body")
		http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
		return
	}

	logger.Info("Received intent", "intent", requestData.Intent)

	// Call the RAG service
	ragResponse, err := callRAGService(requestData.Intent)
	if err != nil {
		logger.Error(err, "Error calling RAG service")
		http.Error(w, fmt.Sprintf("Error calling RAG service: %v", err), http.StatusInternalServerError)
		return
	}

	// Create the NetworkIntent resource
	if err := createNetworkIntent(ragResponse); err != nil {
		logger.Error(err, "Error creating NetworkIntent")
		http.Error(w, fmt.Sprintf("Error creating NetworkIntent: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Info("NetworkIntent created successfully")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("NetworkIntent created successfully"))
}

func callRAGService(intent string) (map[string]interface{}, error) {
	requestBody, err := json.Marshal(map[string]string{"intent": intent})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(ragServiceURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("RAG service returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	var ragResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ragResponse); err != nil {
		return nil, err
	}

	return ragResponse, nil
}

func createNetworkIntent(ragResponse map[string]interface{}) error {
	config, err := getKubeConfig(kubeconfig)
	if err != nil {
		return err
	}

	nephoranv1alpha1.AddToScheme(scheme.Scheme)
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &nephoranv1alpha1.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		return err
	}

	intentName, ok := ragResponse["name"].(string)
	if !ok {
		// Fallback to a generated name if not provided by the LLM
		intentName = fmt.Sprintf("intent-%d", os.Getpid())
	}

	params, err := json.Marshal(ragResponse)
	if err != nil {
		return err
	}

	intent := &nephoranv1alpha1.NetworkIntent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephoran.com/v1alpha1",
			Kind:       "NetworkIntent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      intentName,
			Namespace: namespace,
		},
		Spec: nephoranv1alpha1.NetworkIntentSpec{
			Intent: ragResponse["intent"].(string),
			Parameters: runtime.RawExtension{
				Raw: params,
			},
		},
	}

	result := &nephoranv1alpha1.NetworkIntent{}
	return restClient.Post().
		Namespace(namespace).
		Resource("networkintents").
		Body(intent).
		Do(context.Background()).
		Into(result)
}

func getKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}