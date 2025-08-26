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

package dependencies

import (
	"time"
)

// Final resolver types

// ConstraintCache represents a cache for constraint solving results
type ConstraintCache struct {
	MaxSize     int                    `json:"maxSize"`
	TTL         time.Duration          `json:"ttl"`
	Entries     map[string]*CacheEntry `json:"entries"`
	Stats       *CacheStats            `json:"stats"`
	LastCleanup time.Time              `json:"lastCleanup"`
}

// CacheEntry represents a single cache entry
type CacheEntry struct {
	Key       string    `json:"key"`
	Value     interface{} `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	HitCount  int64     `json:"hitCount"`
}

// SolverHeuristic represents a heuristic for constraint solving
type SolverHeuristic interface {
	Name() string
	Description() string
	Score(solution *ConstraintSolution) float64
	Compare(a, b *ConstraintSolution) int
}

// ConstraintOptimizer represents an optimizer for constraint solutions
type ConstraintOptimizer struct {
	Strategies []string          `json:"strategies"`
	Objectives []string          `json:"objectives"`
	Weights    map[string]float64 `json:"weights"`
	Timeout    time.Duration     `json:"timeout"`
	MaxIterations int            `json:"maxIterations"`
}

// VersionSolverMetrics represents metrics for version solving
type VersionSolverMetrics struct {
	VersionsEvaluated    int64         `json:"versionsEvaluated"`
	VersionsSelected     int64         `json:"versionsSelected"`
	VersionsRejected     int64         `json:"versionsRejected"`
	AvgSelectionTime     time.Duration `json:"avgSelectionTime"`
	CandidatesGenerated  int64         `json:"candidatesGenerated"`
	PrereleasesConsidered int64        `json:"prereleasesConsidered"`
	StableVersionsPreferred int64      `json:"stableVersionsPreferred"`
	DowngradeCount       int64         `json:"downgradeCount"`
	UpgradeCount         int64         `json:"upgradeCount"`
}

// VersionCache represents a cache for version information
type VersionCache struct {
	MaxSize        int                      `json:"maxSize"`
	TTL            time.Duration            `json:"ttl"`
	Versions       map[string]*VersionInfo  `json:"versions"`
	Metadata       map[string]*VersionMetadata `json:"metadata"`
	Stats          *CacheStats              `json:"stats"`
	LastRefresh    time.Time                `json:"lastRefresh"`
	RefreshInterval time.Duration           `json:"refreshInterval"`
}

// VersionInfo represents information about a package version
type VersionInfo struct {
	Version       string                 `json:"version"`
	Stability     string                 `json:"stability"`
	PublishedAt   time.Time              `json:"publishedAt"`
	Dependencies  []*PackageReference    `json:"dependencies,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	DownloadCount int64                  `json:"downloadCount,omitempty"`
	Size          int64                  `json:"size,omitempty"`
	Checksums     map[string]string      `json:"checksums,omitempty"`
}

// VersionMetadata represents metadata for version caching
type VersionMetadata struct {
	CachedAt     time.Time `json:"cachedAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Source       string    `json:"source"`
	Verified     bool      `json:"verified"`
	HitCount     int64     `json:"hitCount"`
	LastAccessed time.Time `json:"lastAccessed"`
}

// VersionComparator represents a version comparison interface
type VersionComparator interface {
	Compare(version1, version2 string) int
	IsCompatible(version, constraint string) bool
	GetMajorVersion(version string) int
	GetMinorVersion(version string) int
	GetPatchVersion(version string) int
	IsPrerelease(version string) bool
	IsStable(version string) bool
}

// VersionSelector represents a version selection interface
type VersionSelector interface {
	SelectBestVersion(candidates []*VersionCandidate, requirements []*VersionRequirement) (*VersionCandidate, error)
	FilterCandidates(candidates []*VersionCandidate, constraint string) []*VersionCandidate
	RankCandidates(candidates []*VersionCandidate) []*VersionCandidate
	GetSelectionStrategy() string
}

// PrereleaseStrategy represents strategy for handling prerelease versions
type PrereleaseStrategy string

const (
	PrereleaseStrategyExclude PrereleaseStrategy = "exclude"
	PrereleaseStrategyInclude PrereleaseStrategy = "include"
	PrereleaseStrategyPrefer  PrereleaseStrategy = "prefer"
	PrereleaseStrategyAuto    PrereleaseStrategy = "auto"
)

// BuildMetadataStrategy represents strategy for handling build metadata
type BuildMetadataStrategy string

const (
	BuildMetadataStrategyIgnore  BuildMetadataStrategy = "ignore"
	BuildMetadataStrategyInclude BuildMetadataStrategy = "include"
	BuildMetadataStrategyPrefer  BuildMetadataStrategy = "prefer"
)

// ConflictResolverMetrics represents metrics for conflict resolution
type ConflictResolverMetrics struct {
	ConflictsDetected       int64         `json:"conflictsDetected"`
	ConflictsResolved       int64         `json:"conflictsResolved"`
	ConflictsFailed         int64         `json:"conflictsFailed"`
	AvgResolutionTime       time.Duration `json:"avgResolutionTime"`
	MaxResolutionTime       time.Duration `json:"maxResolutionTime"`
	AutoResolvedConflicts   int64         `json:"autoResolvedConflicts"`
	ManualResolvedConflicts int64         `json:"manualResolvedConflicts"`
	VersionDowngrades       int64         `json:"versionDowngrades"`
	VersionUpgrades         int64         `json:"versionUpgrades"`
	PackageExclusions       int64         `json:"packageExclusions"`
	AlternativeSelections   int64         `json:"alternativeSelections"`
	ResolutionStrategiesUsed map[string]int64 `json:"resolutionStrategiesUsed"`
}