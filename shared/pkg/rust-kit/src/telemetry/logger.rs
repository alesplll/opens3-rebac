use std::sync::OnceLock;

use opentelemetry::{trace::TracerProvider as _, KeyValue};
use opentelemetry_appender_tracing::layer::OpenTelemetryTracingBridge;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{runtime::Tokio, Resource};
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

static LOG_PROVIDER: OnceLock<opentelemetry_sdk::logs::LoggerProvider> = OnceLock::new();

pub fn init(cfg: Config) {
    let make_filter =
        || EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new(&cfg.log_level));

    if let Some(ref endpoint) = cfg.otlp_endpoint {
        let resource = Resource::new(vec![
            KeyValue::new("service.name", cfg.service_name.clone()),
            KeyValue::new("deployment.environment", cfg.environment.clone()),
        ]);

        let span_exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(endpoint.as_str())
            .build()
            .expect("failed to build OTLP span exporter");

        let tracer_provider = opentelemetry_sdk::trace::TracerProvider::builder()
            .with_batch_exporter(span_exporter, Tokio)
            .with_resource(resource.clone())
            .build();

        opentelemetry::global::set_tracer_provider(tracer_provider.clone());
        let tracer = tracer_provider.tracer(cfg.service_name.clone());

        let log_exporter = opentelemetry_otlp::LogExporter::builder()
            .with_tonic()
            .with_endpoint(endpoint.as_str())
            .build()
            .expect("failed to build OTLP log exporter");

        let log_provider = opentelemetry_sdk::logs::LoggerProvider::builder()
            .with_batch_exporter(log_exporter, Tokio)
            .with_resource(resource)
            .build();

        opentelemetry::global::set_logger_provider(log_provider.clone());
        let _ = LOG_PROVIDER.set(log_provider.clone());
        let log_bridge = OpenTelemetryTracingBridge::new(&log_provider);

        if cfg.json_format {
            tracing_subscriber::registry()
                .with(make_filter())
                .with(fmt::layer().json().with_span_events(FmtSpan::NONE))
                .with(OpenTelemetryLayer::new(tracer))
                .with(log_bridge)
                .try_init()
                .ok();
        } else {
            tracing_subscriber::registry()
                .with(make_filter())
                .with(fmt::layer().with_span_events(FmtSpan::NONE))
                .with(OpenTelemetryLayer::new(tracer))
                .with(log_bridge)
                .try_init()
                .ok();
        }
    } else if cfg.json_format {
        tracing_subscriber::registry()
            .with(make_filter())
            .with(fmt::layer().json())
            .try_init()
            .ok();
    } else {
        tracing_subscriber::registry()
            .with(make_filter())
            .with(fmt::layer())
            .try_init()
            .ok();
    }
}

pub fn shutdown() {
    opentelemetry::global::shutdown_tracer_provider();
    if let Some(p) = LOG_PROVIDER.get() {
        let _ = p.shutdown();
    }
}
