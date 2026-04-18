//! Reads all env vars once at startup. Singleton via `OnceLock`.
//! Mirrors the pattern used in authz (config.py) and Go services.

use std::sync::OnceLock;

static CFG: OnceLock<Config> = OnceLock::new();

#[derive(Debug, Clone)]
pub struct Config {
    // gRPC
    pub grpc_port: u16,

    // Redis
    pub redis_host: String,
    pub redis_port: u16,
    pub redis_db: u8,
    pub redis_password: Option<String>,
    pub redis_flush_interval_ms: u64,

    // Default quotas (-1 = unlimited)
    pub default_user_bytes_limit: i64,
    pub default_user_objects_limit: i64,
    pub default_user_buckets_limit: i64,
    pub default_bucket_bytes_limit: i64,
    pub default_bucket_objects_limit: i64,

    // Observability
    pub service_name: String,
    pub environment: String,
    pub log_level: String,
    pub log_json: bool,
    pub otlp_endpoint: String,
    pub enable_otlp: bool,
}

impl Config {
    /// Load config from environment. Panics on missing required vars.
    fn load() -> Self {
        Self {
            grpc_port: var_parse("GRPC_PORT", 50055),

            redis_host: var("REDIS_HOST", "localhost"),
            redis_port: var_parse("REDIS_PORT", 6379),
            redis_db: var_parse("REDIS_DB", 1),
            redis_password: std::env::var("REDIS_PASSWORD").ok(),
            redis_flush_interval_ms: var_parse("REDIS_FLUSH_INTERVAL_MS", 5000),

            default_user_bytes_limit: var_parse("DEFAULT_USER_BYTES_LIMIT", 10_737_418_240),
            default_user_objects_limit: var_parse("DEFAULT_USER_OBJECTS_LIMIT", -1),
            default_user_buckets_limit: var_parse("DEFAULT_USER_BUCKETS_LIMIT", 100),
            default_bucket_bytes_limit: var_parse("DEFAULT_BUCKET_BYTES_LIMIT", -1),
            default_bucket_objects_limit: var_parse("DEFAULT_BUCKET_OBJECTS_LIMIT", -1),

            service_name: var("SERVICE_NAME", "quota"),
            environment: var("ENVIRONMENT", "development"),
            log_level: var("LOG_LEVEL", "info"),
            log_json: var_parse("LOG_JSON", true),
            otlp_endpoint: var("OTLP_ENDPOINT", "http://otel-collector:4317"),
            enable_otlp: var_parse("ENABLE_OTLP", false),
        }
    }

    /// gRPC listen address.
    pub fn grpc_addr(&self) -> String {
        format!("0.0.0.0:{}", self.grpc_port)
    }

    /// Redis URL in `redis[s]://[[username][:password]@]host[:port]/db` format.
    pub fn redis_url(&self) -> String {
        match &self.redis_password {
            Some(pw) => format!(
                "redis://:{}@{}:{}/{}",
                pw, self.redis_host, self.redis_port, self.redis_db
            ),
            None => format!(
                "redis://{}:{}/{}",
                self.redis_host, self.redis_port, self.redis_db
            ),
        }
    }
}

/// Get the global singleton config. Panics if `init()` was not called.
pub fn get() -> &'static Config {
    CFG.get()
        .expect("config not initialised — call config::init() first")
}

/// Load env vars into the global singleton. Idempotent.
pub fn init() {
    CFG.get_or_init(Config::load);
}

// ── Helpers ───────────────────────────────────────────────────────────────────

fn var(key: &str, default: &str) -> String {
    std::env::var(key).unwrap_or_else(|_| default.to_string())
}

fn var_parse<T: std::str::FromStr>(key: &str, default: T) -> T
where
    T::Err: std::fmt::Debug,
{
    std::env::var(key)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(default)
}
