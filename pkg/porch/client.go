package porch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

// NewClient creates a new Porch API client
func NewClient(baseURL string, dryRun bool) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		dryRun: dryRun,
	}
}

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
func (c *Client) CreateOrUpdatePackage(req *PackageRequest) (*PorchPackageRevision, error) {
	if c.dryRun {
		return c.dryRunPackage(req)
	}

	// Check if package exists
	existing, err := c.getPackage(req.Repository, req.Package)
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("failed to check existing package: %w", err)
	}

	var revision *PorchPackageRevision
	if existing != nil {
		// Update existing package
		revision, err = c.updatePackage(req, existing)
	} else {
		// Create new package
		revision, err = c.createPackage(req)
	}

	if err != nil {
		return nil, err
	}

	// Apply KRM overlays
	if err := c.applyOverlays(revision, req); err != nil {
		return nil, fmt.Errorf("failed to apply overlays: %w", err)
	}

	return revision, nil
}

// SubmitProposal submits a package proposal for review
func (c *Client) SubmitProposal(revision *PorchPackageRevision) (*Proposal, error) {
	if c.dryRun {
		return &Proposal{
			ID:        "dry-run-proposal",
			Package:   revision.Name,
			Revision:  revision.Revision,
			Status:    "pending",
			CreatedAt: time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/api/v1/proposals", c.baseURL)
	body := map[string]interface{}{
		"packageRevision": revision.Name,
		"namespace":       revision.Namespace,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to submit proposal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to submit proposal: %s", body)
	}

	var proposal Proposal
	if err := json.NewDecoder(resp.Body).Decode(&proposal); err != nil {
		return nil, err
	}

	return &proposal, nil
}

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
	if err != nil {
		return nil, err
	}

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get package: %s", body)
	}

	var revision PorchPackageRevision
	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}

	return &revision, nil
}

func (c *Client) createPackage(req *PackageRequest) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/repositories/%s/packages", c.baseURL, req.Repository)
	
	// Generate vNNN revision format
	revisionNumber := time.Now().Unix() % 1000
	revisionString := fmt.Sprintf("v%03d", revisionNumber)
	
	body := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      req.Package,
			"namespace": req.Namespace,
		},
		"spec": map[string]interface{}{
			"workspace": req.Workspace,
			"intent":    req.Intent,
			"revision":  revisionString,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}
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

	return &revision, nil
}

func (c *Client) updatePackage(req *PackageRequest, existing *PorchPackageRevision) (*PorchPackageRevision, error) {
	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s", c.baseURL, existing.Name)
	
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

	request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/merge-patch+json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to update package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update package: %s", body)
	}

	var revision PorchPackageRevision
	if err := json.NewDecoder(resp.Body).Decode(&revision); err != nil {
		return nil, err
	}

	return &revision, nil
}

func (c *Client) applyOverlays(revision *PorchPackageRevision, req *PackageRequest) error {
	// Generate KRM overlays
	overlays := c.generateOverlays(req.Intent)

	// Apply overlays to the package revision
	url := fmt.Sprintf("%s/api/v1/packagerevisions/%s/resources", c.baseURL, revision.Name)
	
	for path, content := range overlays {
		body := map[string]interface{}{
			"path":    path,
			"content": content,
		}

		data, err := json.Marshal(body)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to apply overlay %s: %w", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to apply overlay %s: status %d", path, resp.StatusCode)
		}
	}

	return nil
}

func (c *Client) generateOverlays(intent *intent.NetworkIntent) map[string]string {
	overlays := make(map[string]string)

	// Generate deployment overlay
	deploymentOverlay := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: default
spec:
  replicas: %d
`, intent.Target, intent.Replicas)

	overlays["overlays/deployment.yaml"] = deploymentOverlay

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
`, intent.Target, string(intentJSON))

	overlays["overlays/configmap.yaml"] = configMapOverlay

	return overlays
}

func (c *Client) dryRunPackage(req *PackageRequest) (*PorchPackageRevision, error) {
	// In dry-run mode, write the request to ./out/
	outDir := "./out"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write package request
	reqFile := filepath.Join(outDir, "porch-package-request.json")
	reqData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, err
	}

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
	}

	fmt.Printf("[porch-client] Dry-run: Files written to %s/\n", outDir)

	// Generate proper vNNN revision format
	revisionNumber := time.Now().Unix() % 1000 // Keep it short for demo
	revision := fmt.Sprintf("v%03d", revisionNumber)
	
	return &PorchPackageRevision{
		Name:      fmt.Sprintf("%s-%s", req.Repository, req.Package),
		Namespace: req.Namespace,
		Revision:  revision,
		Status:    "DRAFT",
	}, nil
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "package not found"
}