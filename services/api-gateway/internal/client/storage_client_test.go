package client

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeStorageServiceServer struct {
	storagepb.UnimplementedStorageServiceServer

	healthCalled      bool
	uploadMediaCalled bool

	receivedUploaderID  string
	receivedFilename    string
	receivedContentType string
	receivedContent     []byte
}

func startFakeStorageGRPCServer(t *testing.T, fakeServer *fakeStorageServiceServer) (*StorageClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	storagepb.RegisterStorageServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(10<<20),
			grpc.MaxCallRecvMsgSize(10<<20),
		),
	)
	require.NoError(t, err)

	client := &StorageClient{
		conn:   conn,
		client: storagepb.NewStorageServiceClient(conn),
		log:    zerolog.Nop(),
	}

	cleanup := func() {
		_ = client.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}

func (s *fakeStorageServiceServer) GetHealth(ctx context.Context, req *storagepb.GetStorageServiceHealthRequest) (*storagepb.GetStorageServiceHealthResponse, error) {
	s.healthCalled = true

	return &storagepb.GetStorageServiceHealthResponse{
		Status:    "ok",
		Service:   "storage-service",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeStorageServiceServer) UploadMedia(ctx context.Context, req *storagepb.UploadMediaRequest) (*storagepb.UploadMediaResponse, error) {
	s.uploadMediaCalled = true
	s.receivedUploaderID = req.GetUploaderId()
	s.receivedFilename = req.GetFilename()
	s.receivedContentType = req.GetContentType()
	s.receivedContent = req.GetContent()

	return &storagepb.UploadMediaResponse{
		Id:         "media-1",
		Url:        "https://example.com/media.png",
		StorageKey: "uploads/user-1/media-1.png",
		MimeType:   req.GetContentType(),
		SizeBytes:  int64(len(req.GetContent())),
		CreatedAt:  "2026-07-02T10:00:00Z",
	}, nil
}

func TestStorageClientGetHealth(t *testing.T) {
	fakeServer := &fakeStorageServiceServer{}
	client, cleanup := startFakeStorageGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.healthCalled)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "storage-service", res.Service)
	assert.Equal(t, "test", res.Env)
}

func TestStorageClientUploadMedia(t *testing.T) {
	fakeServer := &fakeStorageServiceServer{}
	client, cleanup := startFakeStorageGRPCServer(t, fakeServer)
	defer cleanup()

	content := []byte("fake image bytes")

	res, err := client.UploadMedia(
		context.Background(),
		"user-1",
		"avatar.png",
		"image/png",
		content,
	)

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.uploadMediaCalled)

	assert.Equal(t, "user-1", fakeServer.receivedUploaderID)
	assert.Equal(t, "avatar.png", fakeServer.receivedFilename)
	assert.Equal(t, "image/png", fakeServer.receivedContentType)
	assert.True(t, bytes.Equal(content, fakeServer.receivedContent))

	assert.Equal(t, "media-1", res.Id)
	assert.Equal(t, "https://example.com/media.png", res.Url)
	assert.Equal(t, "uploads/user-1/media-1.png", res.StorageKey)
	assert.Equal(t, "image/png", res.MimeType)
	assert.Equal(t, int64(len(content)), res.SizeBytes)
	assert.Equal(t, "2026-07-02T10:00:00Z", res.CreatedAt)
}
