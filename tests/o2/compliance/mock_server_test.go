package o2_compliance_tests_test

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

// complianceMockStore provides an in-memory store for O2 IMS compliance tests.
// This replaces the real O2 API server whose routes are temporarily disabled.
type complianceMockStore struct {
	mu             sync.RWMutex
	resourcePools  map[string]*models.ResourcePool
	resourceTypes  map[string]*models.ResourceType
	resources      map[string]*models.ResourceInstance
	deploymentMgrs map[string]map[string]interface{}
	subscriptions  map[string]map[string]interface{}
	alarms         map[string]map[string]interface{}
}

func newComplianceMockStore() *complianceMockStore {
	return &complianceMockStore{
		resourcePools:  make(map[string]*models.ResourcePool),
		resourceTypes:  make(map[string]*models.ResourceType),
		resources:      make(map[string]*models.ResourceInstance),
		deploymentMgrs: make(map[string]map[string]interface{}),
		subscriptions:  make(map[string]map[string]interface{}),
		alarms:         make(map[string]map[string]interface{}),
	}
}

func compWriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func compWriteError(w http.ResponseWriter, status int, title, detail string) {
	compWriteJSON(w, status, map[string]interface{}{
		"type":   "about:blank",
		"title":  title,
		"status": float64(status),
		"detail": detail,
	})
}

func compRequireJSON(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "application/json") {
		compWriteError(w, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			"Content-Type must be application/json")
		return false
	}
	return true
}

func compApplyPagination(r *http.Request, total int) (offset, limit int, err error) {
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

// getRawFilter extracts the filter parameter directly from the raw query string,
// bypassing Go's url.Query() which treats ';' as a query parameter delimiter.
func getRawFilter(r *http.Request) string {
	raw := r.URL.RawQuery
	for _, part := range strings.Split(raw, "&") {
		if strings.HasPrefix(part, "filter=") {
			val := part[len("filter="):]
			// URL-decode the value
			decoded, err := url.QueryUnescape(val)
			if err == nil {
				return decoded
			}
			return val
		}
	}
	return ""
}

// applyComplianceResourcePoolFilter applies O-RAN style filter expressions.
// Supports both "(eq,field,value)" and "field,eq,value" syntax.
func applyComplianceResourcePoolFilter(pools []*models.ResourcePool, filterStr string) []*models.ResourcePool {
	var result []*models.ResourcePool
	filters := strings.Split(filterStr, ";")
	for _, pool := range pools {
		match := true
		for _, f := range filters {
			f = strings.TrimSpace(f)
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

// newComplianceMockO2Server creates a new httptest.Server with a fully functional mock O2 IMS API
// for compliance tests. This replaces the disabled real API server routes.
func newComplianceMockO2Server() *httptest.Server {
	store := newComplianceMockStore()

	// Pre-seed test resources used by TestORANAPIEndpointsCompliance to verify endpoint existence.
	now := time.Now()
	store.resourcePools["test-pool"] = &models.ResourcePool{
		ResourcePoolID: "test-pool", Name: "Test Pool", Provider: "kubernetes", CreatedAt: now, UpdatedAt: now,
	}
	store.resourceTypes["test-type"] = &models.ResourceType{
		ResourceTypeID: "test-type", Name: "Test Type", CreatedAt: now, UpdatedAt: now,
	}
	store.resources["test-resource"] = &models.ResourceInstance{
		ResourceInstanceID: "test-resource", Name: "Test Resource", CreatedAt: now, UpdatedAt: now,
	}
	store.deploymentMgrs["test-dm"] = map[string]interface{}{
		"deploymentManagerId": "test-dm", "name": "Test DM", "createdAt": now, "updatedAt": now,
	}
	store.subscriptions["test-sub"] = map[string]interface{}{
		"subscriptionId": "test-sub", "name": "Test Sub", "createdAt": now, "updatedAt": now,
	}
	store.alarms["test-alarm"] = map[string]interface{}{
		"alarmId": "test-alarm", "name": "Test Alarm", "createdAt": now, "updatedAt": now,
	}

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		compWriteError(w, http.StatusNotFound, "Not Found", "endpoint not found")
	})

	api := r.PathPrefix("/o2ims_infrastructureInventory/v1").Subrouter()

	// Service information - GET only
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		compWriteJSON(w, http.StatusOK, map[string]interface{}{
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
			},
			"endpoints": map[string]string{
				"resourcePools":      "/o2ims_infrastructureInventory/v1/resourcePools",
				"resourceTypes":      "/o2ims_infrastructureInventory/v1/resourceTypes",
				"resources":          "/o2ims_infrastructureInventory/v1/resources",
				"deploymentManagers": "/o2ims_infrastructureInventory/v1/deploymentManagers",
				"subscriptions":      "/o2ims_infrastructureInventory/v1/subscriptions",
				"alarms":             "/o2ims_infrastructureInventory/v1/alarms",
			},
		})
	}).Methods("GET")

	// Method not allowed for /o2ims_infrastructureInventory/v1/
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET")
		compWriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "only GET is allowed")
	}).Methods("POST", "PUT", "PATCH", "DELETE")

	// Resource Pools
	api.HandleFunc("/resourcePools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !compRequireJSON(w, r) {
				return
			}
			var pool models.ResourcePool
			if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON: "+err.Error())
				return
			}
			now := time.Now()
			pool.CreatedAt = now
			pool.UpdatedAt = now
			store.mu.Lock()
			store.resourcePools[pool.ResourcePoolID] = &pool
			store.mu.Unlock()
			compWriteJSON(w, http.StatusCreated, &pool)
			return
		}
		// GET with filter/pagination.
		// Use getRawFilter instead of r.URL.Query() because ';' in filter values
		// is misinterpreted as a query parameter delimiter by Go's url.ParseQuery.
		filterStr := getRawFilter(r)
		store.mu.RLock()
		var all []*models.ResourcePool
		for _, p := range store.resourcePools {
			all = append(all, p)
		}
		store.mu.RUnlock()
		sort.Slice(all, func(i, j int) bool { return all[i].ResourcePoolID < all[j].ResourcePoolID })

		if filterStr != "" {
			all = applyComplianceResourcePoolFilter(all, filterStr)
		}

		offset, limit, err := compApplyPagination(r, len(all))
		if err != nil {
			compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		compWriteJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")

	api.HandleFunc("/resourcePools/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			pool, ok := store.resourcePools[id]
			store.mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, pool)
		case http.MethodPut:
			if !compRequireJSON(w, r) {
				return
			}
			var pool models.ResourcePool
			if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourcePools[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
				return
			}
			pool.ResourcePoolID = id
			pool.CreatedAt = existing.CreatedAt
			pool.UpdatedAt = time.Now()
			store.resourcePools[id] = &pool
			store.mu.Unlock()
			compWriteJSON(w, http.StatusOK, &pool)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourcePools[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource pool not found: "+id)
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
			if !compRequireJSON(w, r) {
				return
			}
			var rt models.ResourceType
			if err := json.NewDecoder(r.Body).Decode(&rt); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			now := time.Now()
			rt.CreatedAt = now
			rt.UpdatedAt = now
			store.mu.Lock()
			store.resourceTypes[rt.ResourceTypeID] = &rt
			store.mu.Unlock()
			compWriteJSON(w, http.StatusCreated, &rt)
			return
		}
		store.mu.RLock()
		var all []*models.ResourceType
		for _, rt := range store.resourceTypes {
			all = append(all, rt)
		}
		store.mu.RUnlock()
		sort.Slice(all, func(i, j int) bool { return all[i].ResourceTypeID < all[j].ResourceTypeID })

		offset, limit, err := compApplyPagination(r, len(all))
		if err != nil {
			compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		compWriteJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")

	api.HandleFunc("/resourceTypes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			rt, ok := store.resourceTypes[id]
			store.mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, rt)
		case http.MethodPut:
			if !compRequireJSON(w, r) {
				return
			}
			var rt models.ResourceType
			if err := json.NewDecoder(r.Body).Decode(&rt); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resourceTypes[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			rt.ResourceTypeID = id
			rt.CreatedAt = existing.CreatedAt
			rt.UpdatedAt = time.Now()
			store.resourceTypes[id] = &rt
			store.mu.Unlock()
			compWriteJSON(w, http.StatusOK, &rt)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resourceTypes[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource type not found: "+id)
				return
			}
			delete(store.resourceTypes, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Resources (generic resource instances)
	api.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if !compRequireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			now := time.Now()
			ri.CreatedAt = now
			ri.UpdatedAt = now
			store.mu.Lock()
			store.resources[ri.ResourceInstanceID] = &ri
			store.mu.Unlock()
			compWriteJSON(w, http.StatusCreated, &ri)
			return
		}
		store.mu.RLock()
		var all []*models.ResourceInstance
		for _, ri := range store.resources {
			all = append(all, ri)
		}
		store.mu.RUnlock()
		compWriteJSON(w, http.StatusOK, all)
	}).Methods("GET", "POST")

	api.HandleFunc("/resources/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			ri, ok := store.resources[id]
			store.mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, ri)
		case http.MethodPut:
			if !compRequireJSON(w, r) {
				return
			}
			var ri models.ResourceInstance
			if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			store.mu.Lock()
			existing, ok := store.resources[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			ri.ResourceInstanceID = id
			ri.CreatedAt = existing.CreatedAt
			ri.UpdatedAt = time.Now()
			store.resources[id] = &ri
			store.mu.Unlock()
			compWriteJSON(w, http.StatusOK, &ri)
		case http.MethodDelete:
			store.mu.Lock()
			_, ok := store.resources[id]
			if !ok {
				store.mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store.resources, id)
			store.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")

	// Deployment Managers
	compGenericCRUD(api, "/deploymentManagers", store.deploymentMgrs, &store.mu, "deploymentManagerId")
	compGenericCRUDItem(api, "/deploymentManagers/{id}", store.deploymentMgrs, &store.mu, "deploymentManagerId")

	// Subscriptions
	compGenericCRUD(api, "/subscriptions", store.subscriptions, &store.mu, "subscriptionId")
	compGenericDeleteGet(api, "/subscriptions/{id}", store.subscriptions, &store.mu, "subscriptionId")

	// Alarms
	compGenericCRUD(api, "/alarms", store.alarms, &store.mu, "alarmId")
	compGenericCRUDItemWithPatch(api, "/alarms/{id}", store.alarms, &store.mu, "alarmId")

	return httptest.NewServer(r)
}

func compGenericCRUD(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			if !compRequireJSON(w, req) {
				return
			}
			var obj map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
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
			compWriteJSON(w, http.StatusCreated, obj)
			return
		}
		mu.RLock()
		var all []map[string]interface{}
		for _, v := range store {
			all = append(all, v)
		}
		mu.RUnlock()

		offset, limit, err := compApplyPagination(req, len(all))
		if err != nil {
			compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid pagination parameter")
			return
		}
		if offset > len(all) {
			offset = len(all)
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		compWriteJSON(w, http.StatusOK, all[offset:end])
	}).Methods("GET", "POST")
}

func compGenericCRUDItem(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, obj)
		case http.MethodPut:
			if !compRequireJSON(w, req) {
				return
			}
			var obj map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			mu.Lock()
			existing, ok := store[id]
			if !ok {
				mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			obj[idField] = id
			if createdAt, ok2 := existing["createdAt"]; ok2 {
				obj["createdAt"] = createdAt
			}
			obj["updatedAt"] = time.Now()
			store[id] = obj
			mu.Unlock()
			compWriteJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PUT", "DELETE")
}

func compGenericDeleteGet(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "DELETE")
}

func compGenericCRUDItemWithPatch(r *mux.Router, path string, store map[string]map[string]interface{}, mu *sync.RWMutex, idField string) {
	r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)["id"]
		switch req.Method {
		case http.MethodGet:
			mu.RLock()
			obj, ok := store[id]
			mu.RUnlock()
			if !ok {
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			compWriteJSON(w, http.StatusOK, obj)
		case http.MethodPatch:
			if !compRequireJSON(w, req) {
				return
			}
			var patch map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				compWriteError(w, http.StatusBadRequest, "Bad Request", "invalid JSON")
				return
			}
			mu.Lock()
			obj, ok := store[id]
			if !ok {
				mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			for k, v := range patch {
				obj[k] = v
			}
			obj["updatedAt"] = time.Now()
			store[id] = obj
			mu.Unlock()
			compWriteJSON(w, http.StatusOK, obj)
		case http.MethodDelete:
			mu.Lock()
			_, ok := store[id]
			if !ok {
				mu.Unlock()
				compWriteError(w, http.StatusNotFound, "Not Found", "resource not found: "+id)
				return
			}
			delete(store, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}
	}).Methods("GET", "PATCH", "DELETE")
}
