FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /identity-service \
    ./services/identity/cmd

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /identity-service /app/identity-service
COPY --from=builder /app/services/identity/migrations /app/services/identity/migrations

EXPOSE 8080

ENTRYPOINT ["/app/identity-service"]
