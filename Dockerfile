FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o balancer ./cmd/main.go

FROM gcr.io/distroless/static

WORKDIR /app

COPY --from=builder /app/balancer .

COPY --from=builder /app/migrations ./migrations

COPY --from=builder /app/.env .

COPY --from=builder /app/config-remote.json .

EXPOSE 8080

CMD ["/app/balancer", "--config", "config-remote.json"]