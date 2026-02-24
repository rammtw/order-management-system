FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY gen/go.mod ./gen/go.mod
COPY apps/order/go.mod apps/order/go.sum ./apps/order/

WORKDIR /app/apps/order
RUN go mod download

WORKDIR /app
COPY gen/ ./gen/
COPY apps/order/ ./apps/order/

WORKDIR /app/apps/order
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/order-service ./cmd/

FROM alpine:3.21
COPY --from=builder /bin/order-service /bin/order-service
ENTRYPOINT ["/bin/order-service"]