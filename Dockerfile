FROM golang:1.25.0-alpine AS builder

RUN apk add --no-cache git make
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/goserver ./cmd/main.go

FROM alpine:3.21 AS runtime
 
RUN addgroup -S appuser && adduser -S appuser -G appuser

WORKDIR /app

COPY --from=builder /app/bin/server /app/server

RUN chown -R appuser:appuser /app
USER appuser

FROM runtime AS server
ENTRYPOINT ["./goserver"]