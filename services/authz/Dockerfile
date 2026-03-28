# syntax=docker/dockerfile:1

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM python:3.12-slim AS builder

WORKDIR /app

COPY pyproject.toml ./
COPY proto/ proto/
COPY internal/ internal/
COPY entrypoints/ entrypoints/

# gRPC stubs are pre-generated (internal/gen/) and committed to the repo.
# To regenerate locally: bash proto/generate.sh
RUN pip install --no-cache-dir --prefix=/install ".[test]"


# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM python:3.12-slim

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local
# Copy source (with generated stubs)
COPY --from=builder /app /app

EXPOSE 50051

ENTRYPOINT ["python", "entrypoints/server/main.py"]
