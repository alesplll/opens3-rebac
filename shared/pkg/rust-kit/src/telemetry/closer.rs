//! Graceful shutdown coordinator — Rust analogue of go-kit/closer.
//!
//! Usage:
//! ```rust
//! let closer = Closer::new();
//! closer.add("redis", || async { redis.close().await });
//! closer.add("kafka", || async { kafka.close().await });
//!
//! // Blocks until SIGTERM/SIGINT, then drains all closers in parallel with timeout.
//! closer.wait_and_shutdown(Duration::from_secs(5)).await;
//! ```

use std::{future::Future, pin::Pin, sync::Arc, time::Duration};
use tokio::sync::Mutex;
use tracing::{error, info};

type ShutdownFn = Box<dyn Fn() -> Pin<Box<dyn Future<Output = ()> + Send>> + Send + Sync>;

#[derive(Clone)]
pub struct Closer {
    inner: Arc<Mutex<Vec<(String, ShutdownFn)>>>,
}

impl Closer {
    pub fn new() -> Self {
        Self { inner: Arc::new(Mutex::new(Vec::new())) }
    }

    /// Register a named shutdown callback. Executed in reverse registration order.
    pub async fn add<F, Fut>(&self, name: impl Into<String>, f: F)
    where
        F: Fn() -> Fut + Send + Sync + 'static,
        Fut: Future<Output = ()> + Send + 'static,
    {
        let name = name.into();
        let boxed: ShutdownFn = Box::new(move || Box::pin(f()));
        self.inner.lock().await.push((name, boxed));
    }

    /// Wait for SIGTERM or SIGINT, then run all shutdown callbacks with a timeout.
    pub async fn wait_and_shutdown(self, timeout: Duration) {
        wait_for_signal().await;
        info!("shutdown signal received, starting graceful shutdown");
        self.shutdown(timeout).await;
    }

    /// Run all shutdown callbacks immediately (parallel, reverse order, bounded by timeout).
    pub async fn shutdown(self, timeout: Duration) {
        let funcs = {
            let mut guard = self.inner.lock().await;
            let taken: Vec<_> = guard.drain(..).collect();
            taken.into_iter().rev().collect::<Vec<_>>()
        };

        if funcs.is_empty() {
            return;
        }

        let handles: Vec<_> = funcs
            .into_iter()
            .map(|(name, f)| {
                tokio::spawn(async move {
                    info!(component = %name, "closing");
                    let start = std::time::Instant::now();
                    f().await;
                    info!(component = %name, elapsed_ms = start.elapsed().as_millis(), "closed");
                })
            })
            .collect();

        let all = futures::future::join_all(handles);
        if tokio::time::timeout(timeout, all).await.is_err() {
            error!("shutdown timed out after {:?}", timeout);
        }
    }
}

impl Default for Closer {
    fn default() -> Self {
        Self::new()
    }
}

async fn wait_for_signal() {
    use tokio::signal::unix::{signal, SignalKind};

    let mut sigterm = signal(SignalKind::terminate()).expect("failed to register SIGTERM handler");
    let mut sigint = signal(SignalKind::interrupt()).expect("failed to register SIGINT handler");

    tokio::select! {
        _ = sigterm.recv() => info!("received SIGTERM"),
        _ = sigint.recv()  => info!("received SIGINT"),
    }
}
