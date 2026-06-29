package provider

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

type SupabaseStorageProvider interface {
	Upload(ctx context.Context, storageKey string, contentType string, content []byte) (string, error)
}

type supabaseStorageProvider struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewSupabaseStorageProvider(ctx context.Context, cfg config.Config) (SupabaseStorageProvider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.SupabaseS3Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.SupabaseS3AccessKeyID,
				cfg.SupabaseS3SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.SupabaseS3Endpoint)
		o.UsePathStyle = true
	})

	return &supabaseStorageProvider{
		client:    client,
		bucket:    cfg.SupabaseStorageBucket,
		publicURL: strings.TrimRight(cfg.SupabasePublicStorageURL, "/"),
	}, nil
}

func (p *supabaseStorageProvider) Upload(ctx context.Context, storageKey string, contentType string, content []byte) (string, error) {
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(storageKey),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	publicURL := fmt.Sprintf("%s/%s/%s", p.publicURL, p.bucket, storageKey)

	return publicURL, nil
}
