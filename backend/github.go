package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
)

type GitHubPath struct {
	Owner     string
	Repo      string
	Reference string
	Path      string
	IsFile    bool
}

func parseGitHubURL(input string) (GitHubPath, error) {
	u, err := url.Parse(input)
	if err != nil {
		return GitHubPath{}, invalidError(err.Error())
	}
	if u.Scheme != "https" || (u.Hostname() != "github.com" && u.Hostname() != "www.github.com") {
		return GitHubPath{}, invalidError("only https://github.com URLs are supported")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return GitHubPath{}, invalidError("query strings and fragments are not supported")
	}
	parts := make([]string, 0)
	for _, rawPart := range strings.Split(strings.Trim(u.Path, "/"), "/") {
		if rawPart == "" {
			continue
		}
		part, err := url.PathUnescape(rawPart)
		if err != nil {
			return GitHubPath{}, invalidError("invalid GitHub URL path")
		}
		parts = append(parts, part)
	}
	if len(parts) < 2 {
		return GitHubPath{}, invalidError("expected /owner/repository")
	}
	owner, err := validateGitHubSegment(parts[0], "owner")
	if err != nil {
		return GitHubPath{}, err
	}
	repo, err := validateGitHubSegment(strings.TrimSuffix(parts[1], ".git"), "repository")
	if err != nil {
		return GitHubPath{}, err
	}
	result := GitHubPath{Owner: owner, Repo: repo}
	if len(parts) == 2 {
		return result, nil
	}
	if parts[2] != "tree" && parts[2] != "blob" {
		return GitHubPath{}, invalidError("expected /blob/ or /tree/")
	}
	if len(parts) < 4 {
		return GitHubPath{}, invalidError("missing branch, tag, or commit")
	}
	if strings.ContainsAny(parts[3], "?#\\") || parts[3] == "" {
		return GitHubPath{}, invalidError("invalid branch, tag, or commit")
	}
	result.Reference = parts[3]
	result.IsFile = parts[2] == "blob"
	for _, part := range parts[4:] {
		if part == "." || part == ".." {
			return GitHubPath{}, invalidError("path traversal segments are not supported")
		}
	}
	result.Path = strings.Join(parts[4:], "/")
	return result, nil
}

func validateGitHubSegment(value, label string) (string, error) {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\") {
		return "", invalidError("invalid " + label)
	}
	return value, nil
}

type TreeEntry struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Size *int64 `json:"size,omitempty"`
}

type FileEntry struct {
	Path        string
	DownloadURL string
	Size        int64
}

type githubContent struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

type githubTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type GitHubClient struct {
	http          *http.Client
	config        Config
	token         string
	baseURL       string
	requestMu     sync.Mutex
	treeRequests  int
	contentsMu    sync.Mutex
	contentsCache map[string][]githubContent
}

func newGitHubClient(config Config, requestToken string) *GitHubClient {
	token := strings.TrimSpace(requestToken)
	if token == "" {
		token = config.GitHubToken
	}
	return &GitHubClient{
		http:          &http.Client{Timeout: config.GitHubTimeout},
		config:        config,
		token:         token,
		baseURL:       "https://api.github.com",
		contentsCache: make(map[string][]githubContent),
	}
}

func (c *GitHubClient) resolveReference(ctx context.Context, target GitHubPath) (string, error) {
	if target.Reference != "" {
		return target.Reference, nil
	}
	var repository struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.getJSON(ctx, "/repos/"+url.PathEscape(target.Owner)+"/"+url.PathEscape(target.Repo), nil, &repository); err != nil {
		return "", err
	}
	if repository.DefaultBranch == "" {
		return "", &APIError{Status: http.StatusBadGateway, Message: "GitHub returned an empty default branch"}
	}
	return repository.DefaultBranch, nil
}

func (c *GitHubClient) listContents(ctx context.Context, target GitHubPath, reference, contentPath string) ([]githubContent, error) {
	cacheKey := reference + "\x00" + contentPath
	c.contentsMu.Lock()
	if cached, ok := c.contentsCache[cacheKey]; ok {
		result := append([]githubContent(nil), cached...)
		c.contentsMu.Unlock()
		return result, nil
	}
	c.contentsMu.Unlock()

	base := "/repos/" + url.PathEscape(target.Owner) + "/" + url.PathEscape(target.Repo) + "/contents"
	if contentPath != "" {
		for _, segment := range strings.Split(contentPath, "/") {
			base += "/" + url.PathEscape(segment)
		}
	}
	result := make([]githubContent, 0)
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("ref", reference)
		query.Set("per_page", "100")
		query.Set("page", fmt.Sprintf("%d", page))
		var payload json.RawMessage
		if err := c.getJSON(ctx, base, query, &payload); err != nil {
			return nil, err
		}
		if len(payload) == 0 || payload[0] != '[' {
			var item githubContent
			if err := json.Unmarshal(payload, &item); err != nil {
				return nil, &APIError{Status: http.StatusBadGateway, Message: "invalid GitHub contents response", Cause: err}
			}
			result := []githubContent{item}
			c.cacheContents(cacheKey, result)
			return result, nil
		}
		var items []githubContent
		if err := json.Unmarshal(payload, &items); err != nil {
			return nil, &APIError{Status: http.StatusBadGateway, Message: "invalid GitHub contents response", Cause: err}
		}
		result = append(result, items...)
		if len(items) < 100 {
			c.cacheContents(cacheKey, result)
			return result, nil
		}
	}
}

func (c *GitHubClient) cacheContents(key string, contents []githubContent) {
	c.contentsMu.Lock()
	c.contentsCache[key] = append([]githubContent(nil), contents...)
	c.contentsMu.Unlock()
}

func (c *GitHubClient) getJSON(ctx context.Context, apiPath string, query url.Values, target any) error {
	if err := c.reserveTreeRequest(); err != nil {
		return err
	}
	u := strings.TrimRight(c.baseURL, "/") + apiPath
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return &APIError{Status: http.StatusBadGateway, Message: "could not create GitHub request", Cause: err}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "zzz/0.2")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return networkError(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 32*1024*1024))
	if err != nil {
		return networkError(err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var payload struct {
			Message string `json:"message"`
		}
		message := strings.TrimSpace(string(body))
		if json.Unmarshal(body, &payload) == nil && payload.Message != "" {
			message = payload.Message
		}
		return githubError(response.StatusCode, message)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return &APIError{Status: http.StatusBadGateway, Message: "invalid GitHub response", Cause: err}
	}
	return nil
}

func (c *GitHubClient) reserveTreeRequest() error {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	if c.config.MaxTreeRequests > 0 && c.treeRequests >= c.config.MaxTreeRequests {
		return &APIError{
			Status:  http.StatusRequestEntityTooLarge,
			Message: "directory requires too many GitHub metadata requests; choose a narrower path",
		}
	}
	c.treeRequests++
	return nil
}

func (c *GitHubClient) walk(ctx context.Context, target GitHubPath, filter *Filter, selection []string, collect bool) ([]TreeEntry, []FileEntry, error) {
	if !target.IsFile {
		return c.walkGitTree(ctx, target, filter, selection, collect)
	}
	reference, err := c.resolveReference(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	items, err := c.listContents(ctx, target, reference, target.Path)
	if err != nil {
		return nil, nil, err
	}
	if len(items) != 1 || items[0].Type != "file" {
		return nil, nil, &APIError{Status: http.StatusNotFound, Message: "the GitHub URL does not point to a file"}
	}
	item := items[0]
	relative := path.Base(target.Path)
	if filter.shouldIgnore(relative) || (collect && !selectionMatches(selection, relative)) {
		return []TreeEntry{}, []FileEntry{}, nil
	}
	size := item.Size
	entries := []TreeEntry{{Path: relative, Name: path.Base(relative), Kind: "file", Size: &size}}
	if !collect {
		return entries, nil, nil
	}
	if item.Size > c.config.MaxFileSize {
		return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: relative + " is too large"}
	}
	if item.Size > c.config.MaxTotalSize {
		return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "total input is too large"}
	}
	return entries, []FileEntry{{Path: relative, DownloadURL: item.DownloadURL, Size: item.Size}}, nil
}

func (c *GitHubClient) walkGitTree(ctx context.Context, target GitHubPath, filter *Filter, selection []string, collect bool) ([]TreeEntry, []FileEntry, error) {
	reference, err := c.resolveReference(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		Truncated bool              `json:"truncated"`
		Tree      []githubTreeEntry `json:"tree"`
	}
	query := url.Values{}
	query.Set("recursive", "1")
	endpoint := "/repos/" + url.PathEscape(target.Owner) + "/" + url.PathEscape(target.Repo) + "/git/trees/" + url.PathEscape(reference)
	if err := c.getJSON(ctx, endpoint, query, &payload); err != nil {
		return nil, nil, err
	}
	if payload.Truncated {
		return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "repository tree is too large; choose a narrower directory"}
	}

	entries := make([]TreeEntry, 0)
	files := make([]FileEntry, 0)
	seenDirectories := make(map[string]struct{})
	var declaredTotal int64
	addDirectory := func(directory string) error {
		if collect || directory == "" {
			return nil
		}
		if _, exists := seenDirectories[directory]; exists {
			return nil
		}
		if len(entries) >= c.config.MaxFiles*2 {
			return &APIError{Status: http.StatusRequestEntityTooLarge, Message: "directory tree is too large"}
		}
		seenDirectories[directory] = struct{}{}
		entries = append(entries, TreeEntry{Path: directory, Name: path.Base(directory), Kind: "dir"})
		return nil
	}
	for _, item := range payload.Tree {
		if item.Type != "blob" {
			continue
		}
		if target.Path != "" && item.Path != target.Path && !strings.HasPrefix(item.Path, strings.TrimSuffix(target.Path, "/")+"/") {
			continue
		}
		relative := item.Path
		if target.Path != "" {
			relative = strings.TrimPrefix(item.Path, strings.TrimSuffix(target.Path, "/")+"/")
		}
		if relative == "" || filter.shouldIgnore(relative) {
			continue
		}
		parts := strings.Split(relative, "/")
		pruned := false
		for index := 1; index < len(parts); index++ {
			directory := strings.Join(parts[:index], "/")
			if filter.shouldPruneDirectory(directory) {
				pruned = true
				break
			}
			if err := addDirectory(directory); err != nil {
				return nil, nil, err
			}
		}
		if pruned || (collect && !selectionMatches(selection, relative)) {
			continue
		}
		if collect {
			if item.Size > c.config.MaxFileSize {
				return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: relative + " is too large"}
			}
			if len(files) >= c.config.MaxFiles {
				return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "too many files"}
			}
			declaredTotal += item.Size
			if declaredTotal > c.config.MaxTotalSize {
				return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "total input is too large"}
			}
			files = append(files, FileEntry{
				Path:        relative,
				DownloadURL: rawDownloadURL(target, reference, item.Path),
				Size:        item.Size,
			})
			continue
		}
		size := item.Size
		if len(entries) >= c.config.MaxFiles*2 {
			return nil, nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: "directory tree is too large"}
		}
		entries = append(entries, TreeEntry{Path: relative, Name: path.Base(relative), Kind: "file", Size: &size})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return entries, files, nil
}

func rawDownloadURL(target GitHubPath, reference, filePath string) string {
	parts := []string{target.Owner, target.Repo, reference}
	parts = append(parts, strings.Split(filePath, "/")...)
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return "https://raw.githubusercontent.com/" + strings.Join(parts, "/")
}

func (c *GitHubClient) listTree(ctx context.Context, target GitHubPath, filter *Filter) ([]TreeEntry, error) {
	entries, _, err := c.walk(ctx, target, filter, nil, false)
	return entries, err
}

func (c *GitHubClient) collectFiles(ctx context.Context, target GitHubPath, filter *Filter, selection []string) ([]FileEntry, error) {
	_, files, err := c.walk(ctx, target, filter, selection, true)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, &APIError{Status: http.StatusNotFound, Message: "no files matched the filter"}
	}
	return files, nil
}

func (c *GitHubClient) downloadFile(ctx context.Context, file FileEntry) ([]byte, error) {
	downloadURL := file.DownloadURL
	if downloadURL == "" {
		return nil, &APIError{Status: http.StatusBadGateway, Message: "GitHub did not provide a download URL"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, &APIError{Status: http.StatusBadGateway, Message: "could not create file download request", Cause: err}
	}
	req.Header.Set("User-Agent", "zzz/0.2")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return nil, networkError(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
		return nil, githubError(response.StatusCode, strings.TrimSpace(string(body)))
	}
	if response.ContentLength > c.config.MaxFileSize {
		return nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: file.Path + " is too large"}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, c.config.MaxFileSize+1))
	if err != nil {
		return nil, networkError(err)
	}
	if int64(len(data)) > c.config.MaxFileSize || (file.Size > 0 && int64(len(data)) > file.Size+1024*1024) {
		return nil, &APIError{Status: http.StatusRequestEntityTooLarge, Message: file.Path + " is too large"}
	}
	return data, nil
}

func selectionMatches(selection []string, filePath string) bool {
	if len(selection) == 0 {
		return true
	}
	for _, selected := range selection {
		if selected == filePath || strings.HasPrefix(filePath, strings.TrimSuffix(selected, "/")+"/") {
			return true
		}
	}
	return false
}
