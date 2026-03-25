package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type MatchTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     int             `json:"version"`
	MapType     string          `json:"map_type"`
	Cities      []string        `json:"cities"`
	AttackTypes []string        `json:"attack_types"`
	AudioConfig json.RawMessage `json:"audio_config"`
	ScoreRules  json.RawMessage `json:"score_rules"`
	ChangeLog   string          `json:"change_log"`
	CreatedAt   int64           `json:"created_at"`
	UpdatedAt   int64           `json:"updated_at"`
}

type TaskItem struct {
	ID        int64           `json:"id"`
	MatchID   string          `json:"match_id"`
	Category  string          `json:"category"`
	Title     string          `json:"title"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	Assignee  string          `json:"assignee"`
	CreatedBy string          `json:"created_by"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

type EventBookmark struct {
	ID        int64  `json:"id"`
	MatchID   string `json:"match_id"`
	Seq       uint64 `json:"seq"`
	Title     string `json:"title"`
	Note      string `json:"note"`
	CreatedBy string `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
}

type AnalyticsKPI struct {
	TotalEvents         int     `json:"total_events"`
	EffectiveAttackRate float64 `json:"effective_attack_rate"`
	TraceSuccessRate    float64 `json:"trace_success_rate"`
	AvgResponseSeconds  float64 `json:"avg_response_seconds"`
	NetScoreDiff        int     `json:"net_score_diff"`
}

type TrendPoint struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

type AuditLog struct {
	ID        int64  `json:"id"`
	MatchID   string `json:"match_id"`
	Actor     string `json:"actor"`
	Role      string `json:"role"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Before    string `json:"before"`
	After     string `json:"after"`
	CreatedAt int64  `json:"created_at"`
}

type EventFilter struct {
	TeamID     int
	AttackType string
	Status     string
	MinScore   *int
	MaxScore   *int
	Limit      int
}

func (s *Store) ListMatchTemplates() ([]MatchTemplate, error) {
	rows, err := s.db.Query(`SELECT id, name, version, map_type, cities_json, attack_types_json, audio_config_json, score_rules_json, change_log, created_at, updated_at FROM match_templates ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MatchTemplate, 0)
	for rows.Next() {
		var item MatchTemplate
		var citiesJSON, typesJSON string
		var audioJSON, scoreJSON string
		if err := rows.Scan(&item.ID, &item.Name, &item.Version, &item.MapType, &citiesJSON, &typesJSON, &audioJSON, &scoreJSON, &item.ChangeLog, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(citiesJSON), &item.Cities)
		_ = json.Unmarshal([]byte(typesJSON), &item.AttackTypes)
		if strings.TrimSpace(audioJSON) == "" {
			audioJSON = "{}"
		}
		if strings.TrimSpace(scoreJSON) == "" {
			scoreJSON = "{}"
		}
		item.AudioConfig = json.RawMessage(audioJSON)
		item.ScoreRules = json.RawMessage(scoreJSON)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertMatchTemplate(t MatchTemplate) error {
	now := time.Now().Unix()
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("template id required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("template name required")
	}
	if t.Version <= 0 {
		t.Version = 1
	}
	citiesJSON, _ := json.Marshal(t.Cities)
	typesJSON, _ := json.Marshal(t.AttackTypes)
	if len(t.AudioConfig) == 0 {
		t.AudioConfig = json.RawMessage(`{}`)
	}
	if len(t.ScoreRules) == 0 {
		t.ScoreRules = json.RawMessage(`{}`)
	}
	_, err := s.db.Exec(`INSERT INTO match_templates(id,name,version,map_type,cities_json,attack_types_json,audio_config_json,score_rules_json,change_log,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
name=excluded.name,version=excluded.version,map_type=excluded.map_type,cities_json=excluded.cities_json,attack_types_json=excluded.attack_types_json,audio_config_json=excluded.audio_config_json,score_rules_json=excluded.score_rules_json,change_log=excluded.change_log,updated_at=excluded.updated_at`,
		t.ID, t.Name, t.Version, t.MapType, string(citiesJSON), string(typesJSON), string(t.AudioConfig), string(t.ScoreRules), t.ChangeLog, now, now)
	return err
}

func (s *Store) GetMatchTemplate(id string) (*MatchTemplate, error) {
	var item MatchTemplate
	var citiesJSON, typesJSON string
	var audioJSON, scoreJSON string
	if err := s.db.QueryRow(`SELECT id, name, version, map_type, cities_json, attack_types_json, audio_config_json, score_rules_json, change_log, created_at, updated_at FROM match_templates WHERE id=?`, id).
		Scan(&item.ID, &item.Name, &item.Version, &item.MapType, &citiesJSON, &typesJSON, &audioJSON, &scoreJSON, &item.ChangeLog, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(citiesJSON), &item.Cities)
	_ = json.Unmarshal([]byte(typesJSON), &item.AttackTypes)
	if strings.TrimSpace(audioJSON) == "" {
		audioJSON = "{}"
	}
	if strings.TrimSpace(scoreJSON) == "" {
		scoreJSON = "{}"
	}
	item.AudioConfig = json.RawMessage(audioJSON)
	item.ScoreRules = json.RawMessage(scoreJSON)
	return &item, nil
}

func (s *Store) CreateTask(task TaskItem) (int64, error) {
	now := time.Now().Unix()
	if strings.TrimSpace(task.Status) == "" {
		task.Status = "todo"
	}
	if len(task.Payload) == 0 {
		task.Payload = json.RawMessage(`{}`)
	}
	res, err := s.db.Exec(`INSERT INTO tasks(match_id,category,title,payload_json,status,assignee,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		task.MatchID, task.Category, task.Title, string(task.Payload), task.Status, task.Assignee, task.CreatedBy, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListTasks(matchID, status string) ([]TaskItem, error) {
	query := `SELECT id, match_id, category, title, payload_json, status, assignee, created_by, created_at, updated_at FROM tasks WHERE match_id=?`
	args := []any{matchID}
	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY updated_at DESC, id DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TaskItem, 0)
	for rows.Next() {
		var item TaskItem
		var payload string
		if err := rows.Scan(&item.ID, &item.MatchID, &item.Category, &item.Title, &payload, &item.Status, &item.Assignee, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Payload = json.RawMessage(payload)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpdateTaskStatus(matchID string, taskID int64, status string, assignee string) error {
	_, err := s.db.Exec(`UPDATE tasks SET status=?, assignee=?, updated_at=? WHERE id=? AND match_id=?`, status, assignee, time.Now().Unix(), taskID, matchID)
	return err
}

func (s *Store) CreateBookmark(item EventBookmark) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO event_bookmarks(match_id,seq,title,note,created_by,created_at) VALUES(?,?,?,?,?,?)`, item.MatchID, item.Seq, item.Title, item.Note, item.CreatedBy, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListBookmarks(matchID string) ([]EventBookmark, error) {
	rows, err := s.db.Query(`SELECT id, match_id, seq, title, note, created_by, created_at FROM event_bookmarks WHERE match_id=? ORDER BY seq DESC, id DESC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EventBookmark, 0)
	for rows.Next() {
		var b EventBookmark
		if err := rows.Scan(&b.ID, &b.MatchID, &b.Seq, &b.Title, &b.Note, &b.CreatedBy, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) ComputeKPI(matchID string) (AnalyticsKPI, error) {
	var k AnalyticsKPI
	var firstTs, lastTs sql.NullInt64
	if err := s.db.QueryRow(`SELECT COUNT(1), MIN(created_at), MAX(created_at) FROM events WHERE match_id=?`, matchID).Scan(&k.TotalEvents, &firstTs, &lastTs); err != nil {
		return k, err
	}
	if k.TotalEvents > 0 && firstTs.Valid && lastTs.Valid && lastTs.Int64 > firstTs.Int64 {
		k.AvgResponseSeconds = float64(lastTs.Int64-firstTs.Int64) / float64(k.TotalEvents)
	}
	var successCount, traceCount int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM events WHERE match_id=? AND event_type='attack_success'`, matchID).Scan(&successCount)
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM events WHERE match_id=? AND payload_json LIKE '%trace_success%'`, matchID).Scan(&traceCount)
	if k.TotalEvents > 0 {
		k.EffectiveAttackRate = float64(successCount) / float64(k.TotalEvents)
		k.TraceSuccessRate = float64(traceCount) / float64(k.TotalEvents)
	}
	var redScore, blueScore int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(score),0) FROM teams WHERE match_id=? AND type='red'`, matchID).Scan(&redScore)
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(score),0) FROM teams WHERE match_id=? AND type='blue'`, matchID).Scan(&blueScore)
	k.NetScoreDiff = redScore - blueScore
	return k, nil
}

func (s *Store) ListScoreTrend(matchID string) ([]TrendPoint, error) {
	rows, err := s.db.Query(`SELECT type, COALESCE(SUM(score),0) FROM teams WHERE match_id=? GROUP BY type`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TrendPoint, 0)
	for rows.Next() {
		var k string
		var v float64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out = append(out, TrendPoint{Key: k, Value: v})
	}
	return out, rows.Err()
}

func (s *Store) ListTrendsByDimension(matchID string) (map[string][]TrendPoint, error) {
	out := map[string][]TrendPoint{
		"team":        {},
		"tactic":      {},
		"round":       {},
		"status":      {},
		"target_city": {},
	}
	teamRows, err := s.db.Query(`SELECT type, COALESCE(SUM(score),0) FROM teams WHERE match_id=? GROUP BY type`, matchID)
	if err != nil {
		return out, err
	}
	for teamRows.Next() {
		var k string
		var v float64
		if err := teamRows.Scan(&k, &v); err == nil {
			out["team"] = append(out["team"], TrendPoint{Key: k, Value: v})
		}
	}
	_ = teamRows.Close()

	tacticRows, err := s.db.Query(`SELECT COALESCE(json_extract(payload_json, '$.attack_type'), ''), COUNT(1) FROM events WHERE match_id=? GROUP BY 1 ORDER BY COUNT(1) DESC LIMIT 20`, matchID)
	if err != nil {
		return out, err
	}
	for tacticRows.Next() {
		var k string
		var v float64
		if err := tacticRows.Scan(&k, &v); err == nil && strings.TrimSpace(k) != "" {
			out["tactic"] = append(out["tactic"], TrendPoint{Key: k, Value: v})
		}
	}
	_ = tacticRows.Close()

	statusRows, err := s.db.Query(`SELECT COALESCE(json_extract(payload_json, '$.status'), ''), COUNT(1) FROM events WHERE match_id=? AND event_type='attack_success' GROUP BY 1 ORDER BY COUNT(1) DESC`, matchID)
	if err != nil {
		return out, err
	}
	for statusRows.Next() {
		var k string
		var v float64
		if err := statusRows.Scan(&k, &v); err == nil && strings.TrimSpace(k) != "" {
			out["status"] = append(out["status"], TrendPoint{Key: k, Value: v})
		}
	}
	_ = statusRows.Close()

	cityRows, err := s.db.Query(`SELECT COALESCE(json_extract(payload_json, '$.target_city'), ''), COUNT(1) FROM events WHERE match_id=? AND event_type='attack_success' GROUP BY 1 ORDER BY COUNT(1) DESC LIMIT 20`, matchID)
	if err != nil {
		return out, err
	}
	for cityRows.Next() {
		var k string
		var v float64
		if err := cityRows.Scan(&k, &v); err == nil && strings.TrimSpace(k) != "" {
			out["target_city"] = append(out["target_city"], TrendPoint{Key: k, Value: v})
		}
	}
	_ = cityRows.Close()

	roundRows, err := s.db.Query(`SELECT ((seq-1)/20)+1 AS round_bucket, COUNT(1) FROM events WHERE match_id=? GROUP BY round_bucket ORDER BY round_bucket ASC`, matchID)
	if err != nil {
		return out, err
	}
	for roundRows.Next() {
		var roundBucket int
		var cnt float64
		if err := roundRows.Scan(&roundBucket, &cnt); err == nil {
			out["round"] = append(out["round"], TrendPoint{Key: fmt.Sprintf("R%d", roundBucket), Value: cnt})
		}
	}
	_ = roundRows.Close()
	return out, nil
}

func (s *Store) CreateAuditLog(item AuditLog) error {
	_, err := s.db.Exec(`INSERT INTO audit_logs(match_id,actor,role,module,action,before_json,after_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, item.MatchID, item.Actor, item.Role, item.Module, item.Action, item.Before, item.After, time.Now().Unix())
	return err
}

func (s *Store) ListAuditLogs(matchID, actor, module string, fromTS, toTS int64) ([]AuditLog, error) {
	query := `SELECT id, match_id, actor, role, module, action, before_json, after_json, created_at FROM audit_logs WHERE match_id=?`
	args := []any{matchID}
	if strings.TrimSpace(actor) != "" {
		query += ` AND actor=?`
		args = append(args, actor)
	}
	if strings.TrimSpace(module) != "" {
		query += ` AND module=?`
		args = append(args, module)
	}
	if fromTS > 0 {
		query += ` AND created_at>=?`
		args = append(args, fromTS)
	}
	if toTS > 0 {
		query += ` AND created_at<=?`
		args = append(args, toTS)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 500`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AuditLog, 0)
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.MatchID, &a.Actor, &a.Role, &a.Module, &a.Action, &a.Before, &a.After, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ListAuditLogsGlobal(actor, module string, fromTS, toTS int64) ([]AuditLog, error) {
	query := `SELECT id, match_id, actor, role, module, action, before_json, after_json, created_at FROM audit_logs WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(actor) != "" {
		query += ` AND actor=?`
		args = append(args, actor)
	}
	if strings.TrimSpace(module) != "" {
		query += ` AND module=?`
		args = append(args, module)
	}
	if fromTS > 0 {
		query += ` AND created_at>=?`
		args = append(args, fromTS)
	}
	if toTS > 0 {
		query += ` AND created_at<=?`
		args = append(args, toTS)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 500`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditLog, 0)
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.MatchID, &a.Actor, &a.Role, &a.Module, &a.Action, &a.Before, &a.After, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAuditLogByID(id int64) (*AuditLog, error) {
	var a AuditLog
	if err := s.db.QueryRow(
		`SELECT id, match_id, actor, role, module, action, before_json, after_json, created_at FROM audit_logs WHERE id=?`,
		id,
	).Scan(&a.ID, &a.MatchID, &a.Actor, &a.Role, &a.Module, &a.Action, &a.Before, &a.After, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListEventsEnhanced(matchID string, fromSeq uint64, filter EventFilter) ([]EventRecord, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 500
	}
	query := `SELECT seq, event_type, payload_json, created_at FROM events WHERE match_id=? AND seq>=?`
	args := []any{matchID, fromSeq}
	if filter.TeamID > 0 {
		query += ` AND payload_json LIKE ?`
		args = append(args, fmt.Sprintf(`%%\"team_id\":%d%%`, filter.TeamID))
	}
	if strings.TrimSpace(filter.AttackType) != "" {
		query += ` AND payload_json LIKE ?`
		args = append(args, "%"+strings.TrimSpace(filter.AttackType)+"%")
	}
	if strings.TrimSpace(filter.Status) != "" {
		query += ` AND payload_json LIKE ?`
		args = append(args, "%"+strings.TrimSpace(filter.Status)+"%")
	}
	if filter.MinScore != nil {
		query += ` AND CAST(json_extract(payload_json,'$.score_change') AS INTEGER) >= ?`
		args = append(args, *filter.MinScore)
	}
	if filter.MaxScore != nil {
		query += ` AND CAST(json_extract(payload_json,'$.score_change') AS INTEGER) <= ?`
		args = append(args, *filter.MaxScore)
	}
	query += ` ORDER BY seq ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EventRecord, 0)
	for rows.Next() {
		var seq uint64
		var et, payload string
		var ts int64
		if err := rows.Scan(&seq, &et, &payload, &ts); err != nil {
			return nil, err
		}
		out = append(out, EventRecord{Seq: seq, EventType: et, PayloadRaw: json.RawMessage(payload), Timestamp: ts})
	}
	return out, rows.Err()
}
