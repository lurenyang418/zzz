package main

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type requestLimiter struct {
	mu        sync.Mutex
	semaphore chan struct{}
	perMinute int
	clients   map[string]rateWindow
}

type rateWindow struct {
	started time.Time
	count   int
}

func newRequestLimiter(config Config) *requestLimiter {
	concurrent := config.MaxConcurrentJobs
	if concurrent < 1 {
		concurrent = 1
	}
	return &requestLimiter{
		semaphore: make(chan struct{}, concurrent),
		perMinute: config.RateLimitPerMinute,
		clients:   make(map[string]rateWindow),
	}
}

func (l *requestLimiter) allow(ip string, now time.Time) bool {
	if l.perMinute <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	window, ok := l.clients[ip]
	if !ok || now.Sub(window.started) >= time.Minute {
		l.clients[ip] = rateWindow{started: now, count: 1}
		l.prune(now)
		return true
	}
	if window.count >= l.perMinute {
		return false
	}
	window.count++
	l.clients[ip] = window
	return true
}

func (l *requestLimiter) prune(now time.Time) {
	if len(l.clients) <= 4096 {
		return
	}
	for ip, window := range l.clients {
		if now.Sub(window.started) >= time.Minute {
			delete(l.clients, ip)
		}
	}
}

func (l *requestLimiter) acquire() bool {
	select {
	case l.semaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *requestLimiter) release() {
	<-l.semaphore
}

func (s *server) accessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.AccessToken == "" || r.URL.Path == "/api/health" || r.URL.Path == "/api/capabilities" {
			next.ServeHTTP(w, r)
			return
		}
		provided := strings.TrimSpace(r.Header.Get("X-ZZZ-Access-Token"))
		if provided == "" {
			authorization := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
				provided = strings.TrimSpace(authorization[len("Bearer "):])
			}
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.config.AccessToken)) != 1 {
			writeError(w, &APIError{Status: http.StatusUnauthorized, Message: "service access token is required", Code: "auth_required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) limitMiddleware(next http.Handler) http.Handler {
	if s.limiter == nil {
		s.limiter = newRequestLimiter(s.config)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.allow(clientIP(r), time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeError(w, &APIError{Status: http.StatusTooManyRequests, Message: "request rate limit exceeded", Hint: "Please wait a moment and try again.", Code: "rate_limited"})
			return
		}
		if !s.limiter.acquire() {
			w.Header().Set("Retry-After", "5")
			writeError(w, &APIError{Status: http.StatusTooManyRequests, Message: "too many downloads are in progress", Hint: "Please wait for another download to finish.", Code: "server_busy"})
			return
		}
		defer s.limiter.release()
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
