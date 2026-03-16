FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

COPY . .

RUN GOTOOLCHAIN=auto CGO_ENABLED=0 go test -count=1 ./pkg/... ./internal/service/... ./internal/handler/... ./internal/middleware/...

RUN GOTOOLCHAIN=auto CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /server /server
COPY db/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/server"]
