package main

import "os"

type Config struct {
	StorageBackend  string
	DiskStoragePath string

	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3Bucket          string
	S3Region          string
	S3UsePathStyle    bool

	HTTPAddr string
	BaseURL  string
	AutoOpen bool
}

func LoadConfig() Config {
	return Config{
		StorageBackend:  getEnv("STORAGE_BACKEND", "disk"),
		DiskStoragePath: getEnv("DISK_STORAGE_PATH", "/tmp/htmlrenderermcp"),

		S3Endpoint:        os.Getenv("S3_ENDPOINT"),
		S3AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		S3Bucket:          getEnv("S3_BUCKET", "html-renders"),
		S3Region:          getEnv("S3_REGION", "us-east-1"),
		S3UsePathStyle:    getEnv("S3_USE_PATH_STYLE", "true") == "true",

		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		BaseURL:  getEnv("BASE_URL", "http://localhost:8080"),
		AutoOpen: getEnv("AUTO_OPEN", "true") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
