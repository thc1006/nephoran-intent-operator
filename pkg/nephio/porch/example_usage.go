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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// ExamplePorchClientUsage demonstrates how to use the real Porch API client
func ExamplePorchClientUsage() error {
	// Create logger
	logger := zap.New(zap.UseDevMode(true))

	// Create Porch client configuration
	config := DefaultPorchConfig()

	// Optional: Configure authentication for specific environments
	// config.AuthConfig = &AuthConfig{
	//     Type:  "bearer",
	//     Token: "your-auth-token",
	// }

	// Optional: Configure custom endpoint
	// config.Endpoint = "https://your-porch-server:8443"

	// Create client options
	opts := ClientOptions{
		Config:         config,
		Logger:         logger,
		MetricsEnabled: true,
		CacheEnabled:   true,
		CacheSize:      100,
		CacheTTL:       5 * time.Minute,
	}

	// Create the Porch client
	client, err := NewClient(opts)
	if err != nil {
		return fmt.Errorf("failed to create Porch client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: List all repositories
	fmt.Println("=== Listing Repositories ===")
	repos, err := client.ListRepositories(ctx, &ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}
	fmt.Printf("Found %d repositories\n", len(repos.Items))

	// Example 2: Create a new repository (if needed)
	fmt.Println("\n=== Creating Repository ===")
	newRepo := &Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-repo",
		},
		Spec: RepositorySpec{
			Type:   "git",
			URL:    "https://github.com/example/packages.git",
			Branch: "main",
		},
	}

	createdRepo, err := client.CreateRepository(ctx, newRepo)
	if err != nil {
		// Repository might already exist - that's okay
		fmt.Printf("Repository creation note: %v\n", err)
	} else {
		fmt.Printf("Created repository: %s\n", createdRepo.Name)
	}

	// Example 3: List package revisions
	fmt.Println("\n=== Listing Package Revisions ===")
	pkgs, err := client.ListPackageRevisions(ctx, &ListOptions{
		LabelSelector: "porch.kpt.dev/repository=example-repo",
	})
	if err != nil {
		return fmt.Errorf("failed to list package revisions: %w", err)
	}
	fmt.Printf("Found %d package revisions\n", len(pkgs.Items))

	// Example 4: Create a new package revision
	fmt.Println("\n=== Creating Package Revision ===")
	newPackage := &PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-package-v1",
		},
		Spec: PackageRevisionSpec{
			PackageName: "example-package",
			Repository:  "example-repo",
			Revision:    "v1.0.0",
			Lifecycle:   PackageRevisionLifecycleDraft,
			Resources: []KRMResource{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Metadata: map[string]interface{}{
						"name":      "example-config",
						"namespace": "default",
					},
					Data: map[string]interface{}{
						"message": "Hello from Porch!",
						"version": "1.0.0",
					},
				},
			},
		},
	}

	createdPackage, err := client.CreatePackageRevision(ctx, newPackage)
	if err != nil {
		return fmt.Errorf("failed to create package revision: %w", err)
	}
	fmt.Printf("Created package revision: %s\n", createdPackage.Name)

	// Example 5: Get package contents
	fmt.Println("\n=== Getting Package Contents ===")
	contents, err := client.GetPackageContents(ctx, "example-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to get package contents: %w", err)
	}
	fmt.Printf("Package contains %d files:\n", len(contents))
	for filename := range contents {
		fmt.Printf("  - %s\n", filename)
	}

	// Example 6: Validate package
	fmt.Println("\n=== Validating Package ===")
	validation, err := client.ValidatePackage(ctx, "example-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to validate package: %w", err)
	}
	fmt.Printf("Package validation: valid=%v, errors=%d, warnings=%d\n",
		validation.Valid, len(validation.Errors), len(validation.Warnings))

	// Example 7: Render package (if functions are defined)
	fmt.Println("\n=== Rendering Package ===")
	renderResult, err := client.RenderPackage(ctx, "example-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to render package: %w", err)
	}
	fmt.Printf("Rendered package: %d resources, %d results\n",
		len(renderResult.Resources), len(renderResult.Results))

	// Example 8: Package lifecycle management
	fmt.Println("\n=== Package Lifecycle Management ===")

	// Propose the package for review
	err = client.ProposePackageRevision(ctx, "example-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to propose package: %w", err)
	}
	fmt.Println("Package proposed for review")

	// Approve the package (would normally require proper authorization)
	err = client.ApprovePackageRevision(ctx, "example-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to approve package: %w", err)
	}
	fmt.Println("Package approved and published")

	// Example 9: Update package contents
	fmt.Println("\n=== Updating Package Contents ===")
	updatedContents := map[string][]byte{
		"configmap.yaml": []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-updated
  namespace: default
data:
  message: "Updated hello from Porch!"
  version: "1.1.0"
`),
	}

	err = client.UpdatePackageContents(ctx, "example-package", "v1.0.0", updatedContents)
	if err != nil {
		return fmt.Errorf("failed to update package contents: %w", err)
	}
	fmt.Println("Package contents updated")

	// Example 10: Execute a KRM function
	fmt.Println("\n=== Executing KRM Function ===")
	functionReq := &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
			ConfigMap: map[string]interface{}{
				"namespace": "production",
			},
		},
		Resources: []KRMResource{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Metadata: map[string]interface{}{
					"name": "test-config",
				},
				Data: map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	functionResp, err := client.RunFunction(ctx, functionReq)
	if err != nil {
		return fmt.Errorf("failed to run function: %w", err)
	}
	fmt.Printf("Function executed: %d output resources, %d results\n",
		len(functionResp.Resources), len(functionResp.Results))

	// Example 11: Health check
	fmt.Println("\n=== Health Check ===")
	health, err := client.Health(ctx)
	if err != nil {
		return fmt.Errorf("failed to get health status: %w", err)
	}
	fmt.Printf("Porch health: %s\n", health.Status)

	fmt.Println("\n=== Example completed successfully! ===")
	return nil
}

// ExampleAdvancedUsage demonstrates advanced Porch client features
func ExampleAdvancedUsage() error {
	logger := zap.New(zap.UseDevMode(true))

	// Create client with custom configuration
	config := &Config{
		Endpoint: "https://porch.example.com:8443",
		AuthConfig: &AuthConfig{
			Type:  "bearer",
			Token: "your-jwt-token-here",
		},
		TLSConfig: &TLSConfig{
			InsecureSkipVerify: false,
			CAFile:             "/etc/ssl/certs/ca.pem",
		},
		PorchConfig: &PorchConfig{
			DefaultNamespace:  "porch-system",
			DefaultRepository: "blueprints",
			CircuitBreaker: &CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 3,
				Timeout:          60 * time.Second,
				HalfOpenMaxCalls: 2,
			},
			RateLimit: &RateLimitConfig{
				Enabled:           true,
				RequestsPerSecond: 5.0,
				Burst:             10,
			},
			FunctionExecution: &FunctionExecutionConfig{
				DefaultTimeout: 120 * time.Second,
				MaxConcurrency: 3,
				ResourceLimits: map[string]string{
					"cpu":    "200m",
					"memory": "256Mi",
				},
			},
		},
	}

	opts := ClientOptions{
		Config:         config,
		Logger:         logger,
		MetricsEnabled: true,
		CacheEnabled:   true,
		CacheSize:      500,
		CacheTTL:       10 * time.Minute,
	}

	client, err := NewClient(opts)
	if err != nil {
		return fmt.Errorf("failed to create advanced Porch client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Advanced example: Complex package with multiple functions
	complexPackage := &PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: "complex-package-v1",
			Labels: map[string]string{
				"app.kubernetes.io/name":      "complex-package",
				"app.kubernetes.io/version":   "1.0.0",
				"app.kubernetes.io/component": "network-function",
			},
		},
		Spec: PackageRevisionSpec{
			PackageName: "complex-package",
			Repository:  "blueprints",
			Revision:    "v1.0.0",
			Lifecycle:   PackageRevisionLifecycleDraft,
			Functions: []FunctionConfig{
				{
					Image:      "gcr.io/kpt-fn/set-namespace:v0.4.1",
					ConfigPath: "namespace-config.yaml",
				},
				{
					Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
					ConfigMap: map[string]interface{}{
						"replacements": []interface{}{
							map[string]interface{}{
								"source": map[string]interface{}{
									"objref": map[string]interface{}{
										"kind": "ConfigMap",
										"name": "app-config",
									},
									"fieldref": "data.app-name",
								},
								"targets": []interface{}{
									map[string]interface{}{
										"select": map[string]interface{}{
											"kind": "Deployment",
										},
										"fieldPaths": []string{"metadata.name"},
									},
								},
							},
						},
					},
				},
			},
			Resources: []KRMResource{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Metadata: map[string]interface{}{
						"name":      "app-config",
						"namespace": "default",
					},
					Data: map[string]interface{}{
						"app-name": "my-application",
						"version":  "1.0.0",
					},
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Metadata: map[string]interface{}{
						"name":      "placeholder-deployment",
						"namespace": "default",
					},
					Spec: map[string]interface{}{
						"replicas": 3,
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"app": "my-application",
							},
						},
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"app": "my-application",
								},
							},
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "app",
										"image": "nginx:1.21",
										"ports": []interface{}{
											map[string]interface{}{
												"containerPort": 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the complex package
	fmt.Println("Creating complex package with functions...")
	created, err := client.CreatePackageRevision(ctx, complexPackage)
	if err != nil {
		return fmt.Errorf("failed to create complex package: %w", err)
	}
	fmt.Printf("Created complex package: %s\n", created.Name)

	// Render the package to execute all functions
	fmt.Println("Rendering complex package...")
	renderResult, err := client.RenderPackage(ctx, "complex-package", "v1.0.0")
	if err != nil {
		return fmt.Errorf("failed to render complex package: %w", err)
	}

	fmt.Printf("Rendered package successfully:\n")
	fmt.Printf("  - Input resources: %d\n", len(complexPackage.Spec.Resources))
	fmt.Printf("  - Output resources: %d\n", len(renderResult.Resources))
	fmt.Printf("  - Function results: %d\n", len(renderResult.Results))

	// Display function results
	for i, result := range renderResult.Results {
		fmt.Printf("  Result %d: %s (severity: %s)\n", i+1, result.Message, result.Severity)
	}

	fmt.Println("Advanced example completed successfully!")
	return nil
}

// ExampleWorkflowUsage demonstrates package workflow operations
func ExampleWorkflowUsage() error {
	logger := zap.New(zap.UseDevMode(true))
	config := DefaultPorchConfig()

	client, err := NewClient(ClientOptions{
		Config: config,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	packageName := "workflow-example"
	revision := "v1.0.0"

	// Create a package in Draft state
	pkg := &PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", packageName, revision),
		},
		Spec: PackageRevisionSpec{
			PackageName: packageName,
			Repository:  "examples",
			Revision:    revision,
			Lifecycle:   PackageRevisionLifecycleDraft,
			Resources: []KRMResource{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Metadata: map[string]interface{}{
						"name":      "example-service",
						"namespace": "default",
					},
					Spec: map[string]interface{}{
						"selector": map[string]interface{}{
							"app": "example",
						},
						"ports": []interface{}{
							map[string]interface{}{
								"port":       80,
								"targetPort": 8080,
							},
						},
					},
				},
			},
		},
	}

	// Step 1: Create package in Draft
	fmt.Println("Step 1: Creating package in Draft state...")
	_, err = client.CreatePackageRevision(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	// Step 2: Validate the package
	fmt.Println("Step 2: Validating package...")
	validation, err := client.ValidatePackage(ctx, packageName, revision)
	if err != nil {
		return fmt.Errorf("failed to validate package: %w", err)
	}

	if !validation.Valid {
		fmt.Printf("Package validation failed with %d errors:\n", len(validation.Errors))
		for _, e := range validation.Errors {
			fmt.Printf("  - %s: %s\n", e.Path, e.Message)
		}
		return fmt.Errorf("package validation failed")
	}
	fmt.Println("Package validation passed!")

	// Step 3: Propose package for review
	fmt.Println("Step 3: Proposing package for review...")
	err = client.ProposePackageRevision(ctx, packageName, revision)
	if err != nil {
		return fmt.Errorf("failed to propose package: %w", err)
	}
	fmt.Println("Package proposed successfully!")

	// Step 4: Simulate review and approval
	fmt.Println("Step 4: Approving package...")
	err = client.ApprovePackageRevision(ctx, packageName, revision)
	if err != nil {
		return fmt.Errorf("failed to approve package: %w", err)
	}
	fmt.Println("Package approved and published!")

	// Step 5: Verify final state
	fmt.Println("Step 5: Verifying final package state...")
	finalPkg, err := client.GetPackageRevision(ctx, packageName, revision)
	if err != nil {
		return fmt.Errorf("failed to get final package: %w", err)
	}

	fmt.Printf("Final package lifecycle: %s\n", finalPkg.Spec.Lifecycle)
	if finalPkg.Status.PublishTime != nil {
		fmt.Printf("Published at: %s\n", finalPkg.Status.PublishTime.Format(time.RFC3339))
	}

	fmt.Println("Workflow example completed successfully!")
	return nil
}
