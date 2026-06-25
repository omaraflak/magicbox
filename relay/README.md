# Magicbox P2P Relay (Cloud Run / WebSockets)

A standalone Go implementation of a **libp2p Circuit Relay v2** server configured over WebSockets. Suitable for Google Cloud Run.

## Environment Variables

When deploying the container, you must configure the following environment variables:

| Variable | Required | Description |
| :--- | :--- | :--- |
| `PORT` | Yes (in Cloud Run) | The port the WebSocket server will listen on. Cloud Run injects this automatically (defaults to `8080`). |
| `RELAY_SEED` | **Highly Recommended** | A secret seed string used to deterministically generate the relay's private key. This ensures the relay maintains the **same Peer ID** across deployments and container restarts. |

---

## Critical Cloud Run Constraints

- **Set Minimum Instances to 1**:
  You must deploy the container with `--min-instances 1` so that Cloud Run does not scale down to zero when idle, which would drop active P2P sockets and client reservations.
