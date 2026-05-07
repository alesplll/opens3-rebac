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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resource_delta_negate_reverses_sign() {
        let d = ResourceDelta {
            bytes: 100,
            objects: 2,
            buckets: 1,
        };
        let neg = d.negate();
        assert_eq!(neg.bytes, -100);
        assert_eq!(neg.objects, -2);
        assert_eq!(neg.buckets, -1);
    }

    #[test]
    fn resource_delta_negate_of_zero_is_zero() {
        let d = ResourceDelta::default();
        let neg = d.negate();
        assert_eq!(neg.bytes, 0);
        assert_eq!(neg.objects, 0);
        assert_eq!(neg.buckets, 0);
    }

    #[test]
    fn usage_entry_apply_clamps_to_zero_on_underflow() {
        let mut u = UsageEntry {
            bytes: 50,
            objects: 1,
            buckets: 0,
        };
        u.apply(&ResourceDelta {
            bytes: -200,
            objects: -5,
            buckets: -1,
        });
        assert_eq!(u.bytes, 0);
        assert_eq!(u.objects, 0);
        assert_eq!(u.buckets, 0);
    }

    #[test]
    fn usage_entry_from_delta_ignores_negative() {
        let d = ResourceDelta {
            bytes: -100,
            objects: -1,
            buckets: 0,
        };
        let u = UsageEntry::from(&d);
        assert_eq!(u.bytes, 0);
        assert_eq!(u.objects, 0);
    }

    #[test]
    fn quota_entry_unlimited_flags() {
        let q = QuotaEntry {
            bytes_limit: QuotaEntry::UNLIMITED,
            objects_limit: 10,
            buckets_limit: QuotaEntry::UNLIMITED,
        };
        assert!(q.is_bytes_unlimited());
        assert!(!q.is_objects_unlimited());
        assert!(q.is_buckets_unlimited());
    }

    #[test]
    fn deny_reason_human_readable_contains_values() {
        let r = DenyReason::UserStorageExceeded {
            used: 1100,
            limit: 1000,
        };
        let s = r.human_readable();
        assert!(s.contains("1100"));
        assert!(s.contains("1000"));
    }
}
