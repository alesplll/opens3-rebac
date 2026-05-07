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
//!            │  flush every 1s — only dirty entries (background task in app.rs)
//!   ┌────────▼────────────┐
//!   │   RedisRepository   │  AOF persistence
//!   └─────────────────────┘

use dashmap::{DashMap, DashSet};
use tracing::instrument;

use crate::domain::{CheckResult, DenyReason, QuotaEntry, ResourceDelta, UsageEntry};

pub struct MemoryCache {
    usage: DashMap<String, UsageEntry>,
    limits: DashMap<String, QuotaEntry>,
    dirty: DashSet<String>,
}

impl MemoryCache {
    pub fn new() -> Self {
        Self {
            usage: DashMap::new(),
            limits: DashMap::new(),
            dirty: DashSet::new(),
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
        // SetQuota is rare; the 1s flush window is our consistency boundary.
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

        let result = match denied {
            Some(reason) => CheckResult::Denied(reason),
            None => CheckResult::Allowed,
        };
        if result == CheckResult::Allowed {
            self.dirty.insert(subject_id.to_string());
        }
        result
    }

    // ── Fire-and-forget update (after successful operation) ───────────────────

    pub fn update(&self, subject_id: &str, delta: &ResourceDelta) {
        self.usage
            .entry(subject_id.to_string())
            .and_modify(|u| u.apply(delta))
            .or_insert_with(|| UsageEntry::from(delta));
        self.dirty.insert(subject_id.to_string());
    }

    // ── Admin: set / delete limits ────────────────────────────────────────────

    pub fn set_limit(&self, subject_id: &str, quota: QuotaEntry) {
        self.limits.insert(subject_id.to_string(), quota);
    }

    pub fn delete_subject(&self, subject_id: &str) {
        self.usage.remove(subject_id);
        self.limits.remove(subject_id);
        self.dirty.remove(subject_id);
    }

    // ── Flush snapshot for persistence ───────────────────────────────────────

    /// Collect only entries modified since the last flush.
    ///
    /// Atomically drains the dirty set: each key is removed before its value
    /// is read. Any concurrent write that arrives after removal re-inserts the
    /// key, so it appears in the next flush cycle — no data is ever lost.
    pub fn snapshot_dirty(&self) -> Vec<(String, UsageEntry)> {
        let keys: Vec<String> = self.dirty.iter().map(|k| k.clone()).collect();
        for k in &keys {
            self.dirty.remove(k);
        }
        keys.into_iter()
            .filter_map(|k| self.usage.get(&k).map(|v| (k, *v)))
            .collect()
    }
}

impl Default for MemoryCache {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::{CheckResult, DenyReason, QuotaEntry, ResourceDelta};

    fn lim(bytes: i64, objects: i64, buckets: i64) -> QuotaEntry {
        QuotaEntry {
            bytes_limit: bytes,
            objects_limit: objects,
            buckets_limit: buckets,
        }
    }

    fn unlimited() -> QuotaEntry {
        QuotaEntry {
            bytes_limit: QuotaEntry::UNLIMITED,
            objects_limit: QuotaEntry::UNLIMITED,
            buckets_limit: QuotaEntry::UNLIMITED,
        }
    }

    fn d(bytes: i64, objects: i64, buckets: i64) -> ResourceDelta {
        ResourceDelta {
            bytes,
            objects,
            buckets,
        }
    }

    // ── check_and_reserve ─────────────────────────────────────────────────────

    #[test]
    fn allows_first_write_within_limit() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(100, 1, 0), &lim(1000, 10, 5));
        assert_eq!(result, CheckResult::Allowed);
        let usage = cache.get_usage("user:alice").unwrap();
        assert_eq!(usage.bytes, 100);
        assert_eq!(usage.objects, 1);
    }

    #[test]
    fn allows_up_to_exact_limit() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(1000, 0, 0), &lim(1000, -1, -1));
        assert_eq!(result, CheckResult::Allowed);
    }

    #[test]
    fn denies_bytes_exceeded() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(1001, 0, 0), &lim(1000, -1, -1));
        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::UserStorageExceeded { .. })
        ));
    }

    #[test]
    fn denies_objects_exceeded() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(0, 11, 0), &lim(-1, 10, -1));
        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::UserObjectLimitReached { .. })
        ));
    }

    #[test]
    fn denies_buckets_exceeded() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(0, 0, 6), &lim(-1, -1, 5));
        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::UserBucketLimitReached { .. })
        ));
    }

    #[test]
    fn unlimited_always_allows_large_delta() {
        let cache = MemoryCache::new();
        let result = cache.check_and_reserve("user:alice", &d(i64::MAX / 2, 0, 0), &unlimited());
        assert_eq!(result, CheckResult::Allowed);
    }

    #[test]
    fn denied_check_does_not_modify_usage() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(500, 1, 0));

        let result = cache.check_and_reserve("user:alice", &d(600, 0, 0), &lim(1000, -1, -1));
        assert!(matches!(result, CheckResult::Denied(..)));

        // Usage must be unchanged after denial
        let usage = cache.get_usage("user:alice").unwrap();
        assert_eq!(usage.bytes, 500);
        assert_eq!(usage.objects, 1);
    }

    #[test]
    fn explicit_limit_overrides_default() {
        let cache = MemoryCache::new();
        // Custom limit is 500 bytes; default (passed as fallback) is 10 000
        cache.set_limit("user:alice", lim(500, -1, -1));
        let result = cache.check_and_reserve("user:alice", &d(600, 0, 0), &lim(10_000, -1, -1));
        assert!(matches!(
            result,
            CheckResult::Denied(DenyReason::UserStorageExceeded { .. })
        ));
    }

    #[test]
    fn negative_delta_in_check_is_always_allowed() {
        let cache = MemoryCache::new();
        // Releasing resources (delete) should never be denied
        let result = cache.check_and_reserve("user:alice", &d(-100, -1, 0), &lim(1000, 10, 5));
        assert_eq!(result, CheckResult::Allowed);
    }

    // ── update ────────────────────────────────────────────────────────────────

    #[test]
    fn update_accumulates_usage() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(100, 1, 0));
        cache.update("user:alice", &d(200, 2, 1));
        let usage = cache.get_usage("user:alice").unwrap();
        assert_eq!(usage.bytes, 300);
        assert_eq!(usage.objects, 3);
        assert_eq!(usage.buckets, 1);
    }

    #[test]
    fn update_negative_clamps_at_zero() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(100, 1, 0));
        cache.update("user:alice", &d(-999, -999, 0));
        let usage = cache.get_usage("user:alice").unwrap();
        assert_eq!(usage.bytes, 0);
        assert_eq!(usage.objects, 0);
    }

    // ── snapshot_dirty ────────────────────────────────────────────────────────

    #[test]
    fn snapshot_dirty_returns_only_modified_entries() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(100, 1, 0));
        cache.update("user:bob", &d(200, 2, 1));
        let snap = cache.snapshot_dirty();
        assert_eq!(snap.len(), 2);
    }

    #[test]
    fn snapshot_dirty_empty_when_nothing_changed() {
        let cache = MemoryCache::new();
        assert!(cache.snapshot_dirty().is_empty());
    }

    #[test]
    fn snapshot_dirty_clears_after_flush() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(100, 1, 0));
        let first = cache.snapshot_dirty();
        assert_eq!(first.len(), 1);
        // Second flush: no changes since last flush
        let second = cache.snapshot_dirty();
        assert!(second.is_empty());
    }

    #[test]
    fn load_usage_does_not_mark_dirty() {
        let cache = MemoryCache::new();
        // Startup: data loaded from Redis — must NOT be written back
        cache.load_usage(vec![(
            "user:alice".into(),
            UsageEntry {
                bytes: 100,
                objects: 1,
                buckets: 0,
            },
        )]);
        assert!(cache.snapshot_dirty().is_empty());
    }

    #[test]
    fn check_and_reserve_allowed_marks_dirty() {
        let cache = MemoryCache::new();
        let lim = lim(1000, -1, -1);
        cache.check_and_reserve("user:alice", &d(100, 0, 0), &lim);
        assert_eq!(cache.snapshot_dirty().len(), 1);
    }

    #[test]
    fn check_and_reserve_denied_does_not_mark_dirty() {
        let cache = MemoryCache::new();
        let lim = lim(50, -1, -1);
        cache.check_and_reserve("user:alice", &d(100, 0, 0), &lim);
        assert!(cache.snapshot_dirty().is_empty());
    }

    #[test]
    fn delete_subject_removes_from_dirty() {
        let cache = MemoryCache::new();
        cache.update("user:alice", &d(100, 1, 0));
        cache.delete_subject("user:alice");
        // After delete the entry is gone — dirty set should also be cleared
        assert!(cache.snapshot_dirty().is_empty());
    }

    // ── concurrent safety ────────────────────────────────────────────────────

    #[test]
    fn concurrent_updates_are_consistent() {
        use std::{sync::Arc, thread};

        let cache = Arc::new(MemoryCache::new());
        let threads: Vec<_> = (0..10)
            .map(|_| {
                let c = Arc::clone(&cache);
                thread::spawn(move || {
                    for _ in 0..100 {
                        c.update("user:shared", &d(1, 0, 0));
                    }
                })
            })
            .collect();

        for t in threads {
            t.join().unwrap();
        }

        let usage = cache.get_usage("user:shared").unwrap();
        assert_eq!(usage.bytes, 1000); // 10 threads × 100 updates × 1 byte
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
