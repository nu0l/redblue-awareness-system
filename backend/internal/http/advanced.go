package httpserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"redblue-server/internal/db"
)

func (s *Server) handleMatchTemplatesRoot(w http.ResponseWriter, r *http.Request) {
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "templates_auth")
		writeAuthError(w)
		return
	}
	s.handleMatchTemplates(w, r, claims, nil)
}

func (s *Server) handleMatchTemplatesSub(w http.ResponseWriter, r *http.Request) {
	claims, err := requireAuth(r)
	if err != nil {
		auditUnauthorized(r, "templates_auth")
		writeAuthError(w)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/match_templates/")
	parts := splitPath(rest)
	s.handleMatchTemplates(w, r, claims, parts)
}

func (s *Server) handleMatchTemplates(w http.ResponseWriter, r *http.Request, claims *JWTClaims, pathParts []string) {
	if claims.Role != "admin" {
		respondForbidden(w, r)
		return
	}
	if len(pathParts) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListMatchTemplates()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"templates": items})
		case http.MethodPost:
			var req db.MatchTemplate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(req.ID) == "" {
				req.ID = "tpl-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
			}
			if err := s.store.UpsertMatchTemplate(req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, map[string]any{"ok": true, "template_id": req.ID})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id := pathParts[0]
	if r.Method == http.MethodGet {
		item, err := s.store.GetMatchTemplate(id)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"template": item})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleMatchAdvancedEndpoints(w http.ResponseWriter, r *http.Request, claims *JWTClaims, matchID string, parts []string) bool {
	if len(parts) < 2 {
		return false
	}

	if len(parts) == 2 && parts[1] == "tasks" {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListTasks(matchID, r.URL.Query().Get("status"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return true
			}
			writeJSON(w, map[string]any{"tasks": items})
		case http.MethodPost:
			if claims.Role == "observer" {
				respondForbidden(w, r)
				return true
			}
			var req db.TaskItem
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return true
			}
			req.MatchID = matchID
			req.CreatedBy = claims.Sub
			id, err := s.store.CreateTask(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return true
			}
			_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "tasks", Action: "create", After: fmt.Sprintf(`{"task_id":%d}`, id)})
			writeJSON(w, map[string]any{"ok": true, "task_id": id})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return true
	}

	if len(parts) == 3 && parts[1] == "tasks" && r.Method == http.MethodPatch {
		if claims.Role == "observer" {
			respondForbidden(w, r)
			return true
		}
		taskID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return true
		}
		var req struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return true
		}
		if err := s.store.UpdateTaskStatus(matchID, taskID, req.Status, req.Assignee); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		_ = s.store.CreateAuditLog(db.AuditLog{MatchID: matchID, Actor: claims.Sub, Role: claims.Role, Module: "tasks", Action: "update_status", After: fmt.Sprintf(`{"task_id":%d,"status":"%s"}`, taskID, req.Status)})
		writeJSON(w, map[string]any{"ok": true})
		return true
	}

	if len(parts) == 2 && parts[1] == "bookmarks" {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListBookmarks(matchID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return true
			}
			writeJSON(w, map[string]any{"bookmarks": items})
		case http.MethodPost:
			var req db.EventBookmark
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return true
			}
			req.MatchID = matchID
			req.CreatedBy = claims.Sub
			id, err := s.store.CreateBookmark(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return true
			}
			writeJSON(w, map[string]any{"ok": true, "bookmark_id": id})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return true
	}

	if len(parts) == 3 && parts[1] == "analytics" && parts[2] == "kpi" && r.Method == http.MethodGet {
		k, err := s.store.ComputeKPI(matchID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		writeJSON(w, map[string]any{"kpi": k})
		return true
	}

	if len(parts) == 2 && parts[1] == "events_enhanced" && r.Method == http.MethodGet {
		fromSeq := uint64(1)
		if v := strings.TrimSpace(r.URL.Query().Get("from_seq")); v != "" {
			if x, err := strconv.ParseUint(v, 10, 64); err == nil {
				fromSeq = x
			}
		}
		var minScorePtr, maxScorePtr *int
		if v := strings.TrimSpace(r.URL.Query().Get("min_score")); v != "" {
			if x, err := strconv.Atoi(v); err == nil {
				minScorePtr = &x
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("max_score")); v != "" {
			if x, err := strconv.Atoi(v); err == nil {
				maxScorePtr = &x
			}
		}
		teamID := 0
		if v := strings.TrimSpace(r.URL.Query().Get("team_id")); v != "" {
			teamID, _ = strconv.Atoi(v)
		}
		items, err := s.store.ListEventsEnhanced(matchID, fromSeq, db.EventFilter{
			TeamID:     teamID,
			AttackType: r.URL.Query().Get("attack_type"),
			Status:     r.URL.Query().Get("status"),
			MinScore:   minScorePtr,
			MaxScore:   maxScorePtr,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		writeJSON(w, map[string]any{"events": items})
		return true
	}

	if len(parts) == 3 && parts[1] == "analytics" && parts[2] == "trends" && r.Method == http.MethodGet {
		trend, err := s.store.ListScoreTrend(matchID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		writeJSON(w, map[string]any{"trends": trend})
		return true
	}

	if len(parts) == 2 && parts[1] == "report" && r.Method == http.MethodGet {
		kpi, err := s.store.ComputeKPI(matchID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		trend, _ := s.store.ListScoreTrend(matchID)
		report := fmt.Sprintf("# %s 赛后复盘报告\n\n- 总事件数: %d\n- 有效攻击率: %.2f\n- 溯源成功率: %.2f\n- 平均响应时延(秒): %.2f\n- 红蓝净分差: %d\n\n## 维度趋势\n", matchID, kpi.TotalEvents, kpi.EffectiveAttackRate, kpi.TraceSuccessRate, kpi.AvgResponseSeconds, kpi.NetScoreDiff)
		for _, p := range trend {
			report += fmt.Sprintf("- %s: %.0f\n", p.Key, p.Value)
		}
		writeJSON(w, map[string]any{"match_id": matchID, "markdown": report})
		return true
	}

	if len(parts) == 2 && parts[1] == "audit_logs" && r.Method == http.MethodGet {
		if claims.Role != "admin" {
			respondForbidden(w, r)
			return true
		}
		items, err := s.store.ListAuditLogs(matchID, r.URL.Query().Get("actor"), r.URL.Query().Get("module"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		writeJSON(w, map[string]any{"audit_logs": items})
		return true
	}

	return false
}
