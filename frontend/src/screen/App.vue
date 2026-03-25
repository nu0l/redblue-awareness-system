<template>
  <div class="screen-root" :style="screenRootStyle">
    <header class="topbar">
      <div class="topbar-left">
        <div
          class="ws-dot"
          :class="wsConnected ? 'ws-dot-ok' : 'ws-dot-bad'"
          aria-hidden="true"
        ></div>
        <span class="topbar-label font-cyber">WS_LINK</span>
        <div class="topbar-bar"></div>
        <div class="topbar-bar-mini"></div>
      </div>

      <div class="topbar-center">
        <h1
          class="topbar-title glow-text"
          :title="state.screen_title || '实战化红蓝对抗演练指挥中心'"
        >
          {{ state.screen_title || "实战化红蓝对抗演练指挥中心" }}
        </h1>
      </div>

      <div class="topbar-right">
        <div class="topbar-clock font-cyber">
          <div class="topbar-date">{{ dateStr }}</div>
          <div class="topbar-time">{{ clock }}</div>
        </div>
        <div class="topbar-bar-mini"></div>
        <div class="topbar-bar"></div>
      </div>
    </header>

    <div
      v-if="alertVisible"
      id="alert-bar"
      class="alert-bar"
      :class="[`alert-bar-${alertKind}`]"
    >
      <div class="alert-inner">
        <div class="alert-tag">{{ alertKind === "attack" ? "攻击命中" : alertKind === "system" ? "系统广播" : "态势提醒" }}</div>
        <div class="alert-main">{{ alertText }}</div>
      </div>
    </div>

    <audio ref="bgmAudioEl" loop preload="none"></audio>
    <audio ref="successSfxAudioEl" preload="none"></audio>

    <main ref="layoutElRef" class="layout">
      <!-- 左侧面板 -->
      <aside ref="leftPanelEl" class="panel panel-left" :style="useCustomPanelWidths ? leftPanelStyle : undefined">
        <div
          v-show="state.panels?.['panel-leaderboard'] ?? true"
          id="panel-leaderboard"
          class="cyber-panel panel-inner"
          :style="leftTopPanelStyle"
        >
          <h2 class="panel-title">
            <span class="panel-dot"></span> 战队实时得分榜
          </h2>
          <div ref="leaderViewportEl" class="leader-list">
            <div class="leader-track" :class="leaderTrackAnimating ? 'anim' : ''" :style="leaderTrackStyle">
              <div
                v-for="(team, idx) in leaderboardWindow"
                :key="`${team.id}-${idx}`"
                class="leader-item"
              >
              <div class="leader-left">
                <span
                  class="leader-rank font-cyber"
                  :class="getTeamRank(team.id) <= 3 ? 'rank-top' : 'rank-rest'"
                  >#{{ getTeamRank(team.id) }}</span
                >
                <span class="leader-name" :class="team.type === 'red' ? 'name-red' : 'name-blue'">
                  {{ team.name }}
                </span>
              </div>
              <div
                class="leader-score font-cyber"
                :class="[
                  team.type === 'red' ? 'score-red' : 'score-blue',
                  scorePulse[team.id] ? 'score-pulse' : ''
                ]"
              >
                {{ team.score.toLocaleString() }}
              </div>
            </div>
            </div>
          </div>
        </div>

        <div
          class="panel-v-splitter"
          title="拖拽调整左侧上下高度"
          @mousedown.prevent="startResizeLeftVertical"
        ></div>

        <div class="cyber-panel panel-inner radar-panel">
          <h2 class="panel-title">
            <span class="panel-dot"></span> 综合战力评估
          </h2>
          <div ref="radarEl" class="chart radar-chart"></div>
        </div>
      </aside>

      <div
        class="layout-splitter layout-splitter-left"
        title="拖拽调整左侧宽度"
        @mousedown.prevent="startResizeLeft"
      ></div>

      <!-- 中间地图 -->
      <section class="map-section cyber-panel" :class="mapImpactActive ? 'map-impact' : ''">
        <div class="map-decor map-decor-tl"></div>
        <div class="map-decor map-decor-tr"></div>
        <div class="map-decor map-decor-bl"></div>
        <div class="map-decor map-decor-br"></div>

        <button class="map-settings-btn" :disabled="isReplaying" @click="showMapSettings = !showMapSettings">设置</button>
        <div v-if="showMapSettings" class="map-mode-indicator">
          <div class="map-mode-title">显示</div>
          <div class="map-settings-row">
            <span class="map-settings-label">字体</span>
            <input
              v-model.number="fontUserScale"
              type="range"
              min="0.75"
              max="1.45"
              step="0.05"
              class="map-settings-range"
            />
            <span class="map-settings-val">{{ Math.round(fontUserScale * 100) }}%</span>
          </div>
          <div class="map-settings-row map-settings-hint">左右栏宽拖拽两侧竖条调节，自动保存本机</div>
          <button type="button" class="map-reset-layout-btn" :disabled="isReplaying" @click="resetDefaultLayout">
            恢复默认布局
          </button>
          <div class="map-mode-title">音频</div>
          <button type="button" class="map-reset-layout-btn" :disabled="isReplaying" @click="toggleBGMManually">
            {{ bgmManualPlaying ? "BGM停止" : "BGM播放" }}
          </button>
          <div class="map-mode-title">底图</div>
          <div class="map-mode-switch">
            <button
              class="map-mode-btn"
              :class="state.map_type === 'china' ? 'active' : ''"
              :disabled="isReplaying"
              @click="setMapMode('china')"
            >
              全国态势
            </button>
            <button
              class="map-mode-btn"
              :class="state.map_type === 'taizhou' ? 'active' : ''"
              :disabled="isReplaying"
              @click="setMapMode('taizhou')"
            >
              泰州市区县态势
            </button>
          </div>
        </div>

        <div ref="mapEl" class="chart map-chart"></div>
      </section>

      <div
        class="layout-splitter layout-splitter-right"
        title="拖拽调整右侧宽度"
        @mousedown.prevent="startResizeRight"
      ></div>

      <!-- 右侧面板 -->
      <aside ref="rightPanelEl" class="panel panel-right" :style="useCustomPanelWidths ? rightPanelStyle : undefined">
        <div class="cyber-panel panel-inner panel-logs" :style="rightTopPanelStyle">
          <h2 class="panel-title">
            <span class="panel-dot"></span> 实时战况日志
          </h2>
          <div ref="terminalScrollEl" class="terminal-scroll terminal-logs">
            <div class="terminal-track" :class="terminalTrackAnimating ? 'anim' : ''" :style="terminalTrackStyle">
              <div
                v-for="(line, idx) in terminalWindow"
                :key="`${terminalStartIndex}-${idx}`"
                class="log-line"
              >
                <span :class="logLineClass(line)">{{ line }}</span>
              </div>
            </div>
          </div>
        </div>

        <div
          class="panel-v-splitter"
          title="拖拽调整右侧上下高度"
          @mousedown.prevent="startResizeRightVertical"
        ></div>

        <div class="cyber-panel panel-inner panel-pie">
          <h2 class="panel-title">
            <span class="panel-dot"></span> 高频战术统计
          </h2>
          <div ref="pieEl" class="chart pie-chart"></div>
        </div>
      </aside>
    </main>

    <footer class="bottom-credits">
      <div class="credits-line">
        <span class="credits-label">主办方</span>
        <span class="credits-sep">：</span>
        <span class="credits-value">{{ state.screen_organizer || "-" }}</span>
      </div>
      <div class="credits-divider" aria-hidden="true"></div>
      <div class="credits-line">
        <span class="credits-label">支撑方</span>
        <span class="credits-sep">：</span>
        <span class="credits-value">{{ state.screen_supporter || "-" }}</span>
      </div>
    </footer>

    <!-- 历史复盘回放控制台 -->
    <div v-if="showReplayUI" class="replay-ui">
      <div class="replay-row">
        <button class="replay-btn" :disabled="replayLoading" @click="() => loadReplay()">
          {{ replayLoading ? "加载中..." : "加载回放" }}
        </button>
        <button class="replay-btn primary" :disabled="!replayReady || replayPlaying" @click="startReplay">
          播放
        </button>
        <button class="replay-btn" :disabled="!replayPlaying" @click="pauseReplay">
          暂停
        </button>
        <button class="replay-btn" :disabled="!replayReady || replayPlaying" @click="stepReplay">
          单步
        </button>

        <div class="replay-speed">
          <span class="replay-label">倍速</span>
          <select v-model.number="replaySpeed" class="replay-select">
            <option :value="1">1x</option>
            <option :value="2">2x</option>
            <option :value="4">4x</option>
            <option :value="8">8x</option>
          </select>
        </div>

        <div class="replay-meta">
          {{ replayCursor }} / {{ replayEvents.length }}
        </div>

        <button
          v-if="isReplaying"
          class="replay-btn"
          :disabled="replayLoading"
          @click="exitReplay"
          style="margin-left: 8px"
        >
          返回实时
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import * as echarts from "echarts";
import type { MatchStateDTO, TeamDTO, WSMessage } from "../shared/types";
import { connectMatchWS } from "../shared/wsClient";

// 这里保留与旧版一致的点位字典（后续可改为后台下发 GeoJSON/点位配置）。
const CITIES: Record<string, [number, number]> = {
  北京: [116.405285, 39.904989],
  天津: [117.190182, 39.125596],
  石家庄: [114.51488, 38.04214],
  太原: [112.549248, 37.857014],
  呼和浩特: [111.749434, 40.842396],
  沈阳: [123.429159, 41.796767],
  长春: [125.3245, 43.8868],
  哈尔滨: [126.6424, 45.756],
  南京: [118.796877, 32.060255],
  上海: [121.472644, 31.231706],
  杭州: [120.153576, 30.287459],
  合肥: [117.2272, 31.8206],
  福州: [119.3063, 26.0753],
  南昌: [115.8582, 28.6834],
  济南: [117.1582, 36.8701],
  郑州: [113.6254, 34.7466],
  武汉: [114.298572, 30.584355],
  长沙: [112.982279, 28.19409],
  广州: [113.280637, 23.125178],
  成都: [104.065735, 30.659462],
  西安: [108.948024, 34.263161],
  兰州: [103.8343, 36.0611],
  西宁: [101.7782, 36.6173],
  银川: [106.2282, 38.4872],
  乌鲁木齐: [87.616873, 43.825592],
  拉萨: [91.132212, 29.660361],
  贵阳: [106.630156, 26.647521],
  昆明: [102.8329, 24.8801],
  南宁: [108.331, 22.817],
  海口: [110.35, 20.02],
  重庆: [106.551556, 29.56301],
};

const TAIZHOU_CITIES: Record<string, [number, number]> = {
  市区: [120.0, 32.45],
  海陵区: [119.919, 32.493],
  高港区: [119.88, 32.318],
  姜堰区: [120.146, 32.508],
  兴化市: [119.852, 32.909],
  靖江市: [120.273, 32.015],
  泰兴市: [120.015, 32.163],
};

type AttackStatus = "attempt" | "lateral" | "success" | "defense_success" | "trace_success";

const attackStatsDefault = [] as { name: string; value: number }[];

// @ts-ignore: Vetur 对 import.meta 在此文件的类型推断可能不匹配。
const deployMode = String((import.meta as any).env?.VITE_DEPLOY_MODE ?? "proxy").trim().toLowerCase();
const directApiBase = String((import.meta as any).env?.VITE_DIRECT_API_BASE ?? "http://127.0.0.1:8080").trim();
const rawApiBase = String((import.meta as any).env?.VITE_API_BASE ?? "").trim();
const apiBaseEnvRaw =
  deployMode === "direct" && (rawApiBase === "" || rawApiBase === "/api" || rawApiBase === "/api/")
    ? directApiBase
    : (rawApiBase || (deployMode === "direct" ? directApiBase : "/api"));
const apiBaseEnv = apiBaseEnvRaw.replace(/\/$/, "");
const apiBaseRoot = apiBaseEnv.endsWith("/api") ? apiBaseEnv.slice(0, -4) : apiBaseEnv;
const apiBaseHttp = apiBaseRoot || window.location.origin;
const matchId = new URL(window.location.href).searchParams.get("match_id") ?? "";
const jwtToken = new URL(window.location.href).searchParams.get("token") ?? "";
const authHeaders: Record<string, string> = {};
if (jwtToken) {
  authHeaders.Authorization = `Bearer ${jwtToken}`;
}

const dateStr = ref("YYYY-MM-DD");
const clock = ref("00:00:00");
let clockTimer: number | undefined;

const wsConnected = ref(false);
const wsConn = ref<ReturnType<typeof connectMatchWS> | null>(null);
const uiScale = ref(1);
const panelBasis = ref(26);
const panelMaxWidth = ref(420);
const mapFlexGrow = ref(1);
const topbarHeightPx = ref(80);

const LS_FONT_USER = "rb_screen_font_user";
const LS_LEFT_W = "rb_screen_left_px";
const LS_RIGHT_W = "rb_screen_right_px";
const LS_CUSTOM_LAYOUT = "rb_screen_layout_custom";
const LS_LEFT_TOP_RATIO = "rb_screen_left_top_ratio";
const LS_RIGHT_TOP_RATIO = "rb_screen_right_top_ratio";

const fontUserScale = ref(1);
try {
  const s = localStorage.getItem(LS_FONT_USER);
  if (s) {
    const n = Number.parseFloat(s);
    if (Number.isFinite(n) && n >= 0.75 && n <= 1.45) fontUserScale.value = n;
  }
} catch {
  // ignore
}

const layoutElRef = ref<HTMLElement | null>(null);
const leftPanelEl = ref<HTMLElement | null>(null);
const rightPanelEl = ref<HTMLElement | null>(null);
/** 左右栏拖拽：可拉得更窄/更宽，适配超大屏 */
const PANEL_W_MIN = 200;
const PANEL_W_MAX = 1000;
const PANEL_TOP_RATIO_MIN = 24;
const PANEL_TOP_RATIO_MAX = 76;
const leftTopRatio = ref(40);
const rightTopRatio = ref(40);
const useCustomPanelWidths = ref(false);
const leftPanelWidthPx = ref(380);
const rightPanelWidthPx = ref(380);
try {
  if (localStorage.getItem(LS_CUSTOM_LAYOUT) === "1") {
    useCustomPanelWidths.value = true;
    const lw = Number.parseInt(localStorage.getItem(LS_LEFT_W) ?? "", 10);
    const rw = Number.parseInt(localStorage.getItem(LS_RIGHT_W) ?? "", 10);
    if (Number.isFinite(lw) && lw >= PANEL_W_MIN && lw <= PANEL_W_MAX) leftPanelWidthPx.value = lw;
    if (Number.isFinite(rw) && rw >= PANEL_W_MIN && rw <= PANEL_W_MAX) rightPanelWidthPx.value = rw;
  }
  const ltr = Number.parseFloat(localStorage.getItem(LS_LEFT_TOP_RATIO) ?? "");
  const rtr = Number.parseFloat(localStorage.getItem(LS_RIGHT_TOP_RATIO) ?? "");
  if (Number.isFinite(ltr) && ltr >= PANEL_TOP_RATIO_MIN && ltr <= PANEL_TOP_RATIO_MAX) leftTopRatio.value = ltr;
  if (Number.isFinite(rtr) && rtr >= PANEL_TOP_RATIO_MIN && rtr <= PANEL_TOP_RATIO_MAX) rightTopRatio.value = rtr;
} catch {
  // ignore
}

const leftPanelStyle = computed(() => ({
  flex: `0 0 ${leftPanelWidthPx.value}px`,
  width: `${leftPanelWidthPx.value}px`,
  minWidth: `${PANEL_W_MIN}px`,
  maxWidth: `${PANEL_W_MAX}px`,
}));

const rightPanelStyle = computed(() => ({
  flex: `0 0 ${rightPanelWidthPx.value}px`,
  width: `${rightPanelWidthPx.value}px`,
  minWidth: `${PANEL_W_MIN}px`,
  maxWidth: `${PANEL_W_MAX}px`,
}));
const leftTopPanelStyle = computed(() => ({
  flex: `0 0 ${leftTopRatio.value}%`,
  maxHeight: `${leftTopRatio.value}%`,
}));
const rightTopPanelStyle = computed(() => ({
  flex: `0 0 ${rightTopRatio.value}%`,
  maxHeight: `${rightTopRatio.value}%`,
}));

let resizeDrag: { which: "left" | "right"; startX: number; startLeftW: number; startRightW: number } | null = null;
let resizeVDrag:
  | {
      which: "left" | "right";
      startY: number;
      startTopRatio: number;
      panelHeight: number;
    }
  | null = null;

function persistLayout() {
  try {
    localStorage.setItem(LS_CUSTOM_LAYOUT, "1");
    localStorage.setItem(LS_LEFT_W, String(Math.round(leftPanelWidthPx.value)));
    localStorage.setItem(LS_RIGHT_W, String(Math.round(rightPanelWidthPx.value)));
    localStorage.setItem(LS_LEFT_TOP_RATIO, String(Number(leftTopRatio.value.toFixed(2))));
    localStorage.setItem(LS_RIGHT_TOP_RATIO, String(Number(rightTopRatio.value.toFixed(2))));
  } catch {
    // ignore
  }
}

/** 清除本机保存的左右栏宽，恢复为百分比自适应布局 */
function resetDefaultLayout() {
  try {
    localStorage.removeItem(LS_CUSTOM_LAYOUT);
    localStorage.removeItem(LS_LEFT_W);
    localStorage.removeItem(LS_RIGHT_W);
    localStorage.removeItem(LS_LEFT_TOP_RATIO);
    localStorage.removeItem(LS_RIGHT_TOP_RATIO);
  } catch {
    // ignore
  }
  useCustomPanelWidths.value = false;
  leftPanelWidthPx.value = 380;
  rightPanelWidthPx.value = 380;
  leftTopRatio.value = 40;
  rightTopRatio.value = 40;
  updateViewportAdaptiveVars();
  nextTick(() => {
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  });
}

function startResizeLeftVertical(ev: MouseEvent) {
  const panel = leftPanelEl.value;
  if (!panel) return;
  const panelH = panel.getBoundingClientRect().height;
  if (panelH <= 0) return;
  resizeVDrag = {
    which: "left",
    startY: ev.clientY,
    startTopRatio: leftTopRatio.value,
    panelHeight: panelH,
  };
  const onMove = (e: MouseEvent) => {
    if (!resizeVDrag || resizeVDrag.which !== "left") return;
    const dy = e.clientY - resizeVDrag.startY;
    const deltaRatio = (dy / Math.max(1, resizeVDrag.panelHeight)) * 100;
    const next = Math.max(PANEL_TOP_RATIO_MIN, Math.min(PANEL_TOP_RATIO_MAX, resizeVDrag.startTopRatio + deltaRatio));
    leftTopRatio.value = next;
    radarChart?.resize();
  };
  const onUp = () => {
    resizeVDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    radarChart?.resize();
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}

function startResizeRightVertical(ev: MouseEvent) {
  const panel = rightPanelEl.value;
  if (!panel) return;
  const panelH = panel.getBoundingClientRect().height;
  if (panelH <= 0) return;
  resizeVDrag = {
    which: "right",
    startY: ev.clientY,
    startTopRatio: rightTopRatio.value,
    panelHeight: panelH,
  };
  const onMove = (e: MouseEvent) => {
    if (!resizeVDrag || resizeVDrag.which !== "right") return;
    const dy = e.clientY - resizeVDrag.startY;
    const deltaRatio = (dy / Math.max(1, resizeVDrag.panelHeight)) * 100;
    const next = Math.max(PANEL_TOP_RATIO_MIN, Math.min(PANEL_TOP_RATIO_MAX, resizeVDrag.startTopRatio + deltaRatio));
    rightTopRatio.value = next;
    pieChart?.resize();
  };
  const onUp = () => {
    resizeVDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    pieChart?.resize();
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}

function startResizeLeft(ev: MouseEvent) {
  const el = layoutElRef.value;
  if (!el) return;
  resizeDrag = {
    which: "left",
    startX: ev.clientX,
    startLeftW: leftPanelWidthPx.value,
    startRightW: rightPanelWidthPx.value,
  };
  useCustomPanelWidths.value = true;

  const onMove = (e: MouseEvent) => {
    if (!resizeDrag || resizeDrag.which !== "left") return;
    const dx = e.clientX - resizeDrag.startX;
    let nw = resizeDrag.startLeftW + dx;
    nw = Math.max(PANEL_W_MIN, Math.min(PANEL_W_MAX, nw));
    leftPanelWidthPx.value = nw;
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  };
  const onUp = () => {
    resizeDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}

function startResizeRight(ev: MouseEvent) {
  const el = layoutElRef.value;
  if (!el) return;
  resizeDrag = {
    which: "right",
    startX: ev.clientX,
    startLeftW: leftPanelWidthPx.value,
    startRightW: rightPanelWidthPx.value,
  };
  useCustomPanelWidths.value = true;

  const onMove = (e: MouseEvent) => {
    if (!resizeDrag || resizeDrag.which !== "right") return;
    const dx = resizeDrag.startX - e.clientX;
    let nw = resizeDrag.startRightW + dx;
    nw = Math.max(PANEL_W_MIN, Math.min(PANEL_W_MAX, nw));
    rightPanelWidthPx.value = nw;
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  };
  const onUp = () => {
    resizeDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}

watch(fontUserScale, (v) => {
  try {
    localStorage.setItem(LS_FONT_USER, String(v));
  } catch {
    // ignore
  }
  nextTick(() => {
    radarChart?.resize();
    pieChart?.resize();
    if (pieChart) updatePie(state.attack_stats);
    if (mapChart) refreshMapSeriesOnly(currentMapType);
  });
});

watch(uiScale, () => {
  nextTick(() => {
    if (pieChart) updatePie(state.attack_stats);
  });
});

const screenRootStyle = computed<Record<string, string>>(() => ({
  "--ui-scale": String(uiScale.value),
  "--font-user": String(fontUserScale.value),
  "--panel-basis": `${panelBasis.value}%`,
  "--panel-max-width": `${panelMaxWidth.value}px`,
  "--map-flex-grow": String(mapFlexGrow.value),
  "--topbar-height": `${topbarHeightPx.value}px`,
}));

function updateViewportAdaptiveVars() {
  const w = window.innerWidth;
  const h = window.innerHeight;

  // 2K / 4K 大屏下提升字体和控件尺寸，同时压缩中间地图占比。
  if (w >= 3400 || h >= 1800) {
    uiScale.value = 1.34;
    if (!useCustomPanelWidths.value) {
      panelBasis.value = 30;
      panelMaxWidth.value = 620;
    }
    mapFlexGrow.value = 0.9;
    topbarHeightPx.value = 100;
    return;
  }
  if (w >= 2500 || h >= 1400) {
    uiScale.value = 1.18;
    if (!useCustomPanelWidths.value) {
      panelBasis.value = 28;
      panelMaxWidth.value = 520;
    }
    mapFlexGrow.value = 0.95;
    topbarHeightPx.value = 90;
    return;
  }
  uiScale.value = 1;
  if (!useCustomPanelWidths.value) {
    panelBasis.value = 26;
    panelMaxWidth.value = 420;
  }
  mapFlexGrow.value = 1;
  topbarHeightPx.value = 80;
}

const wsSynced = ref(false);
const isReplaying = ref(false);
const showMapSettings = ref(false);

// === 回放控制 ===
const replayLoading = ref(false);
const replayReady = ref(false);
const replayEvents = ref<any[]>([]);
const replayCursor = ref(0);
const replayPlaying = ref(false);
const replaySpeed = ref(8); // 速度档位：越大间隔越短
let replayTimer: number | undefined;
let resizeHandler: (() => void) | undefined;

const scorePulse = reactive<Record<number, number>>({});

const leaderTrackStyle = computed(() => {
  return {
    transform: `translateY(${-leaderboardShift.value * leaderItemStepPx.value}px)`,
  };
});

function getTeamRank(teamId: number | undefined): number {
  if (!teamId) return 0;
  const arr = leaderboardSorted.value;
  const idx = arr.findIndex((t) => t.id === teamId);
  return idx === -1 ? 0 : idx + 1;
}

function pulseTeamScore(teamId: number) {
  scorePulse[teamId] = Date.now();
  window.setTimeout(() => {
    delete scorePulse[teamId];
  }, 900);
}

// 大屏默认不展示回放控制台（回放仍可通过 admin 下发控制事件工作）。
const showReplayUI = computed(() => false);

async function setMapMode(mode: "china" | "taizhou") {
  if (!matchId) return;
  if (isReplaying.value) return;
  // 通过后端指令切换地图底图（并由后端广播事件给所有大屏）。
  await fetch(`${apiBaseHttp}/api/matches/${matchId}/command`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders },
    body: JSON.stringify({
      event_type: "switch_map",
      data: { map_type: mode },
    }),
  });
}

const alertVisible = ref(false);
const alertKind = ref<"system" | "attack" | "warn">("warn");
const alertText = ref("系统预警信息");
let alertTimer: number | undefined;
const bgmAudioEl = ref<HTMLAudioElement | null>(null);
const successSfxAudioEl = ref<HTMLAudioElement | null>(null);
let audioUnlockBound = false;
const bgmManualPlaying = ref(false);

const state = reactive<MatchStateDTO>({
  screen_title: "实战化红蓝对抗演练指挥中心",
  screen_organizer: "",
  screen_supporter: "",
  bgm_url: "",
  bgm_enabled: false,
  success_sfx_url: "",
  success_sfx_enabled: false,
  map_type: "china",
  leaderboard_visible: true,
  teams: [],
  attack_stats: attackStatsDefault,
  panels: { "panel-leaderboard": true },
});

const leaderboardSorted = computed(() => {
  return [...state.teams].sort((a, b) => b.score - a.score);
});

const LEADERBOARD_PAGE_SIZE = 8;
const leaderboardStartIndex = ref(0);
const leaderboardShift = ref(0); // 0 or 1, 每次上移 1 条
const leaderTrackAnimating = ref(false);
const leaderItemStepPx = ref(56); // 默认估算（会在 mounted 后测量一次）
const leaderViewportEl = ref<HTMLElement | null>(null);

const leaderboardWindow = computed(() => {
  const arr = leaderboardSorted.value;
  if (!arr.length) return [];

  const n = arr.length;
  const visibleCount = Math.min(LEADERBOARD_PAGE_SIZE, n);

  // 不足一屏：不滚动，直接显示全部
  if (n <= visibleCount) return arr;

  // 滚动需要多渲染 1 条，做无缝上移
  const start = ((leaderboardStartIndex.value % n) + n) % n;
  const out: any[] = [];
  for (let i = 0; i < visibleCount + 1; i++) {
    out.push(arr[(start + i) % n]);
  }
  return out;
});

let leaderboardTimer: number | undefined;
function startLeaderboardLoop() {
  if (leaderboardTimer) window.clearInterval(leaderboardTimer);
  leaderboardTimer = window.setInterval(() => {
    const n = leaderboardSorted.value.length;
    if (n <= LEADERBOARD_PAGE_SIZE) return;
    if (leaderTrackAnimating.value) return;

    leaderTrackAnimating.value = true;
    leaderboardShift.value = 1;

    // 动画完成后：推进起点并瞬间复位位移，形成无缝滚动
    window.setTimeout(() => {
      leaderboardStartIndex.value = (leaderboardStartIndex.value + 1) % n;
      leaderTrackAnimating.value = false;
      leaderboardShift.value = 0;
    }, 520);
  }, 2600);
}

const redTopTeam = computed(() => {
  const reds = state.teams.filter((t) => t.type === "red");
  return reds.sort((a, b) => b.score - a.score)[0];
});

const blueTopTeam = computed(() => {
  const blues = state.teams.filter((t) => t.type === "blue");
  return blues.sort((a, b) => b.score - a.score)[0];
});

const topAttackStat = computed(() => state.attack_stats?.[0]);

const terminalScrollEl = ref<HTMLElement | null>(null);
const LS_TERMINAL_LOGS = "rb_screen_terminal_logs_v1";

const terminalLines = ref<string[]>([]);
const terminalLogQueue: string[] = [];
const LOG_FLUSH_INTERVAL_MS = 520;
const MAX_TERMINAL_LOGS = 90;
const terminalStartIndex = ref(0);
const terminalShift = ref(0);
const terminalTrackAnimating = ref(false);
const terminalItemStepPx = ref(22);

/** 不写入终端的噪声日志（连接/同步提示） */
const SKIP_LOG_RULES: RegExp[] = [
  /^\[?SYSTEM\]\s*WebSocket\s*实战链路建立成功/,
  /^\[?SYSTEM\]\s*已同步当前态势状态/,
  /^\[?SYSTEM\]\s*WS\s*链路断开，尝试重连/,
];

const terminalVisibleLines = ref(10);
const terminalTrackStyle = computed(() => ({ transform: `translateY(${-terminalShift.value * terminalItemStepPx.value}px)` }));
const terminalWindow = computed(() => {
  const arr = trimTrailingEmptyLines(terminalLines.value);
  const n = arr.length;
  if (!n) return [];
  const visibleCount = Math.max(1, terminalVisibleLines.value);
  if (n <= visibleCount) return arr;
  const start = ((terminalStartIndex.value % n) + n) % n;
  const out: string[] = [];
  for (let i = 0; i < visibleCount + 1; i++) out.push(arr[(start + i) % n]);
  return out;
});

let terminalLoopTimer: number | undefined;
function startTerminalLoop() {
  if (terminalLoopTimer) window.clearInterval(terminalLoopTimer);
  terminalLoopTimer = window.setInterval(() => {
    const n = trimTrailingEmptyLines(terminalLines.value).length;
    const visibleCount = Math.max(1, terminalVisibleLines.value);
    if (n <= visibleCount) return;
    if (terminalTrackAnimating.value) return;
    terminalTrackAnimating.value = true;
    terminalShift.value = 1;
    window.setTimeout(() => {
      terminalStartIndex.value = (terminalStartIndex.value + 1) % n;
      terminalTrackAnimating.value = false;
      terminalShift.value = 0;
    }, 520);
  }, 2400);
}

function trimTrailingEmptyLines(lines: string[]) {
  const out = [...lines];
  while (out.length && out[out.length - 1] === "") out.pop();
  return out;
}

function ensureTerminalFilled() {
  const target = terminalVisibleLines.value;
  if (!target || target <= 0) return;
  // 只在尾部补空行，让框尽量“填满”，不影响顶部真实日志
  if (terminalLines.value.length < target) {
    const missing = target - terminalLines.value.length;
    terminalLines.value = [...terminalLines.value, ...Array(missing).fill("")];
  }
}

function persistTerminalLogsToSession() {
  if (!matchId) return;
  try {
    const trimmed = trimTrailingEmptyLines(terminalLines.value);
    sessionStorage.setItem(
      `${LS_TERMINAL_LOGS}_${matchId}`,
      JSON.stringify(trimmed.slice(-MAX_TERMINAL_LOGS))
    );
  } catch {
    // ignore
  }
}

function loadTerminalLogsFromSession() {
  if (!matchId) {
    terminalLines.value = ["[SYSTEM] 演练监控系统初始化完成..."];
    return;
  }
  try {
    const raw = sessionStorage.getItem(`${LS_TERMINAL_LOGS}_${matchId}`);
    if (raw) {
      const arr = JSON.parse(raw) as unknown;
      if (Array.isArray(arr) && arr.length && arr.every((x) => typeof x === "string")) {
        terminalLines.value = arr
          .filter((line) => !SKIP_LOG_RULES.some((re) => re.test(line)))
          .slice(-MAX_TERMINAL_LOGS);
        ensureTerminalFilled();
        persistTerminalLogsToSession();
        return;
      }
    }
  } catch {
    // ignore
  }
  terminalLines.value = ["[SYSTEM] 演练监控系统初始化完成..."];
  ensureTerminalFilled();
}

function flushTerminalQueue() {
  if (!terminalLogQueue.length) return;
  const batch = terminalLogQueue.splice(0, terminalLogQueue.length);
  // 用新日志覆盖尾部空行，避免一直向下推空白
  const trimmed = trimTrailingEmptyLines(terminalLines.value);
  terminalLines.value = [...trimmed, ...batch].slice(-MAX_TERMINAL_LOGS);
  terminalStartIndex.value = 0;
  terminalTrackAnimating.value = false;
  terminalShift.value = 0;
  ensureTerminalFilled();
  persistTerminalLogsToSession();
}

let terminalFlushTimer: number | undefined;
function pushLog(line: string) {
  if (SKIP_LOG_RULES.some((re) => re.test(line))) return;
  terminalLogQueue.push(line);
  // 队列无限增长会导致“突然爆量”，这里做兜底截断
  if (terminalLogQueue.length > 300) {
    terminalLogQueue.splice(0, terminalLogQueue.length - 300);
  }
  if (!terminalFlushTimer) {
    terminalFlushTimer = window.setInterval(flushTerminalQueue, LOG_FLUSH_INTERVAL_MS);
  }
}

function logLineClass(line: string) {
  if (line.startsWith("[SYSTEM]")) return "log-system";
  if (line.startsWith("[JUDGE]")) return "log-judge";
  if (line.startsWith("[")) return "log-event";
  return "log-normal";
}
function triggerAlert(message: string, kind: "system" | "attack" | "warn" = "warn") {
  alertKind.value = kind;
  alertText.value = message;
  alertVisible.value = true;
  if (alertTimer) window.clearTimeout(alertTimer);
  alertTimer = window.setTimeout(() => (alertVisible.value = false), 5000);
}

function resolveMediaURL(raw: string | undefined) {
  const s = String(raw ?? "").trim();
  if (!s) return "";
  if (/^https?:\/\//i.test(s)) return s;
  if (s.startsWith("/")) return `${apiBaseHttp.replace(/\/$/, "")}${s}`;
  return s;
}

function syncBGMPlayback() {
  const el = bgmAudioEl.value;
  if (!el) return;
  const url = resolveMediaURL(state.bgm_url);
  const enabled = !!state.bgm_enabled && !!url;
  if (!enabled) {
    el.pause();
    bgmManualPlaying.value = false;
    return;
  }
  if (el.src !== url) el.src = url;
  el.volume = 0.32;
  void el.play().catch(() => {
    // 浏览器自动播放策略可能拦截，等待用户手势解锁后重试
  });
  bgmManualPlaying.value = !el.paused;
}

function playSuccessSfx() {
  if (!state.success_sfx_enabled) return;
  const url = resolveMediaURL(state.success_sfx_url);
  if (!url) return;
  const el = successSfxAudioEl.value;
  if (!el) return;
  if (el.src !== url) el.src = url;
  el.currentTime = 0;
  el.volume = 0.9;
  void el.play().catch(() => {
    // 自动播放限制场景下可能失败，保持静默
  });
}

function bindAudioUnlock() {
  if (audioUnlockBound) return;
  audioUnlockBound = true;
  const tryUnlock = () => {
    syncBGMPlayback();
    window.removeEventListener("pointerdown", tryUnlock);
    window.removeEventListener("keydown", tryUnlock);
    audioUnlockBound = false;
  };
  window.addEventListener("pointerdown", tryUnlock, { once: true });
  window.addEventListener("keydown", tryUnlock, { once: true });
}

async function toggleBGMManually() {
  const el = bgmAudioEl.value;
  if (!el) return;
  if (bgmManualPlaying.value) {
    el.pause();
    bgmManualPlaying.value = false;
    return;
  }
  const url = resolveMediaURL(state.bgm_url);
  if (!url) {
    triggerAlert("未配置背景音乐 URL", "warn");
    return;
  }
  if (el.src !== url) el.src = url;
  el.loop = true;
  el.volume = 0.32;
  try {
    await el.play();
    bgmManualPlaying.value = true;
  } catch {
    triggerAlert("浏览器阻止自动播放，请先点击页面后再试", "warn");
  }
}

// ==== ECharts 实例 ====
const mapEl = ref<HTMLElement | null>(null);
const radarEl = ref<HTMLElement | null>(null);
const pieEl = ref<HTMLElement | null>(null);

let mapChart: echarts.ECharts | null = null;
let radarChart: echarts.ECharts | null = null;
let pieChart: echarts.ECharts | null = null;

let currentMapType: "china" | "taizhou" = "china";
type MapLineMeta = {
  sourceCoord: [number, number];
  targetCoord: [number, number];
  lineColor: string;
  lineWidth: number;
  lineDash?: "dashed";
  effectPeriod?: number;
  effectTrail?: number;
  effectSymbolSize?: number;
  createdAt: number;
  ttlMs: number;
};
type MapHitMarker = {
  coord: [number, number];
  cityName: string;
  status: AttackStatus;
  role?: "source" | "target";
  createdAt: number;
  ttlMs: number;
};
type MapRegionFlash = {
  cityName: string;
  status: AttackStatus;
  createdAt: number;
  ttlMs: number;
};
let mapLineEntries: MapLineMeta[] = [];
let mapHitMarkers: MapHitMarker[] = [];
let mapRegionFlashes: MapRegionFlash[] = [];
let mapLineRefreshTimer: number | undefined;
let mapHasGeo = true;
const mapImpactActive = ref(false);
let mapImpactTimer: number | undefined;

// 事件驱动的“脉冲点”（用于让飞线起止点更有层次）
const cityPulse = reactive<Record<string, number>>({});
const CITY_PULSE_LIFETIME_MS = 3500;
const CITY_PULSE_MAX = 8;
const MAP_LINE_TTL_MS = 18000;
const MAP_LINE_MAX = 90;
const MAP_HIT_MARK_TTL_MS = 7600;
const MAP_HIT_MARK_MAX = 60;
const MAP_REGION_FLASH_TTL_MS = 1800;

function buildActiveMapLines(nowTs: number = Date.now()) {
  mapLineEntries = mapLineEntries.filter((it) => nowTs - it.createdAt <= it.ttlMs);
  return mapLineEntries.map((it) => {
    const age = Math.max(0, Math.min(1, (nowTs - it.createdAt) / it.ttlMs));
    const opacity = 0.88 - age * 0.55;
    const width = Math.max(1.1, it.lineWidth - age * 0.9);
    const lineStyle: any = {
      color: it.lineColor,
      width,
      opacity,
      curveness: 0.26,
    };
    if (it.lineDash) lineStyle.type = it.lineDash;
    // 飞线样式仅放在 lineStyle；箭头由 series.lines 的 effect + symbol 统一绘制（避免与 geo 下逐条 effect 冲突）
    return {
      coords: [it.sourceCoord, it.targetCoord],
      lineStyle,
    };
  });
}

function buildHitMarkerSeriesData(nowTs: number = Date.now()) {
  mapHitMarkers = mapHitMarkers.filter((it) => nowTs - it.createdAt <= it.ttlMs);

  const successData: any[] = [];
  const defenseData: any[] = [];
  const traceData: any[] = [];
  const sourceSuccessData: any[] = [];

  for (const it of mapHitMarkers) {
    const age = Math.max(0, Math.min(1, (nowTs - it.createdAt) / it.ttlMs));
    const opacity = 0.95 - age * 0.65;
    const item = {
      name: it.cityName,
      value: it.coord,
      itemStyle: { opacity },
    };
    if (it.status === "success" && it.role === "source") sourceSuccessData.push(item);
    else if (it.status === "success") successData.push(item);
    else if (it.status === "defense_success") defenseData.push(item);
    else if (it.status === "trace_success") traceData.push(item);
  }

  return { successData, defenseData, traceData, sourceSuccessData };
}

function buildMapFlashRegions(nowTs: number = Date.now()) {
  mapRegionFlashes = mapRegionFlashes.filter((it) => nowTs - it.createdAt <= it.ttlMs);
  return mapRegionFlashes.map((it) => {
    const age = Math.max(0, Math.min(1, (nowTs - it.createdAt) / it.ttlMs));
    const opacity = 0.72 - age * 0.5;
    const isSuccess = it.status === "success";
    const baseArea = isSuccess ? `rgba(255, 76, 76, ${opacity.toFixed(3)})` : `rgba(0, 183, 255, ${opacity.toFixed(3)})`;
    const baseBorder = isSuccess ? "rgba(255, 184, 184, 0.95)" : "rgba(150, 235, 255, 0.92)";
    return {
      name: it.cityName,
      itemStyle: {
        areaColor: baseArea,
        borderColor: baseBorder,
        borderWidth: 2,
      },
      emphasis: {
        itemStyle: {
          areaColor: baseArea,
          borderColor: baseBorder,
          borderWidth: 2.2,
        },
      },
    };
  });
}

function triggerMapImpact(status: AttackStatus) {
  if (status !== "success" && status !== "defense_success" && status !== "trace_success") return;
  mapImpactActive.value = false;
  if (mapImpactTimer) window.clearTimeout(mapImpactTimer);
  // 先清零后下一帧再打开，保证连续命中也能重新触发动画
  requestAnimationFrame(() => {
    mapImpactActive.value = true;
    mapImpactTimer = window.setTimeout(() => {
      mapImpactActive.value = false;
    }, 620);
  });
}

function updateRadar(teams: TeamDTO[]) {
  if (!radarChart) return;
  const indicators = [
    { name: "漏洞挖掘", max: 100 },
    { name: "内网渗透", max: 100 },
    { name: "权限维持", max: 100 },
    { name: "隐蔽穿透", max: 100 },
    { name: "情报搜集", max: 100 },
    { name: "响应速度", max: 100 },
  ];

  // 将“综合战力评估”的雷达维度从固定值，改为随攻击统计动态变化。
  // attack_type -> 雷达维度映射：
  // 0 漏洞挖掘 / 1 内网渗透 / 2 权限维持 / 3 隐蔽穿透 / 4 情报搜集 / 5 响应速度
  const dimIndexByAttackType: Record<string, number> = {
    // 漏洞挖掘（Web 漏洞利用）
    "Weblogic反序列化": 0,
    "反序列化漏洞": 0,
    "SQL盲注": 0,
    "SQL注入(盲注/报错注入)": 0,
    "Struts2 OGNL注入": 0,
    "模板注入/表达式注入": 0,
    "命令执行(RCE)": 0,
    "命令注入(RCE)": 0,
    "路径遍历任意文件读取": 0,
    "目录遍历任意文件读取": 0,
    "任意文件上传(上传Webshell)": 0,
    "文件包含漏洞(LFI/RFI)": 0,
    "XXE外部实体注入": 0,

    // 内网渗透（Web 打洞到内网）
    "SSRF内网探测": 1,
    "远程服务登录(RDP/SSH)": 1,

    // 权限维持（认证/授权绕过与持久化）
    "业务逻辑漏洞(认证/状态绕过)": 2,
    "未授权访问(身份/权限校验缺失)": 2,
    "授权绕过(越权访问)": 2,
    "0day提权": 2,
    "票据传递(Pass-the-Ticket)": 2,
    "哈希传递(Pass-the-Hash)": 2,
    "后门/持久化(Webshell/计划任务)": 2,

    // 隐蔽穿透（非纯 Web 也保留兼容）
    "横向移动(SMB)": 3,
    "隐蔽隧道(DNS/C2隧道)": 3,

    // 情报搜集（页面/会话信息窃取）
    "XSS跨站脚本(窃取Cookie)": 4,
    "CSRF跨站请求伪造": 4,
    "数据窃取/外传(Exfiltration)": 4,

    // 响应速度（爆破类与高频尝试）
    "弱口令爆破": 5,
    "账号撞库/凭证填充": 5,
    "密码喷洒(Password Spraying)": 5,
    "DDoS攻击": 5,
  };
  const hashToDim = (s: string) => {
    let h = 0;
    for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) % 6;
    return h;
  };

  const axisCounts: number[] = Array.from({ length: 6 }, () => 0);
  const attackStats = state.attack_stats ?? [];
  for (const s of attackStats) {
    const dim = dimIndexByAttackType[s.name] ?? hashToDim(s.name);
    const v = Number(s.value ?? 0);
    if (!Number.isFinite(v) || v <= 0) continue;
    axisCounts[dim] += v;
  }
  const maxAxis = Math.max(...axisCounts);

  const series = teams.slice(0, 2).map((t, idx) => {
    // 评分因子：让同一攻击分布下，两队形状也会有差异
    const scoreFactor = Math.max(0, Math.min(1, t.score / 20000));
    const scoreBoost = scoreFactor * 25;

    const value = axisCounts.map((c) => {
      const base = maxAxis > 0 ? (c / maxAxis) * 75 : 0;
      const v = Math.round(base * 0.85 + scoreBoost);
      return Math.max(0, Math.min(100, v));
    });

    return {
      value,
      name: t.name,
      areaStyle: { color: idx === 0 ? "rgba(255,42,42,0.28)" : "rgba(255,193,7,0.28)" },
    };
  });

  radarChart.setOption({
    color: ["#ff2a2a", "#ffc107", "#00f3ff"],
    radar: {
      indicator: indicators,
      shape: "polygon",
      radius: "65%",
      axisName: { color: "#00f3ff" },
      splitLine: {
        lineStyle: {
          color: [
            "rgba(0,243,255,0.1)",
            "rgba(0,243,255,0.2)",
            "rgba(0,243,255,0.4)",
            "rgba(0,243,255,0.6)",
          ],
        },
      },
      splitArea: { show: false },
      axisLine: { lineStyle: { color: "rgba(0,243,255,0.5)" } },
    },
    series: [{ type: "radar", data: series }],
  });
}

function pieTextPx(base: number) {
  return Math.max(10, Math.round(base * uiScale.value * fontUserScale.value));
}

function updatePie(attackStats: { name: string; value: number }[]) {
  if (!pieChart) return;
  const dataList = attackStats ?? [];
  const g = (echarts as any).graphic;
  const mkGradient = (c1: string, c2: string) =>
    g?.LinearGradient
      ? new g.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: c1 },
          { offset: 1, color: c2 },
        ])
      : c1;

  const palette = [
    ["#22d3ee", "#0891b2"],
    ["#38bdf8", "#2563eb"],
    ["#a78bfa", "#7c3aed"],
    ["#f472b6", "#db2777"],
    ["#fb923c", "#ea580c"],
    ["#4ade80", "#16a34a"],
  ];

  if (!dataList.length) {
    pieChart.setOption({
      backgroundColor: "transparent",
      title: {
        text: "暂无战术数据",
        left: "center",
        top: "center",
        textStyle: {
          color: "rgba(148,163,184,0.9)",
          fontSize: pieTextPx(13),
          fontWeight: 600,
        },
      },
      legend: { show: false },
      tooltip: { show: false },
      series: [
        {
          type: "pie",
          radius: ["48%", "68%"],
          center: ["50%", "48%"],
          silent: true,
          data: [{ value: 1, name: "", itemStyle: { color: "rgba(15,23,42,0.5)", borderColor: "rgba(56,189,248,0.15)", borderWidth: 1 } }],
          label: { show: false },
          emphasis: { disabled: true },
        },
      ],
    });
    return;
  }

  const sorted = [...dataList].sort((a, b) => b.value - a.value);
  const escapeHtml = (s: string) =>
    s.replace(/[&<>"']/g, (ch) => {
      switch (ch) {
        case "&":
          return "&amp;";
        case "<":
          return "&lt;";
        case ">":
          return "&gt;";
        case '"':
          return "&quot;";
        case "'":
          return "&#39;";
        default:
          return ch;
      }
    });

  const total = sorted.reduce((s, x) => s + Number(x.value ?? 0), 0);

  const seriesData = sorted.map((d, i) => {
    const [a, b] = palette[i % palette.length];
    return {
      name: d.name,
      value: d.value,
      itemStyle: {
        color: mkGradient(a, b),
        borderRadius: 8,
        borderColor: "#0a1628",
        borderWidth: 2,
        shadowBlur: 12,
        shadowColor: "rgba(56,189,248,0.15)",
      },
    };
  });

  pieChart.setOption({
    backgroundColor: "transparent",
    color: palette.map((p) => p[0]),
    title: {
      text: "战术分布",
      subtext: total > 0 ? `合计 ${total} 次` : "",
      left: "center",
      top: "8%",
      textStyle: {
        color: "rgba(226,232,240,0.95)",
        fontSize: pieTextPx(14),
        fontWeight: 700,
      },
      subtextStyle: {
        color: "rgba(148,163,184,0.9)",
        fontSize: pieTextPx(11),
      },
    },
    tooltip: {
      trigger: "item",
      backgroundColor: "rgba(15,23,42,0.92)",
      borderColor: "rgba(56,189,248,0.35)",
      textStyle: { color: "#e2e8f0", fontSize: pieTextPx(12) },
      formatter: (p: any) =>
        `${escapeHtml(String(p.name ?? ""))}<br/><span style="opacity:.85">次数</span>：<b>${p.value}</b>（${p.percent}%）`,
    },
    legend: {
      type: "scroll",
      orient: "horizontal",
      bottom: "2%",
      left: "center",
      width: "88%",
      pageIconColor: "#22d3ee",
      pageTextStyle: { color: "#94a3b8" },
      textStyle: {
        color: "rgba(203,213,225,0.92)",
        fontSize: pieTextPx(10),
      },
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 10,
    },
    series: [
      {
        name: "战术",
        type: "pie",
        radius: ["42%", "62%"],
        center: ["50%", "52%"],
        avoidLabelOverlap: true,
        minShowLabelAngle: 4,
        itemStyle: {
          borderRadius: 8,
          borderColor: "#0a1628",
          borderWidth: 2,
        },
        label: {
          show: true,
          position: "outside",
          color: "rgba(241,245,249,0.95)",
          fontSize: pieTextPx(10),
          formatter: (p: any) => {
            const name = String(p.name ?? "");
            const short = name.length > 5 ? `${name.slice(0, 5)}…` : name;
            return `{n|${short}}\n{v|${p.percent}%}`;
          },
          rich: {
            n: { fontSize: pieTextPx(10), color: "rgba(226,232,240,0.95)", lineHeight: pieTextPx(14) },
            v: { fontSize: pieTextPx(9), color: "rgba(56,189,248,0.95)", fontWeight: 700 },
          },
        },
        labelLine: {
          length: 10,
          length2: 14,
          lineStyle: { color: "rgba(148,163,184,0.45)" },
        },
        emphasis: {
          scale: true,
          scaleSize: 6,
          itemStyle: {
            shadowBlur: 22,
            shadowColor: "rgba(56,189,248,0.35)",
          },
        },
        data: seriesData,
      },
    ],
  });
}

function getCityPulsePoints(mapType: "china" | "taizhou") {
  const currentCities = mapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const now = Date.now();
  const entries = Object.entries(cityPulse)
    .filter(([k, ts]) => currentCities[k] && now - ts <= CITY_PULSE_LIFETIME_MS)
    .sort((a, b) => b[1] - a[1])
    .slice(0, CITY_PULSE_MAX);
  return entries.map(([k]) => ({ name: k, value: currentCities[k] }));
}

function renderMap(hasMap = true, mapType: "china" | "taizhou" = currentMapType) {
  if (!mapChart) return;
  mapHasGeo = hasMap;
  const currentCities = mapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const points = Object.keys(currentCities).map((key) => ({ name: key, value: currentCities[key] }));

  const pulsePoints = getCityPulsePoints(mapType);
  const { successData, defenseData, traceData, sourceSuccessData } = buildHitMarkerSeriesData();
  const regions = buildMapFlashRegions();

  mapChart.setOption(
    {
      // 维持对比，但不过分抢眼
      backgroundColor: "rgba(3, 16, 42, 0.78)",
      geo: hasMap
        ? {
            map: mapType,
            roam: true,
            zoom: 1.2,
            // 市区县图（泰州）向下偏移：减少下部留白
            center: mapType === "taizhou" ? [120.0, 32.54] : [104.2, 36.0],
            regions,
            label:
              mapType === "taizhou"
                ? {
                    show: true,
                    color: "rgba(236, 250, 255, 0.98)",
                    fontSize: Math.max(18, Math.round(16 * uiScale.value * fontUserScale.value)),
                    fontWeight: 700,
                    textBorderColor: "rgba(1, 10, 26, 0.98)",
                    textBorderWidth: 3,
                    textShadowColor: "rgba(0, 243, 255, 0.45)",
                    textShadowBlur: 8,
                  }
                : { show: false },
            itemStyle: {
              areaColor: (echarts as any).graphic?.LinearGradient
                ? new (echarts as any).graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: "rgba(42, 95, 160, 0.95)" },
                    { offset: 0.45, color: "rgba(20, 58, 118, 0.9)" },
                    { offset: 1, color: "rgba(8, 26, 66, 0.82)" },
                  ])
                : "rgba(24, 70, 142, 0.9)",
              borderColor: "rgba(120, 225, 245, 0.78)",
              borderWidth: 1.35,
              shadowBlur: 24,
              shadowColor: "rgba(0, 243, 255, 0.36)",
              shadowOffsetY: 2,
            },
            emphasis: {
              itemStyle: {
                areaColor: "rgba(58, 122, 194, 0.96)",
                borderColor: "rgba(210, 248, 255, 0.95)",
                borderWidth: 1.8,
                shadowBlur: 18,
                shadowColor: "rgba(0, 243, 255, 0.5)",
              },
              label: { show: mapType === "taizhou", color: "#fff" },
            },
          }
        : null,
      series: [
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 2,
          rippleEffect: { brushType: "stroke", scale: 3.8 },
          label: { show: false },
          symbolSize: 6,
          itemStyle: {
            color: "rgba(0,243,255,0.55)",
            shadowBlur: 8,
            shadowColor: "rgba(0,243,255,0.35)",
          },
          data: points,
        },
        {
          type: "lines",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 3,
          polyline: false,
          // 起点无标记，终点静态箭头；effect 为沿轨迹运动的箭头流光
          symbol: ["none", "arrow"],
          symbolSize: [0, 12],
          effect: hasMap
            ? {
                show: true,
                period: 2.6,
                trailLength: 0.62,
                symbol: "arrow",
                symbolSize: 10,
                color: "rgba(255,255,255,0.92)",
                constantSpeed: 80,
              }
            : { show: false },
          lineStyle: { width: 1.6, opacity: 0.65, curveness: 0.26 },
          data: buildActiveMapLines(),
        },
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 4,
          rippleEffect: { brushType: "stroke", scale: 6 },
          // 不显示区县文字：与 geo 底图区县名重复；脉冲仅保留光效
          label: { show: false },
          symbolSize: 10,
          itemStyle: {
            color: "rgba(255,255,255,0.9)",
            shadowBlur: 14,
            shadowColor: "rgba(0,243,255,0.25)",
          },
          data: pulsePoints,
        },
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 5,
          rippleEffect: { brushType: "stroke", scale: 7 },
          symbol: "circle",
          symbolSize: 14,
          itemStyle: { color: "rgba(255,76,76,0.9)", shadowBlur: 18, shadowColor: "rgba(255,76,76,0.45)" },
          label: { show: false },
          data: successData,
        },
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 5,
          rippleEffect: { brushType: "stroke", scale: 5.2 },
          symbol: "pin",
          symbolSize: 16,
          itemStyle: { color: "rgba(0,183,255,0.95)", shadowBlur: 16, shadowColor: "rgba(0,183,255,0.4)" },
          label: { show: false },
          data: defenseData,
        },
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 5,
          rippleEffect: { brushType: "stroke", scale: 5.4 },
          symbol: "diamond",
          symbolSize: 13,
          itemStyle: { color: "rgba(255,209,102,0.95)", shadowBlur: 16, shadowColor: "rgba(255,209,102,0.38)" },
          label: { show: false },
          data: traceData,
        },
        {
          type: "effectScatter",
          coordinateSystem: hasMap ? "geo" : "cartesian2d",
          zlevel: 5,
          rippleEffect: { brushType: "stroke", scale: 4.8 },
          symbol: "diamond",
          symbolSize: 11,
          itemStyle: { color: "rgba(255,170,120,0.92)", shadowBlur: 14, shadowColor: "rgba(255,130,90,0.35)" },
          label: { show: false },
          data: sourceSuccessData,
        },
      ],
    },
    true
  );
}

function refreshMapSeriesOnly(mapType: "china" | "taizhou") {
  if (!mapChart) return;
  const hasMap = mapHasGeo;
  const currentCities = mapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const points = Object.keys(currentCities).map((key) => ({ name: key, value: currentCities[key] }));
  const pulsePoints = getCityPulsePoints(mapType);
  const { successData, defenseData, traceData, sourceSuccessData } = buildHitMarkerSeriesData();
  const regions = buildMapFlashRegions();

  mapChart.setOption({
    geo:
      hasMap
        ? {
            map: mapType,
            center: mapType === "taizhou" ? [120.0, 32.52] : [104.2, 36.0],
            regions: regions ?? [],
          }
        : {
            map: mapType,
            show: false,
            regions: [],
          },
    series: [
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 2,
        rippleEffect: { brushType: "stroke", scale: 3.8 },
        label: { show: false },
        symbolSize: 6,
        itemStyle: {
          color: "rgba(0,243,255,0.55)",
          shadowBlur: 8,
          shadowColor: "rgba(0,243,255,0.35)",
        },
        data: points,
      },
      {
        type: "lines",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 3,
        polyline: false,
        symbol: ["none", "arrow"],
        symbolSize: [0, 12],
        effect: hasMap
          ? {
              show: true,
              period: 2.6,
              trailLength: 0.62,
              symbol: "arrow",
              symbolSize: 10,
              color: "rgba(255,255,255,0.92)",
              constantSpeed: 80,
            }
          : { show: false },
        lineStyle: { width: 1.6, opacity: 0.65, curveness: 0.26 },
        data: buildActiveMapLines(),
      },
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 4,
        rippleEffect: { brushType: "stroke", scale: 6 },
        label: { show: false },
        symbolSize: 10,
        itemStyle: {
          color: "rgba(255,255,255,0.9)",
          shadowBlur: 14,
          shadowColor: "rgba(0,243,255,0.25)",
        },
        data: pulsePoints,
      },
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 5,
        rippleEffect: { brushType: "stroke", scale: 7 },
        symbol: "circle",
        symbolSize: 14,
        itemStyle: { color: "rgba(255,76,76,0.9)", shadowBlur: 18, shadowColor: "rgba(255,76,76,0.45)" },
        label: { show: false },
        data: successData,
      },
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 5,
        rippleEffect: { brushType: "stroke", scale: 5.2 },
        symbol: "pin",
        symbolSize: 16,
        itemStyle: { color: "rgba(0,183,255,0.95)", shadowBlur: 16, shadowColor: "rgba(0,183,255,0.4)" },
        label: { show: false },
        data: defenseData,
      },
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 5,
        rippleEffect: { brushType: "stroke", scale: 5.4 },
        symbol: "diamond",
        symbolSize: 13,
        itemStyle: { color: "rgba(255,209,102,0.95)", shadowBlur: 16, shadowColor: "rgba(255,209,102,0.38)" },
        label: { show: false },
        data: traceData,
      },
      {
        type: "effectScatter",
        coordinateSystem: hasMap ? "geo" : "cartesian2d",
        zlevel: 5,
        rippleEffect: { brushType: "stroke", scale: 4.8 },
        symbol: "diamond",
        symbolSize: 11,
        itemStyle: { color: "rgba(255,170,120,0.92)", shadowBlur: 14, shadowColor: "rgba(255,130,90,0.35)" },
        label: { show: false },
        data: sourceSuccessData,
      },
    ],
  });
}

async function loadMapData(mapType: "china" | "taizhou") {
  const url = `${apiBaseHttp}/api/geojson/${mapType}`;
  try {
    const res = await fetch(url, { headers: authHeaders });
    if (!res.ok) throw new Error(`geojson proxy status=${res.status}`);
    const geoJson = await res.json();
    echarts.registerMap(mapType, geoJson);
    mapLineEntries = [];
    mapHitMarkers = [];
  mapRegionFlashes = [];
    renderMap(true, mapType);
  } catch {
    mapLineEntries = [];
    mapHitMarkers = [];
    mapRegionFlashes = [];
    renderMap(false, mapType);
  }
}

function updateFromState() {
  // Map/图表都依赖 state。
  const nextMapType: "china" | "taizhou" = state.map_type === "taizhou" ? "taizhou" : "china";
  if (mapChart && nextMapType !== currentMapType) {
    currentMapType = nextMapType;
    loadMapData(currentMapType);
  }
  updateRadar(state.teams);
  updatePie(state.attack_stats);
}

const TEAM_COLORS = ["#ff2a2a", "#00f3ff", "#ffc107", "#ff007f", "#22c55e", "#a855f7"];
function getTeamLineColor(teamId: number | undefined, teamType?: string) {
  const tid = Number(teamId ?? 0);
  if (!Number.isFinite(tid) || tid <= 0) return teamType === "red" ? "#ff2a2a" : "#00f3ff";
  const idx = Math.abs(tid) % TEAM_COLORS.length;
  return TEAM_COLORS[idx];
}

function pickTeamSourceCity(teamId: number | undefined, cities: Record<string, [number, number]>) {
  const cityNames = Object.keys(cities);
  if (!cityNames.length) return "";
  const teamsSorted = [...state.teams].sort((a, b) => a.id - b.id);
  const idx = teamsSorted.findIndex((t) => t.id === Number(teamId ?? 0));
  if (idx < 0) return cityNames[Math.floor(Math.random() * cityNames.length)];
  return cityNames[idx % cityNames.length];
}

function updateMapLineByAttack(data: any) {
  const currentCities = currentMapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const cityNames = Object.keys(currentCities);

  const sourceMode = String(data.source_mode ?? "city");
  let sourceCity = "";
  if (sourceMode === "team") {
    sourceCity = pickTeamSourceCity(Number(data.source_team_id ?? data.team_id), currentCities);
  } else {
    sourceCity =
      data.source_city && currentCities[data.source_city]
        ? data.source_city
        : cityNames[Math.floor(Math.random() * cityNames.length)];
  }

  const lineColor = getTeamLineColor(data.team_id, data.team_type);
  let lineWidth = 1.8;
  const status = data.status as AttackStatus;
  let lineDash: string | undefined;
  const sourceCoord = currentCities[sourceCity];
  const targetCoord = currentCities[data.target_city];
  let drawSource = sourceCoord;
  let drawTarget = targetCoord;
  let drawColor = lineColor;
  let ttlMs = MAP_LINE_TTL_MS;
  let effectPeriod = 2.8;
  let effectTrail = 0.68;
  let effectSymbolSize = 7;
  // 状态主要影响线宽/线型；颜色由队伍决定
  if (status === "lateral") {
    lineWidth = 2.1;
    effectPeriod = 2.4;
    effectTrail = 0.6;
  }
  if (status === "success") {
    lineWidth = 3.2;
    effectPeriod = 2.0;
    effectTrail = 0.74;
    effectSymbolSize = 8;
    ttlMs = MAP_LINE_TTL_MS + 5000;
  }
  // 防守成功：更偏“拦截”，使用冷色调
  if (status === "defense_success") {
    drawColor = "#00b7ff";
    lineWidth = 2.8;
    effectPeriod = 2.2;
    effectTrail = 0.64;
  }
  // 溯源成功：回溯方向（目标 -> 来源）+ 虚线
  if (status === "trace_success") {
    lineWidth = 2.3;
    lineDash = "dashed";
    effectPeriod = 2.1;
    effectTrail = 0.6;
    drawColor = "#ffd166";
  }
  if (!targetCoord || !sourceCoord || !drawTarget || !drawSource) return;

  if (status === "trace_success") {
    drawSource = targetCoord;
    drawTarget = sourceCoord;
  }

  mapLineEntries.push({
    sourceCoord: drawSource,
    targetCoord: drawTarget,
    lineColor: drawColor,
    lineWidth,
    lineDash: lineDash === "dashed" ? "dashed" : undefined,
    effectPeriod,
    effectTrail,
    effectSymbolSize,
    createdAt: Date.now(),
    ttlMs,
  });
  if (mapLineEntries.length > MAP_LINE_MAX) mapLineEntries.shift();

  if (status === "success" || status === "defense_success" || status === "trace_success") {
    mapHitMarkers.push({
      coord: targetCoord,
      cityName: String(data.target_city ?? ""),
      status,
      role: "target",
      createdAt: Date.now(),
      ttlMs: MAP_HIT_MARK_TTL_MS,
    });
    if (status === "success" && sourceCoord) {
      mapHitMarkers.push({
        coord: sourceCoord,
        cityName: String(sourceCity ?? ""),
        status,
        role: "source",
        createdAt: Date.now(),
        ttlMs: Math.max(1800, Math.floor(MAP_HIT_MARK_TTL_MS * 0.55)),
      });
    }
    if (mapHitMarkers.length > MAP_HIT_MARK_MAX) mapHitMarkers.shift();
  }

  // 终点/起点做脉冲高亮（让画面更有层次）
  if (sourceCity && currentCities[sourceCity]) cityPulse[sourceCity] = Date.now();
  if (data.target_city && currentCities[data.target_city]) cityPulse[data.target_city] = Date.now();

  if (data.target_city && currentCities[data.target_city]) {
    mapRegionFlashes.push({
      cityName: String(data.target_city),
      status,
      createdAt: Date.now(),
      ttlMs: MAP_REGION_FLASH_TTL_MS,
    });
    if (mapRegionFlashes.length > 24) mapRegionFlashes.shift();
  }
  triggerMapImpact(status);

  refreshMapSeriesOnly(currentMapType);
}

function computeReplayDelayMs() {
  // replaySpeed 越大，间隔越短
  return Math.max(20, Math.round(220 / replaySpeed.value));
}

// 保留接口：当前实现里 loadReplay 已经做了基线初始化，无需额外操作。
function resetReplayBaseUI() {}

async function loadReplay(fromSeq: number = 1) {
  try {
    pauseReplay();
    isReplaying.value = true;

    replayLoading.value = true;
    replayReady.value = false;
    replayPlaying.value = false;
    replayCursor.value = 0;
    replayEvents.value = [];

    if (!matchId) {
      terminalLines.value = ["[SYSTEM] 未提供 match_id，无法回放"];
      return;
    }

    const initRes = await fetch(`${apiBaseHttp}/api/matches/${matchId}/initial_state`, { headers: authHeaders });
    if (!initRes.ok) throw new Error(`load initial_state failed: ${initRes.status}`);
    const initJson = await initRes.json();
    const initState = initJson.state as MatchStateDTO;

    // 基线：match 创建时的“初始配置 + 初始比分”
    state.map_type = initState.map_type;
    state.leaderboard_visible = initState.leaderboard_visible;
    state.teams = initState.teams;
    state.attack_stats = initState.attack_stats;
    state.panels = initState.panels;

    currentMapType = state.map_type === "taizhou" ? "taizhou" : "china";
    mapLineEntries = [];
    mapHitMarkers = [];
    mapRegionFlashes = [];

    // 拉取全量事件流（用于 fast-forward）
    terminalLines.value = ["[SYSTEM] 正在加载回放事件流..."];
    const evRes = await fetch(`${apiBaseHttp}/api/matches/${matchId}/events?from_seq=1&limit=5000`, { headers: authHeaders });
    if (!evRes.ok) throw new Error(`load events failed: ${evRes.status}`);
    const evJson = await evRes.json();
    const allEvents = (evJson.events ?? []) as any[];
    if (!allEvents.length) {
      replayReady.value = false;
      terminalLines.value = ["[SYSTEM] 本场次暂无事件，无法回放"];
      return;
    }

    const startSeq = Math.max(1, Math.floor(Number(fromSeq || 1)));
    const startIndex = Math.min(allEvents.length, Math.max(0, startSeq - 1));

    // Fast-forward：静默应用到起点（不播放动画/飞线/告警/日志）
    for (let i = 0; i < startIndex; i++) {
      // eslint-disable-next-line no-await-in-loop
      await applyReplayEvent(allEvents[i], { animate: false });
    }

    // 起点状态就绪后，再加载一次地图底图与基础图层
    mapLineEntries = [];
    mapHitMarkers = [];
    if (mapChart) {
      await loadMapData(currentMapType);
    }
    updateRadar(state.teams);
    updatePie(state.attack_stats);

    // 仅对起点之后的“可播放”事件做播放（跳过回放控制命令，避免空步长）
    replayEvents.value = allEvents
      .slice(startIndex)
      .filter((ev) => ev.event_type !== "replay_start" && ev.event_type !== "replay_exit");
    replayCursor.value = 0;
    replayReady.value = replayEvents.value.length > 0;

    terminalLines.value = ["[SYSTEM] 回放加载完成，点击“播放”开始回放..."];
  } finally {
    replayLoading.value = false;
  }
}

function pauseReplay() {
  replayPlaying.value = false;
  if (replayTimer) window.clearTimeout(replayTimer);
  replayTimer = undefined;
}

function stopReplayIfEnd() {
  if (replayCursor.value >= replayEvents.value.length) {
    pauseReplay();
    replayReady.value = replayEvents.value.length > 0;
    // 回放结束后应恢复实时模式，否则后续 live event 会被忽略。
    isReplaying.value = false;
  }
}

function exitReplay() {
  isReplaying.value = false;
  pauseReplay();

  replayReady.value = false;
  replayPlaying.value = false;
  replayEvents.value = [];
  replayCursor.value = 0;

  terminalLines.value = ["[SYSTEM] 已退出回放，恢复实时模式..."];

  // 回到实时后主动拉一次 state，避免仅靠 WS 后续事件更新导致短暂不同步。
  if (!matchId) return;
  void (async () => {
    const res = await fetch(`${apiBaseHttp}/api/matches/${matchId}/state`, { headers: authHeaders });
    if (!res.ok) return;
    const json = await res.json();
    const s = json.state as MatchStateDTO;
    Object.assign(state, s);
    currentMapType = state.map_type === "taizhou" ? "taizhou" : "china";
    mapLineEntries = [];
    mapHitMarkers = [];
    await loadMapData(currentMapType);
    updateFromState();
  })();
}

async function applyReplayEvent(ev: any, opts: { animate?: boolean } = { animate: true }) {
  const animate = opts.animate ?? true;
  const eventType: string = ev.event_type ?? ev.EventType ?? ev.eventType ?? "";
  const data: any = ev.payload_raw ?? ev.PayloadRaw ?? ev.payloadRaw ?? ev.payload ?? {};

  if (!eventType) return;

  if (eventType === "system_broadcast") {
    const text = data.message ?? "";
    if (animate && text) {
      triggerAlert(text, "system");
      pushLog(`[SYSTEM] ${text}`);
    }
    // state 不变，仅在动画模式下刷新一次
    if (animate) updateRadar(state.teams);
    return;
  }

  if (eventType === "switch_map") {
    const mapType = data.map_type === "taizhou" ? "taizhou" : "china";
    state.map_type = mapType;
    currentMapType = mapType;
    mapLineEntries = [];
    mapHitMarkers = [];
    if (animate) {
      await loadMapData(currentMapType);
      pushLog(
        `[SYSTEM] 回放切换地图为 ${mapType === "taizhou" ? "泰州市区县态势" : "全国态势"}`
      );
      updateRadar(state.teams);
      updatePie(state.attack_stats);
    }
    return;
  }

  if (eventType === "toggle_panel") {
    const panelID = data.panel_id ?? "";
    const visible = !!data.visible;
    if (panelID) {
      state.panels[panelID] = visible;
      if (panelID === "panel-leaderboard") state.leaderboard_visible = visible;
      if (animate) pushLog(`[SYSTEM] 回放面板 ${panelID} => ${visible ? "显示" : "隐藏"}`);
    }
    return;
  }

  if (eventType === "manual_score") {
    const teamID = Number(data.team_id);
    const delta = Number(data.score_change ?? 0);
    const team = state.teams.find((t) => t.id === teamID);
    if (team) {
      team.score += delta;
      if (animate) {
        pulseTeamScore(teamID);
        const message = data.message ?? "";
        if (message) triggerAlert(message, "warn");
        pushLog(`[JUDGE] ${team.name} 计分变化: ${delta}`);
        updateRadar(state.teams);
      }
    }
    return;
  }

  if (eventType === "attack_success") {
    const teamID = Number(data.team_id);
    const scoreChange = Number(data.score_change ?? 0);
    const attacker = state.teams.find((t) => t.id === teamID);
    if (!attacker) return;

    attacker.score += scoreChange;
    if (animate) pulseTeamScore(attacker.id);

    const deltaBlue = Math.floor(scoreChange / 2);
    if (deltaBlue !== 0) {
      state.teams.forEach((t) => {
        if (t.type === "blue") {
          t.score -= deltaBlue;
          if (animate) pulseTeamScore(t.id);
        }
      });
    }

    const attackType = data.attack_type ?? "";
    if (attackType) {
      const stat = state.attack_stats.find((s) => s.name === attackType);
      if (stat) stat.value += 1;
      else state.attack_stats.push({ name: attackType, value: 1 });
    }

    const source = data.source_city ?? "未知源";
    const target = data.target_city ?? "未知目标";
    const status = data.status ?? "attempt";
    if (animate) {
      const action = `[${attackType || "attack"}] ${status}`;
      pushLog(`[${source}] => ${target} ${action}`);

      if (
        (status === "success" || status === "defense_success" || status === "trace_success") &&
        data.message
      ) {
        triggerAlert(data.message, "attack");
      }

      // 飞线动效
      updateMapLineByAttack({
        ...data,
        team_type: attacker.type,
      });

      updateRadar(state.teams);
      updatePie(state.attack_stats);
    }
    return;
  }

  // 其他类型目前忽略（例如 teams_updated 在回放里可作为后续增强）。
}

async function replayTick() {
  if (!replayPlaying.value) return;
  if (replayCursor.value >= replayEvents.value.length) {
    stopReplayIfEnd();
    return;
  }
  const ev = replayEvents.value[replayCursor.value];
  await applyReplayEvent(ev);
  replayCursor.value += 1;

  if (replayCursor.value >= replayEvents.value.length) {
    stopReplayIfEnd();
    return;
  }

  replayTimer = window.setTimeout(() => {
    void replayTick();
  }, computeReplayDelayMs());
}

async function startReplay() {
  if (!replayReady.value || replayPlaying.value) return;
  replayPlaying.value = true;
  isReplaying.value = true;
  terminalLines.value = ["[SYSTEM] 回放开始..."];
  // 防止重复触发（只清计时器，不要把 replayPlaying 置回 false）
  if (replayTimer) window.clearTimeout(replayTimer);
  replayTimer = window.setTimeout(() => {
    void replayTick();
  }, 60);
}

function stepReplay() {
  if (!replayReady.value || replayPlaying.value) return;
  if (replayCursor.value >= replayEvents.value.length) return;
  void applyReplayEvent(replayEvents.value[replayCursor.value]).then(() => {
    replayCursor.value += 1;
    stopReplayIfEnd();
  });
}

loadTerminalLogsFromSession();

onMounted(async () => {
  updateViewportAdaptiveVars();
  bindAudioUnlock();

  clockTimer = window.setInterval(() => {
    const now = new Date();
    clock.value = now.toLocaleTimeString("en-US", { hour12: false });
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, "0");
    const d = String(now.getDate()).padStart(2, "0");
    dateStr.value = `${y}-${m}-${d}`;
  }, 1000);

  // 先启动榜单轮播（若不足一屏会自动不滚动）
  startLeaderboardLoop();
  startTerminalLoop();

  await nextTick();
  // 计算日志可视行数，确保框能更“填满”
  try {
    const el = terminalScrollEl.value;
    if (el) {
      const lineEl = el.querySelector(".log-line") as HTMLElement | null;
      const lineH = lineEl ? lineEl.getBoundingClientRect().height + 6 : 19; // 额外估算 margin
      const h = el.getBoundingClientRect().height;
      terminalVisibleLines.value = Math.max(8, Math.floor(h / Math.max(12, lineH)));
      terminalItemStepPx.value = Math.max(14, lineH);
      ensureTerminalFilled();
      persistTerminalLogsToSession();
    }
  } catch {
    // ignore
  }
  if (!mapEl.value || !radarEl.value || !pieEl.value) return;

  // 计算榜单单行步进高度（含 margin），用于平滑滚动
  try {
    const el = leaderViewportEl.value?.querySelector(".leader-item") as HTMLElement | null;
    if (el) {
      const styles = window.getComputedStyle(el);
      const mb = Number.parseFloat(styles.marginBottom || "0") || 0;
      leaderItemStepPx.value = el.getBoundingClientRect().height + mb;
    }
  } catch {
    // ignore: fallback to default estimate
  }

  mapChart = echarts.init(mapEl.value);
  radarChart = echarts.init(radarEl.value);
  pieChart = echarts.init(pieEl.value);

  // 让飞线随时间渐隐，不需要新事件也能保持“持续一段时间”的视觉效果。
  mapLineRefreshTimer = window.setInterval(() => {
    if (!mapChart) return;
    refreshMapSeriesOnly(currentMapType);
  }, 900);

  resizeHandler = () => {
    updateViewportAdaptiveVars();
    mapChart?.resize();
    radarChart?.resize();
    pieChart?.resize();
  };
  window.addEventListener("resize", resizeHandler);

  // 如果 URL 没带 match_id，提示即可（后续可加“选择场次”弹层）。
  if (!matchId) {
    pushLog("[SYSTEM] 未提供 match_id，请在 URL 加 ?match_id=xxx");
    return;
  }
  if (!jwtToken) {
    pushLog("[SECURITY] 未提供 token，当前大屏无法连接（请使用 Admin 跳转或手动在 URL 加 &token=xxx）");
    return;
  }

  wsConn.value = connectMatchWS({
    matchId,
    apiBaseHttp,
    token: jwtToken || undefined,
    onOpen: () => {
      wsConnected.value = true;
    },
    onClose: () => {
      wsConnected.value = false;
    },
    onMessage: (msg) => {
      const m = msg as WSMessage;
      const inReplayMode = isReplaying.value;

      if (m.type === "sync_state") {
        wsSynced.value = true;
        Object.assign(state, m.state);
        syncBGMPlayback();
        currentMapType = state.map_type === "taizhou" ? "taizhou" : "china";
        mapLineEntries = [];
        mapHitMarkers = [];
        loadMapData(currentMapType);
        updateFromState();
        return;
      }

      if (m.type === "event") {
        // 回放期间（屏幕不展示回放UI时）也必须让“分数/统计”保持同步，
        // 但跳过飞线、告警、日志等副作用，避免与回放视觉冲突。
        if (
          inReplayMode &&
          m.event !== "replay_exit" &&
          m.event !== "replay_start" &&
          m.event !== "attack_success"
        ) {
          Object.assign(state, m.state);
          updateFromState();
          return;
        }

        if (m.event === "replay_exit") {
          exitReplay();
          return;
        }
        if (m.event === "replay_start") {
          const fromSeq = Number((m.data as any)?.from_seq ?? 1);
          const speed = Number((m.data as any)?.speed ?? replaySpeed.value);
          if (Number.isFinite(speed) && speed > 0) replaySpeed.value = speed;
          void (async () => {
            await loadReplay(fromSeq);
            await startReplay();
          })();
          return;
        }

        const prevScores = new Map<number, number>(state.teams.map((t) => [t.id, t.score]));
        // state 由后端全量给前端，避免重复计算产生偏差。
        Object.assign(state, m.state);
        syncBGMPlayback();

        const data = m.data ?? {};
        if (m.event === "system_broadcast") {
          const text = data.message ?? "";
          if (text) {
            triggerAlert(text, "system");
            pushLog(`[SYSTEM] ${text}`);
          }
        } else if (m.event === "switch_map") {
          pushLog(`[SYSTEM] 切换地图为 ${data.map_type}`);
        } else if (m.event === "toggle_panel") {
          pushLog(`[SYSTEM] 面板 ${data.panel_id} => ${data.visible ? "显示" : "隐藏"}`);
        } else if (m.event === "manual_score") {
          const team = state.teams.find((t) => t.id === data.team_id);
          const delta = data.score_change;
          const message = data.message ?? "";
          if (message) triggerAlert(message, "warn");
          pushLog(`[JUDGE] ${team?.name ?? "未知队伍"} 计分变化: ${delta}`);
        } else if (m.event === "attack_success") {
          const team = state.teams.find((t) => t.id === data.team_id);
          const action = `[${data.attack_type ?? "attack"}] ${data.status ?? ""}`;
          if (!inReplayMode) {
            const broadcastText = data.message ? String(data.message) : "";
            if (broadcastText) {
              pushLog(`[播报] ${broadcastText}`);
            } else {
              pushLog(`[${data.source_city ?? "未知源"}] => ${data.target_city ?? "未知目标"} ${action}`);
            }
            if (
              (data.status === "success" || data.status === "defense_success" || data.status === "trace_success") &&
              data.message
            ) {
              triggerAlert(data.message, "attack");
            }
            if (data.status === "success") {
              playSuccessSfx();
            }
          }

          // 飞线动效：使用事件里的城市/状态。
          // 注意：后端目前未下发 team_type 字段，飞线颜色先按队伍阵营推断。
          const enriched = {
            ...data,
            team_type: team?.type,
          };
          updateMapLineByAttack(enriched);
        }

        // 让图表与地图在 state 改动后保持一致。
        // （map_type 切换时 loadMapData 会清空飞线数组，这是期望行为）
        updateFromState();

        // 评分脉冲：只对比分实际变化的队伍触发。
        if (!inReplayMode) {
          for (const [teamId, oldScore] of prevScores.entries()) {
            const team = state.teams.find((t) => t.id === teamId);
            if (!team) continue;
            if (team.score !== oldScore) pulseTeamScore(teamId);
          }
        }
      }
    },
  });
});

onBeforeUnmount(() => {
  if (clockTimer) window.clearInterval(clockTimer);
  if (alertTimer) window.clearTimeout(alertTimer);
  if (replayTimer) window.clearTimeout(replayTimer);
  if (leaderboardTimer) window.clearInterval(leaderboardTimer);
  if (terminalLoopTimer) window.clearInterval(terminalLoopTimer);
  if (terminalFlushTimer) window.clearInterval(terminalFlushTimer);
  if (mapLineRefreshTimer) window.clearInterval(mapLineRefreshTimer);
  if (mapImpactTimer) window.clearTimeout(mapImpactTimer);
  if (resizeHandler) window.removeEventListener("resize", resizeHandler);
  bgmAudioEl.value?.pause();
  successSfxAudioEl.value?.pause();
  wsConn.value?.close();
});
</script>

<style scoped>
.screen-root {
  height: 100vh;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
  box-sizing: border-box;
  --ui-scale: 1;
  --panel-basis: 26%;
  --panel-max-width: 420px;
  --map-flex-grow: 1;
  --topbar-height: 80px;
  --font-user: 1;
}

.topbar {
  height: var(--topbar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 calc(32px * var(--ui-scale));
  position: relative;
  z-index: 10;
}

.topbar-left {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  min-width: 0;
}

.topbar-center {
  flex: 1 1 0;
  min-width: 0;
  text-align: center;
  overflow: hidden;
  padding: 0 calc(10px * var(--ui-scale));
}

.topbar-right {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
}

.ws-dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  margin-right: 12px;
  box-shadow: 0 0 8px currentColor;
}
.ws-dot-ok {
  background: #10b981;
  color: #10b981;
}
.ws-dot-bad {
  background: #ff2a2a;
  color: #ff2a2a;
}

.topbar-label {
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  color: #06b6d4;
  margin-right: calc(16px * var(--ui-scale));
}
.topbar-bar {
  width: calc(96px * var(--ui-scale));
  height: calc(2px * var(--ui-scale));
  background: #22d3ee;
  box-shadow: 0 0 10px #00f3ff;
  opacity: 0.9;
}
.topbar-bar-mini {
  width: calc(16px * var(--ui-scale));
  height: calc(2px * var(--ui-scale));
  background: #22d3ee;
  opacity: 0.9;
}

.topbar-title {
  margin: 0;
  font-size: calc(28px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 700;
  letter-spacing: 0.08em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
.topbar-sub {
  margin-top: calc(6px * var(--ui-scale));
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  color: #22d3ee;
  letter-spacing: 0.12em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.topbar-clock {
  color: #22d3ee;
  min-width: 92px;
  text-align: right;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.topbar-date {
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(34, 211, 238, 0.9);
}

.topbar-time {
  font-size: calc(14px * var(--ui-scale) * var(--font-user, 1));
}

.alert-bar {
  position: absolute;
  top: calc(var(--topbar-height) + calc(20px * var(--ui-scale)));
  left: 50%;
  transform: translateX(-50%);
  width: auto;
  max-width: min(82vw, 1320px);
  z-index: 50;
  border-radius: 12px;
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.22);
  box-shadow: 0 14px 36px rgba(0, 0, 0, 0.36), inset 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.alert-bar-system {
  background: linear-gradient(90deg, rgba(20, 64, 98, 0.86), rgba(17, 80, 112, 0.86));
  border-color: rgba(130, 233, 255, 0.45);
}

.alert-bar-attack {
  background: linear-gradient(90deg, rgba(122, 24, 24, 0.9), rgba(154, 42, 42, 0.9) 45%, rgba(86, 30, 30, 0.88));
  border-color: rgba(255, 170, 170, 0.58);
  box-shadow: 0 14px 36px rgba(62, 0, 0, 0.45), inset 0 0 0 1px rgba(255, 210, 210, 0.18), 0 0 26px rgba(255, 92, 92, 0.26);
}

.alert-bar-warn {
  background: linear-gradient(90deg, rgba(89, 64, 15, 0.9), rgba(124, 86, 22, 0.88));
  border-color: rgba(253, 205, 120, 0.48);
}

.alert-inner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 12px 16px;
  width: fit-content;
  max-width: min(82vw, 1320px);
}

.alert-tag {
  flex: 0 0 auto;
  font-size: calc(13px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 800;
  letter-spacing: 0.08em;
  padding: 6px 10px;
  border-radius: 999px;
  color: rgba(248, 250, 252, 0.98);
  background: rgba(255, 255, 255, 0.16);
}

.alert-main {
  flex: 0 1 auto;
  min-width: 1px;
  font-size: calc(18px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 800;
  color: #f8fafc;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.layout {
  flex: 1;
  display: flex;
  align-items: stretch;
  gap: 0;
  min-height: 0;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
  padding: 0 calc(12px * var(--ui-scale)) calc(12px * var(--ui-scale)) calc(12px * var(--ui-scale));
}

.layout-splitter {
  flex: 0 0 6px;
  width: 6px;
  cursor: col-resize;
  align-self: stretch;
  background: linear-gradient(90deg, transparent, rgba(56, 189, 248, 0.12), transparent);
  position: relative;
  z-index: 12;
}
.layout-splitter:hover {
  background: linear-gradient(90deg, transparent, rgba(56, 189, 248, 0.35), transparent);
}
.layout-splitter::after {
  content: "";
  position: absolute;
  top: 18%;
  bottom: 18%;
  left: 50%;
  width: 2px;
  transform: translateX(-50%);
  border-radius: 2px;
  background: rgba(56, 189, 248, 0.35);
}

.panel {
  width: auto;
  flex: 0 0 var(--panel-basis);
  min-width: 200px;
  max-width: var(--panel-max-width);
  display: flex;
  flex-direction: column;
  gap: calc(8px * var(--ui-scale));
  box-sizing: border-box;
  flex-shrink: 0;
}
/* 左右栏宽度由 flex-basis（--panel-basis）或拖拽像素宽度控制，勿再写死 25%，否则易与 flex 分配冲突产生右侧空白 */

.panel-inner {
  flex: 1;
  padding: calc(12px * var(--ui-scale));
}

/* 限制左侧两块面板的高度比例，避免榜单过高把下方雷达挤压 */
#panel-leaderboard {
  flex: 0 0 40%;
  max-height: 40%;
}
.panel-left .radar-panel {
  flex: 1 1 60%;
  min-height: 0;
}

.panel-right .panel-logs {
  flex: 0 0 40%;
  max-height: 40%;
}
.panel-right .panel-pie {
  flex: 1 1 60%;
  min-height: 0;
}

.panel-v-splitter {
  flex: 0 0 8px;
  height: 8px;
  cursor: row-resize;
  border-radius: 4px;
  background: linear-gradient(180deg, transparent, rgba(56, 189, 248, 0.2), transparent);
  position: relative;
}
.panel-v-splitter:hover {
  background: linear-gradient(180deg, transparent, rgba(56, 189, 248, 0.4), transparent);
}
.panel-v-splitter::after {
  content: "";
  position: absolute;
  left: 22%;
  right: 22%;
  top: 50%;
  height: 2px;
  transform: translateY(-50%);
  border-radius: 2px;
  background: rgba(56, 189, 248, 0.5);
}

.radar-panel {
  padding-top: 6px;
}

.panel-title {
  margin: 0 0 12px 0;
  font-size: calc(16px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 700;
  color: var(--neon-blue);
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid rgba(0, 243, 255, 0.2);
  padding-bottom: 10px;
}
.panel-dot {
  width: calc(10px * var(--ui-scale));
  height: calc(4px * var(--ui-scale));
  background: var(--neon-blue);
  box-shadow: 0 0 8px rgba(0, 243, 255, 0.6);
}

.leader-list {
  height: calc(100% - 54px);
  overflow: hidden;
  padding-right: 4px;
}

/* 固定榜单展示容量：避免队伍过多导致视觉“挤压感” */
.leader-item:last-child {
  margin-bottom: 0;
}

.leader-track {
  will-change: transform;
}
.leader-track.anim {
  transition: transform 0.52s ease;
}

.terminal-track {
  will-change: transform;
}
.terminal-track.anim {
  transition: transform 0.52s ease;
}

.leader-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: calc(8px * var(--ui-scale)) calc(10px * var(--ui-scale));
  background: rgba(0, 0, 0, 0.25);
  border-radius: 6px;
  border: 1px solid rgba(0, 243, 255, 0.08);
  margin-bottom: calc(10px * var(--ui-scale));
  transition: transform 0.3s ease, opacity 0.3s ease, box-shadow 0.3s ease;
}
.leader-left {
  display: flex;
  align-items: center;
  gap: calc(12px * var(--ui-scale));
}
.leader-rank {
  width: calc(26px * var(--ui-scale));
  text-align: center;
  font-size: calc(16px * var(--ui-scale) * var(--font-user, 1));
}
.rank-top {
  color: var(--neon-yellow);
}
.rank-rest {
  color: rgba(229, 231, 235, 0.6);
}
.leader-logo {
  font-size: 26px;
}
.leader-name {
  font-weight: 700;
  white-space: nowrap;
  font-size: calc(15px * var(--ui-scale) * var(--font-user, 1));
}
.name-red {
  color: #feb2b2;
}
.name-blue {
  color: #93c5fd;
}
.leader-score {
  font-weight: 800;
  font-size: calc(22px * var(--ui-scale) * var(--font-user, 1));
}
.score-red {
  color: var(--neon-red);
  text-shadow: 0 0 10px rgba(255, 42, 42, 0.7);
}
.score-blue {
  color: var(--neon-blue);
  text-shadow: 0 0 10px rgba(0, 243, 255, 0.6);
}

@keyframes scorePulse {
  0% {
    transform: scale(1);
    filter: brightness(1);
  }
  50% {
    transform: scale(1.12);
    filter: brightness(1.6);
  }
  100% {
    transform: scale(1);
    filter: brightness(1);
  }
}

.score-pulse {
  animation: scorePulse 800ms ease-in-out;
}

.replay-ui {
  position: absolute;
  top: 96px;
  right: 20px;
  z-index: 80;
  width: 520px;
  max-width: calc(100vw - 40px);
  background: rgba(3, 8, 22, 0.7);
  border: 1px solid rgba(0, 243, 255, 0.25);
  box-shadow: 0 0 18px rgba(0, 243, 255, 0.12);
  border-radius: 8px;
  padding: 12px 14px;
  backdrop-filter: blur(6px);
}

.replay-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.replay-btn {
  padding: 8px 12px;
  border: 1px solid rgba(0, 243, 255, 0.25);
  background: rgba(0, 0, 0, 0.25);
  color: #a5f3fc;
  border-radius: 6px;
  cursor: pointer;
}

.replay-btn.primary {
  border-color: rgba(34, 211, 238, 0.6);
  box-shadow: 0 0 10px rgba(0, 243, 255, 0.2);
  color: #cffafe;
}

.replay-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.replay-speed {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.replay-label {
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(229, 231, 235, 0.7);
}

.replay-select {
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(0, 243, 255, 0.2);
  color: #e5eaf3;
  border-radius: 6px;
  padding: 6px 8px;
}

.replay-meta {
  font-size: 12px;
  color: rgba(229, 231, 235, 0.75);
  min-width: 88px;
  text-align: right;
}

.map-section {
  /* flex-basis:0 + grow：高 DPI/缩放时仍吃满中间，避免右侧漏缝 */
  flex-grow: var(--map-flex-grow);
  flex-shrink: 1;
  flex-basis: 0;
  min-width: 0;
  position: relative;
  padding: 0;
  overflow: hidden;
  isolation: isolate;
}

/* 用 ::after 叠在 canvas 之上，边缘压暗 + 青边光（不拦截拖拽/缩放） */
.map-section::after {
  content: "";
  position: absolute;
  inset: 0;
  z-index: 7;
  pointer-events: none;
  box-shadow:
    inset 0 0 86px 16px rgba(2, 10, 28, 0.54),
    inset 0 0 44px 8px rgba(0, 243, 255, 0.14),
    inset 0 0 0 1px rgba(120, 240, 255, 0.32),
    0 0 18px rgba(0, 243, 255, 0.12);
  border-radius: 2px;
}

/* 攻击命中短促冲击：地图轻微震荡 + 暖色闪光 */
.map-impact {
  animation: mapImpactShake 560ms cubic-bezier(0.22, 0.61, 0.36, 1);
}

.map-impact::before {
  content: "";
  position: absolute;
  inset: 0;
  z-index: 6;
  pointer-events: none;
  background: radial-gradient(circle at 50% 50%, rgba(255, 92, 92, 0.22), rgba(255, 92, 92, 0) 58%);
  animation: mapImpactFlash 560ms ease-out;
}

@keyframes mapImpactShake {
  0% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  18% {
    transform: translate3d(-1px, 1px, 0) scale(1.003);
  }
  38% {
    transform: translate3d(1px, -1px, 0) scale(1.001);
  }
  58% {
    transform: translate3d(-0.6px, 0.8px, 0) scale(1.002);
  }
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
}

@keyframes mapImpactFlash {
  0% {
    opacity: 0;
  }
  22% {
    opacity: 1;
  }
  100% {
    opacity: 0;
  }
}

.map-chart {
  width: 100%;
  height: 100%;
}

.chart {
  width: 100%;
  height: 100%;
}

.map-mode-indicator {
  position: absolute;
  top: 16px;
  right: 16px;
  left: auto;
  transform: none;
  z-index: 10;
  padding: 10px 14px;
  border-radius: 10px;
  background: rgba(3, 8, 22, 0.45);
  border: 1px solid rgba(0, 243, 255, 0.35);
  box-shadow: 0 0 18px rgba(0, 243, 255, 0.12);
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: stretch;
  min-width: 220px;
}

.map-settings-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.map-settings-label {
  flex: 0 0 auto;
  font-size: calc(11px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(165, 243, 252, 0.85);
}
.map-settings-range {
  flex: 1;
  min-width: 0;
  accent-color: #22d3ee;
}
.map-settings-val {
  flex: 0 0 40px;
  text-align: right;
  font-size: calc(11px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(226, 232, 240, 0.95);
  font-variant-numeric: tabular-nums;
}
.map-settings-hint {
  font-size: calc(10px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(148, 163, 184, 0.85);
  line-height: 1.35;
}

.map-reset-layout-btn {
  width: 100%;
  margin-top: 4px;
  padding: calc(8px * var(--ui-scale)) calc(10px * var(--ui-scale));
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  border-radius: 8px;
  border: 1px solid rgba(251, 191, 36, 0.45);
  background: rgba(251, 191, 36, 0.12);
  color: rgba(254, 243, 199, 0.95);
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease;
}
.map-reset-layout-btn:hover:not(:disabled) {
  background: rgba(251, 191, 36, 0.2);
  border-color: rgba(251, 191, 36, 0.65);
}
.map-reset-layout-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.map-settings-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 11;
  padding: calc(8px * var(--ui-scale)) calc(12px * var(--ui-scale));
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  border-radius: 10px;
  background: rgba(3, 8, 22, 0.45);
  border: 1px solid rgba(0, 243, 255, 0.35);
  color: rgba(165, 243, 252, 0.95);
  cursor: pointer;
  box-shadow: 0 0 18px rgba(0, 243, 255, 0.08);
}

.map-settings-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}


.map-mode-title {
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(165, 243, 252, 0.95);
  letter-spacing: 0.02em;
}

.map-mode-switch {
  display: flex;
  gap: 8px;
}

.map-mode-btn {
  padding: calc(6px * var(--ui-scale)) calc(10px * var(--ui-scale));
  font-size: calc(12px * var(--ui-scale) * var(--font-user, 1));
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(0, 243, 255, 0.25);
  color: rgba(165, 243, 252, 0.95);
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.2s ease, border-color 0.2s ease, background 0.2s ease;
}

.map-mode-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 0 14px rgba(0, 243, 255, 0.2);
}

.map-mode-btn.active {
  background: rgba(0, 243, 255, 0.1);
  border-color: rgba(0, 243, 255, 0.7);
  box-shadow: 0 0 18px rgba(0, 243, 255, 0.12);
}

.map-mode-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.map-metrics {
  position: absolute;
  top: 88px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10;
  display: flex;
  gap: 10px;
  padding: 10px 12px;
  background: rgba(3, 8, 22, 0.45);
  border: 1px solid rgba(0, 243, 255, 0.25);
  border-radius: 10px;
  box-shadow: 0 0 18px rgba(0, 243, 255, 0.08);
}

.metric {
  min-width: 150px;
}
.metric-label {
  font-size: 11px;
  color: rgba(165, 243, 252, 0.85);
  margin-bottom: 6px;
}
.metric-value {
  font-size: 14px;
  font-weight: 800;
  color: rgba(229, 231, 235, 0.95);
  text-shadow: 0 0 10px rgba(0, 243, 255, 0.12);
}
.metric-value.red {
  color: rgba(248, 113, 113, 0.95);
  text-shadow: 0 0 12px rgba(255, 42, 42, 0.25);
}
.metric-value.blue {
  color: rgba(147, 197, 253, 0.95);
  text-shadow: 0 0 12px rgba(0, 243, 255, 0.2);
}

.map-decor {
  position: absolute;
  width: 36px;
  height: 36px;
  border-color: var(--neon-blue);
  border-style: solid;
  opacity: 0.5;
}
.map-decor-tl {
  top: 10px;
  left: 10px;
  border-width: 2px 0 0 2px;
}
.map-decor-tr {
  top: 10px;
  right: 10px;
  border-width: 2px 2px 0 0;
}
.map-decor-bl {
  bottom: 10px;
  left: 10px;
  border-width: 0 0 2px 2px;
}
.map-decor-br {
  bottom: 10px;
  right: 10px;
  border-width: 0 2px 2px 0;
}

.terminal-logs {
  height: calc(100% - 54px);
  overflow-y: auto;
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(0, 243, 255, 0.08);
  padding: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New",
    monospace;
  color: #34d399;
}
.log-line {
  margin-bottom: 6px;
  opacity: 0.95;
  font-size: calc(13px * var(--ui-scale) * var(--font-user, 1));
}

.log-system {
  color: #34d399;
  text-shadow: 0 0 10px rgba(52, 211, 153, 0.2);
}
.log-judge {
  color: #fbbf24;
  text-shadow: 0 0 10px rgba(251, 191, 36, 0.15);
}
.log-event {
  color: #22d3ee;
}
.log-normal {
  color: #e5e7eb;
}

.pie-chart {
  height: calc(100% - 54px);
}
.radar-chart {
  height: calc(100% - 54px);
}

.bottom-credits {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 6px;
  z-index: 20;
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  padding: 4px 8px;
  pointer-events: none; /* 避免遮挡交互 */
}

.credits-line {
  flex: 0 1 auto;
  max-width: 100%;
  font-size: calc(11px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(229, 231, 235, 0.95);
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.credits-label {
  font-weight: 700;
  color: rgba(148, 224, 255, 0.9);
}

.credits-sep {
  color: rgba(229, 231, 235, 0.75);
  font-weight: 700;
}

.credits-value {
  font-weight: 700;
  text-shadow: 0 0 4px rgba(0, 243, 255, 0.2);
  word-break: break-word;
}

.credits-divider {
  width: 0;
  height: 0;
  border-left: 1px solid rgba(0, 243, 255, 0.22);
}

@media (max-width: 920px) {
  .credits-divider {
    display: none;
  }
}
</style>

