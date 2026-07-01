# Magicbox App Development Guide

This document describes how to build, configure, and integrate a new application into the Magicbox OS container stack. This guide is optimized for direct consumption by coding AI agents.

---

## 1. App Manifest (`magicbox.json`)
Every application must include a `magicbox.json` file in its root directory defining its capabilities, routing, and access requirements. 

### Manifest Schema & Validation Rules
```json
{
  "app_id": "com.magicbox.myapp",
  "name": "My App",
  "version": "1.0.0",
  "image": "docker.io/omaraflak/magicbox-myapp:latest",
  "entry_port": 9090,
  "route_slug": "myapp",
  "webhook_path": "/internal/magicbox-webhook",
  "required_scopes": [
    "profile:read",
    "shared:storage:rw"
  ],
  "volume_mounts": [
    {
      "type": "shared",
      "name": "storage",
      "access": "read-write"
    }
  ],
  "resource_limits": {
    "memory_mb": 256,
    "cpu_cores": 0.5
  }
}
```

* **`app_id`** (Required): Reverse-DNS format with at least 3 segments (e.g. `com.magicbox.myapp`). Regex: `^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*){2,}$`.
* **`name`** (Required): Human-readable display name (1-64 characters).
* **`version`** (Required): Strictly formatted SemVer string (e.g., `1.0.0`). Regex: `^\d+\.\d+\.\d+$`.
* **`image`** (Required): Fully qualified Docker image reference.
* **`entry_port`** (Required): The internal port of the web server running inside the container (e.g., `9090`).
* **`route_slug`** (Required): Subpath slug for routing. Must be 2-32 lowercase alphanumeric characters or hyphens, and cannot be a reserved OS path (`api`, `admin`, `setup`, `auth`, `static`, `health`, `u`). Regex: `^[a-z0-9][a-z0-9-]{0,30}[a-z0-9]$`.
* **`webhook_path`** (Optional): Path to the app's POST webhook endpoint. Defaults to `/internal/magicbox-webhook`. Must start with `/`.
* **`required_scopes`** (Required): List of capabilities. Allowed scopes:
  * `profile:read`: Allows calling `GetProfile`.
  * `contacts:read`: Allows calling `ListContacts`.
  * `shared:<volume_name>:<ro|rw>`: Grants access to shared filesystem volume `<volume_name>`.
* **`volume_mounts`** (Optional): Describes shared filesystem volumes.
  * `type` must be `"shared"`.
  * `name` must be lowercase alphanumeric with hyphens.
  * `access` must be `"read-only"` or `"read-write"`.
  * **Critical Check**: Any volume name used in `volume_mounts` must have a corresponding `shared:<name>:ro` (for read-only) or `shared:<name>:rw` (for read-write) scope in `required_scopes`.
* **`resource_limits`** (Optional): Specifies memory (1-4096 MB) and CPU (0.1-4.0 cores) limits. Defaults to 256 MB memory and 0.5 CPU cores.

---

## 2. Container Lifecycle & Environment

### Injected Environment Variables
The Magicbox OS gateway automatically injects these environment variables when starting your container:
* **`MAGICBOX_API_TOKEN`**: A JWT token signed by the host OS encoding the owning `user_id`, `app_id`, and `required_scopes`. Must be supplied as a Bearer token in all gRPC calls back to the host OS.
* **`MAGICBOX_CORE_URL`**: The internal gRPC address of the host OS gateway (e.g., `magicbox_core:50051`).
* **`MAGICBOX_USER_ID`**: The unique identifier of the user who owns this app instance.
* **`MAGICBOX_APP_ID`**: The unique application identifier (as defined in `magicbox.json`).

### Volume Mounts
The host OS mounts directories at predefined filesystem paths inside your container:
* **Private State Directory** (`/data/app_state`): Mounted `rw` (read-write). Unique and isolated for each instance of your app. Use this directory to persist local state (e.g., SQLite databases, private files).
* **Transit Directory** (`/data/transit`): Mounted `rw` (read-write). Shared across all apps of all users. Use this as a temporary staging folder to copy files when transferring data between apps.
* **Shared Volumes** (`/data/shared/<volume_name>`): Mounted based on `volume_mounts` in `magicbox.json` (read-write `:rw` or read-only `:ro` based on configuration).

---

## 3. Web Proxy & HTML Hosting
App containers are not directly exposed to the host machine or public internet. Web traffic is routed through Traefik and the Core OS reverse-proxy gateway.

### Proxy Pathing & Prefix
A user accesses your app's frontend via `/u/{username}/{route_slug}/`.
* The gateway handles user login and session verification, then reverse-proxies the request to your container's `entry_port`.
* **Prefix Stripping**: The gateway strips `/u/{username}/{route_slug}` from the request path before forwarding it. E.g., `/u/omar/drive/api/files` is received by the container as `/api/files`.
* **Headers Injected**:
  * `X-Forwarded-Prefix`: `/u/{username}/{route_slug}`
  * `X-Original-URI`: `/u/{username}/{route_slug}/api/files`

### Proxied HTML & Asset Routing (Base Tag)
If your app hosts a frontend (whether an SPA or a traditional multi-page site), the browser needs to resolve assets relative to the proxy subpath.
To support this, your server must inspect `X-Forwarded-Prefix` and inject a `<base href="...">` tag into the `<head>` of your HTML responses.

If building a Go-based application, you can use the pre-built `HTMLHandler` from the **`github.com/magicbox/core/sdk`** package to automatically handle this static asset lookup and base tag injection:

```go
import "github.com/magicbox/core/sdk"

// Register the HTML fallback handler for asset routing
mux.Handle("/", sdk.NewHTMLHandler("/web"))
```

---

## 4. Exposed Core OS gRPC Services
To communicate with the host OS, dial `MAGICBOX_CORE_URL` using insecure gRPC credentials and attach `MAGICBOX_API_TOKEN` as a metadata header.

### Protobuf Definition (`api/proto/v1/magicbox.proto`)
The available RPC methods are:

```protobuf
service MagicboxOS {
  // SendWebhook dispatches a payload to another app's webhook endpoint.
  rpc SendWebhook(SendWebhookRequest) returns (SendWebhookResponse);

  // GetProfile returns the owning user's profile information.
  // Requires scope: profile:read
  rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);

  // ListSharedVolumes returns the shared volumes accessible to the calling app.
  rpc ListSharedVolumes(ListSharedVolumesRequest) returns (ListSharedVolumesResponse);

  // SendToContact signs, encrypts, and dispatches payload to federated contact over libp2p.
  rpc SendToContact(SendToContactRequest) returns (SendToContactResponse);

  // ListContacts returns the owning user's contact list.
  // Requires scope: contacts:read
  rpc ListContacts(ListContactsRequest) returns (ListContactsResponse);
}
```

### Implementing gRPC Authorization Metadata (Go Example)
```go
import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	pb "github.com/magicbox/core/api/proto/v1"
)

func CallGetProfile() (*pb.GetProfileResponse, error) {
	conn, err := grpc.Dial(os.Getenv("MAGICBOX_CORE_URL"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewMagicboxOSClient(conn)
	
	// Inject token into outgoing metadata context
	token := os.Getenv("MAGICBOX_API_TOKEN")
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer " + token))

	return client.GetProfile(ctx, &pb.GetProfileRequest{})
}
```

---

## 5. Webhook Receiver
Inter-Container Communication (ICC) is disabled by default on the virtual docker network. Apps can communicate with each other (locally) or receive remote peer payloads (federated network) by sending/receiving webhooks.

### Webhook HTTP Specification
Your app container must expose a POST route at the `webhook_path` defined in `magicbox.json`. When a payload is dispatched to your webhook, the container receives a POST request:
* **Headers**:
  * `Content-Type`: `application/octet-stream`
  * `X-Magicbox-Source-App`: The ID of the sending app (or `"p2p-gateway"` for federated messages).
  * `X-Magicbox-Source-User`: The User ID of the sender (or the sender's libp2p Peer ID string if from the federated network).
  * `X-Magicbox-Source-Type`: `"local"` (for local communication on the same device) or `"remote"` (incoming payload from the federated libp2p network).
* **Body**: The raw bytes payload.

---

## 6. Checklist for New App Creation
1. Write backend code exposing HTTP port `9090` (or custom `entry_port`).
2. Implement HTML response handler injecting the `<base href>` tag from `X-Forwarded-Prefix`.
3. Expose the webhook POST handler at `webhook_path` (defaults to `/internal/magicbox-webhook`).
4. Read environment variables (`MAGICBOX_API_TOKEN` and `MAGICBOX_CORE_URL`) to build your host connection client.
5. Create `magicbox.json` manifest file inside the app folder.
6. Package the app as a Docker image and host it on an accessible registry.
