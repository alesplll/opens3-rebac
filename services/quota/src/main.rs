mod app;
mod cache;
mod config;
mod domain;
mod metrics;
mod repository;
mod service;
mod transport;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load .env if present (dev only; in Docker env vars come from docker-compose)
    dotenvy::dotenv().ok();

    app::run().await
}
