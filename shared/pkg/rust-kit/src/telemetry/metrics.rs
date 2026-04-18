use opentelemetry::{global, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    metrics::{PeriodicReader, SdkMeterProvider},
    runtime::Tokio,
    Resource,
};

pub struct Config {
    pub service_name: String,
    pub environment: String,
    pub otlp_endpoint: String,
}

pub fn init(cfg: Config) -> Result<SdkMeterProvider, Box<dyn std::error::Error + Send + Sync>> {
    let resource = Resource::new(vec![
        KeyValue::new("service.name", cfg.service_name),
        KeyValue::new("deployment.environment", cfg.environment),
    ]);

    let exporter = opentelemetry_otlp::MetricExporter::builder()
        .with_tonic()
        .with_endpoint(cfg.otlp_endpoint)
        .build()?;

    let reader = PeriodicReader::builder(exporter, Tokio).build();

    let provider = SdkMeterProvider::builder()
        .with_resource(resource)
        .with_reader(reader)
        .build();

    global::set_meter_provider(provider.clone());
    Ok(provider)
}

/// Service-level gRPC metrics. Instantiate once per service at startup.
pub struct GrpcMetrics {
    pub requests_total: opentelemetry::metrics::Counter<u64>,
    pub responses_total: opentelemetry::metrics::Counter<u64>,
    pub response_time_seconds: opentelemetry::metrics::Histogram<f64>,
}

impl GrpcMetrics {
    pub fn new(service_name: &str) -> Self {
        // leak once at startup — global::meter requires &'static str
        let name: &'static str = Box::leak(service_name.to_string().into_boxed_str());
        let meter = global::meter(name);

        let requests_total = meter
            .u64_counter(format!("grpc_{service_name}_requests_total"))
            .build();

        let responses_total = meter
            .u64_counter(format!("grpc_{service_name}_response_total"))
            .build();

        let response_time_seconds = meter
            .f64_histogram(format!(
                "grpc_{service_name}_histogram_response_time_seconds"
            ))
            .with_unit("s")
            .with_boundaries(vec![
                0.0001, 0.0002, 0.0004, 0.0008, 0.0016, 0.0032, 0.0064, 0.0128,
                0.0256, 0.0512, 0.1024, 0.2048, 0.4096, 0.8192, 1.6384, 3.2768,
            ])
            .build();

        Self { requests_total, responses_total, response_time_seconds }
    }
}
