@startuml ObjectDiagram_opens3

skinparam objectAttributeIconSize 0
skinparam objectBackgroundColor #FAFAFA
skinparam objectBorderColor #888
skinparam arrowColor #444
skinparam objectHeaderBackgroundColor #E8F4F8
skinparam packageBackgroundColor #F5F5F5
skinparam packageBorderColor #AAAAAA

' ──────────────────────────────────────
'  СНИМОК: alice загрузила photos/cat.jpg
' ──────────────────────────────────────

package "Доменная модель" {

  object "alice : User" as alice {
    id = "a1b2c3d4-0001"
    email = "alice@example.com"
    password_hash = "$2b$12$..."
    created_at = "2026-01-10 09:00:00"
  }

  object "accessToken : Token" as token {
    access_token = "eyJhbGci...abc"
    refresh_token = "eyJhbGci...xyz"
    expires_at = "2026-03-30 12:30:00"
  }

  object "photos : Bucket" as bucket {
    id = "b1b2c3d4-0002"
    name = "photos"
    owner_id = "a1b2c3d4-0001"
    created_at = "2026-02-01 10:00:00"
  }

  object "catObj : Object" as obj {
    id = "c1b2c3d4-0003"
    bucket_id = "b1b2c3d4-0002"
    key = "photos/2026/cat.jpg"
    current_version_id = "v2b2c3d4-0005"
  }

  object "v1 : Version" as v1 {
    id = "v1b2c3d4-0004"
    object_id = "c1b2c3d4-0003"
    blob_id = "bl1b2c3d4-0006"
    size = 245760
    etag = "abc123ef"
    content_type = "image/jpeg"
    created_at = "2026-03-01 11:00:00"
  }

  object "v2 : Version" as v2 {
    id = "v2b2c3d4-0005"
    object_id = "c1b2c3d4-0003"
    blob_id = "bl2b2c3d4-0007"
    size = 258048
    etag = "def456ab"
    content_type = "image/jpeg"
    created_at = "2026-03-30 11:45:00"
  }

  object "blob1 : Blob" as blob1 {
    id = "bl1b2c3d4-0006"
    path = "/data/bl1b2c3d4-0006.bin"
    size = 245760
    checksum = "sha256:a1b2c3..."
  }

  object "blob2 : Blob" as blob2 {
    id = "bl2b2c3d4-0007"
    path = "/data/bl2b2c3d4-0007.bin"
    size = 258048
    checksum = "sha256:d4e5f6..."
  }

}

' ──────────────────────────────────────
'  СВЯЗИ
' ──────────────────────────────────────

alice "имеет" --> token
alice "владеет" --> bucket
bucket "содержит" --> obj
obj "текущая" --> v2
obj "устаревшая" --> v1
v1 --> blob1
v2 --> blob2

@enduml
