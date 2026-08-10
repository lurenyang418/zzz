package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// The Docker build copies frontend/dist into this directory before compiling.
// Explicitly embedding .gitkeep keeps local Go checks valid before a frontend build.
//
//go:embed static/.gitkeep static/*
var embeddedStatic embed.FS

type server struct {
	config  Config
	limiter *requestLimiter
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	config := loadConfig()
	application := &server{config: config, limiter: newRequestLimiter(config)}
	httpServer := &http.Server{
		Addr:              config.ListenAddress,
		Handler:           application.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 * 1024,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownContext)
	}()
	log.Printf("event=server_start address=%s host_export=%t access_token_required=%t", config.ListenAddress, config.DownloadRoot != "", config.AccessToken != "")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/capabilities", s.handleCapabilities)
	mux.Handle("/api/tree", s.limitMiddleware(http.HandlerFunc(s.handleTree)))
	mux.Handle("/api/download", s.limitMiddleware(http.HandlerFunc(s.handleDownload)))
	mux.Handle("/api/export", s.limitMiddleware(http.HandlerFunc(s.handleExport)))
	return loggingMiddleware(s.accessMiddleware(mux), http.HandlerFunc(s.handleStatic))
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "OK")
}

func (s *server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	hostExport := s.config.DownloadRoot != "" && isWritableDirectory(s.config.DownloadRoot)
	writeJSON(w, http.StatusOK, map[string]bool{
		"host_export":   hostExport,
		"auth_required": s.config.AccessToken != "",
	})
}

func (s *server) handleTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	target, filter, client, err := s.prepareGitHubRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	entries, err := client.listTree(r.Context(), target, filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	target, filter, client, err := s.prepareGitHubRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	selection := parseSelection(r.URL.Query().Get("select"))
	if r.URL.Query().Get("all") == "1" {
		selection = nil
	}
	files, err := client.collectFiles(r.Context(), target, filter, selection)
	if err != nil {
		writeError(w, err)
		return
	}
	zipData, err := createZIP(r.Context(), files, client, s.config, "")
	if err != nil {
		writeError(w, err)
		return
	}
	filename := safeFilename(target.Repo + "-" + target.Reference + ".zip")
	if target.Reference == "" {
		filename = safeFilename(target.Repo + "-default.zip")
	}
	file, err := os.Open(zipData.path)
	if err != nil {
		_ = os.Remove(zipData.path)
		writeError(w, &APIError{Status: http.StatusInternalServerError, Message: "could not open generated ZIP", Cause: err})
		return
	}
	defer file.Close()
	defer os.Remove(zipData.path)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", zipData.size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (s *server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request exportRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2*1024*1024))
	if err := decoder.Decode(&request); err != nil {
		writeError(w, invalidError("invalid JSON request: "+err.Error()))
		return
	}
	if strings.TrimSpace(request.URL) == "" {
		writeError(w, invalidError("missing 'url' parameter"))
		return
	}
	target, err := parseGitHubURL(request.URL)
	if err != nil {
		writeError(w, err)
		return
	}
	filter, err := filterFromSlices(request.Ignore, request.Include)
	if err != nil {
		writeError(w, err)
		return
	}
	client := newGitHubClient(s.config, requestToken(r))
	selection := normalizeSelection(request.Select, request.All)
	request.Select = selection
	files, err := client.collectFiles(r.Context(), target, filter, selection)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := exportToHost(r.Context(), files, client, s.config, request, target.Repo)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) prepareGitHubRequest(r *http.Request) (GitHubPath, *Filter, *GitHubClient, error) {
	input := strings.TrimSpace(r.URL.Query().Get("url"))
	if input == "" {
		return GitHubPath{}, nil, nil, invalidError("missing 'url' parameter")
	}
	target, err := parseGitHubURL(input)
	if err != nil {
		return GitHubPath{}, nil, nil, err
	}
	filter, err := filterFromQuery(r.URL.Query().Get("ignore"), r.URL.Query().Get("include"))
	if err != nil {
		return GitHubPath{}, nil, nil, err
	}
	return target, filter, newGitHubClient(s.config, requestToken(r)), nil
}

func filterFromQuery(ignore, include string) (*Filter, error) {
	return filterFromSlices(splitPatterns(ignore), splitPatterns(include))
}

func filterFromSlices(ignore, include []string) (*Filter, error) {
	if len(ignore) == 0 && len(include) == 0 {
		return defaultFilter()
	}
	return newFilter(ignore, include)
}

func requestToken(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-GitHub-Token"))
}

func parseSelection(value string) []string {
	result := make([]string, 0)
	for _, path := range strings.Split(value, ",") {
		path = strings.TrimSpace(path)
		if path != "" && !strings.Contains(path, "..") && !strings.HasPrefix(path, "/") {
			result = append(result, path)
		}
	}
	return result
}

func safeFilename(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
	}
	return builder.String()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *server) handleStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	file, err := embeddedStatic.Open("static/" + name)
	if err != nil {
		file, err = embeddedStatic.Open("static/index.html")
		name = "index.html"
	}
	if err != nil {
		http.Error(w, "frontend assets are not embedded", http.StatusNotFound)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "could not read frontend asset", http.StatusInternalServerError)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

var requestSequence uint64

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	count, err := w.ResponseWriter.Write(data)
	w.bytes += count
	return count, err
}

func (w *loggingResponseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func loggingMiddleware(mux http.Handler, fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := fmt.Sprintf("%d", atomic.AddUint64(&requestSequence, 1))
		w.Header().Set("X-Request-ID", requestID)
		response := &loggingResponseWriter{ResponseWriter: w}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			mux.ServeHTTP(response, r)
		} else {
			fallback.ServeHTTP(response, r)
		}
		if response.status == 0 {
			response.status = http.StatusOK
		}
		log.Printf("event=http_request request_id=%s remote=%s method=%s path=%q status=%d bytes=%d duration=%s", requestID, clientIP(r), r.Method, r.URL.Path, response.status, response.bytes, time.Since(start).Round(time.Millisecond))
	})
}

func stringContainsFold(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func init() {
	if os.Getenv("TZ") == "" {
		_ = os.Setenv("TZ", "UTC")
	}
}
