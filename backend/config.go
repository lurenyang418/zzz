package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AccessToken        string
	GitHubToken        string
	GitHubTimeout      time.Duration
	WriteTimeout       time.Duration
	MaxFiles           int
	MaxFileSize        int64
	MaxTotalSize       int64
	MaxZipSize         int64
	MaxTreeRequests    int
	MaxConcurrentJobs  int
	RateLimitPerMinute int
	DownloadRoot       string
	ListenAddress      string
}

func loadConfig() Config {
	root := strings.TrimSpace(os.Getenv("DOWNLOAD_ROOT"))
	if root != "" {
		root = filepath.Clean(root)
	}
	return Config{
		AccessToken:        strings.TrimSpace(os.Getenv("ACCESS_TOKEN")),
		GitHubToken:        strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
		GitHubTimeout:      time.Duration(envInt64("GITHUB_TIMEOUT_SECS", 180)) * time.Second,
		WriteTimeout:       time.Duration(envInt64("WRITE_TIMEOUT_SECS", 1800)) * time.Second,
		MaxFiles:           int(envInt64("MAX_FILES", 10_000)),
		MaxFileSize:        envInt64("MAX_FILE_SIZE_BYTES", 100*1024*1024),
		MaxTotalSize:       envInt64("MAX_TOTAL_SIZE_BYTES", 512*1024*1024),
		MaxZipSize:         envInt64("MAX_ZIP_SIZE_BYTES", 512*1024*1024),
		MaxTreeRequests:    int(envInt64("MAX_TREE_REQUESTS", 500)),
		MaxConcurrentJobs:  int(envInt64("MAX_CONCURRENT_JOBS", 2)),
		RateLimitPerMinute: int(envInt64("RATE_LIMIT_PER_MINUTE", 30)),
		DownloadRoot:       root,
		ListenAddress:      envString("LISTEN_ADDR", "0.0.0.0:8080"),
	}
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt64(name string, fallback int64) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(name)), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
