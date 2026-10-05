package filesystem_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/persistence/filesystem"
)

func TestNewObjectStorage(t *testing.T) {
	t.Run("creates storage root", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "objects")

		storage, err := filesystem.NewObjectStorage(root)
		if err != nil {
			t.Fatalf("NewObjectStorage() error = %v", err)
		}

		if storage == nil {
			t.Fatal("NewObjectStorage() returned nil storage")
		}

		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("storage root was not created: %v", err)
		}

		if !info.IsDir() {
			t.Fatal("storage root is not a directory")
		}
	})

	t.Run("rejects empty root", func(t *testing.T) {
		_, err := filesystem.NewObjectStorage("")
		if !errors.Is(err, filesystem.ErrInvalidRoot) {
			t.Fatalf("expected ErrInvalidRoot, got %v", err)
		}
	})
}

func TestObjectStorage_PutObject(t *testing.T) {
	t.Run("stores streamed content and content type", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		content := &chunkedReader{
			chunks: [][]byte{
				[]byte("hello "),
				[]byte("storage"),
			},
		}

		err = storage.PutObject(
			context.Background(),
			"stores/store-123.png",
			content,
			"image/png",
		)
		if err != nil {
			t.Fatalf("PutObject() error = %v", err)
		}

		reader, contentType, err := storage.GetPublicObject(
			context.Background(),
			"stores/store-123.png",
		)
		if err != nil {
			t.Fatalf("GetPublicObject() error = %v", err)
		}
		defer reader.Close()

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(body) != "hello storage" {
			t.Fatalf("content = %q, want %q", string(body), "hello storage")
		}

		if contentType != "image/png" {
			t.Fatalf("content type = %q, want %q", contentType, "image/png")
		}
	})

	t.Run("replaces existing object", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()

		if err := storage.PutObject(
			ctx,
			"objects/example.txt",
			io.NewSectionReader(
				fileReaderAt("first"),
				0,
				int64(len("first")),
			),
			"text/plain",
		); err != nil {
			t.Fatal(err)
		}

		if err := storage.PutObject(
			ctx,
			"objects/example.txt",
			io.NewSectionReader(
				fileReaderAt("second"),
				0,
				int64(len("second")),
			),
			"text/plain",
		); err != nil {
			t.Fatal(err)
		}

		reader, _, err := storage.GetPublicObject(ctx, "objects/example.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()

		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}

		if string(body) != "second" {
			t.Fatalf("content = %q, want %q", string(body), "second")
		}
	})

	t.Run("rejects nil content", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		err = storage.PutObject(
			context.Background(),
			"objects/example.txt",
			nil,
			"text/plain",
		)

		if err == nil {
			t.Fatal("expected error for nil content")
		}
	})

	t.Run("respects cancelled context", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = storage.PutObject(
			ctx,
			"objects/example.txt",
			strings.NewReader("content"),
			"text/plain",
		)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func TestObjectStorage_GetPublicObject(t *testing.T) {
	storage, err := filesystem.NewObjectStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	if err := storage.PutObject(
		ctx,
		"products/store-123/product-123.webp",
		strings.NewReader("image bytes"),
		"image/webp",
	); err != nil {
		t.Fatal(err)
	}

	reader, contentType, err := storage.GetPublicObject(
		ctx,
		"products/store-123/product-123.webp",
	)
	if err != nil {
		t.Fatalf("GetPublicObject() error = %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "image bytes" {
		t.Fatalf("content = %q, want %q", string(body), "image bytes")
	}

	if contentType != "image/webp" {
		t.Fatalf("content type = %q, want %q", contentType, "image/webp")
	}
}

func TestObjectStorage_GetPrivateObject(t *testing.T) {
	storage, err := filesystem.NewObjectStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	if err := storage.PutObject(
		ctx,
		"knowledge/store-123/manual.pdf",
		strings.NewReader("private knowledge"),
		"application/pdf",
	); err != nil {
		t.Fatal(err)
	}

	reader, contentType, err := storage.GetPrivateObject(
		ctx,
		"knowledge/store-123/manual.pdf",
	)
	if err != nil {
		t.Fatalf("GetPrivateObject() error = %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "private knowledge" {
		t.Fatalf("content = %q, want %q", string(body), "private knowledge")
	}

	if contentType != "application/pdf" {
		t.Fatalf("content type = %q, want %q", contentType, "application/pdf")
	}
}

func TestObjectStorage_GetObject_NotFound(t *testing.T) {
	storage, err := filesystem.NewObjectStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = storage.GetPublicObject(
		context.Background(),
		"missing/file.txt",
	)

	if !errors.Is(err, filesystem.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestObjectStorage_DeleteObject(t *testing.T) {
	t.Run("deletes object and metadata", func(t *testing.T) {
		root := t.TempDir()

		storage, err := filesystem.NewObjectStorage(root)
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()

		if err := storage.PutObject(
			ctx,
			"stores/store-123.png",
			strings.NewReader("image"),
			"image/png",
		); err != nil {
			t.Fatal(err)
		}

		if err := storage.DeleteObject(ctx, "stores/store-123.png"); err != nil {
			t.Fatalf("DeleteObject() error = %v", err)
		}

		if _, _, err := storage.GetPublicObject(
			ctx,
			"stores/store-123.png",
		); !errors.Is(err, filesystem.ErrObjectNotFound) {
			t.Fatalf("expected ErrObjectNotFound after delete, got %v", err)
		}

		if _, err := os.Stat(
			filepath.Join(root, "stores", "store-123.png.meta"),
		); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("metadata still exists: %v", err)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		err = storage.DeleteObject(
			context.Background(),
			"missing/file.txt",
		)
		if err != nil {
			t.Fatalf("DeleteObject() error = %v", err)
		}
	})
}

func TestObjectStorage_RejectsPathTraversal(t *testing.T) {
	storage, err := filesystem.NewObjectStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tests := []string{
		"",
		".",
		"..",
		"../outside.txt",
		"objects/../../outside.txt",
		"/absolute/path.txt",
	}

	for _, reference := range tests {
		t.Run(reference, func(t *testing.T) {
			err := storage.PutObject(
				context.Background(),
				reference,
				strings.NewReader("content"),
				"text/plain",
			)

			if !errors.Is(err, filesystem.ErrInvalidReference) {
				t.Fatalf(
					"expected ErrInvalidReference, got %v",
					err,
				)
			}
		})
	}
}

type chunkedReader struct {
	chunks [][]byte
	index  int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}

	chunk := r.chunks[r.index]
	r.index++

	return copy(p, chunk), nil
}

type fileReaderAt string

func (r fileReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	data := []byte(string(r))

	if offset >= int64(len(data)) {
		return 0, io.EOF
	}

	n := copy(p, data[offset:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func TestObjectStorage_HeadObject(t *testing.T) {
	t.Run("returns content type for existing object", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()

		if err := storage.PutObject(
			ctx,
			"models/gte/model.tar",
			strings.NewReader("model artifact"),
			"application/x-tar",
		); err != nil {
			t.Fatal(err)
		}

		contentType, err := storage.HeadObject(
			ctx,
			"models/gte/model.tar",
		)
		if err != nil {
			t.Fatalf("HeadObject() error = %v", err)
		}

		if contentType != "application/x-tar" {
			t.Fatalf("content type = %q, want %q", contentType, "application/x-tar")
		}
	})

	t.Run("returns not found for missing object", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		_, err = storage.HeadObject(
			context.Background(),
			"models/gte/missing.tar",
		)

		if !errors.Is(err, filesystem.ErrObjectNotFound) {
			t.Fatalf("expected ErrObjectNotFound, got %v", err)
		}
	})

	t.Run("rejects invalid reference", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		_, err = storage.HeadObject(
			context.Background(),
			"../outside/model.tar",
		)

		if !errors.Is(err, filesystem.ErrInvalidReference) {
			t.Fatalf("expected ErrInvalidReference, got %v", err)
		}
	})

	t.Run("respects cancelled context", func(t *testing.T) {
		storage, err := filesystem.NewObjectStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = storage.HeadObject(
			ctx,
			"models/gte/model.tar",
		)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}
