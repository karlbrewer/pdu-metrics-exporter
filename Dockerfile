# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /pdu-metrics-exporter ./...

# Final stage
FROM alpine:3.23.3
RUN apk add --no-cache ca-certificates
COPY --from=builder /pdu-metrics-exporter /pdu-metrics-exporter
RUN addgroup -S app && adduser -S -G app app
USER app
EXPOSE 8080
ENTRYPOINT ["/pdu-metrics-exporter"]

