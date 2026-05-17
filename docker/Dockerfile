# Multi-stage build for ChainPulse
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chainpulse ./cmd/monolithic/chainpulse

FROM alpine:3.19

RUN adduser -S -D -H appuser

WORKDIR /home/appuser/

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /app/chainpulse .

USER appuser

EXPOSE 8080 50051 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./chainpulse"]
