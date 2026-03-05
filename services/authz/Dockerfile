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

# Install all deps into a prefix we can copy to the final image
RUN pip install --no-cache-dir --prefix=/install ".[test]" grpcio-tools

# Generate gRPC stubs
COPY proto/generate.sh proto/generate.sh
RUN bash proto/generate.sh


# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM python:3.12-slim

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local
# Copy source (with generated stubs)
COPY --from=builder /app /app

ENV NEO4J_URI=bolt://neo4j:7687 \
    NEO4J_USER=neo4j \
    NEO4J_PASSWORD=password123 \
    REDIS_HOST=redis \
    REDIS_PORT=6379 \
    KAFKA_BOOTSTRAP=kafka:29092 \
    GRPC_PORT=50051

EXPOSE 50051

ENTRYPOINT ["python", "entrypoints/server/main.py"]
