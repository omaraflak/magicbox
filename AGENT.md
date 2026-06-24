# Magicbox OS Core - Agent Guidelines

## Architecture Overview
Magicbox OS runs decentralized containerized applications (like Drive) locally per user, using Traefik as the ingress gateway and libp2p for peer-to-peer data synchronization.

```
                  ┌───────────────────────┐
                  │        Browser        │
                  └───────────┬───────────┘
                              │ HTTP
                              ▼
                  ┌───────────────────────┐
                  │    Traefik Gateway    │
                  └─────┬───────────┬─────┘
           /api         │           │ /u/{user}/{app}/
          (REST)        ▼           ▼ (Proxy)
                  ┌───────────┐  ┌───────────┐
                  │ Core Auth │  │ User App  │
                  │  Service  │  │ Container │
                  └─────┬─────┘  └─────┬─────┘
                        │              │ gRPC
                        │              ▼
                        │        ┌───────────┐
                        │        │  Core OS  │
                        └────────►   gRPC    │
                                 └─────┬─────┘
                                       │ libp2p
                                       ▼
                                 ┌───────────┐
                                 │ Peer Node │
                                 └───────────┘
```

## Directory Structure
* `api/proto/v1/`: Protobuf definition for the Magicbox OS gRPC service (inter-app APIs).
* `cmd/server/`: Main application entry point (`main.go`).
* `internal/config/`: Configuration structure and loading from env/JSON.
* `internal/core/`: Application orchestrator, container provisioning, manifest parsing, and permissions validation.
* `internal/crypto/`: Secure keys, hashing, and signature flows.
* `internal/db/`: SQLite database connectivity, migrations, and query execution.
* `internal/docker/`: Docker client wrapper for lifecycle management.
* `internal/p2p/`: Libp2p host implementation and network stream handlers.
* `internal/rest/`: REST API handlers split into modular domain concerns (`handlers_auth.go`, `handlers_apps.go`, `handlers_contacts.go`, `handlers_admin.go`).
* `internal/rpc/`: gRPC server handlers (`server.go`) for inter-app RPC requests.
* `web/`: React SPA source files.

## Where to Add Features
* **New REST API Endpoints**: Add to a domain handler file in `internal/rest/` (or create a new domain file `handlers_*.go` if introducing a new category). Register the path in `internal/rest/router.go`.
* **New Inter-App Service Functionality**: Add a RPC method definition to `api/proto/v1/core.proto`, regenerate the stubs, and implement the handler in `internal/rpc/server.go`.
* **New Database Operations**: Define SQL query methods in `internal/db/queries.go` and schema updates in `internal/db/migrations.go`.
* **New Container Provisioning/Volume Mounting Rules**: Modify orchestrator logic in `internal/core/orchestrator.go` or permissions validation in `internal/core/permissions.go`.

## Code Guidelines
1. **Modular Concerns**: Keep functions short and single-purpose. Factorize REST handlers into separate domain files rather than one giant file.
2. **Naming Conventions**: Use descriptive naming. Avoid abbreviated variable or parameter names (e.g. use `config` instead of `cfg`, `orchestrator` instead of `orch`), except for standard Go idiomatic shorthands (e.g., `db` for database connection, `w` for response writer, `r` for request).
3. **Testing Standards**:
   * Every component must have a corresponding test suite (`*_test.go`).
   * **Individual Test Functions**: Avoid wrapping multiple independent validation scenarios inside a single large flow test. Create separate, focused test functions (e.g. `TestGrpcGetProfile`, `TestGrpcListContacts`) to isolate failures.
   * **Mock External Services**: Mock network and external components (e.g. using `bufconn` for gRPC tests, mock interfaces for P2P transport) to keep unit tests fast and deterministic.

## Docker & Deployment Commands
When making Core OS changes, you must rebuild the image and recreate the container stack:

```bash
# 1. Rebuild the core docker image with updated code changes
docker build -t docker.io/magicbox/core:latest .

# 2. Recreate and restart the docker compose container stack to apply the new image
docker compose down && docker compose up -d
```

### Refreshing Dynamic App Containers
Do **not** manually delete or restart user app containers (e.g., `magicbox_app_omar_com.magicbox.drive`) using raw Docker CLI commands. This will desynchronize the Core database state and break reverse proxy routing.

Instead, trigger a clean rebuild and recreate action via the Core REST API:
```bash
POST /api/apps/{app_id}/rebuild
```


## Testing Commands
Run unit and integration tests from the workspace root:

```bash
# Run all tests in the project
go test ./...

# Run all tests in the project with verbose output
go test -v ./...

# Run tests under a specific package (e.g. internal/rest)
go test -v ./internal/rest

# Run a specific test case (e.g. TestGrpcGetProfile)
go test -v ./internal/rpc -run TestGrpcGetProfile
```


