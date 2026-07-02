package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeStorageHandlerServiceServer struct {
	storagepb.UnimplementedStorageServiceServer

	err error

	healthCalled      bool
	uploadMediaCalled bool

	receivedUploaderID  string
	receivedFilename    string
	receivedContentType string
	receivedContent     []byte
}

func startStorageHandlerTestClient(t *testing.T, fakeServer *fakeStorageHandlerServiceServer) (*client.StorageClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	storagepb.RegisterStorageServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	storageClient, err := client.NewStorageClient(config.Config{
		StorageServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = storageClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return storageClient, cleanup
}

func (s *fakeStorageHandlerServiceServer) GetHealth(ctx context.Context, req *storagepb.GetStorageServiceHealthRequest) (*storagepb.GetStorageServiceHealthResponse, error) {
	s.healthCalled = true

	if s.err != nil {
		return nil, s.err
	}

	return &storagepb.GetStorageServiceHealthResponse{
		Service:   "storage-service",
		Status:    "ok",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeStorageHandlerServiceServer) UploadMedia(ctx context.Context, req *storagepb.UploadMediaRequest) (*storagepb.UploadMediaResponse, error) {
	s.uploadMediaCalled = true
	s.receivedUploaderID = req.GetUploaderId()
	s.receivedFilename = req.GetFilename()
	s.receivedContentType = req.GetContentType()
	s.receivedContent = req.GetContent()

	if s.err != nil {
		return nil, s.err
	}

	return &storagepb.UploadMediaResponse{
		Id:         "media-1",
		Url:        "https://example.com/media.png",
		StorageKey: "uploads/user-1/media-1.png",
		MimeType:   req.GetContentType(),
		SizeBytes:  int64(len(req.GetContent())),
		CreatedAt:  "2026-07-02T10:00:00Z",
	}, nil
}

func newStorageHandlerTestRouter(handler *StorageHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if withUser {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "user-1")
			c.Next()
		})
	}

	router.GET("/storages/health", handler.GetHealth)
	router.POST("/storages/upload", handler.UploadMedia)

	return router
}

func newMultipartUploadRequest(t *testing.T, target string, fieldName string, filename string, contentType string, content []byte) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, target, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestStorageHandlerGetHealthSuccess(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storages/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Storage service available")
	assert.Contains(t, w.Body.String(), "storage-service")
}

func TestStorageHandlerGetHealthServiceUnavailable(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{
		err: errors.New("storage service down"),
	}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storages/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Storage service unavailable")
}

func TestStorageHandlerUploadMediaSuccess(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	content := []byte("fake image bytes")

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "file", "avatar.png", "image/png", content)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	assert.True(t, fakeServer.uploadMediaCalled)
	assert.Equal(t, "user-1", fakeServer.receivedUploaderID)
	assert.Equal(t, "avatar.png", fakeServer.receivedFilename)
	assert.Equal(t, "image/png", fakeServer.receivedContentType)
	assert.Equal(t, content, fakeServer.receivedContent)

	assert.Contains(t, w.Body.String(), "Media uploaded successfully")
	assert.Contains(t, w.Body.String(), "media-1")
	assert.Contains(t, w.Body.String(), "https://example.com/media.png")
}

func TestStorageHandlerUploadMediaUnauthorized(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "file", "avatar.png", "image/png", []byte("content"))

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestStorageHandlerUploadMediaMissingFile(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/storages/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "file is required")
}

func TestStorageHandlerUploadMediaWrongFormField(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "wrong_file_field", "avatar.png", "image/png", []byte("content"))

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "file is required")
}

func TestStorageHandlerUploadMediaTooLarge(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	tooLargeContent := bytes.Repeat([]byte("a"), (5<<20)+1)

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "file", "big.png", "image/png", tooLargeContent)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "file size must be less than 5MB")
}

func TestStorageHandlerUploadMediaGRPCInvalidArgument(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{
		err: status.Error(codes.InvalidArgument, "invalid media file"),
	}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "file", "avatar.txt", "text/plain", []byte("bad content"))

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "invalid media file")
}

func TestStorageHandlerUploadMediaGRPCInternalError(t *testing.T) {
	fakeServer := &fakeStorageHandlerServiceServer{
		err: status.Error(codes.Internal, "storage failed"),
	}
	storageClient, cleanup := startStorageHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewStorageHandler(storageClient, zerolog.Nop())
	router := newStorageHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/storages/upload", "file", "avatar.png", "image/png", []byte("content"))

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, fakeServer.uploadMediaCalled)
	assert.Contains(t, w.Body.String(), "Internal server error")
}
