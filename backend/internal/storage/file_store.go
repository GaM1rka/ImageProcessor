package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"imageprocessor/backend/internal/models"
)

var ErrNotFound = errors.New("image not found")

type FileStore struct {
	root string
	mu   sync.RWMutex
}

func NewFileStore(root string) (*FileStore, error) {
	for _, dir := range []string{"originals", "processed", "thumbnails", "meta"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return nil, err
		}
	}
	return &FileStore{root: root}, nil
}

func (s *FileStore) SaveOriginal(ctx context.Context, image models.Image, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeFile(filepath.Join(s.root, "originals", image.ID), reader); err != nil {
		return err
	}
	return s.writeMeta(image)
}

func (s *FileStore) OpenOriginal(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openExisting(filepath.Join(s.root, "originals", id))
}

func (s *FileStore) SaveProcessed(ctx context.Context, id string, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return writeFile(filepath.Join(s.root, "processed", id+".jpg"), reader)
}

func (s *FileStore) OpenProcessed(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openExisting(filepath.Join(s.root, "processed", id+".jpg"))
}

func (s *FileStore) SaveThumbnail(ctx context.Context, id string, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return writeFile(filepath.Join(s.root, "thumbnails", id+".jpg"), reader)
}

func (s *FileStore) OpenThumbnail(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openExisting(filepath.Join(s.root, "thumbnails", id+".jpg"))
}

func (s *FileStore) Get(ctx context.Context, id string) (models.Image, error) {
	if err := ctx.Err(); err != nil {
		return models.Image{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := os.Open(s.metaPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return models.Image{}, ErrNotFound
	}
	if err != nil {
		return models.Image{}, err
	}
	defer file.Close()

	var image models.Image
	if err := json.NewDecoder(file).Decode(&image); err != nil {
		return models.Image{}, err
	}
	return image, nil
}

func (s *FileStore) List(ctx context.Context) ([]models.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.root, "meta"))
	if err != nil {
		return nil, err
	}

	images := make([]models.Image, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		image, err := s.readMeta(entry.Name()[:len(entry.Name())-len(".json")])
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

func (s *FileStore) UpdateStatus(ctx context.Context, id string, status models.Status, message string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image, err := s.readMeta(id)
	if err != nil {
		return err
	}
	image.Status = status
	image.Error = message
	image.UpdatedAt = time.Now().UTC()
	return s.writeMeta(image)
}

func (s *FileStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, path := range []string{
		filepath.Join(s.root, "originals", id),
		filepath.Join(s.root, "processed", id+".jpg"),
		filepath.Join(s.root, "thumbnails", id+".jpg"),
		s.metaPath(id),
	} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *FileStore) metaPath(id string) string {
	return filepath.Join(s.root, "meta", id+".json")
}

func (s *FileStore) readMeta(id string) (models.Image, error) {
	file, err := os.Open(s.metaPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return models.Image{}, ErrNotFound
	}
	if err != nil {
		return models.Image{}, err
	}
	defer file.Close()

	var image models.Image
	if err := json.NewDecoder(file).Decode(&image); err != nil {
		return models.Image{}, err
	}
	return image, nil
}

func (s *FileStore) writeMeta(image models.Image) error {
	path := s.metaPath(image.ID)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(image); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func writeFile(path string, reader io.Reader) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func openExisting(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return file, err
}
