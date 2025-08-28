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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// packageRevisionManager implements the PackageRevisionManager interface.
type packageRevisionManager struct {
	// Parent client for API operations.
	client *Client

	// Logger for package operations.
	logger logr.Logger

	// Metrics for package operations.
	metrics *PackageRevisionMetrics

	// Package state tracking.
	packages   map[string]*packageState
	stateMutex sync.RWMutex

	// Content processors for different package types.
	contentProcessors map[string]ContentProcessor

	// Validation engines for package content.
	validators []PackageValidator

	// State change handlers.
	stateChangeHandlers []StateChangeHandler

	// Approval workflow engine.
	workflowEngine WorkflowEngine

	// Diff engine for package comparison.
	diffEngine DiffEngine

	// Content storage interface.
	contentStorage ContentStorage

	// Lock manager for concurrent access.
	lockManager LockManager
}

// packageState tracks the runtime state of a package revision.
type packageState struct {
	// Package reference.
	reference *PackageReference

	// Current lifecycle state.
	lifecycle PackageRevisionLifecycle

	// Content metadata.
	contentSize  int64
	contentHash  string
	lastModified time.Time

	// Validation results.
	validationResults []*ValidationResult
	lastValidation    time.Time

	// Approval workflow status.
	workflowState  WorkflowPhase
	approvalStatus string
	approvers      []string

	// Performance metrics.
	renderTime     time.Duration
	validationTime time.Duration

	// Deployment tracking.
	deployments []DeploymentTarget

	// Lock information.
	locked     bool
	lockedBy   string
	lockedAt   time.Time
	lockReason string

	// Mutex for thread-safe access.
	mutex sync.RWMutex
}

// PackageRevisionMetrics defines Prometheus metrics for package operations.
type PackageRevisionMetrics struct {
	packageOperations  *prometheus.CounterVec
	packageStates      *prometheus.GaugeVec
	validationDuration *prometheus.HistogramVec
	renderDuration     *prometheus.HistogramVec
	contentSize        *prometheus.GaugeVec
	approvalStatus     *prometheus.GaugeVec
	deploymentStatus   *prometheus.GaugeVec
}

// ContentProcessor processes different types of package content.
type ContentProcessor interface {
	ProcessContent(ctx context.Context, content map[string][]byte) (map[string][]byte, error)
	ValidateContent(ctx context.Context, content map[string][]byte) error
	GetContentType() string
}

// PackageValidator validates package content according to specific rules.
type PackageValidator interface {
	Validate(ctx context.Context, pkg *PackageRevision, content map[string][]byte) (*ValidationResult, error)
	GetValidatorName() string
	GetSeverity() string
}

// StateChangeHandler handles package lifecycle state changes.
type StateChangeHandler interface {
	OnStateChange(ctx context.Context, pkg *PackageRevision, oldState, newState PackageRevisionLifecycle)
	GetHandlerName() string
}

// PackageWorkflowEngine manages approval workflows for packages.
type PackageWorkflowEngine interface {
	StartWorkflow(ctx context.Context, pkg *PackageRevision, workflowType string) error
	ApproveWorkflow(ctx context.Context, pkg *PackageRevision, approver string) error
	RejectWorkflow(ctx context.Context, pkg *PackageRevision, approver string, reason string) error
	GetWorkflowStatus(ctx context.Context, pkg *PackageRevision) (*WorkflowStatus, error)
}

// WorkflowStatus is defined in types.go.

// ApprovalRecord tracks individual approval/rejection actions.
type ApprovalRecord struct {
	Approver  string
	Action    string // approve, reject
	Timestamp time.Time
	Reason    string
}

// DiffEngine compares package revisions.
type DiffEngine interface {
	ComparePackages(ctx context.Context, pkg1, pkg2 *PackageRevision) (*ComparisonResult, error)
	CompareContent(ctx context.Context, content1, content2 map[string][]byte) (*ComparisonResult, error)
}

// ContentStorage provides persistent storage for package content.
type ContentStorage interface {
	StoreContent(ctx context.Context, ref *PackageReference, content map[string][]byte) error
	RetrieveContent(ctx context.Context, ref *PackageReference) (map[string][]byte, error)
	DeleteContent(ctx context.Context, ref *PackageReference) error
	GetContentMetadata(ctx context.Context, ref *PackageReference) (*ContentMetadata, error)
}

// ContentMetadata provides metadata about stored content.
type ContentMetadata struct {
	Size         int64
	Hash         string
	ModifiedTime time.Time
	ContentType  string
}

// LockManager manages locks on packages to prevent concurrent modifications.
type LockManager interface {
	AcquireLock(ctx context.Context, ref *PackageReference, owner string, reason string) error
	ReleaseLock(ctx context.Context, ref *PackageReference, owner string) error
	IsLocked(ctx context.Context, ref *PackageReference) (bool, string, error)
	GetLockInfo(ctx context.Context, ref *PackageReference) (*LockInfo, error)
}

// LockInfo provides information about a package lock.
type LockInfo struct {
	Locked     bool
	Owner      string
	Reason     string
	AcquiredAt time.Time
	ExpiresAt  *time.Time
}

// NewPackageRevisionManager creates a new package revision manager.
func NewPackageRevisionManager(client *Client) PackageRevisionManager {
	return &packageRevisionManager{
		client:            client,
		logger:            log.Log.WithName("package-revision-manager"),
		packages:          make(map[string]*packageState),
		contentProcessors: make(map[string]ContentProcessor),
		metrics:           initPackageRevisionMetrics(),
	}
}

// CreatePackage creates a new package from a specification.
func (prm *packageRevisionManager) CreatePackage(ctx context.Context, spec *PackageSpec) (*PackageRevision, error) {
	prm.logger.Info("Creating package", "repository", spec.Repository, "package", spec.PackageName)

	// Validate package specification.
	if err := prm.validatePackageSpec(spec); err != nil {
		return nil, fmt.Errorf("invalid package specification: %w", err)
	}

	// Check if package already exists.
	ref := &PackageReference{
		Repository:  spec.Repository,
		PackageName: spec.PackageName,
		Revision:    spec.Revision,
	}

	if spec.Revision == "" {
		ref.Revision = "v1"
	}

	// Create package revision.
	pkg := &PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", spec.PackageName, ref.Revision),
			Labels: map[string]string{
				LabelComponent:   "package-revision",
				LabelRepository:  spec.Repository,
				LabelPackageName: spec.PackageName,
				LabelRevision:    ref.Revision,
				LabelLifecycle:   string(PackageRevisionLifecycleDraft),
			},
			Annotations: map[string]string{
				AnnotationManagedBy:   "nephoran-porch-client",
				AnnotationPackageName: spec.PackageName,
				AnnotationRevision:    ref.Revision,
			},
		},
		Spec: PackageRevisionSpec{
			PackageName: spec.PackageName,
			Repository:  spec.Repository,
			Revision:    ref.Revision,
			Lifecycle:   PackageRevisionLifecycleDraft,
		},
		Status: PackageRevisionStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Created",
					Status:  metav1.ConditionTrue,
					Reason:  "PackageCreated",
					Message: "Package revision successfully created",
				},
			},
		},
	}

	// Apply labels and annotations from spec.
	if spec.Labels != nil {
		for k, v := range spec.Labels {
			pkg.Labels[k] = v
		}
	}
	if spec.Annotations != nil {
		for k, v := range spec.Annotations {
			pkg.Annotations[k] = v
		}
	}

	// Create package using Porch client.
	created, err := prm.client.CreatePackageRevision(ctx, pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to create package revision: %w", err)
	}

	// Initialize package state.
	state := &packageState{
		reference:     ref,
		lifecycle:     PackageRevisionLifecycleDraft,
		lastModified:  time.Now(),
		workflowState: WorkflowPhasePending,
	}

	prm.stateMutex.Lock()
	prm.packages[ref.GetPackageKey()] = state
	prm.stateMutex.Unlock()

	// Update metrics.
	if prm.metrics != nil {
		prm.metrics.packageOperations.WithLabelValues("create", "success").Inc()
		prm.metrics.packageStates.WithLabelValues(spec.Repository, spec.PackageName, string(PackageRevisionLifecycleDraft)).Inc()
	}

	prm.logger.Info("Successfully created package", "repository", spec.Repository, "package", spec.PackageName, "revision", ref.Revision)
	return created, nil
}

// ClonePackage creates a new package by cloning an existing one.
func (prm *packageRevisionManager) ClonePackage(ctx context.Context, source *PackageReference, target *PackageSpec) (*PackageRevision, error) {
	prm.logger.Info("Cloning package", "source", source.GetPackageKey(), "target", target.PackageName)

	// Get source package content.
	sourceContent, err := prm.GetContent(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to get source package content: %w", err)
	}

	// Create target package.
	targetPkg, err := prm.CreatePackage(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create target package: %w", err)
	}

	// Copy content to target.
	targetRef := &PackageReference{
		Repository:  target.Repository,
		PackageName: target.PackageName,
		Revision:    target.Revision,
	}
	if targetRef.Revision == "" {
		targetRef.Revision = "v1"
	}

	if err := prm.UpdateContent(ctx, targetRef, sourceContent.Files); err != nil {
		return nil, fmt.Errorf("failed to update target package content: %w", err)
	}

	prm.logger.Info("Successfully cloned package", "source", source.GetPackageKey(), "target", targetRef.GetPackageKey())
	return targetPkg, nil
}

// DeletePackage deletes a package and all its revisions.
func (prm *packageRevisionManager) DeletePackage(ctx context.Context, ref *PackageReference) error {
	prm.logger.Info("Deleting package", "package", ref.GetPackageKey())

	// Check if package is locked.
	if prm.lockManager != nil {
		locked, owner, err := prm.lockManager.IsLocked(ctx, ref)
		if err != nil {
			return fmt.Errorf("failed to check package lock status: %w", err)
		}
		if locked {
			return fmt.Errorf("package is locked by %s and cannot be deleted", owner)
		}
	}

	// Delete package revision from Porch.
	if err := prm.client.DeletePackageRevision(ctx, ref.PackageName, ref.Revision); err != nil {
		return fmt.Errorf("failed to delete package revision: %w", err)
	}

	// Delete content from storage.
	if prm.contentStorage != nil {
		if err := prm.contentStorage.DeleteContent(ctx, ref); err != nil {
			prm.logger.Error(err, "Failed to delete package content from storage", "package", ref.GetPackageKey())
		}
	}

	// Remove from state tracking.
	prm.stateMutex.Lock()
	delete(prm.packages, ref.GetPackageKey())
	prm.stateMutex.Unlock()

	// Update metrics.
	if prm.metrics != nil {
		prm.metrics.packageOperations.WithLabelValues("delete", "success").Inc()
	}

	prm.logger.Info("Successfully deleted package", "package", ref.GetPackageKey())
	return nil
}

// CreateRevision creates a new revision of an existing package.
func (prm *packageRevisionManager) CreateRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	prm.logger.Info("Creating revision", "package", ref.GetPackageKey())

	// Get existing package revisions.
	revisions, err := prm.ListRevisions(ctx, ref.PackageName)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing revisions: %w", err)
	}

	// Determine next revision number.
	nextRevision := prm.calculateNextRevision(revisions)

	// Create new revision spec.
	spec := &PackageSpec{
		Repository:  ref.Repository,
		PackageName: ref.PackageName,
		Revision:    nextRevision,
		Lifecycle:   PackageRevisionLifecycleDraft,
	}

	// Clone from the latest published revision if available.
	latestPublished := prm.findLatestPublishedRevision(revisions)
	if latestPublished != nil {
		return prm.ClonePackage(ctx, &PackageReference{
			Repository:  ref.Repository,
			PackageName: ref.PackageName,
			Revision:    latestPublished.Spec.Revision,
		}, spec)
	}

	// Create new revision.
	return prm.CreatePackage(ctx, spec)
}

// GetRevision retrieves a specific package revision.
func (prm *packageRevisionManager) GetRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	pkg, err := prm.client.GetPackageRevision(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision: %w", err)
	}

	// Update state tracking.
	prm.updatePackageState(ref, pkg)

	return pkg, nil
}

// ListRevisions lists all revisions of a package.
func (prm *packageRevisionManager) ListRevisions(ctx context.Context, packageName string) ([]*PackageRevision, error) {
	opts := &ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", LabelPackageName, packageName),
	}

	list, err := prm.client.ListPackageRevisions(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list package revisions: %w", err)
	}

	// Convert to slice and sort by revision.
	revisions := make([]*PackageRevision, len(list.Items))
	for i := range list.Items {
		revisions[i] = &list.Items[i]
	}

	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Spec.Revision < revisions[j].Spec.Revision
	})

	return revisions, nil
}

// CompareRevisions compares two package revisions.
func (prm *packageRevisionManager) CompareRevisions(ctx context.Context, ref1, ref2 *PackageReference) (*ComparisonResult, error) {
	prm.logger.V(1).Info("Comparing revisions", "ref1", ref1.GetPackageKey(), "ref2", ref2.GetPackageKey())

	// Get package revisions.
	pkg1, err := prm.GetRevision(ctx, ref1)
	if err != nil {
		return nil, fmt.Errorf("failed to get first package revision: %w", err)
	}

	pkg2, err := prm.GetRevision(ctx, ref2)
	if err != nil {
		return nil, fmt.Errorf("failed to get second package revision: %w", err)
	}

	// Use diff engine if available.
	if prm.diffEngine != nil {
		return prm.diffEngine.ComparePackages(ctx, pkg1, pkg2)
	}

	// Basic comparison.
	result := &ComparisonResult{}

	// Compare lifecycle states.
	if pkg1.Spec.Lifecycle != pkg2.Spec.Lifecycle {
		result.Modified = append(result.Modified, "lifecycle")
	}

	// Compare content if available.
	content1, err1 := prm.GetContent(ctx, ref1)
	content2, err2 := prm.GetContent(ctx, ref2)

	if err1 == nil && err2 == nil {
		contentDiff := prm.compareContent(content1.Files, content2.Files)
		result.Added = append(result.Added, contentDiff.Added...)
		result.Modified = append(result.Modified, contentDiff.Modified...)
		result.Deleted = append(result.Deleted, contentDiff.Deleted...)
	}

	return result, nil
}

// PromoteToProposed promotes a package revision to proposed state.
func (prm *packageRevisionManager) PromoteToProposed(ctx context.Context, ref *PackageReference) error {
	return prm.changeLifecycle(ctx, ref, PackageRevisionLifecycleProposed)
}

// PromoteToPublished promotes a package revision to published state.
func (prm *packageRevisionManager) PromoteToPublished(ctx context.Context, ref *PackageReference) error {
	return prm.changeLifecycle(ctx, ref, PackageRevisionLifecyclePublished)
}

// RevertToRevision reverts a package to a previous revision.
func (prm *packageRevisionManager) RevertToRevision(ctx context.Context, ref *PackageReference, targetRevision string) error {
	prm.logger.Info("Reverting to revision", "package", ref.GetPackageKey(), "targetRevision", targetRevision)

	// Get target revision content.
	targetRef := &PackageReference{
		Repository:  ref.Repository,
		PackageName: ref.PackageName,
		Revision:    targetRevision,
	}

	targetContent, err := prm.GetContent(ctx, targetRef)
	if err != nil {
		return fmt.Errorf("failed to get target revision content: %w", err)
	}

	// Update current revision content.
	if err := prm.UpdateContent(ctx, ref, targetContent.Files); err != nil {
		return fmt.Errorf("failed to update package content: %w", err)
	}

	prm.logger.Info("Successfully reverted to revision", "package", ref.GetPackageKey(), "targetRevision", targetRevision)
	return nil
}

// UpdateContent updates the content of a package revision.
func (prm *packageRevisionManager) UpdateContent(ctx context.Context, ref *PackageReference, updates map[string][]byte) error {
	prm.logger.V(1).Info("Updating content", "package", ref.GetPackageKey(), "files", len(updates))

	// Check if package is locked.
	if prm.lockManager != nil {
		locked, owner, err := prm.lockManager.IsLocked(ctx, ref)
		if err != nil {
			return fmt.Errorf("failed to check package lock status: %w", err)
		}
		if locked {
			return fmt.Errorf("package is locked by %s and cannot be modified", owner)
		}
	}

	// Validate content.
	if err := prm.validateContent(ctx, ref, updates); err != nil {
		return fmt.Errorf("content validation failed: %w", err)
	}

	// Process content through processors.
	processedContent, err := prm.processContent(ctx, updates)
	if err != nil {
		return fmt.Errorf("content processing failed: %w", err)
	}

	// Store content.
	if prm.contentStorage != nil {
		if err := prm.contentStorage.StoreContent(ctx, ref, processedContent); err != nil {
			return fmt.Errorf("failed to store content: %w", err)
		}
	}

	// Update package using Porch client.
	if err := prm.client.UpdatePackageContents(ctx, ref.PackageName, ref.Revision, processedContent); err != nil {
		return fmt.Errorf("failed to update package contents in Porch: %w", err)
	}

	// Update package state.
	prm.updateContentMetrics(ref, processedContent)

	// Trigger validation.
	go func() {
		if _, err := prm.ValidateContent(ctx, ref); err != nil {
			prm.logger.Error(err, "Content validation failed after update", "package", ref.GetPackageKey())
		}
	}()

	prm.logger.Info("Successfully updated content", "package", ref.GetPackageKey(), "files", len(processedContent))
	return nil
}

// GetContent retrieves the content of a package revision.
func (prm *packageRevisionManager) GetContent(ctx context.Context, ref *PackageReference) (*PackageContent, error) {
	// Try content storage first.
	if prm.contentStorage != nil {
		content, err := prm.contentStorage.RetrieveContent(ctx, ref)
		if err == nil {
			return &PackageContent{Files: content}, nil
		}
		prm.logger.V(1).Info("Content not found in storage, falling back to Porch", "package", ref.GetPackageKey(), "error", err)
	}

	// Fall back to Porch client.
	content, err := prm.client.GetPackageContents(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get package contents: %w", err)
	}

	return &PackageContent{Files: content}, nil
}

// ValidateContent validates the content of a package revision.
func (prm *packageRevisionManager) ValidateContent(ctx context.Context, ref *PackageReference) (*ValidationResult, error) {
	start := time.Now()
	defer func() {
		if prm.metrics != nil {
			prm.metrics.validationDuration.WithLabelValues(ref.Repository, ref.PackageName).Observe(time.Since(start).Seconds())
		}
	}()

	prm.logger.V(1).Info("Validating content", "package", ref.GetPackageKey())

	// Get package and content.
	pkg, err := prm.GetRevision(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision: %w", err)
	}

	content, err := prm.GetContent(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get package content: %w", err)
	}

	// Run all validators.
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	for _, validator := range prm.validators {
		validatorResult, err := validator.Validate(ctx, pkg, content.Files)
		if err != nil {
			prm.logger.Error(err, "Validator failed", "validator", validator.GetValidatorName(), "package", ref.GetPackageKey())
			result.Errors = append(result.Errors, ValidationError{
				Message:  fmt.Sprintf("Validator %s failed: %v", validator.GetValidatorName(), err),
				Severity: "error",
				Code:     "VALIDATOR_ERROR",
			})
			result.Valid = false
			continue
		}

		// Merge results.
		if !validatorResult.Valid {
			result.Valid = false
		}
		result.Errors = append(result.Errors, validatorResult.Errors...)
		result.Warnings = append(result.Warnings, validatorResult.Warnings...)
	}

	// Update package state with validation results.
	prm.updateValidationResults(ref, result)

	prm.logger.V(1).Info("Content validation complete", "package", ref.GetPackageKey(), "valid", result.Valid, "errors", len(result.Errors), "warnings", len(result.Warnings))
	return result, nil
}

// Private helper methods.

// validatePackageSpec validates a package specification.
func (prm *packageRevisionManager) validatePackageSpec(spec *PackageSpec) error {
	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if spec.PackageName == "" {
		return fmt.Errorf("package name is required")
	}
	return nil
}

// changeLifecycle changes the lifecycle state of a package revision.
func (prm *packageRevisionManager) changeLifecycle(ctx context.Context, ref *PackageReference, newLifecycle PackageRevisionLifecycle) error {
	prm.logger.Info("Changing lifecycle", "package", ref.GetPackageKey(), "newLifecycle", newLifecycle)

	// Get current package.
	pkg, err := prm.GetRevision(ctx, ref)
	if err != nil {
		return fmt.Errorf("failed to get package revision: %w", err)
	}

	oldLifecycle := pkg.Spec.Lifecycle

	// Validate transition (simplified check).
	if oldLifecycle == newLifecycle {
		return fmt.Errorf("lifecycle already at target state %s", newLifecycle)
	}

	// Workflow requirements for certain transitions are skipped for now.

	// Update lifecycle.
	pkg.Spec.Lifecycle = newLifecycle

	// Update labels.
	pkg.Labels[LabelLifecycle] = string(newLifecycle)

	// Update package.
	updated, err := prm.client.UpdatePackageRevision(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to update package revision: %w", err)
	}

	// Update state tracking.
	prm.updatePackageState(ref, updated)

	// Update metrics.
	if prm.metrics != nil {
		prm.metrics.packageStates.WithLabelValues(ref.Repository, ref.PackageName, string(oldLifecycle)).Dec()
		prm.metrics.packageStates.WithLabelValues(ref.Repository, ref.PackageName, string(newLifecycle)).Inc()
	}

	// Notify state change handlers.
	for _, handler := range prm.stateChangeHandlers {
		handler.OnStateChange(ctx, updated, oldLifecycle, newLifecycle)
	}

	prm.logger.Info("Successfully changed lifecycle", "package", ref.GetPackageKey(), "oldLifecycle", oldLifecycle, "newLifecycle", newLifecycle)
	return nil
}

// calculateNextRevision calculates the next revision number.
func (prm *packageRevisionManager) calculateNextRevision(revisions []*PackageRevision) string {
	if len(revisions) == 0 {
		return "v1"
	}

	// Find highest revision number.
	maxVersion := 0
	for _, revision := range revisions {
		if strings.HasPrefix(revision.Spec.Revision, "v") {
			var version int
			if _, err := fmt.Sscanf(revision.Spec.Revision, "v%d", &version); err == nil {
				if version > maxVersion {
					maxVersion = version
				}
			}
		}
	}

	return fmt.Sprintf("v%d", maxVersion+1)
}

// findLatestPublishedRevision finds the latest published revision.
func (prm *packageRevisionManager) findLatestPublishedRevision(revisions []*PackageRevision) *PackageRevision {
	var latest *PackageRevision
	for _, revision := range revisions {
		if revision.Spec.Lifecycle == PackageRevisionLifecyclePublished {
			if latest == nil || revision.Spec.Revision > latest.Spec.Revision {
				latest = revision
			}
		}
	}
	return latest
}

// validateContent validates package content using all validators.
func (prm *packageRevisionManager) validateContent(ctx context.Context, ref *PackageReference, content map[string][]byte) error {
	// Basic validation.
	if len(content) == 0 {
		return fmt.Errorf("package content cannot be empty")
	}

	// Check for required files (could be configurable).
	requiredFiles := []string{"Kptfile"}
	for _, file := range requiredFiles {
		if _, exists := content[file]; !exists {
			return fmt.Errorf("required file %s is missing", file)
		}
	}

	// Validate Kptfile format.
	if kptfileData, exists := content["Kptfile"]; exists {
		var kptfile map[string]interface{}
		if err := json.Unmarshal(kptfileData, &kptfile); err != nil {
			// Try YAML.
			if err2 := json.Unmarshal(kptfileData, &kptfile); err2 != nil {
				return fmt.Errorf("invalid Kptfile format: %w", err)
			}
		}

		// Validate required Kptfile fields.
		if _, exists := kptfile["apiVersion"]; !exists {
			return fmt.Errorf("Kptfile missing required field: apiVersion")
		}
		if _, exists := kptfile["kind"]; !exists {
			return fmt.Errorf("Kptfile missing required field: kind")
		}
	}

	return nil
}

// processContent processes content through registered processors.
func (prm *packageRevisionManager) processContent(ctx context.Context, content map[string][]byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for k, v := range content {
		result[k] = v
	}

	// Apply content processors.
	for contentType, processor := range prm.contentProcessors {
		if prm.shouldApplyProcessor(contentType, content) {
			processed, err := processor.ProcessContent(ctx, result)
			if err != nil {
				return nil, fmt.Errorf("content processor %s failed: %w", contentType, err)
			}
			result = processed
		}
	}

	return result, nil
}

// shouldApplyProcessor determines if a processor should be applied.
func (prm *packageRevisionManager) shouldApplyProcessor(contentType string, content map[string][]byte) bool {
	// Simple heuristic - apply processor if relevant files are present.
	switch contentType {
	case "yaml":
		for filename := range content {
			if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
				return true
			}
		}
	case "kustomize":
		_, exists := content["kustomization.yaml"]
		return exists
	case "helm":
		_, exists := content["Chart.yaml"]
		return exists
	}
	return false
}

// compareContent compares two content maps.
func (prm *packageRevisionManager) compareContent(content1, content2 map[string][]byte) *ComparisonResult {
	result := &ComparisonResult{}

	// Find added and modified files.
	for filename, data2 := range content2 {
		if data1, exists := content1[filename]; exists {
			// File exists in both - check if modified.
			if string(data1) != string(data2) {
				result.Modified = append(result.Modified, filename)
			}
		} else {
			// File only exists in content2 - added.
			result.Added = append(result.Added, filename)
		}
	}

	// Find deleted files.
	for filename := range content1 {
		if _, exists := content2[filename]; !exists {
			result.Deleted = append(result.Deleted, filename)
		}
	}

	return result
}

// updatePackageState updates the cached package state.
func (prm *packageRevisionManager) updatePackageState(ref *PackageReference, pkg *PackageRevision) {
	prm.stateMutex.Lock()
	defer prm.stateMutex.Unlock()

	state, exists := prm.packages[ref.GetPackageKey()]
	if !exists {
		state = &packageState{reference: ref}
		prm.packages[ref.GetPackageKey()] = state
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.lifecycle = pkg.Spec.Lifecycle
	state.lastModified = time.Now()

	// Note: ValidationResults field doesn't exist in multicluster.PackageRevisionStatus.
	// In a real implementation, validation results would be stored separately or in annotations.
	// For now, we'll check for validation-related conditions.
	for _, condition := range pkg.Status.Conditions {
		if condition.Type == "ValidationPassed" || condition.Type == "ValidationFailed" {
			// Create a validation result based on the condition.
			validationResult := &ValidationResult{
				Valid:    condition.Status == "True",
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}
			if condition.Status != "True" {
				validationResult.Errors = append(validationResult.Errors, ValidationError{
					Message:  condition.Message,
					Severity: "error",
				})
			}
			state.validationResults = []*ValidationResult{validationResult}
			state.lastValidation = time.Now()
			break
		}
	}
}

// updateContentMetrics updates content-related metrics.
func (prm *packageRevisionManager) updateContentMetrics(ref *PackageReference, content map[string][]byte) {
	if prm.metrics == nil {
		return
	}

	// Calculate total content size.
	totalSize := int64(0)
	for _, data := range content {
		totalSize += int64(len(data))
	}

	prm.metrics.contentSize.WithLabelValues(ref.Repository, ref.PackageName).Set(float64(totalSize))

	// Update package state.
	prm.stateMutex.RLock()
	state, exists := prm.packages[ref.GetPackageKey()]
	prm.stateMutex.RUnlock()

	if exists {
		state.mutex.Lock()
		state.contentSize = totalSize
		state.lastModified = time.Now()
		state.mutex.Unlock()
	}
}

// updateValidationResults updates validation results in package state.
func (prm *packageRevisionManager) updateValidationResults(ref *PackageReference, result *ValidationResult) {
	prm.stateMutex.RLock()
	state, exists := prm.packages[ref.GetPackageKey()]
	prm.stateMutex.RUnlock()

	if exists {
		state.mutex.Lock()
		state.validationResults = []*ValidationResult{result}
		state.lastValidation = time.Now()
		state.mutex.Unlock()
	}
}

// initPackageRevisionMetrics initializes Prometheus metrics.
func initPackageRevisionMetrics() *PackageRevisionMetrics {
	return &PackageRevisionMetrics{
		packageOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_package_operations_total",
				Help: "Total number of package operations",
			},
			[]string{"operation", "status"},
		),
		packageStates: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_package_states_total",
				Help: "Number of packages in each state",
			},
			[]string{"repository", "package", "state"},
		),
		validationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "porch_package_validation_duration_seconds",
				Help:    "Duration of package validation operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"repository", "package"},
		),
		renderDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "porch_package_render_duration_seconds",
				Help:    "Duration of package rendering operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"repository", "package"},
		),
		contentSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_package_content_size_bytes",
				Help: "Size of package content in bytes",
			},
			[]string{"repository", "package"},
		),
		approvalStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_package_approval_status",
				Help: "Package approval status (1=approved, 0=pending/rejected)",
			},
			[]string{"repository", "package"},
		),
		deploymentStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_package_deployment_status",
				Help: "Package deployment status (1=deployed, 0=not deployed)",
			},
			[]string{"repository", "package", "cluster"},
		),
	}
}

// GetPackageRevisionMetrics returns package revision metrics.
func (prm *packageRevisionManager) GetPackageRevisionMetrics() *PackageRevisionMetrics {
	return prm.metrics
}

// AddContentProcessor adds a content processor.
func (prm *packageRevisionManager) AddContentProcessor(processor ContentProcessor) {
	prm.contentProcessors[processor.GetContentType()] = processor
}

// AddValidator adds a package validator.
func (prm *packageRevisionManager) AddValidator(validator PackageValidator) {
	prm.validators = append(prm.validators, validator)
}

// AddStateChangeHandler adds a state change handler.
func (prm *packageRevisionManager) AddStateChangeHandler(handler StateChangeHandler) {
	prm.stateChangeHandlers = append(prm.stateChangeHandlers, handler)
}

// SetWorkflowEngine sets the workflow engine.
func (prm *packageRevisionManager) SetWorkflowEngine(engine WorkflowEngine) {
	prm.workflowEngine = engine
}

// SetDiffEngine sets the diff engine.
func (prm *packageRevisionManager) SetDiffEngine(engine DiffEngine) {
	prm.diffEngine = engine
}

// SetContentStorage sets the content storage.
func (prm *packageRevisionManager) SetContentStorage(storage ContentStorage) {
	prm.contentStorage = storage
}

// SetLockManager sets the lock manager.
func (prm *packageRevisionManager) SetLockManager(manager LockManager) {
	prm.lockManager = manager
}

// Close gracefully shuts down the package revision manager.
func (prm *packageRevisionManager) Close() error {
	prm.logger.Info("Shutting down package revision manager")

	// Clear state.
	prm.stateMutex.Lock()
	prm.packages = make(map[string]*packageState)
	prm.stateMutex.Unlock()

	prm.logger.Info("Package revision manager shut down complete")
	return nil
}
