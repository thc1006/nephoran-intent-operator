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

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Note: Priority types are defined in common_types.go

// TargetComponent defines a component targeted for disaster recovery
type TargetComponent struct {
	// Name of the component
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of component (deployment, statefulset, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=deployment;statefulset;service;configmap;secret;pvc
	Type string `json:"type"`

	// Namespace of the component
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Label selector for the component
	// +optional
	LabelSelector map[string]string `json:"labelSelector,omitempty"`

	// Dependencies on other components
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`
}

// ResourceConstraints defines resource constraints for operations
type ResourceConstraints struct {
	// CPU limits
	// +optional
	CPU *resource.Quantity `json:"cpu,omitempty"`

	// Memory limits
	// +optional
	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage limits
	// +optional
	Storage *resource.Quantity `json:"storage,omitempty"`

	// Network bandwidth limits
	// +optional
	NetworkBandwidth *resource.Quantity `json:"networkBandwidth,omitempty"`

	// Maximum CPU allocation (for blueprint engine compatibility)
	// +optional
	MaxCPU *resource.Quantity `json:"maxCPU,omitempty"`
	// Maximum memory allocation (for blueprint engine compatibility)
	// +optional
	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`
	// Maximum concurrent operations
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	MaxConcurrency *int32 `json:"maxConcurrency,omitempty"`
}

// DisasterRecoveryPlan defines a comprehensive disaster recovery plan
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.planType`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="RTO Target",type=string,JSONPath=`.spec.rtoTarget`
// +kubebuilder:printcolumn:name="RPO Target",type=string,JSONPath=`.spec.rpoTarget`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=drp;drplan
type DisasterRecoveryPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DisasterRecoveryPlanSpec   `json:"spec,omitempty"`
	Status DisasterRecoveryPlanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DisasterRecoveryPlanList contains a list of DisasterRecoveryPlan
type DisasterRecoveryPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DisasterRecoveryPlan `json:"items"`
}

// DisasterRecoveryPlanSpec defines the desired state of DisasterRecoveryPlan
type DisasterRecoveryPlanSpec struct {
	// Type of disaster recovery plan
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=backup;failover;full-recovery
	// +kubebuilder:default="backup"
	PlanType string `json:"planType"`

	// Description of the disaster recovery plan
	// +optional
	Description string `json:"description,omitempty"`

	// Recovery Time Objective - maximum acceptable downtime
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="1h"
	RTOTarget string `json:"rtoTarget"`

	// Recovery Point Objective - maximum acceptable data loss
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="15m"
	RPOTarget string `json:"rpoTarget"`

	// Priority level for disaster recovery execution
	// +optional
	// +kubebuilder:validation:Enum=low;medium;high;critical
	// +kubebuilder:default="medium"
	Priority Priority `json:"priority,omitempty"`

	// Target components for disaster recovery
	// +optional
	// +kubebuilder:validation:MinItems=1
	TargetComponents []TargetComponent `json:"targetComponents,omitempty"`

	// Backup configuration
	// +optional
	BackupConfig *BackupPolicySpec `json:"backupConfig,omitempty"`

	// Failover configuration
	// +optional
	FailoverConfig *FailoverPolicySpec `json:"failoverConfig,omitempty"`

	// Automation settings
	// +optional
	AutomationConfig *DRAutomationConfig `json:"automationConfig,omitempty"`

	// Validation and testing configuration
	// +optional
	TestingConfig *DRTestingConfig `json:"testingConfig,omitempty"`

	// Notification configuration
	// +optional
	NotificationConfig *DRNotificationConfig `json:"notificationConfig,omitempty"`

	// Resource constraints for DR operations
	// +optional
	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`

	// Dependencies on other disaster recovery plans
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`

	// Regions where this plan applies
	// +optional
	Regions []string `json:"regions,omitempty"`
}

// DRAutomationConfig defines automation settings for disaster recovery
type DRAutomationConfig struct {
	// Enable automatic disaster recovery
	// +kubebuilder:default=false
	AutomaticRecovery bool `json:"automaticRecovery,omitempty"`

	// Conditions that trigger automatic recovery
	// +optional
	TriggerConditions []DRTriggerCondition `json:"triggerConditions,omitempty"`

	// Approval requirements for automatic recovery
	// +optional
	ApprovalConfig *DRApprovalConfig `json:"approvalConfig,omitempty"`

	// Maximum number of automatic recovery attempts
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	MaxRetries *int32 `json:"maxRetries,omitempty"`

	// Cooldown period between recovery attempts
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="10m"
	CooldownPeriod string `json:"cooldownPeriod,omitempty"`
}

// DRTriggerCondition defines conditions that trigger disaster recovery
type DRTriggerCondition struct {
	// Type of trigger condition
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=component-failure;resource-threshold;manual;scheduled
	Type string `json:"type"`

	// Component name (for component-failure type)
	// +optional
	Component string `json:"component,omitempty"`

	// Threshold configuration (for resource-threshold type)
	// +optional
	Threshold *DRThreshold `json:"threshold,omitempty"`

	// Schedule configuration (for scheduled type)
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+$`
	Schedule string `json:"schedule,omitempty"`
}

// DRThreshold defines threshold-based trigger conditions
type DRThreshold struct {
	// Metric name to monitor
	// +kubebuilder:validation:Required
	Metric string `json:"metric"`

	// Operator for comparison
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=gt;gte;lt;lte;eq;ne
	Operator string `json:"operator"`

	// Threshold value
	// +kubebuilder:validation:Required
	Value string `json:"value"`

	// Duration the condition must persist
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="5m"
	Duration string `json:"duration,omitempty"`
}

// DRApprovalConfig defines approval requirements
type DRApprovalConfig struct {
	// Require manual approval before execution
	// +kubebuilder:default=true
	RequireApproval bool `json:"requireApproval,omitempty"`

	// List of approvers (users or groups)
	// +optional
	Approvers []string `json:"approvers,omitempty"`

	// Number of approvals required
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	RequiredApprovals *int32 `json:"requiredApprovals,omitempty"`

	// Approval timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="30m"
	ApprovalTimeout string `json:"approvalTimeout,omitempty"`
}

// DRTestingConfig defines testing and validation settings
type DRTestingConfig struct {
	// Enable regular disaster recovery testing
	// +kubebuilder:default=false
	EnableTesting bool `json:"enableTesting,omitempty"`

	// Schedule for regular DR tests
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+$`
	TestSchedule string `json:"testSchedule,omitempty"`

	// Types of tests to perform
	// +optional
	TestTypes []string `json:"testTypes,omitempty"`

	// Test environment configuration
	// +optional
	TestEnvironment *DRTestEnvironment `json:"testEnvironment,omitempty"`

	// Validation criteria for DR tests
	// +optional
	ValidationCriteria []DRValidationCriterion `json:"validationCriteria,omitempty"`
}

// DRTestEnvironment defines test environment settings
type DRTestEnvironment struct {
	// Namespace for testing
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Namespace string `json:"namespace"`

	// Use isolated environment for testing
	// +kubebuilder:default=true
	Isolated bool `json:"isolated,omitempty"`

	// Resource limits for test environment
	// +optional
	ResourceLimits *ResourceConstraints `json:"resourceLimits,omitempty"`

	// Cleanup policy after testing
	// +optional
	// +kubebuilder:validation:Enum=always;on-success;never
	// +kubebuilder:default="on-success"
	CleanupPolicy string `json:"cleanupPolicy,omitempty"`
}

// DRValidationCriterion defines validation criteria for DR operations
type DRValidationCriterion struct {
	// Name of the validation criterion
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of validation
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=health-check;data-integrity;performance;functional
	Type string `json:"type"`

	// Target for validation (component, service, etc.)
	// +optional
	Target string `json:"target,omitempty"`

	// Validation parameters
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters runtime.RawExtension `json:"parameters,omitempty"`

	// Timeout for validation
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="5m"
	Timeout string `json:"timeout,omitempty"`
}

// DRNotificationConfig defines notification settings
type DRNotificationConfig struct {
	// Enable notifications
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Notification channels
	// +optional
	Channels []DRNotificationChannel `json:"channels,omitempty"`

	// Events that trigger notifications
	// +optional
	Events []string `json:"events,omitempty"`

	// Notification template
	// +optional
	Template string `json:"template,omitempty"`
}

// DRNotificationChannel defines a notification channel
type DRNotificationChannel struct {
	// Name of the notification channel
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of notification channel
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=email;slack;webhook;sms;pagerduty
	Type string `json:"type"`

	// Configuration for the notification channel
	// +kubebuilder:pruning:PreserveUnknownFields
	Config runtime.RawExtension `json:"config"`

	// Severity levels that trigger notifications
	// +optional
	Severities []string `json:"severities,omitempty"`
}

// DisasterRecoveryPlanStatus defines the observed state of DisasterRecoveryPlan
type DisasterRecoveryPlanStatus struct {
	// Current phase of the disaster recovery plan
	// +optional
	// +kubebuilder:validation:Enum=Pending;Active;Testing;Failed;Suspended
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last execution details
	// +optional
	LastExecution *DRExecutionStatus `json:"lastExecution,omitempty"`

	// Test results from the most recent DR test
	// +optional
	LastTestResults *DRTestResults `json:"lastTestResults,omitempty"`

	// Current RTO and RPO metrics
	// +optional
	Metrics *DRMetrics `json:"metrics,omitempty"`

	// Next scheduled execution
	// +optional
	NextScheduledExecution *metav1.Time `json:"nextScheduledExecution,omitempty"`

	// Number of successful executions
	// +optional
	SuccessfulExecutions int32 `json:"successfulExecutions,omitempty"`

	// Number of failed executions
	// +optional
	FailedExecutions int32 `json:"failedExecutions,omitempty"`

	// Observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// DRExecutionStatus defines the status of a DR plan execution
type DRExecutionStatus struct {
	// Execution ID
	ID string `json:"id"`

	// Execution status
	// +kubebuilder:validation:Enum=Running;Completed;Failed;Cancelled
	Status string `json:"status"`

	// Start time
	StartTime metav1.Time `json:"startTime"`

	// End time
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration of execution
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// RTO achieved (actual recovery time)
	// +optional
	RTOAchieved *metav1.Duration `json:"rtoAchieved,omitempty"`

	// Components involved in execution
	// +optional
	Components []string `json:"components,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`

	// Detailed execution steps
	// +optional
	Steps []DRExecutionStep `json:"steps,omitempty"`
}

// DRExecutionStep defines a step in DR plan execution
type DRExecutionStep struct {
	// Step name
	Name string `json:"name"`

	// Step status
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;Skipped
	Status string `json:"status"`

	// Start time
	StartTime metav1.Time `json:"startTime"`

	// End time
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`
}

// DRTestResults defines results from DR testing
type DRTestResults struct {
	// Test ID
	TestID string `json:"testId"`

	// Test timestamp
	Timestamp metav1.Time `json:"timestamp"`

	// Overall test status
	// +kubebuilder:validation:Enum=Passed;Failed;Partial
	Status string `json:"status"`

	// Individual test results
	// +optional
	TestCases []DRTestCase `json:"testCases,omitempty"`

	// RTO achieved during test
	// +optional
	RTOAchieved *metav1.Duration `json:"rtoAchieved,omitempty"`

	// Issues found during testing
	// +optional
	Issues []string `json:"issues,omitempty"`
}

// DRTestCase defines a single test case result
type DRTestCase struct {
	// Test case name
	Name string `json:"name"`

	// Test case status
	// +kubebuilder:validation:Enum=Passed;Failed;Skipped
	Status string `json:"status"`

	// Duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`
}

// DRMetrics defines disaster recovery metrics
type DRMetrics struct {
	// Current RTO (based on last execution)
	// +optional
	CurrentRTO *metav1.Duration `json:"currentRTO,omitempty"`

	// Average RTO over last executions
	// +optional
	AverageRTO *metav1.Duration `json:"averageRTO,omitempty"`

	// Current RPO (time since last backup)
	// +optional
	CurrentRPO *metav1.Duration `json:"currentRPO,omitempty"`

	// Success rate percentage
	// +optional
	SuccessRate *int32 `json:"successRate,omitempty"`

	// Availability percentage
	// +optional
	Availability *int32 `json:"availability,omitempty"`

	// Last updated timestamp
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// BackupPolicy defines automated backup policies
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Retention",type=string,JSONPath=`.spec.retention.dailyBackups`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Last Backup",type=date,JSONPath=`.status.lastBackupTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=bp;backuppol
type BackupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupPolicySpec   `json:"spec,omitempty"`
	Status BackupPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupPolicyList contains a list of BackupPolicy
type BackupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupPolicy `json:"items"`
}

// BackupPolicySpec defines the desired state of BackupPolicy
type BackupPolicySpec struct {
	// Backup schedule in cron format
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+\s+[0-9*\-/,]+$`
	// +kubebuilder:default="0 2 * * *"
	Schedule string `json:"schedule"`

	// Types of backups to perform
	// +optional
	// +kubebuilder:validation:MinItems=1
	BackupTypes []string `json:"backupTypes,omitempty"`

	// Components to include in backup
	// +optional
	Components []TargetComponent `json:"components,omitempty"`

	// Namespaces to backup
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// Storage configuration
	// +kubebuilder:validation:Required
	StorageConfig BackupStorageConfig `json:"storageConfig"`

	// Retention policy
	// +kubebuilder:validation:Required
	Retention BackupRetentionPolicy `json:"retention"`

	// Encryption configuration
	// +optional
	Encryption *BackupEncryptionConfig `json:"encryption,omitempty"`

	// Compression settings
	// +optional
	Compression *BackupCompressionConfig `json:"compression,omitempty"`

	// Validation settings
	// +optional
	Validation *BackupValidationConfig `json:"validation,omitempty"`

	// Resource limits for backup operations
	// +optional
	ResourceLimits *ResourceConstraints `json:"resourceLimits,omitempty"`

	// Pause backup operations
	// +optional
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`
}

// BackupStorageConfig defines storage configuration for backups
type BackupStorageConfig struct {
	// Storage provider type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;gcs;azure;local
	Provider string `json:"provider"`

	// Storage bucket or path
	// +kubebuilder:validation:Required
	Location string `json:"location"`

	// Region for cloud storage
	// +optional
	Region string `json:"region,omitempty"`

	// Storage class
	// +optional
	StorageClass string `json:"storageClass,omitempty"`

	// Credentials secret name
	// +optional
	CredentialsSecret string `json:"credentialsSecret,omitempty"`

	// Additional configuration
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config runtime.RawExtension `json:"config,omitempty"`
}

// BackupRetentionPolicy defines how long backups are kept
type BackupRetentionPolicy struct {
	// Number of daily backups to keep
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=7
	DailyBackups int32 `json:"dailyBackups"`

	// Number of weekly backups to keep
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=4
	WeeklyBackups *int32 `json:"weeklyBackups,omitempty"`

	// Number of monthly backups to keep
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=12
	MonthlyBackups *int32 `json:"monthlyBackups,omitempty"`

	// Number of yearly backups to keep
	// +optional
	// +kubebuilder:validation:Minimum=0
	YearlyBackups *int32 `json:"yearlyBackups,omitempty"`
}

// BackupEncryptionConfig defines encryption settings for backups
type BackupEncryptionConfig struct {
	// Enable encryption
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Encryption algorithm
	// +optional
	// +kubebuilder:validation:Enum=AES-256;AES-128
	// +kubebuilder:default="AES-256"
	Algorithm string `json:"algorithm,omitempty"`

	// Secret containing encryption key
	// +optional
	KeySecret string `json:"keySecret,omitempty"`

	// Key rotation policy
	// +optional
	KeyRotationPolicy *KeyRotationPolicy `json:"keyRotationPolicy,omitempty"`
}

// KeyRotationPolicy defines key rotation settings
type KeyRotationPolicy struct {
	// Enable automatic key rotation
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Rotation interval
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(d|w|m|y)$`
	// +kubebuilder:default="90d"
	RotationInterval string `json:"rotationInterval,omitempty"`

	// Number of old keys to keep
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	KeepOldKeys *int32 `json:"keepOldKeys,omitempty"`
}

// BackupCompressionConfig defines compression settings
type BackupCompressionConfig struct {
	// Enable compression
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Compression algorithm
	// +optional
	// +kubebuilder:validation:Enum=gzip;lz4;zstd
	// +kubebuilder:default="gzip"
	Algorithm string `json:"algorithm,omitempty"`

	// Compression level (1-9 for gzip)
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	// +kubebuilder:default=6
	Level *int32 `json:"level,omitempty"`
}

// BackupValidationConfig defines validation settings for backups
type BackupValidationConfig struct {
	// Enable backup validation
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Validation types to perform
	// +optional
	ValidationTypes []string `json:"validationTypes,omitempty"`

	// Checksum validation
	// +optional
	// +kubebuilder:default=true
	ChecksumValidation *bool `json:"checksumValidation,omitempty"`

	// Restore validation (test restore)
	// +optional
	// +kubebuilder:default=false
	RestoreValidation *bool `json:"restoreValidation,omitempty"`

	// Validation schedule (separate from backup schedule)
	// +optional
	ValidationSchedule string `json:"validationSchedule,omitempty"`
}

// BackupPolicyStatus defines the observed state of BackupPolicy
type BackupPolicyStatus struct {
	// Current phase
	// +optional
	// +kubebuilder:validation:Enum=Active;Paused;Failed;Suspended
	Phase string `json:"phase,omitempty"`

	// Conditions
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last backup time
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// Next scheduled backup time
	// +optional
	NextBackupTime *metav1.Time `json:"nextBackupTime,omitempty"`

	// Last backup status
	// +optional
	LastBackupStatus *BackupExecutionStatus `json:"lastBackupStatus,omitempty"`

	// Number of successful backups
	// +optional
	SuccessfulBackups int32 `json:"successfulBackups,omitempty"`

	// Number of failed backups
	// +optional
	FailedBackups int32 `json:"failedBackups,omitempty"`

	// Current backup size
	// +optional
	CurrentBackupSize *resource.Quantity `json:"currentBackupSize,omitempty"`

	// Total storage used
	// +optional
	TotalStorageUsed *resource.Quantity `json:"totalStorageUsed,omitempty"`

	// Observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// BackupExecutionStatus defines the status of a backup execution
type BackupExecutionStatus struct {
	// Backup ID
	BackupID string `json:"backupId"`

	// Status
	// +kubebuilder:validation:Enum=Running;Completed;Failed;Cancelled
	Status string `json:"status"`

	// Start time
	StartTime metav1.Time `json:"startTime"`

	// End time
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// Backup size
	// +optional
	Size *resource.Quantity `json:"size,omitempty"`

	// Components included
	// +optional
	Components []string `json:"components,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`
}

// FailoverPolicy defines automated failover policies
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Primary Region",type=string,JSONPath=`.spec.primaryRegion`
// +kubebuilder:printcolumn:name="Auto Failover",type=boolean,JSONPath=`.spec.autoFailover`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Current Region",type=string,JSONPath=`.status.currentActiveRegion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=fp;failoverpol
type FailoverPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FailoverPolicySpec   `json:"spec,omitempty"`
	Status FailoverPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FailoverPolicyList contains a list of FailoverPolicy
type FailoverPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FailoverPolicy `json:"items"`
}

// FailoverPolicySpec defines the desired state of FailoverPolicy
type FailoverPolicySpec struct {
	// Primary region
	// +kubebuilder:validation:Required
	PrimaryRegion string `json:"primaryRegion"`

	// Failover regions in order of preference
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	FailoverRegions []string `json:"failoverRegions"`

	// Enable automatic failover
	// +optional
	// +kubebuilder:default=false
	AutoFailover bool `json:"autoFailover,omitempty"`

	// Failover trigger conditions
	// +optional
	TriggerConditions []FailoverTriggerCondition `json:"triggerConditions,omitempty"`

	// Health check configuration
	// +optional
	HealthCheck *FailoverHealthCheck `json:"healthCheck,omitempty"`

	// DNS failover configuration
	// +optional
	DNSConfig *FailoverDNSConfig `json:"dnsConfig,omitempty"`

	// Traffic splitting configuration
	// +optional
	TrafficConfig *FailoverTrafficConfig `json:"trafficConfig,omitempty"`

	// Data synchronization requirements
	// +optional
	DataSyncConfig *FailoverDataSyncConfig `json:"dataSyncConfig,omitempty"`

	// Rollback configuration
	// +optional
	RollbackConfig *FailoverRollbackConfig `json:"rollbackConfig,omitempty"`

	// Components to include in failover
	// +optional
	Components []TargetComponent `json:"components,omitempty"`

	// RTO target for failover
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="1h"
	RTOTarget string `json:"rtoTarget,omitempty"`

	// Resource constraints for failover operations
	// +optional
	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`
}

// FailoverTriggerCondition defines conditions that trigger failover
type FailoverTriggerCondition struct {
	// Condition name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Condition type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=health-check;metric-threshold;manual;external
	Type string `json:"type"`

	// Health check configuration (for health-check type)
	// +optional
	HealthCheckConfig *FailoverHealthCheck `json:"healthCheckConfig,omitempty"`

	// Metric threshold configuration (for metric-threshold type)
	// +optional
	MetricThreshold *FailoverMetricThreshold `json:"metricThreshold,omitempty"`

	// External webhook configuration (for external type)
	// +optional
	WebhookConfig *FailoverWebhookConfig `json:"webhookConfig,omitempty"`

	// Severity level
	// +optional
	// +kubebuilder:validation:Enum=low;medium;high;critical
	// +kubebuilder:default="medium"
	Severity string `json:"severity,omitempty"`

	// Cooldown period after triggering
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="10m"
	Cooldown string `json:"cooldown,omitempty"`
}

// FailoverHealthCheck defines health check configuration
type FailoverHealthCheck struct {
	// Endpoints to check
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Endpoints []string `json:"endpoints"`

	// Check interval
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="30s"
	Interval string `json:"interval,omitempty"`

	// Timeout for each check
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="10s"
	Timeout string `json:"timeout,omitempty"`

	// Number of failed checks before triggering
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`

	// Number of successful checks to recover
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	SuccessThreshold *int32 `json:"successThreshold,omitempty"`
}

// FailoverMetricThreshold defines metric-based trigger conditions
type FailoverMetricThreshold struct {
	// Metric name
	// +kubebuilder:validation:Required
	MetricName string `json:"metricName"`

	// Metric query
	// +kubebuilder:validation:Required
	Query string `json:"query"`

	// Threshold value
	// +kubebuilder:validation:Required
	Threshold string `json:"threshold"`

	// Comparison operator
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=gt;gte;lt;lte;eq;ne
	Operator string `json:"operator"`

	// Duration the condition must persist
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="5m"
	Duration string `json:"duration,omitempty"`
}

// FailoverWebhookConfig defines webhook-based trigger conditions
type FailoverWebhookConfig struct {
	// Webhook URL
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// HTTP method
	// +optional
	// +kubebuilder:validation:Enum=GET;POST;PUT
	// +kubebuilder:default="POST"
	Method string `json:"method,omitempty"`

	// Request headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Request body template
	// +optional
	Body string `json:"body,omitempty"`

	// Expected response codes for success
	// +optional
	ExpectedCodes []int32 `json:"expectedCodes,omitempty"`

	// Timeout for webhook call
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="30s"
	Timeout string `json:"timeout,omitempty"`
}

// FailoverDNSConfig defines DNS failover configuration
type FailoverDNSConfig struct {
	// DNS provider
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=route53;cloudflare;external
	Provider string `json:"provider"`

	// Zone ID or domain name
	// +kubebuilder:validation:Required
	Zone string `json:"zone"`

	// DNS records to manage
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Records []FailoverDNSRecord `json:"records"`

	// TTL for DNS records
	// +optional
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:default=300
	TTL *int32 `json:"ttl,omitempty"`

	// Credentials secret
	// +optional
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

// FailoverDNSRecord defines a DNS record for failover
type FailoverDNSRecord struct {
	// Record name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Record type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=A;AAAA;CNAME
	Type string `json:"type"`

	// Regional values (region -> value mapping)
	// +kubebuilder:validation:Required
	RegionalValues map[string]string `json:"regionalValues"`

	// Health check configuration for this record
	// +optional
	HealthCheck *FailoverDNSHealthCheck `json:"healthCheck,omitempty"`
}

// FailoverDNSHealthCheck defines DNS-level health checking
type FailoverDNSHealthCheck struct {
	// Enable DNS health checking
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Health check endpoint
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// Port for health check
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=80
	Port *int32 `json:"port,omitempty"`

	// Protocol for health check
	// +optional
	// +kubebuilder:validation:Enum=HTTP;HTTPS;TCP
	// +kubebuilder:default="HTTP"
	Protocol string `json:"protocol,omitempty"`

	// Path for HTTP(S) health checks
	// +optional
	// +kubebuilder:default="/healthz"
	Path string `json:"path,omitempty"`
}

// FailoverTrafficConfig defines traffic management during failover
type FailoverTrafficConfig struct {
	// Traffic splitting mode
	// +optional
	// +kubebuilder:validation:Enum=immediate;gradual;canary
	// +kubebuilder:default="immediate"
	Mode string `json:"mode,omitempty"`

	// Gradual failover configuration
	// +optional
	GradualConfig *FailoverGradualConfig `json:"gradualConfig,omitempty"`

	// Canary failover configuration
	// +optional
	CanaryConfig *FailoverCanaryConfig `json:"canaryConfig,omitempty"`

	// Load balancer configuration
	// +optional
	LoadBalancerConfig *FailoverLoadBalancerConfig `json:"loadBalancerConfig,omitempty"`
}

// FailoverGradualConfig defines gradual traffic shift configuration
type FailoverGradualConfig struct {
	// Traffic shift steps (percentages)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Steps []int32 `json:"steps"`

	// Duration between steps
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	StepDuration string `json:"stepDuration"`

	// Success criteria to proceed to next step
	// +optional
	SuccessCriteria []FailoverSuccessCriterion `json:"successCriteria,omitempty"`
}

// FailoverCanaryConfig defines canary failover configuration
type FailoverCanaryConfig struct {
	// Canary traffic percentage
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=50
	// +kubebuilder:default=10
	CanaryPercentage int32 `json:"canaryPercentage"`

	// Duration of canary phase
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	Duration string `json:"duration"`

	// Success criteria for promoting canary
	// +optional
	SuccessCriteria []FailoverSuccessCriterion `json:"successCriteria,omitempty"`

	// Automatic promotion if criteria are met
	// +optional
	// +kubebuilder:default=false
	AutoPromote bool `json:"autoPromote,omitempty"`
}

// FailoverSuccessCriterion defines success criteria for failover phases
type FailoverSuccessCriterion struct {
	// Criterion name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Metric to evaluate
	// +kubebuilder:validation:Required
	Metric string `json:"metric"`

	// Threshold for success
	// +kubebuilder:validation:Required
	Threshold string `json:"threshold"`

	// Comparison operator
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=gt;gte;lt;lte;eq;ne
	Operator string `json:"operator"`
}

// FailoverLoadBalancerConfig defines load balancer configuration
type FailoverLoadBalancerConfig struct {
	// Load balancer type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=nginx;haproxy;envoy;aws-alb;gcp-lb;azure-lb
	Type string `json:"type"`

	// Configuration template
	// +optional
	ConfigTemplate string `json:"configTemplate,omitempty"`

	// Health check configuration
	// +optional
	HealthCheck *FailoverHealthCheck `json:"healthCheck,omitempty"`
}

// FailoverDataSyncConfig defines data synchronization requirements
type FailoverDataSyncConfig struct {
	// Enable data synchronization before failover
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Data sources to synchronize
	// +optional
	DataSources []FailoverDataSource `json:"dataSources,omitempty"`

	// Synchronization timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="5m"
	SyncTimeout string `json:"syncTimeout,omitempty"`

	// Consistency level required
	// +optional
	// +kubebuilder:validation:Enum=eventual;strong;session
	// +kubebuilder:default="eventual"
	ConsistencyLevel string `json:"consistencyLevel,omitempty"`
}

// FailoverDataSource defines a data source for synchronization
type FailoverDataSource struct {
	// Data source name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Data source type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=database;storage;cache;custom
	Type string `json:"type"`

	// Regional endpoints for the data source
	// +kubebuilder:validation:Required
	RegionalEndpoints map[string]string `json:"regionalEndpoints"`

	// Synchronization method
	// +optional
	// +kubebuilder:validation:Enum=async;sync;custom
	// +kubebuilder:default="async"
	SyncMethod string `json:"syncMethod,omitempty"`

	// Custom synchronization configuration
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	SyncConfig runtime.RawExtension `json:"syncConfig,omitempty"`
}

// FailoverRollbackConfig defines rollback configuration
type FailoverRollbackConfig struct {
	// Enable automatic rollback on failure
	// +kubebuilder:default=true
	AutoRollback bool `json:"autoRollback,omitempty"`

	// Rollback trigger conditions
	// +optional
	TriggerConditions []FailoverTriggerCondition `json:"triggerConditions,omitempty"`

	// Rollback timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="10m"
	RollbackTimeout string `json:"rollbackTimeout,omitempty"`

	// Data recovery requirements
	// +optional
	DataRecovery *FailoverDataRecovery `json:"dataRecovery,omitempty"`
}

// FailoverDataRecovery defines data recovery during rollback
type FailoverDataRecovery struct {
	// Enable data recovery
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Point-in-time recovery target
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	RecoveryTarget string `json:"recoveryTarget,omitempty"`

	// Recovery validation
	// +optional
	Validation *FailoverRecoveryValidation `json:"validation,omitempty"`
}

// FailoverRecoveryValidation defines validation for data recovery
type FailoverRecoveryValidation struct {
	// Enable validation
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Validation checks
	// +optional
	Checks []string `json:"checks,omitempty"`

	// Validation timeout
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(h|m|s)$`
	// +kubebuilder:default="5m"
	Timeout string `json:"timeout,omitempty"`
}

// FailoverPolicyStatus defines the observed state of FailoverPolicy
type FailoverPolicyStatus struct {
	// Current phase
	// +optional
	// +kubebuilder:validation:Enum=Active;Failed;Failover;Rollback;Suspended
	Phase string `json:"phase,omitempty"`

	// Current active region
	// +optional
	CurrentActiveRegion string `json:"currentActiveRegion,omitempty"`

	// Conditions
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last failover details
	// +optional
	LastFailover *FailoverExecutionStatus `json:"lastFailover,omitempty"`

	// Regional health status
	// +optional
	RegionHealth map[string]RegionHealthInfo `json:"regionHealth,omitempty"`

	// Failover history (last 10 executions)
	// +optional
	FailoverHistory []FailoverExecutionStatus `json:"failoverHistory,omitempty"`

	// Current RTO metrics
	// +optional
	RTOMetrics *FailoverRTOMetrics `json:"rtoMetrics,omitempty"`

	// Observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// FailoverExecutionStatus defines the status of a failover execution
type FailoverExecutionStatus struct {
	// Execution ID
	ID string `json:"id"`

	// Trigger reason
	TriggerReason string `json:"triggerReason"`

	// Source region
	SourceRegion string `json:"sourceRegion"`

	// Target region
	TargetRegion string `json:"targetRegion"`

	// Status
	// +kubebuilder:validation:Enum=Running;Completed;Failed;Cancelled;Rolled-back
	Status string `json:"status"`

	// Start time
	StartTime metav1.Time `json:"startTime"`

	// End time
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// RTO achieved
	// +optional
	RTOAchieved *metav1.Duration `json:"rtoAchieved,omitempty"`

	// Failover steps
	// +optional
	Steps []FailoverExecutionStep `json:"steps,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`
}

// FailoverExecutionStep defines a step in failover execution
type FailoverExecutionStep struct {
	// Step name
	Name string `json:"name"`

	// Step type
	// +kubebuilder:validation:Enum=dns;traffic;data-sync;validation;cleanup
	Type string `json:"type"`

	// Status
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;Skipped
	Status string `json:"status"`

	// Start time
	StartTime metav1.Time `json:"startTime"`

	// End time
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// Error message if failed
	// +optional
	Error string `json:"error,omitempty"`
}

// RegionHealthInfo defines health information for a region
type RegionHealthInfo struct {
	// Health status
	// +kubebuilder:validation:Enum=Healthy;Unhealthy;Unknown
	Status string `json:"status"`

	// Last check time
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Response time
	// +optional
	ResponseTime *metav1.Duration `json:"responseTime,omitempty"`

	// Error count
	// +optional
	ErrorCount int32 `json:"errorCount,omitempty"`

	// Health check details
	// +optional
	HealthCheckDetails map[string]string `json:"healthCheckDetails,omitempty"`
}

// FailoverRTOMetrics defines RTO metrics for failover
type FailoverRTOMetrics struct {
	// Target RTO
	// +optional
	TargetRTO *metav1.Duration `json:"targetRTO,omitempty"`

	// Last achieved RTO
	// +optional
	LastRTO *metav1.Duration `json:"lastRTO,omitempty"`

	// Average RTO over last 10 failovers
	// +optional
	AverageRTO *metav1.Duration `json:"averageRTO,omitempty"`

	// Best RTO achieved
	// +optional
	BestRTO *metav1.Duration `json:"bestRTO,omitempty"`

	// Worst RTO achieved
	// +optional
	WorstRTO *metav1.Duration `json:"worstRTO,omitempty"`

	// Success rate percentage
	// +optional
	SuccessRate *int32 `json:"successRate,omitempty"`
}

func init() {
	SchemeBuilder.Register(&DisasterRecoveryPlan{}, &DisasterRecoveryPlanList{})
	SchemeBuilder.Register(&BackupPolicy{}, &BackupPolicyList{})
	SchemeBuilder.Register(&FailoverPolicy{}, &FailoverPolicyList{})
}
