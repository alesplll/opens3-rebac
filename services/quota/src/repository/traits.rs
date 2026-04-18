//! Repository trait — контракт между сервисным слоем и хранилищем.
//! Следует паттерну из authz (interfaces.py / GraphStore Protocol).

use crate::domain::{QuotaEntry, QuotaError, UsageEntry};

/// Persistent storage for quota data.
/// In production this is Redis; in tests it can be an in-memory mock.
#[async_trait::async_trait]
pub trait QuotaRepository: Send + Sync + 'static {
    /// Load all usage entries from storage (called once at startup).
    async fn load_all_usage(&self) -> Result<Vec<(String, UsageEntry)>, QuotaError>;

    /// Load all quota limits from storage (called once at startup).
    async fn load_all_limits(&self) -> Result<Vec<(String, QuotaEntry)>, QuotaError>;

    /// Persist a batch of usage entries (called by the flush task every 1s, dirty-only).
    async fn flush_usage(&self, entries: &[(String, UsageEntry)]) -> Result<(), QuotaError>;

    /// Persist a batch of quota limits.
    async fn flush_limits(&self, entries: &[(String, QuotaEntry)]) -> Result<(), QuotaError>;

    /// Get a single quota limit (fallback when memory cache misses after restart).
    async fn get_limit(&self, subject_id: &str) -> Result<Option<QuotaEntry>, QuotaError>;

    /// Delete all data for a subject (e.g. when a user is removed). Phase 2.
    #[allow(dead_code)]
    async fn delete_subject(&self, subject_id: &str) -> Result<(), QuotaError>;

    /// Check Redis connectivity (for HealthCheck RPC).
    async fn health(&self) -> Result<(), QuotaError>;
}
