@startuml ClassDiagram_opens3

skinparam classAttributeIconSize 0
skinparam classFontStyle bold
skinparam classBackgroundColor #FAFAFA
skinparam classBorderColor #888
skinparam arrowColor #444
skinparam classHeaderBackgroundColor #E8F4F8
skinparam packageBackgroundColor #F5F5F5
skinparam packageBorderColor #AAAAAA

' ──────────────────────────────────────
'  DOMAIN MODEL
' ──────────────────────────────────────

package "Доменная модель" {

  class User {
    + id : UUID
    + email : String  <<unique>>
    + password_hash : String  <<bcrypt>>
    + created_at : Timestamp
    + updated_at : Timestamp
    --
    + validateCredentials(password) : Boolean
  }

  class Token {
    + access_token : String  <<JWT>>
    + refresh_token : String  <<JWT>>
    + expires_at : Timestamp
    --
    + isExpired() : Boolean
  }

  class Bucket {
    + id : UUID
    + name : String  <<unique>>
    + owner_id : UUID
    + created_at : Timestamp
    --
    + getObjects() : Object[]
  }

  class Object {
    + id : UUID
    + bucket_id : UUID
    + key : String
    + current_version_id : UUID  <<nullable>>
    --
    + getCurrentVersion() : Version
    + listVersions() : Version[]
  }

  class Version {
    + id : UUID
    + object_id : UUID
    + blob_id : UUID
    + size : Int64
    + etag : String
    + content_type : String
    + created_at : Timestamp
    --
    + getBlob() : Blob
  }

  class Blob {
    + id : UUID
    + path : String
    + size : Int64
    + checksum : String
    --
    + exists() : Boolean
  }

}

' ──────────────────────────────────────
'  SERVICE LAYER
' ──────────────────────────────────────

package "Слой сервисов" {

  class GatewayService <<service>> {
    - authClient : AuthService
    - authzClient : AuthZService
    - metadataClient : MetadataService
    - storageClient : StorageService
    --
    + putObject(req) : PutObjectResponse
    + getObject(req) : GetObjectResponse
    + deleteObject(req) : void
    + createBucket(req) : Bucket
    + deleteBucket(req) : void
    + listObjects(req) : Object[]
  }

  class AuthService <<service>> {
    - usersClient : UsersService
    - cache : RedisClient
    - rateLimiter : RateLimiter
    --
    + login(email, password) : Token
    + getRefreshToken(refreshToken) : Token
    + getAccessToken(refreshToken) : String
    + validateToken(token) : Boolean
    + healthCheck() : Status
  }

  class UsersService <<service>> {
    - db : PostgreSQLClient
    - kafka : KafkaProducer
    --
    + create(email, password) : User
    + get(id) : User
    + update(id, data) : User
    + delete(id) : void
    + validateCredentials(email, password) : Boolean
    + healthCheck() : Status
  }

  class AuthZService <<service>> {
    - graph : Neo4jClient
    - cache : RedisClient
    - kafka : KafkaProducer
    --
    + check(subject, action, resource) : Boolean
    + writeTuple(subject, relation, resource) : void
    + deleteTuple(subject, relation, resource) : void
    + read(resource) : Tuple[]
    + healthCheck() : Status
  }

  class MetadataService <<service>> {
    - db : PostgreSQLClient
    - kafka : KafkaProducer
    --
    + createBucket(name, ownerId) : Bucket
    + deleteBucket(bucketId) : void
    + commitVersion(objectId, blobId, size, etag) : Version
    + markDeleted(objectId) : void
    + listObjects(bucketId, prefix, limit) : Object[]
    + headObject(bucketId, key) : Version
    + healthCheck() : Status
  }

  class StorageService <<service>> {
    - dataDir : String
    - kafka : KafkaProducer
    --
    + storeBlob(data : Stream) : Blob
    + getBlob(blobId) : Stream
    + deleteBlob(blobId) : void
    + healthCheck() : Status
  }

}

' ──────────────────────────────────────
'  RELATIONS — DOMAIN
' ──────────────────────────────────────

' Пользователь владеет бакетами (агрегация: бакеты могут передаваться)
User "1" o-- "0..*" Bucket : владеет >

' Композиция: Object не существует без Bucket
Bucket "1" *-- "0..*" Object : содержит >

' Композиция: Version не существует без Object
Object "1" *-- "0..*" Version : хранит >

' Ассоциация: Version ссылается на Blob (Blob живёт в Storage независимо)
Version "0..*" --> "1" Blob : ссылается на >

' Зависимость: login порождает Token
AuthService ..> Token : <<creates>>

' Ассоциация: Token принадлежит User
User "1" --> "0..*" Token : имеет >

' ──────────────────────────────────────
'  RELATIONS — SERVICES
' ──────────────────────────────────────

' Gateway orchestrates все сервисы
GatewayService ..> AuthService    : <<uses>>
GatewayService ..> AuthZService   : <<uses>>
GatewayService ..> MetadataService : <<uses>>
GatewayService ..> StorageService  : <<uses>>

' Auth зависит от Users для проверки credentials
AuthService ..> UsersService : <<uses>>

' Сервисы управляют своими доменными классами
UsersService ..> User    : <<manages>>
MetadataService ..> Bucket  : <<manages>>
MetadataService ..> Object  : <<manages>>
MetadataService ..> Version : <<manages>>
StorageService ..> Blob     : <<manages>>

@enduml
