package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
)

type APIError struct {
	Status  int
	Message string
	Hint    string
	Code    string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

type errorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
	Hint   string `json:"hint,omitempty"`
	Code   string `json:"code,omitempty"`
}

func writeError(w http.ResponseWriter, err error) {
	apiErr := asAPIError(err)
	if apiErr.Cause != nil {
		log.Printf("api error: status=%d message=%q cause=%v", apiErr.Status, apiErr.Message, apiErr.Cause)
	}
	if apiErr.Status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="zzz"`)
	}
	if apiErr.Code != "" {
		w.Header().Set("X-ZZZ-Error-Code", apiErr.Code)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(apiErr.Status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:  apiErr.Message,
		Status: apiErr.Status,
		Hint:   apiErr.Hint,
		Code:   apiErr.Code,
	})
}

func asAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return &APIError{Status: http.StatusInternalServerError, Message: "internal server error", Cause: err}
}

func invalidError(message string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Message: message}
}

func networkError(err error) *APIError {
	status := http.StatusBadGateway
	message := "GitHub request failed"
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		status = http.StatusGatewayTimeout
		message = "GitHub request timed out"
	}
	return &APIError{
		Status:  status,
		Message: message,
		Hint:    "请检查服务器或 Docker 容器是否能够访问 api.github.com；如果仓库较大，可适当增加 GITHUB_TIMEOUT_SECS。",
		Cause:   err,
	}
}

func githubError(status int, message string) *APIError {
	apiStatus := status
	if status < 400 || status >= 600 {
		apiStatus = http.StatusBadGateway
	}
	if status != http.StatusUnauthorized && status != http.StatusForbidden && status != http.StatusNotFound && status != http.StatusTooManyRequests {
		apiStatus = http.StatusBadGateway
	}
	apiErr := &APIError{
		Status:  apiStatus,
		Message: fmt.Sprintf("GitHub returned %d: %s", status, message),
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		apiErr.Code = "github_auth"
	}
	if status == http.StatusTooManyRequests {
		apiErr.Code = "github_rate_limit"
	}
	switch status {
	case http.StatusUnauthorized:
		apiErr.Hint = "GitHub Token 无效或已过期，请检查 Token 后重试。"
	case http.StatusForbidden:
		if containsRateLimit(message) {
			apiErr.Hint = "GitHub API 已达到速率限制，请配置有效 Token 或稍后重试。"
		} else {
			apiErr.Hint = "GitHub 拒绝了请求，可能是 API 限流或仓库权限不足。"
		}
	case http.StatusNotFound:
		apiErr.Hint = "仓库、分支或路径不存在，私有仓库还可能是 Token 无权访问。"
	case http.StatusTooManyRequests:
		apiErr.Hint = "GitHub API 已达到速率限制，请稍后重试。"
	}
	return apiErr
}

func containsRateLimit(message string) bool {
	for _, value := range []string{"rate limit", "api rate", "secondary rate"} {
		if containsFold(message, value) {
			return true
		}
	}
	return false
}

func containsFold(value, needle string) bool {
	return len(value) >= len(needle) && stringContainsFold(value, needle)
}
