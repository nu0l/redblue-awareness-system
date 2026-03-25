package match

import (
	"encoding/json"
	"errors"
	"math"
	"sync"
	"strings"
	"time"

	"redblue-server/internal/db"
	"redblue-server/internal/protocol"
	"redblue-server/internal/ws"
)

type MatchRuntime struct {
	ID string

	mu sync.Mutex

	MapType           string
	LeaderboardVisible bool
	Panels            map[string]bool
	CountdownEndTS    int64
	CountdownBroadcastMsg   string
	CountdownTogglePanelID  string
	CountdownToggleVisible  bool
	CountdownTriggered      bool
	ScreenTitle       string
	ScreenOrganizer   string
	ScreenSupporter   string
	LeaderboardBGURL  string
	BGMURL            string
	BGMEnabled        bool
	SuccessSFXURL     string
	SuccessSFXEnabled bool
	LeaderboardMainAlpha float64

	Teams       []protocol.TeamDTO
	AttackStats map[string]int // attack_type -> count

	NextSeq uint64
}

type Service struct {
	store *db.Store
	hub   *ws.Hub

	// matches runtime cache: 用于减少频繁从 DB 拉取的开销。
	cacheMu sync.Mutex
	cache   map[string]*MatchRuntime

	countdownMu     sync.Mutex
	countdownTimers map[string]*time.Timer
}

func NewService(store *db.Store, hub *ws.Hub) *Service {
	return &Service{
		store: store,
		hub:   hub,
		cache: make(map[string]*MatchRuntime),
		countdownTimers: make(map[string]*time.Timer),
	}
}

// InvalidateCache 清除场次运行态缓存（如上传得分总榜背景后需重新从 DB 加载）。
func (s *Service) InvalidateCache(matchID string) {
	s.cacheMu.Lock()
	delete(s.cache, matchID)
	s.cacheMu.Unlock()
}

func (s *Service) cancelCountdownTimer(matchID string) {
	s.countdownMu.Lock()
	if t, ok := s.countdownTimers[matchID]; ok && t != nil {
		t.Stop()
		delete(s.countdownTimers, matchID)
	}
	s.countdownMu.Unlock()
}

// jsonStringEscape 返回 JSON 字符串片段（已包含引号），用于拼接 RawMessage。
func jsonStringEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (s *Service) ensureCountdownTimer(matchID string, endTS int64, broadcastMsg string, togglePanelID string, toggleVisible bool, triggered bool) {
	if endTS <= 0 || triggered {
		s.cancelCountdownTimer(matchID)
		return
	}
	delay := time.Until(time.Unix(endTS, 0))
	if delay < 0 {
		delay = 0
	}

	s.cancelCountdownTimer(matchID)
	timer := time.AfterFunc(delay, func() {
		// CAS：只有“仍然是这个 endTS 且未触发”的配置才能真正触发。
		ok, err := s.store.TryTriggerCountdown(matchID, endTS)
		if err != nil || !ok {
			return
		}

		beforeState, _ := s.GetStateDTO(matchID)
		beforeJSON := ""
		if beforeState != nil {
			if b, e := json.Marshal(beforeState); e == nil {
				beforeJSON = string(b)
			}
		}

		// 1) 广播通知
		if strings.TrimSpace(broadcastMsg) != "" {
			wsMsg, err := s.ApplyCommand(matchID, CmdMessage{
				EventType: "system_broadcast",
				Data:      json.RawMessage(`{"message":` + jsonStringEscape(broadcastMsg) + `}`),
			})
			if err == nil && wsMsg != nil && s.hub != nil {
				s.hub.Broadcast(matchID, *wsMsg)
				afterJSON := ""
				if wsMsg.State != nil {
					if b, e := json.Marshal(wsMsg.State); e == nil {
						afterJSON = string(b)
					}
				}
				_ = s.store.CreateAuditLog(db.AuditLog{
					MatchID: matchID,
					Actor:   "system",
					Role:    "system",
					Module:  "countdown",
					Action:  "trigger_system_broadcast",
					Before:  beforeJSON,
					After:   afterJSON,
				})
			}
		}

		// 2) 切换面板
		if strings.TrimSpace(togglePanelID) != "" {
			beforeState2, _ := s.GetStateDTO(matchID)
			beforeJSON2 := beforeJSON
			if beforeState2 != nil {
				if b, e := json.Marshal(beforeState2); e == nil {
					beforeJSON2 = string(b)
				}
			}

			wsMsg, err := s.ApplyCommand(matchID, CmdMessage{
				EventType: "toggle_panel",
				Data: func() json.RawMessage {
					b, _ := json.Marshal(map[string]any{"panel_id": togglePanelID, "visible": toggleVisible})
					return json.RawMessage(b)
				}(),
			})
			if err == nil && wsMsg != nil && s.hub != nil {
				s.hub.Broadcast(matchID, *wsMsg)
				afterJSON := ""
				if wsMsg.State != nil {
					if b, e := json.Marshal(wsMsg.State); e == nil {
						afterJSON = string(b)
					}
				}
				_ = s.store.CreateAuditLog(db.AuditLog{
					MatchID: matchID,
					Actor:   "system",
					Role:    "system",
					Module:  "countdown",
					Action:  "trigger_toggle_panel",
					Before:  beforeJSON2,
					After:   afterJSON,
				})
			}
		}
	})

	s.countdownMu.Lock()
	s.countdownTimers[matchID] = timer
	s.countdownMu.Unlock()
}

func (s *Service) LoadRuntime(matchID string) (*MatchRuntime, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if rt, ok := s.cache[matchID]; ok {
		return rt, nil
	}

	mapType,
		leaderboardVisible,
		panels,
		countdownEndTS,
		countdownBroadcastMsg,
		countdownTogglePanelID,
		countdownToggleVisible,
		countdownTriggered,
		screenTitle,
		screenOrganizer,
		screenSupporter,
		leaderboardBG,
		bgmURL,
		bgmEnabled,
		successSFXURL,
		successSFXEnabled,
		leaderboardMainAlpha,
		err := s.store.GetMatchPanels(matchID)
	if err != nil {
		return nil, err
	}
	teams, err := s.store.ListTeams(matchID)
	if err != nil {
		return nil, err
	}
	lastSeq, err := s.store.GetLastSeq(matchID)
	if err != nil {
		return nil, err
	}

	// 为了支持“历史复盘”和“启动后恢复一致态”，需要从历史事件重建 attack_stats。
	evs, err := s.store.ListEvents(matchID, 1, 1000000)
	if err != nil {
		return nil, err
	}
	attackStats := make(map[string]int)
	for _, ev := range evs {
		if ev.EventType != "attack_success" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(ev.PayloadRaw, &payload); err != nil {
			continue
		}
		at, _ := payload["attack_type"].(string)
		if at != "" {
			attackStats[at]++
		}
	}

	rt := &MatchRuntime{
		ID:                 matchID,
		MapType:            mapType,
		LeaderboardVisible: leaderboardVisible,
		Panels:             panels,
		CountdownEndTS:     countdownEndTS,
		CountdownBroadcastMsg:  countdownBroadcastMsg,
		CountdownTogglePanelID: countdownTogglePanelID,
		CountdownToggleVisible: countdownToggleVisible,
		CountdownTriggered:     countdownTriggered,
		ScreenTitle:        screenTitle,
		ScreenOrganizer:    screenOrganizer,
		ScreenSupporter:    screenSupporter,
		LeaderboardBGURL:   leaderboardBG,
		BGMURL:             bgmURL,
		BGMEnabled:         bgmEnabled,
		SuccessSFXURL:      successSFXURL,
		SuccessSFXEnabled:  successSFXEnabled,
		LeaderboardMainAlpha: leaderboardMainAlpha,
		Teams:              teams,
		AttackStats:        attackStats,
		NextSeq:            lastSeq + 1,
	}
	if rt.Panels == nil {
		rt.Panels = map[string]bool{
			"panel-leaderboard": rt.LeaderboardVisible,
		}
	}

	s.ensureCountdownTimer(
		matchID,
		rt.CountdownEndTS,
		rt.CountdownBroadcastMsg,
		rt.CountdownTogglePanelID,
		rt.CountdownToggleVisible,
		rt.CountdownTriggered,
	)
	return rt, nil
}

func (s *Service) GetStateDTO(matchID string) (*protocol.MatchStateDTO, error) {
	rt, err := s.LoadRuntime(matchID)
	if err != nil {
		return nil, err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return runtimeToStateDTO(rt), nil
}

	// GetInitialStateDTO 用于历史复盘：从 match 创建时的“初始配置 + 初始比分”开始，
	// 再按事件序列回放，从而保证 scoreboard 可复现。
func (s *Service) GetInitialStateDTO(matchID string) (*protocol.MatchStateDTO, error) {
	initMapType,
		initLeaderboardVisible,
		initPanels,
		initCountdownEndTS,
		initCountdownBroadcastMsg,
		initCountdownTogglePanelID,
		initCountdownToggleVisible,
		initCountdownTriggered,
		screenTitle,
		initOrganizer,
		initSupporter,
		initLeaderboardBG,
		initBGMURL,
		initBGMEnabled,
		initSuccessSFXURL,
		initSuccessSFXEnabled,
		initLeaderboardMainAlpha,
		err := s.store.GetMatchInitialConfig(matchID)
	if err != nil {
		return nil, err
	}
	teams, err := s.store.ListTeams(matchID)
	if err != nil {
		return nil, err
	}

	// 初始化比分：用初始分作为 replay 基线。
	for i := range teams {
		teams[i].Score = teams[i].InitialScore
	}

	// 保底 panels
	if initPanels == nil {
		initPanels = make(map[string]bool)
	}
	if _, ok := initPanels["panel-leaderboard"]; !ok {
		initPanels["panel-leaderboard"] = initLeaderboardVisible
	}

	return &protocol.MatchStateDTO{
		MapType:            initMapType,
		LeaderboardVisible: initLeaderboardVisible,
		Teams:              teams,
		AttackStats:        []protocol.AttackStatDTO{},
		Panels:             initPanels,
		CountdownEndTS:     initCountdownEndTS,
		CountdownBroadcastMsg:  initCountdownBroadcastMsg,
		CountdownTogglePanelID: initCountdownTogglePanelID,
		CountdownToggleVisible: initCountdownToggleVisible,
		CountdownTriggered:     initCountdownTriggered,
		ScreenTitle:        screenTitle,
		ScreenOrganizer:    initOrganizer,
		ScreenSupporter:    initSupporter,
		LeaderboardBGURL:   initLeaderboardBG,
		BGMURL:             initBGMURL,
		BGMEnabled:         initBGMEnabled,
		SuccessSFXURL:      initSuccessSFXURL,
		SuccessSFXEnabled:  initSuccessSFXEnabled,
		LeaderboardMainAlpha: initLeaderboardMainAlpha,
	}, nil
}

func runtimeToStateDTO(rt *MatchRuntime) *protocol.MatchStateDTO {
	attackStats := make([]protocol.AttackStatDTO, 0, len(rt.AttackStats))
	for name, val := range rt.AttackStats {
		attackStats = append(attackStats, protocol.AttackStatDTO{Name: name, Value: val})
	}
	// 让前端展示更稳定
	// （这里避免依赖前端排序逻辑）
	for i := 0; i < len(attackStats); i++ {
		for j := i + 1; j < len(attackStats); j++ {
			if attackStats[j].Value > attackStats[i].Value {
				attackStats[i], attackStats[j] = attackStats[j], attackStats[i]
			}
		}
	}

	panels := make(map[string]bool, len(rt.Panels))
	for k, v := range rt.Panels {
		panels[k] = v
	}

	return &protocol.MatchStateDTO{
		MapType:            rt.MapType,
		LeaderboardVisible: rt.LeaderboardVisible,
		Teams:              rt.Teams,
		AttackStats:        attackStats,
		Panels:             panels,
		CountdownEndTS:     rt.CountdownEndTS,
		CountdownBroadcastMsg:  rt.CountdownBroadcastMsg,
		CountdownTogglePanelID: rt.CountdownTogglePanelID,
		CountdownToggleVisible: rt.CountdownToggleVisible,
		CountdownTriggered:     rt.CountdownTriggered,
		ScreenTitle:        rt.ScreenTitle,
		ScreenOrganizer:    rt.ScreenOrganizer,
		ScreenSupporter:    rt.ScreenSupporter,
		LeaderboardBGURL:   rt.LeaderboardBGURL,
		BGMURL:             rt.BGMURL,
		BGMEnabled:         rt.BGMEnabled,
		SuccessSFXURL:      rt.SuccessSFXURL,
		SuccessSFXEnabled:  rt.SuccessSFXEnabled,
		LeaderboardMainAlpha: rt.LeaderboardMainAlpha,
	}
}

// CmdMessage 是 HTTP command 的统一输入格式。
type CmdMessage struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

func (s *Service) ApplyCommand(matchID string, cmd CmdMessage) (*protocol.WSMessage, error) {
	if cmd.EventType == "" {
		return nil, errors.New("event_type is required")
	}
	rt, err := s.LoadRuntime(matchID)
	if err != nil {
		return nil, err
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	seq := rt.NextSeq
	rt.NextSeq++

	var nowTs = time.Now().Unix()

	// 默认：对 state 不做改动时，仍会广播一条事件，便于前端更新日志/播报。
	// 同步对象在最后统一拿 rt => dto。
	switch cmd.EventType {
	case "attack_success":
		var payload struct {
			SourceCity  string `json:"source_city"`
			TargetCity  string `json:"target_city"`
			TeamID      int    `json:"team_id"`
			AttackType  string `json:"attack_type"`
			ScoreChange int    `json:"score_change"`
			Message     string `json:"message"`
			Status      string `json:"status"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq-- // 回滚 seq
			return nil, err
		}

		// 更新红队（发起方）比分
		var attacker *protocol.TeamDTO
		for i := range rt.Teams {
			if rt.Teams[i].ID == payload.TeamID {
				attacker = &rt.Teams[i]
				break
			}
		}
		if attacker == nil {
			rt.NextSeq-- // 回滚 seq
			return nil, errors.New("team_id not found")
		}
		attacker.Score += payload.ScoreChange
		if err := s.store.UpdateTeamScore(rt.ID, attacker.ID, attacker.Score); err != nil {
			rt.NextSeq--
			return nil, err
		}

		// 模拟蓝队响应扣减（保持与旧版前端行为一致）
		// 说明：目前策略是扣减所有蓝队的一半，取整。
		// 后续可改成“具体被攻击单位归属”的精细规则。
		deltaBlue := int(math.Floor(float64(payload.ScoreChange) / 2.0))
		if deltaBlue != 0 {
			for i := range rt.Teams {
				if rt.Teams[i].Type == "blue" {
					rt.Teams[i].Score -= deltaBlue
					_ = s.store.UpdateTeamScore(rt.ID, rt.Teams[i].ID, rt.Teams[i].Score)
				}
			}
		}

		// 统计 attack_type 命中次数
		if payload.AttackType != "" {
			rt.AttackStats[payload.AttackType]++
		}

	case "manual_score":
		var payload struct {
			TeamID      int    `json:"team_id"`
			ScoreChange int    `json:"score_change"`
			Message     string `json:"message"`
			Reason      string `json:"reason"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		var t *protocol.TeamDTO
		for i := range rt.Teams {
			if rt.Teams[i].ID == payload.TeamID {
				t = &rt.Teams[i]
				break
			}
		}
		if t == nil {
			rt.NextSeq--
			return nil, errors.New("team_id not found")
		}
		t.Score += payload.ScoreChange
		if err := s.store.UpdateTeamScore(rt.ID, t.ID, t.Score); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "system_broadcast":
		// 不改动 state，仅做事件日志/播报

	case "switch_map":
		var payload struct {
			MapType string `json:"map_type"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		if payload.MapType != "china" && payload.MapType != "taizhou" {
			rt.NextSeq--
			return nil, errors.New("invalid map_type")
		}
		rt.MapType = payload.MapType
		if err := s.storeUpdateMatchConfig(rt); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "set_screen_title":
		// 设置单场次大屏标题（用于屏幕端标题栏显示）
		var payload struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		rt.ScreenTitle = payload.Title
		if err := s.store.UpdateMatchScreenTitle(rt.ID, payload.Title); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "set_screen_credits":
		// 设置单场次大屏主办方/支撑方（用于屏幕端底部展示）
		var payload struct {
			Organizer string `json:"organizer"`
			Supporter string `json:"supporter"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		rt.ScreenOrganizer = payload.Organizer
		rt.ScreenSupporter = payload.Supporter
		if err := s.store.UpdateMatchScreenCredits(rt.ID, payload.Organizer, payload.Supporter); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "set_audio_config":
		var payload struct {
			BGMURL            string `json:"bgm_url"`
			BGMEnabled        bool   `json:"bgm_enabled"`
			SuccessSFXURL     string `json:"success_sfx_url"`
			SuccessSFXEnabled bool   `json:"success_sfx_enabled"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		rt.BGMURL = payload.BGMURL
		rt.BGMEnabled = payload.BGMEnabled
		rt.SuccessSFXURL = payload.SuccessSFXURL
		rt.SuccessSFXEnabled = payload.SuccessSFXEnabled
		if err := s.store.UpdateMatchAudioConfig(rt.ID, payload.BGMURL, payload.BGMEnabled, payload.SuccessSFXURL, payload.SuccessSFXEnabled); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "set_leaderboard_style":
		var payload struct {
			MainAlpha float64 `json:"main_alpha"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		if payload.MainAlpha < 0 {
			payload.MainAlpha = 0
		}
		if payload.MainAlpha > 1 {
			payload.MainAlpha = 1
		}
		rt.LeaderboardMainAlpha = payload.MainAlpha
		if err := s.store.UpdateMatchLeaderboardMainAlpha(rt.ID, payload.MainAlpha); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "set_countdown":
		var payload struct {
			EndTS              int64  `json:"end_ts"`
			BroadcastMsg       string `json:"broadcast_msg"`
			TogglePanelID      string `json:"toggle_panel_id"`
			TogglePanelVisible bool   `json:"toggle_panel_visible"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		if payload.EndTS < 0 {
			payload.EndTS = 0
		}
		rt.CountdownEndTS = payload.EndTS
		rt.CountdownBroadcastMsg = payload.BroadcastMsg
		rt.CountdownTogglePanelID = payload.TogglePanelID
		rt.CountdownToggleVisible = payload.TogglePanelVisible
		rt.CountdownTriggered = false
		if err := s.store.UpdateMatchCountdownConfig(
			rt.ID,
			payload.EndTS,
			payload.BroadcastMsg,
			payload.TogglePanelID,
			payload.TogglePanelVisible,
		); err != nil {
			rt.NextSeq--
			return nil, err
		}
		s.ensureCountdownTimer(
			rt.ID,
			rt.CountdownEndTS,
			rt.CountdownBroadcastMsg,
			rt.CountdownTogglePanelID,
			rt.CountdownToggleVisible,
			rt.CountdownTriggered,
		)

	case "toggle_panel":
		var payload struct {
			PanelID string `json:"panel_id"`
			Visible bool   `json:"visible"`
		}
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			rt.NextSeq--
			return nil, err
		}
		if payload.PanelID == "" {
			rt.NextSeq--
			return nil, errors.New("panel_id is required")
		}
		if rt.Panels == nil {
			rt.Panels = make(map[string]bool)
		}
		rt.Panels[payload.PanelID] = payload.Visible
		if payload.PanelID == "panel-leaderboard" {
			rt.LeaderboardVisible = payload.Visible
		}
		if err := s.storeUpdateMatchConfig(rt); err != nil {
			rt.NextSeq--
			return nil, err
		}

	case "teams_updated":
		teams, err := s.store.ListTeams(rt.ID)
		if err != nil {
			rt.NextSeq--
			return nil, err
		}
		rt.Teams = teams
		// 攻击统计由 events 重建/增量维护，这里无需额外处理。

	case "replay_start":
		// 回放控制：不改变运行态 state，但会广播给屏幕端/复盘端。
		var payload struct {
			FromSeq uint64 `json:"from_seq"`
		}
		// cmd.Data 可能是 {}，此时 FromSeq 为 0，兜底到 1。
		if len(cmd.Data) > 0 && strings.TrimSpace(string(cmd.Data)) != "" && string(cmd.Data) != "null" {
			if err := json.Unmarshal(cmd.Data, &payload); err != nil {
				rt.NextSeq--
				return nil, err
			}
		}
		if payload.FromSeq == 0 {
			payload.FromSeq = 1
		}
		// 不写 rt 状态
		_ = payload

	case "replay_exit":
		// 回放退出：不改变运行态 state，仅广播控制事件给屏幕端。

	default:
		rt.NextSeq--
		return nil, errors.New("unsupported event_type")
	}

	// 持久化事件（用于历史复盘）
	// payload 以原始 json 形式入库，避免丢字段。
	var payloadObj any
	if len(cmd.Data) > 0 && strings.TrimSpace(string(cmd.Data)) != "" && string(cmd.Data) != "null" {
		if err := json.Unmarshal(cmd.Data, &payloadObj); err != nil {
			rt.NextSeq--
			return nil, err
		}
	} else {
		payloadObj = nil
	}
	if err := s.store.InsertEvent(rt.ID, seq, cmd.EventType, payloadObj); err != nil {
		rt.NextSeq--
		return nil, err
	}

	state := runtimeToStateDTO(rt)
	return &protocol.WSMessage{
		Type:      "event",
		MatchID:   matchID,
		Seq:       seq,
		Timestamp: nowTs,
		Event:     cmd.EventType,
		Data:      cmd.Data,
		State:     state,
	}, nil
}

func (s *Service) storeUpdateMatchConfig(rt *MatchRuntime) error {
	return s.store.UpdateMatchConfig(rt.ID, rt.MapType, rt.LeaderboardVisible, rt.Panels)
}

