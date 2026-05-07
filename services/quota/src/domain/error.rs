use thiserror::Error;

#[derive(Debug, Error)]
pub enum QuotaError {
    #[error("redis error: {0}")]
    Redis(#[from] fred::error::RedisError),

    #[error("invalid argument: {0}")]
    InvalidArgument(String),

    #[error("internal error: {0}")]
    Internal(String),
}
