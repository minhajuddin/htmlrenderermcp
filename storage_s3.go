package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(cfg Config) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKeyID, cfg.S3SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &S3Storage{client: client, bucket: cfg.S3Bucket}, nil
}

func (s *S3Storage) htmlKey(id string) string {
	return "renders/" + id + ".html"
}

func (s *S3Storage) metaKey(id string) string {
	return "renders/" + id + ".json"
}

func (s *S3Storage) Upload(ctx context.Context, meta RenderMeta, html []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         aws.String(s.htmlKey(meta.ID)),
		Body:        bytes.NewReader(html),
		ContentType: aws.String("text/html; charset=utf-8"),
	})
	if err != nil {
		return fmt.Errorf("uploading html to s3: %w", err)
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         aws.String(s.metaKey(meta.ID)),
		Body:        bytes.NewReader(metaBytes),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("uploading metadata to s3: %w", err)
	}

	return nil
}

func (s *S3Storage) Fetch(ctx context.Context, id string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    aws.String(s.htmlKey(id)),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching from s3: %w", err)
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (s *S3Storage) List(ctx context.Context) ([]RenderMeta, error) {
	prefix := "renders/"

	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("listing s3 objects: %w", err)
	}

	var renders []RenderMeta
	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)
		if len(key) < 6 || key[len(key)-5:] != ".json" {
			continue
		}

		getResult, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &s.bucket,
			Key:    obj.Key,
		})
		if err != nil {
			continue
		}
		data, err := io.ReadAll(getResult.Body)
		getResult.Body.Close()
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
