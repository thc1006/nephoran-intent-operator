package modeladapter

import (
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/oran/o2/models"
)

// InternalResourceType represents a stable internal representation of a resource type.

// This isolates our business logic from changes in the generated O-RAN O2 IMS models.

type InternalResourceType struct {

	// Core identification.

	ResourceTypeID string `json:"resourceTypeId"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Vendor string `json:"vendor"`

	Model string `json:"model,omitempty"`

	Version string `json:"version"`

	// Internal specification representation.

	Specifications *InternalResourceTypeSpec `json:"specifications,omitempty"`

	// Supported actions (mapped from SupportedOperations).

	SupportedActions []string `json:"supportedActions,omitempty"`

	// Lifecycle information.

	Status string `json:"status"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`

	CreatedBy string `json:"createdBy,omitempty"`

	UpdatedBy string `json:"updatedBy,omitempty"`
}

// InternalResourceTypeSpec represents internal resource type specifications.

// This provides a stable interface for resource type configuration.

type InternalResourceTypeSpec struct {

	// Category from the generated model.

	Category string `json:"category"`

	// Resource classification.

	ResourceClass string `json:"resourceClass,omitempty"`

	ResourceKind string `json:"resourceKind,omitempty"`

	// Resource limits and capabilities (simplified).

	MinResources map[string]string `json:"minResources,omitempty"`

	DefaultResources map[string]string `json:"defaultResources,omitempty"`

	MaxResources map[string]string `json:"maxResources,omitempty"`

	// Properties and features.

	Properties map[string]interface{} `json:"properties,omitempty"`

	Capabilities []string `json:"capabilities,omitempty"`

	Features []string `json:"features,omitempty"`
}

// FromGenerated converts a generated ResourceType to our internal representation.

// This function adapts the current O-RAN model structure to our stable internal format.

func FromGenerated(generated *models.ResourceType) *InternalResourceType {

	if generated == nil {

		return nil

	}

	internal := &InternalResourceType{

		ResourceTypeID: generated.ResourceTypeID,

		Name: generated.Name,

		Description: generated.Description,

		Vendor: generated.Vendor,

		Model: generated.Model,

		Version: generated.Version,

		Status: generated.Status,

		CreatedAt: generated.CreatedAt,

		UpdatedAt: generated.UpdatedAt,

		CreatedBy: generated.CreatedBy,

		UpdatedBy: generated.UpdatedBy,
	}

	// Map SupportedOperations to SupportedActions for backward compatibility.

	internal.SupportedActions = make([]string, len(generated.SupportedOperations))

	copy(internal.SupportedActions, generated.SupportedOperations)

	// Create internal specifications from current model structure.

	internal.Specifications = &InternalResourceTypeSpec{

		Category: generated.Category,

		ResourceClass: generated.ResourceClass,

		ResourceKind: generated.ResourceKind,
	}

	// Extract capabilities from the current model.

	if len(generated.Capabilities) > 0 {

		internal.Specifications.Capabilities = make([]string, len(generated.Capabilities))

		for i, cap := range generated.Capabilities {

			internal.Specifications.Capabilities[i] = cap.Name

		}

	}

	// Extract features from the current model.

	if len(generated.Features) > 0 {

		internal.Specifications.Features = make([]string, len(generated.Features))

		for i, feature := range generated.Features {

			internal.Specifications.Features[i] = feature.Name

		}

	}

	// Map resource limits to simplified format for backward compatibility.

	if generated.ResourceLimits != nil {

		internal.Specifications.MinResources = make(map[string]string)

		internal.Specifications.DefaultResources = make(map[string]string)

		internal.Specifications.MaxResources = make(map[string]string)

		// Map CPU limits.

		if generated.ResourceLimits.CPULimits != nil {

			if generated.ResourceLimits.CPULimits.MinValue != "" {

				internal.Specifications.MinResources["cpu"] = generated.ResourceLimits.CPULimits.MinValue

			}

			if generated.ResourceLimits.CPULimits.DefaultValue != "" {

				internal.Specifications.DefaultResources["cpu"] = generated.ResourceLimits.CPULimits.DefaultValue

			}

			if generated.ResourceLimits.CPULimits.MaxValue != "" {

				internal.Specifications.MaxResources["cpu"] = generated.ResourceLimits.CPULimits.MaxValue

			}

		}

		// Map Memory limits.

		if generated.ResourceLimits.MemoryLimits != nil {

			if generated.ResourceLimits.MemoryLimits.MinValue != "" {

				internal.Specifications.MinResources["memory"] = generated.ResourceLimits.MemoryLimits.MinValue

			}

			if generated.ResourceLimits.MemoryLimits.DefaultValue != "" {

				internal.Specifications.DefaultResources["memory"] = generated.ResourceLimits.MemoryLimits.DefaultValue

			}

			if generated.ResourceLimits.MemoryLimits.MaxValue != "" {

				internal.Specifications.MaxResources["memory"] = generated.ResourceLimits.MemoryLimits.MaxValue

			}

		}

		// Map Storage limits.

		if generated.ResourceLimits.StorageLimits != nil {

			if generated.ResourceLimits.StorageLimits.MinValue != "" {

				internal.Specifications.MinResources["storage"] = generated.ResourceLimits.StorageLimits.MinValue

			}

			if generated.ResourceLimits.StorageLimits.DefaultValue != "" {

				internal.Specifications.DefaultResources["storage"] = generated.ResourceLimits.StorageLimits.DefaultValue

			}

			if generated.ResourceLimits.StorageLimits.MaxValue != "" {

				internal.Specifications.MaxResources["storage"] = generated.ResourceLimits.StorageLimits.MaxValue

			}

		}

	}

	// Create properties map from extensions.

	internal.Specifications.Properties = make(map[string]interface{})

	if generated.Extensions != nil {

		for k, v := range generated.Extensions {

			internal.Specifications.Properties[k] = v

		}

	}

	return internal

}

// ToGenerated converts our internal representation back to the generated model format.

// This allows us to work with the current O-RAN model when needed.

func (i *InternalResourceType) ToGenerated() *models.ResourceType {

	if i == nil {

		return nil

	}

	generated := &models.ResourceType{

		ResourceTypeID: i.ResourceTypeID,

		Name: i.Name,

		Description: i.Description,

		Vendor: i.Vendor,

		Model: i.Model,

		Version: i.Version,

		Status: i.Status,

		CreatedAt: i.CreatedAt,

		UpdatedAt: i.UpdatedAt,

		CreatedBy: i.CreatedBy,

		UpdatedBy: i.UpdatedBy,
	}

	// Map SupportedActions back to SupportedOperations.

	generated.SupportedOperations = make([]string, len(i.SupportedActions))

	copy(generated.SupportedOperations, i.SupportedActions)

	// Map specifications back to current model structure.

	if i.Specifications != nil {

		generated.Category = i.Specifications.Category

		generated.ResourceClass = i.Specifications.ResourceClass

		generated.ResourceKind = i.Specifications.ResourceKind

		// Map extensions from properties.

		if i.Specifications.Properties != nil {

			generated.Extensions = make(map[string]interface{})

			for k, v := range i.Specifications.Properties {

				generated.Extensions[k] = v

			}

		}

	}

	return generated

}

// CreateDefaultComputeResourceType creates a default compute resource type using our internal format.

func CreateDefaultComputeResourceType() *InternalResourceType {

	return &InternalResourceType{

		ResourceTypeID: "compute-deployment",

		Name: "Kubernetes Deployment",

		Description: "Kubernetes Deployment for compute workloads",

		Vendor: "Kubernetes",

		Model: "Deployment",

		Version: "apps/v1",

		Specifications: &InternalResourceTypeSpec{

			Category: models.ResourceCategoryCompute,

			MinResources: map[string]string{

				"cpu": "100m",

				"memory": "128Mi",
			},

			DefaultResources: map[string]string{

				"cpu": "500m",

				"memory": "512Mi",
			},
		},

		SupportedActions: []string{"create", "update", "delete", "scale"},

		Status: models.ResourceTypeStatusActive,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

}

// CreateDefaultNetworkResourceType creates a default network resource type using our internal format.

func CreateDefaultNetworkResourceType() *InternalResourceType {

	return &InternalResourceType{

		ResourceTypeID: "network-service",

		Name: "Kubernetes Service",

		Description: "Kubernetes Service for network connectivity",

		Vendor: "Kubernetes",

		Model: "Service",

		Version: "v1",

		Specifications: &InternalResourceTypeSpec{

			Category: models.ResourceCategoryNetwork,

			Properties: map[string]interface{}{

				"serviceTypes": []string{"ClusterIP", "NodePort", "LoadBalancer"},

				"protocols": []string{"TCP", "UDP"},
			},
		},

		SupportedActions: []string{"create", "update", "delete"},

		Status: models.ResourceTypeStatusActive,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

}

// CreateDefaultStorageResourceType creates a default storage resource type using our internal format.

func CreateDefaultStorageResourceType() *InternalResourceType {

	return &InternalResourceType{

		ResourceTypeID: "storage-pvc",

		Name: "Persistent Volume Claim",

		Description: "Kubernetes Persistent Volume Claim for storage",

		Vendor: "Kubernetes",

		Model: "PersistentVolumeClaim",

		Version: "v1",

		Specifications: &InternalResourceTypeSpec{

			Category: models.ResourceCategoryStorage,

			Properties: map[string]interface{}{

				"accessModes": []string{"ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany"},

				"volumeModes": []string{"Filesystem", "Block"},
			},
		},

		SupportedActions: []string{"create", "delete"},

		Status: models.ResourceTypeStatusActive,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

}
