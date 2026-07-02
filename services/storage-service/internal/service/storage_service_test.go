package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
)

type fakeMediaRepo struct {
	created *dbmodel.Media
	err     error
}

func (f *fakeMediaRepo) Create(ctx context.Context, media *dbmodel.Media) error {
	if f.err != nil {
		return f.err
	}

	f.created = media
	return nil
}

type fakeStorageProvider struct {
	url string
	err error

	receivedStorageKey  string
	receivedContentType string
	receivedContent     []byte
}

func (f *fakeStorageProvider) Upload(ctx context.Context, storageKey string, contentType string, content []byte) (string, error) {
	f.receivedStorageKey = storageKey
	f.receivedContentType = contentType
	f.receivedContent = content

	if f.err != nil {
		return "", f.err
	}

	return f.url, nil
}

func TestUploadMediaSuccess(t *testing.T) {
	repo := &fakeMediaRepo{}
	provider := &fakeStorageProvider{
		url: "https://example.com/threadly-media/users/user-id/uploads/file.png",
	}

	svc := NewStorageService(repo, provider)

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "https://example.com/threadly-media/users/user-id/uploads/file.png", res.URL)
	assert.Equal(t, "image/png", res.MimeType)
	assert.Equal(t, int64(len([]byte("fake image bytes"))), res.SizeBytes)

	require.NotNil(t, repo.created)
	assert.Equal(t, "image/png", repo.created.MimeType)
	assert.Equal(t, int64(len([]byte("fake image bytes"))), repo.created.SizeBytes)
	assert.Equal(t, "uploaded", repo.created.Status)
	assert.Nil(t, repo.created.PostID)

	assert.True(t, strings.HasPrefix(
		repo.created.StorageKey,
		"users/11111111-1111-1111-1111-111111111111/uploads/",
	))

	assert.True(t, strings.HasSuffix(repo.created.StorageKey, ".png"))
	assert.Equal(t, repo.created.StorageKey, provider.receivedStorageKey)
	assert.Equal(t, "image/png", provider.receivedContentType)
	assert.Equal(t, []byte("fake image bytes"), provider.receivedContent)
}

func TestUploadMediaInvalidUploaderID(t *testing.T) {
	svc := NewStorageService(&fakeMediaRepo{}, &fakeStorageProvider{})

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "bad-id",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.ErrorIs(t, err, ErrInvalidUploaderID)
	assert.Nil(t, res)
}

func TestUploadMediaEmptyFile(t *testing.T) {
	svc := NewStorageService(&fakeMediaRepo{}, &fakeStorageProvider{})

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     nil,
	})

	require.ErrorIs(t, err, ErrEmptyFile)
	assert.Nil(t, res)
}

func TestUploadMediaInvalidMimeType(t *testing.T) {
	svc := NewStorageService(&fakeMediaRepo{}, &fakeStorageProvider{})

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "file.txt",
		ContentType: "text/plain",
		Content:     []byte("hello"),
	})

	require.ErrorIs(t, err, ErrInvalidMimeType)
	assert.Nil(t, res)
}

func TestUploadMediaProviderError(t *testing.T) {
	expectedErr := errors.New("supabase upload failed")

	svc := NewStorageService(
		&fakeMediaRepo{},
		&fakeStorageProvider{
			err: expectedErr,
		},
	)

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUploadMediaRepositoryError(t *testing.T) {
	expectedErr := errors.New("database create failed")

	repo := &fakeMediaRepo{
		err: expectedErr,
	}

	provider := &fakeStorageProvider{
		url: "https://example.com/threadly-media/users/user-id/uploads/file.png",
	}

	svc := NewStorageService(repo, provider)

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar.png",
		ContentType: "image/png",
		Content:     []byte("fake image bytes"),
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestUploadMediaUsesExtensionFromMimeWhenFilenameHasNoExtension(t *testing.T) {
	repo := &fakeMediaRepo{}
	provider := &fakeStorageProvider{
		url: "https://example.com/threadly-media/users/user-id/uploads/file.jpg",
	}

	svc := NewStorageService(repo, provider)

	res, err := svc.UploadMedia(context.Background(), UploadMediaRequest{
		UploaderID:  "11111111-1111-1111-1111-111111111111",
		Filename:    "avatar",
		ContentType: "image/jpeg",
		Content:     []byte("fake image bytes"),
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, repo.created)

	assert.True(t, strings.HasSuffix(repo.created.StorageKey, ".jpg"))
}

func TestExtensionFromMime(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{
			name:        "jpeg",
			contentType: "image/jpeg",
			expected:    ".jpg",
		},
		{
			name:        "png",
			contentType: "image/png",
			expected:    ".png",
		},
		{
			name:        "webp",
			contentType: "image/webp",
			expected:    ".webp",
		},
		{
			name:        "gif",
			contentType: "image/gif",
			expected:    ".gif",
		},
		{
			name:        "unknown",
			contentType: "application/octet-stream",
			expected:    ".bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := extensionFromMime(tt.contentType)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
