<template>
  <div class="lb-root" :style="lbRootStyle">
    <!-- 背景：后台上传图（state.leaderboard_bg_url）或默认 public/leaderboard-cyber-bg.png -->
    <div class="lb-bg-photo" :style="bgPhotoStyle" aria-hidden="true"></div>
    <div class="lb-bg-base" aria-hidden="true"></div>
    <div class="lb-bg-red-zone" aria-hidden="true"></div>
    <div class="lb-bg-blue-zone" aria-hidden="true"></div>
    <div class="lb-bg-grid" aria-hidden="true"></div>
    <div class="lb-bg-scan" aria-hidden="true"></div>
    <div class="lb-bg-vline" aria-hidden="true"></div>

    <header class="lb-header">
      <div class="lb-header-left">
        <span class="lb-ws" :class="wsConnected ? 'ok' : 'bad'" aria-hidden="true"></span>
        <span class="lb-live font-cyber">LIVE</span>
      </div>
      <div class="lb-header-center">
        <h1 class="lb-title font-cyber">{{ state.screen_title || "实战化红蓝对抗演练指挥中心" }}</h1>
        <p class="lb-theme-line font-cyber">红蓝对抗 · 网络攻防</p>
        <p class="lb-subtitle">
          <span class="lb-badge">得分总榜</span>
          <span class="lb-dot">·</span>
          <span class="lb-mid">{{ matchIdShort }}</span>
        </p>
      </div>
      <div class="lb-header-right">
        <div class="lb-clock-box font-cyber">
          <span class="lb-date">{{ dateStr }}</span>
          <span class="lb-time">{{ clock }}</span>
        </div>
      </div>
    </header>

    <div v-if="!jwtToken" class="lb-error">
      缺少 token：请从管理后台「跳转得分总榜」打开，或在 URL 增加 <code>&amp;token=...</code>
    </div>
    <div v-else-if="!matchId" class="lb-error">缺少 match_id：请在 URL 增加 <code>?match_id=场次ID</code></div>

    <main v-else class="lb-main">
      <div class="lb-panel">
        <div class="lb-panel-inner">
          <template v-if="sortedTeams.length">
            <div class="lb-table-head">
              <span class="th-rank">排名</span>
              <span class="th-name">战队</span>
              <span class="th-side">阵营</span>
              <span class="th-score">得分</span>
            </div>
            <div class="lb-table-body">
              <div
                v-for="item in rankedRows"
                :key="item.team.id"
                class="lb-row"
                :class="{
                  'row-gold': item.rank === 1,
                  'row-silver': item.rank === 2,
                  'row-bronze': item.rank === 3,
                }"
              >
                <span class="cell-rank font-cyber">
                  <span v-if="item.rank === 1" class="medal" aria-hidden="true">🥇</span>
                  <span v-else-if="item.rank === 2" class="medal" aria-hidden="true">🥈</span>
                  <span v-else-if="item.rank === 3" class="medal" aria-hidden="true">🥉</span>
                  <span class="rank-num">{{ item.rank }}</span>
                </span>
                <span
                  class="cell-name"
                  :class="item.team.type === 'red' ? 'name-red' : 'name-blue'"
                  :title="item.team.name"
                  >{{ item.team.name }}</span
                >
                <span class="cell-side">
                  <span class="pill" :class="item.team.type === 'red' ? 'pill-red' : 'pill-blue'">{{
                    item.team.type === "red" ? "红方" : "蓝方"
                  }}</span>
                </span>
                <span
                  class="cell-score font-cyber"
                  :class="[
                    item.team.type === 'red' ? 'score-red' : 'score-blue',
                    scorePulse[item.team.id] ? 'score-pulse' : '',
                  ]"
                  >{{ item.team.score.toLocaleString() }}</span
                >
              </div>
            </div>
          </template>
          <div v-else class="lb-empty">暂无队伍数据</div>
        </div>
      </div>
    </main>

    <footer v-if="state.screen_organizer || state.screen_supporter" class="lb-footer">
      <span v-if="state.screen_organizer">主办方 {{ state.screen_organizer }}</span>
      <span v-if="state.screen_organizer && state.screen_supporter" class="lb-foot-sep">·</span>
      <span v-if="state.screen_supporter">支撑 {{ state.screen_supporter }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import type { MatchStateDTO, WSMessage } from "../shared/types";
import { connectMatchWS } from "../shared/wsClient";

const lbPageParams = new URL(window.location.href).searchParams;
/** 与后台、上传资源同源；优先 URL ?api_base= 再 VITE_API_BASE（支持 /api 相对路径） */
const apiFromQuery = lbPageParams.get("api_base")?.trim() || lbPageParams.get("api")?.trim();
// @ts-ignore
const deployMode = String((import.meta as any).env?.VITE_DEPLOY_MODE ?? "proxy").trim().toLowerCase();
const directApiBase = String((import.meta as any).env?.VITE_DIRECT_API_BASE ?? "http://127.0.0.1:8080").trim();
const rawApiBase = String((import.meta as any).env?.VITE_API_BASE ?? "").trim();
const envApiBaseRaw =
  deployMode === "direct" && (rawApiBase === "" || rawApiBase === "/api" || rawApiBase === "/api/")
    ? directApiBase
    : (rawApiBase || (deployMode === "direct" ? directApiBase : "/api"));
// @ts-ignore
const apiBaseRaw = (
  apiFromQuery && apiFromQuery.length > 0
    ? apiFromQuery
    : envApiBaseRaw
).replace(/\/$/, "");
const apiBaseRoot = apiBaseRaw.endsWith("/api") ? apiBaseRaw.slice(0, -4) : apiBaseRaw;
const apiBaseHttp = apiBaseRoot || window.location.origin;

const matchId = lbPageParams.get("match_id") ?? "";
const jwtToken = lbPageParams.get("token") ?? "";

/** 主容器透明度：URL ?lb_main_alpha=0.18（0~1），用于调节 lb-main 遮罩层强度 */
function readLbMainAlpha() {
  const v = parseFloat(lbPageParams.get("lb_main_alpha") || "");
  if (Number.isFinite(v)) return Math.max(0, Math.min(1, v));
  return 0.14;
}
const lbMainAlphaFromURL = readLbMainAlpha();
const lbRootStyle = computed(() => {
  const fromState = Number(state.leaderboard_main_alpha);
  const alpha = Number.isFinite(fromState) ? Math.max(0, Math.min(1, fromState)) : lbMainAlphaFromURL;
  // 让“透明度”调节更明显：同时影响主区遮罩与榜单面板不透明度
  const main = 0.02 + alpha * 0.62;
  const panelA = 0.18 + alpha * 0.76;
  const panelB = Math.min(0.98, panelA + 0.04);
  return {
    "--lb-panel-a": String(panelA),
    "--lb-panel-b": String(panelB),
    "--lb-main-alpha": String(main),
  } as Record<string, string>;
});

const matchIdShort = computed(() => (matchId.length > 12 ? `${matchId.slice(0, 8)}…${matchId.slice(-4)}` : matchId || "—"));

const clock = ref("--:--:--");
const dateStr = ref("");

const wsConnected = ref(false);
let wsClose: (() => void) | null = null;

const state = reactive<MatchStateDTO>({
  screen_title: "",
  screen_organizer: "",
  screen_supporter: "",
  leaderboard_main_alpha: lbMainAlphaFromURL,
  leaderboard_bg_url: "",
  map_type: "china",
  leaderboard_visible: true,
  teams: [],
  attack_stats: [],
  panels: {},
});

/** 自定义背景走 API 同源；未配置则用前端默认图 */
const defaultLbBgPath = "/leaderboard-cyber-bg.png";
/** 背景 URL 变化时 bump，避免浏览器缓存旧图 */
const bgCacheNonce = ref(0);
watch(
  () => (state.leaderboard_bg_url ?? "").trim(),
  (n, o) => {
    if (n !== o) bgCacheNonce.value++;
  }
);

const bgPhotoStyle = computed(() => {
  const raw = (state.leaderboard_bg_url ?? "").trim();
  const base = apiBaseHttp;
  let url: string;
  if (raw.startsWith("/")) {
    const sep = raw.includes("?") ? "&" : "?";
    url = `${base}${raw}${sep}_v=${bgCacheNonce.value}`;
  } else if (/^https?:\/\//i.test(raw)) {
    const sep = raw.includes("?") ? "&" : "?";
    url = `${raw}${sep}_v=${bgCacheNonce.value}`;
  } else {
    url = defaultLbBgPath;
  }
  return { backgroundImage: `url("${url}")` };
});

async function hydrateStateFromHttp() {
  if (!matchId || !jwtToken) return;
  try {
    const res = await fetch(`${apiBaseHttp}/api/matches/${encodeURIComponent(matchId)}/state`, {
      headers: { Authorization: `Bearer ${jwtToken}` },
    });
    if (!res.ok) return;
    const json = await res.json();
    if (json?.state && typeof json.state === "object") {
      Object.assign(state, json.state);
    }
  } catch {
    /* 仅依赖 WS 亦可 */
  }
}

const sortedTeams = computed(() => [...state.teams].sort((a, b) => b.score - a.score));

const rankedRows = computed(() => sortedTeams.value.map((team, i) => ({ rank: i + 1, team })));

const scorePulse = reactive<Record<number, number>>({});

function pulseTeamScore(teamId: number) {
  scorePulse[teamId] = Date.now();
  window.setTimeout(() => delete scorePulse[teamId], 900);
}

let clockTimer: number | undefined;

onMounted(async () => {
  clockTimer = window.setInterval(() => {
    const now = new Date();
    clock.value = now.toLocaleTimeString("en-US", { hour12: false });
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, "0");
    const d = String(now.getDate()).padStart(2, "0");
    dateStr.value = `${y}-${m}-${d}`;
  }, 1000);

  if (!matchId || !jwtToken) return;

  await hydrateStateFromHttp();

  const wsConn = connectMatchWS({
    matchId,
    apiBaseHttp,
    token: jwtToken,
    onOpen: () => {
      wsConnected.value = true;
    },
    onClose: () => {
      wsConnected.value = false;
    },
    onMessage: (msg) => {
      const m = msg as WSMessage;
      if (m.type === "sync_state" && m.state) {
        Object.assign(state, m.state);
        return;
      }
      if (m.type === "event" && m.state) {
        const prev = new Map<number, number>(state.teams.map((t) => [t.id, t.score]));
        Object.assign(state, m.state);
        for (const t of state.teams) {
          const old = prev.get(t.id);
          if (old !== undefined && t.score !== old) pulseTeamScore(t.id);
        }
      }
    },
  });
  wsClose = () => wsConn.close();
});

onBeforeUnmount(() => {
  if (clockTimer) window.clearInterval(clockTimer);
  wsClose?.();
});
</script>

<style scoped>
.lb-root {
  position: relative;
  min-height: 100vh;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: #e8eef5;
  font-family: "Noto Sans SC", "Source Han Sans SC", system-ui, -apple-system, sans-serif;
  -webkit-font-smoothing: antialiased;
}

/* 红蓝对抗 / 网络攻防：底图 + 遮罩 + 原有光效叠层 */
.lb-bg-photo {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background-color: #050810;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}

.lb-bg-base {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  /* 减轻遮罩，底图更易辨认 */
  background: linear-gradient(180deg, rgba(5, 8, 16, 0.38) 0%, rgba(5, 8, 16, 0.26) 45%, rgba(5, 8, 16, 0.44) 100%);
}

.lb-bg-red-zone {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background: radial-gradient(ellipse 85% 120% at 0% 50%, rgba(220, 38, 38, 0.16), transparent 58%),
    radial-gradient(ellipse 50% 60% at 15% 100%, rgba(239, 68, 68, 0.09), transparent 50%);
}

.lb-bg-blue-zone {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background: radial-gradient(ellipse 85% 120% at 100% 50%, rgba(37, 99, 235, 0.14), transparent 58%),
    radial-gradient(ellipse 50% 60% at 85% 0%, rgba(56, 189, 248, 0.08), transparent 50%);
}

.lb-bg-grid {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  opacity: 0.45;
  background-image: linear-gradient(rgba(248, 113, 113, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(56, 189, 248, 0.07) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: linear-gradient(90deg, transparent 0%, #000 12%, #000 88%, transparent 100%);
}

.lb-bg-scan {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background: repeating-linear-gradient(
    0deg,
    transparent,
    transparent 2px,
    rgba(0, 0, 0, 0.12) 2px,
    rgba(0, 0, 0, 0.12) 4px
  );
  opacity: 0.22;
  animation: lbScan 10s linear infinite;
}

@keyframes lbScan {
  0% {
    transform: translateY(0);
  }
  100% {
    transform: translateY(8px);
  }
}

.lb-bg-vline {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background: linear-gradient(
    90deg,
    transparent calc(50% - 2px),
    rgba(255, 255, 255, 0.06) 50%,
    transparent calc(50% + 2px)
  );
  box-shadow: inset 0 0 80px rgba(0, 0, 0, 0.5);
}

.lb-header,
.lb-main,
.lb-footer {
  position: relative;
  z-index: 1;
}

.lb-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  padding: 14px 32px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(180deg, rgba(10, 15, 30, 0.92), rgba(10, 15, 30, 0.55));
  backdrop-filter: blur(10px);
  box-shadow: 0 1px 0 rgba(248, 113, 113, 0.12), 0 1px 0 rgba(56, 189, 248, 0.1);
}

.lb-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 80px;
}

.lb-live {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.22em;
  color: rgba(52, 211, 153, 0.95);
}

.lb-header-center {
  flex: 1;
  text-align: center;
  padding: 0 16px;
}

.lb-title {
  margin: 0;
  font-size: clamp(22px, 2.8vw, 40px);
  font-weight: 800;
  letter-spacing: 0.05em;
  line-height: 1.2;
  background: linear-gradient(90deg, #fca5a5 0%, #f8fafc 48%, #7dd3fc 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  filter: drop-shadow(0 0 24px rgba(248, 113, 113, 0.2)) drop-shadow(0 0 24px rgba(56, 189, 248, 0.2));
}

.lb-theme-line {
  margin: 6px 0 0;
  font-size: clamp(13px, 1.35vw, 18px);
  font-weight: 600;
  letter-spacing: 0.35em;
  color: rgba(148, 163, 184, 0.95);
  text-transform: uppercase;
}

.lb-subtitle {
  margin: 8px 0 0;
  font-size: clamp(13px, 1.2vw, 16px);
  color: rgba(203, 213, 225, 0.92);
  letter-spacing: 0.06em;
}

.lb-badge {
  display: inline-block;
  padding: 4px 14px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 0.95em;
  color: #0c1220;
  background: linear-gradient(90deg, #f87171, #38bdf8);
  box-shadow: 0 0 20px rgba(248, 113, 113, 0.25), 0 0 20px rgba(56, 189, 248, 0.25);
}

.lb-dot {
  margin: 0 8px;
  opacity: 0.5;
}
.lb-mid {
  font-family: ui-monospace, monospace;
  font-size: 1em;
  opacity: 0.88;
}

.lb-header-right {
  min-width: 140px;
  display: flex;
  justify-content: flex-end;
}

.lb-clock-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  line-height: 1.2;
}

.lb-date {
  font-size: 13px;
  color: rgba(148, 163, 184, 0.95);
}
.lb-time {
  font-size: clamp(20px, 2vw, 28px);
  font-weight: 700;
  color: #bae6fd;
  letter-spacing: 0.08em;
}

.lb-ws {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  box-shadow: 0 0 10px currentColor;
}
.lb-ws.ok {
  background: #34d399;
  color: #34d399;
}
.lb-ws.bad {
  background: #f87171;
  color: #f87171;
}

.lb-error {
  margin: 24px auto;
  max-width: 640px;
  padding: 18px 22px;
  background: rgba(127, 29, 29, 0.28);
  border: 1px solid rgba(248, 113, 113, 0.35);
  border-radius: 12px;
  color: #fecaca;
  text-align: center;
  font-size: 15px;
  line-height: 1.55;
}
.lb-error code {
  color: #fde68a;
}

.lb-main {
  flex: 1;
  min-height: 0;
  padding: 14px 28px 12px;
  display: flex;
  flex-direction: column;
  background: rgba(5, 8, 16, var(--lb-main-alpha, 0.14));
}

.lb-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.lb-panel-inner {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  /* 半透明面板：--lb-panel-a / --lb-panel-b 可由 URL ?lb_panel= / ?lb_panel_b= 调节（0.2–1） */
  background: linear-gradient(
    165deg,
    rgb(15 23 42 / var(--lb-panel-a, 0.82)) 0%,
    rgb(8 12 28 / var(--lb-panel-b, 0.86)) 100%
  );
  box-shadow: 0 0 0 1px rgba(248, 113, 113, 0.08) inset, 0 0 0 1px rgba(56, 189, 248, 0.08) inset,
    0 24px 60px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(8px);
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(56, 189, 248, 0.4) transparent;
}

.lb-table-head {
  display: grid;
  grid-template-columns: 88px 1fr 100px minmax(120px, 160px);
  gap: 12px;
  align-items: center;
  padding: 14px 24px 14px 20px;
  font-size: clamp(14px, 1.25vw, 18px);
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: rgba(186, 230, 253, 0.85);
  background: rgba(15, 23, 42, 0.95);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.th-rank {
  text-align: center;
}
.th-score {
  text-align: right;
}

.lb-table-body {
  flex: 1;
  padding: 12px 16px 20px;
}

.lb-row {
  display: grid;
  grid-template-columns: 88px 1fr 100px minmax(120px, 160px);
  gap: 12px;
  align-items: center;
  padding: 14px 18px 14px 14px;
  margin-bottom: 8px;
  border-radius: 12px;
  border: 1px solid transparent;
  background: rgba(255, 255, 255, 0.03);
  transition: background 0.15s ease, border-color 0.15s ease;
}
.lb-row:nth-child(even) {
  background: rgba(255, 255, 255, 0.05);
}
.lb-row:hover {
  background: rgba(56, 189, 248, 0.08);
  border-color: rgba(56, 189, 248, 0.2);
}

.row-gold {
  border-color: rgba(250, 204, 21, 0.45);
  background: linear-gradient(90deg, rgba(250, 204, 21, 0.14), transparent 72%) !important;
}
.row-silver {
  border-color: rgba(203, 213, 225, 0.35);
  background: linear-gradient(90deg, rgba(203, 213, 225, 0.12), transparent 72%) !important;
}
.row-bronze {
  border-color: rgba(217, 119, 6, 0.4);
  background: linear-gradient(90deg, rgba(217, 119, 6, 0.12), transparent 72%) !important;
}

.cell-rank {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: clamp(20px, 1.9vw, 28px);
  font-weight: 800;
  color: rgba(241, 245, 249, 0.92);
}
.medal {
  font-size: 1em;
  line-height: 1;
}
.rank-num {
  min-width: 1.2em;
  text-align: center;
}

.cell-name {
  font-size: clamp(20px, 1.85vw, 30px);
  font-weight: 800;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.name-red {
  color: #fecaca;
  text-shadow: 0 0 20px rgba(248, 113, 113, 0.25);
}
.name-blue {
  color: #93c5fd;
  text-shadow: 0 0 20px rgba(59, 130, 246, 0.25);
}

.cell-side {
  display: flex;
  justify-content: center;
}
.pill {
  display: inline-block;
  min-width: 56px;
  padding: 6px 12px;
  text-align: center;
  font-size: clamp(14px, 1.2vw, 18px);
  font-weight: 800;
  border-radius: 8px;
  letter-spacing: 0.08em;
}
.pill-red {
  color: #fecaca;
  background: rgba(239, 68, 68, 0.28);
  border: 1px solid rgba(248, 113, 113, 0.45);
  box-shadow: 0 0 16px rgba(248, 113, 113, 0.15);
}
.pill-blue {
  color: #dbeafe;
  background: rgba(59, 130, 246, 0.28);
  border: 1px solid rgba(96, 165, 250, 0.45);
  box-shadow: 0 0 16px rgba(59, 130, 246, 0.15);
}

.cell-score {
  text-align: right;
  font-size: clamp(22px, 2.1vw, 36px);
  font-weight: 800;
  letter-spacing: 0.03em;
  font-variant-numeric: tabular-nums;
}
.score-red {
  color: #fca5a5;
  text-shadow: 0 0 18px rgba(248, 113, 113, 0.45);
}
.score-blue {
  color: #7dd3fc;
  text-shadow: 0 0 18px rgba(56, 189, 248, 0.4);
}

@keyframes scorePulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.05);
    filter: brightness(1.25);
  }
  100% {
    transform: scale(1);
  }
}
.score-pulse {
  animation: scorePulse 750ms ease-in-out;
}

.lb-empty {
  padding: 48px;
  text-align: center;
  color: rgba(148, 163, 184, 0.9);
  font-size: clamp(18px, 1.6vw, 22px);
}

.lb-footer {
  flex-shrink: 0;
  padding: 10px 20px 14px;
  text-align: center;
  font-size: clamp(13px, 1.1vw, 16px);
  color: rgba(203, 213, 225, 0.88);
  letter-spacing: 0.05em;
}
.lb-foot-sep {
  margin: 0 12px;
  opacity: 0.45;
}

.font-cyber {
  font-family: "Orbitron", ui-sans-serif, system-ui, sans-serif;
}

@media (max-width: 720px) {
  .lb-table-head,
  .lb-row {
    grid-template-columns: 64px 1fr 72px minmax(88px, 1fr);
    gap: 8px;
    padding-left: 10px;
    padding-right: 12px;
  }
}
</style>
