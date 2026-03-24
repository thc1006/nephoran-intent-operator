FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scaling-xapp .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /home/nobody
COPY --from=builder /app/scaling-xapp .

USER 65534
EXPOSE 2112
CMD ["./scaling-xapp"]
