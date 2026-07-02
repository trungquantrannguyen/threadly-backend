package client

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

func testClientConfig() config.Config {
	return config.Config{
		UserServiceGRPCAddr:         "127.0.0.1:1",
		ContentServiceGRPCAddr:      "127.0.0.1:2",
		FeedServiceGRPCAddr:         "127.0.0.1:3",
		NotificationServiceGRPCAddr: "127.0.0.1:4",
		StorageServiceGRPCAddr:      "127.0.0.1:5",
	}
}

func TestNewUserClient(t *testing.T) {
	client, err := NewUserClient(testClientConfig(), zerolog.Nop())

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func TestNewContentClient(t *testing.T) {
	client, err := NewContentClient(testClientConfig(), zerolog.Nop())

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func TestNewFeedClient(t *testing.T) {
	client, err := NewFeedClient(testClientConfig(), zerolog.Nop())

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func TestNewNotificationClient(t *testing.T) {
	client, err := NewNotificationClient(testClientConfig(), zerolog.Nop())

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func TestNewStorageClient(t *testing.T) {
	client, err := NewStorageClient(testClientConfig(), zerolog.Nop())

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}
