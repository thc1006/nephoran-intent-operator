package controllers

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
)

// MockGitClient is a mock implementation of the GitClient for testing.
type MockGitClient struct {
	CommitAndPushFunc func(files map[string]string, message string) error
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) error {
	if m.CommitAndPushFunc != nil {
		return m.CommitAndPushFunc(files, message)
	}
	return nil
}

func (m *MockGitClient) InitRepo() error {
	return nil
}
