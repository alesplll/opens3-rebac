package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const uploadChunkSize = 64 * 1024

type Handler struct {
	metadata metadatav1.MetadataServiceClient
	storage  storagev1.DataStorageServiceClient
}

func NewHandler(metadata metadatav1.MetadataServiceClient, storage storagev1.DataStorageServiceClient) http.Handler {
	h := &Handler{metadata: metadata, storage: storage}
	return http.HandlerFunc(h.serveHTTP)
}

func (h *Handler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
		return
	}

	bucket, key, ok := objectPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.putObject(w, r, bucket, key)
	case http.MethodGet:
		h.getObject(w, r, bucket, key)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func objectPath(path string) (bucket, key string, ok bool) {
	bucket, key, ok = strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return bucket, key, ok && bucket != "" && key != ""
}

func (h *Handler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	ctx := r.Context()
	exists, err := h.metadata.HeadBucket(ctx, &metadatav1.HeadBucketRequest{BucketName: bucket})
	if err != nil {
		writeRPCError(w, err)
		return
	}
	if !exists.GetExists() {
		http.Error(w, "bucket not found", http.StatusNotFound)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := h.storage.StoreObject(streamCtx)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	header := &storagev1.StoreObjectHeader{ContentType: contentType}
	if r.ContentLength >= 0 {
		size := r.ContentLength
		header.Size = &size
	}
	if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Header{Header: header}}); err != nil {
		writeRPCError(w, err)
		return
	}

	buf := make([]byte, uploadChunkSize)
	var size int64
	for {
		n, readErr := r.Body.Read(buf)
		if n > 0 {
			if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Chunk{Chunk: &storagev1.StoreObjectChunk{Data: buf[:n]}}}); err != nil {
				writeRPCError(w, err)
				return
			}
			size += int64(n)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			cancel()
			http.Error(w, "could not read request body", http.StatusBadRequest)
			return
		}
	}
	stored, err := stream.CloseAndRecv()
	if err != nil {
		writeRPCError(w, err)
		return
	}
	version, err := h.metadata.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
		BucketName:  bucket,
		Key:         key,
		BlobId:      stored.GetBlobId(),
		SizeBytes:   size,
		Etag:        stored.GetChecksumMd5(),
		ContentType: contentType,
	})
	if err != nil {
		slog.Error("metadata registration failed after blob commit", "blob_id", stored.GetBlobId(), "bucket", bucket, "key", key, "error", err)
		writeRPCError(w, err)
		return
	}
	w.Header().Set("ETag", strconv.Quote(stored.GetChecksumMd5()))
	w.Header().Set("X-Object-Version-Id", version.GetVersionId())
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	meta, err := h.metadata.GetObjectMeta(r.Context(), &metadatav1.GetObjectMetaRequest{BucketName: bucket, Key: key})
	if err != nil {
		writeRPCError(w, err)
		return
	}
	stream, err := h.storage.RetrieveObject(r.Context(), &storagev1.RetrieveObjectRequest{BlobId: meta.GetBlobId()})
	if err != nil {
		writeRPCError(w, err)
		return
	}
	first, err := stream.Recv()
	if err != nil && !errors.Is(err, io.EOF) {
		writeRPCError(w, err)
		return
	}
	if errors.Is(err, io.EOF) && meta.GetSizeBytes() != 0 {
		http.Error(w, "storage returned an empty stream", http.StatusBadGateway)
		return
	}
	if meta.GetContentType() != "" {
		w.Header().Set("Content-Type", meta.GetContentType())
	}
	w.Header().Set("Content-Length", strconv.FormatInt(meta.GetSizeBytes(), 10))
	w.Header().Set("ETag", strconv.Quote(meta.GetEtag()))
	w.Header().Set("X-Object-Version-Id", meta.GetVersionId())
	w.WriteHeader(http.StatusOK)
	if first != nil {
		if _, err := w.Write(first.GetData()); err != nil {
			return
		}
	}
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			slog.Error("storage read failed after response started", "blob_id", meta.GetBlobId(), "error", err)
			return
		}
		if _, err := w.Write(chunk.GetData()); err != nil {
			return
		}
	}
}

func writeRPCError(w http.ResponseWriter, err error) {
	code := http.StatusBadGateway
	switch status.Code(err) {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.ResourceExhausted:
		code = http.StatusInsufficientStorage
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		code = http.StatusGatewayTimeout
	}
	http.Error(w, http.StatusText(code), code)
}
