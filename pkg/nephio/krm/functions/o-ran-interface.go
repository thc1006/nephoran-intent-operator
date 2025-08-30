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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nephio-project/nephoran-intent-operator/pkg/nephio/porch"
)

// ORANInterfaceConfigFunction implements O-RAN interface configuration.

// Supports A1, O1, O2, and E2 interfaces according to O-RAN Alliance specifications.

type ORANInterfaceConfigFunction struct {
	tracer trace.Tracer
}

// ORANInterfaceConfig defines the configuration structure for O-RAN interfaces.

type ORANInterfaceConfig struct {

	// Interface specifications.

	Interfaces []*ORANInterface `json:"interfaces" yaml:"interfaces"`

	// Global settings.

	Compliance *ComplianceConfig `json:"compliance,omitempty" yaml:"compliance,omitempty"`

	Security *InterfaceSecurityConfig `json:"security,omitempty" yaml:"security,omitempty"`

	Monitoring *InterfaceMonitoringConfig `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`

	// RIC configuration.

	RICConfig *RICConfiguration `json:"ricConfig,omitempty" yaml:"ricConfig,omitempty"`

	// Service Model Registry.

	ServiceModels []*ServiceModelConfig `json:"serviceModels,omitempty" yaml:"serviceModels,omitempty"`

	// xApp configurations.

	XAppConfigs []*XAppConfiguration `json:"xappConfigs,omitempty" yaml:"xappConfigs,omitempty"`
}

// ORANInterface defines a single O-RAN interface configuration.

type ORANInterface struct {

	// Basic properties.

	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"` // A1, O1, O2, E2

	Version string `json:"version" yaml:"version"`

	Enabled bool `json:"enabled" yaml:"enabled"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Endpoint configuration.

	Endpoint *EndpointConfig `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	// Protocol configuration.

	Protocol *ProtocolConfig `json:"protocol,omitempty" yaml:"protocol,omitempty"`

	// Security configuration.

	Security *InterfaceSecurity `json:"security,omitempty" yaml:"security,omitempty"`

	// Interface-specific configuration.

	A1Config *A1InterfaceConfig `json:"a1Config,omitempty" yaml:"a1Config,omitempty"`

	O1Config *O1InterfaceConfig `json:"o1Config,omitempty" yaml:"o1Config,omitempty"`

	O2Config *O2InterfaceConfig `json:"o2Config,omitempty" yaml:"o2Config,omitempty"`

	E2Config *E2InterfaceConfig `json:"e2Config,omitempty" yaml:"e2Config,omitempty"`

	// Quality of Service.

	QoS *InterfaceQoSConfig `json:"qos,omitempty" yaml:"qos,omitempty"`

	// Monitoring and observability.

	Observability *InterfaceObservability `json:"observability,omitempty" yaml:"observability,omitempty"`
}

// EndpointConfig defines endpoint configuration.

type EndpointConfig struct {
	URL string `json:"url" yaml:"url"`

	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	Host string `json:"host,omitempty" yaml:"host,omitempty"`

	LoadBalancer *LoadBalancerConfig `json:"loadBalancer,omitempty" yaml:"loadBalancer,omitempty"`

	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
}

// LoadBalancerConfig defines load balancer configuration.

type LoadBalancerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"` // round-robin, least-connections, ip-hash

	HealthCheck bool `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`

	StickySession bool `json:"stickySession,omitempty" yaml:"stickySession,omitempty"`

	Backends []BackendConfig `json:"backends,omitempty" yaml:"backends,omitempty"`
}

// BackendConfig defines backend server configuration.

type BackendConfig struct {
	Host string `json:"host" yaml:"host"`

	Port int32 `json:"port" yaml:"port"`

	Weight int32 `json:"weight,omitempty" yaml:"weight,omitempty"`

	Enabled bool `json:"enabled" yaml:"enabled"`
}

// HealthCheckConfig defines health check configuration.

type HealthCheckConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`

	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	Retries int32 `json:"retries,omitempty" yaml:"retries,omitempty"`

	SuccessThreshold int32 `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`

	FailureThreshold int32 `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`
}

// ProtocolConfig defines protocol configuration.

type ProtocolConfig struct {
	Type string `json:"type" yaml:"type"` // HTTP, HTTPS, gRPC, SCTP

	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	Compression *CompressionConfig `json:"compression,omitempty" yaml:"compression,omitempty"`

	Timeout *TimeoutConfig `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// CompressionConfig defines compression settings.

type CompressionConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"` // gzip, deflate, lz4

	Level int32 `json:"level,omitempty" yaml:"level,omitempty"`

	MinSize int32 `json:"minSize,omitempty" yaml:"minSize,omitempty"`
}

// TimeoutConfig defines timeout settings.

type TimeoutConfig struct {
	Connect string `json:"connect,omitempty" yaml:"connect,omitempty"`

	Read string `json:"read,omitempty" yaml:"read,omitempty"`

	Write string `json:"write,omitempty" yaml:"write,omitempty"`

	Idle string `json:"idle,omitempty" yaml:"idle,omitempty"`
}

// InterfaceSecurity defines security configuration for an interface.

type InterfaceSecurity struct {
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`

	Authentication *AuthenticationConfig `json:"authentication,omitempty" yaml:"authentication,omitempty"`

	Authorization *AuthorizationConfig `json:"authorization,omitempty" yaml:"authorization,omitempty"`

	Encryption *EncryptionConfig `json:"encryption,omitempty" yaml:"encryption,omitempty"`
}

// TLSConfig defines TLS configuration.

type TLSConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	CipherSuites []string `json:"cipherSuites,omitempty" yaml:"cipherSuites,omitempty"`

	CertificateRef string `json:"certificateRef,omitempty" yaml:"certificateRef,omitempty"`

	KeyRef string `json:"keyRef,omitempty" yaml:"keyRef,omitempty"`

	CARef string `json:"caRef,omitempty" yaml:"caRef,omitempty"`

	ClientAuth bool `json:"clientAuth,omitempty" yaml:"clientAuth,omitempty"`
}

// AuthenticationConfig defines authentication configuration.

type AuthenticationConfig struct {
	Method string `json:"method" yaml:"method"` // oauth2, jwt, basic, certificate

	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	TokenEndpoint string `json:"tokenEndpoint,omitempty" yaml:"tokenEndpoint,omitempty"`

	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// AuthorizationConfig defines authorization configuration.

type AuthorizationConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Policies []AuthPolicy `json:"policies,omitempty" yaml:"policies,omitempty"`

	DefaultAction string `json:"defaultAction,omitempty" yaml:"defaultAction,omitempty"` // allow, deny

}

// AuthPolicy defines an authorization policy.

type AuthPolicy struct {
	Name string `json:"name" yaml:"name"`

	Rules []AuthRule `json:"rules,omitempty" yaml:"rules,omitempty"`

	Effect string `json:"effect" yaml:"effect"` // allow, deny

}

// AuthRule defines an authorization rule.

type AuthRule struct {
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`

	Action string `json:"action,omitempty" yaml:"action,omitempty"`

	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`

	Condition string `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// EncryptionConfig defines encryption configuration.

type EncryptionConfig struct {
	AtRest bool `json:"atRest,omitempty" yaml:"atRest,omitempty"`

	InTransit bool `json:"inTransit,omitempty" yaml:"inTransit,omitempty"`

	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`

	KeySize int32 `json:"keySize,omitempty" yaml:"keySize,omitempty"`
}

// A1InterfaceConfig defines A1 interface specific configuration.

type A1InterfaceConfig struct {

	// Policy Type Management.

	PolicyTypes []*PolicyTypeConfig `json:"policyTypes,omitempty" yaml:"policyTypes,omitempty"`

	// Policy Instance Management.

	PolicyInstances []*PolicyInstanceConfig `json:"policyInstances,omitempty" yaml:"policyInstances,omitempty"`

	// Enrichment Information.

	EITypes []*EITypeConfig `json:"eiTypes,omitempty" yaml:"eiTypes,omitempty"`

	// Near-RT RIC configuration.

	NearRTRIC *NearRTRICConfig `json:"nearRtRic,omitempty" yaml:"nearRtRic,omitempty"`

	// Consumer configuration.

	Consumers []*A1ConsumerConfig `json:"consumers,omitempty" yaml:"consumers,omitempty"`
}

// PolicyTypeConfig defines policy type configuration.

type PolicyTypeConfig struct {
	PolicyTypeID int32 `json:"policyTypeId" yaml:"policyTypeId"`

	PolicyTypeName string `json:"policyTypeName" yaml:"policyTypeName"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	CreateSchema map[string]interface{} `json:"createSchema" yaml:"createSchema"`

	PolicySchema map[string]interface{} `json:"policySchema,omitempty" yaml:"policySchema,omitempty"`
}

// PolicyInstanceConfig defines policy instance configuration.

type PolicyInstanceConfig struct {
	PolicyID string `json:"policyId" yaml:"policyId"`

	PolicyTypeID int32 `json:"policyTypeId" yaml:"policyTypeId"`

	ServiceID string `json:"serviceId,omitempty" yaml:"serviceId,omitempty"`

	PolicyData map[string]interface{} `json:"policyData" yaml:"policyData"`

	StatusNotificationURI string `json:"statusNotificationUri,omitempty" yaml:"statusNotificationUri,omitempty"`
}

// EITypeConfig defines Enrichment Information type configuration.

type EITypeConfig struct {
	EITypeID string `json:"eiTypeId" yaml:"eiTypeId"`

	EITypeName string `json:"eiTypeName" yaml:"eiTypeName"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	EIJobDataSchema map[string]interface{} `json:"eiJobDataSchema" yaml:"eiJobDataSchema"`
}

// NearRTRICConfig defines Near-RT RIC configuration.

type NearRTRICConfig struct {
	RICID string `json:"ricId" yaml:"ricId"`

	ManagedElementIDs []string `json:"managedElementIds,omitempty" yaml:"managedElementIds,omitempty"`

	RICState string `json:"ricState,omitempty" yaml:"ricState,omitempty"`

	PLMNs []PLMN `json:"plmns,omitempty" yaml:"plmns,omitempty"`
}

// PLMN defines Public Land Mobile Network.

type PLMN struct {
	MCC string `json:"mcc" yaml:"mcc"`

	MNC string `json:"mnc" yaml:"mnc"`
}

// A1ConsumerConfig defines A1 consumer configuration.

type A1ConsumerConfig struct {
	ConsumerID string `json:"consumerId" yaml:"consumerId"`

	ConsumerName string `json:"consumerName" yaml:"consumerName"`

	CallbackURL string `json:"callbackUrl" yaml:"callbackUrl"`

	SubscribedServices []string `json:"subscribedServices,omitempty" yaml:"subscribedServices,omitempty"`
}

// O1InterfaceConfig defines O1 interface specific configuration.

type O1InterfaceConfig struct {

	// FCAPS Management.

	FaultManagement *FaultManagementConfig `json:"faultManagement,omitempty" yaml:"faultManagement,omitempty"`

	ConfigManagement *ConfigManagementConfig `json:"configManagement,omitempty" yaml:"configManagement,omitempty"`

	AccountingManagement *AccountingManagementConfig `json:"accountingManagement,omitempty" yaml:"accountingManagement,omitempty"`

	PerformanceManagement *PerformanceManagementConfig `json:"performanceManagement,omitempty" yaml:"performanceManagement,omitempty"`

	SecurityManagement *SecurityManagementConfig `json:"securityManagement,omitempty" yaml:"securityManagement,omitempty"`

	// NETCONF/YANG configuration.

	NETCONF *NETCONFConfig `json:"netconf,omitempty" yaml:"netconf,omitempty"`

	YANGModels []*YANGModelConfig `json:"yangModels,omitempty" yaml:"yangModels,omitempty"`
}

// FaultManagementConfig defines fault management configuration.

type FaultManagementConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	AlarmList []AlarmConfig `json:"alarmList,omitempty" yaml:"alarmList,omitempty"`

	NotificationURL string `json:"notificationUrl,omitempty" yaml:"notificationUrl,omitempty"`

	CorrelationRules []CorrelationRule `json:"correlationRules,omitempty" yaml:"correlationRules,omitempty"`
}

// AlarmConfig defines alarm configuration.

type AlarmConfig struct {
	AlarmID string `json:"alarmId" yaml:"alarmId"`

	AlarmText string `json:"alarmText" yaml:"alarmText"`

	Severity string `json:"severity" yaml:"severity"`

	ProbableCause string `json:"probableCause,omitempty" yaml:"probableCause,omitempty"`

	ProposedRepairAction string `json:"proposedRepairAction,omitempty" yaml:"proposedRepairAction,omitempty"`
}

// CorrelationRule defines alarm correlation rule.

type CorrelationRule struct {
	RuleID string `json:"ruleId" yaml:"ruleId"`

	Condition string `json:"condition" yaml:"condition"`

	Action string `json:"action" yaml:"action"`

	TimeWindow string `json:"timeWindow,omitempty" yaml:"timeWindow,omitempty"`
}

// ConfigManagementConfig defines configuration management.

type ConfigManagementConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	ConfigDatastores []ConfigDatastore `json:"configDatastores,omitempty" yaml:"configDatastores,omitempty"`

	BackupSchedule string `json:"backupSchedule,omitempty" yaml:"backupSchedule,omitempty"`

	ValidationEnabled bool `json:"validationEnabled,omitempty" yaml:"validationEnabled,omitempty"`
}

// ConfigDatastore defines configuration datastore.

type ConfigDatastore struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"` // running, candidate, startup

	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// AccountingManagementConfig defines accounting management.

type AccountingManagementConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	UsageReporting *UsageReportingConfig `json:"usageReporting,omitempty" yaml:"usageReporting,omitempty"`

	BillingIntegration *BillingIntegrationConfig `json:"billingIntegration,omitempty" yaml:"billingIntegration,omitempty"`
}

// UsageReportingConfig defines usage reporting configuration.

type UsageReportingConfig struct {
	ReportingInterval string `json:"reportingInterval" yaml:"reportingInterval"`

	Metrics []string `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	DestinationURL string `json:"destinationUrl" yaml:"destinationUrl"`
}

// BillingIntegrationConfig defines billing system integration.

type BillingIntegrationConfig struct {
	Provider string `json:"provider" yaml:"provider"`

	APIEndpoint string `json:"apiEndpoint" yaml:"apiEndpoint"`

	Credentials map[string]string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
}

// PerformanceManagementConfig defines performance management.

type PerformanceManagementConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	KPIs []KPIDefinition `json:"kpis,omitempty" yaml:"kpis,omitempty"`

	CollectionInterval string `json:"collectionInterval,omitempty" yaml:"collectionInterval,omitempty"`

	ThresholdMonitoring *ThresholdMonitoringConfig `json:"thresholdMonitoring,omitempty" yaml:"thresholdMonitoring,omitempty"`
}

// KPIDefinition defines KPI configuration.

type KPIDefinition struct {
	Name string `json:"name" yaml:"name"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Unit string `json:"unit,omitempty" yaml:"unit,omitempty"`

	MeasurementType string `json:"measurementType" yaml:"measurementType"`

	Formula string `json:"formula,omitempty" yaml:"formula,omitempty"`
}

// ThresholdMonitoringConfig defines threshold monitoring.

type ThresholdMonitoringConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Thresholds []ThresholdConfig `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`

	CrossingAlert bool `json:"crossingAlert,omitempty" yaml:"crossingAlert,omitempty"`
}

// ThresholdConfig defines threshold configuration.

type ThresholdConfig struct {
	KPIName string `json:"kpiName" yaml:"kpiName"`

	WarningThreshold float64 `json:"warningThreshold,omitempty" yaml:"warningThreshold,omitempty"`

	CriticalThreshold float64 `json:"criticalThreshold,omitempty" yaml:"criticalThreshold,omitempty"`

	Direction string `json:"direction,omitempty" yaml:"direction,omitempty"` // up, down

}

// SecurityManagementConfig defines security management.

type SecurityManagementConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Certificates []CertificateConfig `json:"certificates,omitempty" yaml:"certificates,omitempty"`

	AccessControl []AccessControlRule `json:"accessControl,omitempty" yaml:"accessControl,omitempty"`

	AuditLogging *AuditLoggingConfig `json:"auditLogging,omitempty" yaml:"auditLogging,omitempty"`
}

// CertificateConfig defines certificate configuration.

type CertificateConfig struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"`

	CertData string `json:"certData,omitempty" yaml:"certData,omitempty"`

	KeyData string `json:"keyData,omitempty" yaml:"keyData,omitempty"`

	ExpiryDate *time.Time `json:"expiryDate,omitempty" yaml:"expiryDate,omitempty"`
}

// AccessControlRule defines access control rule.

type AccessControlRule struct {
	RuleID string `json:"ruleId" yaml:"ruleId"`

	Subject string `json:"subject" yaml:"subject"`

	Resource string `json:"resource" yaml:"resource"`

	Action string `json:"action" yaml:"action"`

	Permission string `json:"permission" yaml:"permission"` // allow, deny

}

// AuditLoggingConfig defines audit logging configuration.

type AuditLoggingConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	LogLevel string `json:"logLevel,omitempty" yaml:"logLevel,omitempty"`

	RetentionPeriod string `json:"retentionPeriod,omitempty" yaml:"retentionPeriod,omitempty"`

	DestinationURL string `json:"destinationUrl,omitempty" yaml:"destinationUrl,omitempty"`
}

// NETCONFConfig defines NETCONF configuration.

type NETCONFConfig struct {
	Port int32 `json:"port" yaml:"port"`

	CallHome bool `json:"callHome,omitempty" yaml:"callHome,omitempty"`

	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`

	SessionTimeout string `json:"sessionTimeout,omitempty" yaml:"sessionTimeout,omitempty"`
}

// YANGModelConfig defines YANG model configuration.

type YANGModelConfig struct {
	Name string `json:"name" yaml:"name"`

	Version string `json:"version" yaml:"version"`

	Namespace string `json:"namespace" yaml:"namespace"`

	Prefix string `json:"prefix" yaml:"prefix"`

	ModelData string `json:"modelData,omitempty" yaml:"modelData,omitempty"`
}

// O2InterfaceConfig defines O2 interface specific configuration.

type O2InterfaceConfig struct {

	// Infrastructure Management Service.

	IMS *IMSConfig `json:"ims,omitempty" yaml:"ims,omitempty"`

	// Deployment Management Service.

	DMS *DMSConfig `json:"dms,omitempty" yaml:"dms,omitempty"`

	// Resource lifecycle management.

	ResourceTypes []*ResourceTypeConfig `json:"resourceTypes,omitempty" yaml:"resourceTypes,omitempty"`

	// Cloud providers.

	CloudProviders []*CloudProviderConfig `json:"cloudProviders,omitempty" yaml:"cloudProviders,omitempty"`
}

// IMSConfig defines Infrastructure Management Service configuration.

type IMSConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	InventoryURL string `json:"inventoryUrl" yaml:"inventoryUrl"`

	ResourcePools []*ResourcePoolConfig `json:"resourcePools,omitempty" yaml:"resourcePools,omitempty"`

	SubscriptionURL string `json:"subscriptionUrl,omitempty" yaml:"subscriptionUrl,omitempty"`
}

// ResourcePoolConfig defines resource pool configuration.

type ResourcePoolConfig struct {
	PoolID string `json:"poolId" yaml:"poolId"`

	PoolName string `json:"poolName" yaml:"poolName"`

	ResourceTypeID string `json:"resourceTypeId" yaml:"resourceTypeId"`

	Location string `json:"location,omitempty" yaml:"location,omitempty"`

	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// DMSConfig defines Deployment Management Service configuration.

type DMSConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	DeploymentURL string `json:"deploymentUrl" yaml:"deploymentUrl"`

	HelmRepository string `json:"helmRepository,omitempty" yaml:"helmRepository,omitempty"`

	LifecycleHooks []*LifecycleHook `json:"lifecycleHooks,omitempty" yaml:"lifecycleHooks,omitempty"`
}

// LifecycleHook defines deployment lifecycle hook.

type LifecycleHook struct {
	Phase string `json:"phase" yaml:"phase"` // pre-install, post-install, pre-delete, post-delete

	Script string `json:"script" yaml:"script"`

	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
}

// ResourceTypeConfig defines resource type configuration.

type ResourceTypeConfig struct {
	ResourceTypeID string `json:"resourceTypeId" yaml:"resourceTypeId"`

	Name string `json:"name" yaml:"name"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// CloudProviderConfig defines cloud provider configuration.

type CloudProviderConfig struct {
	ProviderID string `json:"providerId" yaml:"providerId"`

	ProviderType string `json:"providerType" yaml:"providerType"` // aws, azure, gcp, openstack

	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	Credentials map[string]string `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	ResourceMapping map[string]string `json:"resourceMapping,omitempty" yaml:"resourceMapping,omitempty"`
}

// E2InterfaceConfig defines E2 interface specific configuration.

type E2InterfaceConfig struct {

	// E2 Node configuration.

	E2Nodes []*E2NodeConfig `json:"e2Nodes,omitempty" yaml:"e2Nodes,omitempty"`

	// Service Model configuration.

	ServiceModels []*E2ServiceModelConfig `json:"serviceModels,omitempty" yaml:"serviceModels,omitempty"`

	// Subscription management.

	Subscriptions []*E2SubscriptionConfig `json:"subscriptions,omitempty" yaml:"subscriptions,omitempty"`

	// Control message configuration.

	ControlMessages []*ControlMessageConfig `json:"controlMessages,omitempty" yaml:"controlMessages,omitempty"`

	// SCTP configuration.

	SCTP *SCTPConfig `json:"sctp,omitempty" yaml:"sctp,omitempty"`
}

// E2NodeConfig defines E2 node configuration.

type E2NodeConfig struct {
	NodeID string `json:"nodeId" yaml:"nodeId"`

	GlobalE2NodeID *GlobalE2NodeID `json:"globalE2NodeId" yaml:"globalE2NodeId"`

	RAN_Function []*RANFunctionConfig `json:"ranFunctions,omitempty" yaml:"ranFunctions,omitempty"`

	ConnectionStatus string `json:"connectionStatus,omitempty" yaml:"connectionStatus,omitempty"`
}

// GlobalE2NodeID defines global E2 node identifier.

type GlobalE2NodeID struct {
	PLMNIdentity string `json:"plmnIdentity" yaml:"plmnIdentity"`

	GNBIdentity *GNBIdentity `json:"gnbIdentity,omitempty" yaml:"gnbIdentity,omitempty"`

	ENBIdentity *ENBIdentity `json:"enbIdentity,omitempty" yaml:"enbIdentity,omitempty"`

	NgENBIdentity *NgENBIdentity `json:"ngenBIdentity,omitempty" yaml:"ngenBIdentity,omitempty"`
}

// GNBIdentity defines gNB identity.

type GNBIdentity struct {
	GNBID string `json:"gnbId" yaml:"gnbId"`

	GNBIDLength int32 `json:"gnbIdLength,omitempty" yaml:"gnbIdLength,omitempty"`
}

// ENBIdentity defines eNB identity.

type ENBIdentity struct {
	MacroENBID string `json:"macroEnbId,omitempty" yaml:"macroEnbId,omitempty"`

	HomeENBID string `json:"homeEnbId,omitempty" yaml:"homeEnbId,omitempty"`

	ShortMacroENBID string `json:"shortMacroEnbId,omitempty" yaml:"shortMacroEnbId,omitempty"`

	LongMacroENBID string `json:"longMacroEnbId,omitempty" yaml:"longMacroEnbId,omitempty"`
}

// NgENBIdentity defines ng-eNB identity.

type NgENBIdentity struct {
	MacroNgENBID string `json:"macroNgenBId,omitempty" yaml:"macroNgenBId,omitempty"`

	ShortMacroNgENBID string `json:"shortMacroNgenBId,omitempty" yaml:"shortMacroNgenBId,omitempty"`

	LongMacroNgENBID string `json:"longMacroNgenBId,omitempty" yaml:"longMacroNgenBId,omitempty"`
}

// RANFunctionConfig defines RAN function configuration.

type RANFunctionConfig struct {
	RANFunctionID int32 `json:"ranFunctionId" yaml:"ranFunctionId"`

	RANFunctionDefinition string `json:"ranFunctionDefinition" yaml:"ranFunctionDefinition"`

	RANFunctionRevision int32 `json:"ranFunctionRevision" yaml:"ranFunctionRevision"`

	RANFunctionOID string `json:"ranFunctionOid,omitempty" yaml:"ranFunctionOid,omitempty"`
}

// E2ServiceModelConfig defines E2 service model configuration.

type E2ServiceModelConfig struct {
	ServiceModelID string `json:"serviceModelId" yaml:"serviceModelId"`

	ServiceModelName string `json:"serviceModelName" yaml:"serviceModelName"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	ServiceModelOID string `json:"serviceModelOid" yaml:"serviceModelOid"`

	Functions []string `json:"functions,omitempty" yaml:"functions,omitempty"`
}

// E2SubscriptionConfig defines E2 subscription configuration.

type E2SubscriptionConfig struct {
	SubscriptionID string `json:"subscriptionId" yaml:"subscriptionId"`

	RANFunctionID int32 `json:"ranFunctionId" yaml:"ranFunctionId"`

	EventTriggers []*EventTrigger `json:"eventTriggers,omitempty" yaml:"eventTriggers,omitempty"`

	Actions []*E2Action `json:"actions,omitempty" yaml:"actions,omitempty"`

	ReportingPeriod string `json:"reportingPeriod,omitempty" yaml:"reportingPeriod,omitempty"`
}

// EventTrigger defines event trigger configuration.

type EventTrigger struct {
	TriggerType string `json:"triggerType" yaml:"triggerType"`

	TriggerCondition string `json:"triggerCondition,omitempty" yaml:"triggerCondition,omitempty"`

	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// E2Action defines E2 action configuration.

type E2Action struct {
	ActionID int32 `json:"actionId" yaml:"actionId"`

	ActionType string `json:"actionType" yaml:"actionType"` // report, insert, policy

	ActionDefinition map[string]interface{} `json:"actionDefinition,omitempty" yaml:"actionDefinition,omitempty"`

	SubsequentAction string `json:"subsequentAction,omitempty" yaml:"subsequentAction,omitempty"`
}

// ControlMessageConfig defines control message configuration.

type ControlMessageConfig struct {
	MessageType string `json:"messageType" yaml:"messageType"`

	RANFunctionID int32 `json:"ranFunctionId" yaml:"ranFunctionId"`

	ControlHeader map[string]interface{} `json:"controlHeader,omitempty" yaml:"controlHeader,omitempty"`

	ControlMessage map[string]interface{} `json:"controlMessage,omitempty" yaml:"controlMessage,omitempty"`

	CallProcessID string `json:"callProcessId,omitempty" yaml:"callProcessId,omitempty"`
}

// SCTPConfig defines SCTP configuration.

type SCTPConfig struct {
	Port int32 `json:"port" yaml:"port"`

	Streams int32 `json:"streams,omitempty" yaml:"streams,omitempty"`

	MaxInStreams int32 `json:"maxInStreams,omitempty" yaml:"maxInStreams,omitempty"`

	MaxOutStreams int32 `json:"maxOutStreams,omitempty" yaml:"maxOutStreams,omitempty"`

	HeartbeatInterval string `json:"heartbeatInterval,omitempty" yaml:"heartbeatInterval,omitempty"`
}

// InterfaceQoSConfig defines QoS configuration for interfaces.

type InterfaceQoSConfig struct {
	Priority int32 `json:"priority,omitempty" yaml:"priority,omitempty"`

	Bandwidth string `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"`

	Latency string `json:"latency,omitempty" yaml:"latency,omitempty"`

	PacketLoss float64 `json:"packetLoss,omitempty" yaml:"packetLoss,omitempty"`

	Jitter string `json:"jitter,omitempty" yaml:"jitter,omitempty"`
}

// InterfaceObservability defines observability configuration.

type InterfaceObservability struct {
	Logging *LoggingConfig `json:"logging,omitempty" yaml:"logging,omitempty"`

	Metrics *MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	Tracing *TracingConfig `json:"tracing,omitempty" yaml:"tracing,omitempty"`

	HealthChecks []string `json:"healthChecks,omitempty" yaml:"healthChecks,omitempty"`
}

// LoggingConfig defines logging configuration.

type LoggingConfig struct {
	Level string `json:"level,omitempty" yaml:"level,omitempty"`

	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	Output string `json:"output,omitempty" yaml:"output,omitempty"`

	Retention string `json:"retention,omitempty" yaml:"retention,omitempty"`
}

// MetricsConfig defines metrics configuration.

type MetricsConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`

	CustomMetrics []string `json:"customMetrics,omitempty" yaml:"customMetrics,omitempty"`
}

// TracingConfig defines tracing configuration.

type TracingConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	SamplingRate float64 `json:"samplingRate,omitempty" yaml:"samplingRate,omitempty"`

	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Supporting configuration types.

// ComplianceConfig defines overall compliance configuration.

type ComplianceConfig struct {
	Standard string `json:"standard" yaml:"standard"` // O-RAN.WG3.O-RAN-ARCH-v03.00

	Version string `json:"version" yaml:"version"`

	Validations []ComplianceValidation `json:"validations,omitempty" yaml:"validations,omitempty"`

	Certification bool `json:"certification,omitempty" yaml:"certification,omitempty"`
}

// ComplianceValidation defines compliance validation.

type ComplianceValidation struct {
	RuleID string `json:"ruleId" yaml:"ruleId"`

	Description string `json:"description" yaml:"description"`

	Severity string `json:"severity" yaml:"severity"`

	Enabled bool `json:"enabled" yaml:"enabled"`
}

// InterfaceSecurityConfig defines security configuration.

type InterfaceSecurityConfig struct {
	GlobalPolicies []SecurityPolicy `json:"globalPolicies,omitempty" yaml:"globalPolicies,omitempty"`

	ThreatDetection *ThreatDetection `json:"threatDetection,omitempty" yaml:"threatDetection,omitempty"`

	IncidentResponse *IncidentResponse `json:"incidentResponse,omitempty" yaml:"incidentResponse,omitempty"`
}

// SecurityPolicy defines security policy.

type SecurityPolicy struct {
	PolicyID string `json:"policyId" yaml:"policyId"`

	PolicyName string `json:"policyName" yaml:"policyName"`

	PolicyType string `json:"policyType" yaml:"policyType"`

	Rules []SecurityRule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// SecurityRule defines security rule.

type SecurityRule struct {
	RuleID string `json:"ruleId" yaml:"ruleId"`

	Condition string `json:"condition" yaml:"condition"`

	Action string `json:"action" yaml:"action"`

	Priority int32 `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// ThreatDetection defines threat detection configuration.

type ThreatDetection struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	DetectionMethods []string `json:"detectionMethods,omitempty" yaml:"detectionMethods,omitempty"`

	AlertThresholds map[string]float64 `json:"alertThresholds,omitempty" yaml:"alertThresholds,omitempty"`
}

// IncidentResponse defines incident response configuration.

type IncidentResponse struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	ResponseTeam []string `json:"responseTeam,omitempty" yaml:"responseTeam,omitempty"`

	EscalationProcedure string `json:"escalationProcedure,omitempty" yaml:"escalationProcedure,omitempty"`

	AutoResponse []AutoResponse `json:"autoResponse,omitempty" yaml:"autoResponse,omitempty"`
}

// AutoResponse defines automatic response configuration.

type AutoResponse struct {
	TriggerCondition string `json:"triggerCondition" yaml:"triggerCondition"`

	Action string `json:"action" yaml:"action"`

	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// InterfaceMonitoringConfig defines monitoring configuration.

type InterfaceMonitoringConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	KPIs []InterfaceKPI `json:"kpis,omitempty" yaml:"kpis,omitempty"`

	Dashboards []DashboardConfig `json:"dashboards,omitempty" yaml:"dashboards,omitempty"`

	Alerting *InterfaceAlerting `json:"alerting,omitempty" yaml:"alerting,omitempty"`
}

// InterfaceKPI defines interface KPI.

type InterfaceKPI struct {
	Name string `json:"name" yaml:"name"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Unit string `json:"unit,omitempty" yaml:"unit,omitempty"`

	CollectionInterval string `json:"collectionInterval,omitempty" yaml:"collectionInterval,omitempty"`

	Target float64 `json:"target,omitempty" yaml:"target,omitempty"`
}

// DashboardConfig defines dashboard configuration.

type DashboardConfig struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"`

	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// InterfaceAlerting defines interface alerting.

type InterfaceAlerting struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Rules []AlertingRule `json:"rules,omitempty" yaml:"rules,omitempty"`

	Channels []AlertingChannel `json:"channels,omitempty" yaml:"channels,omitempty"`
}

// AlertingRule defines alerting rule.

type AlertingRule struct {
	RuleID string `json:"ruleId" yaml:"ruleId"`

	Metric string `json:"metric" yaml:"metric"`

	Condition string `json:"condition" yaml:"condition"`

	Threshold float64 `json:"threshold" yaml:"threshold"`

	Duration string `json:"duration,omitempty" yaml:"duration,omitempty"`

	Severity string `json:"severity,omitempty" yaml:"severity,omitempty"`
}

// AlertingChannel defines alerting channel.

type AlertingChannel struct {
	ChannelID string `json:"channelId" yaml:"channelId"`

	Type string `json:"type" yaml:"type"`

	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`

	Enabled bool `json:"enabled" yaml:"enabled"`
}

// RICConfiguration defines RIC configuration.

type RICConfiguration struct {
	NonRTRIC *NonRTRICConfig `json:"nonRtRic,omitempty" yaml:"nonRtRic,omitempty"`

	NearRTRIC *NearRTRICConfig `json:"nearRtRic,omitempty" yaml:"nearRtRic,omitempty"`

	RICPlatform *RICPlatformConfig `json:"ricPlatform,omitempty" yaml:"ricPlatform,omitempty"`
}

// NonRTRICConfig defines Non-RT RIC configuration.

type NonRTRICConfig struct {
	PolicyManagement *PolicyManagementConfig `json:"policyManagement,omitempty" yaml:"policyManagement,omitempty"`

	EnrichmentInfo *EnrichmentInfoConfig `json:"enrichmentInfo,omitempty" yaml:"enrichmentInfo,omitempty"`

	APPCatalog *APPCatalogConfig `json:"appCatalog,omitempty" yaml:"appCatalog,omitempty"`
}

// PolicyManagementConfig defines policy management configuration.

type PolicyManagementConfig struct {
	ServiceURL string `json:"serviceUrl" yaml:"serviceUrl"`

	Policies []PolicyConfig `json:"policies,omitempty" yaml:"policies,omitempty"`
}

// PolicyConfig defines policy configuration.

type PolicyConfig struct {
	PolicyID string `json:"policyId" yaml:"policyId"`

	PolicyType string `json:"policyType" yaml:"policyType"`

	ServiceID string `json:"serviceId" yaml:"serviceId"`

	PolicyData map[string]interface{} `json:"policyData" yaml:"policyData"`
}

// EnrichmentInfoConfig defines enrichment information configuration.

type EnrichmentInfoConfig struct {
	ServiceURL string `json:"serviceUrl" yaml:"serviceUrl"`

	DataTypes []DataTypeConfig `json:"dataTypes,omitempty" yaml:"dataTypes,omitempty"`
}

// DataTypeConfig defines data type configuration.

type DataTypeConfig struct {
	InfoTypeID string `json:"infoTypeId" yaml:"infoTypeId"`

	InfoTypeName string `json:"infoTypeName" yaml:"infoTypeName"`

	Schema map[string]interface{} `json:"schema" yaml:"schema"`
}

// APPCatalogConfig defines APP catalog configuration.

type APPCatalogConfig struct {
	ServiceURL string `json:"serviceUrl" yaml:"serviceUrl"`

	Applications []ApplicationConfig `json:"applications,omitempty" yaml:"applications,omitempty"`
}

// ApplicationConfig defines application configuration.

type ApplicationConfig struct {
	ApplicationID string `json:"applicationId" yaml:"applicationId"`

	ApplicationName string `json:"applicationName" yaml:"applicationName"`

	Version string `json:"version" yaml:"version"`

	Artifacts []ArtifactConfig `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
}

// ArtifactConfig defines artifact configuration.

type ArtifactConfig struct {
	ArtifactID string `json:"artifactId" yaml:"artifactId"`

	Type string `json:"type" yaml:"type"`

	URL string `json:"url" yaml:"url"`

	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
}

// RICPlatformConfig defines RIC platform configuration.

type RICPlatformConfig struct {
	Components []ComponentConfig `json:"components,omitempty" yaml:"components,omitempty"`

	Messaging *MessagingConfig `json:"messaging,omitempty" yaml:"messaging,omitempty"`

	Database *DatabaseConfig `json:"database,omitempty" yaml:"database,omitempty"`
}

// ComponentConfig defines platform component configuration.

type ComponentConfig struct {
	ComponentID string `json:"componentId" yaml:"componentId"`

	ComponentName string `json:"componentName" yaml:"componentName"`

	Image string `json:"image" yaml:"image"`

	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	Environment map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// MessagingConfig defines messaging configuration.

type MessagingConfig struct {
	Type string `json:"type" yaml:"type"` // rmr, kafka, rabbitmq

	Endpoints []string `json:"endpoints" yaml:"endpoints"`

	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// DatabaseConfig defines database configuration.

type DatabaseConfig struct {
	Type string `json:"type" yaml:"type"` // redis, etcd, postgresql

	Endpoints []string `json:"endpoints" yaml:"endpoints"`

	Credentials map[string]string `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// ServiceModelConfig defines service model configuration.

type ServiceModelConfig struct {
	ServiceModelID string `json:"serviceModelId" yaml:"serviceModelId"`

	ServiceModelName string `json:"serviceModelName" yaml:"serviceModelName"`

	ServiceModelOID string `json:"serviceModelOid" yaml:"serviceModelOid"`

	Functions []ServiceModelFunction `json:"functions,omitempty" yaml:"functions,omitempty"`

	Schema map[string]interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// ServiceModelFunction defines service model function.

type ServiceModelFunction struct {
	FunctionID int32 `json:"functionId" yaml:"functionId"`

	FunctionName string `json:"functionName" yaml:"functionName"`

	FunctionType string `json:"functionType" yaml:"functionType"` // REPORT, INSERT, CONTROL, POLICY

	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// XAppConfiguration defines xApp configuration.

type XAppConfiguration struct {
	XAppName string `json:"xappName" yaml:"xappName"`

	XAppVersion string `json:"xappVersion" yaml:"xappVersion"`

	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	HelmChart *HelmChartConfig `json:"helmChart,omitempty" yaml:"helmChart,omitempty"`

	Configuration map[string]interface{} `json:"configuration,omitempty" yaml:"configuration,omitempty"`

	Resources *XAppResources `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// HelmChartConfig defines Helm chart configuration.

type HelmChartConfig struct {
	ChartName string `json:"chartName" yaml:"chartName"`

	ChartVersion string `json:"chartVersion" yaml:"chartVersion"`

	Repository string `json:"repository" yaml:"repository"`

	Values map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
}

// XAppResources defines xApp resource requirements.

type XAppResources struct {
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	Storage string `json:"storage,omitempty" yaml:"storage,omitempty"`

	Replicas int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}

// NewORANInterfaceConfigFunction creates a new O-RAN interface configuration function.

func NewORANInterfaceConfigFunction() *ORANInterfaceConfigFunction {

	return &ORANInterfaceConfigFunction{

		tracer: otel.Tracer("oran-interface-config-function"),
	}

}

// Execute implements the KRM function for O-RAN interface configuration.

func (f *ORANInterfaceConfigFunction) Execute(ctx context.Context, resources []porch.KRMResource, config map[string]interface{}) ([]porch.KRMResource, []*porch.FunctionResult, error) {

	ctx, span := f.tracer.Start(ctx, "oran-interface-config-execute")

	defer span.End()

	logger := log.FromContext(ctx).WithName("oran-interface-config")

	span.SetAttributes(

		attribute.Int("input.resources", len(resources)),
	)

	// Parse configuration.

	oranConfig, err := f.parseConfig(config)

	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "config parsing failed")

		return nil, nil, fmt.Errorf("failed to parse O-RAN interface configuration: %w", err)

	}

	span.SetAttributes(

		attribute.Int("interfaces.count", len(oranConfig.Interfaces)),
	)

	logger.Info("Configuring O-RAN interfaces",

		"interfaces", len(oranConfig.Interfaces),

		"resources", len(resources),
	)

	// Process existing resources.

	processedResources, results, err := f.processResources(ctx, resources, oranConfig)

	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "resource processing failed")

		return nil, results, fmt.Errorf("failed to process O-RAN interface resources: %w", err)

	}

	// Generate additional resources for O-RAN interfaces.

	additionalResources, additionalResults, err := f.generateInterfaceResources(ctx, oranConfig)

	if err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, "interface resource generation failed")

		logger.Error(err, "Failed to generate O-RAN interface resources")

		// Continue with what we have.

	} else {

		processedResources = append(processedResources, additionalResources...)

		results = append(results, additionalResults...)

	}

	span.SetAttributes(

		attribute.Int("output.resources", len(processedResources)),

		attribute.Int("output.results", len(results)),
	)

	span.SetStatus(codes.Ok, "O-RAN interface configuration completed")

	logger.Info("O-RAN interface configuration completed successfully",

		"processedResources", len(processedResources),
	)

	return processedResources, results, nil

}

// parseConfig parses the function configuration into ORANInterfaceConfig.

func (f *ORANInterfaceConfigFunction) parseConfig(config map[string]interface{}) (*ORANInterfaceConfig, error) {

	if config == nil {

		return nil, fmt.Errorf("configuration is required")

	}

	// Convert to JSON and back to struct for type safety.

	configJSON, err := json.Marshal(config)

	if err != nil {

		return nil, fmt.Errorf("failed to marshal config: %w", err)

	}

	var oranConfig ORANInterfaceConfig

	if err := json.Unmarshal(configJSON, &oranConfig); err != nil {

		return nil, fmt.Errorf("failed to unmarshal config: %w", err)

	}

	// Validate configuration.

	if err := f.validateConfig(&oranConfig); err != nil {

		return nil, fmt.Errorf("config validation failed: %w", err)

	}

	return &oranConfig, nil

}

// validateConfig validates the O-RAN interface configuration.

func (f *ORANInterfaceConfigFunction) validateConfig(config *ORANInterfaceConfig) error {

	if len(config.Interfaces) == 0 {

		return fmt.Errorf("at least one interface configuration is required")

	}

	// Validate each interface.

	for i, iface := range config.Interfaces {

		if iface.Name == "" {

			return fmt.Errorf("interface %d: name is required", i)

		}

		if iface.Type == "" {

			return fmt.Errorf("interface %d: type is required", i)

		}

		// Validate interface type.

		validTypes := []string{"A1", "O1", "O2", "E2"}

		validType := false

		for _, validIfaceType := range validTypes {

			if iface.Type == validIfaceType {

				validType = true

				break

			}

		}

		if !validType {

			return fmt.Errorf("interface %d: invalid type %s, must be one of: %v", i, iface.Type, validTypes)

		}

		// Validate type-specific configuration.

		switch iface.Type {

		case "A1":

			if iface.A1Config == nil {

				return fmt.Errorf("interface %d: A1 configuration is required for A1 interface", i)

			}

		case "O1":

			if iface.O1Config == nil {

				return fmt.Errorf("interface %d: O1 configuration is required for O1 interface", i)

			}

		case "O2":

			if iface.O2Config == nil {

				return fmt.Errorf("interface %d: O2 configuration is required for O2 interface", i)

			}

		case "E2":

			if iface.E2Config == nil {

				return fmt.Errorf("interface %d: E2 configuration is required for E2 interface", i)

			}

		}

	}

	return nil

}

// processResources processes existing resources based on O-RAN interface configuration.

func (f *ORANInterfaceConfigFunction) processResources(ctx context.Context, resources []porch.KRMResource, config *ORANInterfaceConfig) ([]porch.KRMResource, []*porch.FunctionResult, error) {

	var processedResources []porch.KRMResource

	var results []*porch.FunctionResult

	for i, resource := range resources {

		// Process based on resource type.

		switch {

		case resource.Kind == "Deployment":

			processed, result := f.processDeployment(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {

				results = append(results, result)

			}

		case resource.Kind == "Service":

			processed, result := f.processService(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {

				results = append(results, result)

			}

		case resource.Kind == "ConfigMap":

			processed, result := f.processConfigMap(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {

				results = append(results, result)

			}

		case resource.Kind == "Secret":

			processed, result := f.processSecret(resource, config)

			processedResources = append(processedResources, processed)

			if result != nil {

				results = append(results, result)

			}

		default:

			// Pass through unmodified.

			processedResources = append(processedResources, resource)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Resource %d passed through unchanged", i),

				Severity: "info",
			})

		}

	}

	return processedResources, results, nil

}

// processDeployment configures Deployment resources for O-RAN interfaces.

func (f *ORANInterfaceConfigFunction) processDeployment(resource porch.KRMResource, config *ORANInterfaceConfig) (porch.KRMResource, *porch.FunctionResult) {

	// Add O-RAN interface labels and annotations.

	f.addORANLabelsAndAnnotations(&resource, config)

	// Configure environment variables for O-RAN interfaces.

	f.configureORANEnvironment(&resource, config)

	// Configure ports for O-RAN interfaces.

	f.configureORANPorts(&resource, config)

	return resource, &porch.FunctionResult{

		Message: "Configured Deployment for O-RAN interfaces",

		Severity: "info",
	}

}

// processService configures Service resources for O-RAN interfaces.

func (f *ORANInterfaceConfigFunction) processService(resource porch.KRMResource, config *ORANInterfaceConfig) (porch.KRMResource, *porch.FunctionResult) {

	// Add O-RAN interface labels and annotations.

	f.addORANLabelsAndAnnotations(&resource, config)

	// Configure service ports for O-RAN interfaces.

	f.configureServicePorts(&resource, config)

	return resource, &porch.FunctionResult{

		Message: "Configured Service for O-RAN interfaces",

		Severity: "info",
	}

}

// processConfigMap configures ConfigMap resources for O-RAN interfaces.

func (f *ORANInterfaceConfigFunction) processConfigMap(resource porch.KRMResource, config *ORANInterfaceConfig) (porch.KRMResource, *porch.FunctionResult) {

	// Add O-RAN interface configuration to ConfigMap.

	if resource.Data == nil {

		resource.Data = make(map[string]interface{})

	}

	// Add interface configurations.

	for _, iface := range config.Interfaces {

		configData, _ := json.Marshal(iface)

		resource.Data[fmt.Sprintf("%s-interface-config.json", strings.ToLower(iface.Type))] = string(configData)

	}

	// Add service model configurations.

	if len(config.ServiceModels) > 0 {

		serviceModelsData, _ := json.Marshal(config.ServiceModels)

		resource.Data["service-models.json"] = string(serviceModelsData)

	}

	// Add RIC configuration if present.

	if config.RICConfig != nil {

		ricConfigData, _ := json.Marshal(config.RICConfig)

		resource.Data["ric-config.json"] = string(ricConfigData)

	}

	return resource, &porch.FunctionResult{

		Message: "Configured ConfigMap for O-RAN interfaces",

		Severity: "info",
	}

}

// processSecret configures Secret resources for O-RAN interface security.

func (f *ORANInterfaceConfigFunction) processSecret(resource porch.KRMResource, config *ORANInterfaceConfig) (porch.KRMResource, *porch.FunctionResult) {

	// Add O-RAN interface security configurations.

	if resource.Data == nil {

		resource.Data = make(map[string]interface{})

	}

	// Add security configurations for interfaces that require them.

	for _, iface := range config.Interfaces {

		if iface.Security != nil && iface.Security.TLS != nil && iface.Security.TLS.Enabled {

			// Add TLS certificate references.

			if iface.Security.TLS.CertificateRef != "" {

				resource.Data[fmt.Sprintf("%s-tls-cert", iface.Name)] = iface.Security.TLS.CertificateRef

			}

			if iface.Security.TLS.KeyRef != "" {

				resource.Data[fmt.Sprintf("%s-tls-key", iface.Name)] = iface.Security.TLS.KeyRef

			}

		}

	}

	return resource, &porch.FunctionResult{

		Message: "Configured Secret for O-RAN interface security",

		Severity: "info",
	}

}

// generateInterfaceResources generates additional resources for O-RAN interfaces.

func (f *ORANInterfaceConfigFunction) generateInterfaceResources(ctx context.Context, config *ORANInterfaceConfig) ([]porch.KRMResource, []*porch.FunctionResult, error) {

	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	// Generate resources for each interface.

	for _, iface := range config.Interfaces {

		switch iface.Type {

		case "A1":

			ifaceResources, ifaceResults := f.generateA1Resources(iface)

			resources = append(resources, ifaceResources...)

			results = append(results, ifaceResults...)

		case "O1":

			ifaceResources, ifaceResults := f.generateO1Resources(iface)

			resources = append(resources, ifaceResources...)

			results = append(results, ifaceResults...)

		case "O2":

			ifaceResources, ifaceResults := f.generateO2Resources(iface)

			resources = append(resources, ifaceResources...)

			results = append(results, ifaceResults...)

		case "E2":

			ifaceResources, ifaceResults := f.generateE2Resources(iface)

			resources = append(resources, ifaceResources...)

			results = append(results, ifaceResults...)

		}

	}

	// Generate ServiceMonitor for monitoring.

	if config.Monitoring != nil && config.Monitoring.Enabled {

		monitor := f.generateServiceMonitor(config)

		resources = append(resources, monitor)

		results = append(results, &porch.FunctionResult{

			Message: "Generated ServiceMonitor for O-RAN interfaces",

			Severity: "info",
		})

	}

	return resources, results, nil

}

// Helper methods for resource configuration.

func (f *ORANInterfaceConfigFunction) addORANLabelsAndAnnotations(resource *porch.KRMResource, config *ORANInterfaceConfig) {

	// Add labels.

	if resource.Metadata == nil {

		resource.Metadata = make(map[string]interface{})

	}

	if resource.Metadata["labels"] == nil {

		resource.Metadata["labels"] = make(map[string]interface{})

	}

	labels := resource.Metadata["labels"].(map[string]interface{})

	labels["nephoran.com/oran-enabled"] = "true"

	labels["nephoran.com/managed-by"] = "oran-interface-config"

	// Add interface types as labels.

	var interfaceTypes []string

	for _, iface := range config.Interfaces {

		interfaceTypes = append(interfaceTypes, iface.Type)

	}

	labels["nephoran.com/oran-interfaces"] = strings.Join(interfaceTypes, ",")

	// Add annotations.

	if resource.Metadata["annotations"] == nil {

		resource.Metadata["annotations"] = make(map[string]interface{})

	}

	annotations := resource.Metadata["annotations"].(map[string]interface{})

	annotations["nephoran.com/oran-config-timestamp"] = time.Now().UTC().Format(time.RFC3339)

	// Add compliance annotations.

	if config.Compliance != nil {

		annotations["nephoran.com/oran-standard"] = config.Compliance.Standard

		annotations["nephoran.com/oran-version"] = config.Compliance.Version

	}

}

func (f *ORANInterfaceConfigFunction) configureORANEnvironment(resource *porch.KRMResource, config *ORANInterfaceConfig) {

	// Configure environment variables in deployment containers.

	if resource.Spec == nil {

		return

	}

	if template, ok := resource.Spec["template"].(map[string]interface{}); ok {

		if spec, ok := template["spec"].(map[string]interface{}); ok {

			if containers, ok := spec["containers"].([]interface{}); ok {

				for _, container := range containers {

					if containerMap, ok := container.(map[string]interface{}); ok {

						if containerMap["env"] == nil {

							containerMap["env"] = []interface{}{}

						}

						envVars := containerMap["env"].([]interface{})

						// Add environment variables for each interface.

						for _, iface := range config.Interfaces {

							envVar := map[string]interface{}{

								"name": fmt.Sprintf("ORAN_%s_ENABLED", iface.Type),

								"value": fmt.Sprintf("%t", iface.Enabled),
							}

							envVars = append(envVars, envVar)

							if iface.Endpoint != nil {

								envVar = map[string]interface{}{

									"name": fmt.Sprintf("ORAN_%s_ENDPOINT", iface.Type),

									"value": iface.Endpoint.URL,
								}

								envVars = append(envVars, envVar)

							}

						}

						containerMap["env"] = envVars

					}

				}

			}

		}

	}

}

func (f *ORANInterfaceConfigFunction) configureORANPorts(resource *porch.KRMResource, config *ORANInterfaceConfig) {

	// Configure container ports in deployment.

	if resource.Spec == nil {

		return

	}

	if template, ok := resource.Spec["template"].(map[string]interface{}); ok {

		if spec, ok := template["spec"].(map[string]interface{}); ok {

			if containers, ok := spec["containers"].([]interface{}); ok {

				for _, container := range containers {

					if containerMap, ok := container.(map[string]interface{}); ok {

						if containerMap["ports"] == nil {

							containerMap["ports"] = []interface{}{}

						}

						ports := containerMap["ports"].([]interface{})

						// Add ports for each interface.

						for _, iface := range config.Interfaces {

							if iface.Endpoint != nil && iface.Endpoint.Port != 0 {

								port := map[string]interface{}{

									"name": fmt.Sprintf("%s-port", strings.ToLower(iface.Name)),

									"containerPort": iface.Endpoint.Port,

									"protocol": "TCP",
								}

								if iface.Protocol != nil {

									port["protocol"] = strings.ToUpper(iface.Protocol.Type)

								}

								ports = append(ports, port)

							}

						}

						containerMap["ports"] = ports

					}

				}

			}

		}

	}

}

func (f *ORANInterfaceConfigFunction) configureServicePorts(resource *porch.KRMResource, config *ORANInterfaceConfig) {

	// Configure service ports.

	if resource.Spec == nil {

		resource.Spec = make(map[string]interface{})

	}

	if resource.Spec["ports"] == nil {

		resource.Spec["ports"] = []interface{}{}

	}

	ports := resource.Spec["ports"].([]interface{})

	// Add ports for each interface.

	for _, iface := range config.Interfaces {

		if iface.Endpoint != nil && iface.Endpoint.Port != 0 {

			port := map[string]interface{}{

				"name": fmt.Sprintf("%s-port", strings.ToLower(iface.Name)),

				"port": iface.Endpoint.Port,

				"targetPort": iface.Endpoint.Port,

				"protocol": "TCP",
			}

			if iface.Protocol != nil {

				port["protocol"] = strings.ToUpper(iface.Protocol.Type)

			}

			ports = append(ports, port)

		}

	}

	resource.Spec["ports"] = ports

}

// Interface-specific resource generation methods.

func (f *ORANInterfaceConfigFunction) generateA1Resources(iface *ORANInterface) ([]porch.KRMResource, []*porch.FunctionResult) {

	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	if iface.A1Config != nil {

		// Generate A1 policy ConfigMap.

		if len(iface.A1Config.PolicyTypes) > 0 {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-a1-policies", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "A1",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: make(map[string]interface{}),
			}

			for _, policyType := range iface.A1Config.PolicyTypes {

				policyData, _ := json.Marshal(policyType)

				configMap.Data[fmt.Sprintf("policy-type-%d.json", policyType.PolicyTypeID)] = string(policyData)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated A1 policy ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

	}

	return resources, results

}

func (f *ORANInterfaceConfigFunction) generateO1Resources(iface *ORANInterface) ([]porch.KRMResource, []*porch.FunctionResult) {

	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	if iface.O1Config != nil {

		// Generate O1 NETCONF ConfigMap.

		if iface.O1Config.NETCONF != nil {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-o1-netconf", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "O1",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: map[string]interface{}{

					"netconf-port": fmt.Sprintf("%d", iface.O1Config.NETCONF.Port),

					"call-home": fmt.Sprintf("%t", iface.O1Config.NETCONF.CallHome),
				},
			}

			if len(iface.O1Config.NETCONF.Capabilities) > 0 {

				capabilities, _ := json.Marshal(iface.O1Config.NETCONF.Capabilities)

				configMap.Data["capabilities.json"] = string(capabilities)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated O1 NETCONF ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

		// Generate YANG models ConfigMap.

		if len(iface.O1Config.YANGModels) > 0 {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-o1-yang", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "O1",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: make(map[string]interface{}),
			}

			for _, yangModel := range iface.O1Config.YANGModels {

				yangData, _ := json.Marshal(yangModel)

				configMap.Data[fmt.Sprintf("%s-model.json", yangModel.Name)] = string(yangData)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated O1 YANG models ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

	}

	return resources, results

}

func (f *ORANInterfaceConfigFunction) generateO2Resources(iface *ORANInterface) ([]porch.KRMResource, []*porch.FunctionResult) {

	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	if iface.O2Config != nil {

		// Generate O2 IMS ConfigMap.

		if iface.O2Config.IMS != nil && iface.O2Config.IMS.Enabled {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-o2-ims", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "O2",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: map[string]interface{}{

					"inventory-url": iface.O2Config.IMS.InventoryURL,

					"subscription-url": iface.O2Config.IMS.SubscriptionURL,
				},
			}

			if len(iface.O2Config.IMS.ResourcePools) > 0 {

				poolsData, _ := json.Marshal(iface.O2Config.IMS.ResourcePools)

				configMap.Data["resource-pools.json"] = string(poolsData)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated O2 IMS ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

		// Generate O2 cloud providers Secret if credentials are present.

		if len(iface.O2Config.CloudProviders) > 0 {

			secret := porch.KRMResource{

				APIVersion: "v1",

				Kind: "Secret",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-o2-cloud-creds", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "O2",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: make(map[string]interface{}),
			}

			for _, provider := range iface.O2Config.CloudProviders {

				if len(provider.Credentials) > 0 {

					credsData, _ := json.Marshal(provider.Credentials)

					secret.Data[fmt.Sprintf("%s-credentials", provider.ProviderID)] = string(credsData)

				}

			}

			resources = append(resources, secret)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated O2 cloud credentials Secret for %s", iface.Name),

				Severity: "info",
			})

		}

	}

	return resources, results

}

func (f *ORANInterfaceConfigFunction) generateE2Resources(iface *ORANInterface) ([]porch.KRMResource, []*porch.FunctionResult) {

	var resources []porch.KRMResource

	var results []*porch.FunctionResult

	if iface.E2Config != nil {

		// Generate E2 nodes ConfigMap.

		if len(iface.E2Config.E2Nodes) > 0 {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-e2-nodes", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "E2",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: make(map[string]interface{}),
			}

			for _, e2Node := range iface.E2Config.E2Nodes {

				nodeData, _ := json.Marshal(e2Node)

				configMap.Data[fmt.Sprintf("%s-node.json", e2Node.NodeID)] = string(nodeData)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated E2 nodes ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

		// Generate E2 service models ConfigMap.

		if len(iface.E2Config.ServiceModels) > 0 {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-e2-service-models", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "E2",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: make(map[string]interface{}),
			}

			for _, serviceModel := range iface.E2Config.ServiceModels {

				modelData, _ := json.Marshal(serviceModel)

				configMap.Data[fmt.Sprintf("%s-service-model.json", serviceModel.ServiceModelID)] = string(modelData)

			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated E2 service models ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

		// Generate SCTP configuration if present.

		if iface.E2Config.SCTP != nil {

			configMap := porch.KRMResource{

				APIVersion: "v1",

				Kind: "ConfigMap",

				Metadata: map[string]interface{}{

					"name": fmt.Sprintf("%s-e2-sctp", iface.Name),

					"namespace": "default",

					"labels": map[string]interface{}{

						"nephoran.com/oran-interface": "E2",

						"nephoran.com/interface-name": iface.Name,
					},
				},

				Data: map[string]interface{}{

					"sctp-port": fmt.Sprintf("%d", iface.E2Config.SCTP.Port),

					"sctp-streams": fmt.Sprintf("%d", iface.E2Config.SCTP.Streams),

					"sctp-max-in-streams": fmt.Sprintf("%d", iface.E2Config.SCTP.MaxInStreams),

					"sctp-max-out-streams": fmt.Sprintf("%d", iface.E2Config.SCTP.MaxOutStreams),

					"sctp-heartbeat-interval": iface.E2Config.SCTP.HeartbeatInterval,
				},
			}

			resources = append(resources, configMap)

			results = append(results, &porch.FunctionResult{

				Message: fmt.Sprintf("Generated E2 SCTP ConfigMap for %s", iface.Name),

				Severity: "info",
			})

		}

	}

	return resources, results

}

func (f *ORANInterfaceConfigFunction) generateServiceMonitor(config *ORANInterfaceConfig) porch.KRMResource {

	return porch.KRMResource{

		APIVersion: "monitoring.coreos.com/v1",

		Kind: "ServiceMonitor",

		Metadata: map[string]interface{}{

			"name": "oran-interfaces-monitor",

			"namespace": "default",

			"labels": map[string]interface{}{

				"nephoran.com/oran-interfaces": "enabled",
			},
		},

		Spec: map[string]interface{}{

			"selector": map[string]interface{}{

				"matchLabels": map[string]interface{}{

					"nephoran.com/oran-enabled": "true",
				},
			},

			"endpoints": []map[string]interface{}{

				{

					"port": "metrics",

					"interval": "30s",

					"path": "/metrics",
				},
			},
		},
	}

}
