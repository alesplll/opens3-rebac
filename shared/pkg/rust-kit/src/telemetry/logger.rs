use opentelemetry::{trace::TracerProvider as _, KeyValue};
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

pub fn init(cfg: Config) {
    // EnvFilter is created per-branch because it's not Clone-safe to share
    let make_filter =
        || EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new(&cfg.log_level));

    if let Some(ref endpoint) = cfg.otlp_endpoint {
        let resource = Resource::new(vec![
            KeyValue::new("service.name", cfg.service_name.clone()),
            KeyValue::new("deployment.environment", cfg.environment.clone()),
        ]);

        let exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(endpoint.as_str())
            .build()
            .expect("failed to build OTLP span exporter");

        let provider = opentelemetry_sdk::trace::TracerProvider::builder()
            .with_batch_exporter(exporter, Tokio)
            .with_resource(resource)
            .build();

        opentelemetry::global::set_tracer_provider(provider.clone());
        let tracer = provider.tracer(cfg.service_name.clone());

        // Each branch builds its own full subscriber chain so the OTel layer
        // type-parameter S is correctly inferred per-branch.
        if cfg.json_format {
            tracing_subscriber::registry()
                .with(make_filter())
                .with(fmt::layer().json().with_span_events(FmtSpan::NONE))
                .with(OpenTelemetryLayer::new(tracer))
                .try_init()
                .ok();
        } else {
            tracing_subscriber::registry()
                .with(make_filter())
                .with(fmt::layer().with_span_events(FmtSpan::NONE))
                .with(OpenTelemetryLayer::new(tracer))
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
}
