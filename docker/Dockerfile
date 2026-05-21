# Multi-stage build for ChainPulse
# Build specific target: docker build --target=monolithic -f docker/Dockerfile .
# Build all: docker build --target=all -f docker/Dockerfile .

FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Monolithic binary
FROM builder AS monolithic-builder
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out/chainpulse ./cmd/monolithic/chainpulse

# Microservice binaries
FROM builder AS microservices-builder
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out/puller ./cmd/microservices/puller && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out/event-processor ./cmd/microservices/event-processor && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out/api-service ./cmd/microservices/api-service && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out/api-gateway ./cmd/microservices/api-gateway

FROM alpine:3.19 AS monolithic
RUN adduser -S -D -H appuser && apk --no-cache add ca-certificates curl
WORKDIR /home/appuser/
COPY --from=monolithic-builder /out/chainpulse .
USER appuser
EXPOSE 8080 50051 8081
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["./chainpulse"]

FROM alpine:3.19 AS puller
RUN adduser -S -D -H appuser && apk --no-cache add ca-certificates curl
WORKDIR /home/appuser/
COPY --from=microservices-builder /out/puller .
USER appuser
EXPOSE 8083
CMD ["./puller"]

FROM alpine:3.19 AS event-processor
RUN adduser -S -D -H appuser && apk --no-cache add ca-certificates curl
WORKDIR /home/appuser/
COPY --from=microservices-builder /out/event-processor .
USER appuser
EXPOSE 8082
CMD ["./event-processor"]

FROM alpine:3.19 AS api-service
RUN adduser -S -D -H appuser && apk --no-cache add ca-certificates curl
WORKDIR /home/appuser/
COPY --from=microservices-builder /out/api-service .
USER appuser
EXPOSE 8081
CMD ["./api-service"]

FROM alpine:3.19 AS api-gateway
RUN adduser -S -D -H appuser && apk --no-cache add ca-certificates curl
WORKDIR /home/appuser/
COPY --from=microservices-builder /out/api-gateway .
USER appuser
EXPOSE 8080
CMD ["./api-gateway"]

# Default target: monolithic (for backward compat)
FROM monolithic