package porch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

<<<<<<< HEAD
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// Client represents a Porch API client.

type Client struct {
	baseURL string

	httpClient *http.Client

	dryRun bool
}

// PackageRequest represents a request to create/update a package.

type PackageRequest struct {
	Repository string `json:"repository"`

	Package string `json:"package"`

	Workspace string `json:"workspace"`

	Namespace string `json:"namespace"`

	Intent *intent.NetworkIntent `json:"intent"`

	Files json.RawMessage `json:"files,omitempty"`
}

// PorchPackageRevision represents a package revision in Porch.

type PorchPackageRevision struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Revision string `json:"revision"`

	Status string `json:"status"`

	Labels map[string]string `json:"labels,omitempty"`
}

// Proposal represents a package proposal in Porch.

type Proposal struct {
	ID string `json:"id"`

	Package string `json:"package"`

	Revision string `json:"revision"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewClient creates a new Porch API client.

func NewClient(baseURL string, dryRun bool) *Client {
	return &Client{
		baseURL: baseURL,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},

=======
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// Client represents a Porch API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	dryRun     bool
}

// PackageRequest represents a request to create/update a package
type PackageRequest struct {
	Repository string                 `json:"repository"`
	Package    string                 `json:"package"`
	Workspace  string                 `json:"workspace"`
	Namespace  string                 `json:"namespace"`
	Intent     *intent.NetworkIntent  `json:"intent"`
	Files      map[string]interface{} `json:"files,omitempty"`
}

// PorchPackageRevision represents a package revision in Porch
type PorchPackageRevision struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Revision  string            `json:"revision"`
	Status    string            `json:"status"` // DRAFT, REVIEW, APPROVED, PUBLISHED
	Labels    map[string]string `json:"labels,omitempty"`
}

// Proposal represents a package proposal in Porch
type Proposal struct {
	ID        string    `json:"id"`
	Package   string    `json:"package"`
	Revision  string    `json:"revision"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// Package represents a Porch package resource for client interface
type Package struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClientPackageSpec   `json:"spec,omitempty"`
	Status            ClientPackageStatus `json:"status,omitempty"`
}

// ClientPackageSpec defines the desired state of Package for client interface
type ClientPackageSpec struct {
	Repository         string             `json:"repository,omitempty"`
	Workspacev1Package Workspacev1Package `json:"workspacev1Package,omitempty"`
}

// ClientPackageStatus defines the observed state of Package for client interface
type ClientPackageStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Workspacev1Package represents workspace package configuration
type Workspacev1Package struct {
	Description string                 `json:"description,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NewClient creates a new Porch API client
func NewClient(baseURL string, dryRun bool) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		dryRun: dryRun,
	}
}

<<<<<<< HEAD
// CreateOrUpdatePackage creates or updates a package in Porch.

=======
// NewClientWithAuth creates a new Porch API client with authentication
func NewClientWithAuth(baseURL, token string, dryRun bool) *Client {
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		dryRun: dryRun,
	}
	
	if token != "" {
		client.httpClient.Transport = &authTransport{
			token: token,
			base:  http.DefaultTransport,
		}
	}
	
	return client
}

// authTransport adds authentication to HTTP requests
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// CreateOrUpdatePackage creates or updates a package in Porch
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (c *Client) CreateOrUpdatePackage(req *PackageRequest) (*PorchPackageRevision, error) {
	if c.dryRun {
		return c.dryRunPackage(req)
	}

<<<<<<< HEAD
	// Check if package exists.

	existing, err := c.getPackage(req.Repository, req.Package)

=======
	// Check if package exists
	existing, err := c.getPackage(req.Repository, req.Package)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("failed to check existing package: %w", err)
	}

	var revision *PorchPackageRevision
<<<<<<< HEAD

	if existing != nil {
		// Update existing package.

		revision, err = c.updatePackage(req, existing)
	} else {
		// Create new package.

=======
	if existing != nil {
		// Update existing package
		revision, err = c.updatePackage(req, existing)
	} else {
		// Create new package
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		revision, err = c.createPackage(req)
	}

	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
	// Apply KRM overlays.

=======
	// Apply KRM overlays
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := c.applyOverlays(revision, req); err != nil {
		return nil, fmt.Errorf("failed to apply overlays: %w", err)
	}

	return revision, nil
}

<<<<<<< HEAD
// SubmitProposal submits a package proposal for review.

func (c *Client) SubmitProposal(revision *PorchPackageRevision) (*Proposal, error) {
	if c.dryRun {
		return &Proposal{
			ID: "dry-run-proposal",

			Package: revision.Name,

			Revision: revision.Revision,

			Status: "pending",

=======
// SubmitProposal submits a package proposal for review
func (c *Client) SubmitProposal(revision *PorchPackageRevision) (*Proposal, error) {
	if c.dryRun {
		return &Proposal{
			ID:        "dry-run-proposal",
			Package:   revision.Name,
			Revision:  revision.Revision,
			Status:    "pending",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			CreatedAt: time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/api/v1/proposals", c.baseURL)
<<<<<<< HEAD

	body := json.RawMessage(`{}`)
=======
	body := map[string]interface{}{
		"packageRevision": revision.Name,
		"namespace":       revision.Namespace,
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to submit proposal: %w", err)
	}
<<<<<<< HEAD

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusCreated {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to submit proposal: %s", body)

	}

	var proposal Proposal

=======
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to submit proposal: %s", body)
	}

	var proposal Proposal
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := json.NewDecoder(resp.Body).Decode(&proposal); err != nil {
		return nil, err
	}

	return &proposal, nil
}

<<<<<<< HEAD
// ApprovePackage approves a package revision.

func (c *Client) ApprovePackage(revision *PorchPackageRevision) error {
	if c.dryRun {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s/approve", c.baseURL, revision.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to approve package: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {

		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to approve package: %s", body)

	}

	return nil
}

// Private helper methods.

func (c *Client) getPackage(repo, pkg string) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/repositories/%s/packages/%s", c.baseURL, repo, pkg)

	resp, err := c.httpClient.Get(url)
=======
// SubmitForReview moves a package revision from DRAFT to REVIEW state
func (c *Client) SubmitForReview(revision *PorchPackageRevision) (*PorchPackageRevision, error) {
	if c.dryRun {
		return &PorchPackageRevision{
			Name:      revision.Name,
			Namespace: revision.Namespace,
			Revision:  revision.Revision,
			Status:    "REVIEW",
			Labels:    revision.Labels,
		}, nil
	}

	return c.updateRevisionStatus(revision, "REVIEW")
}

// ApprovePackage moves a package revision from REVIEW to APPROVED state
func (c *Client) ApprovePackage(revision *PorchPackageRevision) (*PorchPackageRevision, error) {
	if c.dryRun {
		return &PorchPackageRevision{
			Name:      revision.Name,
			Namespace: revision.Namespace,
			Revision:  revision.Revision,
			Status:    "APPROVED",
			Labels:    revision.Labels,
		}, nil
	}

	return c.updateRevisionStatus(revision, "APPROVED")
}

// PublishPackage moves a package revision from APPROVED to PUBLISHED state
func (c *Client) PublishPackage(revision *PorchPackageRevision) (*PorchPackageRevision, error) {
	if c.dryRun {
		return &PorchPackageRevision{
			Name:      revision.Name,
			Namespace: revision.Namespace,
			Revision:  revision.Revision,
			Status:    "PUBLISHED",
			Labels:    revision.Labels,
		}, nil
	}

	return c.updateRevisionStatus(revision, "PUBLISHED")
}

// updateRevisionStatus updates the status of a package revision
func (c *Client) updateRevisionStatus(revision *PorchPackageRevision, status string) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s", c.baseURL, revision.Name)
	body := map[string]interface{}{
		"spec": map[string]interface{}{
			"lifecycle": status,
		},
	}

	data, err := json.Marshal(body)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
	defer resp.Body.Close() // #nosec G307 - Error handled in defer
=======
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/merge-patch+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update package revision status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update package revision status: %s", body)
	}

	var updatedRevision PorchPackageRevision
	if err := json.NewDecoder(resp.Body).Decode(&updatedRevision); err != nil {
		return nil, err
	}

	return &updatedRevision, nil
}

// Private helper methods

func (c *Client) getPackage(repo, pkg string) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/repositories/%s/packages/%s", c.baseURL, repo, pkg)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package not found")
	}

	if resp.StatusCode != http.StatusOK {
<<<<<<< HEAD

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to get package: %s", body)

	}

	var revision PorchPackageRevision

=======
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get package: %s", body)
	}

	var revision PorchPackageRevision
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}

	return &revision, nil
}

func (c *Client) createPackage(req *PackageRequest) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/repositories/%s/packages", c.baseURL, req.Repository)
<<<<<<< HEAD

=======
	
	// Generate vNNN revision format
	revisionNumber := time.Now().Unix() % 1000
	revisionString := fmt.Sprintf("v%03d", revisionNumber)
	
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	body := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      req.Package,
			"namespace": req.Namespace,
		},
<<<<<<< HEAD
		"spec": map[string]interface{}{},
=======
		"spec": map[string]interface{}{
			"workspace": req.Workspace,
			"intent":    req.Intent,
			"revision":  revisionString,
		},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}
<<<<<<< HEAD

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusCreated {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to create package: %s", body)

	}

	var revision PorchPackageRevision

	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}
=======
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create package: %s", body)
	}

	var revision PorchPackageRevision
	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}
	
	// Ensure revision format is vNNN if not set by server
	if revision.Revision == "" {
		revision.Revision = revisionString
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	return &revision, nil
}

func (c *Client) updatePackage(req *PackageRequest, existing *PorchPackageRevision) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s", c.baseURL, existing.Name)
<<<<<<< HEAD

=======
	
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	body := map[string]interface{}{
		"spec": map[string]interface{}{
			"workspace": req.Workspace,
			"intent":    req.Intent,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

=======
	request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	request.Header.Set("Content-Type", "application/merge-patch+json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to update package: %w", err)
	}
<<<<<<< HEAD

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to update package: %s", body)

	}

	var revision PorchPackageRevision

=======
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update package: %s", body)
	}

	var revision PorchPackageRevision
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}

	return &revision, nil
}

func (c *Client) applyOverlays(revision *PorchPackageRevision, req *PackageRequest) error {
<<<<<<< HEAD
	// Generate KRM overlays.

	overlays := c.generateOverlays(req.Intent)

	// Apply overlays to the package revision.

	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s/resources", c.baseURL, revision.Name)

	for path, _ := range overlays {

		body := json.RawMessage(`{}`)
=======
	// Generate KRM overlays
	overlays := c.generateOverlays(req.Intent)

	// Apply overlays to the package revision
	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s/resources", c.baseURL, revision.Name)
	
	for path, content := range overlays {
		body := map[string]interface{}{
			"path":    path,
			"content": content,
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

		data, err := json.Marshal(body)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to apply overlay %s: %w", path, err)
		}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to apply overlay %s: status %d", path, resp.StatusCode)
		}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	return nil
}

func (c *Client) generateOverlays(intent *intent.NetworkIntent) map[string]string {
	overlays := make(map[string]string)

<<<<<<< HEAD
	// Generate deployment overlay.

	deploymentOverlay := fmt.Sprintf(`apiVersion: apps/v1

kind: Deployment

metadata:

  name: %s

  namespace: default

spec:

  replicas: %d

=======
	// Generate deployment overlay
	deploymentOverlay := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: default
spec:
  replicas: %d
>>>>>>> 6835433495e87288b95961af7173d866977175ff
`, intent.Target, intent.Replicas)

	overlays["overlays/deployment.yaml"] = deploymentOverlay

<<<<<<< HEAD
	// Generate ConfigMap with intent.

	intentJSON, _ := json.MarshalIndent(intent, "", "  ")

	configMapOverlay := fmt.Sprintf(`apiVersion: v1

kind: ConfigMap

metadata:

  name: %s-intent

  namespace: default

data:

  intent.json: |

    %s

=======
	// Generate ConfigMap with intent
	intentJSON, _ := json.MarshalIndent(intent, "", "  ")
	configMapOverlay := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-intent
  namespace: default
data:
  intent.json: |
    %s
>>>>>>> 6835433495e87288b95961af7173d866977175ff
`, intent.Target, string(intentJSON))

	overlays["overlays/configmap.yaml"] = configMapOverlay

	return overlays
}

func (c *Client) dryRunPackage(req *PackageRequest) (*PorchPackageRevision, error) {
<<<<<<< HEAD
	// In dry-run mode, write the request to ./out/.

	outDir := "./out"

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write package request.

	reqFile := filepath.Join(outDir, "porch-package-request.json")

=======
	// In dry-run mode, write the request to ./out/
	outDir := "./out"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write package request
	reqFile := filepath.Join(outDir, "porch-package-request.json")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	reqData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
	if err := os.WriteFile(reqFile, reqData, 0o640); err != nil {
		return nil, fmt.Errorf("failed to write request file: %w", err)
	}

	// Write overlays.

	overlays := c.generateOverlays(req.Intent)

	for path, content := range overlays {

		overlayFile := filepath.Join(outDir, path)

		overlayDir := filepath.Dir(overlayFile)

		if err := os.MkdirAll(overlayDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create overlay directory: %w", err)
		}

		if err := os.WriteFile(overlayFile, []byte(content), 0o640); err != nil {
			return nil, fmt.Errorf("failed to write overlay file %s: %w", path, err)
		}

=======
	if err := os.WriteFile(reqFile, reqData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write request file: %w", err)
	}

	// Write overlays
	overlays := c.generateOverlays(req.Intent)
	for path, content := range overlays {
		overlayFile := filepath.Join(outDir, path)
		overlayDir := filepath.Dir(overlayFile)
		
		if err := os.MkdirAll(overlayDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create overlay directory: %w", err)
		}

		if err := os.WriteFile(overlayFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write overlay file %s: %w", path, err)
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	fmt.Printf("[porch-client] Dry-run: Files written to %s/\n", outDir)

<<<<<<< HEAD
	return &PorchPackageRevision{
		Name: fmt.Sprintf("%s-%s", req.Repository, req.Package),

		Namespace: req.Namespace,

		Revision: "dry-run-v1",

		Status: "draft",
=======
	// Generate proper vNNN revision format
	revisionNumber := time.Now().Unix() % 1000 // Keep it short for demo
	revision := fmt.Sprintf("v%03d", revisionNumber)
	
	return &PorchPackageRevision{
		Name:      fmt.Sprintf("%s-%s", req.Repository, req.Package),
		Namespace: req.Namespace,
		Revision:  revision,
		Status:    "DRAFT",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}, nil
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "package not found"
}
<<<<<<< HEAD
=======

// CRUD methods for Package API compatibility

// Create creates a new Package in Porch
func (c *Client) Create(ctx context.Context, pkg *Package) (*Package, error) {
	if c.dryRun {
		// Return a mock package in dry-run mode
		return &Package{
			ObjectMeta: pkg.ObjectMeta,
			Spec:       pkg.Spec,
			Status: ClientPackageStatus{
				Phase: "DRAFT",
			},
		}, nil
	}

	url := fmt.Sprintf("%s/api/v1/packages", c.baseURL)
	data, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal package: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create package: status %d, body: %s", resp.StatusCode, body)
	}

	var createdPkg Package
	if err := json.NewDecoder(resp.Body).Decode(&createdPkg); err != nil {
		return nil, fmt.Errorf("failed to decode created package: %w", err)
	}

	return &createdPkg, nil
}

// Update updates an existing Package in Porch
func (c *Client) Update(ctx context.Context, pkg *Package) (*Package, error) {
	if c.dryRun {
		// Return the updated package in dry-run mode
		return pkg, nil
	}

	url := fmt.Sprintf("%s/api/v1/packages/%s", c.baseURL, pkg.Name)
	data, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal package: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update package: status %d, body: %s", resp.StatusCode, body)
	}

	var updatedPkg Package
	if err := json.NewDecoder(resp.Body).Decode(&updatedPkg); err != nil {
		return nil, fmt.Errorf("failed to decode updated package: %w", err)
	}

	return &updatedPkg, nil
}

// Delete deletes a Package from Porch
func (c *Client) Delete(ctx context.Context, pkg *Package) error {
	if c.dryRun {
		// Return success in dry-run mode
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/packages/%s", c.baseURL, pkg.Name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete package: status %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

// Get retrieves a Package from Porch
func (c *Client) Get(ctx context.Context, name, namespace string) (*Package, error) {
	if c.dryRun {
		// Return a not found error in dry-run mode
		gvr := schema.GroupVersionResource{
			Group:    "porch.kpt.dev",
			Version:  "v1alpha1",
			Resource: "packages",
		}
		return nil, errors.NewNotFound(gvr.GroupResource(), name)
	}

	url := fmt.Sprintf("%s/api/v1/packages/%s", c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		gvr := schema.GroupVersionResource{
			Group:    "porch.kpt.dev",
			Version:  "v1alpha1",
			Resource: "packages",
		}
		return nil, errors.NewNotFound(gvr.GroupResource(), name)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get package: status %d, body: %s", resp.StatusCode, body)
	}

	var pkg Package
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to decode package: %w", err)
	}

	return &pkg, nil
}

// Rollback rolls back a Package to a previous version
func (c *Client) Rollback(ctx context.Context, pkg *Package) (*Package, error) {
	if c.dryRun {
		// Return the package as-is in dry-run mode
		return pkg, nil
	}

	url := fmt.Sprintf("%s/api/v1/packages/%s/rollback", c.baseURL, pkg.Name)
	data, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal package: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to rollback package: status %d, body: %s", resp.StatusCode, body)
	}

	var rolledBackPkg Package
	if err := json.NewDecoder(resp.Body).Decode(&rolledBackPkg); err != nil {
		return nil, fmt.Errorf("failed to decode rolled back package: %w", err)
	}

	return &rolledBackPkg, nil
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
