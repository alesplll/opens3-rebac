//! Structured logger — Rust analogue of go-kit/logger and py-kit/logger.
//!
//! Features:
//! - JSON or pretty console output to stdout
//! - Automatic context enrichment: trace_id, span_id from active OTel span
//! - Optional OTLP log export via gRPC (same collector as traces/metrics)
//! - Singleton init pattern matching Go/Python versions
//!
//! Usage:
//! ```rust
//! use rust_kit::telemetry::logger;
//!
//! logger::init(logger::Config {
//!     service_name: "quota".into(),
//!     environment:  "dev".into(),
//!     log_level:    "info".into(),
//!     json_format:  true,
//!     otlp_endpoint: Some("http://otel-collector:4317".into()),
//! });
//!
//! tracing::info!(user_id = "abc", "quota check passed");
//! ```

use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::Resource;
use opentelemetry_semantic_conventions::resource::{DEPLOYMENT_ENVIRONMENT_NAME, SERVICE_NAME};
use tracing_opentelemetry::OpenTelemetryLayer;
use tracing_subscriber::{
    fmt::{self, format::FmtSpan},
    layer::SubscriberExt,
    util::SubscriberInitExt,
    EnvFilter,
};

pub struct Config {
    pub service_name: String,
    pub environment: String,
    pub log_level: String,
    pub json_format: bool,
    pub otlp_endpoint: Option<String>,
}

/// Initialise the global tracing subscriber.
/// Idempotent — subsequent calls are no-ops (panics if called twice on a real app
/// are prevented by `try_init`).
pub fn init(cfg: Config) {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&cfg.log_level));

    let resource = Resource::builder()
        .with_attribute(SERVICE_NAME, cfg.service_name.clone())
        .with_attribute(DEPLOYMENT_ENVIRONMENT_NAME, cfg.environment.clone())
        .build();

    let registry = tracing_subscriber::registry().with(env_filter);

    if let Some(endpoint) = cfg.otlp_endpoint {
        let otlp_tracer = opentelemetry_otlp::new_pipeline()
            .tracing()
            .with_exporter(
                opentelemetry_otlp::new_exporter()
                    .tonic()
                    .with_endpoint(&endpoint),
            )
            .with_trace_config(
                opentelemetry_sdk::trace::Config::default().with_resource(resource),
            )
            .install_batch(opentelemetry_sdk::runtime::Tokio)
            .expect("failed to install OTLP tracer");

        let otel_layer = OpenTelemetryLayer::new(otlp_tracer);

        if cfg.json_format {
            registry
                .with(fmt::layer().json().with_span_events(FmtSpan::NONE))
                .with(otel_layer)
                .try_init()
                .ok();
        } else {
            registry
                .with(fmt::layer().pretty().with_span_events(FmtSpan::NONE))
                .with(otel_layer)
                .try_init()
                .ok();
        }
    } else if cfg.json_format {
        registry
            .with(fmt::layer().json())
            .try_init()
            .ok();
    } else {
        registry
            .with(fmt::layer().pretty())
            .try_init()
            .ok();
    }
}

/// Flush all pending OTel spans/logs. Call at shutdown before process exit.
pub fn shutdown() {
    opentelemetry::global::shutdown_tracer_provider();
}
