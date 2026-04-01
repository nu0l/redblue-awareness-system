package match

import "encoding/json"

// DefaultScreenModulesJSON 与前端 DEFAULT_SCREEN_MODULES 保持一致。
const DefaultScreenModulesJSON = `{"left_top":"leaderboard","left_bottom":"region_attack_rank","right_top":"battle_logs","right_bottom":"attack_type_pie"}`

var validScreenSlots = map[string]bool{
	"left_top":    true,
	"left_bottom": true,
	"right_top":   true,
	"right_bottom": true,
}

var validScreenModules = map[string]bool{
	"leaderboard":           true,
	"radar_power":           true,
	"region_attack_rank":    true,
	"battle_logs":           true,
	"attack_type_pie":       true,
	"team_score_bars":       true,
	"attack_type_bars":      true,
	"red_blue_top_compare":  true,
	"attack_metric_cards":   true,
	"team_camp_totals":      true,
	"top_attack_types_list": true,
	"posture_gauge":         true,
}

func defaultScreenModulesMap() map[string]string {
	var m map[string]string
	_ = json.Unmarshal([]byte(DefaultScreenModulesJSON), &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// NormalizeScreenModules 将把未知槽位/模块过滤后合并进默认布局。
func NormalizeScreenModules(raw map[string]string) map[string]string {
	base := defaultScreenModulesMap()
	if raw == nil {
		return base
	}
	for k, v := range raw {
		if validScreenSlots[k] && validScreenModules[v] {
			base[k] = v
		}
	}
	return base
}

func mergeScreenModules(base map[string]string, patch map[string]string) map[string]string {
	out := NormalizeScreenModules(base)
	if patch == nil {
		return out
	}
	for k, v := range patch {
		if validScreenSlots[k] && validScreenModules[v] {
			out[k] = v
		}
	}
	return out
}
