package main

import (
	"context"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type exportRequest struct {
	URL         string   `json:"url"`
	Ignore      []string `json:"ignore"`
	Include     []string `json:"include"`
	Select      []string `json:"select"`
	All         bool     `json:"all"`
	Destination string   `json:"destination"`
	Format      string   `json:"format"`
	PathMode    string   `json:"path_mode"`
}

type exportResult struct {
	Format    string `json:"format"`
	Path      string `json:"path"`
	FileCount int    `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

func exportToHost(ctx context.Context, files []FileEntry, client *GitHubClient, config Config, request exportRequest, repoName string) (exportResult, error) {
	if config.DownloadRoot == "" {
		return exportResult{}, hostExportUnavailableError()
	}
	if err := os.MkdirAll(config.DownloadRoot, 0o755); err != nil {
		return exportResult{}, &APIError{Status: http.StatusServiceUnavailable, Message: "宿主机导出目录不可用", Hint: "请检查 DOWNLOAD_ROOT 对应的 Docker 挂载目录权限。", Code: "host_export_unavailable", Cause: err}
	}
	if !isWritableDirectory(config.DownloadRoot) {
		return exportResult{}, hostExportUnavailableError()
	}
	root, err := filepath.Abs(config.DownloadRoot)
	if err != nil {
		return exportResult{}, &APIError{Status: 500, Message: "could not resolve DOWNLOAD_ROOT", Cause: err}
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return exportResult{}, &APIError{Status: 500, Message: "could not resolve DOWNLOAD_ROOT", Cause: err}
	}
	destination := strings.TrimSpace(request.Destination)
	if destination == "" {
		destination = repoName
	}
	relative, err := safeRelativePath(destination)
	if err != nil {
		return exportResult{}, err
	}
	format := request.Format
	if format == "" {
		format = "zip"
	}
	switch format {
	case "folder":
		pathMode := request.PathMode
		if pathMode == "" {
			pathMode = "original"
		}
		if pathMode != "original" && pathMode != "smart" {
			return exportResult{}, invalidError("unsupported folder path mode: " + pathMode)
		}
		return exportFolder(ctx, root, relative, files, client, config, normalizeSelection(request.Select, request.All), pathMode)
	case "zip":
		return exportZIP(ctx, root, relative, files, client, config)
	default:
		return exportResult{}, invalidError("unsupported export format: " + format)
	}
}

func normalizeSelection(selection []string, all bool) []string {
	if all {
		return nil
	}
	return selection
}

func hostExportUnavailableError() *APIError {
	return &APIError{
		Status:  http.StatusServiceUnavailable,
		Message: "宿主机导出目录不可写",
		Hint:    "请检查 DOWNLOAD_ROOT 对应的 Docker 挂载目录权限。",
		Code:    "host_export_unavailable",
	}
}

func isWritableDirectory(directory string) bool {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return false
	}
	testFile, err := os.CreateTemp(directory, ".zzz-write-test-*")
	if err != nil {
		return false
	}
	testPath := testFile.Name()
	if err := testFile.Close(); err != nil {
		_ = os.Remove(testPath)
		return false
	}
	return os.Remove(testPath) == nil
}

func exportFolder(ctx context.Context, root, destination string, files []FileEntry, client *GitHubClient, config Config, selection []string, pathMode string) (exportResult, error) {
	output, err := safeJoin(root, destination)
	if err != nil {
		return exportResult{}, err
	}
	if err := ensureNoSymlink(root, output); err != nil {
		return exportResult{}, err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return exportResult{}, &APIError{Status: 500, Message: "could not create output folder", Cause: err}
	}
	stripPrefix := ""
	if pathMode == "smart" {
		stripPrefix = smartFolderPrefix(files, selection)
	}
	var totalSize int64
	for _, file := range files {
		content, err := client.downloadFile(ctx, file)
		if err != nil {
			return exportResult{}, err
		}
		totalSize += int64(len(content))
		if totalSize > config.MaxTotalSize {
			return exportResult{}, &APIError{Status: 413, Message: "total downloaded content is too large"}
		}
		relativePath := file.Path
		if stripPrefix != "" {
			relativePath = strings.TrimPrefix(relativePath, stripPrefix+"/")
		}
		outputPath, err := safeJoin(output, relativePath)
		if err != nil {
			return exportResult{}, err
		}
		if err := ensureNoSymlink(root, outputPath); err != nil {
			return exportResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return exportResult{}, &APIError{Status: 500, Message: "could not create output folder", Cause: err}
		}
		if err := os.WriteFile(outputPath, content, 0o644); err != nil {
			return exportResult{}, &APIError{Status: 500, Message: "could not write output file", Cause: err}
		}
	}
	return exportResult{Format: "folder", Path: filepath.ToSlash(destination), FileCount: len(files), TotalSize: totalSize}, nil
}

func smartFolderPrefix(files []FileEntry, selection []string) string {
	if len(files) == 0 || len(selection) == 0 {
		return ""
	}

	parents := make([]string, 0, len(selection))
	for _, root := range reduceSelection(selection) {
		parent := path.Dir(root)
		if parent == "." {
			parent = ""
		}
		parents = append(parents, parent)
	}
	return commonPathPrefix(parents)
}

func reduceSelection(selection []string) []string {
	unique := make(map[string]struct{}, len(selection))
	for _, selected := range selection {
		selected = strings.Trim(selected, "/")
		if selected != "" {
			unique[selected] = struct{}{}
		}
	}
	paths := make([]string, 0, len(unique))
	for selected := range unique {
		paths = append(paths, selected)
	}
	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i]) == len(paths[j]) {
			return paths[i] < paths[j]
		}
		return len(paths[i]) < len(paths[j])
	})
	reduced := make([]string, 0, len(paths))
	for _, candidate := range paths {
		covered := false
		for _, parent := range reduced {
			if candidate == parent || strings.HasPrefix(candidate, parent+"/") {
				covered = true
				break
			}
		}
		if !covered {
			reduced = append(reduced, candidate)
		}
	}
	return reduced
}

func commonPathPrefix(paths []string) string {
	if len(paths) == 0 || paths[0] == "" {
		return ""
	}
	parts := strings.Split(paths[0], "/")
	for _, value := range paths[1:] {
		if value == "" {
			return ""
		}
		other := strings.Split(value, "/")
		limit := len(parts)
		if len(other) < limit {
			limit = len(other)
		}
		index := 0
		for index < limit && parts[index] == other[index] {
			index++
		}
		parts = parts[:index]
		if len(parts) == 0 {
			return ""
		}
	}
	return strings.Join(parts, "/")
}

func exportZIP(ctx context.Context, root, destination string, files []FileEntry, client *GitHubClient, config Config) (exportResult, error) {
	if filepath.Ext(destination) == "" {
		destination += ".zip"
	}
	output, err := safeJoin(root, destination)
	if err != nil {
		return exportResult{}, err
	}
	if err := ensureNoSymlink(root, output); err != nil {
		return exportResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return exportResult{}, &APIError{Status: 500, Message: "could not create output folder", Cause: err}
	}
	zipData, err := createZIP(ctx, files, client, config, filepath.Dir(output))
	if err != nil {
		return exportResult{}, err
	}
	if err := os.Chmod(zipData.path, 0o644); err != nil {
		_ = os.Remove(zipData.path)
		return exportResult{}, &APIError{Status: 500, Message: "could not set ZIP file permissions", Cause: err}
	}
	if err := os.Rename(zipData.path, output); err != nil {
		_ = os.Remove(zipData.path)
		return exportResult{}, &APIError{Status: 500, Message: "could not write ZIP file", Cause: err}
	}
	return exportResult{Format: "zip", Path: filepath.ToSlash(destination), FileCount: len(files), TotalSize: zipData.size}, nil
}

func safeRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") {
		return "", invalidError("output path must be relative and cannot contain '..'")
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", invalidError("output path must be relative and cannot contain '..'")
		}
	}
	clean := pathCleanSlash(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", invalidError("output path must be relative and cannot contain '..'")
	}
	return filepath.FromSlash(clean), nil
}

func pathCleanSlash(value string) string {
	parts := make([]string, 0)
	for _, part := range strings.Split(value, "/") {
		switch part {
		case "", ".":
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			} else {
				return ".."
			}
		default:
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "/")
}

func safeJoin(root, relative string) (string, error) {
	joined := filepath.Join(root, relative)
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", invalidError("output path escapes DOWNLOAD_ROOT")
	}
	return joined, nil
}

func ensureNoSymlink(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return invalidError("output path escapes DOWNLOAD_ROOT")
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return invalidError("output path cannot be a symbolic link")
		}
		if err != nil && !os.IsNotExist(err) {
			return &APIError{Status: 500, Message: "could not inspect output path", Cause: err}
		}
	}
	return nil
}
