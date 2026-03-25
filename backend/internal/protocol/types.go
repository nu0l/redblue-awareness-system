package protocol

import "encoding/json"

// WSMessage 是大屏/回放端与后端之间的统一消息协议。
// 说明：目前后端只推送，客户端无需回传业务数据（但会维持连接/心跳）。
type WSMessage struct {
	Type      string          `json:"type"` // sync_state | event
	MatchID   string          `json:"match_id"`
	Seq       uint64          `json:"seq,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Event     string          `json:"event,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	State     *MatchStateDTO `json:"state,omitempty"`
}

type TeamDTO struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"` // red | blue
	Score        int      `json:"score"`
	InitialScore int      `json:"initial_score,omitempty"`
	Logo         string   `json:"logo,omitempty"`
	Members      []string `json:"members,omitempty"`
}

type AttackStatDTO struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type MatchStateDTO struct {
	MapType            string            `json:"map_type"`
	LeaderboardVisible bool              `json:"leaderboard_visible"`
	Teams              []TeamDTO         `json:"teams"`
	AttackStats        []AttackStatDTO   `json:"attack_stats"`
	Panels             map[string]bool   `json:"panels"`
	ScreenTitle        string            `json:"screen_title"`
	ScreenOrganizer    string            `json:"screen_organizer"`
	ScreenSupporter    string            `json:"screen_supporter"`
	BGMURL             string            `json:"bgm_url"`
	BGMEnabled         bool              `json:"bgm_enabled"`
	SuccessSFXURL      string            `json:"success_sfx_url"`
	SuccessSFXEnabled  bool              `json:"success_sfx_enabled"`
	LeaderboardMainAlpha float64         `json:"leaderboard_main_alpha"`
	// LeaderboardBGURL 为后端托管的得分总榜背景图路径，如 /uploads/{match_id}/leaderboard-bg.png；空则前端用默认图。
	// 不使用 omitempty，避免 WebSocket 同步时前端 Object.assign 无法清空旧值。
	LeaderboardBGURL string `json:"leaderboard_bg_url"`
}

