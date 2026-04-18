//! Lock-free in-memory quota cache — the hot path for CheckQuota and UpdateUsage.
//!
//! Uses `DashMap` (fine-grained shard locking, ~256 shards by default).
//! All operations on a single entry are atomic: the shard lock is held for the
//! duration of the `entry()` call, so check-and-reserve is race-free without
//! an outer Mutex.
//!
//! Architecture:
//!   ┌─────────────────────┐
//!   │   gRPC handler      │  ←── hot path: <1µs per op
//!   └────────┬────────────┘
//!            │
//!   ┌────────▼────────────┐
//!   │   MemoryCache       │  DashMap<SubjectId, UsageEntry>
//!   │   (this module)     │  DashMap<SubjectId, QuotaEntry>
//!   └────────┬────────────┘
//!            │  flush every 500ms (background task in app.rs)
//!   ┌────────▼────────────┐
//!   │   RedisRepository   │  AOF persistence
//!   └─────────────────────┘

use dashmap::DashMap;
use tracing::instrument;

use crate::domain::{CheckResult, DenyReason, QuotaEntry, ResourceDelta, UsageEntry};

pub struct MemoryCache {
    usage: DashMap<String, UsageEntry>,
    limits: DashMap<String, QuotaEntry>,
}

impl MemoryCache {
    pub fn new() -> Self {
        Self {
            usage: DashMap::new(),
            limits: DashMap::new(),
        }
    }

    // ── Bulk load on startup ──────────────────────────────────────────────────

    pub fn load_usage(&self, entries: Vec<(String, UsageEntry)>) {
        for (id, entry) in entries {
            self.usage.insert(id, entry);
        }
    }

    pub fn load_limits(&self, entries: Vec<(String, QuotaEntry)>) {
        for (id, entry) in entries {
            self.limits.insert(id, entry);
        }
    }

    // ── Hot-path read ─────────────────────────────────────────────────────────

    pub fn get_usage(&self, subject_id: &str) -> Option<UsageEntry> {
        self.usage.get(subject_id).map(|e| *e)
    }

    pub fn get_limit(&self, subject_id: &str) -> Option<QuotaEntry> {
        self.limits.get(subject_id).map(|e| *e)
    }

    // ── Check and atomically reserve ─────────────────────────────────────────

    /// Check whether the `delta` fits within `subject`'s quota, and if so,
    /// reserve it (increment usage). Atomic per entry — no TOCTOU race.
    ///
    /// If the subject has no explicit limit, the caller-supplied `default_limit`
    /// is used (from Config::default_user_*).
    #[instrument(skip(self), name = "cache.check_and_reserve")]
    pub fn check_and_reserve(
        &self,
        subject_id: &str,
        delta: &ResourceDelta,
        default_limit: &QuotaEntry,
    ) -> CheckResult {
        // Snapshot the limit outside the usage entry lock to avoid deadlock
        // between the two DashMaps. A stale limit snapshot is acceptable here:
        // SetQuota is rare; the 500ms flush window is our consistency boundary.
        let limit = self
            .limits
            .get(subject_id)
            .map(|e| *e)
            .unwrap_or(*default_limit);

        let mut denied: Option<DenyReason> = None;

        // `entry().and_modify()` / `or_insert_with()` hold the DashMap shard lock
        // for the entire closure — this makes the read-check-write atomic.
        self.usage
            .entry(subject_id.to_string())
            .and_modify(|usage| {
                if let Some(reason) = would_exceed(usage, &limit, delta) {
                    denied = Some(reason);
                } else {
                    usage.apply(delta);
                }
            })
            .or_insert_with(|| {
                let candidate = UsageEntry::from(delta);
                if let Some(reason) = would_exceed(&UsageEntry::default(), &limit, delta) {
                    denied = Some(reason);
                    UsageEntry::default()
                } else {
                    candidate
                }
            });

        match denied {
            Some(reason) => CheckResult::Denied(reason),
            None => CheckResult::Allowed,
        }
    }

    // ── Fire-and-forget update (after successful operation) ───────────────────

    pub fn update(&self, subject_id: &str, delta: &ResourceDelta) {
        self.usage
            .entry(subject_id.to_string())
            .and_modify(|u| u.apply(delta))
            .or_insert_with(|| UsageEntry::from(delta));
    }

    // ── Admin: set / delete limits ────────────────────────────────────────────

    pub fn set_limit(&self, subject_id: &str, quota: QuotaEntry) {
        self.limits.insert(subject_id.to_string(), quota);
    }

    #[allow(dead_code)] // Phase 2: wired up when user-deletion events arrive
    pub fn delete_subject(&self, subject_id: &str) {
        self.usage.remove(subject_id);
        self.limits.remove(subject_id);
    }

    // ── Flush snapshot for persistence ───────────────────────────────────────

    /// Collect all usage entries for the periodic Redis flush.
    pub fn snapshot_usage(&self) -> Vec<(String, UsageEntry)> {
        self.usage.iter().map(|e| (e.key().clone(), *e.value())).collect()
    }

}

impl Default for MemoryCache {
    fn default() -> Self {
        Self::new()
    }
}

// ── Pure logic ────────────────────────────────────────────────────────────────

/// Check whether applying `delta` to `current` would exceed `limit`.
/// Returns `Some(DenyReason)` if it would, `None` otherwise.
fn would_exceed(
    current: &UsageEntry,
    limit: &QuotaEntry,
    delta: &ResourceDelta,
) -> Option<DenyReason> {
    if !limit.is_bytes_unlimited() && delta.bytes > 0 {
        let new_bytes = current.bytes + delta.bytes;
        if new_bytes > limit.bytes_limit {
            return Some(DenyReason::UserStorageExceeded {
                used: new_bytes,
                limit: limit.bytes_limit,
            });
        }
    }

    if !limit.is_objects_unlimited() && delta.objects > 0 {
        let new_objects = current.objects + delta.objects;
        if new_objects > limit.objects_limit {
            return Some(DenyReason::UserObjectLimitReached {
                used: new_objects,
                limit: limit.objects_limit,
            });
        }
    }

    if !limit.is_buckets_unlimited() && delta.buckets > 0 {
        let new_buckets = current.buckets + delta.buckets;
        if new_buckets > limit.buckets_limit {
            return Some(DenyReason::UserBucketLimitReached {
                used: new_buckets,
                limit: limit.buckets_limit,
            });
        }
    }

    None
}
