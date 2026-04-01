/** 大屏四象限槽位：后台可配置每个槽位展示的功能模块 */
export const SCREEN_SLOTS = ["left_top", "left_bottom", "right_top", "right_bottom"] as const;
export type ScreenSlotId = (typeof SCREEN_SLOTS)[number];

export const SCREEN_MODULE_IDS = [
  "leaderboard",
  "radar_power",
  "region_attack_rank",
  "battle_logs",
  "attack_type_pie",
  "team_score_bars",
  "attack_type_bars",
  "red_blue_top_compare",
  "attack_metric_cards",
  "team_camp_totals",
  "top_attack_types_list",
  "posture_gauge",
] as const;
export type ScreenModuleId = (typeof SCREEN_MODULE_IDS)[number];

export const DEFAULT_SCREEN_MODULES: Record<ScreenSlotId, ScreenModuleId> = {
  left_top: "leaderboard",
  left_bottom: "region_attack_rank",
  right_top: "battle_logs",
  right_bottom: "attack_type_pie",
};

export const SCREEN_MODULE_LABELS: Record<ScreenModuleId, string> = {
  leaderboard: "战队实时得分榜",
  radar_power: "综合战力评估（雷达）",
  region_attack_rank: "被攻击区县/城市榜单",
  battle_logs: "实时战况日志",
  attack_type_pie: "高频战术统计（饼图）",
  team_score_bars: "战队积分横向对比",
  attack_type_bars: "战术类型分布（条形）",
  red_blue_top_compare: "红蓝首席得分对比",
  attack_metric_cards: "攻防态势指标卡",
  team_camp_totals: "红蓝阵营总分对比",
  top_attack_types_list: "战术类型命中 TOP 榜",
  posture_gauge: "态势强度仪表盘",
};

/** 纯 HTML 展示、不占用 ECharts 容器的模块 */
export const SCREEN_HTML_MODULE_IDS = [
  "attack_metric_cards",
  "team_camp_totals",
  "top_attack_types_list",
] as const;

export function isHtmlPanelModule(id: ScreenModuleId): boolean {
  return (SCREEN_HTML_MODULE_IDS as readonly string[]).includes(id);
}

const MODULE_SET = new Set<string>(SCREEN_MODULE_IDS);
const SLOT_SET = new Set<string>(SCREEN_SLOTS);

export function normalizeScreenModules(raw?: Record<string, string> | null): Record<ScreenSlotId, ScreenModuleId> {
  const out = { ...DEFAULT_SCREEN_MODULES };
  if (!raw) return out;
  for (const k of SCREEN_SLOTS) {
    const v = raw[k];
    if (v && MODULE_SET.has(v)) {
      out[k] = v as ScreenModuleId;
    }
  }
  return out;
}

export function isValidScreenModule(id: string): id is ScreenModuleId {
  return MODULE_SET.has(id);
}

export function isValidSlot(id: string): id is ScreenSlotId {
  return SLOT_SET.has(id);
}

export function mergeScreenModulesPatch(
  current: Record<ScreenSlotId, ScreenModuleId>,
  patch: Record<string, string>
): Record<ScreenSlotId, ScreenModuleId> {
  const next = { ...current };
  for (const [k, v] of Object.entries(patch)) {
    if (!isValidSlot(k) || !isValidScreenModule(v)) continue;
    next[k] = v;
  }
  return next;
}
