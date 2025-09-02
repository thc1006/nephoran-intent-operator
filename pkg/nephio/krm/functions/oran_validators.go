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

package functions

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// ORANComplianceValidator validates O-RAN interface compliance.

type ORANComplianceValidator struct {
	*BaseFunctionImpl
}

// NewORANComplianceValidator creates a new O-RAN compliance validator.

func NewORANComplianceValidator() *ORANComplianceValidator {
	metadata := &FunctionMetadata{
		Name: "oran-compliance-validator",

		Version: "v1.0.0",

		Description: "Validates O-RAN interface compliance and configuration",

		Type: FunctionTypeValidator,

		Categories: []string{"o-ran", "validation", "compliance"},

		Keywords: []string{"o-ran", "a1", "o1", "o2", "e2", "compliance"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF", Operations: []string{"read"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "NearRTRIC", Operations: []string{"read"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "ODU", Operations: []string{"read"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "OCU", Operations: []string{"read"}},
		},

		Telecom: &TelecomProfile{
			Standards: []StandardCompliance{
				{Name: "O-RAN Alliance", Version: "R1.0", Required: true},

				{Name: "3GPP", Version: "R17", Required: true},
			},

			ORANInterfaces: []ORANInterfaceSupport{
				{Interface: "A1", Version: "v2.0", Role: "both", Required: true},

				{Interface: "O1", Version: "v1.0", Role: "both", Required: true},

				{Interface: "O2", Version: "v1.0", Role: "consumer", Required: false},

				{Interface: "E2", Version: "v2.0", Role: "consumer", Required: false},
			},

			NetworkFunctionTypes: []string{"AMF", "SMF", "UPF", "Near-RT RIC", "O-DU", "O-CU"},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"interfaces": {
				Type: "array",

				Description: "List of O-RAN interfaces to validate",

				Items: &SchemaProperty{
					Type: "string",

					Enum: []interface{}{"A1", "O1", "O2", "E2"},
				},
			},

			"strictMode": {
				Type: "boolean",

				Description: "Enable strict compliance checking",

				Default: false,
			},

			"skipWarnings": {
				Type: "boolean",

				Description: "Skip warnings and only report errors",

				Default: false,
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &ORANComplianceValidator{BaseFunctionImpl: base}
}

// Execute validates O-RAN compliance for network functions.

func (v *ORANComplianceValidator) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting O-RAN compliance validation", "resourceCount", len(input.Items))

	// Parse configuration.

	config := v.parseConfig(input.FunctionConfig)

	// Create output resource list.

	output := &ResourceList{
		Items: input.Items, // Pass through all resources

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	// Validate each resource.

	for i, resource := range input.Items {

		results := v.validateResource(ctx, &resource, config)

		output.Results = append(output.Results, results...)

		// Add compliance annotations.

		v.addComplianceAnnotations(&output.Items[i], results)

	}

	LogInfo(ctx, "O-RAN compliance validation completed",

		"resourceCount", len(output.Items),

		"validationResults", len(output.Results))

	return output, nil
}

// A1PolicyValidator validates A1 interface policy configurations.

type A1PolicyValidator struct {
	*BaseFunctionImpl
}

// NewA1PolicyValidator creates a new A1 policy validator.

func NewA1PolicyValidator() *A1PolicyValidator {
	metadata := &FunctionMetadata{
		Name: "a1-policy-validator",

		Version: "v1.0.0",

		Description: "Validates A1 interface policy configurations",

		Type: FunctionTypeValidator,

		Categories: []string{"o-ran", "a1", "policy", "validation"},

		Keywords: []string{"a1", "policy", "near-rt-ric", "xapp"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "o-ran.org", Version: "v1alpha1", Kind: "A1Policy", Operations: []string{"read"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "PolicyType", Operations: []string{"read"}},

			{Group: "o-ran.org", Version: "v1alpha1", Kind: "NearRTRIC", Operations: []string{"read"}},
		},

		Telecom: &TelecomProfile{
			ORANInterfaces: []ORANInterfaceSupport{
				{Interface: "A1", Version: "v2.0", Role: "both", Required: true},
			},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"policyTypes": {
				Type: "array",

				Description: "List of policy types to validate",

				Items: &SchemaProperty{Type: "string"},
			},

			"validateSyntax": {
				Type: "boolean",

				Description: "Validate policy syntax",

				Default: true,
			},

			"validateSemantics": {
				Type: "boolean",

				Description: "Validate policy semantics",

				Default: true,
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &A1PolicyValidator{BaseFunctionImpl: base}
}

// Execute validates A1 policy configurations.

func (v *A1PolicyValidator) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting A1 policy validation")

	output := &ResourceList{
		Items: input.Items,

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	// Find A1 policies.

	a1Policies := FindResourcesByGVK(input.Items, schema.GroupVersionKind{
		Group: "o-ran.org",

		Version: "v1alpha1",

		Kind: "A1Policy",
	})

	// Find policy types.

	policyTypes := FindResourcesByGVK(input.Items, schema.GroupVersionKind{
		Group: "o-ran.org",

		Version: "v1alpha1",

		Kind: "PolicyType",
	})

	// Build policy type registry.

	policyTypeRegistry := v.buildPolicyTypeRegistry(policyTypes)

	// Validate each A1 policy.

	for _, policy := range a1Policies {

		results := v.validateA1Policy(ctx, &policy, policyTypeRegistry)

		output.Results = append(output.Results, results...)

	}

	LogInfo(ctx, "A1 policy validation completed",

		"policies", len(a1Policies),

		"policyTypes", len(policyTypes),

		"results", len(output.Results))

	return output, nil
}

// FiveGCoreValidator validates 5G Core network function configurations.

type FiveGCoreValidator struct {
	*BaseFunctionImpl
}

// NewFiveGCoreValidator creates a new 5G Core validator.

func NewFiveGCoreValidator() *FiveGCoreValidator {
	metadata := &FunctionMetadata{
		Name: "5g-core-validator",

		Version: "v1.0.0",

		Description: "Validates 5G Core network function configurations",

		Type: FunctionTypeValidator,

		Categories: []string{"5g", "core", "validation"},

		Keywords: []string{"5g", "amf", "smf", "upf", "nssf", "core"},

		ResourceTypes: []ResourceTypeSupport{
			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "NSSF", Operations: []string{"read"}},

			{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "NRF", Operations: []string{"read"}},
		},

		Telecom: &TelecomProfile{
			Standards: []StandardCompliance{
				{Name: "3GPP", Version: "TS 23.501", Release: "R17", Required: true},

				{Name: "3GPP", Version: "TS 23.502", Release: "R17", Required: true},
			},

			NetworkFunctionTypes: []string{"AMF", "SMF", "UPF", "NSSF", "NRF"},

			FiveGCapabilities: []string{"5G-SA", "5G-NSA", "Network-Slicing"},
		},

		Author: "Nephoran Intent Operator",

		License: "Apache-2.0",
	}

	schema := &FunctionSchema{
		Type: "object",

		Properties: map[string]*SchemaProperty{
			"networkFunctions": {
				Type: "array",

				Description: "List of network functions to validate",

				Items: &SchemaProperty{
					Type: "string",

					Enum: []interface{}{"AMF", "SMF", "UPF", "NSSF", "NRF"},
				},
			},

			"validateInterfaces": {
				Type: "boolean",

				Description: "Validate network function interfaces",

				Default: true,
			},

			"validateCapacity": {
				Type: "boolean",

				Description: "Validate capacity planning",

				Default: false,
			},
		},
	}

	base := NewBaseFunctionImpl(metadata, schema)

	return &FiveGCoreValidator{BaseFunctionImpl: base}
}

// Execute validates 5G Core network function configurations.

func (v *FiveGCoreValidator) Execute(ctx context.Context, input *ResourceList) (*ResourceList, error) {
	LogInfo(ctx, "Starting 5G Core validation")

	output := &ResourceList{
		Items: input.Items,

		FunctionConfig: input.FunctionConfig,

		Results: []*porch.FunctionResult{},

		Context: input.Context,
	}

	// Validate each network function type.

	networkFunctions := []string{"AMF", "SMF", "UPF", "NSSF", "NRF"}

	for _, nfType := range networkFunctions {

		nfResources := FindResourcesByGVK(input.Items, schema.GroupVersionKind{
			Group: "workload.nephio.org",

			Version: "v1alpha1",

			Kind: nfType,
		})

		for _, nf := range nfResources {

			results := v.validateNetworkFunction(ctx, &nf, nfType)

			output.Results = append(output.Results, results...)

		}

	}

	// Cross-function validation.

	crossResults := v.validateNetworkFunctionInteractions(ctx, input.Items)

	output.Results = append(output.Results, crossResults...)

	LogInfo(ctx, "5G Core validation completed", "results", len(output.Results))

	return output, nil
}

// Helper methods.

func (v *ORANComplianceValidator) parseConfig(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return json.RawMessage("{}"),

			"strictMode": false,

			"skipWarnings": false,
		}
	}

	return config
}

func (v *ORANComplianceValidator) validateResource(ctx context.Context, resource *porch.KRMResource, config map[string]interface{}) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check if this is an O-RAN related resource.

	if !v.isORANResource(resource) {
		return results
	}

	// Validate interface compliance based on resource type.

	switch resource.Kind {

	case "AMF":

		results = append(results, v.validateAMFCompliance(ctx, resource)...)

	case "SMF":

		results = append(results, v.validateSMFCompliance(ctx, resource)...)

	case "UPF":

		results = append(results, v.validateUPFCompliance(ctx, resource)...)

	case "NearRTRIC":

		results = append(results, v.validateNearRTRICCompliance(ctx, resource)...)

	case "ODU":

		results = append(results, v.validateODUCompliance(ctx, resource)...)

	case "OCU":

		results = append(results, v.validateOCUCompliance(ctx, resource)...)

	default:

		results = append(results, CreateInfo(

			fmt.Sprintf("Resource %s (%s) is not a known O-RAN component", name, resource.Kind),

			map[string]string{"resource": name, "kind": resource.Kind},
		))

	}

	return results
}

func (v *ORANComplianceValidator) isORANResource(resource *porch.KRMResource) bool {
	oranGroups := []string{"o-ran.org", "workload.nephio.org"}

	oranKinds := []string{"AMF", "SMF", "UPF", "NearRTRIC", "ODU", "OCU", "A1Policy", "PolicyType"}

	// Check API group.

	for _, group := range oranGroups {
		if strings.Contains(resource.APIVersion, group) {
			return true
		}
	}

	// Check kind.

	for _, kind := range oranKinds {
		if resource.Kind == kind {
			return true
		}
	}

	return false
}

func (v *ORANComplianceValidator) validateAMFCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check required O1 interface configuration.

	o1Config, err := GetSpecField(resource, "interfaces.o1")

	if err != nil || o1Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("AMF %s missing O1 interface configuration", name),

			map[string]string{"resource": name, "interface": "O1"},
		))
	}

	// Check SBI interface configuration.

	sbiConfig, err := GetSpecField(resource, "interfaces.sbi")

	if err != nil || sbiConfig == nil {
		results = append(results, CreateError(

			fmt.Sprintf("AMF %s missing SBI interface configuration", name),

			map[string]string{"resource": name, "interface": "SBI"},
		))
	}

	// Check N1/N2 interface configuration.

	n1Config, err := GetSpecField(resource, "interfaces.n1")

	if err == nil && n1Config != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s has N1 interface configured", name),
		))
	}

	n2Config, err := GetSpecField(resource, "interfaces.n2")

	if err == nil && n2Config != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("AMF %s has N2 interface configured", name),
		))
	}

	return results
}

func (v *ORANComplianceValidator) validateSMFCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check required interfaces.

	requiredInterfaces := []string{"sbi", "n4", "n7"}

	for _, iface := range requiredInterfaces {

		ifaceConfig, err := GetSpecField(resource, fmt.Sprintf("interfaces.%s", iface))

		if err != nil || ifaceConfig == nil {
			results = append(results, CreateError(

				fmt.Sprintf("SMF %s missing %s interface configuration", name, strings.ToUpper(iface)),

				map[string]string{"resource": name, "interface": strings.ToUpper(iface)},
			))
		}

	}

	// Check PDU session handling configuration.

	pduConfig, err := GetSpecField(resource, "pduSessions")

	if err == nil && pduConfig != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("SMF %s has PDU session configuration", name),
		))
	}

	return results
}

func (v *ORANComplianceValidator) validateUPFCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check N3 interface (gNB connection).

	n3Config, err := GetSpecField(resource, "interfaces.n3")

	if err != nil || n3Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("UPF %s missing N3 interface configuration", name),

			map[string]string{"resource": name, "interface": "N3"},
		))
	}

	// Check N4 interface (SMF connection).

	n4Config, err := GetSpecField(resource, "interfaces.n4")

	if err != nil || n4Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("UPF %s missing N4 interface configuration", name),

			map[string]string{"resource": name, "interface": "N4"},
		))
	}

	// Check data plane configuration.

	dataPlaneConfig, err := GetSpecField(resource, "dataPlane")

	if err == nil && dataPlaneConfig != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("UPF %s has data plane configuration", name),
		))
	}

	return results
}

func (v *ORANComplianceValidator) validateNearRTRICCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check A1 interface.

	a1Config, err := GetSpecField(resource, "interfaces.a1")

	if err != nil || a1Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("Near-RT RIC %s missing A1 interface configuration", name),

			map[string]string{"resource": name, "interface": "A1"},
		))
	}

	// Check E2 interface.

	e2Config, err := GetSpecField(resource, "interfaces.e2")

	if err != nil || e2Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("Near-RT RIC %s missing E2 interface configuration", name),

			map[string]string{"resource": name, "interface": "E2"},
		))
	}

	// Check xApp platform configuration.

	xAppConfig, err := GetSpecField(resource, "xAppPlatform")

	if err == nil && xAppConfig != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("Near-RT RIC %s has xApp platform configured", name),
		))
	}

	return results
}

func (v *ORANComplianceValidator) validateODUCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check Open Fronthaul interface.

	fronthaulConfig, err := GetSpecField(resource, "interfaces.fronthaul")

	if err != nil || fronthaulConfig == nil {
		results = append(results, CreateWarning(

			fmt.Sprintf("O-DU %s missing Open Fronthaul interface configuration", name),

			map[string]string{"resource": name, "interface": "Fronthaul"},
		))
	}

	// Check F1 interface (to O-CU).

	f1Config, err := GetSpecField(resource, "interfaces.f1")

	if err != nil || f1Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("O-DU %s missing F1 interface configuration", name),

			map[string]string{"resource": name, "interface": "F1"},
		))
	}

	return results
}

func (v *ORANComplianceValidator) validateOCUCompliance(ctx context.Context, resource *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(resource)

	// Check F1 interface (from O-DU).

	f1Config, err := GetSpecField(resource, "interfaces.f1")

	if err != nil || f1Config == nil {
		results = append(results, CreateError(

			fmt.Sprintf("O-CU %s missing F1 interface configuration", name),

			map[string]string{"resource": name, "interface": "F1"},
		))
	}

	// Check NG interface (to 5G Core).

	ngConfig, err := GetSpecField(resource, "interfaces.ng")

	if err != nil || ngConfig == nil {
		results = append(results, CreateError(

			fmt.Sprintf("O-CU %s missing NG interface configuration", name),

			map[string]string{"resource": name, "interface": "NG"},
		))
	}

	return results
}

func (v *ORANComplianceValidator) addComplianceAnnotations(resource *porch.KRMResource, results []*porch.FunctionResult) {
	errorCount := 0

	warningCount := 0

	for _, result := range results {
		switch result.Severity {

		case "error":

			errorCount++

		case "warning":

			warningCount++

		}
	}

	// Add compliance status annotation.

	complianceStatus := "compliant"

	if errorCount > 0 {
		complianceStatus = "non-compliant"
	} else if warningCount > 0 {
		complianceStatus = "partially-compliant"
	}

	SetResourceAnnotation(resource, "o-ran.org/compliance-status", complianceStatus)

	SetResourceAnnotation(resource, "o-ran.org/validation-errors", strconv.Itoa(errorCount))

	SetResourceAnnotation(resource, "o-ran.org/validation-warnings", strconv.Itoa(warningCount))

	SetResourceAnnotation(resource, "o-ran.org/validated-at", fmt.Sprintf("%d", time.Now().Unix()))
}

// A1 Policy validation helpers.

func (v *A1PolicyValidator) buildPolicyTypeRegistry(policyTypes []porch.KRMResource) map[string]*porch.KRMResource {
	registry := make(map[string]*porch.KRMResource)

	for _, pt := range policyTypes {
		if name, err := GetResourceName(&pt); err == nil {

			ptCopy := pt

			registry[name] = &ptCopy

		}
	}

	return registry
}

func (v *A1PolicyValidator) validateA1Policy(ctx context.Context, policy *porch.KRMResource, registry map[string]*porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(policy)

	// Check policy type reference.

	policyType, err := GetSpecField(policy, "policyType")

	if err != nil || policyType == nil {

		results = append(results, CreateError(

			fmt.Sprintf("A1 Policy %s missing policyType reference", name),

			map[string]string{"resource": name, "field": "spec.policyType"},
		))

		return results

	}

	policyTypeName, ok := policyType.(string)

	if !ok {

		results = append(results, CreateError(

			fmt.Sprintf("A1 Policy %s has invalid policyType reference", name),

			map[string]string{"resource": name, "field": "spec.policyType"},
		))

		return results

	}

	// Check if policy type exists.

	if _, exists := registry[policyTypeName]; !exists {
		results = append(results, CreateError(

			fmt.Sprintf("A1 Policy %s references non-existent policy type %s", name, policyTypeName),

			map[string]string{"resource": name, "policyType": policyTypeName},
		))
	} else {
		results = append(results, CreateInfo(

			fmt.Sprintf("A1 Policy %s references valid policy type %s", name, policyTypeName),
		))
	}

	// Validate policy syntax.

	policyData, err := GetSpecField(policy, "policyData")

	if err != nil || policyData == nil {
		results = append(results, CreateError(

			fmt.Sprintf("A1 Policy %s missing policy data", name),

			map[string]string{"resource": name, "field": "spec.policyData"},
		))
	}

	return results
}

// 5G Core validation helpers.

func (v *FiveGCoreValidator) validateNetworkFunction(ctx context.Context, nf *porch.KRMResource, nfType string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	switch nfType {

	case "AMF":

		results = append(results, v.validateAMF(ctx, nf)...)

	case "SMF":

		results = append(results, v.validateSMF(ctx, nf)...)

	case "UPF":

		results = append(results, v.validateUPF(ctx, nf)...)

	case "NSSF":

		results = append(results, v.validateNSSF(ctx, nf)...)

	case "NRF":

		results = append(results, v.validateNRF(ctx, nf)...)

	}

	// Common validations for all NFs.

	results = append(results, v.validateCommonNF(ctx, nf, nfType)...)

	return results
}

func (v *FiveGCoreValidator) validateAMF(ctx context.Context, amf *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(amf)

	// Check GUAMI configuration.

	guami, err := GetSpecField(amf, "guami")

	if err != nil || guami == nil {
		results = append(results, CreateError(

			fmt.Sprintf("AMF %s missing GUAMI configuration", name),

			map[string]string{"resource": name, "field": "spec.guami"},
		))
	}

	// Check served PLMN list.

	plmnList, err := GetSpecField(amf, "servedGuamiList")

	if err != nil || plmnList == nil {
		results = append(results, CreateWarning(

			fmt.Sprintf("AMF %s missing served GUAMI list", name),

			map[string]string{"resource": name, "field": "spec.servedGuamiList"},
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateSMF(ctx context.Context, smf *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(smf)

	// Check UE subnet configuration.

	ueSubnet, err := GetSpecField(smf, "ueSubnet")

	if err != nil || ueSubnet == nil {
		results = append(results, CreateWarning(

			fmt.Sprintf("SMF %s missing UE subnet configuration", name),

			map[string]string{"resource": name, "field": "spec.ueSubnet"},
		))
	}

	// Check DNN configuration.

	dnnConfig, err := GetSpecField(smf, "dnn")

	if err != nil || dnnConfig == nil {
		results = append(results, CreateError(

			fmt.Sprintf("SMF %s missing DNN configuration", name),

			map[string]string{"resource": name, "field": "spec.dnn"},
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateUPF(ctx context.Context, upf *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(upf)

	// Check GTPU configuration.

	gtpuConfig, err := GetSpecField(upf, "gtpu")

	if err != nil || gtpuConfig == nil {
		results = append(results, CreateError(

			fmt.Sprintf("UPF %s missing GTPU configuration", name),

			map[string]string{"resource": name, "field": "spec.gtpu"},
		))
	}

	// Check PFCP configuration.

	pfcpConfig, err := GetSpecField(upf, "pfcp")

	if err != nil || pfcpConfig == nil {
		results = append(results, CreateError(

			fmt.Sprintf("UPF %s missing PFCP configuration", name),

			map[string]string{"resource": name, "field": "spec.pfcp"},
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateNSSF(ctx context.Context, nssf *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(nssf)

	// Check NSI configuration.

	nsiConfig, err := GetSpecField(nssf, "nsi")

	if err != nil || nsiConfig == nil {
		results = append(results, CreateWarning(

			fmt.Sprintf("NSSF %s missing NSI configuration", name),

			map[string]string{"resource": name, "field": "spec.nsi"},
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateNRF(ctx context.Context, nrf *porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(nrf)

	// Check OAuth configuration.

	oauthConfig, err := GetSpecField(nrf, "oauth")

	if err == nil && oauthConfig != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("NRF %s has OAuth configuration", name),
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateCommonNF(ctx context.Context, nf *porch.KRMResource, nfType string) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	name, _ := GetResourceName(nf)

	// Check NF instance ID.

	instanceID, err := GetSpecField(nf, "nfInstanceId")

	if err != nil || instanceID == nil {
		results = append(results, CreateWarning(

			fmt.Sprintf("%s %s missing NF instance ID", nfType, name),

			map[string]string{"resource": name, "field": "spec.nfInstanceId"},
		))
	} else {
		// Validate UUID format.

		if instanceIDStr, ok := instanceID.(string); ok {
			if !v.isValidUUID(instanceIDStr) {
				results = append(results, CreateError(

					fmt.Sprintf("%s %s has invalid NF instance ID format", nfType, name),

					map[string]string{"resource": name, "field": "spec.nfInstanceId"},
				))
			}
		}
	}

	// Check heartbeat configuration.

	heartbeat, err := GetSpecField(nf, "heartbeat")

	if err == nil && heartbeat != nil {
		results = append(results, CreateInfo(

			fmt.Sprintf("%s %s has heartbeat configuration", nfType, name),
		))
	}

	return results
}

func (v *FiveGCoreValidator) validateNetworkFunctionInteractions(ctx context.Context, resources []porch.KRMResource) []*porch.FunctionResult {
	var results []*porch.FunctionResult

	// Count network functions.

	amfCount := len(FindResourcesByGVK(resources, schema.GroupVersionKind{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "AMF"}))

	smfCount := len(FindResourcesByGVK(resources, schema.GroupVersionKind{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "SMF"}))

	upfCount := len(FindResourcesByGVK(resources, schema.GroupVersionKind{Group: "workload.nephio.org", Version: "v1alpha1", Kind: "UPF"}))

	// Check minimum requirements.

	if amfCount == 0 {
		results = append(results, CreateWarning("No AMF instances found in package"))
	}

	if smfCount == 0 {
		results = append(results, CreateWarning("No SMF instances found in package"))
	}

	if upfCount == 0 {
		results = append(results, CreateWarning("No UPF instances found in package"))
	}

	// Check ratios.

	if smfCount > 0 && upfCount > 0 {
		if upfCount > smfCount*3 {
			results = append(results, CreateWarning(

				fmt.Sprintf("High UPF to SMF ratio (%d:%d) may indicate over-provisioning", upfCount, smfCount),
			))
		}
	}

	return results
}

func (v *FiveGCoreValidator) isValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	return uuidRegex.MatchString(uuid)
}
