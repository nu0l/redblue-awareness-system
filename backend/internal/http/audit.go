package httpserver

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var jsonLogOnce sync.Once

// ensureJSONLog 若设置 RED_BLUE_JSON_LOG=true，则使用 JSON 结构化日志，便于 ELK/日志平台采集与溯源检索。
func ensureJSONLog() {
	jsonLogOnce.Do(func() {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("RED_BLUE_JSON_LOG")), "true") {
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})))
		}
	})
}

// clientIP 解析客户端 IP（优先 X-Forwarded-For / X-Real-IP，便于反向代理后溯源）。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func auditUnauthorized(r *http.Request, reason string) {
	ensureJSONLog()
	slog.Warn("audit_auth_failure",
		"remote_ip", clientIP(r),
		"method", r.Method,
		"path", r.URL.Path,
		"reason", reason,
	)
}

// respondForbidden 已登录但非 admin（或业务上禁止访问）。
func respondForbidden(w http.ResponseWriter, r *http.Request) {
	ensureJSONLog()
	slog.Warn("audit_forbidden",
		"remote_ip", clientIP(r),
		"method", r.Method,
		"path", r.URL.Path,
	)
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
}

// writeAuthOrForbidden 已鉴权失败时：token 问题走 401；角色不足走 403（便于与“未登录”区分）。
func writeAuthOrForbidden(w http.ResponseWriter, r *http.Request, err error) {
	ensureJSONLog()
	if errors.Is(err, ErrForbidden) {
		slog.Warn("audit_forbidden",
			"remote_ip", clientIP(r),
			"method", r.Method,
			"path", r.URL.Path,
		)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	auditUnauthorized(r, "missing_or_invalid_token")
	writeAuthError(w)
}

// auditLoginSuccess 登录成功（禁止记录密码）。
func auditLoginSuccess(r *http.Request, username string) {
	ensureJSONLog()
	slog.Info("audit_login_success",
		"remote_ip", clientIP(r),
		"username", username,
	)
}

// auditLoginFailure 登录失败（仅记录用户名与原因，不记录密码）。
func auditLoginFailure(r *http.Request, username string, reason string) {
	ensureJSONLog()
	slog.Warn("audit_login_failure",
		"remote_ip", clientIP(r),
		"username", username,
		"reason", reason,
	)
}

func auditWSConnect(r *http.Request, claims *JWTClaims, matchID string) {
	ensureJSONLog()
	slog.Info("audit_ws_connect",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"role", claims.Role,
		"match_id", matchID,
	)
}

func auditAdminCommand(r *http.Request, claims *JWTClaims, matchID string, eventType string, seq uint64) {
	ensureJSONLog()
	slog.Info("audit_admin_command",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"match_id", matchID,
		"event_type", eventType,
		"seq", seq,
	)
}

func auditDataReset(r *http.Request, claims *JWTClaims, matchID string, scope string) {
	ensureJSONLog()
	slog.Warn("audit_data_reset",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"scope", scope,
		"match_id", matchID,
	)
}

func auditMatchCreated(r *http.Request, claims *JWTClaims, matchID string, mapType string) {
	ensureJSONLog()
	slog.Info("audit_match_created",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"match_id", matchID,
		"map_type", mapType,
	)
}

func auditTeamMutation(r *http.Request, claims *JWTClaims, matchID string, action string, teamID int) {
	ensureJSONLog()
	slog.Info("audit_team_mutation",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"match_id", matchID,
		"action", action,
		"team_id", teamID,
	)
}

func auditFileUpload(r *http.Request, claims *JWTClaims, matchID string, kind string, path string) {
	ensureJSONLog()
	slog.Info("audit_file_upload",
		"remote_ip", clientIP(r),
		"subject", claims.Sub,
		"match_id", matchID,
		"kind", kind,
		"path", path,
	)
}

// --- HTTP 访问日志（全链路请求耗时与状态码）---

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.written {
		return
	}
	s.written = true
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(b)
}

// Hijack 透传到底层 ResponseWriter，供 WebSocket Upgrade 使用。
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response does not implement http.Hijacker")
	}
	return hj.Hijack()
}

// Flush 透传到底层 ResponseWriter，兼容流式响应。
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Push 透传到底层 ResponseWriter（HTTP/2 server push）。
func (s *statusRecorder) Push(target string, opts *http.PushOptions) error {
	p, ok := s.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

func withAccessLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ensureJSONLog()
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w}
		h.ServeHTTP(sr, r)
		dur := time.Since(start)
		status := sr.status
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("http_access",
			"remote_ip", clientIP(r),
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", dur.Milliseconds(),
		)
	})
}
