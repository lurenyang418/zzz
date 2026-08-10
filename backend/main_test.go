package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
