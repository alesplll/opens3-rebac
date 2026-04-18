# syntax=docker/dockerfile:1

# ── Stage 1: build ────────────────────────────────────────────────────────────
# rust:alpine → musl target → 100% static binary, no libc dependency
FROM rust:1.82-alpine AS builder

RUN apk add --no-cache musl-dev protobuf-dev

WORKDIR /workspace

# Copy workspace manifest first — layer is cached until Cargo.toml changes
COPY Cargo.toml Cargo.lock* ./

# Copy only what the quota crate needs
COPY shared/pkg/rust-kit ./shared/pkg/rust-kit/
COPY services/quota      ./services/quota/
COPY shared/api/quota    ./shared/api/quota/

RUN cargo build --release -p quota-service

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
# Minimal alpine — static binary needs only the OS for DNS/certs
FROM alpine:3.21

RUN addgroup -S quota && adduser -S quota -G quota

WORKDIR /app

COPY --from=builder /workspace/target/release/server ./server

USER quota

EXPOSE 50055

ENTRYPOINT ["./server"]
