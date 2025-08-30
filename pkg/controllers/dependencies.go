package controllers

import (
	"net/http"

	"k8s.io/client-go/tools/record"

	"github.com/nephio-project/nephoran-intent-operator/pkg/git"
	"github.com/nephio-project/nephoran-intent-operator/pkg/nephio"
	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"
)

// ConcreteDependencies is a concrete implementation of the Dependencies interface.

type ConcreteDependencies struct {
	gitClient git.ClientInterface

	llmClient shared.ClientInterface

	packageGen *nephio.PackageGenerator

	httpClient *http.Client

	eventRecorder record.EventRecorder
}

// NewConcreteDependencies creates a new instance of ConcreteDependencies.

func NewConcreteDependencies(

	gitClient git.ClientInterface,

	llmClient shared.ClientInterface,

	packageGen *nephio.PackageGenerator,

	httpClient *http.Client,

	eventRecorder record.EventRecorder,

) *ConcreteDependencies {

	return &ConcreteDependencies{

		gitClient: gitClient,

		llmClient: llmClient,

		packageGen: packageGen,

		httpClient: httpClient,

		eventRecorder: eventRecorder,
	}

}

// GetGitClient returns the Git client.

func (d *ConcreteDependencies) GetGitClient() git.ClientInterface {

	return d.gitClient

}

// GetLLMClient returns the LLM client.

func (d *ConcreteDependencies) GetLLMClient() shared.ClientInterface {

	return d.llmClient

}

// GetPackageGenerator returns the Nephio package generator.

func (d *ConcreteDependencies) GetPackageGenerator() *nephio.PackageGenerator {

	return d.packageGen

}

// GetHTTPClient returns the HTTP client.

func (d *ConcreteDependencies) GetHTTPClient() *http.Client {

	return d.httpClient

}

// GetEventRecorder returns the event recorder.

func (d *ConcreteDependencies) GetEventRecorder() record.EventRecorder {

	return d.eventRecorder

}
