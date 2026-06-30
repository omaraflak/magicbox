# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY apps/ai/web/package*.json ./
RUN npm install
COPY apps/ai/web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /magicbox/apps/ai
# Create a stub go.mod for the replaced core module to avoid downloading unused core dependencies
RUN mkdir -p /magicbox && echo -e "module github.com/magicbox/core\n\ngo 1.23.0" > /magicbox/go.mod
# Copy shared proto package to /magicbox/api/proto/v1
COPY api/proto/v1/ /magicbox/api/proto/v1/
# Copy shared sdk package to /magicbox/sdk
COPY sdk/ /magicbox/sdk/
# Copy module configuration
COPY apps/ai/go.mod apps/ai/go.sum ./
RUN go mod download
# Copy local source file
COPY apps/ai/*.go ./
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o ai .

# Stage 3: Runtime
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /magicbox/apps/ai/ai /ai
COPY --from=frontend-builder /app/web/dist/ /web/
RUN mkdir -p /data/app_state
EXPOSE 9090
ENTRYPOINT ["/ai"]
