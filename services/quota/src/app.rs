//! Application orchestrator — wires all layers together and runs the server.
//!
//! Startup sequence:
//!   1. Load config from env
//!   2. Init telemetry (logger, metrics)
//!   3. Connect to Redis
//!   4. Load quota data from Redis into MemoryCache
//!   5. Start background flush task (every 500ms)
//!   6. Start gRPC server (health + reflection + QuotaService)
//!   7. Wait for SIGTERM/SIGINT → graceful shutdown

use std::{sync::Arc, time::Duration};

use tonic::transport::Server;
use tonic_health::server::health_reporter;
use tonic_reflection::server::Builder as ReflectionBuilder;
use tracing::info;

use crate::{
    cache::MemoryCache,
    config,
    repository::RedisRepository,
    service::QuotaService,
    transport::grpc::{proto, GrpcHandler},
};

use proto::quota_service_server::QuotaServiceServer;
use rust_kit::telemetry::{closer::Closer, logger, metrics};

pub async fn run() -> anyhow::Result<()> {
    // 1. Config
    config::init();
    let cfg = config::get();

    // 2. Telemetry
    let otlp_endpoint = if cfg.enable_otlp { Some(cfg.otlp_endpoint.clone()) } else { None };

    logger::init(logger::Config {
        service_name:  cfg.service_name.clone(),
        environment:   cfg.environment.clone(),
        log_level:     cfg.log_level.clone(),
        json_format:   cfg.log_json,
        otlp_endpoint: otlp_endpoint.clone(),
    });

    let _metrics_provider = if let Some(ref endpoint) = otlp_endpoint {
        Some(
            metrics::init(metrics::Config {
                service_name:  cfg.service_name.clone(),
                environment:   cfg.environment.clone(),
                otlp_endpoint: endpoint.clone(),
            })
            .map_err(|e| anyhow::anyhow!("metrics init failed: {e}"))?,
        )
    } else {
        None
    };

    info!(
        service = %cfg.service_name,
        port    = cfg.grpc_port,
        env     = %cfg.environment,
        "starting quota service"
    );

    // 3. Redis
    let repo = Arc::new(
        RedisRepository::connect(&cfg.redis_url(), 8)
            .await
            .map_err(|e| anyhow::anyhow!("Redis connection failed: {e}"))?,
    );
    info!(url = %cfg.redis_url(), "connected to Redis");

    // 4. Memory cache + load from Redis
    let cache   = Arc::new(MemoryCache::new());
    let service = Arc::new(QuotaService::new(Arc::clone(&cache), Arc::clone(&repo)));

    service.load_from_storage().await?;

    // 5. Background flush task (Redis persistence every 500ms)
    {
        let flush_service  = Arc::clone(&service);
        let flush_interval = Duration::from_millis(cfg.redis_flush_interval_ms);
        tokio::spawn(async move {
            let mut ticker = tokio::time::interval(flush_interval);
            loop {
                ticker.tick().await;
                if let Err(e) = flush_service.flush_to_storage().await {
                    tracing::warn!(error = %e, "quota flush failed");
                }
            }
        });
    }

    // 6. gRPC server
    let handler = GrpcHandler::new(Arc::clone(&service));
    let svc     = QuotaServiceServer::new(handler);

    let (mut health_reporter, health_svc) = health_reporter();
    health_reporter
        .set_serving::<QuotaServiceServer<GrpcHandler<RedisRepository>>>()
        .await;

    let reflection_svc = ReflectionBuilder::configure()
        .register_encoded_file_descriptor_set(proto::FILE_DESCRIPTOR_SET)
        .build_v1()
        .map_err(|e| anyhow::anyhow!("reflection build failed: {e}"))?;

    let addr = cfg.grpc_addr().parse()?;
    info!(addr = %addr, "gRPC server listening");

    // 7. Graceful shutdown
    let closer = Closer::new();
    closer.add("telemetry", || async {
        opentelemetry::global::shutdown_tracer_provider();
        logger::shutdown();
    }).await;

    Server::builder()
        .add_service(health_svc)
        .add_service(reflection_svc)
        .add_service(svc)
        .serve_with_shutdown(addr, async { closer.wait_and_shutdown(Duration::from_secs(10)).await })
        .await
        .map_err(|e| anyhow::anyhow!("gRPC server error: {e}"))?;

    Ok(())
}
