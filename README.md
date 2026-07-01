# Magicbox OS

## Prerequisites
* [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## Run

### 1. Build Images

```bash
docker build -t docker.io/omaraflak/magicbox-core:latest .
docker build -t docker.io/omaraflak/magicbox-drive:latest -f apps/drive/Dockerfile .
docker build -t docker.io/omaraflak/magicbox-ai:latest -f apps/ai/Dockerfile .
```

### 2. Start Stack

**Linux**:
```bash
docker compose up -d
```

**macOS**:
```bash
MAGICBOX_HOST_ROOT=/tmp/magicbox docker compose up -d
```

Open `http://localhost:9090` (Core Dashboard), `http://localhost:9090/drive` (Drive) or `http://localhost:9090/ai` (AI) once the containers are running.
