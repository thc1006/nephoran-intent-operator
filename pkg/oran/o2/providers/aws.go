package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderTypeAWS is defined in interface.go

// AWSProvider implements CloudProvider for Amazon Web Services
type AWSProvider struct {
	name             string
	config           *ProviderConfiguration
	awsConfig        aws.Config
	ec2Client        *ec2.Client
	eksClient        *eks.Client
	ecsClient        *ecs.Client
	s3Client         *s3.Client
	rdsClient        *rds.Client
	cfnClient        *cloudformation.Client
	elbClient        *elasticloadbalancingv2.Client
	cloudwatchClient *cloudwatch.Client
	iamClient        *iam.Client
	stsClient        *sts.Client
	connected        bool
	eventCallback    EventCallback
	stopChannel      chan struct{}
	mutex            sync.RWMutex
	accountID        string
	region           string
}

// NewAWSProvider creates a new AWS provider instance
func NewAWSProvider(config *ProviderConfiguration) (CloudProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required for AWS provider")
	}

	if config.Type != ProviderTypeAWS {
		return nil, fmt.Errorf("invalid provider type: expected %s, got %s", ProviderTypeAWS, config.Type)
	}

	provider := &AWSProvider{
		name:        config.Name,
		config:      config,
		stopChannel: make(chan struct{}),
		region:      config.Region,
	}

	return provider, nil
}

// GetProviderInfo returns information about this AWS provider
func (a *AWSProvider) GetProviderInfo() *ProviderInfo {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return &ProviderInfo{
		Name:        a.name,
		Type:        ProviderTypeAWS,
		Version:     "1.0.0",
		Description: "Amazon Web Services cloud provider",
		Vendor:      "Amazon",
		Region:      a.region,
		Endpoint:    fmt.Sprintf("https://%s.amazonaws.com", a.region),
		Tags: map[string]string{
			"region":     a.region,
			"account_id": a.accountID,
		},
		LastUpdated: time.Now(),
	}
}

// GetSupportedResourceTypes returns the resource types supported by AWS
func (a *AWSProvider) GetSupportedResourceTypes() []string {
	return []string{
		"ec2_instance",
		"eks_cluster",
		"ecs_cluster",
		"ecs_service",
		"s3_bucket",
		"rds_instance",
		"vpc",
		"subnet",
		"security_group",
		"load_balancer",
		"auto_scaling_group",
		"lambda_function",
		"cloudformation_stack",
		"iam_role",
		"ebs_volume",
		"efs_filesystem",
		"elasticache_cluster",
		"api_gateway",
		"cloudfront_distribution",
		"route53_zone",
	}
}

// GetCapabilities returns the capabilities of this AWS provider
func (a *AWSProvider) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{
		ComputeTypes:     []string{"ec2_instance", "lambda_function", "ecs_task", "eks_node"},
		StorageTypes:     []string{"s3_bucket", "ebs_volume", "efs_filesystem", "fsx"},
		NetworkTypes:     []string{"vpc", "subnet", "security_group", "load_balancer", "api_gateway"},
		AcceleratorTypes: []string{"gpu", "fpga", "inferentia"},

		AutoScaling:    true,
		LoadBalancing:  true,
		Monitoring:     true,
		Logging:        true,
		Networking:     true,
		StorageClasses: true,

		HorizontalPodAutoscaling: true, // EKS
		VerticalPodAutoscaling:   true, // EKS
		ClusterAutoscaling:       true, // EKS/ECS

		Namespaces:      true, // EKS
		ResourceQuotas:  true, // Service Quotas
		NetworkPolicies: true, // Security Groups/NACLs
		RBAC:            true, // IAM

		MultiZone:        true, // Availability Zones
		MultiRegion:      true, // Global services
		BackupRestore:    true, // AWS Backup
		DisasterRecovery: true, // Multi-region

		Encryption:       true, // KMS
		SecretManagement: true, // Secrets Manager
		ImageScanning:    true, // ECR scanning
		PolicyEngine:     true, // IAM policies

		MaxNodes:    10000,  // EKS limit
		MaxPods:     750000, // EKS with multiple node groups
		MaxServices: 100000, // Practical limit
		MaxVolumes:  500000, // EBS volumes
	}
}

// Connect establishes connection to AWS
func (a *AWSProvider) Connect(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("connecting to AWS", "region", a.region)

	// Load AWS configuration
	cfg, err := a.loadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	a.awsConfig = cfg

	// Initialize service clients
	a.initializeClients()

	// Get account ID
	stsOutput, err := a.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get AWS account identity: %w", err)
	}
	a.accountID = *stsOutput.Account

	a.mutex.Lock()
	a.connected = true
	a.mutex.Unlock()

	logger.Info("successfully connected to AWS", "account", a.accountID)
	return nil
}

// loadAWSConfig loads AWS configuration based on provider config
func (a *AWSProvider) loadAWSConfig(ctx context.Context) (aws.Config, error) {
	var optFns []func(*config.LoadOptions) error

	// Set region
	if a.region != "" {
		optFns = append(optFns, config.WithRegion(a.region))
	}

	// Set credentials if provided
	if accessKey, exists := a.config.Credentials["access_key_id"]; exists {
		secretKey := a.config.Credentials["secret_access_key"]
		sessionToken := a.config.Credentials["session_token"]

		optFns = append(optFns, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
		))
	}

	// Use profile if specified
	if profile, exists := a.config.Credentials["profile"]; exists {
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	// Use role ARN if specified
	if roleARN, exists := a.config.Credentials["role_arn"]; exists {
		// Role assumption would be configured here
		_ = roleARN // Placeholder
	}

	return config.LoadDefaultConfig(ctx, optFns...)
}

// initializeClients initializes AWS service clients
func (a *AWSProvider) initializeClients() {
	a.ec2Client = ec2.NewFromConfig(a.awsConfig)
	a.eksClient = eks.NewFromConfig(a.awsConfig)
	a.ecsClient = ecs.NewFromConfig(a.awsConfig)
	a.s3Client = s3.NewFromConfig(a.awsConfig)
	a.rdsClient = rds.NewFromConfig(a.awsConfig)
	a.cfnClient = cloudformation.NewFromConfig(a.awsConfig)
	a.elbClient = elasticloadbalancingv2.NewFromConfig(a.awsConfig)
	a.cloudwatchClient = cloudwatch.NewFromConfig(a.awsConfig)
	a.iamClient = iam.NewFromConfig(a.awsConfig)
	a.stsClient = sts.NewFromConfig(a.awsConfig)
}

// Disconnect closes the connection to AWS
func (a *AWSProvider) Disconnect(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("disconnecting from AWS")

	a.mutex.Lock()
	a.connected = false
	a.mutex.Unlock()

	// Stop event watching if running
	select {
	case a.stopChannel <- struct{}{}:
	default:
	}

	logger.Info("disconnected from AWS")
	return nil
}

// HealthCheck performs a health check on AWS services
func (a *AWSProvider) HealthCheck(ctx context.Context) error {
	// Check EC2 service
	_, err := a.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return fmt.Errorf("health check failed: unable to access EC2 service: %w", err)
	}

	// Check S3 service
	_, err = a.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("health check failed: unable to access S3 service: %w", err)
	}

	// Check IAM service
	_, err = a.iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		// It's okay if we can't get user info (might be using role)
		// Just check if we can make IAM calls
		_, err = a.iamClient.ListRoles(ctx, &iam.ListRolesInput{
			MaxItems: aws.Int32(1),
		})
		if err != nil {
			return fmt.Errorf("health check failed: unable to access IAM service: %w", err)
		}
	}

	return nil
}

// Close closes any resources held by the provider
func (a *AWSProvider) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Stop event watching
	select {
	case a.stopChannel <- struct{}{}:
	default:
	}

	a.connected = false
	return nil
}

// CreateResource creates a new AWS resource
func (a *AWSProvider) CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating AWS resource", "type", req.Type, "name", req.Name)

	switch req.Type {
	case "ec2_instance":
		return a.createEC2Instance(ctx, req)
	case "s3_bucket":
		return a.createS3Bucket(ctx, req)
	case "vpc":
		return a.createVPC(ctx, req)
	case "security_group":
		return a.createSecurityGroup(ctx, req)
	case "ebs_volume":
		return a.createEBSVolume(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", req.Type)
	}
}

// GetResource retrieves an AWS resource
func (a *AWSProvider) GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting AWS resource", "resourceID", resourceID)

	// Parse resourceID format: type/id
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "ec2_instance":
		return a.getEC2Instance(ctx, id)
	case "s3_bucket":
		return a.getS3Bucket(ctx, id)
	case "vpc":
		return a.getVPC(ctx, id)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// UpdateResource updates an AWS resource
func (a *AWSProvider) UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)
	logger.Info("updating AWS resource", "resourceID", resourceID)

	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "ec2_instance":
		return a.updateEC2Instance(ctx, id, req)
	case "s3_bucket":
		return a.updateS3Bucket(ctx, id, req)
	default:
		return nil, fmt.Errorf("resource type %s does not support updates", resourceType)
	}
}

// DeleteResource deletes an AWS resource
func (a *AWSProvider) DeleteResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting AWS resource", "resourceID", resourceID)

	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "ec2_instance":
		return a.deleteEC2Instance(ctx, id)
	case "s3_bucket":
		return a.deleteS3Bucket(ctx, id)
	case "vpc":
		return a.deleteVPC(ctx, id)
	case "ebs_volume":
		return a.deleteEBSVolume(ctx, id)
	default:
		return fmt.Errorf("unsupported resource type for deletion: %s", resourceType)
	}
}

// ListResources lists AWS resources with optional filtering
func (a *AWSProvider) ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("listing AWS resources", "filter", filter)

	var resources []*ResourceResponse

	resourceTypes := filter.Types
	if len(resourceTypes) == 0 {
		// Default to common resource types
		resourceTypes = []string{"ec2_instance", "s3_bucket", "vpc"}
	}

	for _, resourceType := range resourceTypes {
		typeResources, err := a.listResourcesByType(ctx, resourceType, filter)
		if err != nil {
			logger.Error(err, "failed to list resources", "type", resourceType)
			continue
		}
		resources = append(resources, typeResources...)
	}

	return resources, nil
}

// Deploy creates a deployment using CloudFormation or other orchestration
func (a *AWSProvider) Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	logger := log.FromContext(ctx)
	logger.Info("deploying template", "name", req.Name, "type", req.TemplateType)

	switch req.TemplateType {
	case "cloudformation", "cfn":
		return a.deployCloudFormationStack(ctx, req)
	case "cdk":
		return a.deployCDKStack(ctx, req)
	case "terraform":
		return nil, fmt.Errorf("terraform deployment requires external tooling")
	default:
		return nil, fmt.Errorf("unsupported template type: %s", req.TemplateType)
	}
}

// GetDeployment retrieves a deployment (CloudFormation stack)
func (a *AWSProvider) GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error) {
	return a.getCloudFormationStack(ctx, deploymentID)
}

// UpdateDeployment updates a deployment
func (a *AWSProvider) UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {
	return a.updateCloudFormationStack(ctx, deploymentID, req)
}

// DeleteDeployment deletes a deployment
func (a *AWSProvider) DeleteDeployment(ctx context.Context, deploymentID string) error {
	return a.deleteCloudFormationStack(ctx, deploymentID)
}

// ListDeployments lists deployments (CloudFormation stacks)
func (a *AWSProvider) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {
	return a.listCloudFormationStacks(ctx, filter)
}

// ScaleResource scales an AWS resource
func (a *AWSProvider) ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling AWS resource", "resourceID", resourceID, "direction", req.Direction)

	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "auto_scaling_group":
		return a.scaleAutoScalingGroup(ctx, id, req)
	case "ecs_service":
		return a.scaleECSService(ctx, id, req)
	case "eks_nodegroup":
		return a.scaleEKSNodeGroup(ctx, id, req)
	default:
		return fmt.Errorf("scaling not supported for resource type: %s", resourceType)
	}
}

// GetScalingCapabilities returns scaling capabilities for a resource
func (a *AWSProvider) GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error) {
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType := parts[0]

	switch resourceType {
	case "auto_scaling_group":
		return &ScalingCapabilities{
			HorizontalScaling: true,
			VerticalScaling:   false,
			MinReplicas:       0,
			MaxReplicas:       1000,
			SupportedMetrics:  []string{"cpu", "memory", "network", "custom"},
			ScaleUpCooldown:   60 * time.Second,
			ScaleDownCooldown: 300 * time.Second,
		}, nil
	case "ecs_service":
		return &ScalingCapabilities{
			HorizontalScaling: true,
			VerticalScaling:   true,
			MinReplicas:       0,
			MaxReplicas:       1000,
			SupportedMetrics:  []string{"cpu", "memory", "alb_requests"},
			ScaleUpCooldown:   60 * time.Second,
			ScaleDownCooldown: 300 * time.Second,
		}, nil
	default:
		return &ScalingCapabilities{
			HorizontalScaling: false,
			VerticalScaling:   false,
		}, nil
	}
}

// GetMetrics returns cloud-level metrics
func (a *AWSProvider) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get EC2 instance count
	ec2Result, err := a.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err == nil {
		totalInstances := 0
		runningInstances := 0
		for _, reservation := range ec2Result.Reservations {
			for _, instance := range reservation.Instances {
				totalInstances++
				if instance.State != nil && *instance.State.Name == "running" {
					runningInstances++
				}
			}
		}
		metrics["ec2_instances_total"] = totalInstances
		metrics["ec2_instances_running"] = runningInstances
	}

	// Get VPC count
	vpcResult, err := a.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err == nil {
		metrics["vpcs_total"] = len(vpcResult.Vpcs)
	}

	// Get S3 bucket count
	s3Result, err := a.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err == nil {
		metrics["s3_buckets_total"] = len(s3Result.Buckets)
	}

	// Get EBS volume count
	volumeResult, err := a.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{})
	if err == nil {
		totalVolumes := len(volumeResult.Volumes)
		totalVolumeGB := 0
		for _, volume := range volumeResult.Volumes {
			if volume.Size != nil {
				totalVolumeGB += int(*volume.Size)
			}
		}
		metrics["ebs_volumes_total"] = totalVolumes
		metrics["ebs_volume_gb_total"] = totalVolumeGB
	}

	// Get RDS instance count
	rdsResult, err := a.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err == nil {
		metrics["rds_instances_total"] = len(rdsResult.DBInstances)
	}

	metrics["region"] = a.region
	metrics["account_id"] = a.accountID
	metrics["timestamp"] = time.Now().Unix()

	return metrics, nil
}

// GetResourceMetrics returns metrics for a specific resource
func (a *AWSProvider) GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]
	metrics := make(map[string]interface{})

	switch resourceType {
	case "ec2_instance":
		// Get instance details
		result, err := a.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{id},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
			instance := result.Reservations[0].Instances[0]
			metrics["state"] = *instance.State.Name
			metrics["instance_type"] = *instance.InstanceType
			if instance.CpuOptions != nil {
				metrics["vcpus"] = *instance.CpuOptions.CoreCount * *instance.CpuOptions.ThreadsPerCore
			}
			metrics["launch_time"] = instance.LaunchTime
		}

		// Get CloudWatch metrics for the instance
		// This would involve querying CloudWatch for CPU, network, disk metrics

	case "s3_bucket":
		// Get bucket metrics
		// This would involve CloudWatch metrics for bucket size, requests, etc.

	case "rds_instance":
		// Get RDS instance metrics
		result, err := a.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(id),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get RDS instance: %w", err)
		}

		if len(result.DBInstances) > 0 {
			db := result.DBInstances[0]
			metrics["status"] = *db.DBInstanceStatus
			metrics["engine"] = *db.Engine
			metrics["instance_class"] = *db.DBInstanceClass
			metrics["allocated_storage_gb"] = *db.AllocatedStorage
		}
	}

	metrics["timestamp"] = time.Now().Unix()
	return metrics, nil
}

// GetResourceHealth returns the health status of a resource
func (a *AWSProvider) GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "ec2_instance":
		return a.getEC2InstanceHealth(ctx, id)
	case "rds_instance":
		return a.getRDSInstanceHealth(ctx, id)
	case "eks_cluster":
		return a.getEKSClusterHealth(ctx, id)
	default:
		return &HealthStatus{
			Status:      HealthStatusUnknown,
			Message:     fmt.Sprintf("Health check not implemented for resource type: %s", resourceType),
			LastUpdated: time.Now(),
		}, nil
	}
}

// Network operations
func (a *AWSProvider) CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating network service", "type", req.Type, "name", req.Name)

	switch req.Type {
	case "vpc":
		return a.createVPCService(ctx, req)
	case "subnet":
		return a.createSubnetService(ctx, req)
	case "security_group":
		return a.createSecurityGroupService(ctx, req)
	case "load_balancer":
		return a.createLoadBalancer(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported network service type: %s", req.Type)
	}
}

func (a *AWSProvider) GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error) {
	parts := splitResourceID(serviceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid serviceID format: %s", serviceID)
	}

	serviceType, id := parts[0], parts[1]

	switch serviceType {
	case "vpc":
		return a.getVPCService(ctx, id)
	case "subnet":
		return a.getSubnetService(ctx, id)
	case "security_group":
		return a.getSecurityGroupService(ctx, id)
	case "load_balancer":
		return a.getLoadBalancer(ctx, id)
	default:
		return nil, fmt.Errorf("unsupported network service type: %s", serviceType)
	}
}

func (a *AWSProvider) DeleteNetworkService(ctx context.Context, serviceID string) error {
	parts := splitResourceID(serviceID)
	if len(parts) < 2 {
		return fmt.Errorf("invalid serviceID format: %s", serviceID)
	}

	serviceType, id := parts[0], parts[1]

	switch serviceType {
	case "vpc":
		return a.deleteVPC(ctx, id)
	case "subnet":
		return a.deleteSubnet(ctx, id)
	case "security_group":
		return a.deleteSecurityGroup(ctx, id)
	case "load_balancer":
		return a.deleteLoadBalancer(ctx, id)
	default:
		return fmt.Errorf("unsupported network service type: %s", serviceType)
	}
}

func (a *AWSProvider) ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	var services []*NetworkServiceResponse

	serviceTypes := filter.Types
	if len(serviceTypes) == 0 {
		serviceTypes = []string{"vpc", "subnet", "security_group", "load_balancer"}
	}

	for _, serviceType := range serviceTypes {
		typeServices, err := a.listNetworkServicesByType(ctx, serviceType, filter)
		if err != nil {
			continue
		}
		services = append(services, typeServices...)
	}

	return services, nil
}

// Storage operations
func (a *AWSProvider) CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating storage resource", "type", req.Type, "name", req.Name)

	switch req.Type {
	case "s3_bucket":
		return a.createS3BucketResource(ctx, req)
	case "ebs_volume":
		return a.createEBSVolumeResource(ctx, req)
	case "efs_filesystem":
		return a.createEFSFilesystem(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported storage resource type: %s", req.Type)
	}
}

func (a *AWSProvider) GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error) {
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "s3_bucket":
		return a.getS3BucketResource(ctx, id)
	case "ebs_volume":
		return a.getEBSVolumeResource(ctx, id)
	case "efs_filesystem":
		return a.getEFSFilesystem(ctx, id)
	default:
		return nil, fmt.Errorf("unsupported storage resource type: %s", resourceType)
	}
}

func (a *AWSProvider) DeleteStorageResource(ctx context.Context, resourceID string) error {
	parts := splitResourceID(resourceID)
	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {
	case "s3_bucket":
		return a.deleteS3Bucket(ctx, id)
	case "ebs_volume":
		return a.deleteEBSVolume(ctx, id)
	case "efs_filesystem":
		return a.deleteEFSFilesystem(ctx, id)
	default:
		return fmt.Errorf("unsupported storage resource type: %s", resourceType)
	}
}

func (a *AWSProvider) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	var resources []*StorageResourceResponse

	resourceTypes := filter.Types
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"s3_bucket", "ebs_volume", "efs_filesystem"}
	}

	for _, resourceType := range resourceTypes {
		typeResources, err := a.listStorageResourcesByType(ctx, resourceType, filter)
		if err != nil {
			continue
		}
		resources = append(resources, typeResources...)
	}

	return resources, nil
}

// Event handling
func (a *AWSProvider) SubscribeToEvents(ctx context.Context, callback EventCallback) error {
	logger := log.FromContext(ctx)
	logger.Info("subscribing to AWS events")

	a.mutex.Lock()
	a.eventCallback = callback
	a.mutex.Unlock()

	// Start watching CloudWatch Events/EventBridge
	go a.watchEvents(ctx)

	return nil
}

func (a *AWSProvider) UnsubscribeFromEvents(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("unsubscribing from AWS events")

	a.mutex.Lock()
	a.eventCallback = nil
	a.mutex.Unlock()

	select {
	case a.stopChannel <- struct{}{}:
	default:
	}

	return nil
}

// Configuration management
func (a *AWSProvider) ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	logger := log.FromContext(ctx)
	logger.Info("applying provider configuration", "name", config.Name)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.config = config
	a.region = config.Region

	// Reconnect if configuration changed
	if a.connected {
		if err := a.Connect(ctx); err != nil {
			return fmt.Errorf("failed to reconnect with new configuration: %w", err)
		}
	}

	return nil
}

func (a *AWSProvider) GetConfiguration(ctx context.Context) (*ProviderConfiguration, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.config, nil
}

func (a *AWSProvider) ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	if config.Type != ProviderTypeAWS {
		return fmt.Errorf("invalid provider type: expected %s, got %s", ProviderTypeAWS, config.Type)
	}

	// Check for required credentials or authentication method
	hasAccessKey := false
	hasProfile := false
	hasRole := false

	if _, exists := config.Credentials["access_key_id"]; exists {
		if _, exists := config.Credentials["secret_access_key"]; !exists {
			return fmt.Errorf("secret_access_key required when access_key_id is provided")
		}
		hasAccessKey = true
	}

	if _, exists := config.Credentials["profile"]; exists {
		hasProfile = true
	}

	if _, exists := config.Credentials["role_arn"]; exists {
		hasRole = true
	}

	// At least one authentication method should be present
	// If none, SDK will use default credential chain (IAM role, env vars, etc.)
	if !hasAccessKey && !hasProfile && !hasRole {
		// This is okay - will use default credential chain
		logger := log.FromContext(ctx)
		logger.Info("No explicit credentials provided, will use AWS default credential chain")
	}

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	return nil
}

// Placeholder implementations for helper methods
// These would be fully implemented in a production environment

func (a *AWSProvider) createEC2Instance(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("EC2 instance creation not yet implemented")
}

func (a *AWSProvider) getEC2Instance(ctx context.Context, id string) (*ResourceResponse, error) {
	return nil, fmt.Errorf("EC2 instance retrieval not yet implemented")
}

func (a *AWSProvider) updateEC2Instance(ctx context.Context, id string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("EC2 instance update not yet implemented")
}

func (a *AWSProvider) deleteEC2Instance(ctx context.Context, id string) error {
	return fmt.Errorf("EC2 instance deletion not yet implemented")
}

func (a *AWSProvider) createS3Bucket(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("S3 bucket creation not yet implemented")
}

func (a *AWSProvider) getS3Bucket(ctx context.Context, id string) (*ResourceResponse, error) {
	return nil, fmt.Errorf("S3 bucket retrieval not yet implemented")
}

func (a *AWSProvider) updateS3Bucket(ctx context.Context, id string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("S3 bucket update not yet implemented")
}

func (a *AWSProvider) deleteS3Bucket(ctx context.Context, id string) error {
	return fmt.Errorf("S3 bucket deletion not yet implemented")
}

func (a *AWSProvider) createVPC(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("VPC creation not yet implemented")
}

func (a *AWSProvider) getVPC(ctx context.Context, id string) (*ResourceResponse, error) {
	return nil, fmt.Errorf("VPC retrieval not yet implemented")
}

func (a *AWSProvider) deleteVPC(ctx context.Context, id string) error {
	return fmt.Errorf("VPC deletion not yet implemented")
}

func (a *AWSProvider) createSecurityGroup(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("security group creation not yet implemented")
}

func (a *AWSProvider) createEBSVolume(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	return nil, fmt.Errorf("EBS volume creation not yet implemented")
}

func (a *AWSProvider) deleteEBSVolume(ctx context.Context, id string) error {
	return fmt.Errorf("EBS volume deletion not yet implemented")
}

func (a *AWSProvider) listResourcesByType(ctx context.Context, resourceType string, filter *ResourceFilter) ([]*ResourceResponse, error) {
	return nil, fmt.Errorf("resource listing not yet implemented for type: %s", resourceType)
}

func (a *AWSProvider) deployCloudFormationStack(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("CloudFormation stack deployment not yet implemented")
}

func (a *AWSProvider) deployCDKStack(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("CDK stack deployment not yet implemented")
}

func (a *AWSProvider) getCloudFormationStack(ctx context.Context, stackName string) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("CloudFormation stack retrieval not yet implemented")
}

func (a *AWSProvider) updateCloudFormationStack(ctx context.Context, stackName string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("CloudFormation stack update not yet implemented")
}

func (a *AWSProvider) deleteCloudFormationStack(ctx context.Context, stackName string) error {
	return fmt.Errorf("CloudFormation stack deletion not yet implemented")
}

func (a *AWSProvider) listCloudFormationStacks(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {
	return nil, fmt.Errorf("CloudFormation stack listing not yet implemented")
}

func (a *AWSProvider) scaleAutoScalingGroup(ctx context.Context, id string, req *ScaleRequest) error {
	return fmt.Errorf("auto scaling group scaling not yet implemented")
}

func (a *AWSProvider) scaleECSService(ctx context.Context, id string, req *ScaleRequest) error {
	return fmt.Errorf("ECS service scaling not yet implemented")
}

func (a *AWSProvider) scaleEKSNodeGroup(ctx context.Context, id string, req *ScaleRequest) error {
	return fmt.Errorf("EKS node group scaling not yet implemented")
}

func (a *AWSProvider) getEC2InstanceHealth(ctx context.Context, id string) (*HealthStatus, error) {
	return nil, fmt.Errorf("EC2 instance health check not yet implemented")
}

func (a *AWSProvider) getRDSInstanceHealth(ctx context.Context, id string) (*HealthStatus, error) {
	return nil, fmt.Errorf("RDS instance health check not yet implemented")
}

func (a *AWSProvider) getEKSClusterHealth(ctx context.Context, id string) (*HealthStatus, error) {
	return nil, fmt.Errorf("EKS cluster health check not yet implemented")
}

func (a *AWSProvider) createVPCService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("VPC service creation not yet implemented")
}

func (a *AWSProvider) getVPCService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("VPC service retrieval not yet implemented")
}

func (a *AWSProvider) createSubnetService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("subnet service creation not yet implemented")
}

func (a *AWSProvider) getSubnetService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("subnet service retrieval not yet implemented")
}

func (a *AWSProvider) deleteSubnet(ctx context.Context, id string) error {
	return fmt.Errorf("subnet deletion not yet implemented")
}

func (a *AWSProvider) createSecurityGroupService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("security group service creation not yet implemented")
}

func (a *AWSProvider) getSecurityGroupService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("security group service retrieval not yet implemented")
}

func (a *AWSProvider) deleteSecurityGroup(ctx context.Context, id string) error {
	return fmt.Errorf("security group deletion not yet implemented")
}

func (a *AWSProvider) createLoadBalancer(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("load balancer creation not yet implemented")
}

func (a *AWSProvider) getLoadBalancer(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("load balancer retrieval not yet implemented")
}

func (a *AWSProvider) deleteLoadBalancer(ctx context.Context, id string) error {
	return fmt.Errorf("load balancer deletion not yet implemented")
}

func (a *AWSProvider) listNetworkServicesByType(ctx context.Context, serviceType string, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("network service listing not yet implemented for type: %s", serviceType)
}

func (a *AWSProvider) createS3BucketResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("S3 bucket resource creation not yet implemented")
}

func (a *AWSProvider) getS3BucketResource(ctx context.Context, id string) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("S3 bucket resource retrieval not yet implemented")
}

func (a *AWSProvider) createEBSVolumeResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("EBS volume resource creation not yet implemented")
}

func (a *AWSProvider) getEBSVolumeResource(ctx context.Context, id string) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("EBS volume resource retrieval not yet implemented")
}

func (a *AWSProvider) createEFSFilesystem(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("EFS filesystem creation not yet implemented")
}

func (a *AWSProvider) getEFSFilesystem(ctx context.Context, id string) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("EFS filesystem retrieval not yet implemented")
}

func (a *AWSProvider) deleteEFSFilesystem(ctx context.Context, id string) error {
	return fmt.Errorf("EFS filesystem deletion not yet implemented")
}

func (a *AWSProvider) listStorageResourcesByType(ctx context.Context, resourceType string, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	return nil, fmt.Errorf("storage resource listing not yet implemented for type: %s", resourceType)
}

func (a *AWSProvider) watchEvents(ctx context.Context) {
	// Implementation would use CloudWatch Events/EventBridge to watch for events
}
