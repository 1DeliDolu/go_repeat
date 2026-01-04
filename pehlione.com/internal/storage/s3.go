package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3 struct {
	Client        *s3.Client
	Bucket        string
	Prefix        string
	PublicBaseURL string
}

type S3Config struct {
	Region        string
	Bucket        string
	Prefix        string
	PublicBaseURL string
}

func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, err
	}
	return &S3{
		Client:        s3.NewFromConfig(awsCfg),
		Bucket:        cfg.Bucket,
		Prefix:        cfg.Prefix,
		PublicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (s *S3) Put(ctx context.Context, r io.Reader, in PutInput) (PutResult, error) {
	ext := strings.ToLower(filepath.Ext(in.Filename))
	key := uuid.NewString() + ext
	if s.Prefix != "" {
		key = strings.Trim(s.Prefix, "/") + "/" + key
	}

	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.Bucket,
		Key:         &key,
		Body:        r,
		ContentType: &in.ContentType,
	})
	if err != nil {
		return PutResult{}, err
	}

	url := s.PublicBaseURL + "/" + key
	return PutResult{Key: key, URL: url}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	return err
}

func (s *S3) String() string { return fmt.Sprintf("s3(%s/%s)", s.Bucket, s.Prefix) }
