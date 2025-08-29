// Package a1 implements comprehensive HTTP handlers for O-RAN A1 Policy Management Service.

// This module provides handlers for A1-P, A1-C, and A1-EI interfaces according to O-RAN.WG2.A1AP-v03.01.

package a1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
)

// A1Handlers implements HTTP handlers for all A1 interfaces.

type A1Handlers struct {
	service A1Service

	validator A1Validator

	storage A1Storage

	metrics A1Metrics

	logger *logging.StructuredLogger

	config *A1ServerConfig

	circuitBreaker map[string]*CircuitBreaker
}

// NewA1Handlers creates a new A1 handlers instance.

func NewA1Handlers(

	service A1Service,

	validator A1Validator,

	storage A1Storage,

	metrics A1Metrics,

	logger *logging.StructuredLogger,

	config *A1ServerConfig,

) *A1Handlers {

	if logger == nil {

		logger = logging.NewStructuredLogger(logging.DefaultConfig("a1-handlers", "1.0.0", "development"))

	}

	if config == nil {

		config = DefaultA1ServerConfig()

	}

	return &A1Handlers{

		service: service,

		validator: validator,

		storage: storage,

		metrics: metrics,

		logger: logger.WithComponent("a1-handlers"),

		config: config,

		circuitBreaker: make(map[string]*CircuitBreaker),
	}

}

// A1-P Policy Interface Handlers.

// HandleGetPolicyTypes handles GET /A1-P/v2/policytypes.

func (h *A1Handlers) HandleGetPolicyTypes(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	h.logger.InfoWithContext("Getting policy types",

		"request_id", requestID,

		"method", r.Method,

		"path", r.URL.Path,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "GetPolicyTypes", duration)

	}()

	// Get policy types from service.

	policyTypeIDs, err := h.service.GetPolicyTypes(ctx)

	if err != nil {

		h.handleError(w, r, WrapError(err, "Failed to retrieve policy types"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyTypes", http.StatusInternalServerError)

		return

	}

	// Return policy type IDs as JSON array.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(policyTypeIDs); err != nil {

		h.logger.ErrorWithContext("Failed to encode policy types response", err, "request_id", requestID)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyTypes", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyTypes", http.StatusOK)

	h.logger.InfoWithContext("Policy types retrieved successfully",

		"request_id", requestID,

		"count", len(policyTypeIDs),
	)

}

// HandleGetPolicyType handles GET /A1-P/v2/policytypes/{policy_type_id}.

func (h *A1Handlers) HandleGetPolicyType(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract policy type ID from URL.

	vars := mux.Vars(r)

	policyTypeIDStr := vars["policy_type_id"]

	policyTypeID, err := strconv.Atoi(policyTypeIDStr)

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError("Invalid policy type ID format"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyType", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Getting policy type",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "GetPolicyType", duration)

	}()

	// Get policy type from service.

	policyType, err := h.service.GetPolicyType(ctx, policyTypeID)

	if err != nil {

		h.handleError(w, r, NewPolicyTypeNotFoundError(policyTypeID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyType", http.StatusNotFound)

		return

	}

	// Return policy type as JSON.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(policyType); err != nil {

		h.logger.ErrorWithContext("Failed to encode policy type response", err,

			"request_id", requestID,

			"policy_type_id", policyTypeID,
		)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyType", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyType", http.StatusOK)

	h.logger.InfoWithContext("Policy type retrieved successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

}

// HandleCreatePolicyType handles PUT /A1-P/v2/policytypes/{policy_type_id}.

func (h *A1Handlers) HandleCreatePolicyType(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract policy type ID from URL.

	vars := mux.Vars(r)

	policyTypeIDStr := vars["policy_type_id"]

	policyTypeID, err := strconv.Atoi(policyTypeIDStr)

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError("Invalid policy type ID format"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Creating policy type",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "CreatePolicyType", duration)

	}()

	// Validate content type.

	if !h.isContentTypeJSON(r) {

		h.handleError(w, r, NewUnsupportedMediaTypeError(

			r.Header.Get("Content-Type"),

			[]string{ContentTypeJSON},
		))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusUnsupportedMediaType)

		return

	}

	// Read and parse request body.

	body, err := h.readRequestBody(r, 1024*1024) // 1MB limit

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Failed to read request body: %v", err)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusBadRequest)

		return

	}

	var policyType PolicyType

	if err := json.Unmarshal(body, &policyType); err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Invalid JSON: %v", err)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusBadRequest)

		return

	}

	// Set policy type ID from URL.

	policyType.PolicyTypeID = policyTypeID

	policyType.CreatedAt = time.Now()

	policyType.ModifiedAt = time.Now()

	// Validate policy type.

	if validationResult := h.validator.ValidatePolicyType(&policyType); !validationResult.Valid {

		h.handleError(w, r, NewPolicyTypeInvalidSchemaError(policyTypeID,

			fmt.Errorf("validation failed: %v", validationResult.Errors)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusBadRequest)

		return

	}

	// Check if policy type already exists.

	if existingType, err := h.service.GetPolicyType(ctx, policyTypeID); err == nil && existingType != nil {

		h.handleError(w, r, NewPolicyTypeAlreadyExistsError(policyTypeID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusConflict)

		return

	}

	// Create policy type.

	if err := h.service.CreatePolicyType(ctx, policyTypeID, &policyType); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to create policy type"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusInternalServerError)

		return

	}

	// Return 201 Created with policy type.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.Header().Set("Location", fmt.Sprintf("/A1-P/v2/policytypes/%d", policyTypeID))

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(policyType); err != nil {

		h.logger.ErrorWithContext("Failed to encode create policy type response", err,

			"request_id", requestID,

			"policy_type_id", policyTypeID,
		)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyType", http.StatusCreated)

	h.logger.InfoWithContext("Policy type created successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

}

// HandleDeletePolicyType handles DELETE /A1-P/v2/policytypes/{policy_type_id}.

func (h *A1Handlers) HandleDeletePolicyType(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract policy type ID from URL.

	vars := mux.Vars(r)

	policyTypeIDStr := vars["policy_type_id"]

	policyTypeID, err := strconv.Atoi(policyTypeIDStr)

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError("Invalid policy type ID format"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyType", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Deleting policy type",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "DeletePolicyType", duration)

	}()

	// Check if policy type exists.

	if _, err := h.service.GetPolicyType(ctx, policyTypeID); err != nil {

		h.handleError(w, r, NewPolicyTypeNotFoundError(policyTypeID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyType", http.StatusNotFound)

		return

	}

	// Check if there are active policy instances.

	instances, err := h.service.GetPolicyInstances(ctx, policyTypeID)

	if err == nil && len(instances) > 0 {

		h.handleError(w, r, NewInvalidRequestError("Cannot delete policy type with active instances"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyType", http.StatusBadRequest)

		return

	}

	// Delete policy type.

	if err := h.service.DeletePolicyType(ctx, policyTypeID); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to delete policy type"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyType", http.StatusInternalServerError)

		return

	}

	// Return 204 No Content.

	w.WriteHeader(http.StatusNoContent)

	h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyType", http.StatusNoContent)

	h.logger.InfoWithContext("Policy type deleted successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

}

// HandleGetPolicyInstances handles GET /A1-P/v2/policytypes/{policy_type_id}/policies.

func (h *A1Handlers) HandleGetPolicyInstances(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract policy type ID from URL.

	vars := mux.Vars(r)

	policyTypeIDStr := vars["policy_type_id"]

	policyTypeID, err := strconv.Atoi(policyTypeIDStr)

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError("Invalid policy type ID format"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstances", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Getting policy instances",

		"request_id", requestID,

		"policy_type_id", policyTypeID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "GetPolicyInstances", duration)

	}()

	// Check if policy type exists.

	if _, err := h.service.GetPolicyType(ctx, policyTypeID); err != nil {

		h.handleError(w, r, NewPolicyTypeNotFoundError(policyTypeID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstances", http.StatusNotFound)

		return

	}

	// Get policy instances from service.

	policyIDs, err := h.service.GetPolicyInstances(ctx, policyTypeID)

	if err != nil {

		h.handleError(w, r, WrapError(err, "Failed to retrieve policy instances"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstances", http.StatusInternalServerError)

		return

	}

	// Return policy instance IDs as JSON array.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(policyIDs); err != nil {

		h.logger.ErrorWithContext("Failed to encode policy instances response", err,

			"request_id", requestID,

			"policy_type_id", policyTypeID,
		)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstances", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstances", http.StatusOK)

	h.metrics.RecordPolicyCount(policyTypeID, len(policyIDs))

	h.logger.InfoWithContext("Policy instances retrieved successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"count", len(policyIDs),
	)

}

// HandleGetPolicyInstance handles GET /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}.

func (h *A1Handlers) HandleGetPolicyInstance(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract parameters from URL.

	vars := mux.Vars(r)

	policyTypeID, policyID, valid := h.extractPolicyParams(vars, w, r)

	if !valid {

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstance", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Getting policy instance",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "GetPolicyInstance", duration)

	}()

	// Get policy instance from service.

	policyInstance, err := h.service.GetPolicyInstance(ctx, policyTypeID, policyID)

	if err != nil {

		h.handleError(w, r, NewPolicyInstanceNotFoundError(policyTypeID, policyID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstance", http.StatusNotFound)

		return

	}

	// Return policy instance data (not the full object).

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(policyInstance.PolicyData); err != nil {

		h.logger.ErrorWithContext("Failed to encode policy instance response", err,

			"request_id", requestID,

			"policy_type_id", policyTypeID,

			"policy_id", policyID,
		)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstance", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyInstance", http.StatusOK)

	h.logger.InfoWithContext("Policy instance retrieved successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

}

// HandleCreatePolicyInstance handles PUT /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}.

func (h *A1Handlers) HandleCreatePolicyInstance(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract parameters from URL.

	vars := mux.Vars(r)

	policyTypeID, policyID, valid := h.extractPolicyParams(vars, w, r)

	if !valid {

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Creating policy instance",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "CreatePolicyInstance", duration)

	}()

	// Validate content type.

	if !h.isContentTypeJSON(r) {

		h.handleError(w, r, NewUnsupportedMediaTypeError(

			r.Header.Get("Content-Type"),

			[]string{ContentTypeJSON},
		))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusUnsupportedMediaType)

		return

	}

	// Check if policy type exists.

	if _, err := h.service.GetPolicyType(ctx, policyTypeID); err != nil {

		h.handleError(w, r, NewPolicyTypeNotFoundError(policyTypeID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusNotFound)

		return

	}

	// Read and parse request body (policy data).

	body, err := h.readRequestBody(r, 1024*1024) // 1MB limit

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Failed to read request body: %v", err)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusBadRequest)

		return

	}

	var policyData map[string]interface{}

	if err := json.Unmarshal(body, &policyData); err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Invalid JSON: %v", err)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusBadRequest)

		return

	}

	// Create policy instance.

	policyInstance := &PolicyInstance{

		PolicyID: policyID,

		PolicyTypeID: policyTypeID,

		PolicyData: policyData,

		CreatedAt: time.Now(),

		ModifiedAt: time.Now(),
	}

	// Validate policy instance.

	if validationResult := h.validator.ValidatePolicyInstance(policyTypeID, policyInstance); !validationResult.Valid {

		h.handleError(w, r, NewPolicyValidationFailedError(policyTypeID, policyID,

			fmt.Errorf("validation failed: %v", validationResult.Errors)))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusBadRequest)

		return

	}

	// Check if policy instance already exists (for conflict detection).

	isUpdate := false

	if existingInstance, err := h.service.GetPolicyInstance(ctx, policyTypeID, policyID); err == nil && existingInstance != nil {

		isUpdate = true

		policyInstance.CreatedAt = existingInstance.CreatedAt // Preserve creation time

	}

	// Create or update policy instance.

	if err := h.service.CreatePolicyInstance(ctx, policyTypeID, policyID, policyInstance); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to create policy instance"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", http.StatusInternalServerError)

		return

	}

	// Return appropriate status code.

	statusCode := http.StatusCreated

	if isUpdate {

		statusCode = http.StatusOK

	}

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.Header().Set("Location", fmt.Sprintf("/A1-P/v2/policytypes/%d/policies/%s", policyTypeID, policyID))

	w.WriteHeader(statusCode)

	// Return empty response body per O-RAN spec.

	h.metrics.IncrementRequestCount(A1PolicyInterface, "CreatePolicyInstance", statusCode)

	h.logger.InfoWithContext("Policy instance created successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,

		"is_update", isUpdate,
	)

}

// HandleUpdatePolicyInstance handles PUT (same as create per O-RAN spec).

func (h *A1Handlers) HandleUpdatePolicyInstance(w http.ResponseWriter, r *http.Request) {

	h.HandleCreatePolicyInstance(w, r)

}

// HandleDeletePolicyInstance handles DELETE /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}.

func (h *A1Handlers) HandleDeletePolicyInstance(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract parameters from URL.

	vars := mux.Vars(r)

	policyTypeID, policyID, valid := h.extractPolicyParams(vars, w, r)

	if !valid {

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyInstance", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Deleting policy instance",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "DeletePolicyInstance", duration)

	}()

	// Check if policy instance exists.

	if _, err := h.service.GetPolicyInstance(ctx, policyTypeID, policyID); err != nil {

		h.handleError(w, r, NewPolicyInstanceNotFoundError(policyTypeID, policyID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyInstance", http.StatusNotFound)

		return

	}

	// Delete policy instance.

	if err := h.service.DeletePolicyInstance(ctx, policyTypeID, policyID); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to delete policy instance"))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyInstance", http.StatusInternalServerError)

		return

	}

	// Return 202 Accepted (deletion may be asynchronous per O-RAN spec).

	w.WriteHeader(http.StatusAccepted)

	h.metrics.IncrementRequestCount(A1PolicyInterface, "DeletePolicyInstance", http.StatusAccepted)

	h.logger.InfoWithContext("Policy instance deletion initiated",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

}

// HandleGetPolicyStatus handles GET /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}/status.

func (h *A1Handlers) HandleGetPolicyStatus(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract parameters from URL.

	vars := mux.Vars(r)

	policyTypeID, policyID, valid := h.extractPolicyParams(vars, w, r)

	if !valid {

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyStatus", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Getting policy status",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1PolicyInterface, "GetPolicyStatus", duration)

	}()

	// Get policy status from service.

	policyStatus, err := h.service.GetPolicyStatus(ctx, policyTypeID, policyID)

	if err != nil {

		h.handleError(w, r, NewPolicyInstanceNotFoundError(policyTypeID, policyID))

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyStatus", http.StatusNotFound)

		return

	}

	// Return policy status as JSON.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(policyStatus); err != nil {

		h.logger.ErrorWithContext("Failed to encode policy status response", err,

			"request_id", requestID,

			"policy_type_id", policyTypeID,

			"policy_id", policyID,
		)

		h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyStatus", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1PolicyInterface, "GetPolicyStatus", http.StatusOK)

	h.logger.InfoWithContext("Policy status retrieved successfully",

		"request_id", requestID,

		"policy_type_id", policyTypeID,

		"policy_id", policyID,

		"status", policyStatus.EnforcementStatus,
	)

}

// A1-C Consumer Interface Handlers.

// HandleRegisterConsumer handles POST /A1-C/v1/consumers/{consumer_id}.

func (h *A1Handlers) HandleRegisterConsumer(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract consumer ID from URL.

	vars := mux.Vars(r)

	consumerID := vars["consumer_id"]

	if consumerID == "" {

		h.handleError(w, r, NewInvalidRequestError("Consumer ID is required"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Registering consumer",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1ConsumerInterface, "RegisterConsumer", duration)

	}()

	// Validate content type.

	if !h.isContentTypeJSON(r) {

		h.handleError(w, r, NewUnsupportedMediaTypeError(

			r.Header.Get("Content-Type"),

			[]string{ContentTypeJSON},
		))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusUnsupportedMediaType)

		return

	}

	// Read and parse request body.

	body, err := h.readRequestBody(r, 1024*1024) // 1MB limit

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Failed to read request body: %v", err)))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusBadRequest)

		return

	}

	var consumerInfo ConsumerInfo

	if err := json.Unmarshal(body, &consumerInfo); err != nil {

		h.handleError(w, r, NewInvalidRequestError(fmt.Sprintf("Invalid JSON: %v", err)))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusBadRequest)

		return

	}

	// Set consumer ID from URL.

	consumerInfo.ConsumerID = consumerID

	consumerInfo.CreatedAt = time.Now()

	consumerInfo.ModifiedAt = time.Now()

	// Validate consumer info.

	if validationResult := h.validator.ValidateConsumerInfo(&consumerInfo); !validationResult.Valid {

		h.handleError(w, r, NewConsumerInvalidCallbackError(consumerID, consumerInfo.CallbackURL,

			fmt.Errorf("validation failed: %v", validationResult.Errors)))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusBadRequest)

		return

	}

	// Check if consumer already exists.

	if _, err := h.service.GetConsumer(ctx, consumerID); err == nil {

		h.handleError(w, r, NewConsumerAlreadyExistsError(consumerID))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusConflict)

		return

	}

	// Register consumer.

	if err := h.service.RegisterConsumer(ctx, consumerID, &consumerInfo); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to register consumer"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusInternalServerError)

		return

	}

	// Return 201 Created.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.Header().Set("Location", fmt.Sprintf("/A1-C/v1/consumers/%s", consumerID))

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(consumerInfo); err != nil {

		h.logger.ErrorWithContext("Failed to encode register consumer response", err,

			"request_id", requestID,

			"consumer_id", consumerID,
		)

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1ConsumerInterface, "RegisterConsumer", http.StatusCreated)

	h.logger.InfoWithContext("Consumer registered successfully",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

}

// HandleUnregisterConsumer handles DELETE /A1-C/v1/consumers/{consumer_id}.

func (h *A1Handlers) HandleUnregisterConsumer(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract consumer ID from URL.

	vars := mux.Vars(r)

	consumerID := vars["consumer_id"]

	if consumerID == "" {

		h.handleError(w, r, NewInvalidRequestError("Consumer ID is required"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "UnregisterConsumer", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Unregistering consumer",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1ConsumerInterface, "UnregisterConsumer", duration)

	}()

	// Check if consumer exists.

	if _, err := h.service.GetConsumer(ctx, consumerID); err != nil {

		h.handleError(w, r, NewConsumerNotFoundError(consumerID))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "UnregisterConsumer", http.StatusNotFound)

		return

	}

	// Unregister consumer.

	if err := h.service.UnregisterConsumer(ctx, consumerID); err != nil {

		h.handleError(w, r, WrapError(err, "Failed to unregister consumer"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "UnregisterConsumer", http.StatusInternalServerError)

		return

	}

	// Return 204 No Content.

	w.WriteHeader(http.StatusNoContent)

	h.metrics.IncrementRequestCount(A1ConsumerInterface, "UnregisterConsumer", http.StatusNoContent)

	h.logger.InfoWithContext("Consumer unregistered successfully",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

}

// HandleGetConsumer handles GET /A1-C/v1/consumers/{consumer_id}.

func (h *A1Handlers) HandleGetConsumer(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	// Extract consumer ID from URL.

	vars := mux.Vars(r)

	consumerID := vars["consumer_id"]

	if consumerID == "" {

		h.handleError(w, r, NewInvalidRequestError("Consumer ID is required"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "GetConsumer", http.StatusBadRequest)

		return

	}

	h.logger.InfoWithContext("Getting consumer",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1ConsumerInterface, "GetConsumer", duration)

	}()

	// Get consumer from service.

	consumer, err := h.service.GetConsumer(ctx, consumerID)

	if err != nil {

		h.handleError(w, r, NewConsumerNotFoundError(consumerID))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "GetConsumer", http.StatusNotFound)

		return

	}

	// Return consumer as JSON.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(consumer); err != nil {

		h.logger.ErrorWithContext("Failed to encode get consumer response", err,

			"request_id", requestID,

			"consumer_id", consumerID,
		)

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "GetConsumer", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1ConsumerInterface, "GetConsumer", http.StatusOK)

	h.logger.InfoWithContext("Consumer retrieved successfully",

		"request_id", requestID,

		"consumer_id", consumerID,
	)

}

// HandleListConsumers handles GET /A1-C/v1/consumers.

func (h *A1Handlers) HandleListConsumers(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)

	defer cancel()

	startTime := time.Now()

	requestID := h.getRequestID(r)

	h.logger.InfoWithContext("Listing consumers",

		"request_id", requestID,
	)

	defer func() {

		duration := time.Since(startTime)

		h.metrics.RecordRequestDuration(A1ConsumerInterface, "ListConsumers", duration)

	}()

	// Get consumers from service.

	consumers, err := h.service.ListConsumers(ctx)

	if err != nil {

		h.handleError(w, r, WrapError(err, "Failed to list consumers"))

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "ListConsumers", http.StatusInternalServerError)

		return

	}

	// Return consumers as JSON array.

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(consumers); err != nil {

		h.logger.ErrorWithContext("Failed to encode list consumers response", err,

			"request_id", requestID,
		)

		h.metrics.IncrementRequestCount(A1ConsumerInterface, "ListConsumers", http.StatusInternalServerError)

		return

	}

	h.metrics.IncrementRequestCount(A1ConsumerInterface, "ListConsumers", http.StatusOK)

	h.metrics.RecordConsumerCount(len(consumers))

	h.logger.InfoWithContext("Consumers listed successfully",

		"request_id", requestID,

		"count", len(consumers),
	)

}

// Additional A1-EI handlers would be implemented similarly...

// For brevity, I'll add placeholder methods here.

// A1-EI Enrichment Information Interface Handlers.

// HandleGetEITypes handles GET /A1-EI/v1/eitypes.

func (h *A1Handlers) HandleGetEITypes(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to GetPolicyTypes.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleGetEIType handles GET /A1-EI/v1/eitypes/{ei_type_id}.

func (h *A1Handlers) HandleGetEIType(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to GetPolicyType.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleCreateEIType handles PUT /A1-EI/v1/eitypes/{ei_type_id}.

func (h *A1Handlers) HandleCreateEIType(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to CreatePolicyType.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleDeleteEIType handles DELETE /A1-EI/v1/eitypes/{ei_type_id}.

func (h *A1Handlers) HandleDeleteEIType(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to DeletePolicyType.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleGetEIJobs handles GET /A1-EI/v1/eijobs.

func (h *A1Handlers) HandleGetEIJobs(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to GetPolicyInstances.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleGetEIJob handles GET /A1-EI/v1/eijobs/{ei_job_id}.

func (h *A1Handlers) HandleGetEIJob(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to GetPolicyInstance.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleCreateEIJob handles PUT /A1-EI/v1/eijobs/{ei_job_id}.

func (h *A1Handlers) HandleCreateEIJob(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to CreatePolicyInstance.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleUpdateEIJob handles PUT (same as create).

func (h *A1Handlers) HandleUpdateEIJob(w http.ResponseWriter, r *http.Request) {

	h.HandleCreateEIJob(w, r)

}

// HandleDeleteEIJob handles DELETE /A1-EI/v1/eijobs/{ei_job_id}.

func (h *A1Handlers) HandleDeleteEIJob(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to DeletePolicyInstance.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// HandleGetEIJobStatus handles GET /A1-EI/v1/eijobs/{ei_job_id}/status.

func (h *A1Handlers) HandleGetEIJobStatus(w http.ResponseWriter, r *http.Request) {

	// Implementation similar to GetPolicyStatus.

	h.handleNotImplemented(w, r, "A1-EI interface not fully implemented")

}

// Utility methods.

// extractPolicyParams extracts and validates policy type ID and policy ID from URL vars.

func (h *A1Handlers) extractPolicyParams(vars map[string]string, w http.ResponseWriter, r *http.Request) (int, string, bool) {

	policyTypeIDStr := vars["policy_type_id"]

	policyTypeID, err := strconv.Atoi(policyTypeIDStr)

	if err != nil {

		h.handleError(w, r, NewInvalidRequestError("Invalid policy type ID format"))

		return 0, "", false

	}

	policyID := vars["policy_id"]

	if policyID == "" {

		h.handleError(w, r, NewInvalidRequestError("Policy ID is required"))

		return 0, "", false

	}

	return policyTypeID, policyID, true

}

// getRequestID extracts or generates a request ID for tracing.

func (h *A1Handlers) getRequestID(r *http.Request) string {

	requestID := r.Header.Get("X-Request-ID")

	if requestID == "" {

		requestID = r.Header.Get("X-Correlation-ID")

	}

	if requestID == "" {

		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())

	}

	return requestID

}

// isContentTypeJSON checks if the request content type is JSON.

func (h *A1Handlers) isContentTypeJSON(r *http.Request) bool {

	contentType := r.Header.Get("Content-Type")

	return strings.HasPrefix(contentType, ContentTypeJSON)

}

// readRequestBody reads the request body with size limit.

func (h *A1Handlers) readRequestBody(r *http.Request, maxSize int64) ([]byte, error) {

	if r.ContentLength > maxSize {

		return nil, fmt.Errorf("request body too large: %d bytes (max %d)", r.ContentLength, maxSize)

	}

	return io.ReadAll(io.LimitReader(r.Body, maxSize))

}

// handleError handles A1 errors and writes appropriate responses.

func (h *A1Handlers) handleError(w http.ResponseWriter, r *http.Request, err *A1Error) {

	requestID := h.getRequestID(r)

	err.CorrelationID = requestID

	err.Instance = r.URL.Path

	h.logger.ErrorWithContext("A1 request failed", err.Cause,

		"request_id", requestID,

		"method", r.Method,

		"path", r.URL.Path,

		"error_type", err.Type,

		"status", err.Status,
	)

	WriteA1Error(w, err)

}

// handleNotImplemented handles not implemented endpoints.

func (h *A1Handlers) handleNotImplemented(w http.ResponseWriter, r *http.Request, message string) {

	h.handleError(w, r, NewA1Error(

		ErrorTypeInternalServerError,

		"Not Implemented",

		http.StatusNotImplemented,

		message,
	))

}

// HealthCheckHandler handles health check requests.

func (h *A1Handlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {

	healthCheck := &HealthCheck{

		Status: "UP",

		Timestamp: time.Now(),

		Version: "1.0.0",

		Components: map[string]interface{}{

			"a1_policy": "UP",

			"a1_consumer": "UP",

			"a1_enrichment": "DEGRADED",
		},
	}

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(healthCheck)

}

// ReadinessCheckHandler handles readiness check requests.

func (h *A1Handlers) ReadinessCheckHandler(w http.ResponseWriter, r *http.Request) {

	// Check if service dependencies are ready.

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)

	defer cancel()

	ready := true

	checks := []ComponentCheck{}

	// Check storage connectivity.

	if h.storage != nil {

		start := time.Now()

		_, err := h.storage.ListPolicyTypes(ctx)

		duration := time.Since(start)

		status := "UP"

		message := "Storage connectivity OK"

		if err != nil {

			status = "DOWN"

			message = fmt.Sprintf("Storage error: %v", err)

			ready = false

		}

		checks = append(checks, ComponentCheck{

			Name: "storage",

			Status: status,

			Message: message,

			Timestamp: time.Now(),

			Duration: duration,
		})

	}

	healthCheck := &HealthCheck{

		Status: map[bool]string{true: "UP", false: "DOWN"}[ready],

		Timestamp: time.Now(),

		Checks: checks,
	}

	statusCode := map[bool]int{true: http.StatusOK, false: http.StatusServiceUnavailable}[ready]

	w.Header().Set("Content-Type", ContentTypeJSON)

	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(healthCheck)

}
