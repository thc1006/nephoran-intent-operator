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

package webui

import (
	"net/http"
)

// OpenAPISpec represents the OpenAPI 3.0 specification.

type OpenAPISpec struct {
	OpenAPI string `json:"openapi"`

	Info *OpenAPIInfo `json:"info"`

	Servers []*OpenAPIServer `json:"servers,omitempty"`

	Paths map[string]*OpenAPIPath `json:"paths"`

	Components *OpenAPIComponents `json:"components,omitempty"`

	Security []map[string][]string `json:"security,omitempty"`

	Tags []*OpenAPITag `json:"tags,omitempty"`
}

// OpenAPIInfo represents the API information.

type OpenAPIInfo struct {
	Title string `json:"title"`

	Description string `json:"description"`

	Version string `json:"version"`

	Contact *OpenAPIContact `json:"contact,omitempty"`

	License *OpenAPILicense `json:"license,omitempty"`

	TermsOfService string `json:"termsOfService,omitempty"`
}

// OpenAPIContact represents contact information.

type OpenAPIContact struct {
	Name string `json:"name,omitempty"`

	URL string `json:"url,omitempty"`

	Email string `json:"email,omitempty"`
}

// OpenAPILicense represents license information.

type OpenAPILicense struct {
	Name string `json:"name"`

	URL string `json:"url,omitempty"`
}

// OpenAPIServer represents a server.

type OpenAPIServer struct {
	URL string `json:"url"`

	Description string `json:"description,omitempty"`

	Variables map[string]*OpenAPIServerVariable `json:"variables,omitempty"`
}

// OpenAPIServerVariable represents a server variable.

type OpenAPIServerVariable struct {
	Enum []string `json:"enum,omitempty"`

	Default string `json:"default"`

	Description string `json:"description,omitempty"`
}

// OpenAPIPath represents a path item.

type OpenAPIPath struct {
	Get *OpenAPIOperation `json:"get,omitempty"`

	Post *OpenAPIOperation `json:"post,omitempty"`

	Put *OpenAPIOperation `json:"put,omitempty"`

	Delete *OpenAPIOperation `json:"delete,omitempty"`

	Patch *OpenAPIOperation `json:"patch,omitempty"`
}

// OpenAPIOperation represents an operation.

type OpenAPIOperation struct {
	Tags []string `json:"tags,omitempty"`

	Summary string `json:"summary,omitempty"`

	Description string `json:"description,omitempty"`

	OperationID string `json:"operationId,omitempty"`

	Parameters []*OpenAPIParameter `json:"parameters,omitempty"`

	RequestBody *OpenAPIRequestBody `json:"requestBody,omitempty"`

	Responses map[string]*OpenAPIResponse `json:"responses"`

	Security []map[string][]string `json:"security,omitempty"`

	Deprecated bool `json:"deprecated,omitempty"`
}

// OpenAPIParameter represents a parameter.

type OpenAPIParameter struct {
	Name string `json:"name"`

	In string `json:"in"` // query, header, path, cookie

	Description string `json:"description,omitempty"`

	Required bool `json:"required,omitempty"`

	Schema *OpenAPISchema `json:"schema,omitempty"`

	Example interface{} `json:"example,omitempty"`
}

// OpenAPIRequestBody represents a request body.

type OpenAPIRequestBody struct {
	Description string `json:"description,omitempty"`

	Content map[string]*OpenAPIContent `json:"content"`

	Required bool `json:"required,omitempty"`
}

// OpenAPIResponse represents a response.

type OpenAPIResponse struct {
	Description string `json:"description"`

	Headers map[string]*OpenAPIHeader `json:"headers,omitempty"`

	Content map[string]*OpenAPIContent `json:"content,omitempty"`
}

// OpenAPIContent represents content.

type OpenAPIContent struct {
	Schema *OpenAPISchema `json:"schema,omitempty"`

	Example interface{} `json:"example,omitempty"`

	Examples map[string]*OpenAPIExample `json:"examples,omitempty"`
}

// OpenAPIExample represents an example.

type OpenAPIExample struct {
	Summary string `json:"summary,omitempty"`

	Description string `json:"description,omitempty"`

	Value interface{} `json:"value,omitempty"`
}

// OpenAPIHeader represents a header.

type OpenAPIHeader struct {
	Description string `json:"description,omitempty"`

	Required bool `json:"required,omitempty"`

	Schema *OpenAPISchema `json:"schema,omitempty"`
}

// OpenAPISchema represents a schema.

type OpenAPISchema struct {
	Type string `json:"type,omitempty"`

	Format string `json:"format,omitempty"`

	Title string `json:"title,omitempty"`

	Description string `json:"description,omitempty"`

	Default interface{} `json:"default,omitempty"`

	Example interface{} `json:"example,omitempty"`

	Enum []interface{} `json:"enum,omitempty"`

	Items *OpenAPISchema `json:"items,omitempty"`

	Properties map[string]*OpenAPISchema `json:"properties,omitempty"`

	Required []string `json:"required,omitempty"`

	AdditionalProperties interface{} `json:"additionalProperties,omitempty"`

	AllOf []*OpenAPISchema `json:"allOf,omitempty"`

	OneOf []*OpenAPISchema `json:"oneOf,omitempty"`

	AnyOf []*OpenAPISchema `json:"anyOf,omitempty"`

	Not *OpenAPISchema `json:"not,omitempty"`

	Ref string `json:"$ref,omitempty"`

	Minimum *float64 `json:"minimum,omitempty"`

	Maximum *float64 `json:"maximum,omitempty"`

	MinLength *int `json:"minLength,omitempty"`

	MaxLength *int `json:"maxLength,omitempty"`

	Pattern string `json:"pattern,omitempty"`
}

// OpenAPIComponents represents components.

type OpenAPIComponents struct {
	Schemas map[string]*OpenAPISchema `json:"schemas,omitempty"`

	Responses map[string]*OpenAPIResponse `json:"responses,omitempty"`

	Parameters map[string]*OpenAPIParameter `json:"parameters,omitempty"`

	RequestBodies map[string]*OpenAPIRequestBody `json:"requestBodies,omitempty"`

	Headers map[string]*OpenAPIHeader `json:"headers,omitempty"`

	SecuritySchemes map[string]*OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

// OpenAPISecurityScheme represents a security scheme.

type OpenAPISecurityScheme struct {
	Type string `json:"type"`

	Description string `json:"description,omitempty"`

	Name string `json:"name,omitempty"`

	In string `json:"in,omitempty"`

	Scheme string `json:"scheme,omitempty"`

	BearerFormat string `json:"bearerFormat,omitempty"`

	Flows *OpenAPIOAuthFlows `json:"flows,omitempty"`

	OpenIDConnectURL string `json:"openIdConnectUrl,omitempty"`
}

// OpenAPIOAuthFlows represents OAuth flows.

type OpenAPIOAuthFlows struct {
	Implicit *OpenAPIOAuthFlow `json:"implicit,omitempty"`

	Password *OpenAPIOAuthFlow `json:"password,omitempty"`

	ClientCredentials *OpenAPIOAuthFlow `json:"clientCredentials,omitempty"`

	AuthorizationCode *OpenAPIOAuthFlow `json:"authorizationCode,omitempty"`
}

// OpenAPIOAuthFlow represents an OAuth flow.

type OpenAPIOAuthFlow struct {
	AuthorizationURL string `json:"authorizationUrl,omitempty"`

	TokenURL string `json:"tokenUrl,omitempty"`

	RefreshURL string `json:"refreshUrl,omitempty"`

	Scopes map[string]string `json:"scopes,omitempty"`
}

// OpenAPITag represents a tag.

type OpenAPITag struct {
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	ExternalDocs *OpenAPIExternalDocs `json:"externalDocs,omitempty"`
}

// OpenAPIExternalDocs represents external documentation.

type OpenAPIExternalDocs struct {
	Description string `json:"description,omitempty"`

	URL string `json:"url"`
}

// getOpenAPISpec handles GET /openapi.json.

func (s *NephoranAPIServer) getOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := s.generateOpenAPISpec()

	s.writeJSONResponse(w, http.StatusOK, spec)
}

// getAPIDocs handles GET /docs.

func (s *NephoranAPIServer) getAPIDocs(w http.ResponseWriter, r *http.Request) {
	// Serve Swagger UI HTML.

	html := `<!DOCTYPE html>

<html>

<head>

    <title>Nephoran Intent Operator API Documentation</title>

    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />

    <style>

        html {

            box-sizing: border-box;

            overflow: -moz-scrollbars-vertical;

            overflow-y: scroll;

        }

        *, *:before, *:after {

            box-sizing: inherit;

        }

        body {

            margin: 0;

            background: #fafafa;

        }

    </style>

</head>

<body>

    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>

    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>

    <script>

        window.onload = function() {

            const ui = SwaggerUIBundle({

                url: '/openapi.json',

                dom_id: '#swagger-ui',

                deepLinking: true,

                presets: [

                    SwaggerUIBundle.presets.apis,

                    SwaggerUIStandalonePreset

                ],

                plugins: [

                    SwaggerUIBundle.plugins.DownloadUrl

                ],

                layout: "StandaloneLayout"

            });

        };

    </script>

</body>

</html>`

	w.Header().Set("Content-Type", "text/html")

	w.WriteHeader(http.StatusOK)

	w.Write([]byte(html))
}

// generateOpenAPISpec generates the complete OpenAPI 3.0 specification.

func (s *NephoranAPIServer) generateOpenAPISpec() *OpenAPISpec {
	return &OpenAPISpec{
		OpenAPI: "3.0.3",

		Info: &OpenAPIInfo{
			Title: "Nephoran Intent Operator API",

			Description: "Comprehensive REST API for the Nephoran Intent Operator providing intent-driven network function management with real-time streaming capabilities.",

			Version: "1.0.0",

			Contact: &OpenAPIContact{
				Name: "Nephoran Team",

				Email: "support@nephoran.io",
			},

			License: &OpenAPILicense{
				Name: "Apache 2.0",

				URL: "http://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},

		Servers: []*OpenAPIServer{
			{
				URL: "https://api.nephoran.io/api/v1",

				Description: "Production server",
			},

			{
				URL: "https://staging-api.nephoran.io/api/v1",

				Description: "Staging server",
			},

			{
				URL: "http://localhost:8080/api/v1",

				Description: "Development server",
			},
		},

		Paths: s.generatePaths(),

		Components: s.generateComponents(),

		Security: []map[string][]string{
			{"BearerAuth": []string{}},

			{"OAuth2": []string{"read", "write"}},
		},

		Tags: []*OpenAPITag{
			{
				Name: "Intents",

				Description: "Intent management operations",
			},

			{
				Name: "Packages",

				Description: "Package lifecycle management",
			},

			{
				Name: "Clusters",

				Description: "Multi-cluster operations",
			},

			{
				Name: "Realtime",

				Description: "Real-time streaming and WebSocket endpoints",
			},

			{
				Name: "Dashboard",

				Description: "Dashboard and metrics endpoints",
			},

			{
				Name: "System",

				Description: "System management and health endpoints",
			},
		},
	}
}

// generatePaths generates all API paths.

func (s *NephoranAPIServer) generatePaths() map[string]*OpenAPIPath {
	paths := make(map[string]*OpenAPIPath)

	// Intent endpoints.

	paths["/intents"] = &OpenAPIPath{
		Get: &OpenAPIOperation{
			Tags: []string{"Intents"},

			Summary: "List network intents",

			Description: "Retrieve a paginated list of network intents with filtering and sorting options",

			OperationID: "listIntents",

			Parameters: []*OpenAPIParameter{
				{
					Name: "page",

					In: "query",

					Description: "Page number for pagination",

					Schema: &OpenAPISchema{Type: "integer", Default: 1, Minimum: &[]float64{1}[0]},
				},

				{
					Name: "page_size",

					In: "query",

					Description: "Number of items per page",

					Schema: &OpenAPISchema{Type: "integer", Default: 20, Minimum: &[]float64{1}[0], Maximum: &[]float64{100}[0]},
				},

				{
					Name: "status",

					In: "query",

					Description: "Filter by intent status",

					Schema: &OpenAPISchema{Type: "string", Enum: []interface{}{"pending", "processing", "completed", "failed"}},
				},

				{
					Name: "type",

					In: "query",

					Description: "Filter by intent type",

					Schema: &OpenAPISchema{Type: "string", Enum: []interface{}{"deployment", "scaling", "optimization", "maintenance"}},
				},

				{
					Name: "priority",

					In: "query",

					Description: "Filter by priority level",

					Schema: &OpenAPISchema{Type: "string", Enum: []interface{}{"low", "medium", "high", "critical"}},
				},
			},

			Responses: map[string]*OpenAPIResponse{
				"200": {
					Description: "Successful response",

					Content: map[string]*OpenAPIContent{
						"application/json": {
							Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentListResponse"},
						},
					},
				},

				"400": {Description: "Bad request", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},

				"401": {Description: "Unauthorized", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},

				"500": {Description: "Internal server error", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
			},
		},

		Post: &OpenAPIOperation{
			Tags: []string{"Intents"},

			Summary: "Create a new network intent",

			Description: "Create a new network intent with natural language processing",

			OperationID: "createIntent",

			RequestBody: &OpenAPIRequestBody{
				Description: "Intent creation request",

				Required: true,

				Content: map[string]*OpenAPIContent{
					"application/json": {
						Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentRequest"},
					},
				},
			},

			Responses: map[string]*OpenAPIResponse{
				"201": {
					Description: "Intent created successfully",

					Content: map[string]*OpenAPIContent{
						"application/json": {
							Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentResponse"},
						},
					},
				},

				"400": {Description: "Bad request", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},

				"401": {Description: "Unauthorized", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},

				"409": {Description: "Conflict", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},

				"500": {Description: "Internal server error", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
			},
		},
	}

	paths["/intents/{name}"] = &OpenAPIPath{
		Get: &OpenAPIOperation{
			Tags: []string{"Intents"},

			Summary: "Get a specific network intent",

			Description: "Retrieve detailed information about a specific network intent",

			OperationID: "getIntent",

			Parameters: []*OpenAPIParameter{
				{
					Name: "name",

					In: "path",

					Description: "Intent name",

					Required: true,

					Schema: &OpenAPISchema{Type: "string"},
				},

				{
					Name: "namespace",

					In: "query",

					Description: "Kubernetes namespace",

					Schema: &OpenAPISchema{Type: "string", Default: "default"},
				},
			},

			Responses: map[string]*OpenAPIResponse{
				"200": {
					Description: "Successful response",

					Content: map[string]*OpenAPIContent{
						"application/json": {
							Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentResponse"},
						},
					},
				},

				"404": {Description: "Intent not found", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
			},
		},

		Put: &OpenAPIOperation{
			Tags: []string{"Intents"},

			Summary: "Update a network intent",

			Description: "Update an existing network intent",

			OperationID: "updateIntent",

			Parameters: []*OpenAPIParameter{
				{
					Name: "name",

					In: "path",

					Description: "Intent name",

					Required: true,

					Schema: &OpenAPISchema{Type: "string"},
				},
			},

			RequestBody: &OpenAPIRequestBody{
				Description: "Intent update request",

				Required: true,

				Content: map[string]*OpenAPIContent{
					"application/json": {
						Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentRequest"},
					},
				},
			},

			Responses: map[string]*OpenAPIResponse{
				"200": {
					Description: "Intent updated successfully",

					Content: map[string]*OpenAPIContent{
						"application/json": {
							Schema: &OpenAPISchema{Ref: "#/components/schemas/IntentResponse"},
						},
					},
				},

				"404": {Description: "Intent not found", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
			},
		},

		Delete: &OpenAPIOperation{
			Tags: []string{"Intents"},

			Summary: "Delete a network intent",

			Description: "Delete an existing network intent",

			OperationID: "deleteIntent",

			Parameters: []*OpenAPIParameter{
				{
					Name: "name",

					In: "path",

					Description: "Intent name",

					Required: true,

					Schema: &OpenAPISchema{Type: "string"},
				},
			},

			Responses: map[string]*OpenAPIResponse{
				"204": {Description: "Intent deleted successfully"},

				"404": {Description: "Intent not found", Content: map[string]*OpenAPIContent{"application/json": {Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
			},
		},
	}

	// Add more paths for packages, clusters, realtime, dashboard, etc.

	// This is a condensed example showing the pattern.

	return paths
}

// generateComponents generates the OpenAPI components.

func (s *NephoranAPIServer) generateComponents() *OpenAPIComponents {
	return &OpenAPIComponents{
		Schemas: map[string]*OpenAPISchema{
			"IntentRequest": {
				Type: "object",

				Description: "Network intent creation request",

				Required: []string{"name", "intent", "intent_type"},

				Properties: map[string]*OpenAPISchema{
					"name": {
						Type: "string",

						Description: "Intent name",

						Example: "deploy-amf-production",
					},

					"namespace": {
						Type: "string",

						Description: "Kubernetes namespace",

						Default: "default",
					},

					"intent": {
						Type: "string",

						Description: "Natural language intent description",

						MinLength: &[]int{10}[0],

						MaxLength: &[]int{2000}[0],

						Example: "Deploy a high-availability AMF instance for production with auto-scaling",
					},

					"intent_type": {
						Type: "string",

						Description: "Type of intent operation",

						Enum: []interface{}{"deployment", "scaling", "optimization", "maintenance"},

						Example: "deployment",
					},

					"priority": {
						Type: "string",

						Description: "Priority level",

						Enum: []interface{}{"low", "medium", "high", "critical"},

						Default: "medium",
					},

					"target_components": {
						Type: "array",

						Items: &OpenAPISchema{
							Type: "string",

							Enum: []interface{}{"AMF", "SMF", "UPF", "gNodeB", "O-DU", "O-CU-CP", "O-CU-UP"},
						},

						Description: "Target network function components",
					},
				},
			},

			"IntentResponse": {
				Type: "object",

				Description: "Network intent response with detailed information",

				Properties: map[string]*OpenAPISchema{
					"metadata": {
						Type: "object",

						Description: "Kubernetes metadata",
					},

					"spec": {
						Type: "object",

						Description: "Intent specification",
					},

					"status": {
						Type: "object",

						Description: "Intent status",
					},

					"processing_metrics": {
						Ref: "#/components/schemas/ProcessingMetrics",
					},
				},
			},

			"IntentListResponse": {
				Type: "object",

				Description: "Paginated list of network intents",

				Properties: map[string]*OpenAPISchema{
					"success": {
						Type: "boolean",

						Example: true,
					},

					"data": {
						Type: "object",

						Properties: map[string]*OpenAPISchema{
							"items": {
								Type: "array",

								Items: &OpenAPISchema{
									Ref: "#/components/schemas/IntentResponse",
								},
							},

							"meta": {
								Ref: "#/components/schemas/PaginationMeta",
							},

							"links": {
								Ref: "#/components/schemas/HATEOASLinks",
							},
						},
					},

					"timestamp": {
						Type: "string",

						Format: "date-time",
					},
				},
			},

			"ProcessingMetrics": {
				Type: "object",

				Description: "Intent processing performance metrics",

				Properties: map[string]*OpenAPISchema{
					"processing_duration": {
						Type: "string",

						Description: "Total processing time",

						Example: "4.5s",
					},

					"deployment_duration": {
						Type: "string",

						Description: "Deployment time",

						Example: "12.3s",
					},

					"total_duration": {
						Type: "string",

						Description: "Total time from creation to completion",

						Example: "16.8s",
					},
				},
			},

			"PaginationMeta": {
				Type: "object",

				Description: "Pagination metadata",

				Properties: map[string]*OpenAPISchema{
					"page": {
						Type: "integer",

						Example: 1,
					},

					"page_size": {
						Type: "integer",

						Example: 20,
					},

					"total_pages": {
						Type: "integer",

						Example: 5,
					},

					"total_items": {
						Type: "integer",

						Example: 87,
					},
				},
			},

			"HATEOASLinks": {
				Type: "object",

				Description: "HATEOAS navigation links",

				Properties: map[string]*OpenAPISchema{
					"self": {
						Type: "string",

						Example: "/api/v1/intents?page=1",
					},

					"first": {
						Type: "string",

						Example: "/api/v1/intents?page=1",
					},

					"last": {
						Type: "string",

						Example: "/api/v1/intents?page=5",
					},

					"next": {
						Type: "string",

						Example: "/api/v1/intents?page=2",
					},

					"previous": {
						Type: "string",

						Example: "/api/v1/intents?page=0",
					},
				},
			},

			"ErrorResponse": {
				Type: "object",

				Description: "Standard error response",

				Properties: map[string]*OpenAPISchema{
					"success": {
						Type: "boolean",

						Example: false,
					},

					"error": {
						Type: "object",

						Properties: map[string]*OpenAPISchema{
							"code": {
								Type: "string",

								Example: "validation_failed",
							},

							"message": {
								Type: "string",

								Example: "Intent validation failed",
							},

							"details": {
								Type: "object",

								Description: "Additional error details",
							},
						},
					},

					"timestamp": {
						Type: "string",

						Format: "date-time",
					},
				},
			},
		},

		SecuritySchemes: map[string]*OpenAPISecurityScheme{
			"BearerAuth": {
				Type: "http",

				Scheme: "bearer",

				BearerFormat: "JWT",

				Description: "JWT Bearer token authentication",
			},

			"OAuth2": {
				Type: "oauth2",

				Description: "OAuth2 authentication",

				Flows: &OpenAPIOAuthFlows{
					AuthorizationCode: &OpenAPIOAuthFlow{
						AuthorizationURL: "https://auth.nephoran.io/oauth/authorize",

						TokenURL: "https://auth.nephoran.io/oauth/token",

						Scopes: map[string]string{
							"read": "Read access to resources",

							"write": "Write access to resources",
						},
					},
				},
			},
		},

		Responses: map[string]*OpenAPIResponse{
			"UnauthorizedError": {
				Description: "Authentication information is missing or invalid",

				Content: map[string]*OpenAPIContent{
					"application/json": {
						Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
					},
				},
			},

			"ForbiddenError": {
				Description: "Access denied due to insufficient permissions",

				Content: map[string]*OpenAPIContent{
					"application/json": {
						Schema: &OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"},
					},
				},
			},
		},
	}
}

// Helper function to generate parameter references.

func queryParam(name, description string, schema *OpenAPISchema) *OpenAPIParameter {
	return &OpenAPIParameter{
		Name: name,

		In: "query",

		Description: description,

		Schema: schema,
	}
}

func pathParam(name, description string) *OpenAPIParameter {
	return &OpenAPIParameter{
		Name: name,

		In: "path",

		Description: description,

		Required: true,

		Schema: &OpenAPISchema{Type: "string"},
	}
}
