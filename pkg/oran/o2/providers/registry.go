package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Rest of the existing ProviderRegistry implementation 
// (all methods remain exactly the same as in the previous version)

// ProviderSelectionCriteria defines criteria for selecting a provider.
type ProviderSelectionCriteria struct {
	Type string // Provider type (kubernetes, openstack, aws, etc.)

	Region string // Preferred region

	Zone string // Preferred zone

	RequireHealthy bool // Only select healthy providers

	RequiredCapabilities []string // Required capabilities

	SelectionStrategy string // Selection strategy (random, least-loaded, round-robin)
}

// Note: Remove the DefaultProviderFactory and its methods 
// These are now defined in factory.go