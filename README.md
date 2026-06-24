# Magicbox OS

## Prerequisites
* [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## Run

1. **Build Images**:
   ```bash
   docker build -t docker.io/magicbox/core:latest .
   docker build -t docker.io/magicbox/drive:latest -f apps/drive/Dockerfile .
   ```

2. **Start Stack**:

**Linux**:

```bash
docker compose up -d
```

**macOS**:
```bash
MAGICBOX_HOST_ROOT=/tmp/magicbox docker compose up -d
```

Open `http://localhost:8081` once the containers are running.
