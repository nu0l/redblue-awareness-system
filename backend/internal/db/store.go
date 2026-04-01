package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"redblue-server/internal/protocol"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, errors.New("dbPath is empty")
	}

	// _foreign_keys 让 SQLite 在删除/约束上更符合预期（即使我们目前没有复杂外键）。
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_busy_timeout=5000", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	s := &Store{db: conn}
	if err := s.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS matches (
			id TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL,
			map_type TEXT NOT NULL,
			leaderboard_visible INTEGER NOT NULL,
			panels_json TEXT NOT NULL,
			countdown_end_ts INTEGER NOT NULL DEFAULT 0,
			countdown_broadcast_msg TEXT NOT NULL DEFAULT '',
			countdown_toggle_panel_id TEXT NOT NULL DEFAULT '',
			countdown_toggle_visible INTEGER NOT NULL DEFAULT 1,
			countdown_triggered INTEGER NOT NULL DEFAULT 0,
			screen_title TEXT NOT NULL DEFAULT '实战化红蓝对抗演练指挥中心',
			screen_organizer TEXT NOT NULL DEFAULT '',
			screen_supporter TEXT NOT NULL DEFAULT '',
			screen_credits_visible INTEGER NOT NULL DEFAULT 1,
			initial_map_type TEXT,
			initial_leaderboard_visible INTEGER,
			initial_panels_json TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			logo TEXT NOT NULL,
			members_json TEXT NOT NULL DEFAULT '[]',
			score INTEGER NOT NULL,
			initial_score INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(match_id) REFERENCES matches(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			seq INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			dedupe_key TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			UNIQUE(match_id, seq),
			FOREIGN KEY(match_id) REFERENCES matches(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_events_match_seq ON events(match_id, seq);`,
		`CREATE TABLE IF NOT EXISTS match_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			map_type TEXT NOT NULL,
			cities_json TEXT NOT NULL DEFAULT '[]',
			attack_types_json TEXT NOT NULL DEFAULT '[]',
			audio_config_json TEXT NOT NULL DEFAULT '{}',
			score_rules_json TEXT NOT NULL DEFAULT '{}',
			change_log TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			category TEXT NOT NULL,
			title TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			status TEXT NOT NULL,
			assignee TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_match_status ON tasks(match_id, status, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS event_bookmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			seq INTEGER NOT NULL,
			title TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_match_seq ON event_bookmarks(match_id, seq);`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			actor TEXT NOT NULL,
			role TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			before_json TEXT NOT NULL DEFAULT '',
			after_json TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_match_time ON audit_logs(match_id, created_at DESC);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}

	// 对已有数据库做增量列添加：SQLite 没有简单的 IF NOT EXISTS 支持。
	// 如果列已存在会报错，这里尽可能忽略“重复列”错误。
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN initial_map_type TEXT`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN initial_leaderboard_visible INTEGER`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN initial_panels_json TEXT`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN screen_title TEXT`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN screen_organizer TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN screen_supporter TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN screen_credits_visible INTEGER NOT NULL DEFAULT 1`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN leaderboard_bg_url TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN bgm_url TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN bgm_enabled INTEGER NOT NULL DEFAULT 0`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN success_sfx_url TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN success_sfx_enabled INTEGER NOT NULL DEFAULT 0`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN leaderboard_main_alpha REAL NOT NULL DEFAULT 0.14`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN countdown_end_ts INTEGER NOT NULL DEFAULT 0`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN countdown_broadcast_msg TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN countdown_toggle_panel_id TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN countdown_toggle_visible INTEGER NOT NULL DEFAULT 1`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN countdown_triggered INTEGER NOT NULL DEFAULT 0`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN screen_modules_json TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE matches ADD COLUMN initial_screen_modules_json TEXT NOT NULL DEFAULT ''`)
	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE events ADD COLUMN dedupe_key TEXT NOT NULL DEFAULT ''`)

	// 幂等去重：同一场次同一事件类型下，dedupe_key 非空时必须唯一。
	// SQLite 支持部分索引（WHERE 子句）。
	_, _ = s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_dedupe ON events(match_id, event_type, dedupe_key) WHERE TRIM(dedupe_key) <> ''`)

	defScreen := `{"left_top":"leaderboard","left_bottom":"region_attack_rank","right_top":"battle_logs","right_bottom":"attack_type_pie"}`
	_, _ = s.db.Exec(`UPDATE matches SET screen_modules_json = ? WHERE screen_modules_json IS NULL OR TRIM(screen_modules_json) = ''`, defScreen)
	_, _ = s.db.Exec(`UPDATE matches SET initial_screen_modules_json = screen_modules_json WHERE initial_screen_modules_json IS NULL OR TRIM(initial_screen_modules_json) = ''`)

	// 迁移回填：对于旧数据，如果 initial_* 仍为空，则用当前配置做兜底，避免回放读到空值。
	_, _ = s.db.Exec(`UPDATE matches SET initial_map_type = map_type WHERE initial_map_type IS NULL`)
	_, _ = s.db.Exec(`UPDATE matches SET initial_leaderboard_visible = leaderboard_visible WHERE initial_leaderboard_visible IS NULL`)
	_, _ = s.db.Exec(`UPDATE matches SET initial_panels_json = panels_json WHERE initial_panels_json IS NULL`)

	// 迁移回填：旧数据 screen_title 若为空，则写入默认值。
	_, _ = s.db.Exec(
		`UPDATE matches SET screen_title = '实战化红蓝对抗演练指挥中心' WHERE screen_title IS NULL OR TRIM(screen_title) = ''`,
	)

	_ = s.execIgnoreDuplicateColumn(`ALTER TABLE teams ADD COLUMN members_json TEXT NOT NULL DEFAULT '[]'`)
	return nil
}

func (s *Store) execIgnoreDuplicateColumn(stmt string) error {
	_, err := s.db.Exec(stmt)
	if err == nil {
		return nil
	}
	// 简单匹配：SQLite duplicate column name / column already exists。
	low := strings.ToLower(err.Error())
	if strings.Contains(low, "duplicate column") || strings.Contains(low, "already exists") {
		return nil
	}
	// 其他错误直接返回（便于排查）。
	return err
}

type MatchSummary struct {
	ID                 string `json:"id"`
	CreatedAt          int64  `json:"created_at"`
	MapType            string `json:"map_type"`
	LeaderboardVisible bool   `json:"leaderboard_visible"`
}

func (s *Store) CreateMatch(
	matchID string,
	mapType string,
	leaderboardVisible bool,
	panels map[string]bool,
	screenTitle string,
	screenOrganizer string,
	screenSupporter string,
) error {
	panelsJSON, err := json.Marshal(panels)
	if err != nil {
		return err
	}

	leaderboard := 0
	if leaderboardVisible {
		leaderboard = 1
	}
	if strings.TrimSpace(screenTitle) == "" {
		screenTitle = "实战化红蓝对抗演练指挥中心"
	}
	if strings.TrimSpace(screenOrganizer) == "" {
		screenOrganizer = ""
	}
	if strings.TrimSpace(screenSupporter) == "" {
		screenSupporter = ""
	}

	defScreenMods := `{"left_top":"leaderboard","left_bottom":"region_attack_rank","right_top":"battle_logs","right_bottom":"attack_type_pie"}`
	_, err = s.db.Exec(
		`INSERT INTO matches(
			id, created_at, map_type, leaderboard_visible, panels_json,
			screen_title, screen_organizer, screen_supporter,
			screen_credits_visible,
			initial_map_type, initial_leaderboard_visible, initial_panels_json,
			screen_modules_json, initial_screen_modules_json
		) 
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID,
		time.Now().Unix(),
		mapType,
		leaderboard,
		string(panelsJSON),
		screenTitle,
		screenOrganizer,
		screenSupporter,
		1,
		mapType,
		leaderboard,
		string(panelsJSON),
		defScreenMods,
		defScreenMods,
	)
	return err
}

func parseScreenModulesJSON(raw string) map[string]string {
	def := `{"left_top":"leaderboard","left_bottom":"region_attack_rank","right_top":"battle_logs","right_bottom":"attack_type_pie"}`
	if strings.TrimSpace(raw) == "" {
		var m map[string]string
		_ = json.Unmarshal([]byte(def), &m)
		return m
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		_ = json.Unmarshal([]byte(def), &m)
	}
	return m
}

// GetScreenModules 读取当前场次大屏模块配置（JSON 对象）。
func (s *Store) GetScreenModules(matchID string) (map[string]string, error) {
	var raw string
	err := s.db.QueryRow(`SELECT COALESCE(screen_modules_json, '') FROM matches WHERE id = ?`, matchID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	return parseScreenModulesJSON(raw), nil
}

// GetInitialScreenModules 用于回放基线：优先 initial_screen_modules_json，否则回落到当前 screen_modules。
func (s *Store) GetInitialScreenModules(matchID string) (map[string]string, error) {
	var cur, init string
	err := s.db.QueryRow(
		`SELECT COALESCE(screen_modules_json, ''), COALESCE(initial_screen_modules_json, '') FROM matches WHERE id = ?`,
		matchID,
	).Scan(&cur, &init)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(init) != "" {
		return parseScreenModulesJSON(init), nil
	}
	return parseScreenModulesJSON(cur), nil
}

// UpdateMatchScreenModules 持久化大屏模块配置（应传入已归一化的完整 map）。
func (s *Store) UpdateMatchScreenModules(matchID string, modules map[string]string) error {
	if modules == nil {
		modules = map[string]string{}
	}
	b, err := json.Marshal(modules)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE matches SET screen_modules_json = ? WHERE id = ?`, string(b), matchID)
	return err
}

func (s *Store) ListMatches() ([]MatchSummary, error) {
	rows, err := s.db.Query(`SELECT id, created_at, map_type, leaderboard_visible FROM matches ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MatchSummary
	for rows.Next() {
		var (
			id      string
			ca      int64
			mapType string
			lvInt   int
		)
		if err := rows.Scan(&id, &ca, &mapType, &lvInt); err != nil {
			return nil, err
		}
		out = append(out, MatchSummary{
			ID:                 id,
			CreatedAt:          ca,
			MapType:            mapType,
			LeaderboardVisible: lvInt == 1,
		})
	}
	return out, rows.Err()
}

func (s *Store) GetMatchPanels(
	matchID string,
) (mapType string, leaderboardVisible bool, panels map[string]bool, countdownEndTS int64, countdownBroadcastMsg string, countdownTogglePanelID string, countdownToggleVisible bool, countdownTriggered bool, screenTitle string, screenOrganizer string, screenSupporter string, screenCreditsVisible bool, leaderboardBGURL string, bgmURL string, bgmEnabled bool, successSFXURL string, successSFXEnabled bool, leaderboardMainAlpha float64, err error) {
	var (
		mapTypeDB              string
		leaderboardInt         int
		panelsJSON             string
		countdownEndTSDB       int64
		countdownBroadcastMsgDB   string
		countdownTogglePanelIDDB  string
		countdownToggleVisibleInt int
		countdownTriggeredInt     int
		screenTitleDB          string
		screenOrganizerDB      string
		screenSupporterDB      string
		screenCreditsVisibleInt int
		leaderboardBGDB        string
		bgmURLDB               string
		bgmEnabledInt          int
		successSFXURLDB        string
		successSFXEnabledInt   int
		leaderboardMainAlphaDB float64
	)
	err = s.db.QueryRow(
		`SELECT map_type, leaderboard_visible, panels_json,
			COALESCE(countdown_end_ts, 0),
			COALESCE(countdown_broadcast_msg, ''),
			COALESCE(countdown_toggle_panel_id, ''),
			COALESCE(countdown_toggle_visible, 1),
			COALESCE(countdown_triggered, 0),
			screen_title, screen_organizer, screen_supporter,
			COALESCE(screen_credits_visible, 1),
			COALESCE(leaderboard_bg_url, ''), COALESCE(bgm_url, ''), COALESCE(bgm_enabled, 0),
			COALESCE(success_sfx_url, ''), COALESCE(success_sfx_enabled, 0),
			COALESCE(leaderboard_main_alpha, 0.14)
		  FROM matches WHERE id = ?`,
		matchID,
	).Scan(
		&mapTypeDB,
		&leaderboardInt,
		&panelsJSON,
		&countdownEndTSDB,
		&countdownBroadcastMsgDB,
		&countdownTogglePanelIDDB,
		&countdownToggleVisibleInt,
		&countdownTriggeredInt,
		&screenTitleDB,
		&screenOrganizerDB,
		&screenSupporterDB,
		&screenCreditsVisibleInt,
		&leaderboardBGDB,
		&bgmURLDB,
		&bgmEnabledInt,
		&successSFXURLDB,
		&successSFXEnabledInt,
		&leaderboardMainAlphaDB,
	)
	if err != nil {
		return "", false, nil, 0, "", "", false, false, "", "", "", true, "", "", false, "", false, 0.14, err
	}

	panels = make(map[string]bool)
	if panelsJSON != "" {
		if err := json.Unmarshal([]byte(panelsJSON), &panels); err != nil {
			return "", false, nil, 0, "", "", false, false, "", "", "", true, "", "", false, "", false, 0.14, err
		}
	}

	return mapTypeDB,
		leaderboardInt == 1,
		panels,
		countdownEndTSDB,
		countdownBroadcastMsgDB,
		countdownTogglePanelIDDB,
		countdownToggleVisibleInt == 1,
		countdownTriggeredInt == 1,
		screenTitleDB,
		screenOrganizerDB,
		screenSupporterDB,
		screenCreditsVisibleInt == 1,
		leaderboardBGDB,
		bgmURLDB,
		bgmEnabledInt == 1,
		successSFXURLDB,
		successSFXEnabledInt == 1,
		leaderboardMainAlphaDB,
		nil
}

func (s *Store) UpdateMatchConfig(matchID string, mapType string, leaderboardVisible bool, panels map[string]bool) error {
	panelsJSON, err := json.Marshal(panels)
	if err != nil {
		return err
	}
	lvInt := 0
	if leaderboardVisible {
		lvInt = 1
	}
	_, err = s.db.Exec(
		`UPDATE matches SET map_type = ?, leaderboard_visible = ?, panels_json = ? WHERE id = ?`,
		mapType, lvInt, string(panelsJSON), matchID,
	)
	return err
}

func (s *Store) UpdateMatchScreenTitle(matchID string, title string) error {
	if strings.TrimSpace(title) == "" {
		title = "实战化红蓝对抗演练指挥中心"
	}
	_, err := s.db.Exec(`UPDATE matches SET screen_title = ? WHERE id = ?`, title, matchID)
	return err
}

func (s *Store) UpdateMatchScreenCredits(matchID string, organizer string, supporter string) error {
	if strings.TrimSpace(organizer) == "" {
		organizer = ""
	}
	if strings.TrimSpace(supporter) == "" {
		supporter = ""
	}
	_, err := s.db.Exec(
		`UPDATE matches SET screen_organizer = ?, screen_supporter = ? WHERE id = ?`,
		organizer,
		supporter,
		matchID,
	)
	return err
}

func (s *Store) UpdateMatchScreenCreditsVisible(matchID string, visible bool) error {
	v := 0
	if visible {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE matches SET screen_credits_visible = ? WHERE id = ?`, v, matchID)
	return err
}

// UpdateMatchLeaderboardBG 设置得分总榜背景图 URL 路径（相对站点根，如 /uploads/xxx/leaderboard-bg.png）；空字符串表示使用前端默认图。
func (s *Store) UpdateMatchLeaderboardBG(matchID string, url string) error {
	_, err := s.db.Exec(`UPDATE matches SET leaderboard_bg_url = ? WHERE id = ?`, strings.TrimSpace(url), matchID)
	return err
}

func (s *Store) UpdateMatchAudioConfig(matchID string, bgmURL string, bgmEnabled bool, successSFXURL string, successSFXEnabled bool) error {
	bgmEnabledInt := 0
	if bgmEnabled {
		bgmEnabledInt = 1
	}
	successSFXEnabledInt := 0
	if successSFXEnabled {
		successSFXEnabledInt = 1
	}
	_, err := s.db.Exec(
		`UPDATE matches SET bgm_url = ?, bgm_enabled = ?, success_sfx_url = ?, success_sfx_enabled = ? WHERE id = ?`,
		strings.TrimSpace(bgmURL),
		bgmEnabledInt,
		strings.TrimSpace(successSFXURL),
		successSFXEnabledInt,
		matchID,
	)
	return err
}

func (s *Store) UpdateMatchLeaderboardMainAlpha(matchID string, alpha float64) error {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	_, err := s.db.Exec(`UPDATE matches SET leaderboard_main_alpha = ? WHERE id = ?`, alpha, matchID)
	return err
}

func (s *Store) UpdateMatchCountdownConfig(
	matchID string,
	countdownEndTS int64,
	broadcastMsg string,
	togglePanelID string,
	toggleVisible bool,
) error {
	if countdownEndTS < 0 {
		countdownEndTS = 0
	}
	triggeredInt := 0
	// 只要配置被更新（包括清空倒计时），就重置“是否已触发”
	// 后续定时触发会再决定是否触发。
	_, err := s.db.Exec(
		`UPDATE matches
		 SET countdown_end_ts = ?,
		     countdown_broadcast_msg = ?,
		     countdown_toggle_panel_id = ?,
		     countdown_toggle_visible = ?,
		     countdown_triggered = ?
		 WHERE id = ?`,
		countdownEndTS,
		strings.TrimSpace(broadcastMsg),
		strings.TrimSpace(togglePanelID),
		func() int { if toggleVisible { return 1 }; return 0 }(),
		triggeredInt,
		matchID,
	)
	return err
}

func (s *Store) TryTriggerCountdown(matchID string, endTS int64) (bool, error) {
	if endTS <= 0 {
		return false, nil
	}
	res, err := s.db.Exec(
		`UPDATE matches SET countdown_triggered = 1 WHERE id=? AND countdown_end_ts=? AND countdown_triggered=0`,
		matchID,
		endTS,
	)
	if err != nil {
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return ra > 0, nil
}

func (s *Store) ResetAll() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 利用外键级联删除 teams/events（删除 matches 即可）。
	if _, err := tx.Exec(`DELETE FROM matches`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResetMatch(matchID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 通过外键级联删除该 match 下所有关联数据。
	if _, err := tx.Exec(`DELETE FROM matches WHERE id = ?`, matchID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetMatchInitialConfig(
	matchID string,
) (mapType string, leaderboardVisible bool, panels map[string]bool, countdownEndTS int64, countdownBroadcastMsg string, countdownTogglePanelID string, countdownToggleVisible bool, countdownTriggered bool, screenTitle string, screenOrganizer string, screenSupporter string, screenCreditsVisible bool, leaderboardBGURL string, bgmURL string, bgmEnabled bool, successSFXURL string, successSFXEnabled bool, leaderboardMainAlpha float64, err error) {
	var (
		mapTypeDB              string
		lvInt                  int
		panelsJSON             string
		countdownEndTSDB       int64
		countdownBroadcastMsgDB   string
		countdownTogglePanelIDDB  string
		countdownToggleVisibleInt int
		countdownTriggeredInt     int
		screenCreditsVisibleInt   int
		leaderboardBGDB        string
		bgmURLDB               string
		bgmEnabledInt          int
		successSFXURLDB        string
		successSFXEnabledInt   int
		leaderboardMainAlphaDB float64
	)
	row := s.db.QueryRow(
		`SELECT 
			COALESCE(initial_map_type, map_type) as map_type,
			COALESCE(initial_leaderboard_visible, leaderboard_visible) as leaderboard_visible,
			COALESCE(initial_panels_json, panels_json) as panels_json,
			COALESCE(countdown_end_ts, 0) as countdown_end_ts,
			COALESCE(countdown_broadcast_msg, '') as countdown_broadcast_msg,
			COALESCE(countdown_toggle_panel_id, '') as countdown_toggle_panel_id,
			COALESCE(countdown_toggle_visible, 1) as countdown_toggle_visible,
			COALESCE(countdown_triggered, 0) as countdown_triggered,
			COALESCE(screen_title, '实战化红蓝对抗演练指挥中心') as screen_title,
			COALESCE(screen_organizer, '') as screen_organizer,
			COALESCE(screen_supporter, '') as screen_supporter,
			COALESCE(screen_credits_visible, 1) as screen_credits_visible,
			COALESCE(leaderboard_bg_url, '') as leaderboard_bg_url,
			COALESCE(bgm_url, '') as bgm_url,
			COALESCE(bgm_enabled, 0) as bgm_enabled,
			COALESCE(success_sfx_url, '') as success_sfx_url,
			COALESCE(success_sfx_enabled, 0) as success_sfx_enabled,
			COALESCE(leaderboard_main_alpha, 0.14) as leaderboard_main_alpha
		  FROM matches WHERE id = ?`,
		matchID,
	)
	err = row.Scan(&mapTypeDB, &lvInt, &panelsJSON, &countdownEndTSDB, &countdownBroadcastMsgDB, &countdownTogglePanelIDDB, &countdownToggleVisibleInt, &countdownTriggeredInt, &screenTitle, &screenOrganizer, &screenSupporter, &screenCreditsVisibleInt, &leaderboardBGDB, &bgmURLDB, &bgmEnabledInt, &successSFXURLDB, &successSFXEnabledInt, &leaderboardMainAlphaDB)
	if err != nil {
		return "", false, nil, 0, "", "", false, false, "", "", "", true, "", "", false, "", false, 0.14, err
	}

	panels = make(map[string]bool)
	if panelsJSON != "" {
		if err := json.Unmarshal([]byte(panelsJSON), &panels); err != nil {
			return "", false, nil, 0, "", "", false, false, "", "", "", true, "", "", false, "", false, 0.14, err
		}
	}
	return mapTypeDB,
		lvInt == 1,
		panels,
		countdownEndTSDB,
		countdownBroadcastMsgDB,
		countdownTogglePanelIDDB,
		countdownToggleVisibleInt == 1,
		countdownTriggeredInt == 1,
		screenTitle,
		screenOrganizer,
		screenSupporter,
		screenCreditsVisibleInt == 1,
		leaderboardBGDB,
		bgmURLDB,
		bgmEnabledInt == 1,
		successSFXURLDB,
		successSFXEnabledInt == 1,
		leaderboardMainAlphaDB,
		nil
}

func (s *Store) ListTeams(matchID string) ([]protocol.TeamDTO, error) {
	rows, err := s.db.Query(
		`SELECT id, name, type, logo, members_json, score, initial_score FROM teams WHERE match_id = ? ORDER BY id ASC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []protocol.TeamDTO
	for rows.Next() {
		var t protocol.TeamDTO
		var membersJSON string
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.Logo, &membersJSON, &t.Score, &t.InitialScore); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(membersJSON), &t.Members)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTeam(matchID string, team protocol.TeamDTO) (int, error) {
	// score/initial_score 由上层传入，保证 reset 时可用（目前 reset 先不做）。
	if strings.TrimSpace(team.Logo) == "" {
		team.Logo = "?"
	}
	membersJSON, err := json.Marshal(team.Members)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(
		`INSERT INTO teams(match_id, name, type, logo, members_json, score, initial_score, created_at) 
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, team.Name, team.Type, team.Logo, string(membersJSON), team.Score, team.Score, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
}

func (s *Store) UpdateTeam(matchID string, teamID int, team protocol.TeamDTO) error {
	if strings.TrimSpace(team.Logo) == "" {
		team.Logo = "?"
	}
	membersJSON, err := json.Marshal(team.Members)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE teams SET name = ?, type = ?, logo = ?, members_json = ?, score = ?, initial_score = ? 
		 WHERE match_id = ? AND id = ?`,
		team.Name, team.Type, team.Logo, string(membersJSON), team.Score, team.Score, matchID, teamID,
	)
	return err
}

// UpdateTeamScore 只更新“当前比分”，不修改 initial_score（用于后续 reset/回放基准）。
func (s *Store) UpdateTeamScore(matchID string, teamID int, score int) error {
	_, err := s.db.Exec(
		`UPDATE teams SET score = ? WHERE match_id = ? AND id = ?`,
		score, matchID, teamID,
	)
	return err
}

func (s *Store) DeleteTeam(matchID string, teamID int) error {
	_, err := s.db.Exec(`DELETE FROM teams WHERE match_id = ? AND id = ?`, matchID, teamID)
	return err
}

func (s *Store) GetLastSeq(matchID string) (uint64, error) {
	var last int64
	err := s.db.QueryRow(`SELECT IFNULL(MAX(seq), 0) FROM events WHERE match_id = ?`, matchID).Scan(&last)
	if err != nil {
		return 0, err
	}
	return uint64(last), nil
}

type EventRecord struct {
	Seq        uint64          `json:"seq"`
	EventType  string          `json:"event_type"`
	PayloadRaw json.RawMessage `json:"payload_raw"`
	Timestamp  int64           `json:"timestamp"`
}

func (s *Store) InsertEvent(matchID string, seq uint64, eventType string, payload any, dedupeKey string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO events(match_id, seq, event_type, dedupe_key, payload_json, created_at) VALUES(?, ?, ?, ?, ?, ?)`,
		matchID, seq, eventType, strings.TrimSpace(dedupeKey), string(payloadJSON), time.Now().Unix(),
	)
	return err
}

func (s *Store) HasDedupeEvent(matchID string, eventType string, dedupeKey string) (bool, error) {
	key := strings.TrimSpace(dedupeKey)
	if key == "" {
		return false, nil
	}
	var cnt int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM events WHERE match_id = ? AND event_type = ? AND TRIM(dedupe_key) = ?`,
		matchID,
		strings.TrimSpace(eventType),
		key,
	).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (s *Store) ListEvents(matchID string, fromSeq uint64, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT seq, event_type, payload_json, created_at FROM events WHERE match_id = ? AND seq >= ? ORDER BY seq ASC LIMIT ?`,
		matchID, fromSeq, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventRecord
	for rows.Next() {
		var (
			seq        uint64
			eventType  string
			payloadStr string
			ts         int64
		)
		if err := rows.Scan(&seq, &eventType, &payloadStr, &ts); err != nil {
			return nil, err
		}
		out = append(out, EventRecord{
			Seq:        seq,
			EventType:  eventType,
			PayloadRaw: json.RawMessage(payloadStr),
			Timestamp:  ts,
		})
	}
	return out, rows.Err()
}
