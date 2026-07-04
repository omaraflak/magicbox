# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY apps/drive/web/package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm install
COPY apps/drive/web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /magicbox/apps/drive
# Create a stub go.mod for the replaced core module to avoid downloading unused core dependencies
RUN mkdir -p /magicbox && echo -e "module github.com/magicbox/core\n\ngo 1.23.0" > /magicbox/go.mod
# Copy shared proto package to /magicbox/api/proto/v1
COPY api/proto/v1/ /magicbox/api/proto/v1/
# Copy shared sdk package to /magicbox/sdk
COPY sdk/ /magicbox/sdk/
# Copy module configuration
COPY apps/drive/go.mod apps/drive/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
# Copy local source file
COPY apps/drive/*.go ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o drive .

# Stage 3: Runtime
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /magicbox/apps/drive/drive /drive
COPY --from=frontend-builder /app/web/dist/ /web/
RUN mkdir -p /data/app_state /data/transit /data/shared/photos /data/shared/documents
EXPOSE 9090
ENTRYPOINT ["/drive"]
