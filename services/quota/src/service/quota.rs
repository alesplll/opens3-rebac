//! Business logic layer. Orchestrates MemoryCache → Repository.
//!
//! Mirrors the pattern from authz PermissionService:
//!   1. Try hot path (MemoryCache)
//!   2. For rare operations (SetQuota, startup) hit Repository
//!
//! All methods are Send + Sync — shared behind Arc.

use std::{sync::Arc, time::Instant};

use tracing::{debug, instrument, warn};

use crate::{
    cache::MemoryCache,
    config,
    domain::{CheckResult, DenyReason, QuotaEntry, QuotaError, ResourceDelta, UsageEntry},
    metrics::QuotaMetrics,
    repository::traits::QuotaRepository,
};

pub struct QuotaService<R: QuotaRepository> {
    cache: Arc<MemoryCache>,
    repo: Arc<R>,
    metrics: Arc<QuotaMetrics>,
}

impl<R: QuotaRepository> QuotaService<R> {
    pub fn new(cache: Arc<MemoryCache>, repo: Arc<R>, metrics: Arc<QuotaMetrics>) -> Self {
        Self { cache, repo, metrics }
    }

    // ── CheckQuota ────────────────────────────────────────────────────────────

    /// Check and reserve quota for a subject (and optionally a bucket).
    /// Implements the reserve-and-rollback pattern from the wiki:
    ///   1. Reserve user quota
    ///   2. Reserve bucket quota (if bucket_id provided)
    ///   3. On bucket denial → rollback user reservation
    #[instrument(skip(self, delta), name = "service.check_quota", fields(subject = %subject_id, bucket = ?bucket_id))]
    pub fn check_quota(
        &self,
        subject_id: &str,
        bucket_id: Option<&str>,
        delta: &ResourceDelta,
    ) -> Result<CheckResult, QuotaError> {
        if subject_id.is_empty() {
            return Err(QuotaError::InvalidArgument("subject_id is required".into()));
        }

        let cfg = config::get();
        let user_default = QuotaEntry {
            bytes_limit: cfg.default_user_bytes_limit,
            objects_limit: cfg.default_user_objects_limit,
            buckets_limit: cfg.default_user_buckets_limit,
        };

        // Step 1: check user quota
        let user_result = self.cache.check_and_reserve(subject_id, delta, &user_default);

        if let CheckResult::Denied(_) = &user_result {
            debug!(subject = %subject_id, "user quota denied");
            return Ok(user_result);
        }

        // Step 2: check bucket quota (if applicable)
        if let Some(bucket_id) = bucket_id {
            if !bucket_id.is_empty() {
                let bucket_default = QuotaEntry {
                    bytes_limit: cfg.default_bucket_bytes_limit,
                    objects_limit: cfg.default_bucket_objects_limit,
                    buckets_limit: QuotaEntry::UNLIMITED,
                };

                let bucket_result =
                    self.cache.check_and_reserve(bucket_id, delta, &bucket_default);

                if let CheckResult::Denied(ref reason) = bucket_result {
                    // Rollback the user reservation
                    self.cache.update(subject_id, &delta.negate());

                    let bucket_deny = match reason {
                        DenyReason::UserStorageExceeded { used, limit } => {
                            DenyReason::BucketStorageExceeded { used: *used, limit: *limit }
                        }
                        other => other.clone(),
                    };

                    return Ok(CheckResult::Denied(bucket_deny));
                }
            }
        }

        Ok(CheckResult::Allowed)
    }

    // ── UpdateUsage ───────────────────────────────────────────────────────────

    /// Fire-and-forget usage update. Called after a successful S3 operation.
    /// Delta can be negative (object/bucket deletion).
    #[instrument(skip(self, delta), name = "service.update_usage", fields(subject = %subject_id, bucket = ?bucket_id))]
    pub fn update_usage(
        &self,
        subject_id: &str,
        bucket_id: Option<&str>,
        delta: &ResourceDelta,
    ) -> Result<(), QuotaError> {
        if subject_id.is_empty() {
            return Err(QuotaError::InvalidArgument("subject_id is required".into()));
        }

        self.cache.update(subject_id, delta);

        if let Some(bucket_id) = bucket_id {
            if !bucket_id.is_empty() {
                self.cache.update(bucket_id, delta);
            }
        }

        Ok(())
    }

    // ── GetUsage ──────────────────────────────────────────────────────────────

    pub fn get_usage(&self, subject_id: &str) -> Result<UsageEntry, QuotaError> {
        Ok(self.cache.get_usage(subject_id).unwrap_or_default())
    }

    // ── SetQuota ──────────────────────────────────────────────────────────────

    /// Set quota limits for a subject. Writes to cache immediately;
    /// Redis flush happens in the background via the periodic flush task.
    #[instrument(skip(self, quota), name = "service.set_quota", fields(subject = %subject_id))]
    pub async fn set_quota(
        &self,
        subject_id: &str,
        quota: QuotaEntry,
    ) -> Result<(), QuotaError> {
        if subject_id.is_empty() {
            return Err(QuotaError::InvalidArgument("subject_id is required".into()));
        }

        self.cache.set_limit(subject_id, quota);

        // Write-through to Redis immediately for limits (they're rare and important)
        self.repo
            .flush_limits(&[(subject_id.to_string(), quota)])
            .await
    }

    // ── GetQuota ──────────────────────────────────────────────────────────────

    pub async fn get_quota(&self, subject_id: &str) -> Result<Option<QuotaEntry>, QuotaError> {
        if let Some(limit) = self.cache.get_limit(subject_id) {
            return Ok(Some(limit));
        }
        // Cache miss (shouldn't happen after startup load, but safe fallback)
        let limit = self.repo.get_limit(subject_id).await?;
        if let Some(l) = limit {
            self.cache.set_limit(subject_id, l);
        }
        Ok(limit)
    }

    // ── External event handlers (Phase 2) ────────────────────────────────────

    #[allow(dead_code)] // Phase 2: called when user-deletion event arrives
    pub fn on_user_deleted(&self, subject_id: &str) {
        self.cache.delete_subject(subject_id);
    }

    // ── Health ────────────────────────────────────────────────────────────────

    pub async fn health(&self) -> Result<(), QuotaError> {
        self.repo.health().await
    }

    // ── Startup: load from Redis ──────────────────────────────────────────────

    pub async fn load_from_storage(&self) -> Result<(), QuotaError> {
        let usage = self.repo.load_all_usage().await?;
        let limits = self.repo.load_all_limits().await?;

        let usage_count = usage.len();
        let limits_count = limits.len();

        self.cache.load_usage(usage);
        self.cache.load_limits(limits);

        tracing::info!(usage_count, limits_count, "quota data loaded from Redis");
        Ok(())
    }

    // ── Flush snapshot ────────────────────────────────────────────────────────

    /// Called by the periodic flush task in app.rs every 500ms.
    pub async fn flush_to_storage(&self) -> Result<(), QuotaError> {
        let start = Instant::now();
        let usage = self.cache.snapshot_usage();
        let count = usage.len() as u64;

        self.metrics.redis_flush_total.add(1, &[]);

        if !usage.is_empty() {
            if let Err(e) = self.repo.flush_usage(&usage).await {
                warn!(error = %e, "failed to flush usage to Redis");
                self.metrics.redis_flush_errors_total.add(1, &[]);
            }
        }

        self.metrics.redis_flush_entries.record(count, &[]);
        self.metrics.redis_flush_duration_seconds.record(start.elapsed().as_secs_f64(), &[]);

        Ok(())
    }
}
