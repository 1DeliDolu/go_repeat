package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Local struct {
	BaseDir   string
	URLPrefix string
}

func NewLocal(baseDir, urlPrefix string) *Local {
	return &Local{BaseDir: baseDir, URLPrefix: urlPrefix}
}

func (l *Local) Put(ctx context.Context, r io.Reader, in PutInput) (PutResult, error) {
	_ = ctx

	if err := os.MkdirAll(l.BaseDir, 0o755); err != nil {
		return PutResult{}, err
	}

	ext := safeExt(in.Filename)
	key := uuid.NewString() + ext
	dstPath := filepath.Join(l.BaseDir, key)

	f, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return PutResult{}, err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return PutResult{}, err
	}

	url := strings.TrimRight(l.URLPrefix, "/") + "/" + key
	return PutResult{Key: key, URL: url}, nil
}

func (l *Local) Delete(ctx context.Context, key string) error {
	_ = ctx
	key = filepath.Base(key)
	return os.Remove(filepath.Join(l.BaseDir, key))
}

func safeExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return ext
	default:
		return ""
	}
}

func (l *Local) String() string { return fmt.Sprintf("local(%s)", l.BaseDir) }
