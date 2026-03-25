export type TeamType = "red" | "blue";

export type WSMessage =
  | SyncStateMessage
  | EventMessage;

export type TeamDTO = {
  id: number;
  name: string;
  type: TeamType;
  score: number;
  initial_score?: number;
  logo?: string;
  members?: string[];
};

export type AttackStatDTO = {
  name: string;
  value: number;
};

export type MatchStateDTO = {
  map_type: string; // china | taizhou
  leaderboard_visible: boolean;
  teams: TeamDTO[];
  attack_stats: AttackStatDTO[];
  panels: Record<string, boolean>;
  countdown_end_ts?: number;
  countdown_broadcast_msg?: string;
  countdown_toggle_panel_id?: string;
  countdown_toggle_visible?: boolean;
  countdown_triggered?: boolean;
  screen_title?: string;
  screen_organizer?: string;
  screen_supporter?: string;
  bgm_url?: string;
  bgm_enabled?: boolean;
  success_sfx_url?: string;
  success_sfx_enabled?: boolean;
  leaderboard_main_alpha?: number;
  /** 后端托管的得分总榜背景路径，如 /uploads/{match_id}/leaderboard-bg.png；空则用前端默认图 */
  leaderboard_bg_url?: string;
};

export type SyncStateMessage = {
  type: "sync_state";
  match_id: string;
  seq?: number;
  timestamp: number;
  state: MatchStateDTO;
};

export type EventMessage = {
  type: "event";
  match_id: string;
  seq: number;
  timestamp: number;
  event: string;
  data: any;
  state: MatchStateDTO;
};

