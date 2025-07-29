# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /oran-adaptor cmd/oran-adaptor/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /oran-adaptor /oran-adaptor
EXPOSE 8081
ENTRYPOINT ["/oran-adaptor"]
