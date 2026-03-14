package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type DiskStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewDiskStorage(basePath string) (*DiskStorage, error) {
	rendersDir := filepath.Join(basePath, "renders")
	if err := os.MkdirAll(rendersDir, 0755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}
	return &DiskStorage{basePath: basePath}, nil
}

func (d *DiskStorage) htmlPath(id string) string {
	return filepath.Join(d.basePath, "renders", id+".html")
}

func (d *DiskStorage) metaPath(id string) string {
	return filepath.Join(d.basePath, "renders", id+".json")
}

func (d *DiskStorage) Upload(_ context.Context, meta RenderMeta, html []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := os.WriteFile(d.htmlPath(meta.ID), html, 0644); err != nil {
		return fmt.Errorf("writing html file: %w", err)
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(d.metaPath(meta.ID), metaBytes, 0644); err != nil {
		return fmt.Errorf("writing meta file: %w", err)
	}

	return nil
}

func (d *DiskStorage) Fetch(_ context.Context, id string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	data, err := os.ReadFile(d.htmlPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading html file: %w", err)
	}
	return data, nil
}

func (d *DiskStorage) List(_ context.Context) ([]RenderMeta, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pattern := filepath.Join(d.basePath, "renders", "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("listing renders: %w", err)
	}

	var renders []RenderMeta
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var meta RenderMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		renders = append(renders, meta)
	}

	sort.Slice(renders, func(i, j int) bool {
		return renders[i].CreatedAt.After(renders[j].CreatedAt)
	})

	return renders, nil
}
