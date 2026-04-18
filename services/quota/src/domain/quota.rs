//! Core domain types. Zero external dependencies — pure Rust.

/// Current resource consumption for a subject (user or bucket).
#[derive(Debug, Clone, Copy, Default)]
pub struct UsageEntry {
    pub bytes: i64,
    pub objects: i64,
    pub buckets: i64,
}

impl UsageEntry {
    pub fn apply(&mut self, delta: &ResourceDelta) {
        self.bytes = (self.bytes + delta.bytes).max(0);
        self.objects = (self.objects + delta.objects).max(0);
        self.buckets = (self.buckets + delta.buckets).max(0);
    }
}

impl From<&ResourceDelta> for UsageEntry {
    fn from(d: &ResourceDelta) -> Self {
        Self {
            bytes: d.bytes.max(0),
            objects: d.objects.max(0),
            buckets: d.buckets.max(0),
        }
    }
}

/// Resource limits for a subject. `-1` means unlimited.
#[derive(Debug, Clone, Copy)]
pub struct QuotaEntry {
    pub bytes_limit: i64,
    pub objects_limit: i64,
    pub buckets_limit: i64,
}

impl QuotaEntry {
    pub const UNLIMITED: i64 = -1;

    pub fn is_bytes_unlimited(&self) -> bool {
        self.bytes_limit == Self::UNLIMITED
    }

    pub fn is_objects_unlimited(&self) -> bool {
        self.objects_limit == Self::UNLIMITED
    }

    pub fn is_buckets_unlimited(&self) -> bool {
        self.buckets_limit == Self::UNLIMITED
    }
}

/// Intended change in resource consumption for a single operation.
/// Positive = consume, negative = release (e.g. after deletion).
#[derive(Debug, Clone, Copy, Default)]
pub struct ResourceDelta {
    pub bytes: i64,
    pub objects: i64,
    pub buckets: i64,
}

impl ResourceDelta {
    pub fn negate(self) -> Self {
        Self {
            bytes: -self.bytes,
            objects: -self.objects,
            buckets: -self.buckets,
        }
    }
}

/// Result of a quota check — either allowed or denied with a reason.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CheckResult {
    Allowed,
    Denied(DenyReason),
}

/// Structured reason for a quota denial. Maps to `DenyCode` in the proto.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DenyReason {
    UserStorageExceeded { used: i64, limit: i64 },
    BucketStorageExceeded { used: i64, limit: i64 },
    UserBucketLimitReached { used: i64, limit: i64 },
    UserObjectLimitReached { used: i64, limit: i64 },
}

impl DenyReason {
    pub fn human_readable(&self) -> String {
        match self {
            Self::UserStorageExceeded { used, limit } => {
                format!("user storage exceeded: {used}/{limit} bytes")
            }
            Self::BucketStorageExceeded { used, limit } => {
                format!("bucket storage exceeded: {used}/{limit} bytes")
            }
            Self::UserBucketLimitReached { used, limit } => {
                format!("user bucket limit reached: {used}/{limit}")
            }
            Self::UserObjectLimitReached { used, limit } => {
                format!("user object limit reached: {used}/{limit}")
            }
        }
    }
}
