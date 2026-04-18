//! gRPC transport layer — translates proto ↔ domain types and delegates to QuotaService.
//! Mirrors the pattern of authz servicer.py.

use std::sync::Arc;

use tonic::{Request, Response, Status};
use tracing::instrument;

use crate::{
    domain::{CheckResult, DenyReason, QuotaEntry, QuotaError, ResourceDelta},
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
    health_check_response::ServingStatus, quota_service_server::QuotaService as QuotaServiceTrait,
    CheckQuotaRequest, CheckQuotaResponse, DeleteSubjectRequest, DeleteSubjectResponse, DenyCode,
    GetQuotaRequest, GetQuotaResponse, GetUsageRequest, GetUsageResponse, HealthCheckRequest,
    HealthCheckResponse, ResourceUsage, SetQuotaRequest, SetQuotaResponse, UpdateUsageRequest,
    UpdateUsageResponse,
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
    #[instrument(
        skip(self, request), name = "grpc.check_quota",
        fields(subject_id = tracing::field::Empty, allowed = tracing::field::Empty)
    )]
    async fn check_quota(
        &self,
        request: Request<CheckQuotaRequest>,
    ) -> Result<Response<CheckQuotaResponse>, Status> {
        let req = request.into_inner();
        let span = tracing::Span::current();
        span.record("subject_id", req.subject_id.as_str());

        let delta = proto_delta_to_domain(req.delta.unwrap_or_default());
        let bucket_id = if req.bucket_id.is_empty() {
            None
        } else {
            Some(req.bucket_id.as_str())
        };

        match self.service.check_quota(&req.subject_id, bucket_id, &delta) {
            Ok(CheckResult::Allowed) => {
                span.record("allowed", true);
                Ok(Response::new(CheckQuotaResponse {
                    allowed: true,
                    code: DenyCode::Unspecified.into(),
                    reason: String::new(),
                }))
            }
            Ok(CheckResult::Denied(reason)) => {
                span.record("allowed", false);
                let (code, reason_str) = deny_reason_to_proto(&reason);
                Ok(Response::new(CheckQuotaResponse {
                    allowed: false,
                    code: code.into(),
                    reason: reason_str,
                }))
            }
            Err(QuotaError::InvalidArgument(msg)) => Err(Status::invalid_argument(msg)),
            Err(e) => Err(Status::internal(e.to_string())),
        }
    }

    #[instrument(
        skip(self, request), name = "grpc.update_usage",
        fields(subject_id = tracing::field::Empty)
    )]
    async fn update_usage(
        &self,
        request: Request<UpdateUsageRequest>,
    ) -> Result<Response<UpdateUsageResponse>, Status> {
        let req = request.into_inner();
        tracing::Span::current().record("subject_id", req.subject_id.as_str());

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

    #[instrument(
        skip(self, request), name = "grpc.get_usage",
        fields(subject_id = tracing::field::Empty)
    )]
    async fn get_usage(
        &self,
        request: Request<GetUsageRequest>,
    ) -> Result<Response<GetUsageResponse>, Status> {
        let req = request.into_inner();
        tracing::Span::current().record("subject_id", req.subject_id.as_str());

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

    #[instrument(
        skip(self, request), name = "grpc.set_quota",
        fields(subject_id = tracing::field::Empty)
    )]
    async fn set_quota(
        &self,
        request: Request<SetQuotaRequest>,
    ) -> Result<Response<SetQuotaResponse>, Status> {
        let req = request.into_inner();
        tracing::Span::current().record("subject_id", req.subject_id.as_str());

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

    #[instrument(
        skip(self, request), name = "grpc.get_quota",
        fields(subject_id = tracing::field::Empty)
    )]
    async fn get_quota(
        &self,
        request: Request<GetQuotaRequest>,
    ) -> Result<Response<GetQuotaResponse>, Status> {
        let req = request.into_inner();
        tracing::Span::current().record("subject_id", req.subject_id.as_str());

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

    #[instrument(
        skip(self, request), name = "grpc.delete_subject",
        fields(subject_id = tracing::field::Empty)
    )]
    async fn delete_subject(
        &self,
        request: Request<DeleteSubjectRequest>,
    ) -> Result<Response<DeleteSubjectResponse>, Status> {
        let req = request.into_inner();
        tracing::Span::current().record("subject_id", req.subject_id.as_str());

        self.service
            .delete_subject(&req.subject_id)
            .await
            .map_err(|e| match e {
                QuotaError::InvalidArgument(m) => Status::invalid_argument(m),
                other => Status::internal(other.to_string()),
            })?;

        Ok(Response::new(DeleteSubjectResponse { success: true }))
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

        Ok(Response::new(HealthCheckResponse {
            status: status.into(),
        }))
    }
}

// ── Proto ↔ domain conversions ────────────────────────────────────────────────

fn proto_delta_to_domain(d: proto::ResourceDelta) -> ResourceDelta {
    ResourceDelta {
        bytes: d.bytes,
        objects: d.objects,
        buckets: d.buckets,
    }
}

fn deny_reason_to_proto(reason: &DenyReason) -> (DenyCode, String) {
    match reason {
        DenyReason::UserStorageExceeded { .. } => {
            (DenyCode::UserStorageExceeded, reason.human_readable())
        }
        DenyReason::BucketStorageExceeded { .. } => {
            (DenyCode::BucketStorageExceeded, reason.human_readable())
        }
        DenyReason::UserBucketLimitReached { .. } => {
            (DenyCode::UserBucketLimitReached, reason.human_readable())
        }
        DenyReason::UserObjectLimitReached { .. } => {
            (DenyCode::UserObjectLimitReached, reason.human_readable())
        }
    }
}
