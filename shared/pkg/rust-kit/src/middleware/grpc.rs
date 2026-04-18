//! Tower middleware Layer для gRPC-серверов.
//!
//! `MetricsLayer` оборачивает каждый унарный вызов и записывает:
//!   - grpc_{service}_requests_total  (счётчик входящих запросов)
//!   - grpc_{service}_response_total  (счётчик ответов с labels: status, method)
//!   - grpc_{service}_histogram_response_time_seconds (гистограмма латентности)
//!
//! Использование (в app.rs):
//! ```rust
//! use rust_kit::middleware::grpc::MetricsLayer;
//! use rust_kit::telemetry::metrics::GrpcMetrics;
//!
//! let metrics = Arc::new(GrpcMetrics::new("quota"));
//! Server::builder()
//!     .layer(MetricsLayer::new(metrics))
//!     .add_service(...)
//!     .serve(addr)
//!     .await?;
//! ```

use std::{
    future::Future,
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::Instant,
};

use opentelemetry::KeyValue;
use pin_project_lite::pin_project;
use tonic::body::BoxBody;
use tower::{Layer, Service};

use crate::telemetry::metrics::GrpcMetrics;

// ── Layer ─────────────────────────────────────────────────────────────────────

#[derive(Clone)]
pub struct MetricsLayer {
    metrics: Arc<GrpcMetrics>,
}

impl MetricsLayer {
    pub fn new(metrics: Arc<GrpcMetrics>) -> Self {
        Self { metrics }
    }
}

impl<S> Layer<S> for MetricsLayer {
    type Service = MetricsService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        MetricsService { inner, metrics: Arc::clone(&self.metrics) }
    }
}

// ── Service wrapper ───────────────────────────────────────────────────────────

#[derive(Clone)]
pub struct MetricsService<S> {
    inner: S,
    metrics: Arc<GrpcMetrics>,
}

type Req = http::Request<BoxBody>;
type Resp = http::Response<BoxBody>;

impl<S> Service<Req> for MetricsService<S>
where
    S: Service<Req, Response = Resp> + Clone + Send + 'static,
    S::Future: Send + 'static,
    S::Error: std::fmt::Display,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = MetricsFuture<S::Future>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, req: Req) -> Self::Future {
        let method = req.uri().path().to_string();
        self.metrics.requests_total.add(1, &[KeyValue::new("method", method.clone())]);
        let start = Instant::now();
        let inner = self.inner.call(req);
        MetricsFuture { inner, metrics: Arc::clone(&self.metrics), method, start }
    }
}

// ── Future that records response metrics ─────────────────────────────────────

pin_project! {
    pub struct MetricsFuture<F> {
        #[pin]
        inner: F,
        metrics: Arc<GrpcMetrics>,
        method: String,
        start: Instant,
    }
}

impl<F, E> Future for MetricsFuture<F>
where
    F: Future<Output = Result<Resp, E>>,
    E: std::fmt::Display,
{
    type Output = F::Output;

    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        let this = self.project();
        match this.inner.poll(cx) {
            Poll::Pending => Poll::Pending,
            Poll::Ready(result) => {
                let elapsed = this.start.elapsed().as_secs_f64();
                let status = match &result {
                    Ok(resp) => {
                        let grpc_status = resp
                            .headers()
                            .get("grpc-status")
                            .and_then(|v| v.to_str().ok())
                            .unwrap_or("0");
                        grpc_status.to_string()
                    }
                    Err(_) => "error".to_string(),
                };

                let labels = [
                    KeyValue::new("method", this.method.clone()),
                    KeyValue::new("status", status),
                ];
                this.metrics.responses_total.add(1, &labels);
                this.metrics.response_time_seconds.record(elapsed, &labels);

                Poll::Ready(result)
            }
        }
    }
}
