package disaster_recovery

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	nephoran "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// DisasterType defines types of disasters.

type DisasterType string

const (

	// DisasterTypeDataCorruption holds disastertypedatacorruption value.

	DisasterTypeDataCorruption DisasterType = "data_corruption"

	// DisasterTypeControllerFailure holds disastertypecontrollerfailure value.

	DisasterTypeControllerFailure DisasterType = "controller_failure"

	// DisasterTypeEtcdFailure holds disastertypeetcdfailure value.

	DisasterTypeEtcdFailure DisasterType = "etcd_failure"

	// DisasterTypeNodeFailure holds disastertypenodefailure value.

	DisasterTypeNodeFailure DisasterType = "node_failure"

	// DisasterTypeNetworkPartition holds disastertypenetworkpartition value.

	DisasterTypeNetworkPartition DisasterType = "network_partition"

	// DisasterTypeCompleteClusterLoss holds disastertypecompleteclusterloss value.

	DisasterTypeCompleteClusterLoss DisasterType = "complete_cluster_loss"

	// DisasterTypeStorageFailure holds disastertypestoragefailure value.

	DisasterTypeStorageFailure DisasterType = "storage_failure"

	// DisasterTypeBackupCorruption holds disastertypebackupcorruption value.

	DisasterTypeBackupCorruption DisasterType = "backup_corruption"
)

// RecoveryStrategy defines recovery approaches.

type RecoveryStrategy string

const (

	// RecoveryStrategyBackupRestore holds recoverystrategybackuprestore value.

	RecoveryStrategyBackupRestore RecoveryStrategy = "backup_restore"

	// RecoveryStrategyFailover holds recoverystrategyfailover value.

	RecoveryStrategyFailover RecoveryStrategy = "failover"

	// RecoveryStrategyReplication holds recoverystrategyreplication value.

	RecoveryStrategyReplication RecoveryStrategy = "replication"

	// RecoveryStrategyReconciliation holds recoverystrategyreconciliation value.

	RecoveryStrategyReconciliation RecoveryStrategy = "reconciliation"

	// RecoveryStrategyManualIntervention holds recoverystrategymanualintervention value.

	RecoveryStrategyManualIntervention RecoveryStrategy = "manual_intervention"
)

// DisasterScenario represents a disaster recovery test scenario.

// Note: Function fields are defined with interface{} to avoid circular dependency.

// The actual functions should match the signature func(*DisasterRecoveryTestSuite) error.

type DisasterScenario struct {
	Name string

	Description string

	DisasterType DisasterType

	AffectedComponents []string

	RecoveryStrategy RecoveryStrategy

	ExpectedRTO time.Duration // Recovery Time Objective

	ExpectedRPO time.Duration // Recovery Point Objective

	PreConditions interface{} // func(*DisasterRecoveryTestSuite) error

	InjectDisaster interface{} // func(*DisasterRecoveryTestSuite) error

	ValidateFailure interface{} // func(*DisasterRecoveryTestSuite) error

	ExecuteRecovery interface{} // func(*DisasterRecoveryTestSuite) error

	ValidateRecovery interface{} // func(*DisasterRecoveryTestSuite) error

	PostRecoveryCleanup interface{} // func(*DisasterRecoveryTestSuite) error

}

// TestDataSet contains test data for disaster recovery scenarios.

type TestDataSet struct {
	NetworkIntents []nephoran.NetworkIntent

	E2NodeSets []nephoran.E2NodeSet

	Deployments []appsv1.Deployment

	Services []corev1.Service

	ConfigMaps []corev1.ConfigMap

	Secrets []corev1.Secret
}

// BackupMetadata contains backup information.

type BackupMetadata struct {
	Timestamp time.Time

	BackupID string

	Components []string

	DataIntegrity string

	CompressionRatio float64

	BackupSize int64

	BackupPath string
}

// RecoveryMetrics tracks recovery performance.

type RecoveryMetrics struct {
	StartTime time.Time

	EndTime time.Time

	ActualRTO time.Duration

	ActualRPO time.Duration

	DataLossOccurred bool

	ComponentsRecovered int

	ComponentsFailed int

	RecoverySuccess bool

	ErrorMessages []string
}
