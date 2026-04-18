//! gRPC transport layer — translates proto ↔ domain types and delegates to QuotaService.
//! Mirrors the pattern of authz servicer.py.

use std::sync::Arc;

use tonic::{Request, Response, Status};
use tracing::instrument;

use crate::{
    domain::{CheckResult, DenyReason, QuotaEntry, ResourceDelta, QuotaError},
    repository::traits::QuotaRepository,
    service::QuotaService,
};

// Include the generated proto code
pub mod proto {
    tonic::include_proto!("opens3.quota.v1");

    pub const FILE_DESCRIPTOR_SET: &[u8] =
        include_bytes!(concat!(env!("OUT_DIR"), "/quota_descriptor.bin"));
}

use proto::{
    quota_service_server::QuotaService as QuotaServiceTrait,
    CheckQuotaRequest, CheckQuotaResponse, DenyCode,
    GetQuotaRequest, GetQuotaResponse,
    GetUsageRequest, GetUsageResponse,
    HealthCheckRequest, HealthCheckResponse,
    ResourceUsage,
    SetQuotaRequest, SetQuotaResponse,
    UpdateUsageRequest, UpdateUsageResponse,
    health_check_response::ServingStatus,
};

pub struct GrpcHandler<R: QuotaRepository> {
    service: Arc<QuotaService<R>>,
}

impl<R: QuotaRepository> GrpcHandler<R> {
    pub fn new(service: Arc<QuotaService<R>>) -> Self {
        Self { service }
    }
}

#[tonic::async_trait]
impl<R: QuotaRepository> QuotaServiceTrait for GrpcHandler<R> {
    #[instrument(skip(self), name = "grpc.check_quota")]
    async fn check_quota(
        &self,
        request: Request<CheckQuotaRequest>,
    ) -> Result<Response<CheckQuotaResponse>, Status> {
        let req = request.into_inner();
        let delta = proto_delta_to_domain(req.delta.unwrap_or_default());

        let bucket_id = if req.bucket_id.is_empty() {
            None
        } else {
            Some(req.bucket_id.as_str())
        };

        match self.service.check_quota(&req.subject_id, bucket_id, &delta) {
            Ok(CheckResult::Allowed) => Ok(Response::new(CheckQuotaResponse {
                allowed: true,
                code: DenyCode::DenyCodeUnspecified.into(),
                reason: String::new(),
            })),

            Ok(CheckResult::Denied(reason)) => {
                let (code, reason_str) = deny_reason_to_proto(&reason);
                Ok(Response::new(CheckQuotaResponse {
                    allowed: false,
                    code: code.into(),
                    reason: reason_str,
                }))
            }

            Err(QuotaError::InvalidArgument(msg)) => {
                Err(Status::invalid_argument(msg))
            }
            Err(e) => Err(Status::internal(e.to_string())),
        }
    }

    #[instrument(skip(self), name = "grpc.update_usage")]
    async fn update_usage(
        &self,
        request: Request<UpdateUsageRequest>,
    ) -> Result<Response<UpdateUsageResponse>, Status> {
        let req = request.into_inner();
        let delta = proto_delta_to_domain(req.delta.unwrap_or_default());

        let bucket_id = if req.bucket_id.is_empty() {
            None
        } else {
            Some(req.bucket_id.as_str())
        };

        self.service
            .update_usage(&req.subject_id, bucket_id, &delta)
            .map_err(|e| match e {
                QuotaError::InvalidArgument(m) => Status::invalid_argument(m),
                other => Status::internal(other.to_string()),
            })?;

        Ok(Response::new(UpdateUsageResponse {}))
    }

    #[instrument(skip(self), name = "grpc.get_usage")]
    async fn get_usage(
        &self,
        request: Request<GetUsageRequest>,
    ) -> Result<Response<GetUsageResponse>, Status> {
        let req = request.into_inner();

        let usage = self
            .service
            .get_usage(&req.subject_id)
            .map_err(|e| Status::internal(e.to_string()))?;

        Ok(Response::new(GetUsageResponse {
            usage: Some(ResourceUsage {
                bytes: usage.bytes,
                objects: usage.objects,
                buckets: usage.buckets,
            }),
        }))
    }

    #[instrument(skip(self), name = "grpc.set_quota")]
    async fn set_quota(
        &self,
        request: Request<SetQuotaRequest>,
    ) -> Result<Response<SetQuotaResponse>, Status> {
        let req = request.into_inner();

        let quota = QuotaEntry {
            bytes_limit: req.bytes_limit,
            objects_limit: req.objects_limit,
            buckets_limit: req.buckets_limit,
        };

        self.service
            .set_quota(&req.subject_id, quota)
            .await
            .map_err(|e| match e {
                QuotaError::InvalidArgument(m) => Status::invalid_argument(m),
                other => Status::internal(other.to_string()),
            })?;

        Ok(Response::new(SetQuotaResponse { success: true }))
    }

    #[instrument(skip(self), name = "grpc.get_quota")]
    async fn get_quota(
        &self,
        request: Request<GetQuotaRequest>,
    ) -> Result<Response<GetQuotaResponse>, Status> {
        let req = request.into_inner();

        match self.service.get_quota(&req.subject_id).await {
            Ok(Some(quota)) => Ok(Response::new(GetQuotaResponse {
                bytes_limit: quota.bytes_limit,
                objects_limit: quota.objects_limit,
                buckets_limit: quota.buckets_limit,
            })),
            Ok(None) => Err(Status::not_found(format!(
                "no quota set for subject: {}",
                req.subject_id
            ))),
            Err(e) => Err(Status::internal(e.to_string())),
        }
    }

    #[instrument(skip(self), name = "grpc.health_check")]
    async fn health_check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> Result<Response<HealthCheckResponse>, Status> {
        let status = match self.service.health().await {
            Ok(_) => ServingStatus::Serving,
            Err(_) => ServingStatus::NotServing,
        };

        Ok(Response::new(HealthCheckResponse { status: status.into() }))
    }
}

// ── Proto ↔ domain conversions ────────────────────────────────────────────────

fn proto_delta_to_domain(d: proto::ResourceDelta) -> ResourceDelta {
    ResourceDelta { bytes: d.bytes, objects: d.objects, buckets: d.buckets }
}

fn deny_reason_to_proto(reason: &DenyReason) -> (DenyCode, String) {
    match reason {
        DenyReason::UserStorageExceeded { .. } => (
            DenyCode::DenyCodeUserStorageExceeded,
            reason.human_readable(),
        ),
        DenyReason::BucketStorageExceeded { .. } => (
            DenyCode::DenyCodeBucketStorageExceeded,
            reason.human_readable(),
        ),
        DenyReason::UserBucketLimitReached { .. } => (
            DenyCode::DenyCodeUserBucketLimitReached,
            reason.human_readable(),
        ),
        DenyReason::UserObjectLimitReached { .. } => (
            DenyCode::DenyCodeUserObjectLimitReached,
            reason.human_readable(),
        ),
    }
}
