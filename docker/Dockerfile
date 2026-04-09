# Multi-stage build for ChainPulse
FROM golang:1.20-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make curl

RUN curl -L https://foundry.paradigm.xyz | bash && \
    ln -sf /root/.foundry/bin/* /usr/local/bin/

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chainpulse ./cmd/monolithic/chainpulse

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates curl

RUN curl -L https://foundry.paradigm.xyz | bash && \
    ln -sf /root/.foundry/bin/* /usr/local/bin/

COPY --from=builder /app/chainpulse .

EXPOSE 8080 50051 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./chainpulse"]
