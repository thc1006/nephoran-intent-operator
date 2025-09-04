package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderTypeOpenStack is defined in interface.go.

// OpenStackProvider implements CloudProvider for OpenStack clouds.

type OpenStackProvider struct {
	name string

	config *ProviderConfiguration

	authClient *gophercloud.ProviderClient

	computeClient *gophercloud.ServiceClient

	networkClient *gophercloud.ServiceClient

	storageClient *gophercloud.ServiceClient

	heatClient *gophercloud.ServiceClient

	identityClient *gophercloud.ServiceClient

	connected bool

	eventCallback EventCallback

	stopChannel chan struct{}

	mutex sync.RWMutex

	// Cache for frequently accessed data.

	flavorCache map[string]*flavors.Flavor

	imageCache map[string]*images.Image

	networkCache map[string]*networks.Network

	cacheMutex sync.RWMutex

	cacheExpiry time.Time
}

// NewOpenStackProvider creates a new OpenStack provider instance.

func NewOpenStackProvider(config *ProviderConfiguration) (CloudProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required for OpenStack provider")
	}

	if config.Type != ProviderTypeOpenStack {
		return nil, fmt.Errorf("invalid provider type: expected %s, got %s", ProviderTypeOpenStack, config.Type)
	}

	provider := &OpenStackProvider{
		name: config.Name,

		config: config,

		stopChannel: make(chan struct{}),

		flavorCache: make(map[string]*flavors.Flavor),

		imageCache: make(map[string]*images.Image),

		networkCache: make(map[string]*networks.Network),
	}

	return provider, nil
}

// GetProviderInfo returns information about this OpenStack provider.

func (o *OpenStackProvider) GetProviderInfo() *ProviderInfo {
	o.mutex.RLock()

	defer o.mutex.RUnlock()

	return &ProviderInfo{
		Name: o.name,

		Type: ProviderTypeOpenStack,

		Version: "1.0.0",

		Description: "OpenStack cloud provider for telecommunications workloads",

		Vendor: "OpenStack Foundation",

		Region: o.config.Region,

		Endpoint: o.config.Endpoint,

		Tags: map[string]string{
			"auth_url": o.config.Endpoint,

			"region": o.config.Region,

			"domain": o.config.Credentials["domain"],
		},

		LastUpdated: time.Now(),
	}
}

// GetSupportedResourceTypes returns the resource types supported by OpenStack.

func (o *OpenStackProvider) GetSupportedResourceTypes() []string {
	return []string{
		"server", // Compute instances

		"flavor", // Instance types

		"image", // VM images

		"volume", // Block storage

		"network", // Virtual networks

		"subnet", // Network subnets

		"port", // Network ports

		"router", // Network routers

		"floatingip", // Floating IPs

		"securitygroup", // Security groups

		"keypair", // SSH key pairs

		"stack", // Heat stacks

		"loadbalancer", // Load balancers

		"project", // Tenants/Projects

		"quota", // Resource quotas

	}
}

// GetCapabilities returns the capabilities of this OpenStack provider.

func (o *OpenStackProvider) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{
		ComputeTypes: []string{"server", "flavor", "image", "keypair"},

		StorageTypes: []string{"volume", "snapshot", "backup"},

		NetworkTypes: []string{"network", "subnet", "port", "router", "floatingip", "securitygroup"},

		AcceleratorTypes: []string{"gpu", "fpga", "sriov"},

		AutoScaling: true, // Via Heat

		LoadBalancing: true, // Via Octavia

		Monitoring: true, // Via Ceilometer/Gnocchi

		Logging: true, // Via Monasca

		Networking: true, // Neutron

		StorageClasses: true, // Volume types

		HorizontalPodAutoscaling: false, // Not applicable

		VerticalPodAutoscaling: false, // Not applicable

		ClusterAutoscaling: true, // Via Heat/Senlin

		Namespaces: false, // Projects instead

		ResourceQuotas: true, // Nova/Neutron quotas

		NetworkPolicies: true, // Security groups

		RBAC: true, // Keystone RBAC

		MultiZone: true, // Availability zones

		MultiRegion: true, // Multiple regions

		BackupRestore: true, // Volume backups

		DisasterRecovery: true, // Via replication

		Encryption: true, // Volume encryption

		SecretManagement: true, // Barbican

		ImageScanning: false, // Not built-in

		PolicyEngine: true, // Congress

		MaxNodes: 10000, // Typical large deployment

		MaxPods: 0, // Not applicable

		MaxServices: 10000, // Virtual machines

		MaxVolumes: 50000, // Block volumes

	}
}

// Connect establishes connection to the OpenStack cloud.

func (o *OpenStackProvider) Connect(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("connecting to OpenStack cloud", "auth_url", o.config.Endpoint)

	// Create authentication options.

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: o.config.Endpoint,

		Username: o.config.Credentials["username"],

		Password: o.config.Credentials["password"],

		DomainName: o.config.Credentials["domain"],

		TenantName: o.config.Credentials["project"],
	}

	// Alternative: Use application credentials if provided.

	if appCredID := o.config.Credentials["app_credential_id"]; appCredID != "" {
		authOpts = gophercloud.AuthOptions{
			IdentityEndpoint: o.config.Endpoint,

			ApplicationCredentialID: appCredID,

			ApplicationCredentialSecret: o.config.Credentials["app_credential_secret"],
		}
	}

	// Create the provider client.

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}

	o.authClient = provider

	// Initialize service clients.

	if err := o.initializeServiceClients(ctx); err != nil {
		return fmt.Errorf("failed to initialize service clients: %w", err)
	}

	o.mutex.Lock()

	o.connected = true

	o.mutex.Unlock()

	logger.Info("successfully connected to OpenStack cloud")

	return nil
}

// initializeServiceClients initializes all OpenStack service clients.

func (o *OpenStackProvider) initializeServiceClients(ctx context.Context) error {
	endpointOpts := gophercloud.EndpointOpts{
		Region: o.config.Region,
	}

	// Initialize compute (Nova) client.

	computeClient, err := openstack.NewComputeV2(o.authClient, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	o.computeClient = computeClient

	// Initialize network (Neutron) client.

	networkClient, err := openstack.NewNetworkV2(o.authClient, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create network client: %w", err)
	}

	o.networkClient = networkClient

	// Initialize block storage (Cinder) client.

	storageClient, err := openstack.NewBlockStorageV3(o.authClient, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	o.storageClient = storageClient

	// Initialize orchestration (Heat) client.

	heatClient, err := openstack.NewOrchestrationV1(o.authClient, endpointOpts)

	if err != nil {

		// Heat is optional, log warning but don't fail.

		logger := log.FromContext(ctx)

		logger.Info("Heat orchestration service not available")

	} else {
		o.heatClient = heatClient
	}

	// Initialize identity (Keystone) client.

	identityClient, err := openstack.NewIdentityV3(o.authClient, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create identity client: %w", err)
	}

	o.identityClient = identityClient

	return nil
}

// Disconnect closes the connection to the OpenStack cloud.

func (o *OpenStackProvider) Disconnect(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("disconnecting from OpenStack cloud")

	o.mutex.Lock()

	o.connected = false

	o.mutex.Unlock()

	// Stop event watching if running.

	select {

	case o.stopChannel <- struct{}{}:

	default:

	}

	// Clear caches.

	o.cacheMutex.Lock()

	o.flavorCache = make(map[string]*flavors.Flavor)

	o.imageCache = make(map[string]*images.Image)

	o.networkCache = make(map[string]*networks.Network)

	o.cacheMutex.Unlock()

	logger.Info("disconnected from OpenStack cloud")

	return nil
}

// HealthCheck performs a health check on the OpenStack cloud.

func (o *OpenStackProvider) HealthCheck(ctx context.Context) error {
	// Check if we can list hypervisors (requires admin or appropriate role).

	if o.computeClient != nil {

		allPages, err := hypervisors.List(o.computeClient, nil).AllPages()

		if err != nil {

			// Try listing flavors as a fallback (less privileged operation).

			_, err = flavors.ListDetail(o.computeClient, flavors.ListOpts{Limit: 1}).AllPages()
			if err != nil {
				return fmt.Errorf("health check failed: unable to access compute service: %w", err)
			}

		} else {

			// Check hypervisor status.

			hypervisorList, err := hypervisors.ExtractHypervisors(allPages)

			if err == nil && len(hypervisorList) > 0 {

				activeHypervisors := 0

				for _, h := range hypervisorList {
					if h.State == "up" {
						activeHypervisors++
					}
				}

				if activeHypervisors == 0 {
					return fmt.Errorf("health check warning: no active hypervisors found")
				}

			}

		}

	}

	// Check network service.

	if o.networkClient != nil {

		_, err := networks.List(o.networkClient, networks.ListOpts{Limit: 1}).AllPages()
		if err != nil {
			return fmt.Errorf("health check failed: unable to access network service: %w", err)
		}

	}

	// Check storage service.

	if o.storageClient != nil {

		_, err := volumes.List(o.storageClient, volumes.ListOpts{Limit: 1}).AllPages()
		if err != nil {
			return fmt.Errorf("health check failed: unable to access storage service: %w", err)
		}

	}

	return nil
}

// Close closes any resources held by the provider.

func (o *OpenStackProvider) Close() error {
	o.mutex.Lock()

	defer o.mutex.Unlock()

	// Stop event watching.

	select {

	case o.stopChannel <- struct{}{}:

	default:

	}

	o.connected = false

	return nil
}

// CreateResource creates a new OpenStack resource.

func (o *OpenStackProvider) CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)

	logger.Info("creating OpenStack resource", "type", req.Type, "name", req.Name)

	switch req.Type {

	case "server":

		return o.createServer(ctx, req)

	case "volume":

		return o.createVolume(ctx, req)

	case "network":

		return o.createNetwork(ctx, req)

	case "subnet":

		return o.createSubnet(ctx, req)

	case "floatingip":

		return o.createFloatingIP(ctx, req)

	case "securitygroup":

		return o.createSecurityGroup(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported resource type: %s", req.Type)

	}
}

// GetResource retrieves an OpenStack resource.

func (o *OpenStackProvider) GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("getting OpenStack resource", "resourceID", resourceID)

	// Parse resourceID format: type/id.

	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "server":

		return o.getServer(ctx, id)

	case "volume":

		return o.getVolume(ctx, id)

	case "network":

		return o.getNetwork(ctx, id)

	case "subnet":

		return o.getSubnet(ctx, id)

	case "floatingip":

		return o.getFloatingIP(ctx, id)

	default:

		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)

	}
}

// UpdateResource updates an OpenStack resource.

func (o *OpenStackProvider) UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	logger := log.FromContext(ctx)

	logger.Info("updating OpenStack resource", "resourceID", resourceID)

	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "server":

		return o.updateServer(ctx, id, req)

	case "volume":

		return o.updateVolume(ctx, id, req)

	case "network":

		return o.updateNetwork(ctx, id, req)

	default:

		return nil, fmt.Errorf("resource type %s does not support updates", resourceType)

	}
}

// DeleteResource deletes an OpenStack resource.

func (o *OpenStackProvider) DeleteResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)

	logger.Info("deleting OpenStack resource", "resourceID", resourceID)

	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "server":

		return servers.Delete(o.computeClient, id).ExtractErr()

	case "volume":

		return volumes.Delete(o.storageClient, id, volumes.DeleteOpts{}).ExtractErr()

	case "network":

		return networks.Delete(o.networkClient, id).ExtractErr()

	case "subnet":

		return subnets.Delete(o.networkClient, id).ExtractErr()

	case "floatingip":

		// Floating IP deletion would be implemented here.

		return fmt.Errorf("floating IP deletion not yet implemented")

	default:

		return fmt.Errorf("unsupported resource type for deletion: %s", resourceType)

	}
}

// ListResources lists OpenStack resources with optional filtering.

func (o *OpenStackProvider) ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("listing OpenStack resources", "filter", filter)

	var resources []*ResourceResponse

	resourceTypes := filter.Types

	if len(resourceTypes) == 0 {
		// Default to common resource types.

		resourceTypes = []string{"server", "volume", "network"}
	}

	for _, resourceType := range resourceTypes {

		typeResources, err := o.listResourcesByType(ctx, resourceType, filter)
		if err != nil {

			logger.Error(err, "failed to list resources", "type", resourceType)

			continue

		}

		resources = append(resources, typeResources...)

	}

	return resources, nil
}

// Deploy creates a deployment using Heat templates.

func (o *OpenStackProvider) Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	logger := log.FromContext(ctx)

	logger.Info("deploying Heat stack", "name", req.Name)

	if o.heatClient == nil {
		return nil, fmt.Errorf("Heat orchestration service not available")
	}

	// Create Heat stack from template.

	templateContent := req.Template

	if templateContent == "" {
		// Create a minimal Heat template if none provided.

		templateContent = `{

			"heat_template_version": "2018-08-31",

			"description": "Generated template from deployment request",

			"resources": {}

		}`
	}

	// Convert parameters from JSON to map
	var parameters map[string]interface{}
	if req.Parameters != nil {
		if err := json.Unmarshal(req.Parameters, &parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	createOpts := stacks.CreateOpts{
		Name: req.Name,

		TemplateOpts: &stacks.Template{
			TE: stacks.TE{
				Bin: []byte(templateContent),
			},
		},

		Parameters: parameters,

		Tags: convertLabelsToTags(req.Labels),

		Timeout: int(req.Timeout.Minutes()),
	}

	result := stacks.Create(o.heatClient, createOpts)

	stack, err := result.Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create Heat stack: %w", err)
	}

	return o.convertStackToDeploymentResponse(stack), nil
}

// GetDeployment retrieves a Heat stack deployment.

func (o *OpenStackProvider) GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error) {
	if o.heatClient == nil {
		return nil, fmt.Errorf("Heat orchestration service not available")
	}

	// Parse stack name and ID.

	stackName, stackID := parseStackIdentifier(deploymentID)

	result := stacks.Get(o.heatClient, stackName, stackID)

	stack, err := result.Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get Heat stack: %w", err)
	}

	return o.convertRetrievedStackToDeploymentResponse(stack), nil
}

// UpdateDeployment updates a Heat stack deployment.

func (o *OpenStackProvider) UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {
	if o.heatClient == nil {
		return nil, fmt.Errorf("Heat orchestration service not available")
	}

	stackName, stackID := parseStackIdentifier(deploymentID)

	// Update template content.

	templateContent := req.Template

	if templateContent == "" {
		// Create a minimal Heat template if none provided.

		templateContent = `{

			"heat_template_version": "2018-08-31",

			"description": "Updated template from deployment request",

			"resources": {}

		}`
	}

	// Convert parameters from JSON to map
	var parameters map[string]interface{}
	if req.Parameters != nil {
		if err := json.Unmarshal(req.Parameters, &parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	updateOpts := stacks.UpdateOpts{
		TemplateOpts: &stacks.Template{
			TE: stacks.TE{
				Bin: []byte(templateContent),
			},
		},

		Parameters: parameters,

		Tags: convertLabelsToTags(req.Labels),

		Timeout: int(req.Timeout.Minutes()),
	}

	err := stacks.Update(o.heatClient, stackName, stackID, updateOpts).ExtractErr()
	if err != nil {
		return nil, fmt.Errorf("failed to update Heat stack: %w", err)
	}

	return o.GetDeployment(ctx, deploymentID)
}

// DeleteDeployment deletes a Heat stack deployment.

func (o *OpenStackProvider) DeleteDeployment(ctx context.Context, deploymentID string) error {
	if o.heatClient == nil {
		return fmt.Errorf("Heat orchestration service not available")
	}

	stackName, stackID := parseStackIdentifier(deploymentID)

	return stacks.Delete(o.heatClient, stackName, stackID).ExtractErr()
}

// ListDeployments lists Heat stack deployments.

func (o *OpenStackProvider) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {
	if o.heatClient == nil {
		return nil, fmt.Errorf("Heat orchestration service not available")
	}

	listOpts := stacks.ListOpts{
		Limit: filter.Limit,
	}

	if len(filter.Names) > 0 {
		listOpts.Name = filter.Names[0] // Heat only supports single name filter
	}

	if len(filter.Statuses) > 0 {
		listOpts.Status = filter.Statuses[0]
	}

	allPages, err := stacks.List(o.heatClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list Heat stacks: %w", err)
	}

	stackList, err := stacks.ExtractStacks(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract stacks: %w", err)
	}

	var deployments []*DeploymentResponse

	for _, stack := range stackList {
		deployments = append(deployments, o.convertListedStackToDeploymentResponse(&stack))
	}

	return deployments, nil
}

// ScaleResource scales an OpenStack resource (servers).

func (o *OpenStackProvider) ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error {
	logger := log.FromContext(ctx)

	logger.Info("scaling OpenStack resource", "resourceID", resourceID, "direction", req.Direction)

	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	if resourceType != "server" {
		return fmt.Errorf("scaling not supported for resource type: %s", resourceType)
	}

	// For servers, scaling typically means resizing.

	if req.Type == ScaleTypeVertical {

		// Resize server to different flavor.

		flavorID := ""

		if req.Target != nil {
			var targetMap map[string]interface{}
			if err := json.Unmarshal(req.Target, &targetMap); err == nil {
				if fid, ok := targetMap["flavor_id"].(string); ok {
					flavorID = fid
				}
			}
		}

		if flavorID == "" {
			return fmt.Errorf("flavor_id required for vertical scaling")
		}

		resizeOpts := servers.ResizeOpts{
			FlavorRef: flavorID,
		}

		return servers.Resize(o.computeClient, id, resizeOpts).ExtractErr()

	}

	return fmt.Errorf("horizontal scaling requires Heat autoscaling groups")
}

// GetScalingCapabilities returns scaling capabilities for a resource.

func (o *OpenStackProvider) GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error) {
	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType := parts[0]

	switch resourceType {

	case "server":

		return &ScalingCapabilities{
			HorizontalScaling: false, // Requires Heat

			VerticalScaling: true, // Resize operation

			MinReplicas: 1,

			MaxReplicas: 1,

			SupportedMetrics: []string{"cpu", "memory", "disk"},

			ScaleUpCooldown: 60 * time.Second,

			ScaleDownCooldown: 300 * time.Second,
		}, nil

	case "stack":

		// Heat stacks can support autoscaling.

		return &ScalingCapabilities{
			HorizontalScaling: true,

			VerticalScaling: false,

			MinReplicas: 1,

			MaxReplicas: 100,

			SupportedMetrics: []string{"cpu", "memory", "network"},

			ScaleUpCooldown: 60 * time.Second,

			ScaleDownCooldown: 300 * time.Second,
		}, nil

	default:

		return &ScalingCapabilities{
			HorizontalScaling: false,

			VerticalScaling: false,
		}, nil

	}
}

// GetMetrics returns cloud-level metrics.

func (o *OpenStackProvider) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get hypervisor statistics if available.

	if o.computeClient != nil {

		allPages, err := hypervisors.List(o.computeClient, nil).AllPages()

		if err == nil {

			hypervisorList, err := hypervisors.ExtractHypervisors(allPages)

			if err == nil {

				totalVCPUs := 0

				usedVCPUs := 0

				totalMemoryMB := 0

				usedMemoryMB := 0

				totalDiskGB := 0

				usedDiskGB := 0

				runningVMs := 0

				for _, h := range hypervisorList {

					totalVCPUs += h.VCPUs

					usedVCPUs += h.VCPUsUsed

					totalMemoryMB += h.MemoryMB

					usedMemoryMB += h.MemoryMBUsed

					totalDiskGB += h.LocalGB

					usedDiskGB += h.LocalGBUsed

					runningVMs += h.RunningVMs

				}

				metrics["hypervisors_total"] = len(hypervisorList)

				metrics["vcpus_total"] = totalVCPUs

				metrics["vcpus_used"] = usedVCPUs

				metrics["memory_mb_total"] = totalMemoryMB

				metrics["memory_mb_used"] = usedMemoryMB

				metrics["disk_gb_total"] = totalDiskGB

				metrics["disk_gb_used"] = usedDiskGB

				metrics["vms_running"] = runningVMs

			}

		}

		// Get server count.

		serverPages, err := servers.List(o.computeClient, servers.ListOpts{}).AllPages()

		if err == nil {

			serverList, err := servers.ExtractServers(serverPages)

			if err == nil {

				activeServers := 0

				for _, s := range serverList {
					if s.Status == "ACTIVE" {
						activeServers++
					}
				}

				metrics["servers_total"] = len(serverList)

				metrics["servers_active"] = activeServers

			}

		}

	}

	// Get network statistics.

	if o.networkClient != nil {

		networkPages, err := networks.List(o.networkClient, networks.ListOpts{}).AllPages()

		if err == nil {

			networkList, err := networks.ExtractNetworks(networkPages)

			if err == nil {
				metrics["networks_total"] = len(networkList)
			}

		}

		subnetPages, err := subnets.List(o.networkClient, subnets.ListOpts{}).AllPages()

		if err == nil {

			subnetList, err := subnets.ExtractSubnets(subnetPages)

			if err == nil {
				metrics["subnets_total"] = len(subnetList)
			}

		}

	}

	// Get storage statistics.

	if o.storageClient != nil {

		volumePages, err := volumes.List(o.storageClient, volumes.ListOpts{}).AllPages()

		if err == nil {

			volumeList, err := volumes.ExtractVolumes(volumePages)

			if err == nil {

				totalVolumeGB := 0

				availableVolumes := 0

				for _, v := range volumeList {

					totalVolumeGB += v.Size

					if v.Status == "available" {
						availableVolumes++
					}

				}

				metrics["volumes_total"] = len(volumeList)

				metrics["volumes_available"] = availableVolumes

				metrics["volume_gb_total"] = totalVolumeGB

			}

		}

	}

	metrics["timestamp"] = time.Now().Unix()

	return metrics, nil
}

// GetResourceMetrics returns metrics for a specific resource.

func (o *OpenStackProvider) GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	metrics := make(map[string]interface{})

	switch resourceType {

	case "server":

		server, err := servers.Get(o.computeClient, id).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get server: %w", err)
		}

		metrics["status"] = server.Status

		metrics["created"] = server.Created

		metrics["updated"] = server.Updated

		// Note: PowerState, VmState, TaskState fields may not be available in all gophercloud versions.

		// These would require server details with extended attributes.

		// Get flavor details for resource allocation.

		if server.Flavor["id"] != nil {

			flavorID := server.Flavor["id"].(string)

			flavor, err := flavors.Get(o.computeClient, flavorID).Extract()

			if err == nil {

				metrics["vcpus"] = flavor.VCPUs

				metrics["ram_mb"] = flavor.RAM

				metrics["disk_gb"] = flavor.Disk

			}

		}

	case "volume":

		volume, err := volumes.Get(o.storageClient, id).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get volume: %w", err)
		}

		metrics["status"] = volume.Status

		metrics["size_gb"] = volume.Size

		metrics["bootable"] = volume.Bootable

		metrics["encrypted"] = volume.Encrypted

		metrics["created_at"] = volume.CreatedAt

		metrics["updated_at"] = volume.UpdatedAt

	case "network":

		network, err := networks.Get(o.networkClient, id).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get network: %w", err)
		}

		metrics["status"] = network.Status

		metrics["admin_state_up"] = network.AdminStateUp

		metrics["shared"] = network.Shared

		metrics["created_at"] = network.CreatedAt

		metrics["updated_at"] = network.UpdatedAt

		// Note: External and PortSecurityEnabled may be in different fields or extensions.

		if len(network.Tags) > 0 {
			metrics["tags"] = network.Tags
		}

	}

	metrics["timestamp"] = time.Now().Unix()

	return metrics, nil
}

// GetResourceHealth returns the health status of a resource.

func (o *OpenStackProvider) GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {
	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "server":

		server, err := servers.Get(o.computeClient, id).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get server: %w", err)
		}

		health := &HealthStatus{
			LastUpdated: time.Now(),
		}

		switch server.Status {

		case "ACTIVE":

			health.Status = HealthStatusHealthy

			health.Message = "Server is active and running"

		case "ERROR":

			health.Status = HealthStatusUnhealthy

			health.Message = fmt.Sprintf("Server is in error state: %s", server.Fault.Message)

		case "BUILD", "REBUILD", "RESIZE", "VERIFY_RESIZE":

			health.Status = HealthStatusUnknown

			health.Message = fmt.Sprintf("Server is in transitional state: %s", server.Status)

		default:

			health.Status = HealthStatusUnknown

			health.Message = fmt.Sprintf("Server status: %s", server.Status)

		}

		return health, nil

	case "volume":

		volume, err := volumes.Get(o.storageClient, id).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get volume: %w", err)
		}

		health := &HealthStatus{
			LastUpdated: time.Now(),
		}

		switch volume.Status {

		case "available", "in-use":

			health.Status = HealthStatusHealthy

			health.Message = fmt.Sprintf("Volume is %s", volume.Status)

		case "error", "error_deleting", "error_backing-up", "error_restoring", "error_extending":

			health.Status = HealthStatusUnhealthy

			health.Message = fmt.Sprintf("Volume is in error state: %s", volume.Status)

		default:

			health.Status = HealthStatusUnknown

			health.Message = fmt.Sprintf("Volume status: %s", volume.Status)

		}

		return health, nil

	default:

		return &HealthStatus{
			Status: HealthStatusUnknown,

			Message: fmt.Sprintf("Health check not implemented for resource type: %s", resourceType),

			LastUpdated: time.Now(),
		}, nil

	}
}

// Network operations.

func (o *OpenStackProvider) CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	logger := log.FromContext(ctx)

	logger.Info("creating network service", "type", req.Type, "name", req.Name)

	switch req.Type {

	case "network":

		return o.createNetworkService(ctx, req)

	case "subnet":

		return o.createSubnetService(ctx, req)

	case "router":

		return o.createRouterService(ctx, req)

	case "floatingip":

		return o.createFloatingIPService(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported network service type: %s", req.Type)

	}
}

// GetNetworkService performs getnetworkservice operation.

func (o *OpenStackProvider) GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error) {
	parts := splitResourceID(serviceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid serviceID format: %s", serviceID)
	}

	serviceType, id := parts[0], parts[1]

	switch serviceType {

	case "network":

		return o.getNetworkService(ctx, id)

	case "subnet":

		return o.getSubnetService(ctx, id)

	case "router":

		return o.getRouterService(ctx, id)

	default:

		return nil, fmt.Errorf("unsupported network service type: %s", serviceType)

	}
}

// DeleteNetworkService performs deletenetworkservice operation.

func (o *OpenStackProvider) DeleteNetworkService(ctx context.Context, serviceID string) error {
	parts := splitResourceID(serviceID)

	if len(parts) < 2 {
		return fmt.Errorf("invalid serviceID format: %s", serviceID)
	}

	serviceType, id := parts[0], parts[1]

	switch serviceType {

	case "network":

		return networks.Delete(o.networkClient, id).ExtractErr()

	case "subnet":

		return subnets.Delete(o.networkClient, id).ExtractErr()

	case "router":

		// Router deletion would be implemented here.

		return fmt.Errorf("router deletion not yet implemented")

	default:

		return fmt.Errorf("unsupported network service type: %s", serviceType)

	}
}

// ListNetworkServices performs listnetworkservices operation.

func (o *OpenStackProvider) ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	var services []*NetworkServiceResponse

	serviceTypes := filter.Types

	if len(serviceTypes) == 0 {
		serviceTypes = []string{"network", "subnet", "router"}
	}

	for _, serviceType := range serviceTypes {

		typeServices, err := o.listNetworkServicesByType(ctx, serviceType, filter)
		if err != nil {
			continue
		}

		services = append(services, typeServices...)

	}

	return services, nil
}

// Storage operations.

func (o *OpenStackProvider) CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	logger := log.FromContext(ctx)

	logger.Info("creating storage resource", "type", req.Type, "name", req.Name)

	switch req.Type {

	case "volume":

		return o.createVolumeResource(ctx, req)

	case "snapshot":

		return o.createSnapshotResource(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported storage resource type: %s", req.Type)

	}
}

// GetStorageResource performs getstorageresource operation.

func (o *OpenStackProvider) GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error) {
	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "volume":

		return o.getVolumeResource(ctx, id)

	case "snapshot":

		return o.getSnapshotResource(ctx, id)

	default:

		return nil, fmt.Errorf("unsupported storage resource type: %s", resourceType)

	}
}

// DeleteStorageResource performs deletestorageresource operation.

func (o *OpenStackProvider) DeleteStorageResource(ctx context.Context, resourceID string) error {
	parts := splitResourceID(resourceID)

	if len(parts) < 2 {
		return fmt.Errorf("invalid resourceID format: %s", resourceID)
	}

	resourceType, id := parts[0], parts[1]

	switch resourceType {

	case "volume":

		return volumes.Delete(o.storageClient, id, volumes.DeleteOpts{}).ExtractErr()

	case "snapshot":

		// Snapshot deletion would be implemented here.

		return fmt.Errorf("snapshot deletion not yet implemented")

	default:

		return fmt.Errorf("unsupported storage resource type: %s", resourceType)

	}
}

// ListStorageResources performs liststorageresources operation.

func (o *OpenStackProvider) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	var resources []*StorageResourceResponse

	resourceTypes := filter.Types

	if len(resourceTypes) == 0 {
		resourceTypes = []string{"volume", "snapshot"}
	}

	for _, resourceType := range resourceTypes {

		typeResources, err := o.listStorageResourcesByType(ctx, resourceType, filter)
		if err != nil {
			continue
		}

		resources = append(resources, typeResources...)

	}

	return resources, nil
}

// Event handling.

func (o *OpenStackProvider) SubscribeToEvents(ctx context.Context, callback EventCallback) error {
	logger := log.FromContext(ctx)

	logger.Info("subscribing to OpenStack events")

	o.mutex.Lock()

	o.eventCallback = callback

	o.mutex.Unlock()

	// OpenStack events would typically come from message queue (e.g., RabbitMQ).

	// or by polling the API for changes.

	// This is a placeholder for the actual implementation.

	go o.watchEvents(ctx)

	return nil
}

// UnsubscribeFromEvents performs unsubscribefromevents operation.

func (o *OpenStackProvider) UnsubscribeFromEvents(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("unsubscribing from OpenStack events")

	o.mutex.Lock()

	o.eventCallback = nil

	o.mutex.Unlock()

	select {

	case o.stopChannel <- struct{}{}:

	default:

	}

	return nil
}

// Configuration management.

func (o *OpenStackProvider) ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	logger := log.FromContext(ctx)

	logger.Info("applying provider configuration", "name", config.Name)

	o.mutex.Lock()

	defer o.mutex.Unlock()

	o.config = config

	// Re-authenticate if credentials changed.

	if o.connected {
		if err := o.Connect(ctx); err != nil {
			return fmt.Errorf("failed to reconnect with new configuration: %w", err)
		}
	}

	return nil
}

// GetConfiguration performs getconfiguration operation.

func (o *OpenStackProvider) GetConfiguration(ctx context.Context) (*ProviderConfiguration, error) {
	o.mutex.RLock()

	defer o.mutex.RUnlock()

	return o.config, nil
}

// ValidateConfiguration performs validateconfiguration operation.

func (o *OpenStackProvider) ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	if config.Type != ProviderTypeOpenStack {
		return fmt.Errorf("invalid provider type: expected %s, got %s", ProviderTypeOpenStack, config.Type)
	}

	// Check required credentials.

	requiredCreds := []string{"username", "password", "project", "domain"}

	for _, key := range requiredCreds {
		if _, exists := config.Credentials[key]; !exists {
			// Check for application credentials as alternative.

			if _, hasAppCred := config.Credentials["app_credential_id"]; !hasAppCred {
				return fmt.Errorf("missing required credential: %s", key)
			}
		}
	}

	if config.Endpoint == "" {
		return fmt.Errorf("endpoint (auth_url) is required")
	}

	return nil
}

// Helper functions.

func splitResourceID(resourceID string) []string {
	// Simple split by forward slash.

	// Format: type/id or namespace/type/id.

	return strings.Split(resourceID, "/")
}

func convertLabelsToTags(labels map[string]string) []string {
	var tags []string

	for key, value := range labels {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}

	return tags
}

func parseStackIdentifier(deploymentID string) (string, string) {
	parts := strings.Split(deploymentID, "/")

	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return deploymentID, ""
}

// Placeholder for additional helper methods...

// The actual implementation would include all the create*, get*, update*, list* helper methods.

// referenced above, following similar patterns to the Kubernetes provider.

func (o *OpenStackProvider) createServer(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would create an OpenStack server instance.

	return nil, fmt.Errorf("server creation not yet implemented")
}

func (o *OpenStackProvider) getServer(ctx context.Context, id string) (*ResourceResponse, error) {
	// Implementation would retrieve server details.

	return nil, fmt.Errorf("server retrieval not yet implemented")
}

func (o *OpenStackProvider) updateServer(ctx context.Context, id string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	// Implementation would update server properties.

	return nil, fmt.Errorf("server update not yet implemented")
}

func (o *OpenStackProvider) createVolume(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would create a Cinder volume.

	return nil, fmt.Errorf("volume creation not yet implemented")
}

func (o *OpenStackProvider) getVolume(ctx context.Context, id string) (*ResourceResponse, error) {
	// Implementation would retrieve volume details.

	return nil, fmt.Errorf("volume retrieval not yet implemented")
}

func (o *OpenStackProvider) updateVolume(ctx context.Context, id string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	// Implementation would update volume properties.

	return nil, fmt.Errorf("volume update not yet implemented")
}

func (o *OpenStackProvider) createNetwork(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would create a Neutron network.

	return nil, fmt.Errorf("network creation not yet implemented")
}

func (o *OpenStackProvider) getNetwork(ctx context.Context, id string) (*ResourceResponse, error) {
	// Implementation would retrieve network details.

	return nil, fmt.Errorf("network retrieval not yet implemented")
}

func (o *OpenStackProvider) updateNetwork(ctx context.Context, id string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	// Implementation would update network properties.

	return nil, fmt.Errorf("network update not yet implemented")
}

func (o *OpenStackProvider) createSubnet(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would create a subnet.

	return nil, fmt.Errorf("subnet creation not yet implemented")
}

func (o *OpenStackProvider) getSubnet(ctx context.Context, id string) (*ResourceResponse, error) {
	// Implementation would retrieve subnet details.

	return nil, fmt.Errorf("subnet retrieval not yet implemented")
}

func (o *OpenStackProvider) createFloatingIP(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would allocate a floating IP.

	return nil, fmt.Errorf("floating IP creation not yet implemented")
}

func (o *OpenStackProvider) getFloatingIP(ctx context.Context, id string) (*ResourceResponse, error) {
	// Implementation would retrieve floating IP details.

	return nil, fmt.Errorf("floating IP retrieval not yet implemented")
}

func (o *OpenStackProvider) createSecurityGroup(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Implementation would create a security group.

	return nil, fmt.Errorf("security group creation not yet implemented")
}

func (o *OpenStackProvider) listResourcesByType(ctx context.Context, resourceType string, filter *ResourceFilter) ([]*ResourceResponse, error) {
	// Implementation would list resources by type with filtering.

	return nil, fmt.Errorf("resource listing not yet implemented for type: %s", resourceType)
}

func (o *OpenStackProvider) convertStackToDeploymentResponse(stack *stacks.CreatedStack) *DeploymentResponse {
	// Implementation would convert Heat stack to deployment response.

	return &DeploymentResponse{
		ID: stack.ID,

		Name: stack.ID, // CreatedStack doesn't have Name field, use ID

		Status: "creating",

		Phase: "initializing",

		TemplateType: "heat",

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}
}

func (o *OpenStackProvider) convertRetrievedStackToDeploymentResponse(stack *stacks.RetrievedStack) *DeploymentResponse {
	// Convert RetrievedStack to deployment response.

	return &DeploymentResponse{
		ID: stack.ID,

		Name: stack.Name,

		Status: stack.Status,

		Phase: stack.Status,

		TemplateType: "heat",

		CreatedAt: stack.CreationTime,

		UpdatedAt: stack.UpdatedTime,
	}
}

func (o *OpenStackProvider) convertListedStackToDeploymentResponse(stack *stacks.ListedStack) *DeploymentResponse {
	// Convert ListedStack to deployment response.

	return &DeploymentResponse{
		ID: stack.ID,

		Name: stack.Name,

		Status: stack.Status,

		Phase: stack.Status,

		TemplateType: "heat",

		CreatedAt: stack.CreationTime,

		UpdatedAt: stack.UpdatedTime,
	}
}

func (o *OpenStackProvider) createNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	// Implementation would create network service.

	return nil, fmt.Errorf("network service creation not yet implemented")
}

func (o *OpenStackProvider) getNetworkService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	// Implementation would retrieve network service.

	return nil, fmt.Errorf("network service retrieval not yet implemented")
}

func (o *OpenStackProvider) createSubnetService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	// Implementation would create subnet service.

	return nil, fmt.Errorf("subnet service creation not yet implemented")
}

func (o *OpenStackProvider) getSubnetService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	// Implementation would retrieve subnet service.

	return nil, fmt.Errorf("subnet service retrieval not yet implemented")
}

func (o *OpenStackProvider) createRouterService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	// Implementation would create router service.

	return nil, fmt.Errorf("router service creation not yet implemented")
}

func (o *OpenStackProvider) getRouterService(ctx context.Context, id string) (*NetworkServiceResponse, error) {
	// Implementation would retrieve router service.

	return nil, fmt.Errorf("router service retrieval not yet implemented")
}

func (o *OpenStackProvider) createFloatingIPService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	// Implementation would create floating IP service.

	return nil, fmt.Errorf("floating IP service creation not yet implemented")
}

func (o *OpenStackProvider) listNetworkServicesByType(ctx context.Context, serviceType string, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	// Implementation would list network services by type.

	return nil, fmt.Errorf("network service listing not yet implemented for type: %s", serviceType)
}

func (o *OpenStackProvider) createVolumeResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	// Implementation would create volume resource.

	return nil, fmt.Errorf("volume resource creation not yet implemented")
}

func (o *OpenStackProvider) getVolumeResource(ctx context.Context, id string) (*StorageResourceResponse, error) {
	// Implementation would retrieve volume resource.

	return nil, fmt.Errorf("volume resource retrieval not yet implemented")
}

func (o *OpenStackProvider) createSnapshotResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	// Implementation would create snapshot resource.

	return nil, fmt.Errorf("snapshot resource creation not yet implemented")
}

func (o *OpenStackProvider) getSnapshotResource(ctx context.Context, id string) (*StorageResourceResponse, error) {
	// Implementation would retrieve snapshot resource.

	return nil, fmt.Errorf("snapshot resource retrieval not yet implemented")
}

func (o *OpenStackProvider) listStorageResourcesByType(ctx context.Context, resourceType string, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	// Implementation would list storage resources by type.

	return nil, fmt.Errorf("storage resource listing not yet implemented for type: %s", resourceType)
}

func (o *OpenStackProvider) watchEvents(ctx context.Context) {
	// Implementation would watch for OpenStack events.

	// This could involve polling or connecting to message queue.
}
