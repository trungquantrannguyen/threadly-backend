FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/notification-service \
    ./services/notification-service/cmd/server

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S threadly \
    && adduser -S threadly -G threadly

COPY --from=builder /app/bin/notification-service /app/notification-service

USER threadly

EXPOSE 50054

ENTRYPOINT ["/app/notification-service"]
