use std::{
    future::Future,
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::Instant,
};

use http::{HeaderMap, Request, Response};
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

impl<S> Service<Request<BoxBody>> for MetricsService<S>
where
    S: Service<Request<BoxBody>, Response = Response<BoxBody>> + Clone + Send + 'static,
    S::Future: Send + 'static,
    S::Error: std::fmt::Display,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = MetricsFuture<S::Future>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, req: Request<BoxBody>) -> Self::Future {
        let method = req.uri().path().to_string();
        self.metrics.requests_total.add(1, &[KeyValue::new("method", method.clone())]);
        let start = Instant::now();
        let inner = self.inner.call(req);
        MetricsFuture { inner, metrics: Arc::clone(&self.metrics), method, start }
    }
}

// ── Future ────────────────────────────────────────────────────────────────────

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
    F: Future<Output = Result<Response<BoxBody>, E>>,
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
                        let headers: &HeaderMap = resp.headers();
                        headers
                            .get("grpc-status")
                            .and_then(|v: &http::HeaderValue| v.to_str().ok())
                            .unwrap_or("0")
                            .to_string()
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
