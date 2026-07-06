# Magicbox OS

Magicbox OS is a decentralized personal cloud and app hosting platform.

## Prerequisites

* [Docker](https://docs.docker.com/get-docker/) (and Docker Compose for development)

## 🚀 Installation

If you just want to run Magicbox OS on your server or local machine without the source code, you can start it directly using the pre-built Docker image:

```bash
docker network create magicbox_net
docker run -d \
  --name magicbox_core \
  --restart unless-stopped \
  --network magicbox_net \
  -p 9090:80 \
  -p 4001:4001 \
  -p 4001:4001/udp \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/magicbox:/opt/magicbox \
  -e MAGICBOX_HOST_ROOT=/opt/magicbox \
  -e MAGICBOX_PORT=80 \
  omaraflak/magicbox-core:latest
```

* **`9090`**: Web Interface port. Once running, you can access the Magicbox Core dashboard at `http://localhost:9090`.
* **`4001` (TCP/UDP)**: P2P Communication port. Used for secure federated networking (libp2p) to connect and communicate with other Magicbox instances.

## 🛠️ Local Development

If you are developing Magicbox and want to build the stack from source:

1. Build the development images using a `:dev` tag:

```bash
docker build -t docker.io/omaraflak/magicbox-core:dev .
docker build -t docker.io/omaraflak/magicbox-drive:dev -f apps/drive/Dockerfile .
docker build -t docker.io/omaraflak/magicbox-ai:dev -f apps/ai/Dockerfile .
docker build -t docker.io/omaraflak/magicbox-chat:dev -f apps/chat/Dockerfile .
```

2. Start the development stack:

```bash
MAGICBOX_TAG=dev docker compose up -d
```

> [!TIP]
> **Developing a single app:** If you are only developing a specific app (e.g., Magic Chat) and want to use production versions for the rest of the stack, change the image tag to `:dev` in that app's `magicbox.json` manifest, build only that app's image with `:dev`, and leave the other manifests pointing to `:latest`. The development core server will run your local build for the app under development and pull the latest production images from Docker Hub for the others.

If you are on **macOS** you might need to do this instead to start the container:

```bash
MAGICBOX_HOST_ROOT=/tmp/magicbox MAGICBOX_TAG=dev docker compose up -d
```

Open `http://localhost:9090`.
