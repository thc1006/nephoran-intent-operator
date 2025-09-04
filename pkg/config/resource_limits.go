package config

import "time"

// ResourceLimits defines constraints for resource allocation.

type ResourceLimits struct {
	MaxCPU int // Max CPU cores

	MaxMemory int // Max memory in MB

	Timeout time.Duration // Maximum execution time
}

// DefaultResourceLimits provides standard resource constraints.

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxCPU: 4, // 4 cores

		MaxMemory: 8192, // 8 GB

		Timeout: 10 * time.Minute,
	}
}
