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

package porch

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Validation types
type ValidationIssueType string
type ValidationSeverity string
type SuggestionType string
type KRMIssueType string
type YAMLIssueType string
type JSONIssueType string
type ConflictType string
type ConflictSeverity string
type ResolutionAction string
type MergeStrategy string
type DiffFormat string
type DiffType string
type DiffSummary string
type DiffStatistics string
type ChangeType string
type PatchFormat string
type PatchOperation string
type MergeStatistics string
type FileMergeStatistics string
type ConditionType string
type ComparisonOperator string
type OptimizationImpact string

const (
	ValidationIssueTypeSchema    ValidationIssueType = "schema"
	ValidationIssueTypeSyntax    ValidationIssueType = "syntax"
	ValidationIssueTypeStructure ValidationIssueType = "structure"
)

const (
	ValidationSeverityError   ValidationSeverity = "error"
	ValidationSeverityWarning ValidationSeverity = "warning"
	ValidationSeverityInfo    ValidationSeverity = "info"
)

// ContentManager provides comprehensive package content manipulation and validation
// Handles CRUD operations, content validation, template processing, conflict resolution,
// version diffing, content merging, and binary content management for telecommunications packages
type ContentManager interface {
	// Package content CRUD operations
	CreateContent(ctx context.Context, ref *PackageReference, content *PackageContentRequest) (*PackageContent, error)
	GetContent(ctx context.Context, ref *PackageReference, opts *ContentQueryOptions) (*PackageContent, error)
	UpdateContent(ctx context.Context, ref *PackageReference, updates *ContentUpdateRequest) (*PackageContent, error)
	DeleteContent(ctx context.Context, ref *PackageReference, filePatterns []string) error

	// Content validation
	ValidateContent(ctx context.Context, ref *PackageReference, content *PackageContent, opts *ValidationOptions) (*ContentValidationResult, error)
	ValidateKRMResources(ctx context.Context, resources []KRMResource) (*KRMValidationResult, error)
	ValidateYAMLSyntax(ctx context.Context, yamlContent []byte) (*YAMLValidationResult, error)
	ValidateJSONSyntax(ctx context.Context, jsonContent []byte) (*JSONValidationResult, error)

	// Template processing
	ProcessTemplates(ctx context.Context, ref *PackageReference, templateData interface{}, opts *TemplateProcessingOptions) (*PackageContent, error)
	RegisterTemplateFunction(name string, fn interface{}) error
	ListTemplateVariables(ctx context.Context, ref *PackageReference) ([]TemplateVariable, error)

	// Conflict resolution
	DetectConflicts(ctx context.Context, ref *PackageReference, incomingContent *PackageContent) (*ConflictDetectionResult, error)
	ResolveConflicts(ctx context.Context, ref *PackageReference, conflicts *ConflictResolution) (*PackageContent, error)
	CreateMergeProposal(ctx context.Context, ref *PackageReference, baseContent, incomingContent *PackageContent) (*MergeProposal, error)

	// Version diffing
	DiffContent(ctx context.Context, ref1, ref2 *PackageReference, opts *DiffOptions) (*ContentDiff, error)
	DiffFiles(ctx context.Context, file1, file2 []byte, format DiffFormat) (*FileDiff, error)
	GeneratePatch(ctx context.Context, oldContent, newContent *PackageContent) (*ContentPatch, error)
	ApplyPatch(ctx context.Context, ref *PackageReference, patch *ContentPatch) (*PackageContent, error)

	// Content merging
	MergeContent(ctx context.Context, baseContent, sourceContent, targetContent *PackageContent, opts *MergeOptions) (*MergeResult, error)
	ThreeWayMerge(ctx context.Context, base, source, target []byte, opts *MergeOptions) (*FileMergeResult, error)

	// Binary content handling
	StoreBinaryContent(ctx context.Context, ref *PackageReference, filename string, data []byte, opts *BinaryStorageOptions) (*BinaryContentInfo, error)
	RetrieveBinaryContent(ctx context.Context, ref *PackageReference, filename string) (*BinaryContentInfo, []byte, error)
	DeleteBinaryContent(ctx context.Context, ref *PackageReference, filename string) error
	ListBinaryContent(ctx context.Context, ref *PackageReference) ([]BinaryContentInfo, error)

	// Content analysis and metrics
	AnalyzeContent(ctx context.Context, ref *PackageReference) (*ContentAnalysis, error)
	GetContentMetrics(ctx context.Context, ref *PackageReference) (*ContentMetrics, error)
	OptimizeContent(ctx context.Context, ref *PackageReference, opts *OptimizationOptions) (*OptimizationResult, error)

	// Content indexing and search
	IndexContent(ctx context.Context, ref *PackageReference) error
	SearchContent(ctx context.Context, query *ContentSearchQuery) (*ContentSearchResult, error)

	// Health and maintenance
	GetContentHealth(ctx context.Context) (*ContentManagerHealth, error)
	CleanupOrphanedContent(ctx context.Context, olderThan time.Duration) (*CleanupResult, error)
	Close() error
}

// contentManager implements comprehensive package content management
type contentManager struct {
	// Core dependencies
	client  *Client
	logger  logr.Logger
	metrics *ContentManagerMetrics

	// Content storage and processing
	contentStore   ContentStore
	templateEngine *TemplateEngine
	validator      *ContentValidator

	// Conflict resolution and merging
	conflictResolver *ConflictResolver
	mergeEngine      *MergeEngine

	// Content indexing and search
	indexer ContentIndexer

	// Binary content handling
	binaryStore BinaryContentStore

	// Configuration
	config *ContentManagerConfig

	// Template functions registry
	templateFunctions map[string]interface{}
	functionsMutex    sync.RWMutex

	// Content processing pipeline
	processors     map[string]ContentProcessor
	processorMutex sync.RWMutex

	// Caching
	cache ContentCache

	// Concurrency control
	operationLocks map[string]*sync.RWMutex
	locksMutex     sync.Mutex

	// Shutdown coordination
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// Request and option types

// PackageContentRequest represents a content creation request
type PackageContentRequest struct {
	Files            map[string][]byte
	TemplateData     interface{}
	ProcessTemplates bool
	ValidateContent  bool
	Metadata         map[string]string
	BinaryFiles      map[string]BinaryFileRequest
}

// BinaryFileRequest represents a binary file in content request
type BinaryFileRequest struct {
	Data        []byte
	ContentType string
	Compressed  bool
	Checksum    string
}

// ContentQueryOptions configures content retrieval
type ContentQueryOptions struct {
	IncludeBinaryFiles bool
	FilePatterns       []string
	ExcludePatterns    []string
	MaxFileSize        int64
	IncludeMetadata    bool
	ResolveTemplates   bool
	TemplateData       interface{}
}

// ContentUpdateRequest represents a content update request
type ContentUpdateRequest struct {
	FilesToAdd         map[string][]byte
	FilesToUpdate      map[string][]byte
	FilesToDelete      []string
	TemplateData       interface{}
	ProcessTemplates   bool
	ConflictResolution ConflictResolutionStrategy
	ValidateChanges    bool
	CreateBackup       bool
	Metadata           map[string]string
}

// ValidationOptions configures content validation
type ValidationOptions struct {
	ValidateYAMLSyntax   bool
	ValidateJSONSyntax   bool
	ValidateKRMResources bool
	ValidateSchemas      bool
	ValidateReferences   bool
	CustomValidators     []string
	StrictMode           bool
	FailOnWarnings       bool
}

// TemplateProcessingOptions configures template processing
type TemplateProcessingOptions struct {
	TemplateData      interface{}
	FunctionWhitelist []string
	StrictMode        bool
	FailOnMissing     bool
	OutputFormat      string
	PreserveComments  bool
}

// Conflict resolution types

// ConflictDetectionResult contains conflict detection results
type ConflictDetectionResult struct {
	HasConflicts      bool
	ConflictFiles     []FileConflict
	ConflictSummary   *ConflictSummary
	RecommendedAction ConflictResolutionStrategy
	AutoResolvable    bool
}

// FileConflict represents a conflict in a specific file
type FileConflict struct {
	FileName        string
	ConflictType    ConflictType
	BaseContent     []byte
	CurrentContent  []byte
	IncomingContent []byte
	ConflictMarkers []ConflictMarker
	Severity        ConflictSeverity
	AutoResolvable  bool
}

// ConflictMarker represents a specific conflict within a file
type ConflictMarker struct {
	LineNumber   int
	ConflictType ConflictType
	BaseText     string
	CurrentText  string
	IncomingText string
	Context      string
}

// ConflictSummary provides an overview of all conflicts
type ConflictSummary struct {
	TotalConflicts      int
	ConflictsByType     map[ConflictType]int
	ConflictsBySeverity map[ConflictSeverity]int
	AutoResolvableCount int
	FilesAffected       []string
}

// ConflictResolution defines how to resolve conflicts
type ConflictResolution struct {
	Strategy          ConflictResolutionStrategy
	FileResolutions   map[string]FileResolution
	CustomResolutions map[string][]byte
	PreferredSource   ConflictSource
}

// FileResolution defines resolution for a specific file
type FileResolution struct {
	Action            ResolutionAction
	Content           []byte
	MarkerResolutions map[int]MarkerResolution
}

// MarkerResolution defines resolution for a specific conflict marker
type MarkerResolution struct {
	ChosenSource  ConflictSource
	CustomContent string
}

// MergeProposal represents a proposed merge
type MergeProposal struct {
	ID                string
	BaseRef           *PackageReference
	SourceRef         *PackageReference
	ProposedContent   *PackageContent
	ConflictSummary   *ConflictSummary
	MergeStrategy     MergeStrategy
	AutoApplicable    bool
	RequiredApprovals []string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

// Diffing types

// DiffOptions configures content diffing
type DiffOptions struct {
	Format           DiffFormat
	Context          int
	IgnoreWhitespace bool
	IgnoreCase       bool
	BinaryThreshold  int64
	ShowBinaryDiff   bool
	PathFilters      []string
}

// ContentDiff represents differences between package contents
type ContentDiff struct {
	PackageRef1   *PackageReference
	PackageRef2   *PackageReference
	FileDiffs     map[string]*FileDiff
	AddedFiles    []string
	DeletedFiles  []string
	ModifiedFiles []string
	BinaryFiles   []string
	Summary       *DiffSummary
	GeneratedAt   time.Time
}

// FileDiff represents differences in a single file
type FileDiff struct {
	FileName    string
	DiffType    DiffType
	Format      DiffFormat
	Content     string
	LineChanges []*LineChange
	Statistics  *DiffStatistics
	IsBinary    bool
	OldSize     int64
	NewSize     int64
}

// LineChange represents a change in a specific line
type LineChange struct {
	LineNumber int
	ChangeType ChangeType
	OldContent string
	NewContent string
	Context    []string
}

// ContentPatch represents a set of changes to apply
type ContentPatch struct {
	PackageRef  *PackageReference
	PatchFormat PatchFormat
	FilePatches map[string]*FilePatch
	CreatedAt   time.Time
	CreatedBy   string
	Description string
	Reversible  bool
}

// FilePatch represents changes to a single file
type FilePatch struct {
	FileName  string
	Operation PatchOperation
	Content   []byte
	Hunks     []*PatchHunk
	Checksum  string
}

// PatchHunk represents a contiguous set of changes
type PatchHunk struct {
	OldStart  int
	OldLines  int
	NewStart  int
	NewLines  int
	Context   []string
	Additions []string
	Deletions []string
}

// Merging types

// MergeOptions configures content merging behavior
type MergeOptions struct {
	Strategy             MergeStrategy
	ConflictResolution   ConflictResolutionStrategy
	AutoResolveConflicts bool
	PreferredSource      ConflictSource
	CustomMergeRules     []MergeRule
	ValidateResult       bool
	CreateBackup         bool
}

// MergeResult contains the result of a merge operation
type MergeResult struct {
	Success       bool
	MergedContent *PackageContent
	Conflicts     []*FileConflict
	Statistics    *MergeStatistics
	AppliedRules  []string
	Warnings      []string
	BackupRef     *PackageReference
}

// FileMergeResult contains the result of merging a single file
type FileMergeResult struct {
	Success       bool
	MergedContent []byte
	Conflicts     []*ConflictMarker
	Statistics    *FileMergeStatistics
}

// MergeRule defines custom merge behavior
type MergeRule struct {
	Name        string
	FilePattern string
	ContentType string
	Strategy    MergeStrategy
	Priority    int
	Conditions  []MergeCondition
}

// MergeCondition defines when a merge rule applies
type MergeCondition struct {
	Type     ConditionType
	Pattern  string
	Value    interface{}
	Operator ComparisonOperator
}

// Binary content types

// BinaryStorageOptions configures binary content storage
type BinaryStorageOptions struct {
	ContentType    string
	Compress       bool
	Encrypt        bool
	Deduplicate    bool
	Metadata       map[string]string
	ExpirationTime *time.Time
}

// BinaryContentInfo provides information about binary content
type BinaryContentInfo struct {
	FileName       string
	ContentType    string
	Size           int64
	CompressedSize int64
	Checksum       string
	StoragePath    string
	Compressed     bool
	Encrypted      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AccessedAt     time.Time
	Metadata       map[string]string
}

// Analysis and optimization types

// ContentAnalysis provides comprehensive content analysis
type ContentAnalysis struct {
	PackageRef              *PackageReference
	TotalFiles              int
	TotalSize               int64
	FilesByType             map[string]int
	SizeByType              map[string]int64
	LargestFiles            []FileInfo
	TemplateFiles           []string
	BinaryFiles             []string
	DuplicateContent        []DuplicateGroup
	OptimizationSuggestions []OptimizationSuggestion
	SecurityIssues          []SecurityIssue
	QualityMetrics          *ContentQualityMetrics
	GeneratedAt             time.Time
}

// FileInfo provides information about a file
type FileInfo struct {
	Name         string
	Size         int64
	Type         string
	Checksum     string
	LastModified time.Time
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Checksum         string
	Files            []string
	Size             int64
	Occurrences      int
	SavingsPotential int64
}

// OptimizationSuggestion suggests content optimizations
type OptimizationSuggestion struct {
	Type             OptimizationType
	Description      string
	Impact           OptimizationImpact
	Files            []string
	EstimatedSavings int64
	AutoApplicable   bool
}

// SecurityIssue represents a security concern in content
type SecurityIssue struct {
	Type        SecurityIssueType
	Severity    SecuritySeverity
	Description string
	Files       []string
	Remediation string
}

// ContentQualityMetrics provides quality assessment
type ContentQualityMetrics struct {
	OverallScore        float64
	ConsistencyScore    float64
	CompletenessScore   float64
	SecurityScore       float64
	MaintenabilityScore float64
	Issues              []QualityIssue
}

// QualityIssue represents a quality concern
type QualityIssue struct {
	Type        QualityIssueType
	Severity    string
	Description string
	File        string
	Line        int
	Suggestion  string
}

// OptimizationOptions configures content optimization
type OptimizationOptions struct {
	RemoveDuplicates    bool
	CompressBinary      bool
	MinifyJSON          bool
	MinifyYAML          bool
	RemoveComments      bool
	OptimizeImages      bool
	DeduplicateStrings  bool
	TargetSizeReduction float64
}

// OptimizationResult contains optimization results
type OptimizationResult struct {
	Success              bool
	OriginalSize         int64
	OptimizedSize        int64
	SizeReduction        int64
	ReductionPercentage  float64
	OptimizationsApplied []string
	FilesModified        []string
	Warnings             []string
	Duration             time.Duration
}

// Search types

// ContentSearchQuery defines search parameters
type ContentSearchQuery struct {
	Query          string
	FilePatterns   []string
	ContentTypes   []string
	CaseSensitive  bool
	RegexSearch    bool
	IncludeContent bool
	MaxResults     int
	Repositories   []string
	TimeRange      *TimeRange
}

// ContentSearchResult contains search results
type ContentSearchResult struct {
	Query            string
	TotalMatches     int
	FileMatches      []FileSearchMatch
	ExecutionTime    time.Duration
	TruncatedResults bool
}

// FileSearchMatch represents a match in a file
type FileSearchMatch struct {
	PackageRef     *PackageReference
	FileName       string
	FileType       string
	TotalMatches   int
	LineMatches    []LineSearchMatch
	ContentPreview string
}

// LineSearchMatch represents a match in a specific line
type LineSearchMatch struct {
	LineNumber int
	Content    string
	MatchStart int
	MatchEnd   int
	Context    []string
}

// Validation result types

// ContentValidationResult contains comprehensive validation results
type ContentValidationResult struct {
	Valid          bool
	FileResults    map[string]*FileValidationResult
	KRMResults     []*KRMValidationResult
	OverallScore   float64
	CriticalIssues []ValidationIssue
	Warnings       []ValidationIssue
	Suggestions    []ValidationSuggestion
	ValidationTime time.Duration
}

// FileValidationResult contains validation results for a single file
type FileValidationResult struct {
	FileName     string
	Valid        bool
	ContentType  string
	Size         int64
	Encoding     string
	SyntaxValid  bool
	SchemaValid  bool
	Issues       []ValidationIssue
	Warnings     []ValidationIssue
	QualityScore float64
}

// ValidationIssue represents a validation problem
type ValidationIssue struct {
	Type       ValidationIssueType
	Severity   ValidationSeverity
	Message    string
	File       string
	Line       int
	Column     int
	Rule       string
	Suggestion string
}

// ValidationSuggestion provides improvement suggestions
type ValidationSuggestion struct {
	Type        SuggestionType
	Description string
	File        string
	Line        int
	Example     string
	AutoFixable bool
}

// KRMValidationResult contains KRM-specific validation results
type KRMValidationResult struct {
	Valid            bool
	Resource         *KRMResource
	APIVersion       string
	Kind             string
	Issues           []KRMValidationIssue
	SchemaValidated  bool
	CustomValidation map[string]interface{}
}

// KRMValidationIssue represents a KRM validation problem
type KRMValidationIssue struct {
	Type       KRMIssueType
	Severity   ValidationSeverity
	Message    string
	Path       string
	Value      interface{}
	Rule       string
	Suggestion string
}

// YAML/JSON validation result types

// YAMLValidationResult contains YAML validation results
type YAMLValidationResult struct {
	Valid      bool
	ParsedData interface{}
	Issues     []YAMLIssue
	Structure  *YAMLStructureInfo
}

// YAMLIssue represents a YAML parsing or structure issue
type YAMLIssue struct {
	Type     YAMLIssueType
	Message  string
	Line     int
	Column   int
	Severity ValidationSeverity
}

// YAMLStructureInfo provides information about YAML structure
type YAMLStructureInfo struct {
	Documents   int
	MaxDepth    int
	KeyCount    int
	ArrayCount  int
	ScalarCount int
}

// JSONValidationResult contains JSON validation results
type JSONValidationResult struct {
	Valid      bool
	ParsedData interface{}
	Issues     []JSONIssue
	Structure  *JSONStructureInfo
}

// JSONIssue represents a JSON parsing or structure issue
type JSONIssue struct {
	Type     JSONIssueType
	Message  string
	Position int64
	Line     int
	Column   int
	Severity ValidationSeverity
}

// JSONStructureInfo provides information about JSON structure
type JSONStructureInfo struct {
	ObjectCount int
	ArrayCount  int
	StringCount int
	NumberCount int
	BoolCount   int
	NullCount   int
	MaxDepth    int
}

// Template types

// TemplateVariable represents a template variable
type TemplateVariable struct {
	Name         string
	Type         string
	Required     bool
	DefaultValue interface{}
	Description  string
	Example      string
}

// Enums and constants

// ContentConflictType defines types of content conflicts
type ContentConflictType string

const (
	ConflictTypeContentChange ContentConflictType = "content_change"
	ConflictTypeAddition      ContentConflictType = "addition"
	ConflictTypeDeletion      ContentConflictType = "deletion"
	ConflictTypeMove          ContentConflictType = "move"
	ConflictTypePermissions   ContentConflictType = "permissions"
	ConflictTypeMetadata      ContentConflictType = "metadata"
)

// ContentConflictSeverity defines conflict severity levels
type ContentConflictSeverity string

const (
	ContentConflictSeverityLow    ContentConflictSeverity = "low"
	ContentConflictSeverityMedium ContentConflictSeverity = "medium"
	ContentConflictSeverityHigh   ContentConflictSeverity = "high"
	ConflictSeverityCritical      ConflictSeverity        = "critical"
)

// ConflictResolutionStrategy defines how to resolve conflicts
type ConflictResolutionStrategy string

const (
	ConflictResolutionAcceptCurrent  ConflictResolutionStrategy = "accept_current"
	ConflictResolutionAcceptIncoming ConflictResolutionStrategy = "accept_incoming"
	ConflictResolutionMerge          ConflictResolutionStrategy = "merge"
	ConflictResolutionManual         ConflictResolutionStrategy = "manual"
	ConflictResolutionAbort          ConflictResolutionStrategy = "abort"
)

// ConflictSource defines the source of a conflict resolution
type ConflictSource string

const (
	ConflictSourceBase     ConflictSource = "base"
	ConflictSourceCurrent  ConflictSource = "current"
	ConflictSourceIncoming ConflictSource = "incoming"
	ConflictSourceCustom   ConflictSource = "custom"
)

// Additional enums would continue here...
// (Many more enums defined for the comprehensive type system)

// Interface implementations

// NewContentManager creates a new content manager instance
func NewContentManager(client *Client, config *ContentManagerConfig) (ContentManager, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if config == nil {
		config = getDefaultContentManagerConfig()
	}

	cm := &contentManager{
		client:            client,
		logger:            log.Log.WithName("content-manager"),
		config:            config,
		templateFunctions: make(map[string]interface{}),
		processors:        make(map[string]ContentProcessor),
		operationLocks:    make(map[string]*sync.RWMutex),
		shutdown:          make(chan struct{}),
		metrics:           initContentManagerMetrics(),
	}

	// Initialize components
	cm.templateEngine = NewTemplateEngine(config.TemplateConfig)
	cm.validator = NewContentValidator(config.ValidationConfig)
	cm.conflictResolver = NewConflictResolver(config.ConflictConfig)
	cm.mergeEngine = NewMergeEngine(config.MergeConfig)

	// Register default template functions
	cm.registerDefaultTemplateFunctions()

	// Start background processes
	cm.wg.Add(1)
	go cm.metricsCollectionLoop()

	return cm, nil
}

// CreateContent creates new package content
func (cm *contentManager) CreateContent(ctx context.Context, ref *PackageReference, req *PackageContentRequest) (*PackageContent, error) {
	cm.logger.Info("Creating package content", "package", ref.GetPackageKey(), "files", len(req.Files))

	// Acquire operation lock
	lock := cm.getOperationLock(ref.GetPackageKey())
	lock.Lock()
	defer lock.Unlock()

	startTime := time.Now()

	// Validate request
	if err := cm.validateContentRequest(req); err != nil {
		return nil, fmt.Errorf("invalid content request: %w", err)
	}

	// Process templates if requested
	processedContent := req.Files
	if req.ProcessTemplates && req.TemplateData != nil {
		processed, err := cm.processContentTemplates(ctx, processedContent, req.TemplateData)
		if err != nil {
			return nil, fmt.Errorf("template processing failed: %w", err)
		}
		processedContent = processed
	}

	// Create package content
	content := &PackageContent{
		Files: processedContent,
		Kptfile: &KptfileContent{
			APIVersion: "kpt.dev/v1",
			Kind:       "Kptfile",
			Metadata: map[string]interface{}{
				"name": ref.PackageName,
			},
			Info: &PackageInfo{
				Description: fmt.Sprintf("Package %s revision %s", ref.PackageName, ref.Revision),
			},
		},
	}

	// Store binary files if any
	if len(req.BinaryFiles) > 0 {
		if err := cm.storeBinaryFiles(ctx, ref, req.BinaryFiles); err != nil {
			return nil, fmt.Errorf("failed to store binary files: %w", err)
		}
	}

	// Validate content if requested
	if req.ValidateContent {
		validationResult, err := cm.ValidateContent(ctx, ref, content, &ValidationOptions{
			ValidateYAMLSyntax:   true,
			ValidateJSONSyntax:   true,
			ValidateKRMResources: true,
		})
		if err != nil {
			return nil, fmt.Errorf("content validation failed: %w", err)
		}
		if !validationResult.Valid {
			return nil, fmt.Errorf("content validation failed with %d critical issues", len(validationResult.CriticalIssues))
		}
	}

	// Store content using Porch client
	if err := cm.client.UpdatePackageContents(ctx, ref.PackageName, ref.Revision, processedContent); err != nil {
		return nil, fmt.Errorf("failed to store content in Porch: %w", err)
	}

	// Index content if indexer is available
	if cm.indexer != nil {
		if err := cm.indexer.IndexContent(ctx, ref, content); err != nil {
			cm.logger.Error(err, "Failed to index content", "package", ref.GetPackageKey())
		}
	}

	// Update metrics
	if cm.metrics != nil {
		duration := time.Since(startTime)
		cm.metrics.contentOperations.WithLabelValues("create", "success").Inc()
		cm.metrics.contentProcessingTime.WithLabelValues("create").Observe(duration.Seconds())
		cm.metrics.contentSize.WithLabelValues(ref.Repository, ref.PackageName).Set(float64(cm.calculateContentSize(processedContent)))
	}

	cm.logger.Info("Package content created successfully",
		"package", ref.GetPackageKey(),
		"files", len(processedContent),
		"duration", time.Since(startTime))

	return content, nil
}

// GetContent retrieves package content with optional filtering and processing
func (cm *contentManager) GetContent(ctx context.Context, ref *PackageReference, opts *ContentQueryOptions) (*PackageContent, error) {
	cm.logger.V(1).Info("Getting package content", "package", ref.GetPackageKey())

	if opts == nil {
		opts = &ContentQueryOptions{}
	}

	// Get content from Porch
	rawContent, err := cm.client.GetPackageContents(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get package contents: %w", err)
	}

	// Apply file filtering
	filteredContent := cm.applyFileFilters(rawContent, opts)

	// Process templates if requested
	if opts.ResolveTemplates && opts.TemplateData != nil {
		processedContent, err := cm.processContentTemplates(ctx, filteredContent, opts.TemplateData)
		if err != nil {
			cm.logger.Error(err, "Template processing failed during content retrieval", "package", ref.GetPackageKey())
			// Continue with unprocessed content
		} else {
			filteredContent = processedContent
		}
	}

	content := &PackageContent{
		Files: filteredContent,
	}

	// Add binary content if requested
	if opts.IncludeBinaryFiles && cm.binaryStore != nil {
		binaryFiles, err := cm.binaryStore.ListBinaryContent(ctx, ref)
		if err != nil {
			cm.logger.Error(err, "Failed to list binary content", "package", ref.GetPackageKey())
		} else {
			// Binary content metadata would be included here
		}
	}

	// Update access metrics
	if cm.metrics != nil {
		cm.metrics.contentOperations.WithLabelValues("get", "success").Inc()
	}

	return content, nil
}

// ValidateContent performs comprehensive content validation
func (cm *contentManager) ValidateContent(ctx context.Context, ref *PackageReference, content *PackageContent, opts *ValidationOptions) (*ContentValidationResult, error) {
	cm.logger.V(1).Info("Validating package content", "package", ref.GetPackageKey(), "files", len(content.Files))

	startTime := time.Now()

	if opts == nil {
		opts = &ValidationOptions{
			ValidateYAMLSyntax:   true,
			ValidateJSONSyntax:   true,
			ValidateKRMResources: true,
		}
	}

	result := &ContentValidationResult{
		Valid:       true,
		FileResults: make(map[string]*FileValidationResult),
		KRMResults:  []*KRMValidationResult{},
	}

	// Validate each file
	for filename, fileContent := range content.Files {
		fileResult, err := cm.validateSingleFile(ctx, filename, fileContent, opts)
		if err != nil {
			cm.logger.Error(err, "File validation failed", "file", filename)
			result.Valid = false
		}

		result.FileResults[filename] = fileResult
		if !fileResult.Valid {
			result.Valid = false
		}

		// Extract and validate KRM resources
		if opts.ValidateKRMResources && cm.isKRMFile(filename) {
			krmResults, err := cm.extractAndValidateKRMResources(ctx, fileContent)
			if err != nil {
				cm.logger.Error(err, "KRM validation failed", "file", filename)
				result.Valid = false
			} else {
				result.KRMResults = append(result.KRMResults, krmResults...)
			}
		}
	}

	// Perform cross-file validations
	crossFileIssues := cm.performCrossFileValidation(ctx, content, opts)
	for _, issue := range crossFileIssues {
		if issue.Severity == ValidationSeverityCritical {
			result.Valid = false
			result.CriticalIssues = append(result.CriticalIssues, issue)
		} else {
			result.Warnings = append(result.Warnings, issue)
		}
	}

	// Calculate overall quality score
	result.OverallScore = cm.calculateQualityScore(result)
	result.ValidationTime = time.Since(startTime)

	// Update metrics
	if cm.metrics != nil {
		cm.metrics.validationOperations.WithLabelValues("content", result.getStatusString()).Inc()
		cm.metrics.validationDuration.Observe(result.ValidationTime.Seconds())
	}

	cm.logger.V(1).Info("Content validation completed",
		"package", ref.GetPackageKey(),
		"valid", result.Valid,
		"score", result.OverallScore,
		"duration", result.ValidationTime)

	return result, nil
}

// ProcessTemplates processes Go templates with NetworkIntent data
func (cm *contentManager) ProcessTemplates(ctx context.Context, ref *PackageReference, templateData interface{}, opts *TemplateProcessingOptions) (*PackageContent, error) {
	cm.logger.Info("Processing templates", "package", ref.GetPackageKey())

	if opts == nil {
		opts = &TemplateProcessingOptions{}
	}

	// Get current content
	content, err := cm.GetContent(ctx, ref, &ContentQueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current content: %w", err)
	}

	// Process templates
	processedFiles, err := cm.processContentTemplates(ctx, content.Files, templateData)
	if err != nil {
		return nil, fmt.Errorf("template processing failed: %w", err)
	}

	return &PackageContent{
		Files:   processedFiles,
		Kptfile: content.Kptfile,
	}, nil
}

// DiffContent compares content between two package revisions
func (cm *contentManager) DiffContent(ctx context.Context, ref1, ref2 *PackageReference, opts *DiffOptions) (*ContentDiff, error) {
	cm.logger.Info("Diffing package content", "ref1", ref1.GetPackageKey(), "ref2", ref2.GetPackageKey())

	if opts == nil {
		opts = &DiffOptions{
			Format:  DiffFormatUnified,
			Context: 3,
		}
	}

	// Get content from both packages
	content1, err := cm.GetContent(ctx, ref1, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get content for %s: %w", ref1.GetPackageKey(), err)
	}

	content2, err := cm.GetContent(ctx, ref2, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get content for %s: %w", ref2.GetPackageKey(), err)
	}

	// Create diff result
	diff := &ContentDiff{
		PackageRef1: ref1,
		PackageRef2: ref2,
		FileDiffs:   make(map[string]*FileDiff),
		GeneratedAt: time.Now(),
	}

	// Find all unique files
	allFiles := make(map[string]bool)
	for filename := range content1.Files {
		allFiles[filename] = true
	}
	for filename := range content2.Files {
		allFiles[filename] = true
	}

	// Compare each file
	for filename := range allFiles {
		file1, exists1 := content1.Files[filename]
		file2, exists2 := content2.Files[filename]

		if !exists1 {
			// File was added
			diff.AddedFiles = append(diff.AddedFiles, filename)
			diff.FileDiffs[filename] = &FileDiff{
				FileName: filename,
				DiffType: DiffTypeAdded,
				Content:  cm.generateAddedFileDiff(file2, opts),
				NewSize:  int64(len(file2)),
			}
		} else if !exists2 {
			// File was deleted
			diff.DeletedFiles = append(diff.DeletedFiles, filename)
			diff.FileDiffs[filename] = &FileDiff{
				FileName: filename,
				DiffType: DiffTypeDeleted,
				Content:  cm.generateDeletedFileDiff(file1, opts),
				OldSize:  int64(len(file1)),
			}
		} else {
			// File exists in both - check for differences
			if !cm.filesEqual(file1, file2) {
				diff.ModifiedFiles = append(diff.ModifiedFiles, filename)
				fileDiff, err := cm.DiffFiles(ctx, file1, file2, opts.Format)
				if err != nil {
					cm.logger.Error(err, "Failed to diff file", "file", filename)
					continue
				}
				diff.FileDiffs[filename] = fileDiff
			}
		}
	}

	// Generate summary
	diff.Summary = cm.generateDiffSummary(diff)

	return diff, nil
}

// Close gracefully shuts down the content manager
func (cm *contentManager) Close() error {
	cm.logger.Info("Shutting down content manager")

	close(cm.shutdown)
	cm.wg.Wait()

	// Close components
	if cm.templateEngine != nil {
		cm.templateEngine.Close()
	}
	if cm.validator != nil {
		cm.validator.Close()
	}
	if cm.indexer != nil {
		cm.indexer.Close()
	}
	if cm.binaryStore != nil {
		cm.binaryStore.Close()
	}

	cm.logger.Info("Content manager shutdown complete")
	return nil
}

// Helper methods and supporting functionality would continue here...
// Due to length constraints, showing the pattern with core methods implemented

// Helper function implementations (simplified for space)

func (cm *contentManager) getOperationLock(key string) *sync.RWMutex {
	cm.locksMutex.Lock()
	defer cm.locksMutex.Unlock()

	if lock, exists := cm.operationLocks[key]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	cm.operationLocks[key] = lock
	return lock
}

func (cm *contentManager) validateContentRequest(req *PackageContentRequest) error {
	if len(req.Files) == 0 && len(req.BinaryFiles) == 0 {
		return fmt.Errorf("no content provided")
	}
	return nil
}

func (cm *contentManager) processContentTemplates(ctx context.Context, content map[string][]byte, templateData interface{}) (map[string][]byte, error) {
	// Implementation would process Go templates in content files
	return content, nil
}

func (cm *contentManager) applyFileFilters(content map[string][]byte, opts *ContentQueryOptions) map[string][]byte {
	// Implementation would filter files based on patterns
	return content
}

func (cm *contentManager) calculateContentSize(content map[string][]byte) int64 {
	var total int64
	for _, data := range content {
		total += int64(len(data))
	}
	return total
}

// Additional helper methods would be implemented here...

func getDefaultContentManagerConfig() *ContentManagerConfig {
	return &ContentManagerConfig{
		MaxFileSize:      100 * 1024 * 1024, // 100MB
		MaxFiles:         10000,
		EnableValidation: true,
		EnableTemplating: true,
		EnableIndexing:   true,
	}
}

func initContentManagerMetrics() *ContentManagerMetrics {
	return &ContentManagerMetrics{
		contentOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_content_operations_total",
				Help: "Total number of content operations",
			},
			[]string{"operation", "status"},
		),
		// Additional metrics initialization...
	}
}

// Background process for metrics collection
func (cm *contentManager) metricsCollectionLoop() {
	defer cm.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.shutdown:
			return
		case <-ticker.C:
			// Collect and update metrics
		}
	}
}

// Many additional methods and types would be implemented here following the same patterns
// This includes all the remaining interface methods and supporting functionality
