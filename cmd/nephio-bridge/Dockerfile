# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /nephio-bridge cmd/nephio-bridge/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /nephio-bridge /nephio-bridge
EXPOSE 8081
ENTRYPOINT ["/nephio-bridge"]
