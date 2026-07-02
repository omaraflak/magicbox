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


## Cryptography and Key Management

Magicbox uses standard asymmetric cryptographic primitives to manage user identities, secure peer-to-peer data transport, and encrypt message payloads. All operational keys are derived deterministically from a single master seed.

### 1. Deterministic Key Derivation
To simplify backup and recovery, Magicbox derives all keys from a single **12-word BIP-39 mnemonic phrase**.
* **Seed Generation**: The mnemonic phrase is converted into a 64-byte master seed.
* **Separation of Concerns**: We enforce strict isolation between Identity Keys (used for signatures and node discovery) and Operational Encryption Keys (used for message payloads).
* **Deterministic Derivation Paths**: Keys are derived by hashing the master seed concatenated with a virtual path structure:
  * **Ed25519 Identity Key (Index $i$)**: Derived via `SHA256(MasterSeed || "/ed25519/i")`
  * **X25519 Encryption Key (Index $j$)**: Derived via `SHA256(MasterSeed || "/x25519/j")`

This path separation allows independent rotation of encryption keys while keeping the user's network identity static.

### 2. Peer Identity (Ed25519)
* **Usage**: Used for signatures, message verification, and establishing P2P connections.
* **Peer ID**: The node's libp2p Peer ID is the cryptographic hash of the Ed25519 public key.
* **Connection Handshake**: Nodes identify and authenticate each other using their static Ed25519 keys during the libp2p secure transport handshake (Noise protocol).

### 3. Payload Encryption (Hybrid X25519 + AES-GCM)
Message payloads are encrypted end-to-end to protect user privacy.
* **Asymmetric Exchange**: Encryption keys are X25519 key pairs.
* **Symmetric Encryption**: Actual data payload encryption is performed symmetrically using **AES-256-GCM**.
* **One-Way Hybrid Protocol (No network handshake)**:
  1. Alice wants to send a message to Bob. She retrieves Bob's static X25519 public key from her database.
  2. Alice generates a temporary **ephemeral X25519 key pair**.
  3. Alice calculates a shared secret using Diffie-Hellman (ECDH) between her ephemeral private key and Bob's static public key.
  4. The shared secret is passed through a key derivation function to yield an AES-256 key.
  5. Alice encrypts the payload with AES-GCM and packages the ciphertext along with her ephemeral public key.
  6. Bob receives the package, performs ECDH between Alice's ephemeral public key and his static private key to derive the same AES key, and decrypts the payload.

### 4. Key Rotation & Recovery Workflows
The database keeps track of the active indices in the `system_settings` table under `identity_key_index` and `encryption_key_index`.

#### A. Routine Rotation (Encryption Key Only)
* **Trigger**: Initiated by the administrator to rotate encryption keys regularly.
* **Action**: Derives the next X25519 encryption key by incrementing `encryption_key_index` ($j \rightarrow j+1$). The Ed25519 identity remains untouched.
* **Propagation**: The server automatically broadcasts the new X25519 public key to all of the user's contacts using the secure P2P stream (`system:key-update` app ID).
* **Authentication**: The propagation packet is signed with the user's static Ed25519 private key. Because the identity key is unchanged, contacts verify the signature to authenticate the update and save the new key.
* **Activation**: A container restart is required to load the new key from disk into the memory cache.

#### B. Reset (Identity & Encryption Keys)
* **Trigger**: System reset or emergency recovery in case of private key compromise.
* **Action**: Generates a brand new mnemonic (or accepts a custom one) and resets both `identity_key_index` and `encryption_key_index` to `0`.
* **Consequences**: This resets the user's libp2p Peer ID. To the P2P network, the user is a completely new node. Existing contacts will see the user go offline.
* **Re-Authentication**: To restore connection, the user must manually distribute a new Invite Link (containing the new Peer ID and X25519 key) to all contacts.

### 5. Disk & Database Storage Rules
To prevent unauthorized access to sensitive cryptographic material:
* **Mnemonic**: The 12-word mnemonic is displayed *once* to the user during first boot. Once acknowledged, it is wiped from disk and memory.
* **Private Keys**: Stored in `/opt/magicbox/core/` (`identity.key` and `encryption.key`) with restricted read/write permissions (`0600`).
* **Key Indices**: Stored in the SQLite database `system_settings` table.
