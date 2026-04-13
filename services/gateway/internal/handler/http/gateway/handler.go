package gateway

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Handler struct {
	service        service.GatewayService
	authService    service.AuthService
	maxUploadSize  int64
	maxPartSize    int64
	throttleLimit  int64
	throttlePeriod time.Duration
	verifier       tokens.TokenVerifier
	router         chi.Router
}

func NewHandler(service service.GatewayService, authService service.AuthService, maxUploadSize int64, verifier tokens.TokenVerifier) *Handler {
	rateLimiterCfg := config.AppConfig().RateLimiter
	h := &Handler{
		service:        service,
		authService:    authService,
		maxUploadSize:  maxUploadSize,
		maxPartSize:    maxUploadSize,
		throttleLimit:  rateLimiterCfg.Limit(),
		throttlePeriod: rateLimiterCfg.Period(),
		verifier:       verifier,
	}
	h.router = h.newRouter()
	return h
}

func (h *Handler) Router() http.Handler {
	return h.router
}

func (h *Handler) newRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(h.rateLimitMiddleware)
	r.Use(h.requestLoggerMiddleware)
	r.Get("/health", h.health)
	r.Get("/ready", h.ready)
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh/access", h.refreshAccessToken)
	r.Post("/auth/refresh/refresh", h.refreshRefreshToken)
	r.Route("/", func(r chi.Router) {
		r.Get("/", h.listBuckets)
		r.Route("/{bucket}", func(r chi.Router) {
			r.Put("/", h.createBucket)
			r.Delete("/", h.deleteBucket)
			r.Get("/", h.bucketDispatch)
			r.Head("/", h.headBucket)
			r.Put("/*", h.putObject)
			r.Get("/*", h.getObject)
			r.Head("/*", h.headObject)
			r.Delete("/*", h.deleteObject)
			r.Post("/*", h.postObject)
		})
	})
	return r
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ready(r.Context()); err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listBuckets(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := h.service.ListBuckets(r.Context(), service.ListBucketsRequest{UserID: userID})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result := listAllMyBucketsResult{Buckets: make([]bucketListItem, 0, len(resp.Buckets))}
	for _, bucket := range resp.Buckets {
		result.Buckets = append(result.Buckets, bucketListItem{
			Name:         bucket.Name,
			CreationDate: bucket.CreatedAt.Format(time.RFC3339),
		})
	}

	writeXML(w, http.StatusOK, result)
}

func (h *Handler) createBucket(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := h.service.CreateBucket(r.Context(), service.CreateBucketRequest{
		UserID: userID,
		Bucket: chi.URLParam(r, "bucket"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/"+chi.URLParam(r, "bucket"))
	w.Header().Set("x-amz-bucket-id", resp.BucketID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteBucket(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	err = h.service.DeleteBucket(r.Context(), service.DeleteBucketRequest{UserID: userID, Bucket: chi.URLParam(r, "bucket")})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) bucketDispatch(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("list-type")
	if listType == "" || listType == "1" || listType == "2" {
		h.listObjects(w, r)
		return
	}

	h.writeError(w, r, fmt.Errorf("%w: unsupported bucket operation", domainerrors.ErrInvalidRequest))
}

func (h *Handler) headBucket(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	err = h.service.HeadBucket(r.Context(), service.HeadBucketRequest{UserID: userID, Bucket: chi.URLParam(r, "bucket")})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) putObject(w http.ResponseWriter, r *http.Request) {
	if isMultipartPartUpload(r) {
		h.uploadPart(w, r)
		return
	}

	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if r.ContentLength < 0 {
		h.writeError(w, r, fmt.Errorf("%w: content length is required", domainerrors.ErrInvalidRequest))
		return
	}

	body := http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	defer body.Close()

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp, err := h.service.PutObject(r.Context(), service.PutObjectRequest{
		UserID:      userID,
		Bucket:      chi.URLParam(r, "bucket"),
		Key:         objectKey(r),
		Body:        body,
		Size:        r.ContentLength,
		ContentType: contentType,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("ETag", resp.ETag)
	w.Header().Set("x-amz-version-id", resp.VersionID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var objectRange *service.ByteRange
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		parsedRange, err := parseRangeHeader(rangeHeader)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		objectRange = parsedRange
	}

	resp, err := h.service.GetObject(r.Context(), service.GetObjectRequest{
		UserID:    userID,
		Bucket:    chi.URLParam(r, "bucket"),
		Key:       objectKey(r),
		VersionID: r.URL.Query().Get("versionId"),
		Range:     objectRange,
		Writer:    w,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.Header().Set("ETag", resp.ETag)
	w.Header().Set("Last-Modified", resp.LastModified.UTC().Format(http.TimeFormat))
	if resp.VersionID != "" {
		w.Header().Set("x-amz-version-id", resp.VersionID)
	}
	statusCode := http.StatusOK
	if resp.Range != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", resp.Range.Start, resp.Range.End, resp.TotalSize))
		statusCode = http.StatusPartialContent
	}
	w.WriteHeader(statusCode)
}

func (h *Handler) headObject(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := h.service.HeadObject(r.Context(), service.HeadObjectRequest{
		UserID:    userID,
		Bucket:    chi.URLParam(r, "bucket"),
		Key:       objectKey(r),
		VersionID: r.URL.Query().Get("versionId"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.Header().Set("ETag", resp.ETag)
	w.Header().Set("Last-Modified", resp.LastModified.UTC().Format(http.TimeFormat))
	if resp.VersionID != "" {
		w.Header().Set("x-amz-version-id", resp.VersionID)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteObject(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("uploadId") != "" {
		h.abortMultipartUpload(w, r)
		return
	}

	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	err = h.service.DeleteObject(r.Context(), service.DeleteObjectRequest{
		UserID: userID,
		Bucket: chi.URLParam(r, "bucket"),
		Key:    objectKey(r),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) postObject(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Query().Has("uploads"):
		h.createMultipartUpload(w, r)
	case r.URL.Query().Get("uploadId") != "":
		h.completeMultipartUpload(w, r)
	default:
		h.writeError(w, r, fmt.Errorf("%w: unsupported object operation", domainerrors.ErrInvalidRequest))
	}
}

func (h *Handler) listObjects(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	maxKeys := int32(defaultMaxKeys)
	if raw := r.URL.Query().Get("max-keys"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			h.writeError(w, r, fmt.Errorf("%w: invalid max-keys", domainerrors.ErrInvalidRequest))
			return
		}
		maxKeys = int32(parsed)
	}

	resp, err := h.service.ListObjects(r.Context(), service.ListObjectsRequest{
		UserID:            userID,
		Bucket:            chi.URLParam(r, "bucket"),
		Prefix:            r.URL.Query().Get("prefix"),
		ContinuationToken: r.URL.Query().Get("continuation-token"),
		MaxKeys:           maxKeys,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result := listBucketResult{
		Name:                  chi.URLParam(r, "bucket"),
		Prefix:                r.URL.Query().Get("prefix"),
		MaxKeys:               maxKeys,
		ContinuationToken:     r.URL.Query().Get("continuation-token"),
		NextContinuationToken: resp.NextContinuationToken,
		IsTruncated:           resp.IsTruncated,
		Contents:              make([]objectListItem, 0, len(resp.Objects)),
	}
	for _, object := range resp.Objects {
		result.Contents = append(result.Contents, objectListItem{
			Key:          object.Key,
			LastModified: object.LastModified.Format(time.RFC3339),
			ETag:         ensureQuotedETag(object.ETag),
			Size:         object.Size,
		})
	}

	writeXML(w, http.StatusOK, result)
}

func (h *Handler) createMultipartUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := h.service.CreateMultipartUpload(r.Context(), service.CreateMultipartUploadRequest{
		UserID:      userID,
		Bucket:      chi.URLParam(r, "bucket"),
		Key:         objectKey(r),
		ContentType: r.Header.Get("Content-Type"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeXML(w, http.StatusOK, initiateMultipartUploadResult{
		Bucket:   chi.URLParam(r, "bucket"),
		Key:      objectKey(r),
		UploadID: resp.UploadID,
	})
}

func (h *Handler) uploadPart(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	partNumber, err := strconv.Atoi(r.URL.Query().Get("partNumber"))
	if err != nil || partNumber < 1 {
		h.writeError(w, r, fmt.Errorf("%w: invalid partNumber", domainerrors.ErrInvalidRequest))
		return
	}

	body := http.MaxBytesReader(w, r.Body, h.maxPartSize)
	defer body.Close()

	resp, err := h.service.UploadPart(r.Context(), service.UploadPartRequest{
		UserID:     userID,
		Bucket:     chi.URLParam(r, "bucket"),
		Key:        objectKey(r),
		UploadID:   r.URL.Query().Get("uploadId"),
		PartNumber: int32(partNumber),
		Body:       body,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("ETag", resp.ETag)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) completeMultipartUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var payload completeMultipartUploadXML
	if err := xml.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, r, fmt.Errorf("%w: invalid complete multipart payload", domainerrors.ErrInvalidRequest))
		return
	}

	parts := make([]service.CompletedPart, 0, len(payload.Parts))
	for _, part := range payload.Parts {
		parts = append(parts, service.CompletedPart{PartNumber: part.PartNumber, ETag: part.ETag})
	}

	resp, err := h.service.CompleteMultipartUpload(r.Context(), service.CompleteMultipartUploadRequest{
		UserID:   userID,
		Bucket:   chi.URLParam(r, "bucket"),
		Key:      objectKey(r),
		UploadID: r.URL.Query().Get("uploadId"),
		Parts:    parts,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeXML(w, http.StatusOK, completeMultipartUploadResult{
		Bucket:    chi.URLParam(r, "bucket"),
		Key:       objectKey(r),
		ETag:      resp.ETag,
		VersionID: resp.VersionID,
	})
}

func (h *Handler) abortMultipartUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractBearerUserID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	err = h.service.AbortMultipartUpload(r.Context(), service.AbortMultipartUploadRequest{
		UserID:   userID,
		Bucket:   chi.URLParam(r, "bucket"),
		Key:      objectKey(r),
		UploadID: r.URL.Query().Get("uploadId"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var payload loginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, r, fmt.Errorf("%w: invalid login payload", domainerrors.ErrInvalidRequest))
		return
	}

	resp, err := h.authService.Login(r.Context(), service.LoginRequest{
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{RefreshToken: resp.RefreshToken})
}

func (h *Handler) refreshAccessToken(w http.ResponseWriter, r *http.Request) {
	var payload refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, r, fmt.Errorf("%w: invalid refresh payload", domainerrors.ErrInvalidRequest))
		return
	}

	resp, err := h.authService.RefreshAccessToken(r.Context(), service.RefreshAccessTokenRequest{RefreshToken: payload.RefreshToken})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, refreshAccessTokenResponse{AccessToken: resp.AccessToken})
}

func (h *Handler) refreshRefreshToken(w http.ResponseWriter, r *http.Request) {
	var payload refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, r, fmt.Errorf("%w: invalid refresh payload", domainerrors.ErrInvalidRequest))
		return
	}

	resp, err := h.authService.RefreshRefreshToken(r.Context(), service.RefreshRefreshTokenRequest{RefreshToken: payload.RefreshToken})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, refreshRefreshTokenResponse{RefreshToken: resp.RefreshToken})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode, code, message := mapHTTPError(err)
	writeXMLError(w, statusCode, errorResponse{
		Code:      code,
		Message:   message,
		Resource:  r.URL.Path,
		RequestID: requestIDFromRequest(r),
	})
}

func (h *Handler) extractBearerUserID(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		return "", domainerrors.ErrUnauthorized
	}
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", domainerrors.ErrUnauthorized
	}

	claims, err := h.verifier.VerifyAccessToken(r.Context(), strings.TrimSpace(parts[1]))
	if err != nil || claims == nil || strings.TrimSpace(claims.UserId) == "" {
		return "", domainerrors.ErrUnauthorized
	}

	return strings.TrimSpace(claims.UserId), nil
}

func parseRangeHeader(value string) (*service.ByteRange, error) {
	if !strings.HasPrefix(value, "bytes=") {
		return nil, domainerrors.ErrInvalidRange
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "bytes="), "-", 2)
	if len(parts) != 2 {
		return nil, domainerrors.ErrInvalidRange
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, domainerrors.ErrInvalidRange
	}
	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, domainerrors.ErrInvalidRange
	}
	return &service.ByteRange{Start: start, End: end}, nil
}

func objectKey(r *http.Request) string {
	key := chi.URLParam(r, "*")
	return strings.TrimPrefix(key, "/")
}

func isMultipartPartUpload(r *http.Request) bool {
	return r.URL.Query().Get("uploadId") != "" && r.URL.Query().Get("partNumber") != ""
}

func ensureQuotedETag(etag string) string {
	trimmed := strings.Trim(etag, `"`)
	return fmt.Sprintf("\"%s\"", trimmed)
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := chimiddleware.GetReqID(r.Context()); requestID != "" {
		return requestID
	}
	if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
		return requestID
	}
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (h *Handler) requestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		startedAt := time.Now()
		next.ServeHTTP(ww, r)
		logger.Info(
			r.Context(),
			"http request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.Int("status", ww.Status()),
			zap.Int("bytes_written", ww.BytesWritten()),
			zap.Duration("duration", time.Since(startedAt)),
			zap.String("request_id", requestIDFromRequest(r)),
		)
	})
}

func (h *Handler) rateLimitMiddleware(next http.Handler) http.Handler {
	if h.throttleLimit < 1 || h.throttlePeriod <= 0 {
		return next
	}

	sem := make(chan struct{}, h.throttleLimit)
	var once sync.Once
	startRefill := func() {
		go func() {
			ticker := time.NewTicker(h.throttlePeriod)
			defer ticker.Stop()
			for range ticker.C {
				for {
					select {
					case <-sem:
					default:
						break
					}
				}
			}
		}()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(startRefill)
		select {
		case sem <- struct{}{}:
			next.ServeHTTP(w, r)
		default:
			h.writeError(w, r, fmt.Errorf("%w: rate limit exceeded", domainerrors.ErrServiceUnavailable))
		}
	})
}

func writeXML(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(payload)
}

func writeXMLError(w http.ResponseWriter, statusCode int, payload errorResponse) {
	writeXML(w, statusCode, payload)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapHTTPError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domainerrors.ErrUnauthorized):
		return http.StatusUnauthorized, "Unauthorized", "Authentication required"
	case errors.Is(err, domainerrors.ErrForbidden):
		return http.StatusForbidden, "AccessDenied", "Access denied"
	case errors.Is(err, domainerrors.ErrBucketAlreadyExist):
		return http.StatusConflict, "BucketAlreadyExists", "The requested bucket name is not available"
	case errors.Is(err, domainerrors.ErrBucketNotFound):
		return http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist"
	case errors.Is(err, domainerrors.ErrObjectNotFound):
		return http.StatusNotFound, "NoSuchKey", "The specified key does not exist"
	case errors.Is(err, domainerrors.ErrBucketNotEmpty):
		return http.StatusConflict, "BucketNotEmpty", "The bucket you tried to delete is not empty"
	case errors.Is(err, domainerrors.ErrInvalidRange):
		return http.StatusRequestedRangeNotSatisfiable, "InvalidRange", "The requested range is not satisfiable"
	case errors.Is(err, domainerrors.ErrInvalidRequest):
		return http.StatusBadRequest, "InvalidRequest", err.Error()
	case errors.Is(err, domainerrors.ErrInsufficientSpace):
		return http.StatusInsufficientStorage, "InsufficientStorage", "Insufficient storage"
	case errors.Is(err, domainerrors.ErrServiceUnavailable):
		return http.StatusServiceUnavailable, "ServiceUnavailable", "Service unavailable"
	default:
		return http.StatusInternalServerError, "InternalError", "We encountered an internal error"
	}
}

const defaultMaxKeys = 1000
