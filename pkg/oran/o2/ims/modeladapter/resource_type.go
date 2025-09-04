package modeladapter

import (
	"encoding/json"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// InternalResourceType represents internal adapter structure for resource types
type InternalResourceType struct {
	ResourceTypeID   string
	Name            string
	Description     string
	Vendor          string
	Model           string
	Version         string
	Specifications  *InternalResourceTypeSpec
	SupportedActions []string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// InternalResourceTypeSpec represents internal specifications structure
type InternalResourceTypeSpec struct {
	Category   string
	Properties json.RawMessage
}

// CreateDefaultNetworkResourceType creates a default network resource type
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

			Properties: json.RawMessage(`{
				"protocols": ["TCP", "UDP"]
			}`),
		},

		SupportedActions: []string{"create", "update", "delete"},

		Status: models.ResourceTypeStatusActive,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}
}

// ToGenerated converts internal type to generated models type
func (i *InternalResourceType) ToGenerated() *models.ResourceType {
	return &models.ResourceType{
		ResourceTypeID:   i.ResourceTypeID,
		Name:            i.Name,
		Description:     i.Description,
		Vendor:          i.Vendor,
		Model:           i.Model,
		Version:         i.Version,
		Category:        i.Specifications.Category,
		SupportedActions: i.SupportedActions,
		Status:          i.Status,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
		Specifications:  &models.ResourceTypeSpec{
			Category:   i.Specifications.Category,
			Properties: i.Specifications.Properties,
		},
	}
}

// FromGenerated converts generated models type to internal type
func FromGenerated(rt *models.ResourceType) *InternalResourceType {
	internal := &InternalResourceType{
		ResourceTypeID:   rt.ResourceTypeID,
		Name:            rt.Name,
		Description:     rt.Description,
		Vendor:          rt.Vendor,
		Model:           rt.Model,
		Version:         rt.Version,
		SupportedActions: rt.SupportedActions,
		Status:          rt.Status,
		CreatedAt:       rt.CreatedAt,
		UpdatedAt:       rt.UpdatedAt,
	}
	
	// Handle specifications
	if rt.Specifications != nil {
		internal.Specifications = &InternalResourceTypeSpec{
			Category:   rt.Specifications.Category,
			Properties: rt.Specifications.Properties,
		}
	} else if rt.Category != "" {
		// Handle direct category field for backward compatibility
		internal.Specifications = &InternalResourceTypeSpec{
			Category: rt.Category,
		}
	}
	
	return internal
}

// CreateDefaultComputeResourceType creates a default compute resource type
func CreateDefaultComputeResourceType() *InternalResourceType {
	return &InternalResourceType{
		ResourceTypeID: "compute-vm",
		Name:          "Virtual Machine",
		Description:   "Virtual compute instance",
		Vendor:        "OpenStack",
		Model:         "VM",
		Version:       "v1",
		Specifications: &InternalResourceTypeSpec{
			Category: models.ResourceCategoryCompute,
			Properties: json.RawMessage(`{
				"vcpu": {"min": 1, "max": 64},
				"memory": {"min": "1Gi", "max": "256Gi"},
				"disk": {"min": "10Gi", "max": "1Ti"}
			}`),
		},
		SupportedActions: []string{"create", "update", "delete", "resize"},
		Status:          models.ResourceTypeStatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// CreateDefaultStorageResourceType creates a default storage resource type
func CreateDefaultStorageResourceType() *InternalResourceType {
	return &InternalResourceType{
		ResourceTypeID: "storage-volume",
		Name:          "Block Storage Volume",
		Description:   "Persistent block storage volume",
		Vendor:        "Ceph",
		Model:         "RBD",
		Version:       "v1",
		Specifications: &InternalResourceTypeSpec{
			Category: models.ResourceCategoryStorage,
			Properties: json.RawMessage(`{
				"capacity": {"min": "1Gi", "max": "10Ti"},
				"iops": {"min": 100, "max": 10000},
				"replication": {"default": 3}
			}`),
		},
		SupportedActions: []string{"create", "update", "delete", "snapshot"},
		Status:          models.ResourceTypeStatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// MatchesResourceTypeFilter checks if a resource type matches the given filter
func MatchesResourceTypeFilter(rt *models.ResourceType, filter *models.ResourceTypeFilter) bool {
	if filter == nil {
		return true
	}

	// Check names filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if rt.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check categories filter using adapter for future-proofing
	if len(filter.Categories) > 0 {
		internal := FromGenerated(rt)
		if internal.Specifications != nil {
			found := false
			for _, category := range filter.Categories {
				if internal.Specifications.Category == category {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check vendors filter (only once)
	if len(filter.Vendors) > 0 {
		found := false
		for _, vendor := range filter.Vendors {
			if rt.Vendor == vendor {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check models filter
	if len(filter.Models) > 0 {
		found := false
		for _, model := range filter.Models {
			if rt.Model == model {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check versions filter
	if len(filter.Versions) > 0 {
		found := false
		for _, version := range filter.Versions {
			if rt.Version == version {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}