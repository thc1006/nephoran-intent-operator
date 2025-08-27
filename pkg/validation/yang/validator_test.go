package yang

import (
	"context"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidatorMetrics(t *testing.T) {
	metrics := newValidatorMetrics()
	if metrics == nil {
		t.Fatal("newValidatorMetrics returned nil")
	}

	if metrics.ValidationsTotal == nil {
		t.Error("ValidationsTotal metric is nil")
	}

	if metrics.ValidationDuration == nil {
		t.Error("ValidationDuration metric is nil")
	}

	if metrics.ValidationErrors == nil {
		t.Error("ValidationErrors metric is nil")
	}
}

func TestValidatePackageRevision(t *testing.T) {
	validator := &yangValidator{
		metrics: newValidatorMetrics(),
	}

	ctx := context.Background()
	pkg := &porch.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind: "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-package",
		},
		Spec: porch.PackageRevisionSpec{
			Resources: []interface{}{
				map[string]interface{}{
					"kind": "ConfigMap",
					"data": map[string]interface{}{
						"config": map[string]interface{}{
							"test-key": "test-value",
						},
					},
				},
			},
		},
	}

	result, err := validator.ValidatePackageRevision(ctx, pkg)
	if err != nil {
		t.Fatalf("ValidatePackageRevision failed: %v", err)
	}

	if result == nil {
		t.Fatal("ValidatePackageRevision returned nil result")
	}

	if result.ModelName != "test-package" {
		t.Errorf("Expected ModelName 'test-package', got '%s'", result.ModelName)
	}

	if result.ValidationTime.IsZero() {
		t.Error("ValidationTime is zero")
	}

	if !result.Valid {
		t.Error("Expected valid result for properly formatted package")
	}
}

func TestValidatePackageRevisionInvalidResource(t *testing.T) {
	validator := &yangValidator{
		metrics: newValidatorMetrics(),
	}

	ctx := context.Background()
	pkg := &porch.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind: "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-package",
		},
		Spec: porch.PackageRevisionSpec{
			Resources: []interface{}{
				"invalid-resource-type",
			},
		},
	}

	result, err := validator.ValidatePackageRevision(ctx, pkg)
	if err != nil {
		t.Fatalf("ValidatePackageRevision failed: %v", err)
	}

	if result == nil {
		t.Fatal("ValidatePackageRevision returned nil result")
	}

	if result.Valid {
		t.Error("Expected invalid result for improperly formatted package")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid resource")
	}
}
