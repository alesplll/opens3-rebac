//! gRPC transport layer tests — in-process server + real client.
//!
//! Tests the full proto ↔ domain translation without external deps:
//! NoopRepository keeps everything in-memory, no Redis needed.

use std::sync::Arc;

use tokio::net::TcpListener;
use tokio_stream::wrappers::TcpListenerStream;
use tonic::transport::{Channel, Endpoint};

use quota_service::{
    cache::MemoryCache,
    config,
    domain::{QuotaEntry, QuotaError, UsageEntry},
    metrics::QuotaMetrics,
    repository::traits::QuotaRepository,
    service::QuotaService,
    transport::grpc::{proto, GrpcHandler},
};

use proto::{
    quota_service_client::QuotaServiceClient, quota_service_server::QuotaServiceServer,
    CheckQuotaRequest, DeleteSubjectRequest, GetUsageRequest, ResourceDelta, SetQuotaRequest,
    UpdateUsageRequest,
};

// ── No-op repository ──────────────────────────────────────────────────────────

#[derive(Default)]
struct NoopRepo;

#[async_trait::async_trait]
impl QuotaRepository for NoopRepo {
    async fn load_all_usage(&self) -> Result<Vec<(String, UsageEntry)>, QuotaError> {
        Ok(vec![])
    }
    async fn load_all_limits(&self) -> Result<Vec<(String, QuotaEntry)>, QuotaError> {
        Ok(vec![])
    }
    async fn flush_usage(&self, _: &[(String, UsageEntry)]) -> Result<(), QuotaError> {
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

// ── Test server helpers ───────────────────────────────────────────────────────

fn init_config() {
    std::env::set_var("DEFAULT_USER_BYTES_LIMIT", "10737418240");
    std::env::set_var("DEFAULT_USER_OBJECTS_LIMIT", "-1");
    std::env::set_var("DEFAULT_USER_BUCKETS_LIMIT", "100");
    std::env::set_var("DEFAULT_BUCKET_BYTES_LIMIT", "-1");
    std::env::set_var("DEFAULT_BUCKET_OBJECTS_LIMIT", "-1");
    config::init();
}

async fn spawn_server() -> QuotaServiceClient<Channel> {
    init_config();

    let cache = Arc::new(MemoryCache::new());
    let repo = Arc::new(NoopRepo);
    let metrics = Arc::new(QuotaMetrics::new());
    let service = Arc::new(QuotaService::new(cache, repo, metrics));
    let handler = GrpcHandler::new(service);
    let svc = QuotaServiceServer::new(handler);

    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();

    tokio::spawn(async move {
        tonic::transport::Server::builder()
            .add_service(svc)
            .serve_with_incoming(TcpListenerStream::new(listener))
            .await
            .unwrap();
    });

    let channel = Endpoint::from_shared(format!("http://{addr}"))
        .unwrap()
        .connect()
        .await
        .unwrap();

    QuotaServiceClient::new(channel)
}

fn delta(bytes: i64, objects: i64, buckets: i64) -> Option<ResourceDelta> {
    Some(ResourceDelta {
        bytes,
        objects,
        buckets,
    })
}

// ── CheckQuota ────────────────────────────────────────────────────────────────

#[tokio::test]
async fn check_quota_allows_within_limit() {
    let mut client = spawn_server().await;

    let resp = client
        .check_quota(CheckQuotaRequest {
            subject_id: "user:alice".into(),
            bucket_id: String::new(),
            delta: delta(1024, 1, 0),
        })
        .await
        .unwrap()
        .into_inner();

    assert!(resp.allowed);
    assert_eq!(resp.reason, "");
}

#[tokio::test]
async fn check_quota_denies_when_limit_exceeded() {
    let mut client = spawn_server().await;

    // First set a tiny limit for the user
    client
        .set_quota(SetQuotaRequest {
            subject_id: "user:bob".into(),
            bytes_limit: 100,
            objects_limit: -1,
            buckets_limit: -1,
        })
        .await
        .unwrap();

    let resp = client
        .check_quota(CheckQuotaRequest {
            subject_id: "user:bob".into(),
            bucket_id: String::new(),
            delta: delta(200, 0, 0),
        })
        .await
        .unwrap()
        .into_inner();

    assert!(!resp.allowed);
    assert!(!resp.reason.is_empty());
}

#[tokio::test]
async fn check_quota_empty_subject_returns_invalid_argument() {
    let mut client = spawn_server().await;

    let status = client
        .check_quota(CheckQuotaRequest {
            subject_id: String::new(),
            bucket_id: String::new(),
            delta: delta(100, 0, 0),
        })
        .await
        .unwrap_err();

    assert_eq!(status.code(), tonic::Code::InvalidArgument);
}

// ── UpdateUsage ───────────────────────────────────────────────────────────────

#[tokio::test]
async fn update_usage_then_get_usage_reflects_change() {
    let mut client = spawn_server().await;

    client
        .update_usage(UpdateUsageRequest {
            subject_id: "user:carol".into(),
            bucket_id: "bucket:photos".into(),
            delta: delta(512, 1, 0),
        })
        .await
        .unwrap();

    let user_usage = client
        .get_usage(GetUsageRequest {
            subject_id: "user:carol".into(),
        })
        .await
        .unwrap()
        .into_inner()
        .usage
        .unwrap();

    assert_eq!(user_usage.bytes, 512);
    assert_eq!(user_usage.objects, 1);

    let bucket_usage = client
        .get_usage(GetUsageRequest {
            subject_id: "bucket:photos".into(),
        })
        .await
        .unwrap()
        .into_inner()
        .usage
        .unwrap();

    assert_eq!(bucket_usage.bytes, 512);
}

#[tokio::test]
async fn update_usage_empty_subject_returns_invalid_argument() {
    let mut client = spawn_server().await;

    let status = client
        .update_usage(UpdateUsageRequest {
            subject_id: String::new(),
            bucket_id: String::new(),
            delta: delta(100, 0, 0),
        })
        .await
        .unwrap_err();

    assert_eq!(status.code(), tonic::Code::InvalidArgument);
}

// ── SetQuota / GetQuota ───────────────────────────────────────────────────────

#[tokio::test]
async fn set_quota_then_get_quota_roundtrip() {
    let mut client = spawn_server().await;

    client
        .set_quota(SetQuotaRequest {
            subject_id: "user:dave".into(),
            bytes_limit: 5_000_000_000,
            objects_limit: 1000,
            buckets_limit: 10,
        })
        .await
        .unwrap();

    let quota = client
        .get_quota(proto::GetQuotaRequest {
            subject_id: "user:dave".into(),
        })
        .await
        .unwrap()
        .into_inner();

    assert_eq!(quota.bytes_limit, 5_000_000_000);
    assert_eq!(quota.objects_limit, 1000);
    assert_eq!(quota.buckets_limit, 10);
}

#[tokio::test]
async fn get_quota_not_found_for_unknown_subject() {
    let mut client = spawn_server().await;

    let status = client
        .get_quota(proto::GetQuotaRequest {
            subject_id: "user:nobody".into(),
        })
        .await
        .unwrap_err();

    assert_eq!(status.code(), tonic::Code::NotFound);
}

// ── DeleteSubject ─────────────────────────────────────────────────────────────

#[tokio::test]
async fn delete_subject_removes_usage() {
    let mut client = spawn_server().await;

    client
        .update_usage(UpdateUsageRequest {
            subject_id: "user:eve".into(),
            bucket_id: String::new(),
            delta: delta(1024, 3, 0),
        })
        .await
        .unwrap();

    client
        .delete_subject(DeleteSubjectRequest {
            subject_id: "user:eve".into(),
        })
        .await
        .unwrap();

    let usage = client
        .get_usage(GetUsageRequest {
            subject_id: "user:eve".into(),
        })
        .await
        .unwrap()
        .into_inner()
        .usage
        .unwrap();

    assert_eq!(usage.bytes, 0, "usage must be zero after delete");
    assert_eq!(usage.objects, 0);
}

#[tokio::test]
async fn delete_subject_empty_id_returns_invalid_argument() {
    let mut client = spawn_server().await;

    let status = client
        .delete_subject(DeleteSubjectRequest {
            subject_id: String::new(),
        })
        .await
        .unwrap_err();

    assert_eq!(status.code(), tonic::Code::InvalidArgument);
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

#[tokio::test]
async fn health_check_returns_serving() {
    let mut client = spawn_server().await;

    let resp = client
        .health_check(proto::HealthCheckRequest {
            service: String::new(),
        })
        .await
        .unwrap()
        .into_inner();

    assert_eq!(
        resp.status,
        proto::health_check_response::ServingStatus::Serving as i32
    );
}
