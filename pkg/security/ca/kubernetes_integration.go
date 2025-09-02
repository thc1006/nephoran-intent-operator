package ca

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// KubernetesIntegration handles Kubernetes integration for certificate automation
func (e *AutomationEngine) runKubernetesIntegration() {
	defer e.wg.Done()

	e.logger.Info("starting Kubernetes integration")

	// Start service watcher
	if e.config.KubernetesIntegration.ServiceSelector != "" {
		go e.watchServices()
	}

	// Start pod watcher
	if e.config.KubernetesIntegration.PodSelector != "" {
		go e.watchPods()
	}

	// Start ingress watcher
	if e.config.KubernetesIntegration.IngressSelector != "" {
		go e.watchIngresses()
	}

	<-e.ctx.Done()
	e.logger.Info("Kubernetes integration stopped")
}

// Watch Kubernetes Services for certificate automation
func (e *AutomationEngine) watchServices() {
	e.logger.Info("starting service watcher")

	// Create list watcher for services
	watchlist := cache.NewListWatchFromClient(
		e.kubeClient.CoreV1().RESTClient(),
		"services",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	// Create informer
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Service{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if service, ok := obj.(*v1.Service); ok {
					e.handleServiceEvent("add", service)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if service, ok := newObj.(*v1.Service); ok {
					e.handleServiceEvent("update", service)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if service, ok := obj.(*v1.Service); ok {
					e.handleServiceEvent("delete", service)
				}
			},
		},
	)

	// Run controller
	go controller.Run(e.ctx.Done())
}

// Watch Kubernetes Pods for certificate automation
func (e *AutomationEngine) watchPods() {
	e.logger.Info("starting pod watcher")

	// Create list watcher for pods
	watchlist := cache.NewListWatchFromClient(
		e.kubeClient.CoreV1().RESTClient(),
		"pods",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	// Create informer
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if pod, ok := obj.(*v1.Pod); ok {
					e.handlePodEvent("add", pod)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if pod, ok := newObj.(*v1.Pod); ok {
					e.handlePodEvent("update", pod)
				}
			},
		},
	)

	// Run controller
	go controller.Run(e.ctx.Done())
}

// Watch Kubernetes Ingresses for certificate automation
func (e *AutomationEngine) watchIngresses() {
	e.logger.Info("starting ingress watcher")

	// This would implement ingress watching for automatic certificate provisioning
	// based on TLS requirements in ingress resources
}

// Handle service events
func (e *AutomationEngine) handleServiceEvent(eventType string, service *v1.Service) {
	// Check if service matches selector
	if !e.serviceMatchesSelector(service) {
		return
	}

	e.logger.Debug("handling service event",
		"event", eventType,
		"service", service.Name,
		"namespace", service.Namespace)

	switch eventType {
	case "add", "update":
		// Check if service needs certificate
		if e.serviceNeedsCertificate(service) {
			e.provisionCertificateForService(service)
		}
	case "delete":
		// Optionally revoke certificate when service is deleted
		e.revokeCertificateForService(service)
	}
}

// Handle pod events
func (e *AutomationEngine) handlePodEvent(eventType string, pod *v1.Pod) {
	// Check if pod matches selector
	if !e.podMatchesSelector(pod) {
		return
	}

	e.logger.Debug("handling pod event",
		"event", eventType,
		"pod", pod.Name,
		"namespace", pod.Namespace)

	// Check if pod needs certificate injection
	if e.podNeedsCertificateInjection(pod) {
		e.injectCertificateIntoPod(pod)
	}
}

// Check if service matches selector
func (e *AutomationEngine) serviceMatchesSelector(service *v1.Service) bool {
	if e.config.KubernetesIntegration.ServiceSelector == "" {
		return false
	}

	// Check namespace filter
	if len(e.config.KubernetesIntegration.Namespaces) > 0 {
		namespaceAllowed := false
		for _, ns := range e.config.KubernetesIntegration.Namespaces {
			if service.Namespace == ns {
				namespaceAllowed = true
				break
			}
		}
		if !namespaceAllowed {
			return false
		}
	}

	// Check annotation-based selector
	annotationKey := fmt.Sprintf("%s/auto-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := service.Annotations[annotationKey]; exists && value == "true" {
		return true
	}

	// Check label-based selector
	// This would implement label selector matching
	return false
}

// Check if pod matches selector
func (e *AutomationEngine) podMatchesSelector(pod *v1.Pod) bool {
	if e.config.KubernetesIntegration.PodSelector == "" {
		return false
	}

	// Check namespace filter
	if len(e.config.KubernetesIntegration.Namespaces) > 0 {
		namespaceAllowed := false
		for _, ns := range e.config.KubernetesIntegration.Namespaces {
			if pod.Namespace == ns {
				namespaceAllowed = true
				break
			}
		}
		if !namespaceAllowed {
			return false
		}
	}

	// Check annotation-based selector
	annotationKey := fmt.Sprintf("%s/inject-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := pod.Annotations[annotationKey]; exists && value == "true" {
		return true
	}

	return false
}

// Check if service needs certificate
func (e *AutomationEngine) serviceNeedsCertificate(service *v1.Service) bool {
	// Check if service has TLS ports
	for _, port := range service.Spec.Ports {
		if port.Name == "https" || port.Port == 443 || port.Port == 8443 {
			return true
		}
	}

	// Check service annotations
	annotationKey := fmt.Sprintf("%s/auto-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := service.Annotations[annotationKey]; exists && value == "true" {
		return true
	}

	return false
}

// Check if pod needs certificate injection
func (e *AutomationEngine) podNeedsCertificateInjection(pod *v1.Pod) bool {
	// Check pod annotations
	annotationKey := fmt.Sprintf("%s/inject-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := pod.Annotations[annotationKey]; exists && value == "true" {
		return true
	}

	return false
}

// Provision certificate for service
func (e *AutomationEngine) provisionCertificateForService(service *v1.Service) {
	// Extract service information for certificate provisioning
	dnsNames := []string{
		fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
		fmt.Sprintf("%s.%s", service.Name, service.Namespace),
		service.Name,
	}

	// Add external DNS names from annotations
	if externalDNS, exists := service.Annotations[fmt.Sprintf("%s/external-dns", e.config.KubernetesIntegration.AnnotationPrefix)]; exists {
		for _, dns := range strings.Split(externalDNS, ",") {
			dnsNames = append(dnsNames, strings.TrimSpace(dns))
		}
	}

	// Get certificate template from annotations
	template := "default"
	if tmpl, exists := service.Annotations[fmt.Sprintf("%s/certificate-template", e.config.KubernetesIntegration.AnnotationPrefix)]; exists {
		template = tmpl
	}

	// Create provisioning request
	req := &ProvisioningRequest{
		ID:          fmt.Sprintf("service-%s-%s", service.Namespace, service.Name),
		ServiceName: service.Name,
		Namespace:   service.Namespace,
		Template:    template,
		DNSNames:    dnsNames,
		Priority:    PriorityNormal,
		Metadata: map[string]string{
			"kubernetes_service": "true",
			"service_uid":        string(service.UID),
		},
	}

	// Submit provisioning request
	if err := e.RequestProvisioning(req); err != nil {
		e.logger.Error("failed to request certificate provisioning for service",
			"service", service.Name,
			"namespace", service.Namespace,
			"error", err)
	} else {
		e.logger.Info("requested certificate provisioning for service",
			"service", service.Name,
			"namespace", service.Namespace,
			"dns_names", dnsNames)
	}
}

// Revoke certificate for service
func (e *AutomationEngine) revokeCertificateForService(service *v1.Service) {
	// Find certificates associated with the service
	filters := map[string]string{
		"service_name": service.Name,
		"namespace":    service.Namespace,
		"status":       string(StatusIssued),
	}

	certificates, err := e.caManager.ListCertificates(filters)
	if err != nil {
		e.logger.Error("failed to list certificates for service revocation",
			"service", service.Name,
			"namespace", service.Namespace,
			"error", err)
		return
	}

	// Revoke each certificate
	for _, cert := range certificates {
		if err := e.caManager.RevokeCertificate(e.ctx, cert.SerialNumber, 1, cert.Metadata["tenant_id"]); err != nil {
			e.logger.Error("failed to revoke certificate for deleted service",
				"service", service.Name,
				"namespace", service.Namespace,
				"serial_number", cert.SerialNumber,
				"error", err)
		} else {
			e.logger.Info("revoked certificate for deleted service",
				"service", service.Name,
				"namespace", service.Namespace,
				"serial_number", cert.SerialNumber)
		}
	}
}

// Inject certificate into pod
func (e *AutomationEngine) injectCertificateIntoPod(pod *v1.Pod) {
	e.logger.Debug("injecting certificate into pod",
		"pod", pod.Name,
		"namespace", pod.Namespace)

	// This would implement certificate injection into running pods
	// via volume mounts or environment variables
}

// Admission Webhook Implementation

// runAdmissionWebhook starts the admission webhook server
func (e *AutomationEngine) runAdmissionWebhook() {
	defer e.wg.Done()

	webhook := e.config.KubernetesIntegration.AdmissionWebhook
	e.logger.Info("starting admission webhook server",
		"port", webhook.Port,
		"cert_dir", webhook.CertDir)

	// Create HTTP server for webhook
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", e.handleMutatingAdmission)
	mux.HandleFunc("/validate", e.handleValidatingAdmission)
	mux.HandleFunc("/health", e.handleWebhookHealth)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", webhook.Port),
		Handler: mux,
	}

	// Start server
	go func() {
		if err := server.ListenAndServeTLS(
			fmt.Sprintf("%s/tls.crt", webhook.CertDir),
			fmt.Sprintf("%s/tls.key", webhook.CertDir),
		); err != nil && err != http.ErrServerClosed {
			e.logger.Error("admission webhook server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-e.ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		e.logger.Error("admission webhook shutdown error", "error", err)
	}

	e.logger.Info("admission webhook stopped")
}

// Handle mutating admission webhook
func (e *AutomationEngine) handleMutatingAdmission(w http.ResponseWriter, r *http.Request) {
	var admissionReview admissionv1.AdmissionReview

	if err := json.NewDecoder(r.Body).Decode(&admissionReview); err != nil {
		e.logger.Error("failed to decode admission review", "error", err)
		// Never expose internal error details to external clients
		http.Error(w, "Invalid admission review format", http.StatusBadRequest)
		return
	}

	req := admissionReview.Request
	var patches []byte
	var err error

	// Handle different resource types
	switch req.Kind.Kind {
	case "Pod":
		patches, err = e.mutatePod(req)
	case "Deployment":
		patches, err = e.mutateDeployment(req)
	case "Service":
		patches, err = e.mutateService(req)
	default:
		// No mutation needed for other resource types
		patches = []byte("[]")
	}

	if err != nil {
		e.logger.Error("mutation failed",
			"kind", req.Kind.Kind,
			"name", req.Name,
			"namespace", req.Namespace,
			"error", err)

		// Sanitize error message to avoid information leakage
		safeMessage := "Mutation operation failed due to policy violation"
		if strings.Contains(err.Error(), "certificate") {
			safeMessage = "Certificate validation failed"
		} else if strings.Contains(err.Error(), "unauthorized") {
			safeMessage = "Authorization check failed"
		}

		response := &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result:  &metav1.Status{Message: safeMessage},
		}
		admissionReview.Response = response
	} else {
		response := &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: true,
			Patch:   patches,
		}

		if len(patches) > 2 { // More than empty array "[]"
			patchType := admissionv1.PatchTypeJSONPatch
			response.PatchType = &patchType
		}

		admissionReview.Response = response
	}

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		e.logger.Error("failed to marshal admission response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// Handle validating admission webhook
func (e *AutomationEngine) handleValidatingAdmission(w http.ResponseWriter, r *http.Request) {
	var admissionReview admissionv1.AdmissionReview

	if err := json.NewDecoder(r.Body).Decode(&admissionReview); err != nil {
		e.logger.Error("failed to decode admission review", "error", err)
		// Never expose internal error details to external clients
		http.Error(w, "Invalid admission review format", http.StatusBadRequest)
		return
	}

	req := admissionReview.Request
	allowed := true
	var message string

	// Validate certificate-related resources
	switch req.Kind.Kind {
	case "Secret":
		allowed, message = e.validateSecretCertificate(req)
	}

	response := &admissionv1.AdmissionResponse{
		UID:     req.UID,
		Allowed: allowed,
	}

	if !allowed {
		response.Result = &metav1.Status{Message: message}
	}

	admissionReview.Response = response

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		e.logger.Error("failed to marshal admission response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// Handle webhook health check
func (e *AutomationEngine) handleWebhookHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Mutate pod for certificate injection
func (e *AutomationEngine) mutatePod(req *admissionv1.AdmissionRequest) ([]byte, error) {
	var pod v1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	var patches []map[string]interface{}

	// Check if pod needs certificate injection
	if e.shouldInjectCertificate(&pod) {
		// Add certificate volume
		volumePatches := e.createCertificateVolumePatches(&pod)
		patches = append(patches, volumePatches...)

		// Add certificate volume mounts
		volumeMountPatches := e.createCertificateVolumeMountPatches(&pod)
		patches = append(patches, volumeMountPatches...)

		// Add environment variables for certificate paths
		envPatches := e.createCertificateEnvPatches(&pod)
		patches = append(patches, envPatches...)
	}

	if len(patches) == 0 {
		return []byte("[]"), nil
	}

	return json.Marshal(patches)
}

// Mutate deployment for certificate injection
func (e *AutomationEngine) mutateDeployment(req *admissionv1.AdmissionRequest) ([]byte, error) {
	var deployment appsv1.Deployment
	if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	// Check if deployment pods should have certificates injected
	if e.shouldInjectCertificateIntoDeployment(&deployment) {
		// Add annotations to pod template
		patches := []map[string]interface{}{
			{
				"op":    "add",
				"path":  "/spec/template/metadata/annotations",
				"value": map[string]string{},
			},
		}
		return json.Marshal(patches)
	}

	return []byte("[]"), nil
}

// Mutate service for certificate provisioning
func (e *AutomationEngine) mutateService(req *admissionv1.AdmissionRequest) ([]byte, error) {
	var service v1.Service
	if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service: %w", err)
	}

	// Check if service should have automatic certificate provisioning
	if e.shouldProvisionCertificateForService(&service) {
		// Add annotation to trigger certificate provisioning
		patches := []map[string]interface{}{
			{
				"op":    "add",
				"path":  "/metadata/annotations",
				"value": map[string]string{},
			},
		}
		return json.Marshal(patches)
	}

	return []byte("[]"), nil
}

// Validate secret certificate
func (e *AutomationEngine) validateSecretCertificate(req *admissionv1.AdmissionRequest) (bool, string) {
	var secret v1.Secret
	if err := json.Unmarshal(req.Object.Raw, &secret); err != nil {
		return false, fmt.Sprintf("failed to unmarshal secret: %v", err)
	}

	// Only validate TLS secrets
	if secret.Type != v1.SecretTypeTLS {
		return true, ""
	}

	// Validate certificate data if present
	if certData, exists := secret.Data["tls.crt"]; exists {
		if err := e.validateCertificateData(certData); err != nil {
			return false, fmt.Sprintf("invalid certificate data: %v", err)
		}
	}

	return true, ""
}

// Helper methods for mutation

func (e *AutomationEngine) shouldInjectCertificate(pod *v1.Pod) bool {
	annotationKey := fmt.Sprintf("%s/inject-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := pod.Annotations[annotationKey]; exists && value == "true" {
		return true
	}
	return false
}

func (e *AutomationEngine) shouldInjectCertificateIntoDeployment(deployment *appsv1.Deployment) bool {
	annotationKey := fmt.Sprintf("%s/inject-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := deployment.Annotations[annotationKey]; exists && value == "true" {
		return true
	}
	return false
}

func (e *AutomationEngine) shouldProvisionCertificateForService(service *v1.Service) bool {
	annotationKey := fmt.Sprintf("%s/auto-certificate", e.config.KubernetesIntegration.AnnotationPrefix)
	if value, exists := service.Annotations[annotationKey]; exists && value == "true" {
		return true
	}
	return false
}

func (e *AutomationEngine) createCertificateVolumePatches(pod *v1.Pod) []map[string]interface{} {
	secretName := fmt.Sprintf("%s-%s-tls", e.config.KubernetesIntegration.SecretPrefix, pod.Name)

	volume := map[string]interface{}{
		"name": "tls-cert",
		"secret": map[string]interface{}{
			"secretName": secretName,
		},
	}

	return []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/volumes/-",
			"value": volume,
		},
	}
}

func (e *AutomationEngine) createCertificateVolumeMountPatches(pod *v1.Pod) []map[string]interface{} {
	volumeMount := map[string]interface{}{
		"name":      "tls-cert",
		"mountPath": "/etc/tls",
		"readOnly":  true,
	}

	var patches []map[string]interface{}
	for i := range pod.Spec.Containers {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
			"value": volumeMount,
		})
	}

	return patches
}

func (e *AutomationEngine) createCertificateEnvPatches(pod *v1.Pod) []map[string]interface{} {
	envVars := []map[string]string{
		{
			"name":  "TLS_CERT_PATH",
			"value": "/etc/ssl/certs/service/tls.crt",
		},
		{
			"name":  "TLS_KEY_PATH",
			"value": "/etc/ssl/certs/service/tls.key",
		},
		{
			"name":  "CA_CERT_PATH",
			"value": "/etc/ssl/certs/service/ca.crt",
		},
	}

	var patches []map[string]interface{}
	for i := range pod.Spec.Containers {
		for _, envVar := range envVars {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("/spec/containers/%d/env/-", i),
				"value": envVar,
			})
		}
	}

	return patches
}

func (e *AutomationEngine) validateCertificateData(certData []byte) error {
	// Parse and validate certificate
	// This would implement comprehensive certificate validation
	return nil
}

