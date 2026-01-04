package storage

import (
	"context"
	"io"
)

type PutInput struct {
	Filename    string
	ContentType string
	Size        int64
}

type PutResult struct {
	Key string
	URL string
}

type Storage interface {
	Put(ctx context.Context, r io.Reader, in PutInput) (PutResult, error)
	Delete(ctx context.Context, key string) error
}
