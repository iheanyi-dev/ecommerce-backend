package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

var (
	ErrInvalidRoot      = errors.New("storage root is invalid")
	ErrInvalidReference = errors.New("storage reference is invalid")
	ErrObjectNotFound   = errors.New("storage object not found")
)

// ObjectStorage implements ports.ObjectStorage using the local filesystem.
//
// The filesystem backend deliberately knows nothing about business concepts
// such as stores, products, users, or knowledge files. The complete object
// reference is supplied by the calling service.
//
// Content is stored at the supplied reference and its MIME type is stored in
// a small sidecar metadata file. The sidecar is an implementation detail and
// is never exposed through the application layer.
type ObjectStorage struct {
	root string
}

// NewObjectStorage creates a filesystem-backed object storage.
//
// The root directory is created when it does not already exist.
func NewObjectStorage(root string) (*ObjectStorage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrInvalidRoot
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}

	if err := os.MkdirAll(absoluteRoot, 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}

	return &ObjectStorage{
		root: absoluteRoot,
	}, nil
}

// PutObject writes or replaces an object.
//
// The content is streamed directly from the supplied reader rather than
// buffering the complete object in memory.
func (s *ObjectStorage) PutObject(
	ctx context.Context,
	reference string,
	content io.Reader,
	contentType string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if content == nil {
		return errors.New("object content is nil")
	}

	objectPath, err := s.objectPath(reference)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(objectPath), ".object-*")
	if err != nil {
		return fmt.Errorf("create temporary object: %w", err)
	}

	tempPath := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := copyWithContext(ctx, tempFile, content); err != nil {
		cleanup()
		return fmt.Errorf("write object: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temporary object: %w", err)
	}

	if err := os.Rename(tempPath, objectPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace object: %w", err)
	}

	if err := s.writeContentType(ctx, objectPath, contentType); err != nil {
		return err
	}

	return nil
}

// GetPublicObject retrieves an object intended for public retrieval.
//
// Authorization that determines whether an object is actually public belongs
// outside this backend. This method only performs generic object retrieval.
func (s *ObjectStorage) GetPublicObject(
	ctx context.Context,
	reference string,
) (io.ReadCloser, string, error) {
	return s.getObject(ctx, reference)
}

// GetPrivateObject retrieves an object intended for trusted internal access.
//
// Authorization belongs to the trusted internal caller. The filesystem
// backend only performs generic object retrieval.
func (s *ObjectStorage) GetPrivateObject(
	ctx context.Context,
	reference string,
) (io.ReadCloser, string, error) {
	return s.getObject(ctx, reference)
}

// DeleteObject deletes an object.
//
// Deleting an object that does not exist is intentionally idempotent.
func (s *ObjectStorage) DeleteObject(
	ctx context.Context,
	reference string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	objectPath, err := s.objectPath(reference)
	if err != nil {
		return err
	}

	err = os.Remove(objectPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}

	metadataPath := metadataPath(objectPath)

	if err := os.Remove(metadataPath); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object metadata: %w", err)
	}

	return nil
}

func (s *ObjectStorage) getObject(
	ctx context.Context,
	reference string,
) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}

	objectPath, err := s.objectPath(reference)
	if err != nil {
		return nil, "", err
	}

	file, err := os.Open(objectPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrObjectNotFound
		}

		return nil, "", fmt.Errorf("open object: %w", err)
	}

	contentType, err := s.readContentType(objectPath)
	if err != nil {
		_ = file.Close()
		return nil, "", err
	}

	return file, contentType, nil
}

func (s *ObjectStorage) writeContentType(
	ctx context.Context,
	objectPath string,
	contentType string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	metadataPath := metadataPath(objectPath)

	tempFile, err := os.CreateTemp(filepath.Dir(metadataPath), ".metadata-*")
	if err != nil {
		return fmt.Errorf("create temporary metadata: %w", err)
	}

	tempPath := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := io.WriteString(tempFile, contentType); err != nil {
		cleanup()
		return fmt.Errorf("write object metadata: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temporary metadata: %w", err)
	}

	if err := os.Rename(tempPath, metadataPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace object metadata: %w", err)
	}

	return nil
}

func (s *ObjectStorage) readContentType(objectPath string) (string, error) {
	content, err := os.ReadFile(metadataPath(objectPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("read object metadata: %w", ErrObjectNotFound)
		}

		return "", fmt.Errorf("read object metadata: %w", err)
	}

	return string(content), nil
}

func (s *ObjectStorage) objectPath(reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", ErrInvalidReference
	}

	if filepath.IsAbs(reference) {
		return "", ErrInvalidReference
	}

	cleanReference := filepath.Clean(reference)
	if cleanReference == "." || cleanReference == ".." {
		return "", ErrInvalidReference
	}

	root := filepath.Clean(s.root)
	objectPath := filepath.Join(root, cleanReference)

	relative, err := filepath.Rel(root, objectPath)
	if err != nil {
		return "", ErrInvalidReference
	}

	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", ErrInvalidReference
	}

	return objectPath, nil
}

func metadataPath(objectPath string) string {
	return objectPath + ".meta"
}

func copyWithContext(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		n, readErr := src.Read(buffer)

		if n > 0 {
			writtenNow, writeErr := dst.Write(buffer[:n])
			written += int64(writtenNow)

			if writeErr != nil {
				return written, writeErr
			}

			if writtenNow != n {
				return written, io.ErrShortWrite
			}
		}

		if errors.Is(readErr, io.EOF) {
			return written, nil
		}

		if readErr != nil {
			return written, readErr
		}
	}
}

var _ ports.ObjectStorage = (*ObjectStorage)(nil)
