use thiserror::Error;

#[derive(Debug, Error)]
pub enum QuotaError {
    #[error("redis error: {0}")]
    Redis(#[from] fred::error::RedisError),

    #[error("subject not found: {0}")]
    NotFound(String),

    #[error("invalid argument: {0}")]
    InvalidArgument(String),

    #[error("kafka error: {0}")]
    Kafka(String),

    #[error("internal error: {0}")]
    Internal(String),
}

impl QuotaError {
    pub fn internal(msg: impl Into<String>) -> Self {
        Self::Internal(msg.into())
    }
}
