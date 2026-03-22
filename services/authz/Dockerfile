# syntax=docker/dockerfile:1

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM python:3.12-slim AS builder

WORKDIR /app

# Install build tools and grpcio-tools (needs gcc for proto generation)
RUN apt-get update && apt-get install -y --no-install-recommends gcc && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml ./
COPY proto/ proto/
COPY internal/ internal/
COPY entrypoints/ entrypoints/

# Install grpcio-tools system-wide for proto generation
RUN pip install --no-cache-dir grpcio-tools

# Generate gRPC stubs
COPY proto/generate.sh proto/generate.sh
RUN bash proto/generate.sh

# Install all deps into a prefix we can copy to the final image
# --ignore-installed: force reinstall even if already present system-wide from grpcio-tools step
RUN pip install --no-cache-dir --prefix=/install --ignore-installed ".[test]"


# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM python:3.12-slim

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local
# Copy source (with generated stubs)
COPY --from=builder /app /app

EXPOSE 50051

ENTRYPOINT ["python", "entrypoints/server/main.py"]
