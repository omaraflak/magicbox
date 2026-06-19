# Stage 1: Build the React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build the Go binary
FROM golang:alpine AS builder

# Install build dependencies for cgo (sqlite3 requires cgo)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the main server binary with cgo enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o magicbox cmd/server/main.go

# Stage 3: Create the runner image
FROM alpine:3.19

# Install runtime dependencies (like ca-certificates)
RUN apk add --no-cache ca-certificates tzdata

# Create directory structure for Magicbox
RUN mkdir -p /opt/magicbox/core/web \
             /opt/magicbox/backups \
             /opt/magicbox/transit \
             /opt/magicbox/users

# Copy the compiled binary
COPY --from=builder /app/magicbox /usr/local/bin/magicbox

# Copy the static web frontend assets from the frontend builder stage
COPY --from=frontend-builder /app/web/dist/ /app/web/

# Set defaults
ENV MAGICBOX_ROOT=/opt/magicbox
ENV MAGICBOX_PORT=80

EXPOSE 80 50051

ENTRYPOINT ["/usr/local/bin/magicbox"]
