/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

var (
	kubeconfig string
	ragServiceURL string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig file")
	flag.StringVar(&ragServiceURL, "rag-service-url", "http://localhost:5001/process_intent", "URL of the RAG service")
	flag.Parse()

	http.HandleFunc("/intent", handleIntentRequest)
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "error starting server: %s\n", err)
		os.Exit(1)
	}
}

func handleIntentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var requestData struct {
		Intent string `json:"intent"`
	}
	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
		return
	}

	// Call the RAG service
	ragResponse, err := callRAGService(requestData.Intent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calling RAG service: %v", err), http.StatusInternalServerError)
		return
	}

	// Create the NetworkIntent resource
	if err := createNetworkIntent(ragResponse); err != nil {
		http.Error(w, fmt.Sprintf("Error creating NetworkIntent: %v", err), http.StatusInternalServerError)
		return
	}

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
		return nil, fmt.Errorf("RAG service returned non-OK status: %d", resp.StatusCode)
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

	// Create a REST client
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

	// Create the NetworkIntent object
	intent := &nephoranv1alpha1.NetworkIntent{}
	intent.APIVersion = "nephoran.com/v1alpha1"
	intent.Kind = "NetworkIntent"
	intent.Name = "my-network-intent" // This should be generated dynamically
	intent.Spec.Intent = ragResponse["intent"].(string)
	
	params, err := json.Marshal(ragResponse)
	if err != nil {
		return err
	}
	intent.Spec.Parameters.Raw = params


	// Create the resource
	result := &nephoranv1alpha1.NetworkIntent{}
	err = restClient.Post().
		Namespace("default").
		Resource("networkintents").
		Body(intent).
		Do(context.Background()).
		Into(result)

	return err
}

func getKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}
