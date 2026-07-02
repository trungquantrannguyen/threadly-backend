package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	storagesvc "github.com/trungquantrannguyen/threadly/services/storage-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeStorageService struct {
	res *storagesvc.UploadMediaResponse
	err error

	called bool
	req    storagesvc.UploadMediaRequest
}

func (f *fakeStorageService) UploadMedia(ctx context.Context, req storagesvc.UploadMediaRequest) (*storagesvc.UploadMediaResponse, error) {
	f.called = true
	f.req = req

	if f.err != nil {
		return nil, f.err
	}

	return f.res, nil
}

func TestGetHealth(t *testing.T) {
	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeStorageService{},
	)

	res, err := server.GetHealth(context.Background(), &storagepb.GetStorageServiceHealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "ok", res.GetStatus())
	assert.Equal(t, "storage-service", res.GetService())
	assert.Equal(t, "test", res.GetEnv())
	assert.NotEmpty(t, res.GetCheckedAt())
}

func TestUploadMediaSuccess(t *testing.T) {
	fakeSvc := &fakeStorageService{
		res: &storagesvc.UploadMediaResponse{
			ID:         "media-123",
			URL:        "https://example.com/media.png",
			StorageKey: "users/user-123/uploads/media.png",
			MimeType:   "image/png",
			SizeBytes:  123,
			CreatedAt:  "2026-07-02T08:00:00Z",
		},
	}

	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		fakeSvc,
	)

	res, err := server.UploadMedia(context.Background(), &storagepb.UploadMediaRequest{
		UploaderId:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.called)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", fakeSvc.req.UploaderID)
	assert.Equal(t, "avatar.png", fakeSvc.req.Filename)
	assert.Equal(t, "image/png", fakeSvc.req.ContentType)
	assert.Equal(t, []byte("fake image bytes"), fakeSvc.req.Content)

	assert.Equal(t, "media-123", res.GetId())
	assert.Equal(t, "https://example.com/media.png", res.GetUrl())
	assert.Equal(t, "users/user-123/uploads/media.png", res.GetStorageKey())
	assert.Equal(t, "image/png", res.GetMimeType())
	assert.Equal(t, int64(123), res.GetSizeBytes())
	assert.Equal(t, "2026-07-02T08:00:00Z", res.GetCreatedAt())
}

func TestUploadMediaInvalidUploaderIDReturnsInvalidArgument(t *testing.T) {
	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeStorageService{
			err: storagesvc.ErrInvalidUploaderID,
		},
	)

	res, err := server.UploadMedia(context.Background(), &storagepb.UploadMediaRequest{
		UploaderId:  "bad-id",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid uploader id")
}

func TestUploadMediaEmptyFileReturnsInvalidArgument(t *testing.T) {
	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeStorageService{
			err: storagesvc.ErrEmptyFile,
		},
	)

	res, err := server.UploadMedia(context.Background(), &storagepb.UploadMediaRequest{
		UploaderId:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     nil,
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "file is empty")
}

func TestUploadMediaInvalidMimeTypeReturnsInvalidArgument(t *testing.T) {
	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeStorageService{
			err: storagesvc.ErrInvalidMimeType,
		},
	)

	res, err := server.UploadMedia(context.Background(), &storagepb.UploadMediaRequest{
		UploaderId:  "11111111-1111-1111-1111-111111111111",
		Filename:    "file.txt",
		ContentType: "text/plain",
		Content:     []byte("hello"),
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid file type")
}

func TestUploadMediaUnexpectedErrorReturnsInternal(t *testing.T) {
	server := NewStorageServiceServer(
		config.Config{
			ServiceName: "storage-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeStorageService{
			err: errors.New("supabase failed"),
		},
	)

	res, err := server.UploadMedia(context.Background(), &storagepb.UploadMediaRequest{
		UploaderId:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "failed to upload media", st.Message())
}
