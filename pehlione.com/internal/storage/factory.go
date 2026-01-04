package storage

import (
	"context"
	"fmt"
	"os"
)

type FactoryResult struct {
	Driver  string
	Storage Storage
}

func FromEnv(ctx context.Context) (FactoryResult, error) {
	driver := os.Getenv("STORAGE_DRIVER")
	if driver == "" {
		driver = "local"
	}

	switch driver {
	case "local":
		baseDir := envOr("LOCAL_UPLOAD_DIR", "./storage/uploads")
		urlPrefix := envOr("LOCAL_UPLOAD_URL_PREFIX", "/uploads")
		return FactoryResult{Driver: "local", Storage: NewLocal(baseDir, urlPrefix)}, nil

	case "s3":
		region := envOr("S3_REGION", "")
		bucket := envOr("S3_BUCKET", "")
		publicBase := envOr("S3_PUBLIC_BASE_URL", "")
		prefix := envOr("S3_PREFIX", "uploads")
		if region == "" || bucket == "" || publicBase == "" {
			return FactoryResult{}, fmt.Errorf("S3 config missing: S3_REGION, S3_BUCKET, S3_PUBLIC_BASE_URL required")
		}
		s, err := NewS3(ctx, S3Config{
			Region:        region,
			Bucket:        bucket,
			Prefix:        prefix,
			PublicBaseURL: publicBase,
		})
		if err != nil {
			return FactoryResult{}, err
		}
		return FactoryResult{Driver: "s3", Storage: s}, nil

	default:
		return FactoryResult{}, fmt.Errorf("unknown STORAGE_DRIVER: %s", driver)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
