# syntax=docker/dockerfile:1

# ── Stage 1: chef ─────────────────────────────────────────────────────────────
FROM rust:1.95-alpine AS chef

RUN apk add --no-cache musl-dev protobuf-dev && \
    cargo install cargo-chef --locked

WORKDIR /workspace

# ── Stage 2: planner ──────────────────────────────────────────────────────────
# Analyses the workspace and produces a recipe.json with the dependency list.
# Re-runs only when Cargo.toml / Cargo.lock change.
FROM chef AS planner

COPY Cargo.toml Cargo.lock* ./
COPY shared/pkg/rust-kit ./shared/pkg/rust-kit/
COPY services/quota      ./services/quota/
COPY shared/api/quota    ./shared/api/quota/

RUN cargo chef prepare --recipe-path recipe.json

# ── Stage 3: builder ──────────────────────────────────────────────────────────
# cook = compile only dependencies (cached layer).
# Then build the actual crate on top.
FROM chef AS builder

COPY --from=planner /workspace/recipe.json recipe.json

RUN cargo chef cook --release --recipe-path recipe.json

COPY Cargo.toml Cargo.lock* ./
COPY shared/pkg/rust-kit ./shared/pkg/rust-kit/
COPY services/quota      ./services/quota/
COPY shared/api/quota    ./shared/api/quota/

RUN cargo build --release -p quota-service

# ── Stage 4: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21

RUN addgroup -S quota && adduser -S quota -G quota

WORKDIR /app

COPY --from=builder /workspace/target/release/server ./server

USER quota

EXPOSE 50055

ENTRYPOINT ["./server"]
