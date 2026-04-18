//! Redis-backed quota repository using the `fred` async client.
//!
//! Key schema:
//!   quota:usage:{subject_id}  → HASH { bytes, objects, buckets }
//!   quota:limit:{subject_id}  → HASH { bytes_limit, objects_limit, buckets_limit }
//!
//! All writes use pipelining (fred Pool) to minimise round trips.
//! SCAN is used instead of KEYS to safely iterate all quota entries at startup.

use std::sync::Arc;

use fred::{interfaces::ScanInterface, prelude::*};
use tracing::{debug, instrument, warn};

use crate::domain::{QuotaEntry, QuotaError, UsageEntry};
use super::traits::QuotaRepository;

const USAGE_PREFIX: &str = "quota:usage:";
const LIMIT_PREFIX: &str = "quota:limit:";

pub struct RedisRepository {
    pool: Arc<RedisPool>,
}

impl RedisRepository {
    pub async fn connect(url: &str, pool_size: usize) -> Result<Self, QuotaError> {
        let config = RedisConfig::from_url(url)
            .map_err(|e| QuotaError::Internal(format!("invalid Redis URL: {e}")))?;

        let pool = Builder::from_config(config)
            .build_pool(pool_size)
            .map_err(|e| QuotaError::Internal(format!("failed to build Redis pool: {e}")))?;

        pool.connect();
        pool.wait_for_connect()
            .await
            .map_err(QuotaError::Redis)?;

        Ok(Self { pool: Arc::new(pool) })
    }
}

#[async_trait::async_trait]
impl QuotaRepository for RedisRepository {
    #[instrument(skip(self), name = "redis.load_all_usage")]
    async fn load_all_usage(&self) -> Result<Vec<(String, UsageEntry)>, QuotaError> {
        let keys = scan_keys(&self.pool, &format!("{USAGE_PREFIX}*")).await?;

        if keys.is_empty() {
            return Ok(vec![]);
        }

        let mut result = Vec::with_capacity(keys.len());
        for key in &keys {
            match load_usage_entry(&self.pool, key).await {
                Ok(Some(entry)) => {
                    let subject = key.trim_start_matches(USAGE_PREFIX).to_string();
                    result.push((subject, entry));
                }
                Ok(None) => {}
                Err(e) => warn!(key = %key, error = %e, "failed to load usage entry, skipping"),
            }
        }

        debug!(count = result.len(), "loaded usage entries from Redis");
        Ok(result)
    }

    #[instrument(skip(self), name = "redis.load_all_limits")]
    async fn load_all_limits(&self) -> Result<Vec<(String, QuotaEntry)>, QuotaError> {
        let keys = scan_keys(&self.pool, &format!("{LIMIT_PREFIX}*")).await?;

        if keys.is_empty() {
            return Ok(vec![]);
        }

        let mut result = Vec::with_capacity(keys.len());
        for key in &keys {
            match load_limit_entry(&self.pool, key).await {
                Ok(Some(entry)) => {
                    let subject = key.trim_start_matches(LIMIT_PREFIX).to_string();
                    result.push((subject, entry));
                }
                Ok(None) => {}
                Err(e) => warn!(key = %key, error = %e, "failed to load limit entry, skipping"),
            }
        }

        debug!(count = result.len(), "loaded limit entries from Redis");
        Ok(result)
    }

    #[instrument(skip(self, entries), name = "redis.flush_usage", fields(count = entries.len()))]
    async fn flush_usage(&self, entries: &[(String, UsageEntry)]) -> Result<(), QuotaError> {
        for (subject, usage) in entries {
            let key = format!("{USAGE_PREFIX}{subject}");
            self.pool
                .hset::<(), _, _>(
                    &key,
                    vec![
                        ("bytes", usage.bytes),
                        ("objects", usage.objects),
                        ("buckets", usage.buckets),
                    ],
                )
                .await
                .map_err(QuotaError::Redis)?;
        }
        Ok(())
    }

    #[instrument(skip(self, entries), name = "redis.flush_limits", fields(count = entries.len()))]
    async fn flush_limits(&self, entries: &[(String, QuotaEntry)]) -> Result<(), QuotaError> {
        for (subject, quota) in entries {
            let key = format!("{LIMIT_PREFIX}{subject}");
            self.pool
                .hset::<(), _, _>(
                    &key,
                    vec![
                        ("bytes_limit", quota.bytes_limit),
                        ("objects_limit", quota.objects_limit),
                        ("buckets_limit", quota.buckets_limit),
                    ],
                )
                .await
                .map_err(QuotaError::Redis)?;
        }
        Ok(())
    }

    #[instrument(skip(self), name = "redis.get_limit")]
    async fn get_limit(&self, subject_id: &str) -> Result<Option<QuotaEntry>, QuotaError> {
        let key = format!("{LIMIT_PREFIX}{subject_id}");
        load_limit_entry(&self.pool, &key).await
    }

    #[instrument(skip(self), name = "redis.delete_subject")]
    async fn delete_subject(&self, subject_id: &str) -> Result<(), QuotaError> {
        let usage_key = format!("{USAGE_PREFIX}{subject_id}");
        let limit_key = format!("{LIMIT_PREFIX}{subject_id}");
        self.pool
            .del::<(), _>(vec![usage_key, limit_key])
            .await
            .map_err(QuotaError::Redis)?;
        Ok(())
    }

    async fn health(&self) -> Result<(), QuotaError> {
        self.pool.ping::<()>().await.map_err(QuotaError::Redis)
    }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

async fn scan_keys(pool: &RedisPool, pattern: &str) -> Result<Vec<String>, QuotaError> {
    use futures::StreamExt;
    // Use UFCS to avoid ambiguity with StreamExt::scan combinator
    let mut scanner = ScanInterface::scan(pool, pattern, Some(100_u32), None);
    let mut keys = Vec::new();
    while let Some(result) = scanner.next().await {
        let page: fred::types::ScanResult = result.map_err(QuotaError::Redis)?;
        if let Some(page_keys) = page.results() {
            for key in page_keys {
                if let Some(s) = key.as_str() {
                    keys.push(s.to_string());
                }
            }
        }
    }
    Ok(keys)
}

async fn load_usage_entry(pool: &RedisPool, key: &str) -> Result<Option<UsageEntry>, QuotaError> {
    let fields: Vec<Option<i64>> = pool
        .hmget(key, vec!["bytes", "objects", "buckets"])
        .await
        .map_err(QuotaError::Redis)?;

    if fields.iter().all(|f| f.is_none()) {
        return Ok(None);
    }

    Ok(Some(UsageEntry {
        bytes: fields[0].unwrap_or(0),
        objects: fields[1].unwrap_or(0),
        buckets: fields[2].unwrap_or(0),
    }))
}

async fn load_limit_entry(pool: &RedisPool, key: &str) -> Result<Option<QuotaEntry>, QuotaError> {
    let fields: Vec<Option<i64>> = pool
        .hmget(key, vec!["bytes_limit", "objects_limit", "buckets_limit"])
        .await
        .map_err(QuotaError::Redis)?;

    if fields.iter().all(|f| f.is_none()) {
        return Ok(None);
    }

    Ok(Some(QuotaEntry {
        bytes_limit: fields[0].unwrap_or(-1),
        objects_limit: fields[1].unwrap_or(-1),
        buckets_limit: fields[2].unwrap_or(-1),
    }))
}
