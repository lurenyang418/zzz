package main

import (
	"archive/zip"
	"context"
	"net/http"
	"os"
)

type zipArtifact struct {
	path      string
	size      int64
	totalSize int64
}

func createZIP(ctx context.Context, files []FileEntry, client *GitHubClient, config Config, directory string) (zipArtifact, error) {
	temporary, err := os.CreateTemp(directory, ".zzz-*.zip")
	if err != nil {
		return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not create temporary ZIP", Cause: err}
	}
	name := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(name)
	}
	writer := zip.NewWriter(temporary)
	var totalSize int64
	for _, file := range files {
		content, err := client.downloadFile(ctx, file)
		if err != nil {
			_ = writer.Close()
			cleanup()
			return zipArtifact{}, err
		}
		totalSize += int64(len(content))
		if totalSize > config.MaxTotalSize {
			_ = writer.Close()
			cleanup()
			return zipArtifact{}, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "total downloaded content is too large"}
		}
		header := &zip.FileHeader{Name: file.Path, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			cleanup()
			return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not create ZIP entry", Cause: err}
		}
		if _, err := entry.Write(content); err != nil {
			_ = writer.Close()
			cleanup()
			return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not write ZIP entry", Cause: err}
		}
		if info, statErr := temporary.Stat(); statErr == nil && info.Size() > config.MaxZipSize {
			_ = writer.Close()
			cleanup()
			return zipArtifact{}, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "generated ZIP is too large"}
		}
	}
	if err := writer.Close(); err != nil {
		cleanup()
		return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not finish ZIP", Cause: err}
	}
	info, err := temporary.Stat()
	if err != nil {
		cleanup()
		return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not inspect generated ZIP", Cause: err}
	}
	if info.Size() > config.MaxZipSize {
		cleanup()
		return zipArtifact{}, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "generated ZIP is too large"}
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(name)
		return zipArtifact{}, &APIError{Status: http.StatusInternalServerError, Message: "could not close generated ZIP", Cause: err}
	}
	return zipArtifact{path: name, size: info.Size(), totalSize: totalSize}, nil
}
