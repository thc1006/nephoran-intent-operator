package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// A1-EI (Enrichment Information) Interface Implementation.

// This implements the O-RAN A1-EI specification for enrichment information service.

// A1EIInterface defines the A1-EI interface operations.

type A1EIInterface interface {

	// EI Type Management.

	CreateEIType(ctx context.Context, eiType *EnrichmentInfoType) error

	GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error)

	ListEITypes(ctx context.Context) ([]*EnrichmentInfoType, error)

	DeleteEIType(ctx context.Context, eiTypeID string) error

	// EI Job Management.

	CreateEIJob(ctx context.Context, eiTypeID string, eiJob *EnrichmentInfoJob) error

	GetEIJob(ctx context.Context, eiTypeID, eiJobID string) (*EnrichmentInfoJob, error)

	ListEIJobs(ctx context.Context, eiTypeID string) ([]*EnrichmentInfoJob, error)

	DeleteEIJob(ctx context.Context, eiTypeID, eiJobID string) error

	// EI Job Status.

	GetEIJobStatus(ctx context.Context, eiTypeID, eiJobID string) (*EIJobStatus, error)
}

// EIJobStatus represents the status of an enrichment information job.

type EIJobStatus struct {
	OperationalState string `json:"operational_state"` // ENABLED, DISABLED

	DataDeliveryState string `json:"data_delivery_state"` // OK, SUSPENDED

	LastUpdated time.Time `json:"last_updated"`

	AdditionalInfo string `json:"additional_info,omitempty"`
}

// EIProducerInfo represents information about an EI data producer.

type EIProducerInfo struct {
	ProducerID string `json:"producer_id"`

	ProducerName string `json:"producer_name"`

	SupportedEITypes []string `json:"supported_ei_types"`

	ProducerCallbackURL string `json:"producer_callback_url"`

	ProducerSupervisionURL string `json:"producer_supervision_url"`
}

// A1EIAdaptor implements the A1-EI interface.

type A1EIAdaptor struct {
	httpClient *http.Client

	ricURL string

	apiVersion string
}

// NewA1EIAdaptor creates a new A1-EI adaptor.

func NewA1EIAdaptor(ricURL, apiVersion string) *A1EIAdaptor {

	return &A1EIAdaptor{

		httpClient: &http.Client{Timeout: 30 * time.Second},

		ricURL: ricURL,

		apiVersion: apiVersion,
	}

}

// CreateEIType creates a new enrichment information type.

func (ei *A1EIAdaptor) CreateEIType(ctx context.Context, eiType *EnrichmentInfoType) error {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s", ei.ricURL, eiType.EiTypeID)

	body, err := json.Marshal(eiType)

	if err != nil {

		return fmt.Errorf("failed to marshal EI type: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusCreated:

		return nil

	case http.StatusOK:

		return nil // Updated existing type

	case http.StatusBadRequest:

		return fmt.Errorf("invalid EI type definition")

	case http.StatusConflict:

		return fmt.Errorf("EI type already exists with different definition")

	default:

		return fmt.Errorf("failed to create EI type: status=%d", resp.StatusCode)

	}

}

// GetEIType retrieves an enrichment information type.

func (ei *A1EIAdaptor) GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error) {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s", ei.ricURL, eiTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:

		var eiType EnrichmentInfoType

		if err := json.NewDecoder(resp.Body).Decode(&eiType); err != nil {

			return nil, fmt.Errorf("failed to decode response: %w", err)

		}

		return &eiType, nil

	case http.StatusNotFound:

		return nil, fmt.Errorf("EI type %s not found", eiTypeID)

	default:

		return nil, fmt.Errorf("failed to get EI type: status=%d", resp.StatusCode)

	}

}

// ListEITypes lists all enrichment information types.

func (ei *A1EIAdaptor) ListEITypes(ctx context.Context) ([]*EnrichmentInfoType, error) {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes", ei.ricURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to list EI types: status=%d", resp.StatusCode)

	}

	var eiTypeIDs []string

	if err := json.NewDecoder(resp.Body).Decode(&eiTypeIDs); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	// Fetch details for each EI type.

	var eiTypes []*EnrichmentInfoType

	for _, id := range eiTypeIDs {

		eiType, err := ei.GetEIType(ctx, id)

		if err != nil {

			return nil, fmt.Errorf("failed to get EI type %s: %w", id, err)

		}

		eiTypes = append(eiTypes, eiType)

	}

	return eiTypes, nil

}

// DeleteEIType deletes an enrichment information type.

func (ei *A1EIAdaptor) DeleteEIType(ctx context.Context, eiTypeID string) error {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s", ei.ricURL, eiTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusNoContent:

		return nil

	case http.StatusNotFound:

		return fmt.Errorf("EI type %s not found", eiTypeID)

	case http.StatusBadRequest:

		return fmt.Errorf("cannot delete EI type with active jobs")

	default:

		return fmt.Errorf("failed to delete EI type: status=%d", resp.StatusCode)

	}

}

// CreateEIJob creates a new enrichment information job.

func (ei *A1EIAdaptor) CreateEIJob(ctx context.Context, eiTypeID string, eiJob *EnrichmentInfoJob) error {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s/eijobs/%s", ei.ricURL, eiTypeID, eiJob.EiJobID)

	body, err := json.Marshal(eiJob)

	if err != nil {

		return fmt.Errorf("failed to marshal EI job: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusCreated:

		return nil

	case http.StatusOK:

		return nil // Updated existing job

	case http.StatusBadRequest:

		return fmt.Errorf("invalid EI job definition")

	case http.StatusNotFound:

		return fmt.Errorf("EI type %s not found", eiTypeID)

	case http.StatusConflict:

		return fmt.Errorf("EI job conflicts with existing job")

	default:

		return fmt.Errorf("failed to create EI job: status=%d", resp.StatusCode)

	}

}

// GetEIJob retrieves an enrichment information job.

func (ei *A1EIAdaptor) GetEIJob(ctx context.Context, eiTypeID, eiJobID string) (*EnrichmentInfoJob, error) {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s/eijobs/%s", ei.ricURL, eiTypeID, eiJobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:

		var eiJob EnrichmentInfoJob

		if err := json.NewDecoder(resp.Body).Decode(&eiJob); err != nil {

			return nil, fmt.Errorf("failed to decode response: %w", err)

		}

		return &eiJob, nil

	case http.StatusNotFound:

		return nil, fmt.Errorf("EI job %s not found for type %s", eiJobID, eiTypeID)

	default:

		return nil, fmt.Errorf("failed to get EI job: status=%d", resp.StatusCode)

	}

}

// ListEIJobs lists all enrichment information jobs for a given type.

func (ei *A1EIAdaptor) ListEIJobs(ctx context.Context, eiTypeID string) ([]*EnrichmentInfoJob, error) {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s/eijobs", ei.ricURL, eiTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to list EI jobs: status=%d", resp.StatusCode)

	}

	var eiJobIDs []string

	if err := json.NewDecoder(resp.Body).Decode(&eiJobIDs); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	// Fetch details for each EI job.

	var eiJobs []*EnrichmentInfoJob

	for _, id := range eiJobIDs {

		eiJob, err := ei.GetEIJob(ctx, eiTypeID, id)

		if err != nil {

			return nil, fmt.Errorf("failed to get EI job %s: %w", id, err)

		}

		eiJobs = append(eiJobs, eiJob)

	}

	return eiJobs, nil

}

// DeleteEIJob deletes an enrichment information job.

func (ei *A1EIAdaptor) DeleteEIJob(ctx context.Context, eiTypeID, eiJobID string) error {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s/eijobs/%s", ei.ricURL, eiTypeID, eiJobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusNoContent:

		return nil

	case http.StatusNotFound:

		return fmt.Errorf("EI job %s not found for type %s", eiJobID, eiTypeID)

	default:

		return fmt.Errorf("failed to delete EI job: status=%d", resp.StatusCode)

	}

}

// GetEIJobStatus retrieves the status of an enrichment information job.

func (ei *A1EIAdaptor) GetEIJobStatus(ctx context.Context, eiTypeID, eiJobID string) (*EIJobStatus, error) {

	url := fmt.Sprintf("%s/A1-EI/v1/eitypes/%s/eijobs/%s/status", ei.ricURL, eiTypeID, eiJobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:

		var status EIJobStatus

		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {

			return nil, fmt.Errorf("failed to decode status response: %w", err)

		}

		return &status, nil

	case http.StatusNotFound:

		return nil, fmt.Errorf("EI job %s not found for type %s", eiJobID, eiTypeID)

	default:

		return nil, fmt.Errorf("failed to get EI job status: status=%d", resp.StatusCode)

	}

}

// RegisterEIProducer registers an enrichment information data producer.

func (ei *A1EIAdaptor) RegisterEIProducer(ctx context.Context, producer *EIProducerInfo) error {

	url := fmt.Sprintf("%s/A1-EI/v1/eiproducers/%s", ei.ricURL, producer.ProducerID)

	body, err := json.Marshal(producer)

	if err != nil {

		return fmt.Errorf("failed to marshal EI producer: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := ei.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusCreated:

		return nil

	case http.StatusOK:

		return nil // Updated existing producer

	case http.StatusBadRequest:

		return fmt.Errorf("invalid EI producer definition")

	default:

		return fmt.Errorf("failed to register EI producer: status=%d", resp.StatusCode)

	}

}
