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




package webhooks



import (

	"context"

	"fmt"

	"net/http"

	"regexp"

	"strings"

	"unicode"



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"



	admissionv1 "k8s.io/api/admission/v1"



	ctrl "sigs.k8s.io/controller-runtime"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

)



// log is for logging in this package.

var networkIntentWebhookLog = logf.Log.WithName("networkintent-webhook")



// NetworkIntentValidator provides validation logic for NetworkIntent resources.

// +kubebuilder:webhook:path=/validate-nephoran-io-v1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=nephoran.io,resources=networkintents,verbs=create;update,versions=v1,name=vnetworkintent.kb.io,admissionReviewVersions=v1.



// NetworkIntentValidator represents a networkintentvalidator.

type NetworkIntentValidator struct {

	decoder admission.Decoder

}



// TelecomKeywords contains telecommunications-related keywords that validate the intent is relevant.

var TelecomKeywords = []string{

	// 5G Core components.

	"amf", "smf", "upf", "nssf", "nrf", "udm", "udr", "ausf", "pcf", "bsf",

	"nef", "chf", "scp", "sepp", "udsf", "unef", "tngf", "twif", "n3iwf",



	// O-RAN components.

	"oran", "o-ran", "ran", "ric", "near-rt-ric", "non-rt-ric", "o-du",

	"o-cu", "o-ru", "e2", "a1", "o1", "o2", "smo", "xapp", "rapp",



	// Network functions and services.

	"vnf", "cnf", "nf", "network function", "network slice", "slicing",

	"qos", "quality of service", "sla", "service level", "urllc", "embb", "mmtc",



	// Infrastructure and deployment terms.

	"deployment", "service", "pod", "container", "kubernetes", "k8s",

	"cluster", "namespace", "ingress", "load balancer", "scaling", "autoscaling",

	"high availability", "ha", "failover", "redundancy", "backup",



	// Performance and optimization.

	"latency", "throughput", "bandwidth", "performance", "optimization",

	"monitoring", "metrics", "alerting", "logging", "observability",



	// Security and compliance.

	"security", "authentication", "authorization", "encryption", "tls", "mtls",

	"certificate", "rbac", "policy", "compliance", "audit",



	// Network and connectivity.

	"network", "connectivity", "routing", "switching", "gateway", "proxy",

	"firewall", "dns", "dhcp", "vlan", "subnet", "cidr", "ip", "ipv4", "ipv6",



	// Cloud native and orchestration.

	"helm", "chart", "yaml", "manifest", "config", "configuration",

	"template", "operator", "controller", "crd", "custom resource",

}



// Note: SecurityPatterns variable has been removed as the new restricted character set.

// prevents most injection attacks at the CRD validation level. The validateSecurity.

// function now checks for suspicious patterns within the allowed character set.



// ComplexityRules defines validation rules for intent complexity.

type ComplexityRules struct {

	MaxWords              int

	MaxSentences          int

	MaxConsecutiveRepeats int

	MinTelecomKeywords    int

}



// DefaultComplexityRules holds defaultcomplexityrules value.

var DefaultComplexityRules = ComplexityRules{

	MaxWords:              300, // Maximum number of words

	MaxSentences:          20,  // Maximum number of sentences

	MaxConsecutiveRepeats: 5,   // Maximum consecutive repeated words/phrases

	MinTelecomKeywords:    1,   // Minimum telecom-related keywords required

}



// NewNetworkIntentValidator creates a new NetworkIntent validator.

func NewNetworkIntentValidator() *NetworkIntentValidator {

	return &NetworkIntentValidator{}

}



// Handle processes admission requests for NetworkIntent validation.

func (v *NetworkIntentValidator) Handle(ctx context.Context, req admission.Request) admission.Response {

	logger := networkIntentWebhookLog.WithValues("namespace", req.Namespace, "name", req.Name)

	logger.Info("Processing NetworkIntent validation request", "operation", req.Operation)



	networkIntent := &nephoranv1.NetworkIntent{}



	// Decode the object based on operation type.

	var err error

	if req.Operation == admissionv1.Delete {

		// For delete operations, we generally allow them.

		logger.Info("Delete operation - allowing request")

		return admission.Allowed("Delete operations are allowed")

	}



	if req.Object.Raw != nil {

		err = v.decoder.Decode(req, networkIntent)

	} else {

		logger.Error(nil, "No object found in admission request")

		return admission.Errored(http.StatusBadRequest, fmt.Errorf("no object found in request"))

	}



	if err != nil {

		logger.Error(err, "Failed to decode NetworkIntent object")

		return admission.Errored(http.StatusBadRequest, err)

	}



	// Perform comprehensive validation.

	if validationErr := v.validateNetworkIntent(ctx, networkIntent); validationErr != nil {

		logger.Info("NetworkIntent validation failed", "error", validationErr.Error())

		return admission.Denied(validationErr.Error())

	}



	logger.Info("NetworkIntent validation successful")

	return admission.Allowed("NetworkIntent validation passed")

}



// validateNetworkIntent performs comprehensive validation of NetworkIntent.

func (v *NetworkIntentValidator) validateNetworkIntent(ctx context.Context, ni *nephoranv1.NetworkIntent) error {

	logger := networkIntentWebhookLog.WithValues("namespace", ni.Namespace, "name", ni.Name)



	// 1. Basic intent content validation.

	if err := v.validateIntentContent(ni.Spec.Intent); err != nil {

		logger.Info("Intent content validation failed", "error", err.Error())

		return fmt.Errorf("intent content validation failed: %w", err)

	}



	// 2. Security validation - check for malicious patterns.

	if err := v.validateSecurity(ni.Spec.Intent); err != nil {

		logger.Info("Security validation failed", "error", err.Error())

		return fmt.Errorf("security validation failed: %w", err)

	}



	// 3. Telecommunications keywords validation.

	if err := v.validateTelecomRelevance(ni.Spec.Intent); err != nil {

		logger.Info("Telecom relevance validation failed", "error", err.Error())

		return fmt.Errorf("telecommunications relevance validation failed: %w", err)

	}



	// 4. Intent complexity and structure validation.

	if err := v.validateComplexity(ni.Spec.Intent); err != nil {

		logger.Info("Complexity validation failed", "error", err.Error())

		return fmt.Errorf("intent complexity validation failed: %w", err)

	}



	// 5. Business logic validation.

	if err := v.validateBusinessLogic(ctx, ni); err != nil {

		logger.Info("Business logic validation failed", "error", err.Error())

		return fmt.Errorf("business logic validation failed: %w", err)

	}



	return nil

}



// validateIntentContent performs basic intent content validation.

func (v *NetworkIntentValidator) validateIntentContent(intent string) error {

	// Check if intent is empty or only whitespace.

	if strings.TrimSpace(intent) == "" {

		return fmt.Errorf("intent cannot be empty or only whitespace")

	}



	// SECURITY: Enforce strict character allowlist matching CRD pattern.

	// Only allow: a-zA-Z0-9, spaces, and safe punctuation: - _ . , ; : ( ) [ ].

	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	if !allowedChars.MatchString(intent) {

		// Find the first invalid character for better error reporting.

		for i, r := range intent {

			if !isAllowedChar(r) {

				return fmt.Errorf("intent contains disallowed character at position %d: %q - only alphanumeric, spaces, and safe punctuation (- _ . , ; : ( ) [ ]) are allowed", i, r)

			}

		}

		return fmt.Errorf("intent contains disallowed characters - only alphanumeric, spaces, and safe punctuation are allowed")

	}



	// Additional length check (redundant with CRD but provides defense in depth).

	if len(intent) > 1000 {

		return fmt.Errorf("intent exceeds maximum length of 1000 characters (got %d)", len(intent))

	}



	// Check for suspicious patterns even within allowed characters.

	// Prevent repeated punctuation that might indicate obfuscation attempts.

	punctuationRepeats := regexp.MustCompile(`[.,;:(){}\[\]]{5,}`)

	if punctuationRepeats.MatchString(intent) {

		return fmt.Errorf("intent contains excessive repeated punctuation")

	}



	// Check for control characters (should not be present with our allowlist, but defense in depth).

	for i, r := range intent {

		if unicode.IsControl(r) && !unicode.IsSpace(r) {

			return fmt.Errorf("intent contains control character at position %d", i)

		}

	}



	return nil

}



// isAllowedChar checks if a rune is in our allowed character set.

func isAllowedChar(r rune) bool {

	// Alphanumeric.

	if unicode.IsLetter(r) || unicode.IsDigit(r) {

		return true

	}

	// Allowed punctuation and spaces.

	allowed := " -_.,;:()[]"

	return strings.ContainsRune(allowed, r)

}



// validateSecurity checks for potentially malicious patterns.

func (v *NetworkIntentValidator) validateSecurity(intent string) error {

	// SECURITY: With our restricted character set, many injection attacks are already prevented.

	// This function provides additional defense-in-depth by checking for suspicious patterns.

	// even within the allowed character set.



	// Check for patterns that might indicate obfuscation attempts.

	// Even with restricted chars, check for suspicious word patterns.

	suspiciousPatterns := []string{

		// SQL-like patterns (even without quotes).

		"drop table", "delete from", "insert into", "select from",

		"union select", "exec sp", "exec xp",

		// Command-like patterns.

		"rm -rf", "cat etc", "wget http", "curl http",

		// Script-like patterns.

		"script type", "javascript void", "onload equals", "onerror equals",

		// Path traversal attempts.

		"dot dot slash", "dot dot backslash",

	}



	intentLower := strings.ToLower(intent)

	for _, pattern := range suspiciousPatterns {

		if strings.Contains(intentLower, pattern) {

			return fmt.Errorf("intent contains suspicious pattern that might indicate an attack attempt: %s", pattern)

		}

	}



	// Check for base64-like patterns (continuous alphanumeric strings).

	// Even without = or +/, long unbroken alphanumeric could be encoding.

	alphanumericPattern := regexp.MustCompile(`[A-Za-z0-9]{40,}`)

	if matches := alphanumericPattern.FindAllString(intent, -1); len(matches) > 0 {

		return fmt.Errorf("intent contains suspiciously long unbroken alphanumeric sequence (possible encoding)")

	}



	// Check for repeated patterns that might indicate fuzzing or automated attacks.

	// Look for unusual repetition of allowed punctuation.

	punctuationCounts := map[rune]int{

		'.': 0, ',': 0, ';': 0, ':': 0,

		'(': 0, ')': 0, '[': 0, ']': 0,

		'-': 0, '_': 0,

	}



	for _, r := range intent {

		if count, exists := punctuationCounts[r]; exists {

			punctuationCounts[r] = count + 1

		}

	}



	// Check for excessive use of any single punctuation mark.

	for char, count := range punctuationCounts {

		maxAllowed := 10

		// Allow more hyphens and underscores as they're common in technical terms.

		if char == '-' || char == '_' {

			maxAllowed = 20

		}

		if count > maxAllowed {

			return fmt.Errorf("intent contains excessive use of '%c' (%d occurrences, max %d)", char, count, maxAllowed)

		}

	}



	// Check for patterns that look like attempts to bypass validation.

	// e.g., "d r o p  t a b l e" (spaced out malicious commands).

	spacedPatterns := []string{

		"d r o p", "s e l e c t", "d e l e t e", "i n s e r t",

		"e x e c", "s c r i p t", "j a v a",

	}



	for _, pattern := range spacedPatterns {

		if strings.Contains(intentLower, pattern) {

			return fmt.Errorf("intent contains suspicious spaced pattern that might be attempting to bypass filters")

		}

	}



	return nil

}



// validateTelecomRelevance ensures the intent is relevant to telecommunications.

func (v *NetworkIntentValidator) validateTelecomRelevance(intent string) error {

	intentLower := strings.ToLower(intent)

	keywordCount := 0

	foundKeywords := make([]string, 0)



	// Check for telecommunications keywords.

	for _, keyword := range TelecomKeywords {

		if strings.Contains(intentLower, strings.ToLower(keyword)) {

			keywordCount++

			foundKeywords = append(foundKeywords, keyword)

		}

	}



	if keywordCount < DefaultComplexityRules.MinTelecomKeywords {

		return fmt.Errorf("intent does not appear to be telecommunications-related (found %d telecom keywords, minimum required: %d)",

			keywordCount, DefaultComplexityRules.MinTelecomKeywords)

	}



	networkIntentWebhookLog.V(1).Info("Telecommunications validation passed",

		"keywordCount", keywordCount,

		"foundKeywords", foundKeywords[:min(len(foundKeywords), 5)]) // Log first 5 keywords



	return nil

}



// validateComplexity checks the complexity and structure of the intent.

func (v *NetworkIntentValidator) validateComplexity(intent string) error {

	// Count words.

	words := strings.Fields(intent)

	if len(words) > DefaultComplexityRules.MaxWords {

		return fmt.Errorf("intent is too complex (%d words, maximum allowed: %d)",

			len(words), DefaultComplexityRules.MaxWords)

	}



	if len(words) == 0 {

		return fmt.Errorf("intent contains no recognizable words")

	}



	// Count sentences (approximate).

	sentences := strings.FieldsFunc(intent, func(r rune) bool {

		return r == '.' || r == '!' || r == '?'

	})

	if len(sentences) > DefaultComplexityRules.MaxSentences {

		return fmt.Errorf("intent contains too many sentences (%d, maximum allowed: %d)",

			len(sentences), DefaultComplexityRules.MaxSentences)

	}



	// Check for excessive repetition.

	if err := v.checkRepetition(words); err != nil {

		return err

	}



	// Check for reasonable word length distribution.

	if err := v.validateWordDistribution(words); err != nil {

		return err

	}



	return nil

}



// checkRepetition checks for excessive word repetition.

func (v *NetworkIntentValidator) checkRepetition(words []string) error {

	if len(words) < 2 {

		return nil

	}



	consecutiveRepeats := 1

	for i := 1; i < len(words); i++ {

		if strings.EqualFold(words[i], words[i-1]) {

			consecutiveRepeats++

			if consecutiveRepeats > DefaultComplexityRules.MaxConsecutiveRepeats {

				return fmt.Errorf("intent contains too many consecutive repeated words: '%s' (max allowed: %d)",

					words[i], DefaultComplexityRules.MaxConsecutiveRepeats)

			}

		} else {

			consecutiveRepeats = 1

		}

	}



	return nil

}



// validateWordDistribution checks for reasonable word length distribution.

func (v *NetworkIntentValidator) validateWordDistribution(words []string) error {

	if len(words) == 0 {

		return nil

	}



	// Check average word length.

	totalLength := 0

	veryLongWords := 0

	for _, word := range words {

		totalLength += len(word)

		if len(word) > 25 {

			veryLongWords++

		}

	}



	avgLength := float64(totalLength) / float64(len(words))

	if avgLength > 20.0 {

		return fmt.Errorf("intent has unusually long average word length: %.1f characters", avgLength)

	}



	if veryLongWords > len(words)/4 {

		return fmt.Errorf("intent contains too many very long words (%d words longer than 25 characters)", veryLongWords)

	}



	return nil

}



// validateBusinessLogic performs advanced business logic validation.

func (v *NetworkIntentValidator) validateBusinessLogic(ctx context.Context, ni *nephoranv1.NetworkIntent) error {

	// 1. Validate resource naming consistency.

	if err := v.validateResourceNaming(ni); err != nil {

		return err

	}



	// 2. Validate intent coherence and actionability.

	if err := v.validateIntentCoherence(ni.Spec.Intent); err != nil {

		return err

	}



	// 3. Validate against conflicting intents (if we had a way to query existing ones).

	// This would require additional context/client to check existing NetworkIntents.

	// For now, we'll skip this but it's a placeholder for future enhancement.



	return nil

}



// validateResourceNaming checks for naming consistency and best practices.

func (v *NetworkIntentValidator) validateResourceNaming(ni *nephoranv1.NetworkIntent) error {

	name := ni.Name



	// Basic Kubernetes naming validation (beyond what's already enforced by k8s).

	if len(name) > 63 {

		return fmt.Errorf("NetworkIntent name is too long (%d characters, max 63)", len(name))

	}



	// Check for meaningful naming.

	if len(name) < 3 {

		return fmt.Errorf("NetworkIntent name is too short (%d characters, minimum 3)", len(name))

	}



	// Check for descriptive naming patterns.

	namePattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	if !namePattern.MatchString(name) {

		return fmt.Errorf("NetworkIntent name does not follow Kubernetes naming conventions")

	}



	// Encourage meaningful names by checking for generic patterns to avoid.

	discouragedPatterns := []string{"test", "tmp", "temp", "foo", "bar", "example"}

	nameLower := strings.ToLower(name)

	for _, pattern := range discouragedPatterns {

		if strings.Contains(nameLower, pattern) {

			networkIntentWebhookLog.Info("NetworkIntent uses discouraged naming pattern",

				"name", name, "pattern", pattern)

			// We log but don't fail - it's just a recommendation.

		}

	}



	return nil

}



// validateIntentCoherence checks if the intent is coherent and actionable.

func (v *NetworkIntentValidator) validateIntentCoherence(intent string) error {

	intentLower := strings.ToLower(intent)



	// Check for action verbs that indicate actionable intent.

	actionVerbs := []string{

		"deploy", "create", "configure", "setup", "install", "provision",

		"scale", "update", "modify", "enable", "disable", "start", "stop",

		"implement", "establish", "initialize", "activate", "deactivate",

		"optimize", "tune", "adjust", "migrate", "backup", "restore",

	}



	hasActionVerb := false

	for _, verb := range actionVerbs {

		if strings.Contains(intentLower, verb) {

			hasActionVerb = true

			break

		}

	}



	if !hasActionVerb {

		return fmt.Errorf("intent does not contain actionable verbs - please specify what action should be performed")

	}



	// Check for contradictory statements.

	contradictions := [][]string{

		{"enable", "disable"},

		{"start", "stop"},

		{"create", "delete"},

		{"scale up", "scale down"},

		{"high availability", "single instance"},

		{"production", "development", "testing"},

	}



	for _, contradiction := range contradictions {

		foundTerms := make([]string, 0)

		for _, term := range contradiction {

			if strings.Contains(intentLower, term) {

				foundTerms = append(foundTerms, term)

			}

		}



		if len(foundTerms) > 1 {

			return fmt.Errorf("intent contains potentially contradictory terms: %v", foundTerms)

		}

	}



	// Check for reasonable specificity - not too vague.

	vaguePhrases := []string{

		"do something", "make it work", "fix it", "handle this", "deal with",

		"just do", "somehow", "whatever", "anything", "everything",

	}



	for _, phrase := range vaguePhrases {

		if strings.Contains(intentLower, phrase) {

			return fmt.Errorf("intent is too vague - please provide more specific requirements")

		}

	}



	return nil

}



// InjectDecoder injects the decoder into the validator.

func (v *NetworkIntentValidator) InjectDecoder(d admission.Decoder) error {

	v.decoder = d

	return nil

}



// min returns the minimum of two integers.

func min(a, b int) int {

	if a < b {

		return a

	}

	return b

}



// SetupNetworkIntentWebhookWithManager registers the NetworkIntent webhook with the manager.

func SetupNetworkIntentWebhookWithManager(mgr ctrl.Manager) error {

	validator := NewNetworkIntentValidator()



	mgr.GetWebhookServer().Register("/validate-nephoran-io-v1-networkintent",

		&admission.Webhook{Handler: validator})



	return nil

}

