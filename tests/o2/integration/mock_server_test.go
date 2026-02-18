package o2_integration_tests_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// mockO2Store provides an in-memory store for O2 IMS resources used by tests.
// This replaces the real O2 API server whose routes are temporarily disabled.
type mockO2Store struct {
	mu               sync.RWMutex
	resourcePools    map[string]*models.ResourcePool
	resourceTypes    map[string]*models.ResourceType
	resourceInstances map[string]*models.ResourceInstance
	deploymentMgrs   map[string]map[string]interface{}
	subscriptions    map[string]map[string]interface{}
	alarms           map[string]map[string]interface{}
}

func newMockO2Store() *mockO2Store {
	return &mockO2Store{
		resourcePools:    make(map[string]*models.ResourcePool),
		resourceTypes:    make(map[string]*models.ResourceType),
		resourceInstances: make(map[string]*models.ResourceInstance),
		deploymentMgrs:   make(map[string]map[string]interface{}),
		subscriptions:    make(map[string]map[string]interface{}),
		alarms:           make(map[string]map[string]interface{}),
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes an O-RAN compliant error response.
func writeError(w http.ResponseWriter, status int, title, detail string) {
	writeJSON(w, status, map[string]interface{}{
		"type":   "about:blank",
		"title":  title,
		"status": float64(status),
		"detail": detail,
	})
}

// requireJSON checks that the request Content-Type is application/json.
func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			"Content-Type must be application/json")
		return false
	}
	return true
}

// applyPagination applies limit/offset query parameters to a slice.
func applyPagination(r *http.Request, total int) (offset, limit int, err error) {
	offset = 0
	limit = total

	if v := r.URL.Query().Get("offset"); v != "" {
		offset, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, err
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, err
		}
	}
	return offset, limit, nil
}

// newMockO2Server creates a new httptest.Server with a fully functional mock O2 IMS API.
func newMockO2Server() *httptest.Server {
	store := newMockO2Store()
	r := mux.NewRouter()

	// Only /o2ims/v1/ prefix is accessible; /o2ims/ without version is not.
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		writeError(w, http.StatusNotFound, "Not Found", "endpoint not found")
	})

	api := r.PathPrefix("/o2ims/v1").Subrouter()

	// Service information
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":            "nephoran-o2ims-001",
			"name":          "Nephoran O2 IMS",
			"description":   "Nephoran O-RAN O2 IMS Interface",
			"version":       "v1.0.0",
			"apiVersion":    "v1.0",
			"specification": "O-RAN.WG6.O2ims-Interface-v01.01",
			"capabilities": []string{
				"InfrastructureInventory",
				"InfrastructureMonitoring",
				"InfrastructureProvisioning",
				"DeploymentTemplates",
			},
			"endpoints": map[string]string{
				"resourcePools":      "/o2ims/v1/resourcePools",
				"resourceTypes":      "/o2ims/v1/resourceTypes",
				"resources":          "/o2ims/v1/resources",
				"deploymentManagers": "/o2ims/v1/deploymentManagers",
				"subscriptions":      "/o2ims/v1/subscriptions",
				"alarms":             "/o2ims/v1/alarms",
			},
		})
	}).Methods("GET")

	// Method not allowed handler for /o2ims/v1/
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "only GET is allowed on this endpoint")
	}).Methods("POST", "PUT", "PATCH", "DELETE")

	// Resource Pools
	api.HandleFunc("/resourcePools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !requireJSON(w, r) {
				return
			}
			var pool models.ResourcePool
			if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON: "+err.Error())
				return
			}
			now := time.Now()
			pool.CreatedAt = now
			pool.UpdatedAt = now
			store.mu.Lock()
			store.resourcePools[pool.ResourcePoolID] = &pool
			store.mu.Unlock()
			writeJSON(w, http.StatusCreated, &pool)
			return
		}
		// GET with filtering/pagination.
		// Use intGetRawFilter instead of r.URL.Query() because ';' in filter values
		// is misinterpreted as a query parameter delimiter by Go's url.ParseQuery.
		filterStr := intGetRawFilter(r)
		store.mu.RLock()
		var all []*models.ResourcePool
		for _, p := range store.resourcePools {
			all = append(all, p)
		}
		store.mu.RUnlock()
		// Sort for consistent pagination
		sort.Slice(all, func(i, j int) bool { return all[i].ResourcePoolID < all[j].ResourcePoolID })

		if filterStr != "" {
			all = applyResourcePoolFilter(all, filterStr)
		}

		offset, limit, err := applyPagination(r, len(all))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		writeJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")

	api.HandleFunc("/resourcePools/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			pool, ok := store.resourcePools[id]
			store.mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, pool)
		case http.MethodPut:
			if !requireJSON(w, r) {
				return
			}
			var pool models.ResourcePool
			if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourcePools[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
				return
			}
			pool.ResourcePoolID = id
			pool.CreatedAt = existing.CreatedAt
			pool.UpdatedAt = time.Now()
			store.resourcePools[id] = &pool
			store.mu.Unlock()
			writeJSON(w, http.StatusOK, &pool)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourcePools[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
				return
			}
			delete(store.resourcePools, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Resource Types
	api.HandleFunc("/resourceTypes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !requireJSON(w, r) {
				return
			}
			var rt models.ResourceType
			if err := json.NewDecoder(r.Body).Decode(&rt); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			now := time.Now()
			rt.CreatedAt = now
			rt.UpdatedAt = now
			store.mu.Lock()
			store.resourceTypes[rt.ResourceTypeID] = &rt
			store.mu.Unlock()
			writeJSON(w, http.StatusCreated, &rt)
			return
		}
		store.mu.RLock()
		var all []*models.ResourceType
		for _, rt := range store.resourceTypes {
			all = append(all, rt)
		}
		store.mu.RUnlock()
		// Sort for consistent pagination
		sort.Slice(all, func(i, j int) bool { return all[i].ResourceTypeID < all[j].ResourceTypeID })

		offset, limit, err := applyPagination(r, len(all))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		writeJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")

	api.HandleFunc("/resourceTypes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			rt, ok := store.resourceTypes[id]
			store.mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, rt)
		case http.MethodPut:
			if !requireJSON(w, r) {
				return
			}
			var rt models.ResourceType
			if err := json.NewDecoder(r.Body).Decode(&rt); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourceTypes[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			rt.ResourceTypeID = id
			rt.CreatedAt = existing.CreatedAt
			rt.UpdatedAt = time.Now()
			store.resourceTypes[id] = &rt
			store.mu.Unlock()
			writeJSON(w, http.StatusOK, &rt)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourceTypes[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			delete(store.resourceTypes, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Resource Instances
	api.HandleFunc("/resourceInstances", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !requireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			now := time.Now()
			ri.CreatedAt = now
			ri.UpdatedAt = now
			store.mu.Lock()
			store.resourceInstances[ri.ResourceInstanceID] = &ri
			store.mu.Unlock()
			writeJSON(w, http.StatusCreated, &ri)
			return
		}
		store.mu.RLock()
		var all []*models.ResourceInstance
		for _, ri := range store.resourceInstances {
			all = append(all, ri)
		}
		store.mu.RUnlock()
		writeJSON(w, http.StatusOK, all)
	}).Methods("GET", "POST")

	api.HandleFunc("/resourceInstances/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			ri, ok := store.resourceInstances[id]
			store.mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource instance not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, ri)
		case http.MethodPut:
			if !requireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourceInstances[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource instance not found: "+id)
				return
			}
			ri.ResourceInstanceID = id
			ri.CreatedAt = existing.CreatedAt
			ri.UpdatedAt = time.Now()
			store.resourceInstances[id] = &ri
			store.mu.Unlock()
			writeJSON(w, http.StatusOK, &ri)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourceInstances[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource instance not found: "+id)
				return
			}
			delete(store.resourceInstances, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Resources (generic - map to resource instances for compatibility)
	api.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !requireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			now := time.Now()
			ri.CreatedAt = now
			ri.UpdatedAt = now
			store.mu.Lock()
			store.resourceInstances[ri.ResourceInstanceID] = &ri
			store.mu.Unlock()
			writeJSON(w, http.StatusCreated, &ri)
			return
		}
		store.mu.RLock()
		var all []*models.ResourceInstance
		for _, ri := range store.resourceInstances {
			all = append(all, ri)
		}
		store.mu.RUnlock()
		writeJSON(w, http.StatusOK, all)
	}).Methods("GET", "POST")

	api.HandleFunc("/resources/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			ri, ok := store.resourceInstances[id]
			store.mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, ri)
		case http.MethodPut:
			if !requireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourceInstances[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			ri.ResourceInstanceID = id
			ri.CreatedAt = existing.CreatedAt
			ri.UpdatedAt = time.Now()
			store.resourceInstances[id] = &ri
			store.mu.Unlock()
			writeJSON(w, http.StatusOK, &ri)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourceInstances[id]
			if !ok {
				store.mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store.resourceInstances, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Deployment Managers
	genericCRUD(api, "/deploymentManagers", store.deploymentMgrs, &store.mu, "deploymentManagerId")
	genericCRUDItem(api, "/deploymentManagers/{id}", store.deploymentMgrs, &store.mu, "deploymentManagerId")

	// Subscriptions
	genericCRUD(api, "/subscriptions", store.subscriptions, &store.mu, "subscriptionId")
	genericDeleteGet(api, "/subscriptions/{id}", store.subscriptions, &store.mu, "subscriptionId")

	// Alarms
	genericCRUD(api, "/alarms", store.alarms, &store.mu, "alarmId")
	genericCRUDItemWithPatch(api, "/alarms/{id}", store.alarms, &store.mu, "alarmId")

	// Metrics endpoint
	api.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods("GET")

	return httptest.NewServer(r)
}

// intGetRawFilter extracts the filter parameter directly from the raw query string,
// bypassing Go's url.Query() which treats ';' as a query parameter delimiter.
func intGetRawFilter(r *http.Request) string {
	raw := r.URL.RawQuery
	for _, part := range strings.Split(raw, "&") {
		if strings.HasPrefix(part, "filter=") {
			val := part[len("filter="):]
			decoded, err := url.QueryUnescape(val)
			if err == nil {
				return decoded
			}
			return val
		}
	}
	return ""
}

// applyResourcePoolFilter applies O-RAN style filter expressions to resource pools.
// Supports both "(eq,field,value)" and "field,eq,value" syntax.
func applyResourcePoolFilter(pools []*models.ResourcePool, filterStr string) []*models.ResourcePool {
	var result []*models.ResourcePool
	// Split on semicolons for AND filters
	filters := strings.Split(filterStr, ";")

	for _, pool := range pools {
		match := true
		for _, f := range filters {
			f = strings.TrimSpace(f)
			// Strip parentheses: (eq,field,value) -> eq,field,value
			f = strings.Trim(f, "()")
			parts := strings.SplitN(f, ",", 3)
			if len(parts) != 3 {
				continue
			}
			op, field, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
			if op != "eq" {
				continue
			}
			var fieldValue string
			switch field {
			case "provider":
				fieldValue = pool.Provider
			case "location":
				fieldValue = pool.Location
			case "state":
				fieldValue = pool.State
			case "zone":
				fieldValue = pool.Zone
			case "region":
				fieldValue = pool.Region
			case "oCloudId":
				fieldValue = pool.OCloudID
			}
			if fieldValue != value {
				match = false
				break
			}
		}
		if match {
			result = append(result, pool)
		}
	}
	return result
}

// genericCRUD registers GET (list) and POST (create) handlers for a resource.
func genericCRUD(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			if !requireJSON(w, req) {
				return
			}
			var obj map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			id, _ := obj[idField].(string)
			if id == "" {
				id = "auto-" + strconv.FormatInt(time.Now().UnixNano(), 10)
				obj[idField] = id
			}
			obj["createdAt"] = time.Now()
			obj["updatedAt"] = time.Now()
			mu.Lock()
			store[id] = obj
			mu.Unlock()
			writeJSON(w, http.StatusCreated, obj)
			return
		}
		mu.RLock()
		var allKeys []string
		for k := range store {
			allKeys = append(allKeys, k)
		}
		sort.Strings(allKeys)
		var all []map[string]interface{}
		for _, k := range allKeys {
			all = append(all, store[k])
		}
		mu.RUnlock()

		offset, limit, err := applyPagination(req, len(all))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		writeJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")
}

// genericCRUDItem registers GET, PUT, DELETE for individual items.
func genericCRUDItem(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, obj)
		case http.MethodPut:
			if !requireJSON(w, req) {
				return
			}
			var obj map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			mu.Lock()
			existing, ok := store[id]
			if !ok {
				mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			obj[idField] = id
			if createdAt, ok2 := existing["createdAt"]; ok2 {
				obj["createdAt"] = createdAt
			}
			obj["updatedAt"] = time.Now()
			store[id] = obj
			mu.Unlock()
			writeJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")
}

// genericDeleteGet registers GET and DELETE for subscription-style items.
func genericDeleteGet(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "DELETE")
}

// genericCRUDItemWithPatch registers GET, PATCH, DELETE for alarm-style items.
func genericCRUDItemWithPatch(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, obj)
		case http.MethodPatch:
			if !requireJSON(w, req) {
				return
			}
			var patch map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				writeError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			mu.Lock()
			obj, ok := store[id]
			if !ok {
				mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			for k, v := range patch {
				obj[k] = v
			}
			obj["updatedAt"] = time.Now()
			store[id] = obj
			mu.Unlock()
			writeJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				writeError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PATCH", "DELETE")
}
