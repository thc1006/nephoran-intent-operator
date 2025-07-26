# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The build will fail here until the environment is fixed, but the file is correct.
RUN CGO_ENABLED=0 GOOS=linux go build -o /llm-processor cmd/llm-processor/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /llm-processor /llm-processor
EXPOSE 8080
ENTRYPOINT ["/llm-processor"]
