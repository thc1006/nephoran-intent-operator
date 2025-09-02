package modeladapter

import (
	"encoding/json"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

func (c *CatalogService) CreateDefaultNetworkResourceType() *InternalResourceType {
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

// Removing the duplicate vendor filter logic from the matchesResourceTypeFilter method
func (c *CatalogService) matchesResourceTypeFilter(rt *models.ResourceType, filter *models.ResourceTypeFilter) bool {
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
		internal := modeladapter.FromGenerated(rt)
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