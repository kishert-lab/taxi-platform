FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/taxi-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/taxi-admin ./cmd/admin


FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 appuser

COPY --from=builder /out/taxi-api /app/taxi-api
COPY --from=builder /out/taxi-admin /app-admin
COPY configs /app/configs

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/taxi-api"]
