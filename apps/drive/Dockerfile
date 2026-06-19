# Stage 1: Build binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o drive main.go

# Stage 2: Create execution container
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/drive /drive

# Create data directories internally to ensure target paths exist
RUN mkdir -p /data/app_state /data/transit /data/shared/photos /data/shared/documents

EXPOSE 8080
ENTRYPOINT ["/drive"]
