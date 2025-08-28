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

package audit

import (
	"context"

	audittypes "github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// Re-export types from the types package for convenience (no redeclaration)
type Backend = audittypes.Backend

// ComplianceLogger interface for compliance logging
type ComplianceLogger interface {
	LogCompliance(ctx context.Context, event *audittypes.AuditEvent) error
}

// ComplianceConfig represents configuration for compliance tracking
type ComplianceConfig struct {
	Enabled             bool                            `json:"enabled"`
	Standards           []audittypes.ComplianceStandard `json:"standards"`
	RetentionPolicyDays int                             `json:"retentionPolicyDays"`
	EncryptionEnabled   bool                            `json:"encryptionEnabled"`
}