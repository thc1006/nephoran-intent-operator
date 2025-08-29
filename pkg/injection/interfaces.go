package injection

import (
	"net/http"

	"github.com/nephio-project/nephoran-intent-operator/pkg/git"
	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"
	"github.com/nephio-project/nephoran-intent-operator/pkg/nephio"
	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"
	"github.com/nephio-project/nephoran-intent-operator/pkg/telecom"

	"k8s.io/client-go/tools/record"
)

// Dependencies defines the interface for dependency injection.

// This matches the interface expected by the controllers.

type Dependencies interface {
	GetGitClient() git.ClientInterface

	GetLLMClient() shared.ClientInterface

	GetPackageGenerator() *nephio.PackageGenerator

	GetHTTPClient() *http.Client

	GetEventRecorder() record.EventRecorder

	GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase

	GetMetricsCollector() *monitoring.MetricsCollector
}

// Ensure Container implements Dependencies interface.

var _ Dependencies = (*Container)(nil)
