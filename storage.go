package main

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("render not found")

type RenderMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Storage interface {
	Upload(ctx context.Context, meta RenderMeta, html []byte) error
	Fetch(ctx context.Context, id string) ([]byte, error)
	List(ctx context.Context) ([]RenderMeta, error)
}
