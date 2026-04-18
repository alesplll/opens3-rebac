//! OTel trace provider initialisation.
//!
//! Re-exports the global OTel tracer and provides a helper to create named tracers.
//! Actual provider is installed by `logger::init` (unified: tracing + traces go through the same pipeline).

use opentelemetry::global;
use opentelemetry::trace::Tracer;

/// Get a named tracer. Call after `logger::init`.
pub fn tracer(name: &'static str) -> impl Tracer {
    global::tracer(name)
}
