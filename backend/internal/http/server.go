package httpserver

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"redblue-server/internal/db"
	"redblue-server/internal/match"
	"redblue-server/internal/protocol"
	"redblue-server/internal/ws"

	"github.com/google/uuid"
)

type Server struct {
	store      *db.Store
	matcher    *match.Service
	hub        *ws.Hub
	uploadsDir string
	upgrader   websocket.Upgrader
}

func isAdminRole(role string) bool {
	return role == "admin" || role == "super_admin"
}

func NewServer(store *db.Store, matcher *match.Service, hub *ws.Hub, uploadsDir string) *Server {
	return &Server{
		store:      store,
		matcher:    matcher,
		hub:        hub,
		uploadsDir: strings.TrimSpace(uploadsDir),
		upgrader: websocket.Upgrader{
			CheckOrigin: websocketCheckOrigin,
		},
	}
}

// websocketCheckOrigin 生产环境请设置 RED_BLUE_WS_ALLOWED_ORIGINS（逗号分隔的完整 Origin，如 https://演练.example.com）。
// 未设置时允许任意 Origin，仅便于本地开发。
func websocketCheckOrigin(r *http.Request) bool {
	allowed := strings.TrimSpace(os.Getenv("RED_BLUE_WS_ALLOWED_ORIGINS"))
	if allowed == "" {
		return true
	}
	origin := normalizeOrigin(strings.TrimSpace(r.Header.Get("Origin")))
	if origin == "" {
		return false
	}
	for _, part := range strings.Split(allowed, ",") {
		if normalizeOrigin(strings.TrimSpace(part)) == origin {
			return true
		}
	}
	return false
}

// normalizeOrigin 统一 Origin 比较，兼容尾斜杠和默认端口写法。
func normalizeOrigin(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	switch {
	case (u.Scheme == "http" && (port == "" || port == "80")),
		(u.Scheme == "https" && (port == "" || port == "443")):
		return strings.ToLower(u.Scheme) + "://" + host
	case port != "":
		return strings.ToLower(u.Scheme) + "://" + host + ":" + port
	default:
		return strings.ToLower(u.Scheme) + "://" + host
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/trigger", s.handleTriggerLegacy)
	mux.HandleFunc("/api/geojson/", s.handleGeoJSONProxy)
	mux.HandleFunc("/api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("/api/admin/reset", s.handleAdminReset)
	mux.HandleFunc("/api/admin/audit_logs", s.handleAdminAuditLogsRoot)
	mux.HandleFunc("/api/admin/audit_logs/", s.handleAdminAuditLogsSub)

	mux.HandleFunc("/api/matches", s.handleMatchesRoot)
	mux.HandleFunc("/api/matches/", s.handleMatchesSub)
	mux.HandleFunc("/api/match_templates", s.handleMatchTemplatesRoot)
	mux.HandleFunc("/api/match_templates/", s.handleMatchTemplatesSub)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": time.Now().Unix()})
	})

	if s.uploadsDir != "" {
		_ = os.MkdirAll(s.uploadsDir, 0o755)
		mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.uploadsDir))))
	}

	return withAccessLog(withCORS(mux))
}

func (s *Server) handleGeoJSONProxy(w http.ResponseWriter, r *http.Request) {
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "geojson_auth")
		writeAuthError(w)
		return
	}
	_ = claims

	rest := strings.TrimPrefix(r.URL.Path, "/api/geojson/")
	key := strings.TrimSpace(rest)
	var upstreamURL string
	switch key {
	case "china":
		upstreamURL = "https://geo.datav.aliyun.com/areas_v3/bound/100000_full.json"
	case "taizhou":
		upstreamURL = "https://geo.datav.aliyun.com/areas_v3/bound/321200_full.json"
	default:
		http.Error(w, "unsupported geojson key", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		http.Error(w, "build upstream request failed", http.StatusInternalServerError)
		return
	}
	// 某些地图源会针对默认 UA 或跨域策略做限制，这里显式设置 UA。
	req.Header.Set("User-Agent", "redblue-awareness-system/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "fetch geojson failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "upstream geojson unavailable", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleAdminReset(w http.ResponseWriter, r *http.Request) {
	// reset 只允许 admin
	claims, err := requireRole(r, "admin")
	if err != nil {
		writeAuthOrForbidden(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST", http.StatusMethodNotAllowed)
		return
	}

	// 保护：必须显式确认字符串，避免误触。
	// 本地开发建议保留：只要你确认点了按钮，就能触发。
	var req struct {
		Confirm string `json:"confirm"`
		MatchID string `json:"match_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Confirm) != "redblue-reset" {
		http.Error(w, "confirm is required", http.StatusBadRequest)
		return
	}

	// 可选开关：如果设置了 RED_BLUE_ALLOW_RESET，就要求等于 true 才允许重置。
	if os.Getenv("RED_BLUE_ALLOW_RESET") != "" && os.Getenv("RED_BLUE_ALLOW_RESET") != "true" {
		http.Error(w, "reset disabled by server config", http.StatusForbidden)
		return
	}

	if strings.TrimSpace(req.MatchID) == "" {
		if err := s.store.ResetAll(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		auditDataReset(r, claims, "", "all")
	} else {
		if err := s.store.ResetMatch(req.MatchID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		auditDataReset(r, claims, strings.TrimSpace(req.MatchID), "match")
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		http.Error(w, "username/password required", http.StatusBadRequest)
		return
	}

	accounts := []struct {
		user string
		pass string
		role string
	}{
		{strings.TrimSpace(os.Getenv("SUPER_ADMIN_USERNAME")), strings.TrimSpace(os.Getenv("SUPER_ADMIN_PASSWORD")), "super_admin"},
		{strings.TrimSpace(os.Getenv("ADMIN_USERNAME")), strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")), "admin"},
		{strings.TrimSpace(os.Getenv("JUDGE_USERNAME")), strings.TrimSpace(os.Getenv("JUDGE_PASSWORD")), "judge"},
		{strings.TrimSpace(os.Getenv("OBSERVER_USERNAME")), strings.TrimSpace(os.Getenv("OBSERVER_PASSWORD")), "observer"},
	}
	found := false
	role := ""
	for _, a := range accounts {
		if a.user == "" || a.pass == "" {
			continue
		}
		if req.Username == a.user && subtleTimeEqual(req.Password, a.pass) {
			found = true
			role = a.role
			break
		}
	}
	if !found && strings.TrimSpace(os.Getenv("ADMIN_USERNAME")) == "" {
		auditLoginFailure(r, req.Username, "admin_not_configured")
		http.Error(w, "admin credentials are not configured", http.StatusInternalServerError)
		return
	}
	if !found {
		auditLoginFailure(r, req.Username, "invalid_username_or_password")
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := signJWT(req.Username, role, 7*24*time.Hour)
	if err != nil {
		auditLoginFailure(r, req.Username, "token_sign_error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auditLoginSuccess(r, req.Username)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"role":  role,
	})
}

func subtleTimeEqual(a, b string) bool {
	// use a constant-time comparison; length check included by compare itself
	ab := []byte(a)
	bb := []byte(b)
	if len(ab) != len(bb) {
		return false
	}
	// local helper avoids importing crypto/subtle again in this file
	var diff byte
	for i := 0; i < len(ab); i++ {
		diff |= ab[i] ^ bb[i]
	}
	return diff == 0
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// 强制鉴权：避免未授权抓取实时/回放数据。
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "ws_auth")
		writeAuthError(w)
		return
	}

	matchID := r.URL.Query().Get("match_id")
	if strings.TrimSpace(matchID) == "" {
		// 兼容旧版大屏：如果未提供 match_id，就取最新一场次。
		matches, err := s.store.ListMatches()
		if err != nil || len(matches) == 0 {
			http.Error(w, "match_id is required", http.StatusBadRequest)
			return
		}
		matchID = matches[0].ID
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade 失败时 gorilla/websocket 通常已写出 HTTP 400，不能再 http.Error 二次写头。
		slog.Warn("audit_ws_upgrade_failed",
			"remote_ip", clientIP(r),
			"match_id", matchID,
			"origin", r.Header.Get("Origin"),
			"upgrade", r.Header.Get("Upgrade"),
			"connection", r.Header.Get("Connection"),
			"error", err.Error(),
		)
		return
	}

	_ = s.hub.AddClient(matchID, conn)
	auditWSConnect(r, claims, matchID)

	// 连接建立后立即发送状态快照。
	state, err := s.matcher.GetStateDTO(matchID)
	if err != nil {
		http.Error(w, "failed to load match state", http.StatusInternalServerError)
		_ = conn.Close()
		return
	}

	// last_seq: 用 events 的 max seq 获取
	lastSeq, err := s.store.GetLastSeq(matchID)
	if err != nil {
		_ = conn.Close()
		return
	}

	nowTs := time.Now().Unix()
	s.hub.Broadcast(matchID, protocol.WSMessage{
		Type:      "sync_state",
		MatchID:   matchID,
		Seq:       lastSeq,
		Timestamp: nowTs,
		State:     state,
	})

	// 给当前 client 发送一次即可，避免 Broadcast 影响其他在线客户端。
	//（为了避免并发复杂度，这里先用单次写入替代；当前 hub 没有单播接口。）
	// 由于 broadcast 会发送给同房间所有客户端，这里“可接受”，但在后续可优化为单播。
	_ = lastSeq
}

// handleTriggerLegacy 兼容旧版 admin.html 的 /api/trigger。
// 旧版 payload 形如：{ "event": "...", "data": {...} }
func (s *Server) handleTriggerLegacy(w http.ResponseWriter, r *http.Request) {
	// 旧版 admin 兼容接口：需要 admin 鉴权。
	claims, err := requireRole(r, "admin")
	if err != nil {
		writeAuthOrForbidden(w, r, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST", http.StatusMethodNotAllowed)
		return
	}

	type legacyBody struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}

	var body legacyBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON 解析失败", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Event) == "" {
		http.Error(w, "event is required", http.StatusBadRequest)
		return
	}

	// 旧版没有 match_id：默认取最新一场次。
	matches, err := s.store.ListMatches()
	if err != nil || len(matches) == 0 {
		http.Error(w, "no matches found", http.StatusBadRequest)
		return
	}
	matchID := matches[0].ID

	wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{
		EventType: body.Event,
		Data:      body.Data,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.hub.Broadcast(matchID, *wsMsg)
	auditAdminCommand(r, claims, matchID, body.Event, wsMsg.Seq)
	_ = wsMsg

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "msg": "指令已推送到大屏"})
}

func (s *Server) handleMatchesRoot(w http.ResponseWriter, r *http.Request) {
	// matches 相关接口也需要 token（后台/大屏均需授权）
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "auth")
		writeAuthError(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListMatches()
		if err != nil {
			http.Error(w, "failed to list matches", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"matches": list})
	case http.MethodPost:
		// 创建新场次只允许 admin
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}

		// 创建新场次（match_id 即为一个独立演练，用于多场同时与历史复盘）
		var req struct {
			MapType    string `json:"map_type"`
			TemplateID string `json:"template_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		mapType := req.MapType
		if mapType == "" {
			mapType = "china"
		}
		if strings.TrimSpace(req.TemplateID) != "" {
			if tpl, err := s.store.GetMatchTemplate(strings.TrimSpace(req.TemplateID)); err == nil && strings.TrimSpace(tpl.MapType) != "" {
				mapType = tpl.MapType
			}
		}

		if mapType != "china" && mapType != "taizhou" {
			http.Error(w, "invalid map_type", http.StatusBadRequest)
			return
		}

		id := uuid.NewString()
		panels := map[string]bool{
			"panel-leaderboard": true,
		}
		if err := s.store.CreateMatch(
			id,
			mapType,
			true,
			panels,
			"实战化红蓝对抗演练指挥中心",
			"",
			"",
		); err != nil {
			http.Error(w, "failed to create match", http.StatusInternalServerError)
			return
		}
		auditMatchCreated(r, claims, id, mapType)
		writeJSON(w, map[string]any{"match_id": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMatchesSub(w http.ResponseWriter, r *http.Request) {
	// 所有匹配子资源都需要 token
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "auth")
		writeAuthError(w)
		return
	}

	// Path: /api/matches/{match_id}/xxx...
	rest := strings.TrimPrefix(r.URL.Path, "/api/matches/")
	parts := splitPath(rest)
	if len(parts) < 1 {
		http.NotFound(w, r)
		return
	}
	matchID := parts[0]
	if matchID == "" {
		http.Error(w, "match_id is required", http.StatusBadRequest)
		return
	}

	if s.handleMatchAdvancedEndpoints(w, r, claims, matchID, parts) {
		return
	}

	// /api/matches/{match_id}/leaderboard_background  （得分总榜背景图上传/清除）
	if len(parts) == 2 && parts[1] == "leaderboard_background" {
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost:
			s.handleLeaderboardBackgroundUpload(w, r, matchID, claims)
		case http.MethodDelete:
			s.handleLeaderboardBackgroundDelete(w, r, matchID, claims)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/matches/{match_id}/audio_upload/{kind} （kind: bgm | success_sfx）
	if len(parts) == 3 && parts[1] == "audio_upload" && r.Method == http.MethodPost {
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}
		kind := parts[2]
		if kind != "bgm" && kind != "success_sfx" {
			http.Error(w, "kind must be bgm or success_sfx", http.StatusBadRequest)
			return
		}
		s.handleAudioUpload(w, r, matchID, kind, claims)
		return
	}

	// /api/matches/{match_id}/state
	if len(parts) == 2 && parts[1] == "state" && r.Method == http.MethodGet {
		state, err := s.matcher.GetStateDTO(matchID)
		if err != nil {
			http.Error(w, "failed to load state", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"state": state})
		return
	}

	// /api/matches/{match_id}/initial_state
	if len(parts) == 2 && parts[1] == "initial_state" && r.Method == http.MethodGet {
		state, err := s.matcher.GetInitialStateDTO(matchID)
		if err != nil {
			http.Error(w, "failed to load initial state", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"state": state})
		return
	}

	// /api/matches/{match_id}/teams
	if len(parts) == 2 && parts[1] == "teams" {
		switch r.Method {
		case http.MethodGet:
			teams, err := s.store.ListTeams(matchID)
			if err != nil {
				http.Error(w, "failed to list teams", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"teams": teams})
		case http.MethodPost:
			if !isAdminRole(claims.Role) {
				respondForbidden(w, r)
				return
			}
			beforeState := s.captureMatchStateJSON(matchID)

			var req protocol.TeamDTO
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
				http.Error(w, "name/type are required", http.StatusBadRequest)
				return
			}
			if req.Type != "red" && req.Type != "blue" {
				http.Error(w, "type must be red or blue", http.StatusBadRequest)
				return
			}
			if req.Logo == "" {
				req.Logo = "?"
			}
			teamID, err := s.store.CreateTeam(matchID, req)
			if err != nil {
				http.Error(w, "failed to create team", http.StatusInternalServerError)
				return
			}
			// 队伍变更需要推送给大屏：复用 teams_updated 的状态快照事件。
			if wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{
				EventType: "teams_updated",
				Data:      json.RawMessage(`{}`),
			}); err == nil {
				s.hub.Broadcast(matchID, *wsMsg)
			}
			auditTeamMutation(r, claims, matchID, "create", teamID)
			afterState := s.captureMatchStateJSON(matchID)
			_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "teams", Action: "create", Before: beforeState, After: afterState})
			writeJSON(w, map[string]any{"team_id": teamID})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/matches/{match_id}/teams/import
	if len(parts) == 3 && parts[1] == "teams" && parts[2] == "import" && r.Method == http.MethodPost {
		beforeState := s.captureMatchStateJSON(matchID)
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}
		var req struct {
			CSVText string `json:"csv_text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		reader := csv.NewReader(bytes.NewBufferString(req.CSVText))
		rows, err := reader.ReadAll()
		if err != nil {
			http.Error(w, "invalid csv", http.StatusBadRequest)
			return
		}
		created := 0
		for i, row := range rows {
			if i == 0 && len(row) >= 2 && strings.Contains(strings.ToLower(row[0]), "name") {
				continue
			}
			if len(row) < 2 {
				continue
			}
			name := strings.TrimSpace(row[0])
			typ := strings.TrimSpace(row[1])
			if name == "" || (typ != "red" && typ != "blue") {
				continue
			}
			dto := protocol.TeamDTO{
				Name:    name,
				Type:    typ,
				Logo:    "?",
				Score:   0,
				Members: []string{},
			}
			if len(row) >= 3 && strings.TrimSpace(row[2]) != "" {
				dto.Members = strings.Split(strings.TrimSpace(row[2]), "|")
			}
			if _, err := s.store.CreateTeam(matchID, dto); err == nil {
				created++
			}
		}
		if wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{EventType: "teams_updated", Data: json.RawMessage(`{}`)}); err == nil {
			s.hub.Broadcast(matchID, *wsMsg)
		}
		afterState := s.captureMatchStateJSON(matchID)
		_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "teams", Action: "import_csv", Before: beforeState, After: afterState})
		writeJSON(w, map[string]any{"ok": true, "created": created})
		return
	}

	// /api/matches/{match_id}/teams/batch_update
	if len(parts) == 3 && parts[1] == "teams" && parts[2] == "batch_update" && r.Method == http.MethodPut {
		beforeState := s.captureMatchStateJSON(matchID)
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}
		var req struct {
			Teams []protocol.TeamDTO `json:"teams"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		for _, t := range req.Teams {
			if t.ID > 0 {
				_ = s.store.UpdateTeam(matchID, t.ID, t)
				continue
			}
			_, _ = s.store.CreateTeam(matchID, t)
		}
		if wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{EventType: "teams_updated", Data: json.RawMessage(`{}`)}); err == nil {
			s.hub.Broadcast(matchID, *wsMsg)
		}
		afterState := s.captureMatchStateJSON(matchID)
		_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "teams", Action: "batch_update", Before: beforeState, After: afterState})
		writeJSON(w, map[string]any{"ok": true, "updated": len(req.Teams)})
		return
	}

	// /api/matches/{match_id}/teams/{team_id}
	if len(parts) == 3 && parts[1] == "teams" && r.Method == http.MethodPut {
		beforeState := s.captureMatchStateJSON(matchID)
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}

		teamID, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "invalid team_id", http.StatusBadRequest)
			return
		}
		var req protocol.TeamDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
			http.Error(w, "name/type are required", http.StatusBadRequest)
			return
		}
		if req.Type != "red" && req.Type != "blue" {
			http.Error(w, "type must be red or blue", http.StatusBadRequest)
			return
		}
		if req.Logo == "" {
			req.Logo = "?"
		}
		req.ID = teamID
		if err := s.store.UpdateTeam(matchID, teamID, req); err != nil {
			http.Error(w, "failed to update team", http.StatusInternalServerError)
			return
		}
		// 推送状态刷新到大屏
		if wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{
			EventType: "teams_updated",
			Data:      json.RawMessage(`{}`),
		}); err == nil {
			s.hub.Broadcast(matchID, *wsMsg)
		}
		auditTeamMutation(r, claims, matchID, "update", teamID)
		afterState := s.captureMatchStateJSON(matchID)
		_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "teams", Action: "update", Before: beforeState, After: afterState})
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	if len(parts) == 3 && parts[1] == "teams" && r.Method == http.MethodDelete {
		beforeState := s.captureMatchStateJSON(matchID)
		if !isAdminRole(claims.Role) {
			respondForbidden(w, r)
			return
		}

		teamID, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "invalid team_id", http.StatusBadRequest)
			return
		}
		if err := s.store.DeleteTeam(matchID, teamID); err != nil {
			http.Error(w, "failed to delete team", http.StatusInternalServerError)
			return
		}
		if wsMsg, err := s.matcher.ApplyCommand(matchID, match.CmdMessage{
			EventType: "teams_updated",
			Data:      json.RawMessage(`{}`),
		}); err == nil {
			s.hub.Broadcast(matchID, *wsMsg)
		}
		auditTeamMutation(r, claims, matchID, "delete", teamID)
		afterState := s.captureMatchStateJSON(matchID)
		_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "teams", Action: "delete", Before: beforeState, After: afterState})
		writeJSON(w, map[string]any{"ok": true})
		return
	}

	// /api/matches/{match_id}/events?from_seq=1&limit=200
	if len(parts) == 2 && parts[1] == "events" && r.Method == http.MethodGet {
		fromSeq := uint64(1)
		limit := 200
		if v := r.URL.Query().Get("from_seq"); v != "" {
			if x, err := strconv.ParseUint(v, 10, 64); err == nil {
				fromSeq = x
			}
		}
		if v := r.URL.Query().Get("limit"); v != "" {
			if x, err := strconv.Atoi(v); err == nil {
				limit = x
			}
		}

		evs, err := s.store.ListEvents(matchID, fromSeq, limit)
		if err != nil {
			http.Error(w, "failed to list events", http.StatusInternalServerError)
			return
		}
		// 直接返回 payload_json（RawMessage）
		writeJSON(w, map[string]any{"events": evs})
		return
	}

	// /api/matches/{match_id}/command
	if len(parts) == 2 && parts[1] == "command" && r.Method == http.MethodPost {
		if claims.Role == "observer" {
			respondForbidden(w, r)
			return
		}

		var req match.CmdMessage
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		beforeState := ""
		if st, err := s.matcher.GetStateDTO(matchID); err == nil {
			if b, e := json.Marshal(st); e == nil {
				beforeState = string(b)
			}
		}
		// 交给 matcher 执行并返回 state+broadcast 消息
		wsMsg, err := s.matcher.ApplyCommand(matchID, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// 广播事件与同步态
		s.hub.Broadcast(matchID, *wsMsg)
		auditAdminCommand(r, claims, matchID, req.EventType, wsMsg.Seq)
		_ = s.store.CreateAuditLog(db.AuditLog{
			MatchID: matchID,
			Actor:   claims.Sub,
			Role:    claims.Role,
			Module:  "command",
			Action:  req.EventType,
			Before:  beforeState,
			After:   string(req.Data),
		})
		switch req.EventType {
		case "attack_success", "manual_score", "system_broadcast":
			taskTitle := "任务流事件"
			switch req.EventType {
			case "attack_success":
				taskTitle = "任务流：攻击事件"
			case "manual_score":
				taskTitle = "任务流：裁判加扣分"
			case "system_broadcast":
				taskTitle = "任务流：广播通知"
			}
			_, _ = s.store.CreateTask(db.TaskItem{
				MatchID:   matchID,
				Category:  "task_flow",
				Title:     taskTitle,
				Status:    "done",
				Assignee:  claims.Sub,
				CreatedBy: claims.Sub,
				Payload:   req.Data,
			})
		}
		writeJSON(w, map[string]any{"ok": true, "seq": wsMsg.Seq})
		return
	}

	// 兜底
	http.Error(w, "not found", http.StatusNotFound)
}

func imageExtFromMagic(b []byte) string {
	if len(b) >= 8 && b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G' {
		return ".png"
	}
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff {
		return ".jpg"
	}
	if len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP" {
		return ".webp"
	}
	return ""
}

func audioExtFromFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	switch ext {
	case ".mp3", ".wav", ".ogg", ".m4a", ".aac":
		return ext
	default:
		return ""
	}
}

func audioExtFromContentType(ct string) string {
	low := strings.ToLower(strings.TrimSpace(ct))
	switch {
	case strings.Contains(low, "audio/mpeg"), strings.Contains(low, "audio/mp3"):
		return ".mp3"
	case strings.Contains(low, "audio/wav"), strings.Contains(low, "audio/x-wav"):
		return ".wav"
	case strings.Contains(low, "audio/ogg"):
		return ".ogg"
	case strings.Contains(low, "audio/mp4"), strings.Contains(low, "audio/m4a"), strings.Contains(low, "audio/aac"):
		return ".m4a"
	default:
		return ""
	}
}

func audioExtFromMagic(b []byte) string {
	if len(b) >= 3 && string(b[:3]) == "ID3" {
		return ".mp3"
	}
	if len(b) >= 2 && b[0] == 0xff && (b[1]&0xe0) == 0xe0 {
		return ".mp3" // MPEG frame sync
	}
	if len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WAVE" {
		return ".wav"
	}
	if len(b) >= 4 && string(b[0:4]) == "OggS" {
		return ".ogg"
	}
	if len(b) >= 12 && string(b[4:8]) == "ftyp" {
		return ".m4a"
	}
	return ""
}

func (s *Server) broadcastSyncState(matchID string) {
	state, err := s.matcher.GetStateDTO(matchID)
	if err != nil {
		return
	}
	lastSeq, err := s.store.GetLastSeq(matchID)
	if err != nil {
		return
	}
	s.hub.Broadcast(matchID, protocol.WSMessage{
		Type:      "sync_state",
		MatchID:   matchID,
		Seq:       lastSeq,
		Timestamp: time.Now().Unix(),
		State:     state,
	})
}

func (s *Server) handleLeaderboardBackgroundUpload(w http.ResponseWriter, r *http.Request, matchID string, claims *JWTClaims) {
	if s.uploadsDir == "" {
		http.Error(w, "upload disabled (uploads dir not configured)", http.StatusServiceUnavailable)
		return
	}
	beforeState := s.captureMatchStateJSON(matchID)
	if _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, err := s.store.GetMatchPanels(matchID); err != nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20+1024)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "multipart parse failed", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required (field name: file)", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 8<<20))
	if err != nil || len(data) == 0 {
		http.Error(w, "empty or invalid file", http.StatusBadRequest)
		return
	}
	ext := imageExtFromMagic(data)
	if ext == "" {
		http.Error(w, "only png/jpeg/webp images are supported", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(s.uploadsDir, matchID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}
	old, _ := filepath.Glob(filepath.Join(dir, "leaderboard-bg.*"))
	for _, p := range old {
		_ = os.Remove(p)
	}
	dstName := "leaderboard-bg" + ext
	dstPath := filepath.Join(dir, dstName)
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		http.Error(w, "failed to write file", http.StatusInternalServerError)
		return
	}

	pubPath := "/uploads/" + matchID + "/" + dstName
	if err := s.store.UpdateMatchLeaderboardBG(matchID, pubPath); err != nil {
		http.Error(w, "failed to update database", http.StatusInternalServerError)
		return
	}
	s.matcher.InvalidateCache(matchID)
	s.broadcastSyncState(matchID)
	auditFileUpload(r, claims, matchID, "leaderboard_bg", pubPath)
	afterState := s.captureMatchStateJSON(matchID)
	_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "screen", Action: "leaderboard_bg_upload", Before: beforeState, After: afterState})
	writeJSON(w, map[string]any{"ok": true, "leaderboard_bg_url": pubPath})
}

func (s *Server) handleLeaderboardBackgroundDelete(w http.ResponseWriter, r *http.Request, matchID string, claims *JWTClaims) {
	beforeState := s.captureMatchStateJSON(matchID)
	if _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, err := s.store.GetMatchPanels(matchID); err != nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}
	if err := s.store.UpdateMatchLeaderboardBG(matchID, ""); err != nil {
		http.Error(w, "failed to update database", http.StatusInternalServerError)
		return
	}
	if s.uploadsDir != "" {
		dir := filepath.Join(s.uploadsDir, matchID)
		old, _ := filepath.Glob(filepath.Join(dir, "leaderboard-bg.*"))
		for _, p := range old {
			_ = os.Remove(p)
		}
	}
	s.matcher.InvalidateCache(matchID)
	s.broadcastSyncState(matchID)
	auditFileUpload(r, claims, matchID, "leaderboard_bg_clear", "")
	afterState := s.captureMatchStateJSON(matchID)
	_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "screen", Action: "leaderboard_bg_clear", Before: beforeState, After: afterState})
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleAudioUpload(w http.ResponseWriter, r *http.Request, matchID string, kind string, claims *JWTClaims) {
	if s.uploadsDir == "" {
		http.Error(w, "upload disabled (uploads dir not configured)", http.StatusServiceUnavailable)
		return
	}
	beforeState := s.captureMatchStateJSON(matchID)
	mapType,
		leaderboardVisible,
		panels,
		_,
		_,
		_,
		_,
		_,
		screenTitle,
		screenOrganizer,
		screenSupporter,
		_,
		leaderboardBG,
		bgmURL,
		bgmEnabled,
		successSFXURL,
		successSFXEnabled,
		leaderboardMainAlpha,
		err := s.store.GetMatchPanels(matchID)
	_ = mapType
	_ = leaderboardVisible
	_ = panels
	_ = screenTitle
	_ = screenOrganizer
	_ = screenSupporter
	_ = leaderboardBG
	_ = leaderboardMainAlpha
	if err != nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 20<<20+1024)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "multipart parse failed", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required (field name: file)", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 20<<20))
	if err != nil || len(data) == 0 {
		http.Error(w, "empty or invalid file", http.StatusBadRequest)
		return
	}
	ext := audioExtFromFilename(header.Filename)
	if ext == "" {
		ext = audioExtFromContentType(header.Header.Get("Content-Type"))
	}
	if ext == "" {
		ext = audioExtFromMagic(data)
	}
	if ext == "" {
		http.Error(w, "unsupported audio format (allowed: mp3/wav/ogg/m4a/aac)", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(s.uploadsDir, matchID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}
	baseName := "bgm"
	if kind == "success_sfx" {
		baseName = "success-sfx"
	}
	old, _ := filepath.Glob(filepath.Join(dir, baseName+".*"))
	for _, p := range old {
		_ = os.Remove(p)
	}
	dstName := baseName + ext
	dstPath := filepath.Join(dir, dstName)
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		http.Error(w, "failed to write file", http.StatusInternalServerError)
		return
	}

	pubPath := "/uploads/" + matchID + "/" + dstName
	newBGMURL := bgmURL
	newSuccessSFXURL := successSFXURL
	if kind == "bgm" {
		newBGMURL = pubPath
	} else {
		newSuccessSFXURL = pubPath
	}
	if err := s.store.UpdateMatchAudioConfig(matchID, newBGMURL, bgmEnabled, newSuccessSFXURL, successSFXEnabled); err != nil {
		http.Error(w, "failed to update database", http.StatusInternalServerError)
		return
	}
	s.matcher.InvalidateCache(matchID)
	s.broadcastSyncState(matchID)
	auditFileUpload(r, claims, matchID, kind, pubPath)
	afterState := s.captureMatchStateJSON(matchID)
	_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "audio", Action: "upload_" + kind, Before: beforeState, After: afterState})
	writeJSON(w, map[string]any{"ok": true, "kind": kind, "url": pubPath})
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) captureMatchStateJSON(matchID string) string {
	st, err := s.matcher.GetStateDTO(matchID)
	if err != nil || st == nil {
		return ""
	}
	b, err := json.Marshal(st)
	if err != nil {
		return ""
	}
	return string(b)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 本地固定部署，暂时放开 CORS，后续可加鉴权/白名单。
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
