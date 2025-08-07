# Nephoran Intent Operator - LLM Processor API Documentation

## Overview

This directory contains the OpenAPI specification and interactive documentation for the Nephoran Intent Operator's LLM Processor Service.

## Components

- `llm-processor-openapi.yaml`: Complete OpenAPI 3.0.3 specification
- `swagger-ui-integration.html`: Interactive API documentation using Swagger UI
- `README.md`: This documentation file

## Getting Started

### Prerequisites

- Modern web browser
- Basic understanding of RESTful APIs
- Valid authentication token for live testing

### Using Swagger UI

1. Open `swagger-ui-integration.html` in a web browser
2. Explore available endpoints
3. Click "Try it out" to test API methods
4. Replace `YOUR_TOKEN_HERE` with a valid JWT token

### Authentication

All endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <your_jwt_token>
```

### API Endpoints

#### Intent Processing
- `POST /process-intent`: Process natural language intents
- `GET /stream-intent`: Stream real-time intent processing
- `POST /validate-intent`: Validate intent feasibility

#### System Endpoints
- `GET /metrics`: Retrieve processing metrics
- `GET /health`: Check service health status

## Security Considerations

- Always use HTTPS in production
- Protect your JWT tokens
- Implement proper token management
- Use rate limiting to prevent abuse

## Support

For issues or questions, contact support@nephoran.io

## License

Apache 2.0 License - See LICENSE file in project root