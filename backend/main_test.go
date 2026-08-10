package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		input   string
		path    string
		isFile  bool
		referen string
	}{
		{input: "https://github.com/owner/repo", referen: ""},
		{input: "https://github.com/owner/repo/tree/main/books", path: "books", referen: "main"},
		{input: "https://github.com/owner/repo/blob/main/books/hello%20world.pdf", path: "books/hello world.pdf", isFile: true, referen: "main"},
	}
	for _, test := range tests {
		got, err := parseGitHubURL(test.input)
		if err != nil {
			t.Fatalf("parseGitHubURL(%q): %v", test.input, err)
		}
		if got.Path != test.path || got.IsFile != test.isFile || got.Reference != test.referen {
			t.Fatalf("parseGitHubURL(%q) = %#v", test.input, got)
		}
	}
	if _, err := parseGitHubURL("https://example.com/owner/repo"); err == nil {
		t.Fatal("expected non-GitHub URL to fail")
	}
}

func TestFilter(t *testing.T) {
	filter, err := newFilter([]string{"node_modules/", "*.log"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"node_modules/a/b.js", "logs/app.log"} {
		if !filter.shouldIgnore(value) {
			t.Errorf("expected %q to be ignored", value)
		}
	}
	if filter.shouldIgnore("src/main.go") {
		t.Error("did not expect src/main.go to be ignored")
	}
}

func TestSafeRelativePath(t *testing.T) {
	if _, err := safeRelativePath("../outside"); err == nil {
		t.Error("expected parent traversal to fail")
	}
	if _, err := safeRelativePath("ebooks/2025"); err != nil {
		t.Fatal(err)
	}
}

func TestSmartFolderPrefix(t *testing.T) {
	tests := []struct {
		name      string
		files     []FileEntry
		selection []string
		want      string
	}{
		{
			name:      "single directory keeps its name",
			files:     []FileEntry{{Path: "a/b/c/file.txt"}},
			selection: []string{"a/b/c"},
			want:      "a/b",
		},
		{
			name:      "branches keep their meaningful names",
			files:     []FileEntry{{Path: "a/b/c/file.txt"}, {Path: "a/d/file.txt"}},
			selection: []string{"a/b/c", "a/d"},
			want:      "a",
		},
		{
			name:      "single file keeps its name",
			files:     []FileEntry{{Path: "a/b/file.txt"}},
			selection: []string{"a/b/file.txt"},
			want:      "a/b",
		},
		{
			name:      "root selections do not strip anything",
			files:     []FileEntry{{Path: "c/file.txt"}, {Path: "d/file.txt"}},
			selection: []string{"c", "d"},
			want:      "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := smartFolderPrefix(test.files, test.selection); got != test.want {
				t.Fatalf("smartFolderPrefix() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeSelection(t *testing.T) {
	selection := []string{"a/b"}
	if got := normalizeSelection(selection, true); got != nil {
		t.Fatalf("normalizeSelection(all=true) = %#v, want nil", got)
	}
	if got := normalizeSelection(selection, false); len(got) != 1 || got[0] != "a/b" {
		t.Fatalf("normalizeSelection(all=false) = %#v, want original selection", got)
	}
}

func TestExportFolderPathModes(t *testing.T) {
	config := Config{MaxFileSize: 1024 * 1024, MaxTotalSize: 1024 * 1024}
	client := newGitHubClient(config, "")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("test content")),
		}, nil
	})}
	files := []FileEntry{
		{Path: "a/b/c/one.txt", DownloadURL: "https://raw.githubusercontent.com/owner/repo/main/one.txt", Size: 12},
		{Path: "a/d/two.txt", DownloadURL: "https://raw.githubusercontent.com/owner/repo/main/two.txt", Size: 12},
	}

	tests := []struct {
		name      string
		pathMode  string
		selection []string
		want      []string
		missing   []string
	}{
		{
			name:      "smart mode strips shared parent",
			pathMode:  "smart",
			selection: []string{"a/b/c", "a/d"},
			want:      []string{"out/b/c/one.txt", "out/d/two.txt"},
			missing:   []string{"out/a/b/c/one.txt", "out/a/d/two.txt"},
		},
		{
			name:      "original mode preserves paths",
			pathMode:  "original",
			selection: []string{"a/b/c", "a/d"},
			want:      []string{"out/a/b/c/one.txt", "out/a/d/two.txt"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := exportFolder(context.Background(), root, "out", files, client, config, test.selection, test.pathMode); err != nil {
				t.Fatalf("exportFolder() error = %v", err)
			}
			for _, relative := range test.want {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
					t.Fatalf("expected exported file %q: %v", relative, err)
				}
			}
			for _, relative := range test.missing {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); !os.IsNotExist(err) {
					t.Fatalf("expected %q to be absent, err=%v", relative, err)
				}
			}
		})
	}
}

func TestCollectFilesReturnsNotFoundWhenFilterRemovesEverything(t *testing.T) {
	config := Config{MaxFileSize: 1024 * 1024, MaxTotalSize: 1024 * 1024}
	client := newGitHubClient(config, "")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(struct {
			Tree []githubTreeEntry `json:"tree"`
		}{Tree: []githubTreeEntry{{Path: "ignored.txt", Type: "blob", Size: 10}}}), nil
	})}
	filter, err := newFilter([]string{"ignored.txt"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.collectFiles(context.Background(), GitHubPath{Owner: "owner", Repo: "repo", Reference: "main"}, filter, nil)
	if asAPIError(err).Status != http.StatusNotFound {
		t.Fatalf("expected no matching files to return 404, got %v", err)
	}
}

func TestWritableDirectory(t *testing.T) {
	if !isWritableDirectory(t.TempDir()) {
		t.Fatal("expected temporary directory to be writable")
	}
}

func TestErrorStatus(t *testing.T) {
	if got := asAPIError(networkError(contextDeadlineError{})).Status; got != http.StatusGatewayTimeout {
		t.Fatalf("expected timeout status 504, got %d", got)
	}
}

func TestErrorDoesNotExposeCause(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, &APIError{Status: http.StatusInternalServerError, Message: "could not write output file", Cause: contextDeadlineError{}})
	if strings.Contains(recorder.Body.String(), "deadline exceeded") {
		t.Fatal("internal error cause was exposed in the response")
	}
}

func TestServiceAccessToken(t *testing.T) {
	application := &server{config: Config{AccessToken: "secret", MaxConcurrentJobs: 1}}
	handler := application.routes()
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/download", nil))
	if unauthorized.Code != http.StatusUnauthorized || unauthorized.Header().Get("X-ZZZ-Error-Code") != "auth_required" {
		t.Fatalf("unexpected unauthorized response: status=%d headers=%v", unauthorized.Code, unauthorized.Header())
	}
	authorized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/download", nil)
	request.Header.Set("X-ZZZ-Access-Token", "secret")
	handler.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusBadRequest {
		t.Fatalf("expected authorized request to reach validation, got status %d", authorized.Code)
	}
}

func TestTreeBrowsingDoesNotApplyDownloadSizeLimits(t *testing.T) {
	config := Config{
		GitHubTimeout:   time.Second,
		MaxFiles:        10,
		MaxFileSize:     1,
		MaxTotalSize:    1,
		MaxTreeRequests: 10,
	}
	client := newGitHubClient(config, "browser-token")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer browser-token" {
			t.Errorf("authorization header = %q", got)
		}
		return jsonHTTPResponse(struct {
			Tree []githubTreeEntry `json:"tree"`
		}{Tree: []githubTreeEntry{{Path: "big.bin", Type: "blob", Size: 10}}}), nil
	})}
	target := GitHubPath{Owner: "owner", Repo: "repo", Reference: "main"}
	filter, err := newFilter(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := client.listTree(context.Background(), target, filter)
	if err != nil {
		t.Fatalf("listTree returned an error: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "big.bin" {
		t.Fatalf("unexpected tree entries: %#v", entries)
	}
	if _, err := client.collectFiles(context.Background(), target, filter, nil); asAPIError(err).Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected download size limit, got %v", err)
	}
}

func TestTreeRequestLimit(t *testing.T) {
	config := Config{GitHubTimeout: time.Second, MaxFiles: 10, MaxTreeRequests: 1}
	client := newGitHubClient(config, "")
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/owner/repo") {
			return jsonHTTPResponse(map[string]string{"default_branch": "main"}), nil
		}
		return jsonHTTPResponse(struct {
			Tree []githubTreeEntry `json:"tree"`
		}{Tree: []githubTreeEntry{{Path: "books/file.txt", Type: "blob", Size: 1}}}), nil
	})}
	filter, _ := newFilter(nil, nil)
	_, err := client.listTree(context.Background(), GitHubPath{Owner: "owner", Repo: "repo"}, filter)
	if asAPIError(err).Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected tree request limit, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonHTTPResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}

type contextDeadlineError struct{}

const serverURLPlaceholder = "https://raw.githubusercontent.com/owner/repo/main/big.bin"

func (contextDeadlineError) Error() string   { return "deadline exceeded" }
func (contextDeadlineError) Timeout() bool   { return true }
func (contextDeadlineError) Temporary() bool { return true }
