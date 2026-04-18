//! OTel metrics provider — Rust analogue of go-kit/metric.
//!
//! Creates a MeterProvider that exports metrics to the OTel Collector via OTLP/gRPC.
//! Metric names follow the same convention as Go-kit:
//!   grpc_{service}_requests_total
//!   grpc_{service}_response_total
//!   grpc_{service}_histogram_response_time_seconds
//!
//! Usage:
//! ```rust
//! let provider = rust_kit::telemetry::metrics::init(MetricsConfig {
//!     service_name:  "quota".into(),
//!     otlp_endpoint: "http://otel-collector:4317".into(),
//! })?;
//! // provider must be kept alive for the duration of the process
//! ```

use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    metrics::{PeriodicReader, SdkMeterProvider},
    runtime::Tokio,
    Resource,
};
use opentelemetry_semantic_conventions::resource::{DEPLOYMENT_ENVIRONMENT_NAME, SERVICE_NAME};

pub struct Config {
    pub service_name: String,
    pub environment: String,
    pub otlp_endpoint: String,
}

/// Initialise and register the global OTel MeterProvider.
/// Returns the provider so the caller can shut it down gracefully.
pub fn init(cfg: Config) -> Result<SdkMeterProvider, opentelemetry::metrics::MetricsError> {
    let resource = Resource::builder()
        .with_attribute(SERVICE_NAME, cfg.service_name)
        .with_attribute(DEPLOYMENT_ENVIRONMENT_NAME, cfg.environment)
        .build();

    let exporter = opentelemetry_otlp::new_exporter()
        .tonic()
        .with_endpoint(cfg.otlp_endpoint)
        .build_metrics_exporter(
            Box::new(opentelemetry_sdk::metrics::reader::DefaultAggregationSelector::new()),
            Box::new(opentelemetry_sdk::metrics::reader::DefaultTemporalitySelector::new()),
        )?;

    let reader = PeriodicReader::builder(exporter, Tokio).build();

    let provider = SdkMeterProvider::builder()
        .with_resource(resource)
        .with_reader(reader)
        .build();

    global::set_meter_provider(provider.clone());

    Ok(provider)
}

/// Service-level gRPC metrics. Instantiate once per service.
pub struct GrpcMetrics {
    pub requests_total:       opentelemetry::metrics::Counter<u64>,
    pub responses_total:      opentelemetry::metrics::Counter<u64>,
    pub response_time_seconds: opentelemetry::metrics::Histogram<f64>,
}

impl GrpcMetrics {
    pub fn new(service_name: &str) -> Self {
        let meter = global::meter(service_name.to_string());

        let requests_total = meter
            .u64_counter(format!("grpc_{service_name}_requests_total"))
            .build();

        let responses_total = meter
            .u64_counter(format!("grpc_{service_name}_response_total"))
            .build();

        let response_time_seconds = meter
            .f64_histogram(format!("grpc_{service_name}_histogram_response_time_seconds"))
            .with_unit("s")
            .with_boundaries(vec![
                0.0001, 0.0002, 0.0004, 0.0008, 0.0016, 0.0032, 0.0064, 0.0128,
                0.0256, 0.0512, 0.1024, 0.2048, 0.4096, 0.8192, 1.6384, 3.2768,
            ])
            .build();

        Self { requests_total, responses_total, response_time_seconds }
    }
}
