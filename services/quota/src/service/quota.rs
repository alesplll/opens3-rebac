//! Business logic layer. Orchestrates MemoryCache → Repository.
//!
//! Mirrors the pattern from authz PermissionService:
//!   1. Try hot path (MemoryCache)
//!   2. For rare operations (SetQuota, startup) hit Repository
//!
//! All methods are Send + Sync — shared behind Arc.

#[cfg(test)]
use std::sync::Mutex;
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
        Self {
            cache,
            repo,
            metrics,
        }
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
        let user_result = self
            .cache
            .check_and_reserve(subject_id, delta, &user_default);

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

                let bucket_result = self
                    .cache
                    .check_and_reserve(bucket_id, delta, &bucket_default);

                if let CheckResult::Denied(ref reason) = bucket_result {
                    // Rollback the user reservation
                    self.cache.update(subject_id, &delta.negate());

                    let bucket_deny = match reason {
                        DenyReason::UserStorageExceeded { used, limit } => {
                            DenyReason::BucketStorageExceeded {
                                used: *used,
                                limit: *limit,
                            }
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
    pub async fn set_quota(&self, subject_id: &str, quota: QuotaEntry) -> Result<(), QuotaError> {
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

    /// Called by the periodic flush task in app.rs.
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
        self.metrics
            .redis_flush_duration_seconds
            .record(start.elapsed().as_secs_f64(), &[]);

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        cache::MemoryCache,
        domain::{CheckResult, DenyReason, QuotaEntry, QuotaError, ResourceDelta, UsageEntry},
        metrics::QuotaMetrics,
        repository::traits::QuotaRepository,
    };
    use std::sync::{Arc, Mutex, Once};

    // ── No-op repository for unit tests ──────────────────────────────────────

    #[derive(Default)]
    struct NoopRepo {
        flushed: Mutex<Vec<(String, UsageEntry)>>,
    }

    #[async_trait::async_trait]
    impl QuotaRepository for NoopRepo {
        async fn load_all_usage(&self) -> Result<Vec<(String, UsageEntry)>, QuotaError> {
            Ok(vec![])
        }
        async fn load_all_limits(&self) -> Result<Vec<(String, QuotaEntry)>, QuotaError> {
            Ok(vec![])
        }
        async fn flush_usage(&self, entries: &[(String, UsageEntry)]) -> Result<(), QuotaError> {
            self.flushed.lock().unwrap().extend_from_slice(entries);
            Ok(())
        }
        async fn flush_limits(&self, _: &[(String, QuotaEntry)]) -> Result<(), QuotaError> {
            Ok(())
        }
        async fn get_limit(&self, _: &str) -> Result<Option<QuotaEntry>, QuotaError> {
            Ok(None)
        }
        async fn delete_subject(&self, _: &str) -> Result<(), QuotaError> {
            Ok(())
        }
        async fn health(&self) -> Result<(), QuotaError> {
            Ok(())
        }
    }

    // ── Setup ────────────────────────────────────────────────────────────────

    static INIT: Once = Once::new();

    fn init_config() {
        INIT.call_once(|| {
            std::env::set_var("DEFAULT_USER_BYTES_LIMIT", "10737418240"); // 10 GiB
            std::env::set_var("DEFAULT_USER_OBJECTS_LIMIT", "-1");
            std::env::set_var("DEFAULT_USER_BUCKETS_LIMIT", "100");
            std::env::set_var("DEFAULT_BUCKET_BYTES_LIMIT", "-1");
            std::env::set_var("DEFAULT_BUCKET_OBJECTS_LIMIT", "-1");
            crate::config::init();
        });
    }

    fn make_service() -> QuotaService<NoopRepo> {
        init_config();
        QuotaService::new(
            Arc::new(MemoryCache::new()),
            Arc::new(NoopRepo::default()),
            Arc::new(QuotaMetrics::new()),
        )
    }

    fn delta(bytes: i64, objects: i64, buckets: i64) -> ResourceDelta {
        ResourceDelta {
            bytes,
            objects,
            buckets,
        }
    }

    // ── check_quota ───────────────────────────────────────────────────────────

    #[test]
    fn check_quota_allows_within_default_limit() {
        let svc = make_service();
        let result = svc
            .check_quota("user:alice", None, &delta(1024, 1, 0))
            .unwrap();
        assert_eq!(result, CheckResult::Allowed);
    }

    #[test]
    fn check_quota_empty_subject_returns_error() {
        let svc = make_service();
        let result = svc.check_quota("", None, &delta(100, 0, 0));
        assert!(matches!(result, Err(QuotaError::InvalidArgument(_))));
    }

    #[test]
    fn check_quota_bucket_deny_rolls_back_user_reservation() {
        let svc = make_service();

        // Give user a small byte limit via cache
        svc.cache.set_limit(
            "user:alice",
            QuotaEntry {
                bytes_limit: 10_000,
                objects_limit: -1,
                buckets_limit: -1,
            },
        );
        // Give bucket an even smaller byte limit
        svc.cache.set_limit(
            "bucket:photos",
            QuotaEntry {
                bytes_limit: 100,
                objects_limit: -1,
                buckets_limit: -1,
            },
        );

        // Delta exceeds bucket limit but not user limit
        let result = svc
            .check_quota("user:alice", Some("bucket:photos"), &delta(500, 1, 0))
            .unwrap();

        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::BucketStorageExceeded { .. })
        ));

        // User usage must be rolled back to 0 — no reservation left
        let user_usage = svc.cache.get_usage("user:alice").unwrap_or_default();
        assert_eq!(user_usage.bytes, 0, "user reservation must be rolled back");
    }

    #[test]
    fn check_quota_user_denied_skips_bucket_check() {
        let svc = make_service();

        svc.cache.set_limit(
            "user:alice",
            QuotaEntry {
                bytes_limit: 100,
                objects_limit: -1,
                buckets_limit: -1,
            },
        );

        let result = svc
            .check_quota("user:alice", Some("bucket:photos"), &delta(500, 0, 0))
            .unwrap();

        // Denied at user level — bucket should have no usage recorded
        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::UserStorageExceeded { .. })
        ));
        assert!(svc.cache.get_usage("bucket:photos").is_none());
    }

    // ── update_usage ──────────────────────────────────────────────────────────

    #[test]
    fn update_usage_empty_subject_returns_error() {
        let svc = make_service();
        let result = svc.update_usage("", None, &delta(100, 0, 0));
        assert!(matches!(result, Err(QuotaError::InvalidArgument(_))));
    }

    #[test]
    fn update_usage_applies_to_user_and_bucket() {
        let svc = make_service();
        svc.update_usage("user:alice", Some("bucket:photos"), &delta(200, 1, 0))
            .unwrap();

        assert_eq!(svc.cache.get_usage("user:alice").unwrap().bytes, 200);
        assert_eq!(svc.cache.get_usage("bucket:photos").unwrap().bytes, 200);
    }

    #[test]
    fn update_usage_negative_delta_releases_space() {
        let svc = make_service();
        svc.update_usage("user:alice", None, &delta(500, 2, 0))
            .unwrap();
        svc.update_usage("user:alice", None, &delta(-200, -1, 0))
            .unwrap();

        let usage = svc.cache.get_usage("user:alice").unwrap();
        assert_eq!(usage.bytes, 300);
        assert_eq!(usage.objects, 1);
    }

    // ── get_usage ─────────────────────────────────────────────────────────────

    #[test]
    fn get_usage_returns_zero_for_unknown_subject() {
        let svc = make_service();
        let usage = svc.get_usage("user:nobody").unwrap();
        assert_eq!(usage.bytes, 0);
        assert_eq!(usage.objects, 0);
    }

    // ── flush ─────────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn flush_to_storage_writes_to_repository() {
        let svc = make_service();
        svc.update_usage("user:alice", None, &delta(100, 1, 0))
            .unwrap();

        svc.flush_to_storage().await.unwrap();

        let flushed = svc.repo.flushed.lock().unwrap();
        assert!(!flushed.is_empty());
        let alice = flushed.iter().find(|(id, _)| id == "user:alice");
        assert!(alice.is_some());
        assert_eq!(alice.unwrap().1.bytes, 100);
    }
}
