use opentelemetry::{
    global,
    metrics::{Counter, Histogram},
};

/// Service-level metrics for the Quota service.
/// Covers Redis flush health — the only I/O on the non-hot path.
pub struct QuotaMetrics {
    /// Total flush cycles attempted (every 1s background task tick)
    pub redis_flush_total: Counter<u64>,
    /// Flush cycles that ended with a Redis error
    pub redis_flush_errors_total: Counter<u64>,
    /// How many usage entries were written per flush cycle
    pub redis_flush_entries: Histogram<u64>,
    /// Wall-clock time each flush cycle took
    pub redis_flush_duration_seconds: Histogram<f64>,
}

impl Default for QuotaMetrics {
    fn default() -> Self {
        Self::new()
    }
}

impl QuotaMetrics {
    pub fn new() -> Self {
        let meter = global::meter("quota");

        Self {
            redis_flush_total: meter
                .u64_counter("quota_redis_flush_total")
                .with_description("Total Redis flush cycles")
                .build(),

            redis_flush_errors_total: meter
                .u64_counter("quota_redis_flush_errors_total")
                .with_description("Redis flush cycles that failed")
                .build(),

            redis_flush_entries: meter
                .u64_histogram("quota_redis_flush_entries")
                .with_description("Usage entries written per flush")
                .with_boundaries(vec![0.0, 1.0, 5.0, 10.0, 50.0, 100.0, 500.0, 1000.0])
                .build(),

            redis_flush_duration_seconds: meter
                .f64_histogram("quota_redis_flush_duration_seconds")
                .with_description("Time taken per Redis flush cycle")
                .with_unit("s")
                .with_boundaries(vec![
                    0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0,
                ])
                .build(),
        }
    }
}
