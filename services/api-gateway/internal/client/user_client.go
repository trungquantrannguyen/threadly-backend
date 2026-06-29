package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
	log    zerolog.Logger
}

func NewUserClient(cfg config.Config, log zerolog.Logger) (*UserClient, error) {
	conn, err := grpc.NewClient(
		cfg.UserServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := userpb.NewUserServiceClient(conn)

	return &UserClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *UserClient) GetHealth(ctx context.Context) (*userpb.GetUserServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &userpb.GetUserServiceHealthRequest{})
}

func (c *UserClient) Register(
	ctx context.Context,
	req *userpb.RegisterRequest,
) (*userpb.AuthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.Register(ctx, req)
}

func (c *UserClient) Login(
	ctx context.Context,
	req *userpb.LoginRequest,
) (*userpb.AuthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.Login(ctx, req)
}

func (c *UserClient) RefreshToken(
	ctx context.Context,
	req *userpb.RefreshTokenRequest,
) (*userpb.AuthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.RefreshToken(ctx, req)
}

func (c *UserClient) Logout(
	ctx context.Context,
	req *userpb.LogoutRequest,
) (*userpb.LogoutResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.Logout(ctx, req)
}

func (c *UserClient) GetMe(
	ctx context.Context,
	req *userpb.GetMeRequest,
) (*userpb.AuthUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetMe(ctx, req)
}

func (c *UserClient) UpdateProfile(
	ctx context.Context,
	req *userpb.UpdateProfileRequest,
) (*userpb.AuthUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.UpdateProfile(ctx, req)
}

func (c *UserClient) DeleteUser(
	ctx context.Context,
	req *userpb.DeleteUserRequest,
) (*userpb.DeleteUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.DeleteUser(ctx, req)
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}
