
package audit



import (

	"encoding/json"

	"fmt"

	"io"

	"net/http"

	"strings"

	"time"



	"github.com/go-logr/logr"



	admissionv1 "k8s.io/api/admission/v1"

	authenticationv1 "k8s.io/api/authentication/v1"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/apimachinery/pkg/types"



	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

)



const (

	// Admission webhook paths.

	ValidatingWebhookPath = "/validate"

	// MutatingWebhookPath holds mutatingwebhookpath value.

	MutatingWebhookPath = "/mutate"



	// Admission result reasons.

	ReasonAllowed = "Allowed"

	// ReasonDenied holds reasondenied value.

	ReasonDenied = "Denied"

	// ReasonPolicyViolation holds reasonpolicyviolation value.

	ReasonPolicyViolation = "PolicyViolation"

	// ReasonSecurityRisk holds reasonsecurityrisk value.

	ReasonSecurityRisk = "SecurityRisk"

	// ReasonCompliance holds reasoncompliance value.

	ReasonCompliance = "ComplianceViolation"



	// High-risk operations that require additional logging.

	HighRiskOperations = "CREATE,UPDATE,DELETE"



	// Sensitive resources that require enhanced auditing.

	SensitiveResources = "secrets,configmaps,serviceaccounts,roles,rolebindings,clusterroles,clusterrolebindings"

)



// AdmissionAuditWebhook provides audit logging for Kubernetes admission controllers.

type AdmissionAuditWebhook struct {

	auditSystem *AuditSystem

	config      *AdmissionAuditConfig

	logger      logr.Logger

	decoder     *admission.Decoder

	scheme      *runtime.Scheme

	codecs      serializer.CodecFactory

}



// AdmissionAuditConfig configures the admission audit webhook.

type AdmissionAuditConfig struct {

	// Enabled controls whether admission auditing is active.

	Enabled bool `json:"enabled" yaml:"enabled"`



	// AuditLevel controls the detail level of admission auditing.

	// Options: minimal, standard, detailed, full.

	AuditLevel string `json:"audit_level" yaml:"audit_level"`



	// CaptureRequestObject controls whether to capture the full request object.

	CaptureRequestObject bool `json:"capture_request_object" yaml:"capture_request_object"`



	// CaptureOldObject controls whether to capture the old object on updates.

	CaptureOldObject bool `json:"capture_old_object" yaml:"capture_old_object"`



	// CapturePatch controls whether to capture patch operations.

	CapturePatch bool `json:"capture_patch" yaml:"capture_patch"`



	// SensitiveResources defines resources that require enhanced auditing.

	SensitiveResources []string `json:"sensitive_resources" yaml:"sensitive_resources"`



	// ExcludeNamespaces defines namespaces to exclude from auditing.

	ExcludeNamespaces []string `json:"exclude_namespaces" yaml:"exclude_namespaces"`



	// ExcludeResources defines resources to exclude from auditing.

	ExcludeResources []string `json:"exclude_resources" yaml:"exclude_resources"`



	// HighRiskOperations defines operations that require additional logging.

	HighRiskOperations []string `json:"high_risk_operations" yaml:"high_risk_operations"`



	// AlertOnDenial controls whether to send alerts for denied requests.

	AlertOnDenial bool `json:"alert_on_denial" yaml:"alert_on_denial"`



	// PolicyEnforcement controls whether to enforce additional security policies.

	PolicyEnforcement *PolicyEnforcementConfig `json:"policy_enforcement" yaml:"policy_enforcement"`

}



// PolicyEnforcementConfig defines security policy enforcement settings.

type PolicyEnforcementConfig struct {

	// Enabled controls whether policy enforcement is active.

	Enabled bool `json:"enabled" yaml:"enabled"`



	// RequireSecurityContext enforces security context requirements.

	RequireSecurityContext bool `json:"require_security_context" yaml:"require_security_context"`



	// BlockPrivilegedContainers blocks containers with privileged: true.

	BlockPrivilegedContainers bool `json:"block_privileged_containers" yaml:"block_privileged_containers"`



	// RequireReadOnlyRootFilesystem requires readOnlyRootFilesystem: true.

	RequireReadOnlyRootFilesystem bool `json:"require_readonly_root_filesystem" yaml:"require_readonly_root_filesystem"`



	// BlockHostNetwork blocks hostNetwork: true.

	BlockHostNetwork bool `json:"block_host_network" yaml:"block_host_network"`



	// BlockHostPID blocks hostPID: true.

	BlockHostPID bool `json:"block_host_pid" yaml:"block_host_pid"`



	// AllowedRegistries defines allowed container registries.

	AllowedRegistries []string `json:"allowed_registries" yaml:"allowed_registries"`



	// RequiredLabels defines labels that must be present.

	RequiredLabels map[string]string `json:"required_labels" yaml:"required_labels"`



	// ForbiddenAnnotations defines annotations that are not allowed.

	ForbiddenAnnotations []string `json:"forbidden_annotations" yaml:"forbidden_annotations"`

}



// AdmissionEventInfo captures information about an admission request.

type AdmissionEventInfo struct {

	UID           string                      `json:"uid"`

	Kind          metav1.GroupVersionKind     `json:"kind"`

	Resource      metav1.GroupVersionResource `json:"resource"`

	SubResource   string                      `json:"sub_resource,omitempty"`

	Name          string                      `json:"name,omitempty"`

	Namespace     string                      `json:"namespace,omitempty"`

	Operation     admissionv1.Operation       `json:"operation"`

	UserInfo      *AdmissionUserInfo          `json:"user_info"`

	Object        runtime.RawExtension        `json:"object,omitempty"`

	OldObject     runtime.RawExtension        `json:"old_object,omitempty"`

	Patch         []byte                      `json:"patch,omitempty"`

	PatchType     *admissionv1.PatchType      `json:"patch_type,omitempty"`

	Options       runtime.RawExtension        `json:"options,omitempty"`

	DryRun        bool                        `json:"dry_run"`

	AdmissionTime time.Time                   `json:"admission_time"`

}



// AdmissionUserInfo captures user information from admission requests.

type AdmissionUserInfo struct {

	Username string              `json:"username"`

	UID      string              `json:"uid"`

	Groups   []string            `json:"groups"`

	Extra    map[string][]string `json:"extra,omitempty"`

}



// AdmissionResponseInfo captures information about an admission response.

type AdmissionResponseInfo struct {

	UID              string                 `json:"uid"`

	Allowed          bool                   `json:"allowed"`

	Result           *metav1.Status         `json:"result,omitempty"`

	Patch            []byte                 `json:"patch,omitempty"`

	PatchType        *admissionv1.PatchType `json:"patch_type,omitempty"`

	Warnings         []string               `json:"warnings,omitempty"`

	AuditAnnotations map[string]string      `json:"audit_annotations,omitempty"`

	ProcessingTime   time.Duration          `json:"processing_time"`

}



// PolicyViolation represents a security policy violation.

type PolicyViolation struct {

	PolicyName  string      `json:"policy_name"`

	Violation   string      `json:"violation"`

	Details     string      `json:"details"`

	Severity    string      `json:"severity"`

	Remediation string      `json:"remediation,omitempty"`

	Metadata    interface{} `json:"metadata,omitempty"`

}



// NewAdmissionAuditWebhook creates a new admission audit webhook.

func NewAdmissionAuditWebhook(auditSystem *AuditSystem, scheme *runtime.Scheme, config *AdmissionAuditConfig) *AdmissionAuditWebhook {

	if config == nil {

		config = DefaultAdmissionAuditConfig()

	}



	return &AdmissionAuditWebhook{

		auditSystem: auditSystem,

		config:      config,

		logger:      log.Log.WithName("admission-audit-webhook"),

		scheme:      scheme,

		codecs:      serializer.NewCodecFactory(scheme),

	}

}



// SetupWebhookServer sets up the admission webhook server.

func (w *AdmissionAuditWebhook) SetupWebhookServer() *http.ServeMux {

	mux := http.NewServeMux()



	// Validating admission controller endpoint.

	mux.HandleFunc(ValidatingWebhookPath, w.handleValidatingAdmission)



	// Mutating admission controller endpoint.

	mux.HandleFunc(MutatingWebhookPath, w.handleMutatingAdmission)



	return mux

}



// handleValidatingAdmission handles validating admission requests.

func (w *AdmissionAuditWebhook) handleValidatingAdmission(rw http.ResponseWriter, req *http.Request) {

	w.handleAdmissionRequest(rw, req, "validating")

}



// handleMutatingAdmission handles mutating admission requests.

func (w *AdmissionAuditWebhook) handleMutatingAdmission(rw http.ResponseWriter, req *http.Request) {

	w.handleAdmissionRequest(rw, req, "mutating")

}



// handleAdmissionRequest processes admission requests and creates audit events.

func (w *AdmissionAuditWebhook) handleAdmissionRequest(rw http.ResponseWriter, req *http.Request, webhookType string) {

	if !w.config.Enabled {

		w.sendErrorResponse(rw, fmt.Errorf("admission auditing is disabled"))

		return

	}



	startTime := time.Now()



	// Decode admission review.

	admissionReview := &admissionv1.AdmissionReview{}

	if err := w.decodeAdmissionReview(req, admissionReview); err != nil {

		w.logger.Error(err, "Failed to decode admission review")

		w.sendErrorResponse(rw, err)

		return

	}



	admissionRequest := admissionReview.Request

	if admissionRequest == nil {

		w.sendErrorResponse(rw, fmt.Errorf("admission request is nil"))

		return

	}



	// Check if this request should be excluded.

	if w.shouldExcludeRequest(admissionRequest) {

		w.sendAllowedResponse(rw, string(admissionRequest.UID))

		return

	}



	// Create admission event info.

	eventInfo := w.createAdmissionEventInfo(admissionRequest, webhookType)



	// Process the request based on webhook type.

	var response *admissionv1.AdmissionResponse

	var violations []PolicyViolation



	if webhookType == "validating" {

		response, violations = w.processValidatingRequest(admissionRequest)

	} else {

		response, violations = w.processMutatingRequest(admissionRequest)

	}



	// Create response info.

	processingTime := time.Since(startTime)

	responseInfo := w.createAdmissionResponseInfo(response, processingTime)



	// Create audit event.

	w.createAdmissionAuditEvent(eventInfo, responseInfo, violations, webhookType)



	// Send response.

	admissionReview.Response = response

	w.sendAdmissionResponse(rw, admissionReview)

}



// processValidatingRequest processes validating admission requests.

func (w *AdmissionAuditWebhook) processValidatingRequest(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, []PolicyViolation) {

	var violations []PolicyViolation



	// Check security policies if enforcement is enabled.

	if w.config.PolicyEnforcement != nil && w.config.PolicyEnforcement.Enabled {

		violations = w.checkSecurityPolicies(req)

	}



	// Create response.

	response := &admissionv1.AdmissionResponse{

		UID:     req.UID,

		Allowed: len(violations) == 0,

	}



	if len(violations) > 0 {

		// Denied due to policy violations.

		messages := make([]string, len(violations))

		for i, v := range violations {

			messages[i] = fmt.Sprintf("%s: %s", v.PolicyName, v.Violation)

		}



		response.Result = &metav1.Status{

			Code:    http.StatusForbidden,

			Reason:  metav1.StatusReasonForbidden,

			Message: fmt.Sprintf("Request denied due to policy violations: %s", strings.Join(messages, "; ")),

		}

	}



	return response, violations

}



// processMutatingRequest processes mutating admission requests.

func (w *AdmissionAuditWebhook) processMutatingRequest(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, []PolicyViolation) {

	// For mutating webhooks, we typically modify the request rather than deny it.

	response := &admissionv1.AdmissionResponse{

		UID:     req.UID,

		Allowed: true,

	}



	// Apply security mutations if needed.

	patches := w.generateSecurityPatches(req)

	if len(patches) > 0 {

		patchBytes, err := json.Marshal(patches)

		if err != nil {

			w.logger.Error(err, "Failed to marshal patches")

		} else {

			patchType := admissionv1.PatchTypeJSONPatch

			response.Patch = patchBytes

			response.PatchType = &patchType

		}

	}



	return response, []PolicyViolation{}

}



// checkSecurityPolicies checks various security policies.

func (w *AdmissionAuditWebhook) checkSecurityPolicies(req *admissionv1.AdmissionRequest) []PolicyViolation {

	var violations []PolicyViolation



	// Only check pods for now (can be extended to other resources).

	if req.Kind.Kind != "Pod" {

		return violations

	}



	// Decode the pod object.

	pod := &corev1.Pod{}

	if err := w.decodeObject(req.Object, pod); err != nil {

		w.logger.Error(err, "Failed to decode pod object")

		return violations

	}



	// Check privileged containers.

	if w.config.PolicyEnforcement.BlockPrivilegedContainers {

		violations = append(violations, w.checkPrivilegedContainers(pod)...)

	}



	// Check security context requirements.

	if w.config.PolicyEnforcement.RequireSecurityContext {

		violations = append(violations, w.checkSecurityContext(pod)...)

	}



	// Check read-only root filesystem.

	if w.config.PolicyEnforcement.RequireReadOnlyRootFilesystem {

		violations = append(violations, w.checkReadOnlyRootFilesystem(pod)...)

	}



	// Check host network.

	if w.config.PolicyEnforcement.BlockHostNetwork && pod.Spec.HostNetwork {

		violations = append(violations, PolicyViolation{

			PolicyName:  "block-host-network",

			Violation:   "hostNetwork is not allowed",

			Details:     "Pod attempts to use host networking",

			Severity:    "high",

			Remediation: "Set hostNetwork to false or remove the field",

		})

	}



	// Check host PID.

	if w.config.PolicyEnforcement.BlockHostPID && pod.Spec.HostPID {

		violations = append(violations, PolicyViolation{

			PolicyName:  "block-host-pid",

			Violation:   "hostPID is not allowed",

			Details:     "Pod attempts to use host PID namespace",

			Severity:    "high",

			Remediation: "Set hostPID to false or remove the field",

		})

	}



	// Check container registries.

	if len(w.config.PolicyEnforcement.AllowedRegistries) > 0 {

		violations = append(violations, w.checkAllowedRegistries(pod)...)

	}



	// Check required labels.

	if len(w.config.PolicyEnforcement.RequiredLabels) > 0 {

		violations = append(violations, w.checkRequiredLabels(pod)...)

	}



	// Check forbidden annotations.

	if len(w.config.PolicyEnforcement.ForbiddenAnnotations) > 0 {

		violations = append(violations, w.checkForbiddenAnnotations(pod)...)

	}



	return violations

}



// checkPrivilegedContainers checks for privileged containers.

func (w *AdmissionAuditWebhook) checkPrivilegedContainers(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)

	for _, container := range allContainers {

		if container.SecurityContext != nil &&

			container.SecurityContext.Privileged != nil &&

			*container.SecurityContext.Privileged {

			violations = append(violations, PolicyViolation{

				PolicyName:  "block-privileged-containers",

				Violation:   fmt.Sprintf("Container '%s' is privileged", container.Name),

				Details:     "Privileged containers have unrestricted access to host resources",

				Severity:    "critical",

				Remediation: "Set privileged to false in the container security context",

				Metadata:    map[string]string{"container": container.Name},

			})

		}

	}



	return violations

}



// checkSecurityContext checks for required security context.

func (w *AdmissionAuditWebhook) checkSecurityContext(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)

	for _, container := range allContainers {

		if container.SecurityContext == nil {

			violations = append(violations, PolicyViolation{

				PolicyName:  "require-security-context",

				Violation:   fmt.Sprintf("Container '%s' missing security context", container.Name),

				Details:     "All containers must have a security context defined",

				Severity:    "medium",

				Remediation: "Add securityContext to the container specification",

				Metadata:    map[string]string{"container": container.Name},

			})

		}

	}



	return violations

}



// checkReadOnlyRootFilesystem checks for read-only root filesystem.

func (w *AdmissionAuditWebhook) checkReadOnlyRootFilesystem(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)

	for _, container := range allContainers {

		if container.SecurityContext == nil ||

			container.SecurityContext.ReadOnlyRootFilesystem == nil ||

			!*container.SecurityContext.ReadOnlyRootFilesystem {

			violations = append(violations, PolicyViolation{

				PolicyName:  "require-readonly-root-filesystem",

				Violation:   fmt.Sprintf("Container '%s' does not have read-only root filesystem", container.Name),

				Details:     "Containers should use read-only root filesystems for security",

				Severity:    "medium",

				Remediation: "Set readOnlyRootFilesystem to true in the container security context",

				Metadata:    map[string]string{"container": container.Name},

			})

		}

	}



	return violations

}



// checkAllowedRegistries checks if container images are from allowed registries.

func (w *AdmissionAuditWebhook) checkAllowedRegistries(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)

	for _, container := range allContainers {

		allowed := false

		for _, allowedRegistry := range w.config.PolicyEnforcement.AllowedRegistries {

			if strings.HasPrefix(container.Image, allowedRegistry) {

				allowed = true

				break

			}

		}



		if !allowed {

			violations = append(violations, PolicyViolation{

				PolicyName:  "allowed-registries",

				Violation:   fmt.Sprintf("Container '%s' uses disallowed registry", container.Name),

				Details:     fmt.Sprintf("Image '%s' is not from an allowed registry", container.Image),

				Severity:    "high",

				Remediation: fmt.Sprintf("Use images from allowed registries: %s", strings.Join(w.config.PolicyEnforcement.AllowedRegistries, ", ")),

				Metadata:    map[string]string{"container": container.Name, "image": container.Image},

			})

		}

	}



	return violations

}



// checkRequiredLabels checks for required labels.

func (w *AdmissionAuditWebhook) checkRequiredLabels(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	for key, expectedValue := range w.config.PolicyEnforcement.RequiredLabels {

		if actualValue, exists := pod.Labels[key]; !exists {

			violations = append(violations, PolicyViolation{

				PolicyName:  "required-labels",

				Violation:   fmt.Sprintf("Required label '%s' is missing", key),

				Details:     "Pod must have all required labels",

				Severity:    "medium",

				Remediation: fmt.Sprintf("Add label '%s: %s' to the pod", key, expectedValue),

				Metadata:    map[string]string{"missing_label": key, "expected_value": expectedValue},

			})

		} else if expectedValue != "" && actualValue != expectedValue {

			violations = append(violations, PolicyViolation{

				PolicyName:  "required-labels",

				Violation:   fmt.Sprintf("Label '%s' has incorrect value", key),

				Details:     fmt.Sprintf("Expected '%s', got '%s'", expectedValue, actualValue),

				Severity:    "medium",

				Remediation: fmt.Sprintf("Set label '%s' to '%s'", key, expectedValue),

				Metadata:    map[string]string{"label": key, "expected_value": expectedValue, "actual_value": actualValue},

			})

		}

	}



	return violations

}



// checkForbiddenAnnotations checks for forbidden annotations.

func (w *AdmissionAuditWebhook) checkForbiddenAnnotations(pod *corev1.Pod) []PolicyViolation {

	var violations []PolicyViolation



	for _, forbiddenAnnotation := range w.config.PolicyEnforcement.ForbiddenAnnotations {

		if _, exists := pod.Annotations[forbiddenAnnotation]; exists {

			violations = append(violations, PolicyViolation{

				PolicyName:  "forbidden-annotations",

				Violation:   fmt.Sprintf("Forbidden annotation '%s' is present", forbiddenAnnotation),

				Details:     "Pod contains annotations that are not allowed",

				Severity:    "medium",

				Remediation: fmt.Sprintf("Remove annotation '%s' from the pod", forbiddenAnnotation),

				Metadata:    map[string]string{"forbidden_annotation": forbiddenAnnotation},

			})

		}

	}



	return violations

}



// generateSecurityPatches generates JSON patches for security mutations.

func (w *AdmissionAuditWebhook) generateSecurityPatches(req *admissionv1.AdmissionRequest) []map[string]interface{} {

	// This is a placeholder for mutating webhook patches.

	// In practice, you would generate appropriate JSON patch operations.

	var patches []map[string]interface{}



	// Example: Add security labels.

	patches = append(patches, map[string]interface{}{

		"op":    "add",

		"path":  "/metadata/labels/security.nephoran.io~1audited",

		"value": "true",

	})



	return patches

}



// Helper methods.



func (w *AdmissionAuditWebhook) createAdmissionEventInfo(req *admissionv1.AdmissionRequest, webhookType string) *AdmissionEventInfo {

	return &AdmissionEventInfo{

		UID:         string(req.UID),

		Kind:        req.Kind,

		Resource:    req.Resource,

		SubResource: req.SubResource,

		Name:        req.Name,

		Namespace:   req.Namespace,

		Operation:   req.Operation,

		UserInfo: &AdmissionUserInfo{

			Username: req.UserInfo.Username,

			UID:      req.UserInfo.UID,

			Groups:   req.UserInfo.Groups,

			Extra:    convertExtraValues(req.UserInfo.Extra),

		},

		Object:        req.Object,

		OldObject:     req.OldObject,

		Patch:         nil, // Patch field doesn't exist in AdmissionRequest

		PatchType:     nil, // PatchType field doesn't exist in AdmissionRequest

		Options:       req.Options,

		DryRun:        req.DryRun != nil && *req.DryRun,

		AdmissionTime: time.Now().UTC(),

	}

}



func (w *AdmissionAuditWebhook) createAdmissionResponseInfo(resp *admissionv1.AdmissionResponse, processingTime time.Duration) *AdmissionResponseInfo {

	return &AdmissionResponseInfo{

		UID:              string(resp.UID),

		Allowed:          resp.Allowed,

		Result:           resp.Result,

		Patch:            resp.Patch,

		PatchType:        resp.PatchType,

		Warnings:         resp.Warnings,

		AuditAnnotations: resp.AuditAnnotations,

		ProcessingTime:   processingTime,

	}

}



func (w *AdmissionAuditWebhook) createAdmissionAuditEvent(eventInfo *AdmissionEventInfo, responseInfo *AdmissionResponseInfo, violations []PolicyViolation, webhookType string) {

	// Determine event type and severity.

	eventType := EventTypeAuthorization

	if eventInfo.Operation == admissionv1.Create {

		eventType = EventTypeDataCreate

	} else if eventInfo.Operation == admissionv1.Update {

		eventType = EventTypeDataUpdate

	} else if eventInfo.Operation == admissionv1.Delete {

		eventType = EventTypeDataDelete

	}



	severity := SeverityInfo

	result := ResultSuccess



	if !responseInfo.Allowed {

		result = ResultDenied

		severity = SeverityWarning



		// Check for critical violations.

		for _, violation := range violations {

			if violation.Severity == "critical" {

				severity = SeverityCritical

				break

			} else if violation.Severity == "high" && severity < SeverityError {

				severity = SeverityError

			}

		}

	}



	// Create audit event.

	event := NewEventBuilder().

		WithEventType(eventType).

		WithSeverity(severity).

		WithComponent("admission-controller").

		WithAction(strings.ToLower(string(eventInfo.Operation))).

		WithDescription(fmt.Sprintf("%s admission %s for %s/%s",

			webhookType, eventInfo.Operation, eventInfo.Kind.Kind, eventInfo.Name)).

		WithResult(result).

		WithUser(eventInfo.UserInfo.Username, eventInfo.UserInfo.Username).

		WithResource(eventInfo.Kind.Kind, eventInfo.Name, string(eventInfo.Operation)).

		WithData("admission_info", eventInfo).

		WithData("response_info", responseInfo).

		WithData("webhook_type", webhookType).

		Build()



	// Add namespace context.

	if eventInfo.Namespace != "" {

		if event.SystemContext == nil {

			event.SystemContext = &SystemContext{}

		}

		event.SystemContext.Namespace = eventInfo.Namespace

	}



	// Add violations if any.

	if len(violations) > 0 {

		event.Data["violations"] = violations

		event.Data["violation_count"] = len(violations)

	}



	// Log the event.

	if err := w.auditSystem.LogEvent(event); err != nil {

		w.logger.Error(err, "Failed to log admission audit event",

			"request_uid", eventInfo.UID,

			"resource", eventInfo.Kind.Kind,

			"operation", eventInfo.Operation)

	}

}



func (w *AdmissionAuditWebhook) shouldExcludeRequest(req *admissionv1.AdmissionRequest) bool {

	// Check excluded namespaces.

	for _, excludedNS := range w.config.ExcludeNamespaces {

		if req.Namespace == excludedNS {

			return true

		}

	}



	// Check excluded resources.

	resourceKey := fmt.Sprintf("%s/%s", req.Resource.Group, req.Resource.Resource)

	for _, excludedResource := range w.config.ExcludeResources {

		if resourceKey == excludedResource || req.Resource.Resource == excludedResource {

			return true

		}

	}



	return false

}



func (w *AdmissionAuditWebhook) decodeAdmissionReview(req *http.Request, ar *admissionv1.AdmissionReview) error {

	decoder := w.codecs.UniversalDeserializer()

	body, err := io.ReadAll(req.Body)

	if err != nil {

		return fmt.Errorf("failed to read request body: %w", err)

	}



	_, _, err = decoder.Decode(body, nil, ar)

	return err

}



func (w *AdmissionAuditWebhook) decodeObject(rawExtension runtime.RawExtension, obj runtime.Object) error {

	decoder := w.codecs.UniversalDeserializer()

	_, _, err := decoder.Decode(rawExtension.Raw, nil, obj)

	return err

}



func (w *AdmissionAuditWebhook) sendAdmissionResponse(rw http.ResponseWriter, ar *admissionv1.AdmissionReview) {

	rw.Header().Set("Content-Type", "application/json")

	json.NewEncoder(rw).Encode(ar)

}



func (w *AdmissionAuditWebhook) sendAllowedResponse(rw http.ResponseWriter, uid string) {

	response := &admissionv1.AdmissionReview{

		TypeMeta: metav1.TypeMeta{

			APIVersion: "admission.k8s.io/v1",

			Kind:       "AdmissionReview",

		},

		Response: &admissionv1.AdmissionResponse{

			UID:     types.UID(uid),

			Allowed: true,

		},

	}

	w.sendAdmissionResponse(rw, response)

}



func (w *AdmissionAuditWebhook) sendErrorResponse(rw http.ResponseWriter, err error) {

	http.Error(rw, err.Error(), http.StatusInternalServerError)

}



// DefaultAdmissionAuditConfig returns a default admission audit configuration.

func DefaultAdmissionAuditConfig() *AdmissionAuditConfig {

	return &AdmissionAuditConfig{

		Enabled:              true,

		AuditLevel:           "standard",

		CaptureRequestObject: false,

		CaptureOldObject:     false,

		CapturePatch:         false,

		SensitiveResources:   strings.Split(SensitiveResources, ","),

		ExcludeNamespaces:    []string{"kube-system", "kube-public", "kube-node-lease"},

		ExcludeResources:     []string{"events", "leases"},

		HighRiskOperations:   strings.Split(HighRiskOperations, ","),

		AlertOnDenial:        true,

		PolicyEnforcement: &PolicyEnforcementConfig{

			Enabled:                       false,

			RequireSecurityContext:        false,

			BlockPrivilegedContainers:     true,

			RequireReadOnlyRootFilesystem: false,

			BlockHostNetwork:              true,

			BlockHostPID:                  true,

			AllowedRegistries:             []string{},

			RequiredLabels:                make(map[string]string),

			ForbiddenAnnotations:          []string{},

		},

	}

}



// isSensitiveResource checks if a resource is considered sensitive.

func (w *AdmissionAuditWebhook) isSensitiveResource(resource string) bool {

	for _, sensitive := range w.config.SensitiveResources {

		if resource == sensitive {

			return true

		}

	}

	return false

}



// isHighRiskOperation checks if an operation is considered high risk.

func (w *AdmissionAuditWebhook) isHighRiskOperation(operation admissionv1.Operation) bool {

	for _, highRisk := range w.config.HighRiskOperations {

		if string(operation) == highRisk {

			return true

		}

	}

	return false

}



// convertExtraValues converts authentication v1 ExtraValue to []string.

func convertExtraValues(extra map[string]authenticationv1.ExtraValue) map[string][]string {

	result := make(map[string][]string)

	for k, v := range extra {

		result[k] = []string(v)

	}

	return result

}

