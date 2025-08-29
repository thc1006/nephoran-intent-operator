
package integration



import (

	"fmt"

	"math"

	"sort"

	"strings"

	"sync"

	"time"



	"github.com/google/uuid"

)



// FakeWeaviateServer provides an in-memory implementation of Weaviate for testing.

type FakeWeaviateServer struct {

	mu          sync.RWMutex

	classes     map[string]*WeaviateClass

	objects     map[string]map[string]*WeaviateObject // className -> objectID -> object

	embeddings  map[string][]float32                  // objectID -> embedding vector

	initialized bool

}



// WeaviateClass represents a Weaviate class schema.

type WeaviateClass struct {

	Name         string                 `json:"class"`

	Description  string                 `json:"description"`

	Properties   []WeaviateProperty     `json:"properties"`

	Vectorizer   string                 `json:"vectorizer"`

	ModuleConfig map[string]interface{} `json:"moduleConfig"`

}



// WeaviateProperty represents a property in a Weaviate class.

type WeaviateProperty struct {

	Name        string   `json:"name"`

	DataType    []string `json:"dataType"`

	Description string   `json:"description"`

}



// WeaviateObject represents an object in Weaviate.

type WeaviateObject struct {

	ID         string                 `json:"id"`

	Class      string                 `json:"class"`

	Properties map[string]interface{} `json:"properties"`

	Vector     []float32              `json:"vector,omitempty"`

	CreatedAt  time.Time              `json:"createdTimeUnix"`

	UpdatedAt  time.Time              `json:"lastUpdateTimeUnix"`

}



// SearchResult represents search results from Weaviate.

type SearchResult struct {

	Objects []SearchObject `json:"objects"`

}



// SearchObject represents an individual search result object.

type SearchObject struct {

	ID         string                 `json:"id"`

	Class      string                 `json:"class"`

	Properties map[string]interface{} `json:"properties"`

	Vector     []float32              `json:"vector,omitempty"`

	Score      float32                `json:"score,omitempty"`

	Distance   float32                `json:"distance,omitempty"`

	Certainty  float32                `json:"certainty,omitempty"`

}



// NewFakeWeaviateServer creates a new fake Weaviate server.

func NewFakeWeaviateServer() *FakeWeaviateServer {

	server := &FakeWeaviateServer{

		classes:    make(map[string]*WeaviateClass),

		objects:    make(map[string]map[string]*WeaviateObject),

		embeddings: make(map[string][]float32),

	}



	// Initialize with telecommunications knowledge base schema.

	server.initializeWithTelecomSchema()



	return server

}



// initializeWithTelecomSchema sets up the telecommunications domain schema.

func (f *FakeWeaviateServer) initializeWithTelecomSchema() {

	f.mu.Lock()

	defer f.mu.Unlock()



	// Create TelecomDocument class.

	telecomDocClass := &WeaviateClass{

		Name:        "TelecomDocument",

		Description: "Telecommunications domain documents and specifications",

		Properties: []WeaviateProperty{

			{

				Name:        "title",

				DataType:    []string{"text"},

				Description: "Document title",

			},

			{

				Name:        "content",

				DataType:    []string{"text"},

				Description: "Document content",

			},

			{

				Name:        "documentType",

				DataType:    []string{"text"},

				Description: "Type of document (3GPP, O-RAN, IETF, etc.)",

			},

			{

				Name:        "category",

				DataType:    []string{"text"},

				Description: "Document category (specification, implementation, guide)",

			},

			{

				Name:        "tags",

				DataType:    []string{"text[]"},

				Description: "Associated tags and keywords",

			},

			{

				Name:        "version",

				DataType:    []string{"text"},

				Description: "Document version",

			},

			{

				Name:        "releaseDate",

				DataType:    []string{"date"},

				Description: "Document release date",

			},

		},

		Vectorizer: "text2vec-openai",

		ModuleConfig: map[string]interface{}{

			"text2vec-openai": map[string]interface{}{

				"model": "text-embedding-3-small",

			},

		},

	}



	// Create NetworkFunction class.

	networkFunctionClass := &WeaviateClass{

		Name:        "NetworkFunction",

		Description: "5G Core and O-RAN network function definitions",

		Properties: []WeaviateProperty{

			{

				Name:        "name",

				DataType:    []string{"text"},

				Description: "Network function name (AMF, SMF, UPF, etc.)",

			},

			{

				Name:        "description",

				DataType:    []string{"text"},

				Description: "Function description and purpose",

			},

			{

				Name:        "type",

				DataType:    []string{"text"},

				Description: "Function type (5GC, O-RAN, RAN)",

			},

			{

				Name:        "interfaces",

				DataType:    []string{"text[]"},

				Description: "Supported interfaces",

			},

			{

				Name:        "deploymentRequirements",

				DataType:    []string{"text"},

				Description: "Deployment requirements and constraints",

			},

			{

				Name:        "scalingPolicy",

				DataType:    []string{"text"},

				Description: "Auto-scaling policies and rules",

			},

		},

		Vectorizer: "text2vec-openai",

	}



	f.classes["TelecomDocument"] = telecomDocClass

	f.classes["NetworkFunction"] = networkFunctionClass

	f.objects["TelecomDocument"] = make(map[string]*WeaviateObject)

	f.objects["NetworkFunction"] = make(map[string]*WeaviateObject)



	// Populate with sample telecommunications data.

	f.populateSampleData()

	f.initialized = true

}



// populateSampleData adds sample telecommunications knowledge to the database.

func (f *FakeWeaviateServer) populateSampleData() {

	// Sample 5G Core network function documents.

	sampleDocuments := []struct {

		title        string

		content      string

		documentType string

		category     string

		tags         []string

	}{

		{

			title:        "3GPP TS 23.501 - System architecture for the 5G System",

			content:      "This specification defines the Stage 2 system architecture for the 5G System (5GS). The 5GS consists of the 5G Access Network (5G-AN), 5G Core Network (5GC), and UE. The AMF provides UE-based authentication, authorization, mobility management, etc.",

			documentType: "3GPP",

			category:     "specification",

			tags:         []string{"5G", "architecture", "AMF", "SMF", "UPF", "core", "network"},

		},

		{

			title:        "O-RAN Architecture Description",

			content:      "The O-RAN ALLIANCE aims to re-shape the RAN industry towards more intelligent, open, virtualized and fully interoperable mobile networks. The O-RAN architecture includes O-DU, O-CU-CP, O-CU-UP, Near-RT RIC, and Non-RT RIC components.",

			documentType: "O-RAN",

			category:     "specification",

			tags:         []string{"O-RAN", "RIC", "O-DU", "O-CU", "xApp", "rApp", "intelligent"},

		},

		{

			title:        "AMF Deployment Guide for High Availability",

			content:      "Access and Mobility Management Function (AMF) deployment requires careful consideration of high availability, scalability, and security. Recommended deployment includes multiple replicas, proper load balancing, and comprehensive monitoring.",

			documentType: "implementation",

			category:     "guide",

			tags:         []string{"AMF", "deployment", "high-availability", "scaling", "monitoring"},

		},

		{

			title:        "SMF Auto-scaling Configuration",

			content:      "Session Management Function (SMF) auto-scaling based on PDU session load. Configure HPA with CPU and memory metrics, with scale-out threshold at 70% CPU utilization and scale-in at 30%.",

			documentType: "implementation",

			category:     "guide",

			tags:         []string{"SMF", "auto-scaling", "HPA", "PDU", "session", "CPU", "memory"},

		},

		{

			title:        "UPF Performance Optimization",

			content:      "User Plane Function (UPF) performance optimization focuses on packet processing throughput, DPDK usage, and CPU affinity. Edge deployment patterns for low-latency applications.",

			documentType: "implementation",

			category:     "guide",

			tags:         []string{"UPF", "performance", "DPDK", "edge", "latency", "throughput"},

		},

	}



	for i, doc := range sampleDocuments {

		objID := uuid.New().String()

		obj := &WeaviateObject{

			ID:    objID,

			Class: "TelecomDocument",

			Properties: map[string]interface{}{

				"title":        doc.title,

				"content":      doc.content,

				"documentType": doc.documentType,

				"category":     doc.category,

				"tags":         doc.tags,

				"version":      "1.0",

				"releaseDate":  time.Now().Add(-time.Duration(i*30) * 24 * time.Hour),

			},

			Vector:    f.generateMockEmbedding(doc.content),

			CreatedAt: time.Now(),

			UpdatedAt: time.Now(),

		}

		f.objects["TelecomDocument"][objID] = obj

		f.embeddings[objID] = obj.Vector

	}



	// Sample network functions.

	sampleFunctions := []struct {

		name                   string

		description            string

		funcType               string

		interfaces             []string

		deploymentRequirements string

		scalingPolicy          string

	}{

		{

			name:                   "AMF",

			description:            "Access and Mobility Management Function handles UE registration, authentication, and mobility management in 5G networks",

			funcType:               "5GC",

			interfaces:             []string{"N1", "N2", "N8", "N11", "N12", "N14"},

			deploymentRequirements: "High availability deployment with minimum 3 replicas, persistent storage for session state, secure communication channels",

			scalingPolicy:          "Horizontal scaling based on registered UE count and signaling load, scale-out threshold: 10000 UEs per instance",

		},

		{

			name:                   "SMF",

			description:            "Session Management Function manages PDU sessions and performs IP address allocation",

			funcType:               "5GC",

			interfaces:             []string{"N4", "N7", "N10", "N11"},

			deploymentRequirements: "Stateful deployment with session persistence, integration with UPF for N4 interface",

			scalingPolicy:          "Scale based on active PDU sessions, target: 5000 sessions per instance",

		},

		{

			name:                   "Near-RT RIC",

			description:            "Near Real-Time RAN Intelligent Controller provides sub-second control loop for RAN optimization",

			funcType:               "O-RAN",

			interfaces:             []string{"A1", "E2", "O1"},

			deploymentRequirements: "Low-latency deployment close to RAN, xApp runtime environment, ML inference capabilities",

			scalingPolicy:          "Vertical scaling for compute-intensive ML workloads, horizontal scaling for multiple cells coverage",

		},

	}



	for _, nf := range sampleFunctions {

		objID := uuid.New().String()

		obj := &WeaviateObject{

			ID:    objID,

			Class: "NetworkFunction",

			Properties: map[string]interface{}{

				"name":                   nf.name,

				"description":            nf.description,

				"type":                   nf.funcType,

				"interfaces":             nf.interfaces,

				"deploymentRequirements": nf.deploymentRequirements,

				"scalingPolicy":          nf.scalingPolicy,

			},

			Vector:    f.generateMockEmbedding(nf.description + " " + nf.deploymentRequirements),

			CreatedAt: time.Now(),

			UpdatedAt: time.Now(),

		}

		f.objects["NetworkFunction"][objID] = obj

		f.embeddings[objID] = obj.Vector

	}

}



// generateMockEmbedding creates a mock embedding vector based on text content.

func (f *FakeWeaviateServer) generateMockEmbedding(text string) []float32 {

	// Simple mock embedding based on text characteristics.

	text = strings.ToLower(text)

	embedding := make([]float32, 1536) // OpenAI embedding size



	// Use text characteristics to generate deterministic but varied embeddings.

	words := strings.Fields(text)

	for i, word := range words {

		if i >= len(embedding) {

			break

		}

		// Simple hash-based mock embedding.

		hash := 0

		for _, char := range word {

			hash = hash*31 + int(char)

		}

		embedding[i] = float32(hash%1000-500) / 1000.0 // Normalize to [-0.5, 0.5]

	}



	// Add some telecommunications domain-specific patterns.

	telecomKeywords := map[string]float32{

		"amf": 0.8, "smf": 0.7, "upf": 0.6, "5g": 0.9, "core": 0.5,

		"oran": 0.8, "ric": 0.7, "ran": 0.6, "deployment": 0.4, "scaling": 0.3,

	}



	for keyword, weight := range telecomKeywords {

		if strings.Contains(text, keyword) {

			for i := range embedding[:10] {

				embedding[i] += weight * 0.1

			}

		}

	}



	// Normalize vector.

	norm := float32(0)

	for _, val := range embedding {

		norm += val * val

	}

	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {

		for i := range embedding {

			embedding[i] /= norm

		}

	}



	return embedding

}



// IsReady checks if the fake Weaviate server is ready.

func (f *FakeWeaviateServer) IsReady() bool {

	f.mu.RLock()

	defer f.mu.RUnlock()

	return f.initialized

}



// GetSchema returns the schema of all classes.

func (f *FakeWeaviateServer) GetSchema() map[string]*WeaviateClass {

	f.mu.RLock()

	defer f.mu.RUnlock()



	result := make(map[string]*WeaviateClass)

	for k, v := range f.classes {

		result[k] = v

	}

	return result

}



// AddObject adds a new object to the specified class.

func (f *FakeWeaviateServer) AddObject(className string, obj *WeaviateObject) error {

	f.mu.Lock()

	defer f.mu.Unlock()



	if _, exists := f.classes[className]; !exists {

		return fmt.Errorf("class %s does not exist", className)

	}



	if obj.ID == "" {

		obj.ID = uuid.New().String()

	}

	obj.Class = className

	obj.CreatedAt = time.Now()

	obj.UpdatedAt = time.Now()



	if f.objects[className] == nil {

		f.objects[className] = make(map[string]*WeaviateObject)

	}



	f.objects[className][obj.ID] = obj

	if len(obj.Vector) > 0 {

		f.embeddings[obj.ID] = obj.Vector

	}



	return nil

}



// GetObject retrieves an object by ID and class.

func (f *FakeWeaviateServer) GetObject(className, objectID string) (*WeaviateObject, error) {

	f.mu.RLock()

	defer f.mu.RUnlock()



	if classObjects, exists := f.objects[className]; exists {

		if obj, exists := classObjects[objectID]; exists {

			return obj, nil

		}

	}



	return nil, fmt.Errorf("object %s not found in class %s", objectID, className)

}



// DeleteObject removes an object from the specified class.

func (f *FakeWeaviateServer) DeleteObject(className, objectID string) error {

	f.mu.Lock()

	defer f.mu.Unlock()



	if classObjects, exists := f.objects[className]; exists {

		delete(classObjects, objectID)

		delete(f.embeddings, objectID)

		return nil

	}



	return fmt.Errorf("object %s not found in class %s", objectID, className)

}



// VectorSearch performs vector similarity search.

func (f *FakeWeaviateServer) VectorSearch(className string, vector []float32, limit int, certainty float32) (*SearchResult, error) {

	f.mu.RLock()

	defer f.mu.RUnlock()



	if _, exists := f.classes[className]; !exists {

		return nil, fmt.Errorf("class %s does not exist", className)

	}



	var results []SearchObject

	classObjects := f.objects[className]



	for objectID, obj := range classObjects {

		if embedding, exists := f.embeddings[objectID]; exists {

			similarity := f.cosineSimilarity(vector, embedding)

			if similarity >= certainty {

				results = append(results, SearchObject{

					ID:         obj.ID,

					Class:      obj.Class,

					Properties: obj.Properties,

					Vector:     embedding,

					Certainty:  similarity,

					Distance:   1.0 - similarity,

					Score:      similarity,

				})

			}

		}

	}



	// Sort by similarity score (descending).

	sort.Slice(results, func(i, j int) bool {

		return results[i].Score > results[j].Score

	})



	// Apply limit.

	if len(results) > limit {

		results = results[:limit]

	}



	return &SearchResult{Objects: results}, nil

}



// HybridSearch performs hybrid search combining vector and keyword search.

func (f *FakeWeaviateServer) HybridSearch(className string, query string, vector []float32, limit int) (*SearchResult, error) {

	f.mu.RLock()

	defer f.mu.RUnlock()



	if _, exists := f.classes[className]; !exists {

		return nil, fmt.Errorf("class %s does not exist", className)

	}



	var results []SearchObject

	classObjects := f.objects[className]

	queryLower := strings.ToLower(query)

	queryTerms := strings.Fields(queryLower)



	for objectID, obj := range classObjects {

		score := float32(0.0)



		// Vector similarity component (70% weight).

		if embedding, exists := f.embeddings[objectID]; exists && len(vector) > 0 {

			vectorScore := f.cosineSimilarity(vector, embedding)

			score += vectorScore * 0.7

		}



		// Keyword search component (30% weight).

		keywordScore := f.calculateKeywordScore(obj, queryTerms)

		score += keywordScore * 0.3



		if score > 0.1 { // Minimum threshold

			results = append(results, SearchObject{

				ID:         obj.ID,

				Class:      obj.Class,

				Properties: obj.Properties,

				Vector:     f.embeddings[objectID],

				Score:      score,

				Certainty:  score,

				Distance:   1.0 - score,

			})

		}

	}



	// Sort by combined score (descending).

	sort.Slice(results, func(i, j int) bool {

		return results[i].Score > results[j].Score

	})



	// Apply limit.

	if len(results) > limit {

		results = results[:limit]

	}



	return &SearchResult{Objects: results}, nil

}



// cosineSimilarity calculates cosine similarity between two vectors.

func (f *FakeWeaviateServer) cosineSimilarity(a, b []float32) float32 {

	if len(a) != len(b) {

		return 0

	}



	var dotProduct, normA, normB float32

	for i := range a {

		dotProduct += a[i] * b[i]

		normA += a[i] * a[i]

		normB += b[i] * b[i]

	}



	if normA == 0 || normB == 0 {

		return 0

	}



	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))

}



// calculateKeywordScore calculates keyword matching score for an object.

func (f *FakeWeaviateServer) calculateKeywordScore(obj *WeaviateObject, queryTerms []string) float32 {

	if len(queryTerms) == 0 {

		return 0

	}



	// Convert object properties to searchable text.

	searchableText := f.extractSearchableText(obj)

	searchableTextLower := strings.ToLower(searchableText)



	matchedTerms := 0

	for _, term := range queryTerms {

		if strings.Contains(searchableTextLower, term) {

			matchedTerms++

		}

	}



	return float32(matchedTerms) / float32(len(queryTerms))

}



// extractSearchableText extracts searchable text from object properties.

func (f *FakeWeaviateServer) extractSearchableText(obj *WeaviateObject) string {

	var textParts []string



	for _, value := range obj.Properties {

		switch v := value.(type) {

		case string:

			textParts = append(textParts, v)

		case []string:

			textParts = append(textParts, v...)

		case []interface{}:

			for _, item := range v {

				if str, ok := item.(string); ok {

					textParts = append(textParts, str)

				}

			}

		}

	}



	return strings.Join(textParts, " ")

}



// GetObjectCount returns the number of objects in a class.

func (f *FakeWeaviateServer) GetObjectCount(className string) int {

	f.mu.RLock()

	defer f.mu.RUnlock()



	if classObjects, exists := f.objects[className]; exists {

		return len(classObjects)

	}

	return 0

}



// Reset clears all data and reinitializes the server.

func (f *FakeWeaviateServer) Reset() {

	f.mu.Lock()

	defer f.mu.Unlock()



	f.classes = make(map[string]*WeaviateClass)

	f.objects = make(map[string]map[string]*WeaviateObject)

	f.embeddings = make(map[string][]float32)

	f.initialized = false



	f.initializeWithTelecomSchema()

}

