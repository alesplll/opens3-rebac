//! Redis repository integration tests.
//!
//! Requires a running Redis instance. Skipped automatically if Redis is not available.
//! Runs against DB 15 to avoid touching production/dev data.
//!
//! Run manually:
//!   cargo test -p quota-service --test redis -- --include-ignored
//!
//! Or with a custom Redis URL:
//!   TEST_REDIS_URL=redis://localhost:6379/15 cargo test -p quota-service --test redis -- --include-ignored

use quota_service::{
    domain::{QuotaEntry, ResourceDelta, UsageEntry},
    repository::{traits::QuotaRepository, RedisRepository},
};

const TEST_DB: &str = "redis://localhost:6379/15";

async fn connect() -> Option<RedisRepository> {
    let url = std::env::var("TEST_REDIS_URL").unwrap_or_else(|_| TEST_DB.to_string());
    RedisRepository::connect(&url, 2).await.ok()
}

// ── flush_usage / load_all_usage ──────────────────────────────────────────────

#[tokio::test]
#[ignore = "requires Redis"]
async fn flush_and_reload_usage_roundtrip() {
    let Some(repo) = connect().await else {
        eprintln!("skipping: Redis not available");
        return;
    };

    let subject = "test:flush_usage_roundtrip";
    let entry = UsageEntry {
        bytes: 1024,
        objects: 3,
        buckets: 1,
    };

    repo.flush_usage(&[(subject.to_string(), entry)])
        .await
        .unwrap();

    let loaded = repo.load_all_usage().await.unwrap();
    let found = loaded.iter().find(|(id, _)| id == subject);

    assert!(found.is_some(), "subject should be present after flush");
    let (_, loaded_entry) = found.unwrap();
    assert_eq!(loaded_entry.bytes, 1024);
    assert_eq!(loaded_entry.objects, 3);
    assert_eq!(loaded_entry.buckets, 1);

    // Cleanup
    repo.delete_subject(subject).await.unwrap();
}

#[tokio::test]
#[ignore = "requires Redis"]
async fn flush_usage_overwrites_existing_entry() {
    let Some(repo) = connect().await else {
        return;
    };

    let subject = "test:flush_overwrite";

    repo.flush_usage(&[(
        subject.to_string(),
        UsageEntry {
            bytes: 100,
            objects: 1,
            buckets: 0,
        },
    )])
    .await
    .unwrap();
    repo.flush_usage(&[(
        subject.to_string(),
        UsageEntry {
            bytes: 999,
            objects: 5,
            buckets: 2,
        },
    )])
    .await
    .unwrap();

    let loaded = repo.load_all_usage().await.unwrap();
    let (_, entry) = loaded.iter().find(|(id, _)| id == subject).unwrap();
    assert_eq!(entry.bytes, 999);

    repo.delete_subject(subject).await.unwrap();
}

// ── flush_limits / load_all_limits / get_limit ────────────────────────────────

#[tokio::test]
#[ignore = "requires Redis"]
async fn flush_and_get_limit_roundtrip() {
    let Some(repo) = connect().await else {
        return;
    };

    let subject = "test:flush_limit_roundtrip";
    let limit = QuotaEntry {
        bytes_limit: 5_000_000,
        objects_limit: 50,
        buckets_limit: 5,
    };

    repo.flush_limits(&[(subject.to_string(), limit)])
        .await
        .unwrap();

    let got = repo.get_limit(subject).await.unwrap();
    assert!(got.is_some());
    let got = got.unwrap();
    assert_eq!(got.bytes_limit, 5_000_000);
    assert_eq!(got.objects_limit, 50);
    assert_eq!(got.buckets_limit, 5);

    repo.delete_subject(subject).await.unwrap();
}

#[tokio::test]
#[ignore = "requires Redis"]
async fn get_limit_returns_none_for_unknown_subject() {
    let Some(repo) = connect().await else {
        return;
    };
    let got = repo
        .get_limit("test:definitely_does_not_exist_xyz123")
        .await
        .unwrap();
    assert!(got.is_none());
}

// ── delete_subject ────────────────────────────────────────────────────────────

#[tokio::test]
#[ignore = "requires Redis"]
async fn delete_subject_removes_usage_and_limit() {
    let Some(repo) = connect().await else {
        return;
    };

    let subject = "test:delete_subject";

    repo.flush_usage(&[(
        subject.to_string(),
        UsageEntry {
            bytes: 100,
            objects: 1,
            buckets: 0,
        },
    )])
    .await
    .unwrap();
    repo.flush_limits(&[(
        subject.to_string(),
        QuotaEntry {
            bytes_limit: 1000,
            objects_limit: -1,
            buckets_limit: -1,
        },
    )])
    .await
    .unwrap();

    repo.delete_subject(subject).await.unwrap();

    assert!(repo.get_limit(subject).await.unwrap().is_none());
    let usage = repo.load_all_usage().await.unwrap();
    assert!(!usage.iter().any(|(id, _)| id == subject));
}

// ── health ────────────────────────────────────────────────────────────────────

#[tokio::test]
#[ignore = "requires Redis"]
async fn health_check_ok_when_redis_available() {
    let Some(repo) = connect().await else {
        return;
    };
    assert!(repo.health().await.is_ok());
}
