<template>
  <div class="screen-root" :style="{ ...screenRootStyle, ...topbarStyle }">
    <div class="crt-overlay"></div>
    
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
        <div v-if="countdownText" class="topbar-countdown font-cyber">
          <div class="countdown-icon"></div>
          <div class="countdown-content">
            <div class="countdown-label">EXERCISE COUNTDOWN</div>
            <div class="countdown-value">{{ countdownText }}</div>
          </div>
        </div>

        <div class="divider-line"></div>

        <div class="topbar-clock font-cyber">
          <div class="topbar-time">{{ clock }}</div>
          <div class="topbar-date">{{ dateStr }}</div>
        </div>

        <div class="signal-bars">
          <div class="topbar-bar-mini"></div>
          <div class="topbar-bar"></div>
          <div class="topbar-bar-mini delayed"></div>
        </div>
      </div>
    </header>
    <div class="topbar-splitter" title="拖拽调整大屏标题高度" @mousedown.prevent="startResizeTopbar"></div>

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
          v-show="slotVisible('left_top')"
          class="cyber-panel panel-inner screen-slot-left-top"
          :style="leftTopPanelStyle"
        >
          <h2 class="panel-title" :style="titleStyle('left_top')">
            <span class="panel-dot"></span> {{ moduleTitle("left_top") }}
          </h2>
          <div class="title-splitter" title="拖拽调整标题高度" @mousedown.prevent="(e) => startResizeTitle('left_top', e)"></div>
          <ScreenLeaderboard
            v-if="moduleAt('left_top') === 'leaderboard'"
            :sorted-teams="leaderboardSorted"
            :score-pulse="scorePulse"
          />
          <div
            v-else-if="moduleAt('left_top') === 'battle_logs'"
            ref="terminalLeftTopEl"
            class="terminal-scroll terminal-logs"
          >
            <div class="terminal-track" :class="terminalTrackAnimating ? 'anim' : ''" :style="terminalTrackStyle">
              <div
                v-for="(line, idx) in terminalWindow"
                :key="`${terminalStartIndex}-${idx}-lt`"
                class="log-line"
              >
                <span :class="logLineClass(line)">{{ line }}</span>
              </div>
            </div>
          </div>
          <ScreenHtmlPanel
            v-else-if="isHtmlSlot('left_top')"
            :variant="moduleAt('left_top')"
            :attack-stats="state.attack_stats"
            :region-attack-stats="state.region_attack_stats ?? []"
            :teams="state.teams"
          />
          <div v-else ref="chartElLeftTop" class="chart radar-chart"></div>
        </div>

        <div
          class="panel-v-splitter"
          title="拖拽调整左侧上下高度"
          @mousedown.prevent="startResizeLeftVertical"
        ></div>

        <div v-show="slotVisible('left_bottom')" class="cyber-panel panel-inner screen-slot-left-bottom">
          <h2 class="panel-title">
            <span class="panel-dot"></span> {{ moduleTitle("left_bottom") }}
          </h2>
          <ScreenLeaderboard
            v-if="moduleAt('left_bottom') === 'leaderboard'"
            :sorted-teams="leaderboardSorted"
            :score-pulse="scorePulse"
          />
          <div
            v-else-if="moduleAt('left_bottom') === 'battle_logs'"
            ref="terminalLeftBottomEl"
            class="terminal-scroll terminal-logs"
          >
            <div class="terminal-track" :class="terminalTrackAnimating ? 'anim' : ''" :style="terminalTrackStyle">
              <div
                v-for="(line, idx) in terminalWindow"
                :key="`${terminalStartIndex}-${idx}-lb`"
                class="log-line"
              >
                <span :class="logLineClass(line)">{{ line }}</span>
              </div>
            </div>
          </div>
          <ScreenHtmlPanel
            v-else-if="isHtmlSlot('left_bottom')"
            :variant="moduleAt('left_bottom')"
            :attack-stats="state.attack_stats"
            :region-attack-stats="state.region_attack_stats ?? []"
            :teams="state.teams"
          />
          <div v-else ref="chartElLeftBottom" class="chart radar-chart"></div>
        </div>
      </aside>

      <div
        class="layout-splitter layout-splitter-left"
        title="拖拽调整左侧宽度"
        @mousedown.prevent="startResizeLeft"
      ></div>

      <!-- 中间地图 -->
      <section class="map-section cyber-panel">
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
              min="0.8"
              max="2.6"
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
              泰州市态势
            </button>
          </div>
          <div class="map-mode-title">队伍起飞</div>
          <div class="map-mode-switch">
            <button
              type="button"
              class="map-mode-btn"
              :class="mapTeamDockLayout === 'dock' ? 'active' : ''"
              :disabled="isReplaying"
              @click="mapTeamDockLayout = 'dock'"
            >
              左侧栏
            </button>
            <button
              type="button"
              class="map-mode-btn"
              :class="mapTeamDockLayout === 'map' ? 'active' : ''"
              :disabled="isReplaying"
              @click="mapTeamDockLayout = 'map'"
            >
              地图锚点
            </button>
          </div>
          <div v-if="mapTeamDockLayout === 'dock'" class="map-settings-row">
            <span class="map-settings-label">侧栏宽度</span>
            <input
              v-model.number="mapTeamDockWidth"
              type="range"
              min="120"
              max="280"
              step="4"
              class="map-settings-range"
            />
            <span class="map-settings-val">{{ Math.round(mapTeamDockWidth) }}px</span>
          </div>
        </div>

        <div class="map-section-inner">
          <div class="map-frame" :class="mapImpactActive ? 'map-impact' : ''">
            <div class="map-body">
              <div
                v-show="mapTeamDockLayout === 'dock'"
                ref="mapTeamDockEl"
                class="map-team-dock"
                :style="{ width: `${mapTeamDockWidth}px`, flex: `0 0 ${mapTeamDockWidth}px` }"
              >
                <div v-if="!teamDockSorted.length" class="map-team-dock-empty">暂无队伍</div>
                <div v-else class="map-team-dock-rows" :style="teamDockGridStyle">
                  <div
                    v-for="t in teamDockSorted"
                    :key="t.id"
                    class="map-team-dock-row"
                    :data-team-id="t.id"
                    :style="{ '--team-line': teamLineColorById(t.id) }"
                  >
                    <span class="dock-team-name">{{ t.name }}</span>
                  </div>
                </div>
              </div>
              <div class="map-chart-shell">
                <div ref="mapEl" class="chart map-chart"></div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div
        class="layout-splitter layout-splitter-right"
        title="拖拽调整右侧宽度"
        @mousedown.prevent="startResizeRight"
      ></div>

      <!-- 右侧面板 -->
      <aside ref="rightPanelEl" class="panel panel-right" :style="useCustomPanelWidths ? rightPanelStyle : undefined">
        <div v-show="slotVisible('right_top')" class="cyber-panel panel-inner screen-slot-right-top" :style="rightTopPanelStyle">
          <h2 class="panel-title" :style="titleStyle('right_top')">
            <span class="panel-dot"></span> {{ moduleTitle("right_top") }}
          </h2>
          <div class="title-splitter" title="拖拽调整标题高度" @mousedown.prevent="(e) => startResizeTitle('right_top', e)"></div>
          <ScreenLeaderboard
            v-if="moduleAt('right_top') === 'leaderboard'"
            :sorted-teams="leaderboardSorted"
            :score-pulse="scorePulse"
          />
          <div
            v-else-if="moduleAt('right_top') === 'battle_logs'"
            ref="terminalScrollEl"
            class="terminal-scroll terminal-logs"
          >
            <div class="terminal-track" :class="terminalTrackAnimating ? 'anim' : ''" :style="terminalTrackStyle">
              <div
                v-for="(line, idx) in terminalWindow"
                :key="`${terminalStartIndex}-${idx}-rt`"
                class="log-line"
              >
                <span :class="logLineClass(line)">{{ line }}</span>
              </div>
            </div>
          </div>
          <ScreenHtmlPanel
            v-else-if="isHtmlSlot('right_top')"
            :variant="moduleAt('right_top')"
            :attack-stats="state.attack_stats"
            :region-attack-stats="state.region_attack_stats ?? []"
            :teams="state.teams"
          />
          <div v-else ref="chartElRightTop" class="chart radar-chart"></div>
        </div>

        <div
          class="panel-v-splitter"
          title="拖拽调整右侧上下高度"
          @mousedown.prevent="startResizeRightVertical"
        ></div>

        <div v-show="slotVisible('right_bottom')" class="cyber-panel panel-inner screen-slot-right-bottom">
          <h2 class="panel-title">
            <span class="panel-dot"></span> {{ moduleTitle("right_bottom") }}
          </h2>
          <ScreenLeaderboard
            v-if="moduleAt('right_bottom') === 'leaderboard'"
            :sorted-teams="leaderboardSorted"
            :score-pulse="scorePulse"
          />
          <div
            v-else-if="moduleAt('right_bottom') === 'battle_logs'"
            ref="terminalRightBottomEl"
            class="terminal-scroll terminal-logs"
          >
            <div class="terminal-track" :class="terminalTrackAnimating ? 'anim' : ''" :style="terminalTrackStyle">
              <div
                v-for="(line, idx) in terminalWindow"
                :key="`${terminalStartIndex}-${idx}-rb`"
                class="log-line"
              >
                <span :class="logLineClass(line)">{{ line }}</span>
              </div>
            </div>
          </div>
          <ScreenHtmlPanel
            v-else-if="isHtmlSlot('right_bottom')"
            :variant="moduleAt('right_bottom')"
            :attack-stats="state.attack_stats"
            :region-attack-stats="state.region_attack_stats ?? []"
            :teams="state.teams"
          />
          <div v-else ref="chartElRightBottom" class="chart pie-chart"></div>
        </div>
      </aside>
    </main>

    <footer v-if="state.screen_credits_visible !== false" class="bottom-credits">
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
import {
  DEFAULT_SCREEN_MODULES,
  SCREEN_MODULE_LABELS,
  SCREEN_SLOTS,
  isHtmlPanelModule,
  mergeScreenModulesPatch,
  normalizeScreenModules,
  type ScreenModuleId,
  type ScreenSlotId,
} from "../shared/screenLayout";
import { connectMatchWS } from "../shared/wsClient";
import ScreenHtmlPanel from "./ScreenHtmlPanel.vue";
import ScreenLeaderboard from "./ScreenLeaderboard.vue";

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
  泰州市: [120.0, 32.45],
  市区: [120.0, 32.45], // 兼容旧数据/旧事件
  市区范围: [120.0, 32.45], // 兼容别名
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
const nowTS = ref(Math.floor(Date.now() / 1000));
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
const LS_MAP_TEAM_DOCK_LAYOUT = "rb_map_team_dock_layout";
const LS_MAP_TEAM_DOCK_WIDTH = "rb_map_team_dock_width";
const LS_TITLE_H_LEFT_TOP = "rb_screen_title_h_left_top";
const LS_TITLE_H_RIGHT_TOP = "rb_screen_title_h_right_top";
const LS_TOPBAR_H = "rb_screen_topbar_h";

const fontUserScale = ref(1.12);
try {
  const s = localStorage.getItem(LS_FONT_USER);
  if (s) {
    const n = Number.parseFloat(s);
    if (Number.isFinite(n) && n >= 0.8 && n <= 2.6) fontUserScale.value = n;
  }
} catch {
  // ignore
}

/** 队伍飞线起点：左侧固定栏（与地图像素对齐）或地图内虚拟锚点 */
const mapTeamDockLayout = ref<"dock" | "map">("dock");
const mapTeamDockWidth = ref(168);
try {
  const dl = localStorage.getItem(LS_MAP_TEAM_DOCK_LAYOUT);
  if (dl === "map" || dl === "dock") mapTeamDockLayout.value = dl;
  const dw = Number.parseInt(localStorage.getItem(LS_MAP_TEAM_DOCK_WIDTH) ?? "", 10);
  if (Number.isFinite(dw) && dw >= 120 && dw <= 280) mapTeamDockWidth.value = dw;
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

const titleHLeftTop = ref(44);
const titleHRightTop = ref(44);
const topbarHeightUserPx = ref<number | null>(null);
try {
  const a = Number.parseInt(localStorage.getItem(LS_TITLE_H_LEFT_TOP) ?? "", 10);
  const b = Number.parseInt(localStorage.getItem(LS_TITLE_H_RIGHT_TOP) ?? "", 10);
  if (Number.isFinite(a) && a >= 28 && a <= 92) titleHLeftTop.value = a;
  if (Number.isFinite(b) && b >= 28 && b <= 92) titleHRightTop.value = b;
  const th = Number.parseInt(localStorage.getItem(LS_TOPBAR_H) ?? "", 10);
  if (Number.isFinite(th) && th >= 64 && th <= 180) topbarHeightUserPx.value = th;
} catch {
  // ignore
}

function titleStyle(kind: "left_top" | "right_top") {
  const h = kind === "left_top" ? titleHLeftTop.value : titleHRightTop.value;
  const baseFont = 18 * uiScale.value * fontUserScale.value;
  const scale = Math.max(0.78, Math.min(1.85, h / 44));
  const fontPx = Math.max(12, Math.round(baseFont * scale));
  return {
    height: `${h}px`,
    minHeight: `${h}px`,
    maxHeight: `${h}px`,
    fontSize: `${fontPx}px`,
  } as Record<string, string>;
}

const topbarStyle = computed<Record<string, string>>(() => {
  const h = topbarHeightUserPx.value ?? topbarHeightPx.value;
  const base = 28 * uiScale.value * fontUserScale.value;
  const scale = Math.max(0.78, Math.min(2.2, h / 90));
  const titlePx = Math.max(16, Math.round(base * scale));
  return {
    "--topbar-height": `${h}px`,
    "--topbar-title-px": `${titlePx}px`,
  };
});

let titleDrag:
  | {
      kind: "left_top" | "right_top";
      startY: number;
      startH: number;
    }
  | null = null;

function startResizeTitle(kind: "left_top" | "right_top", ev: MouseEvent) {
  titleDrag = { kind, startY: ev.clientY, startH: kind === "left_top" ? titleHLeftTop.value : titleHRightTop.value };
  const onMove = (e: MouseEvent) => {
    if (!titleDrag || titleDrag.kind !== kind) return;
    const dy = e.clientY - titleDrag.startY;
    const nh = Math.max(28, Math.min(92, Math.round(titleDrag.startH + dy)));
    if (kind === "left_top") titleHLeftTop.value = nh;
    else titleHRightTop.value = nh;
  };
  const onUp = () => {
    if (titleDrag?.kind !== kind) return;
    titleDrag = null;
    try {
      if (kind === "left_top") localStorage.setItem(LS_TITLE_H_LEFT_TOP, String(titleHLeftTop.value));
      else if (kind === "right_top") localStorage.setItem(LS_TITLE_H_RIGHT_TOP, String(titleHRightTop.value));
    } catch {
      // ignore
    }
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    void nextTick(() => {
      mapChart?.resize();
      resizeAllSlotCharts();
      updateTeamDockGeoCoords();
    });
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}

let topbarDrag: { startY: number; startH: number } | null = null;
function startResizeTopbar(ev: MouseEvent) {
  const cur = topbarHeightUserPx.value ?? topbarHeightPx.value;
  topbarDrag = { startY: ev.clientY, startH: cur };
  const onMove = (e: MouseEvent) => {
    if (!topbarDrag) return;
    const dy = e.clientY - topbarDrag.startY;
    const nh = Math.max(64, Math.min(180, Math.round(topbarDrag.startH + dy)));
    topbarHeightUserPx.value = nh;
  };
  const onUp = () => {
    if (!topbarDrag) return;
    topbarDrag = null;
    try {
      if (topbarHeightUserPx.value != null) localStorage.setItem(LS_TOPBAR_H, String(topbarHeightUserPx.value));
    } catch {
      // ignore
    }
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
  };
  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
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
    localStorage.removeItem(LS_TITLE_H_LEFT_TOP);
    localStorage.removeItem(LS_TITLE_H_RIGHT_TOP);
    localStorage.removeItem(LS_TOPBAR_H);
  } catch {
    // ignore
  }
  useCustomPanelWidths.value = false;
  leftPanelWidthPx.value = 380;
  rightPanelWidthPx.value = 380;
  leftTopRatio.value = 40;
  rightTopRatio.value = 40;
  titleHLeftTop.value = 44;
  titleHRightTop.value = 44;
  topbarHeightUserPx.value = null;
  updateViewportAdaptiveVars();
  nextTick(() => {
    mapChart?.resize();
    resizeAllSlotCharts();
    updateTeamDockGeoCoords();
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
    resizeAllSlotCharts();
  };
  const onUp = () => {
    resizeVDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    resizeAllSlotCharts();
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
    resizeAllSlotCharts();
  };
  const onUp = () => {
    resizeVDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    resizeAllSlotCharts();
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
    resizeAllSlotCharts();
  };
  const onUp = () => {
    resizeDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    mapChart?.resize();
    resizeAllSlotCharts();
    void nextTick(() => updateTeamDockGeoCoords());
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
    resizeAllSlotCharts();
  };
  const onUp = () => {
    resizeDrag = null;
    persistLayout();
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    mapChart?.resize();
    resizeAllSlotCharts();
    void nextTick(() => updateTeamDockGeoCoords());
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
    resizeAllSlotCharts();
    refreshSlotChartData();
    if (mapChart) refreshMapSeriesOnly(currentMapType);
    updateTeamDockGeoCoords();
  });
});

watch(uiScale, () => {
  nextTick(() => {
    refreshSlotChartData();
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

function pulseTeamScore(teamId: number) {
  scorePulse[teamId] = Date.now();
  window.setTimeout(() => {
    delete scorePulse[teamId];
  }, 900);
}

// 大屏默认不展示回放控制台（回放仍可通过 admin 下发控制事件工作）。
const showReplayUI = computed(() => false);
const countdownText = computed(() => {
  const end = Number(state.countdown_end_ts ?? 0);
  if (!Number.isFinite(end) || end <= 0) return "";
  const diff = end - nowTS.value;
  if (diff <= 0) return "00:00:00";
  const h = Math.floor(diff / 3600);
  const m = Math.floor((diff % 3600) / 60);
  const s = diff % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
});

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
  countdown_end_ts: 0,
  map_type: "china",
  leaderboard_visible: true,
  teams: [],
  attack_stats: attackStatsDefault,
  region_attack_stats: [],
  panels: { "panel-leaderboard": true },
  screen_modules: { ...DEFAULT_SCREEN_MODULES },
});

const leaderboardSorted = computed(() => {
  return [...state.teams].sort((a, b) => b.score - a.score);
});

/** 左侧队伍栏：按 id 稳定排序，与飞线锚点行一一对应 */
const teamDockSorted = computed(() => [...state.teams].sort((a, b) => a.id - b.id));

/** 侧栏内队伍行均分高度，避免挤在一团；单行时垂直居中 */
const teamDockGridStyle = computed(() => {
  const n = teamDockSorted.value.length;
  if (n <= 0) return {};
  if (n === 1) {
    return {
      display: "flex",
      flexDirection: "column",
      justifyContent: "center",
    } as Record<string, string>;
  }
  return {
    display: "grid",
    gridTemplateRows: `repeat(${n}, minmax(0, 1fr))`,
    gap: "3px",
    alignContent: "stretch",
  };
});

const normalizedScreenModules = computed(() => normalizeScreenModules(state.screen_modules));

function moduleAt(slot: ScreenSlotId): ScreenModuleId {
  return normalizedScreenModules.value[slot];
}

function moduleTitle(slot: ScreenSlotId): string {
  return SCREEN_MODULE_LABELS[moduleAt(slot)];
}

function slotVisible(slot: ScreenSlotId): boolean {
  const key = `panel-slot-${slot}`;
  if (state.panels?.[key] === false) return false;
  if (slot === "left_top" && state.panels?.["panel-leaderboard"] === false) return false;
  return true;
}

function isHtmlSlot(slot: ScreenSlotId): boolean {
  return isHtmlPanelModule(moduleAt(slot));
}

const chartElLeftTop = ref<HTMLDivElement | null>(null);
const chartElLeftBottom = ref<HTMLDivElement | null>(null);
const chartElRightTop = ref<HTMLDivElement | null>(null);
const chartElRightBottom = ref<HTMLDivElement | null>(null);
const terminalLeftTopEl = ref<HTMLElement | null>(null);
const terminalLeftBottomEl = ref<HTMLElement | null>(null);
const terminalRightBottomEl = ref<HTMLElement | null>(null);

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
const mapTeamDockEl = ref<HTMLElement | null>(null);
/** 队伍栏各行与地图左缘对齐后的地理坐标（用于队伍→区县飞线起点） */
const teamDockGeoCoords = reactive<Record<number, [number, number]>>({});

let mapChart: echarts.ECharts | null = null;
const slotChartInstances: Partial<Record<ScreenSlotId, echarts.ECharts>> = {};
let lastSlotLayoutKey = "";

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
type TeamSourceMarker = {
  coord: [number, number];
  name: string;
  teamType: string;
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
let mapTeamSourceMarkers: TeamSourceMarker[] = [];
let mapLineRefreshTimer: number | undefined;
let mapHasGeo = true;
const mapImpactActive = ref(false);
let mapImpactTimer: number | undefined;
let mapGeoRoamRaf: number | undefined;
function onMapGeoRoamForDock() {
  if (mapGeoRoamRaf != null) return;
  mapGeoRoamRaf = window.requestAnimationFrame(() => {
    mapGeoRoamRaf = undefined;
    updateTeamDockGeoCoords();
  });
}

// 事件驱动的“脉冲点”（用于让飞线起止点更有层次）
const cityPulse = reactive<Record<string, number>>({});
const CITY_PULSE_LIFETIME_MS = 3500;
const CITY_PULSE_MAX = 8;
const MAP_LINE_TTL_MS = 18000;
const MAP_LINE_MAX = 90;
const MAP_HIT_MARK_TTL_MS = 7600;
const MAP_HIT_MARK_MAX = 60;
const MAP_REGION_FLASH_TTL_MS = 1800;
const MAP_TEAM_SOURCE_TTL_MS = 9000;

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

function buildTeamSourceMarkers(nowTs: number = Date.now()) {
  mapTeamSourceMarkers = mapTeamSourceMarkers.filter((it) => nowTs - it.createdAt <= it.ttlMs);
  return mapTeamSourceMarkers.map((it) => ({
    name: it.name,
    value: it.coord,
    teamType: it.teamType,
  }));
}

/** 左侧栏模式下队伍起点由 DOM 表现，不再在 geo 上叠散点 */
function getTeamSourceScatterData() {
  if (mapTeamDockLayout.value === "dock") return [];
  return buildTeamSourceMarkers();
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

function setRadarOption(chart: echarts.ECharts | null, teams: TeamDTO[]) {
  if (!chart) return;
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

  chart.setOption({
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

/** 柱状图坐标轴、标签等：投屏时略放大 */
function barAxisFontPx(base: number) {
  return Math.max(11, Math.round(base * uiScale.value * fontUserScale.value * 1.08));
}

function setPieOption(chart: echarts.ECharts | null, attackStats: { name: string; value: number }[]) {
  if (!chart) return;
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
    chart.setOption({
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

  chart.setOption({
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

function mkHzGradientLR(c1: string, c2: string) {
  const g = (echarts as any).graphic;
  return g?.LinearGradient ? new g.LinearGradient(1, 0, 0, 0, [{ offset: 0, color: c1 }, { offset: 1, color: c2 }]) : c1;
}

function setRegionRankBarOption(chart: echarts.ECharts | null, stats: { name: string; value: number }[]) {
  if (!chart) return;
  const list = [...(stats ?? [])].sort((a, b) => b.value - a.value).slice(0, 12);
  if (!list.length) {
    chart.setOption({
      backgroundColor: "transparent",
      title: {
        text: "暂无受攻击地域数据",
        left: "center",
        top: "center",
        textStyle: { color: "rgba(148,163,184,0.9)", fontSize: pieTextPx(13) },
      },
      xAxis: { show: false },
      yAxis: { show: false },
      series: [],
    });
    return;
  }
  const names = list.map((x) => x.name).reverse();
  const vals = list.map((x) => x.value).reverse();
  chart.setOption({
    backgroundColor: "transparent",
    grid: { left: 4, right: 44, top: 10, bottom: 10, containLabel: true },
    xAxis: {
      type: "value",
      splitLine: { lineStyle: { color: "rgba(0,243,255,0.06)" } },
      axisLabel: { color: "#94a3b8", fontSize: barAxisFontPx(10) },
    },
    yAxis: {
      type: "category",
      data: names,
      axisLabel: { color: "#e2e8f0", fontSize: barAxisFontPx(11) },
      axisLine: { lineStyle: { color: "rgba(0,243,255,0.2)" } },
      axisTick: { show: false },
    },
    series: [
      {
        type: "bar",
        data: vals,
        barMaxWidth: 28,
        barCategoryGap: "20%",
        itemStyle: {
          color: mkHzGradientLR("#22d3ee", "#a855f7"),
          borderRadius: [0, 4, 4, 0],
        },
        label: { show: true, position: "right", color: "#94a3b8", fontSize: barAxisFontPx(10), distance: 6 },
      },
    ],
  });
}

function setTeamScoreBarsOption(chart: echarts.ECharts | null, teams: TeamDTO[]) {
  if (!chart) return;
  const list = [...teams].sort((a, b) => b.score - a.score).slice(0, 12);
  if (!list.length) {
    chart.setOption({
      backgroundColor: "transparent",
      title: {
        text: "暂无队伍",
        left: "center",
        top: "center",
        textStyle: { color: "rgba(148,163,184,0.9)", fontSize: pieTextPx(13) },
      },
      series: [],
    });
    return;
  }
  const rows = list.map((t) => ({
    name: t.name,
    value: t.score,
    color: t.type === "red" ? mkHzGradientLR("#ff4d4d", "#fb7185") : mkHzGradientLR("#38bdf8", "#22d3ee"),
  }));
  const rev = rows.reverse();
  chart.setOption({
    backgroundColor: "transparent",
    grid: { left: 4, right: 12, top: 10, bottom: 10, containLabel: true },
    xAxis: {
      type: "value",
      splitLine: { lineStyle: { color: "rgba(0,243,255,0.06)" } },
      axisLabel: { color: "#94a3b8", fontSize: barAxisFontPx(10) },
    },
    yAxis: {
      type: "category",
      data: rev.map((r) => r.name),
      axisLabel: { color: "#e2e8f0", fontSize: barAxisFontPx(11) },
      axisTick: { show: false },
    },
    series: [
      {
        type: "bar",
        data: rev.map((r) => ({ value: r.value, itemStyle: { color: r.color, borderRadius: [0, 4, 4, 0] } })),
        barMaxWidth: 13,
        barCategoryGap: "56%",
      },
    ],
  });
}

function setAttackTypeBarsOption(chart: echarts.ECharts | null, stats: { name: string; value: number }[]) {
  if (!chart) return;
  const list = [...(stats ?? [])].sort((a, b) => b.value - a.value).slice(0, 10);
  if (!list.length) {
    chart.setOption({
      backgroundColor: "transparent",
      title: {
        text: "暂无战术数据",
        left: "center",
        top: "center",
        textStyle: { color: "rgba(148,163,184,0.9)", fontSize: pieTextPx(13) },
      },
      series: [],
    });
    return;
  }
  const names = list.map((x) => (x.name.length > 8 ? `${x.name.slice(0, 8)}…` : x.name)).reverse();
  const vals = list.map((x) => x.value).reverse();
  chart.setOption({
    backgroundColor: "transparent",
    grid: { left: 4, right: 10, top: 10, bottom: 10, containLabel: true },
    xAxis: {
      type: "value",
      splitLine: { lineStyle: { color: "rgba(0,243,255,0.06)" } },
      axisLabel: { color: "#94a3b8", fontSize: barAxisFontPx(10) },
    },
    yAxis: {
      type: "category",
      data: names,
      axisLabel: { color: "#e2e8f0", fontSize: barAxisFontPx(10) },
      axisTick: { show: false },
    },
    series: [
      {
        type: "bar",
        data: vals,
        barMaxWidth: 12,
        barCategoryGap: "58%",
        itemStyle: { color: mkHzGradientLR("#fbbf24", "#f97316"), borderRadius: [0, 4, 4, 0] },
      },
    ],
  });
}

function setRedBlueTopCompareOption(chart: echarts.ECharts | null) {
  if (!chart) return;
  const red = redTopTeam.value;
  const blue = blueTopTeam.value;
  const rScore = red?.score ?? 0;
  const bScore = blue?.score ?? 0;
  chart.setOption({
    backgroundColor: "transparent",
    grid: { left: 4, right: 12, top: 14, bottom: 14, containLabel: true },
    xAxis: {
      type: "value",
      splitLine: { lineStyle: { color: "rgba(0,243,255,0.06)" } },
      axisLabel: { color: "#94a3b8", fontSize: barAxisFontPx(10) },
    },
    yAxis: {
      type: "category",
      data: ["蓝方首席", "红方首席"],
      axisLabel: { color: "#e2e8f0", fontSize: barAxisFontPx(12), fontWeight: 600 },
      axisTick: { show: false },
    },
    series: [
      {
        type: "bar",
        data: [
          { value: bScore, itemStyle: { color: mkHzGradientLR("#38bdf8", "#0ea5e9"), borderRadius: [0, 4, 4, 0] } },
          { value: rScore, itemStyle: { color: mkHzGradientLR("#f87171", "#ef4444"), borderRadius: [0, 4, 4, 0] } },
        ],
        barMaxWidth: 18,
        barCategoryGap: "68%",
        label: { show: true, position: "right", color: "#e2e8f0", fontSize: barAxisFontPx(11), fontWeight: 700 },
      },
    ],
  });
}

function setPostureGaugeOption(chart: echarts.ECharts | null) {
  if (!chart) return;
  const hits = (state.attack_stats ?? []).reduce((s, x) => s + Number(x.value ?? 0), 0);
  const regions = (state.region_attack_stats ?? []).length;
  const blended = Math.min(100, Math.round(Math.sqrt(hits + 1) * 6 + regions * 0.8));
  const levelText = blended >= 80 ? "高压" : blended >= 55 ? "警戒" : blended >= 30 ? "关注" : "平稳";
  const showLevelText = levelText !== "关注";
  chart.setOption({
    backgroundColor: "transparent",
    title: {
      text: "综合态势强度",
      left: "center",
      top: "1.8%",
      textStyle: { color: "rgba(236, 252, 255, 0.98)", fontSize: barAxisFontPx(13), fontWeight: 800, textShadowBlur: 10, textShadowColor: "rgba(34,211,238,0.35)" },
    },
    series: [
      {
        type: "gauge",
        radius: "84%",
        center: ["50%", "56%"],
        min: 0,
        max: 100,
        splitNumber: 10,
        progress: {
          show: true,
          width: 14,
          roundCap: true,
          itemStyle: {
            color: (echarts as any).graphic?.LinearGradient
              ? new (echarts as any).graphic.LinearGradient(0, 0, 1, 0, [
                  { offset: 0, color: "#22d3ee" },
                  { offset: 0.55, color: "#a855f7" },
                  { offset: 1, color: "#f97316" },
                ])
              : "#22d3ee",
            shadowBlur: 14,
            shadowColor: "rgba(56,189,248,0.45)",
          },
        },
        axisLine: {
          lineStyle: {
            width: 14,
            color: [
              [0.35, "rgba(8,145,178,0.42)"],
              [0.65, "rgba(168,85,247,0.42)"],
              [1, "rgba(234,88,12,0.42)"],
            ],
          },
        },
        pointer: {
          show: true,
          length: "60%",
          width: 6,
          itemStyle: { color: "#fef08a", shadowBlur: 12, shadowColor: "rgba(253,230,138,0.6)" },
        },
        anchor: {
          show: true,
          showAbove: true,
          size: 10,
          itemStyle: {
            color: "#e2e8f0",
            borderWidth: 3,
            borderColor: "rgba(34,211,238,0.55)",
            shadowBlur: 10,
            shadowColor: "rgba(34,211,238,0.45)",
          },
        },
        axisTick: { distance: -18, length: 7, lineStyle: { color: "rgba(148,163,184,0.5)", width: 1.2 } },
        splitLine: { distance: -20, length: 14, lineStyle: { color: "rgba(226,232,240,0.42)", width: 1.4 } },
        axisLabel: { distance: -34, color: "rgba(203,213,225,0.95)", fontSize: barAxisFontPx(9), fontWeight: 700 },
        detail: {
          valueAnimation: true,
          fontSize: barAxisFontPx(24),
          fontWeight: 900,
          color: "#f8fafc",
          offsetCenter: [0, showLevelText ? "22%" : "18%"],
          formatter: (v: number) => (showLevelText ? `{a|${Math.round(v)}}\n{b|${levelText}}` : `{a|${Math.round(v)}}`),
          rich: {
            a: { fontSize: barAxisFontPx(24), fontWeight: 900, color: "#f8fafc" },
            b: { fontSize: barAxisFontPx(11), fontWeight: 700, color: "rgba(125,211,252,0.95)", lineHeight: 18 },
          },
        },
        data: [{ value: blended, name: "指数" }],
      },
    ],
  });
}

function disposeSlotChart(slot: ScreenSlotId) {
  const ch = slotChartInstances[slot];
  if (ch) {
    ch.dispose();
    delete slotChartInstances[slot];
  }
}

function getChartElForSlot(slot: ScreenSlotId): HTMLElement | null {
  switch (slot) {
    case "left_top":
      return chartElLeftTop.value;
    case "left_bottom":
      return chartElLeftBottom.value;
    case "right_top":
      return chartElRightTop.value;
    case "right_bottom":
      return chartElRightBottom.value;
    default:
      return null;
  }
}

function slotNeedsChart(mod: ScreenModuleId): boolean {
  if (mod === "leaderboard" || mod === "battle_logs") return false;
  if (isHtmlPanelModule(mod)) return false;
  return true;
}

function ensureSlotChartsMounted() {
  const mods = normalizedScreenModules.value;
  const key = SCREEN_SLOTS.map((s) => `${s}:${mods[s]}`).join("|");
  if (key !== lastSlotLayoutKey) {
    for (const s of SCREEN_SLOTS) disposeSlotChart(s);
    lastSlotLayoutKey = key;
  }
  for (const slot of SCREEN_SLOTS) {
    const modSlot = mods[slot]!;
    if (!slotNeedsChart(modSlot)) continue;
    const el = getChartElForSlot(slot);
    if (!el) continue;
    if (!slotChartInstances[slot]) {
      slotChartInstances[slot] = echarts.init(el);
    }
  }
}

function refreshSlotChartData() {
  for (const slot of SCREEN_SLOTS) {
    const ch = slotChartInstances[slot];
    if (!ch) continue;
    const mod = normalizedScreenModules.value[slot]!;
    switch (mod) {
      case "radar_power":
        setRadarOption(ch, state.teams);
        break;
      case "region_attack_rank":
        setRegionRankBarOption(ch, state.region_attack_stats ?? []);
        break;
      case "attack_type_pie":
        setPieOption(ch, state.attack_stats);
        break;
      case "team_score_bars":
        setTeamScoreBarsOption(ch, state.teams);
        break;
      case "attack_type_bars":
        setAttackTypeBarsOption(ch, state.attack_stats);
        break;
      case "red_blue_top_compare":
        setRedBlueTopCompareOption(ch);
        break;
      case "posture_gauge":
        setPostureGaugeOption(ch);
        break;
      default:
        break;
    }
  }
}

function resizeAllSlotCharts() {
  for (const ch of Object.values(slotChartInstances)) {
    ch?.resize();
  }
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

/** 将左侧队伍栏各行与地图左缘对齐点转为经纬度，供队伍→区县飞线使用 */
function updateTeamDockGeoCoords() {
  if (!mapChart || !mapEl.value || !mapHasGeo) return;
  if (mapTeamDockLayout.value !== "dock" || !mapTeamDockEl.value) return;
  const mapRect = mapEl.value.getBoundingClientRect();
  if (mapRect.width < 12 || mapRect.height < 12) return;
  const rows = mapTeamDockEl.value.querySelectorAll("[data-team-id]");
  rows.forEach((row) => {
    const id = Number((row as HTMLElement).dataset.teamId);
    if (!Number.isFinite(id)) return;
    const rr = row.getBoundingClientRect();
    const x = Math.min(32, Math.max(4, mapRect.width * 0.03));
    const y = rr.top + rr.height / 2 - mapRect.top;
    if (!Number.isFinite(y) || y < -4 || y > mapRect.height + 4) return;
    try {
      const geo = mapChart!.convertFromPixel({ geoIndex: 0 }, [x, y]);
      if (geo && Array.isArray(geo) && geo.length >= 2) {
        const lng = Number(geo[0]);
        const lat = Number(geo[1]);
        if (Number.isFinite(lng) && Number.isFinite(lat)) {
          teamDockGeoCoords[id] = [lng, lat];
        }
      }
    } catch {
      // ignore
    }
  });
}

function renderMap(hasMap = true, mapType: "china" | "taizhou" = currentMapType) {
  if (!mapChart) return;
  mapHasGeo = hasMap;
  const currentCities = mapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const points = Object.keys(currentCities).map((key) => ({ name: key, value: currentCities[key] }));

  const pulsePoints = getCityPulsePoints(mapType);
  const { successData, defenseData, traceData, sourceSuccessData } = buildHitMarkerSeriesData();
  const teamSourceData = getTeamSourceScatterData();
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
          zlevel: 4,
          rippleEffect: { brushType: "stroke", scale: 3.6 },
          symbol: "circle",
          symbolSize: 8,
          label: {
            show: true,
            formatter: (p: any) => `队伍:${String(p.name ?? "")}`,
            position: "right",
            color: "#e2e8f0",
            fontSize: pieTextPx(10),
            backgroundColor: "rgba(2,6,23,0.72)",
            borderColor: "rgba(56,189,248,0.28)",
            borderWidth: 1,
            borderRadius: 4,
            padding: [2, 5],
          },
          itemStyle: {
            color: (p: any) => (String(p.data?.teamType ?? "") === "red" ? "#fb7185" : "#67e8f9"),
            shadowBlur: 10,
            shadowColor: "rgba(255,255,255,0.2)",
          },
          data: teamSourceData,
        },
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
  void nextTick(() => updateTeamDockGeoCoords());
}

function refreshMapSeriesOnly(mapType: "china" | "taizhou") {
  if (!mapChart) return;
  const hasMap = mapHasGeo;
  const currentCities = mapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const points = Object.keys(currentCities).map((key) => ({ name: key, value: currentCities[key] }));
  const pulsePoints = getCityPulsePoints(mapType);
  const { successData, defenseData, traceData, sourceSuccessData } = buildHitMarkerSeriesData();
  const teamSourceData = getTeamSourceScatterData();
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
        zlevel: 4,
        rippleEffect: { brushType: "stroke", scale: 3.6 },
        symbol: "circle",
        symbolSize: 8,
        label: {
          show: true,
          formatter: (p: any) => `队伍:${String(p.name ?? "")}`,
          position: "right",
          color: "#e2e8f0",
          fontSize: pieTextPx(10),
          backgroundColor: "rgba(2,6,23,0.72)",
          borderColor: "rgba(56,189,248,0.28)",
          borderWidth: 1,
          borderRadius: 4,
          padding: [2, 5],
        },
        itemStyle: {
          color: (p: any) => (String(p.data?.teamType ?? "") === "red" ? "#fb7185" : "#67e8f9"),
          shadowBlur: 10,
          shadowColor: "rgba(255,255,255,0.2)",
        },
        data: teamSourceData,
      },
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
  void nextTick(() => updateTeamDockGeoCoords());
}

async function loadMapData(mapType: "china" | "taizhou") {
  const localUrl = mapType === "taizhou" ? `/geojson/${mapType}.json` : "";
  const url = `${apiBaseHttp}/api/geojson/${mapType}`;
  try {
    let geoJson: any = null;
    if (localUrl) {
      const localRes = await fetch(localUrl);
      if (localRes.ok) {
        geoJson = await localRes.json();
      }
    }
    if (!geoJson) {
      const res = await fetch(url, { headers: authHeaders });
      if (!res.ok) throw new Error(`geojson proxy status=${res.status}`);
      geoJson = await res.json();
    }
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
  const nextMapType: "china" | "taizhou" = state.map_type === "taizhou" ? "taizhou" : "china";
  if (mapChart && nextMapType !== currentMapType) {
    currentMapType = nextMapType;
    loadMapData(currentMapType);
  }
  nextTick(() => {
    ensureSlotChartsMounted();
    refreshSlotChartData();
    updateTeamDockGeoCoords();
  });
}

const TEAM_COLORS = [
  "#ff2a2a",
  "#00f3ff",
  "#ffc107",
  "#ff007f",
  "#22c55e",
  "#a855f7",
  "#f97316",
  "#06b6d4",
  "#eab308",
  "#ec4899",
  "#84cc16",
  "#8b5cf6",
];

function teamLineColorById(teamId: number | undefined): string {
  const tid = Number(teamId ?? 0);
  if (!Number.isFinite(tid) || tid <= 0) return "rgba(148, 163, 184, 0.55)";
  return TEAM_COLORS[Math.abs(tid) % TEAM_COLORS.length];
}

/** 城市/队伍源飞线均按队伍 id 取色，便于多队伍区分 */
function getTeamLineColor(teamId: number | undefined, teamType?: string, _sourceMode?: "city" | "team") {
  const tid = Number(teamId ?? 0);
  const fallback = teamType === "red" ? "#ff2a2a" : "#00f3ff";
  if (!Number.isFinite(tid) || tid <= 0) return fallback;
  return teamLineColorById(tid);
}

function getTeamSourceCoord(teamId: number | undefined, mapType: "china" | "taizhou"): [number, number] {
  const tid = Number(teamId ?? 0);
  if (mapTeamDockLayout.value === "dock") {
    const g = teamDockGeoCoords[tid];
    if (g) return g;
  }
  const teamsSorted = [...state.teams].sort((a, b) => a.id - b.id);
  const idx = Math.max(0, teamsSorted.findIndex((t) => t.id === tid));
  if (mapType === "taizhou") {
    const baseLon = 119.64;
    const baseLat = 32.80;
    const step = 0.12;
    const row = idx % 6;
    return [baseLon, baseLat - row * step];
  }
  const baseLon = 79.5;
  const baseLat = 44.5;
  const step = 2.2;
  const row = idx % 7;
  return [baseLon, baseLat - row * step];
}

function getTeamName(teamId: number | undefined): string {
  const teamsSorted = [...state.teams].sort((a, b) => a.id - b.id);
  const team = teamsSorted.find((t) => t.id === Number(teamId ?? 0));
  return team?.name ?? `#${Number(teamId ?? 0) || 0}`;
}

watch(mapTeamDockLayout, (v) => {
  try {
    localStorage.setItem(LS_MAP_TEAM_DOCK_LAYOUT, v);
  } catch {
    // ignore
  }
  nextTick(() => {
    mapChart?.resize();
    updateTeamDockGeoCoords();
    if (mapChart) refreshMapSeriesOnly(currentMapType);
  });
});

watch(mapTeamDockWidth, (v) => {
  const n = Math.round(v);
  try {
    localStorage.setItem(LS_MAP_TEAM_DOCK_WIDTH, String(n));
  } catch {
    // ignore
  }
  nextTick(() => {
    mapChart?.resize();
    updateTeamDockGeoCoords();
  });
});

watch(
  () => state.teams.map((t) => `${t.id}:${t.name}`),
  () => nextTick(() => updateTeamDockGeoCoords())
);

function updateMapLineByAttack(data: any) {
  const currentCities = currentMapType === "taizhou" ? TAIZHOU_CITIES : CITIES;
  const cityNames = Object.keys(currentCities);

  const sourceMode = String(data.source_mode ?? "city") === "team" ? "team" : "city";
  let sourceCity = "";
  if (sourceMode === "city") {
    sourceCity =
      data.source_city && currentCities[data.source_city]
        ? data.source_city
        : cityNames[Math.floor(Math.random() * cityNames.length)];
  }

  const teamID = Number(data.source_team_id ?? data.team_id);
  const lineColor = getTeamLineColor(teamID, data.team_type, sourceMode);
  let lineWidth = 1.8;
  const status = data.status as AttackStatus;
  let lineDash: string | undefined;
  const sourceCoord = sourceMode === "team" ? getTeamSourceCoord(teamID, currentMapType) : currentCities[sourceCity];
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

  // 同区县攻击同区县：起点轻微偏移，避免飞线完全重叠看不见。
  if (sourceMode === "city" && sourceCity && sourceCity === String(data.target_city ?? "")) {
    const t = Date.now() / 1000;
    const ox = Math.cos(t) * 0.26;
    const oy = Math.sin(t) * 0.18;
    drawSource = [sourceCoord[0] + ox, sourceCoord[1] + oy];
  }

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

  if (sourceMode === "team" && mapTeamDockLayout.value !== "dock") {
    mapTeamSourceMarkers.push({
      coord: sourceCoord,
      name: getTeamName(teamID),
      teamType: String(data.team_type ?? ""),
      createdAt: Date.now(),
      ttlMs: MAP_TEAM_SOURCE_TTL_MS,
    });
    if (mapTeamSourceMarkers.length > 22) mapTeamSourceMarkers.shift();
  }

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
    state.region_attack_stats = initState.region_attack_stats ?? [];
    state.panels = initState.panels;
    state.screen_modules = normalizeScreenModules(initState.screen_modules);

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
    updateFromState();

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
    if (animate) refreshSlotChartData();
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
        `[SYSTEM] 回放切换地图为 ${mapType === "taizhou" ? "泰州市态势" : "全国态势"}`
      );
      refreshSlotChartData();
    }
    return;
  }

  if (eventType === "set_screen_modules") {
    const mods = data.modules ?? {};
    state.screen_modules = mergeScreenModulesPatch(normalizeScreenModules(state.screen_modules), mods);
    if (animate) pushLog("[SYSTEM] 回放：大屏模块配置已更新");
    return;
  }

  if (eventType === "toggle_panel") {
    const panelID = data.panel_id ?? "";
    const visible = !!data.visible;
    if (panelID) {
      state.panels[panelID] = visible;
      if (panelID === "panel-leaderboard") {
        state.leaderboard_visible = visible;
        state.panels["panel-slot-left_top"] = visible;
      }
      if (panelID === "panel-slot-left_top") {
        const mod = normalizeScreenModules(state.screen_modules).left_top;
        if (mod === "leaderboard") {
          state.leaderboard_visible = visible;
          state.panels["panel-leaderboard"] = visible;
        }
      }
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
        refreshSlotChartData();
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

    const targetCity = String(data.target_city ?? "").trim();
    if (targetCity) {
      const rs = state.region_attack_stats ?? [];
      state.region_attack_stats = rs;
      const rst = rs.find((s) => s.name === targetCity);
      if (rst) rst.value += 1;
      else rs.push({ name: targetCity, value: 1 });
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

      refreshSlotChartData();
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
    nowTS.value = Math.floor(now.getTime() / 1000);
    clock.value = now.toLocaleTimeString("en-US", { hour12: false });
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, "0");
    const d = String(now.getDate()).padStart(2, "0");
    dateStr.value = `${y}-${m}-${d}`;
  }, 1000);

  startTerminalLoop();

  await nextTick();
  // 计算日志可视行数，确保框能更“填满”
  try {
    const el =
      terminalScrollEl.value || terminalLeftTopEl.value || terminalLeftBottomEl.value || terminalRightBottomEl.value;
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
  if (!mapEl.value) return;

  mapChart = echarts.init(mapEl.value);
  mapChart.on("georoam", onMapGeoRoamForDock);

  // 让飞线随时间渐隐，不需要新事件也能保持“持续一段时间”的视觉效果。
  mapLineRefreshTimer = window.setInterval(() => {
    if (!mapChart) return;
    refreshMapSeriesOnly(currentMapType);
  }, 900);

  resizeHandler = () => {
    updateViewportAdaptiveVars();
    mapChart?.resize();
    resizeAllSlotCharts();
    void nextTick(() => updateTeamDockGeoCoords());
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
        state.screen_modules = normalizeScreenModules(state.screen_modules);
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
          state.screen_modules = normalizeScreenModules(state.screen_modules);
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
        state.screen_modules = normalizeScreenModules(state.screen_modules);
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
        } else if (m.event === "set_screen_modules") {
          pushLog("[SYSTEM] 已更新大屏四槽位模块配置");
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
  for (const s of SCREEN_SLOTS) disposeSlotChart(s);
  lastSlotLayoutKey = "";
  if (terminalLoopTimer) window.clearInterval(terminalLoopTimer);
  if (terminalFlushTimer) window.clearInterval(terminalFlushTimer);
  if (mapLineRefreshTimer) window.clearInterval(mapLineRefreshTimer);
  if (mapImpactTimer) window.clearTimeout(mapImpactTimer);
  if (mapGeoRoamRaf != null) {
    window.cancelAnimationFrame(mapGeoRoamRaf);
    mapGeoRoamRaf = undefined;
  }
  if (mapChart) {
    mapChart.off("georoam", onMapGeoRoamForDock);
    mapChart.dispose();
    mapChart = null;
  }
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
  font-size: var(--topbar-title-px, calc(28px * var(--ui-scale) * var(--font-user, 1)));
  font-weight: 700;
  letter-spacing: 0.08em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.topbar-splitter {
  height: 10px;
  cursor: row-resize;
  position: relative;
  z-index: 11;
  opacity: 0.7;
}
.topbar-splitter::after {
  content: "";
  position: absolute;
  left: 34%;
  right: 34%;
  top: 50%;
  height: 2px;
  transform: translateY(-50%);
  border-radius: 2px;
  background: rgba(56, 189, 248, 0.28);
}
.topbar-splitter:hover::after {
  background: rgba(56, 189, 248, 0.58);
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
.topbar-countdown {
  margin-right: 12px;
  padding: 4px 10px;
  border-radius: 10px;
  border: 1px solid rgba(248, 113, 113, 0.45);
  background: rgba(127, 29, 29, 0.22);
  color: #fecaca;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.countdown-label {
  font-size: calc(10px * var(--ui-scale) * var(--font-user, 1));
  letter-spacing: 0.12em;
  opacity: 0.9;
}
.countdown-value {
  font-size: calc(16px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 800;
  letter-spacing: 0.06em;
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

/* 左右栏上下槽位高度比例（与拖拽 leftTopRatio / rightTopRatio 一致） */
.panel-left .screen-slot-left-top {
  flex: 0 0 40%;
  max-height: 40%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.panel-left .screen-slot-left-bottom {
  flex: 1 1 60%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.panel-right .screen-slot-right-top {
  flex: 0 0 40%;
  max-height: 40%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.panel-right .screen-slot-right-bottom {
  flex: 1 1 60%;
  min-height: 0;
  display: flex;
  flex-direction: column;
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
  margin: 0;
  font-size: calc(18px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 700;
  color: var(--neon-blue);
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid rgba(0, 243, 255, 0.2);
  box-sizing: border-box;
  padding: 0 calc(2px * var(--ui-scale));
}
.title-splitter {
  flex: 0 0 auto;
  height: 10px;
  position: relative;
  cursor: row-resize;
  opacity: 0.7;
}
.title-splitter::after {
  content: "";
  position: absolute;
  left: 18%;
  right: 18%;
  top: 50%;
  height: 2px;
  transform: translateY(-50%);
  border-radius: 2px;
  background: rgba(56, 189, 248, 0.35);
}
.title-splitter:hover::after {
  background: rgba(56, 189, 248, 0.65);
}

.panel-dot {
  width: calc(10px * var(--ui-scale));
  height: calc(4px * var(--ui-scale));
  background: var(--neon-blue);
  box-shadow: 0 0 8px rgba(0, 243, 255, 0.6);
}

.panel-inner .chart {
  flex: 1;
  min-height: 140px;
  width: 100%;
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
/* ==========================================
   🏆 战队实时得分榜：赛博徽章与动态悬浮系统
========================================== */

/* 基础项：暗黑科幻背景与过渡动画 */
.leader-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: calc(10px * var(--ui-scale)) calc(12px * var(--ui-scale));
  background: linear-gradient(90deg, rgba(3, 11, 26, 0.8) 0%, rgba(2, 6, 23, 0.9) 100%);
  border-radius: 4px;
  border: 1px solid rgba(0, 243, 255, 0.05);
  border-left: 4px solid rgba(0, 243, 255, 0.3); /* 默认左侧指示条 */
  margin-bottom: calc(10px * var(--ui-scale));
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  position: relative;
  overflow: hidden;
}

/* 交互：每一行增加悬浮时的高亮呼吸边框与位移 */
.leader-item:hover {
  transform: translateX(8px) scale(1.02); /* 向右浮出并微缩放 */
  background: linear-gradient(90deg, rgba(0, 243, 255, 0.15) 0%, rgba(2, 6, 23, 0.9) 100%);
  border-color: rgba(0, 243, 255, 0.6);
  border-left: 4px solid #00f3ff;
  box-shadow: 0 0 15px rgba(0, 243, 255, 0.4), inset 0 0 10px rgba(0, 243, 255, 0.2);
  z-index: 10;
}

/* 特效：鼠标悬浮时增加快速扫光动效 */
.leader-item::after {
  content: '';
  position: absolute;
  top: 0; left: -100%;
  width: 50%; height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.1), transparent);
  transform: skewX(-20deg);
  transition: 0.5s;
}
.leader-item:hover::after {
  left: 150%;
}

/* 名次文本基础设置 */
.leader-rank {
  width: calc(32px * var(--ui-scale));
  text-align: center;
  font-size: calc(15px * var(--ui-scale) * var(--font-user, 1));
  font-weight: 900;
  color: rgba(229, 231, 235, 0.4); /* 4名以后的灰暗色 */
  transition: all 0.3s ease;
}
/* ==========================================
   追加：前三名奖杯 🏆 视觉装饰系统
========================================== */

/* 1. 调整名次文本容器的布局，容纳奖杯并左对齐 */
.leader-rank {
  display: flex !important; /* 强制启用 flex */
  align-items: center;
  justify-content: flex-start !important; /* 修改为靠左对齐 */
  width: calc(75px * var(--ui-scale)); /* 稍微加宽一点名次列，给奖杯留空间 */
  gap: calc(2px * var(--ui-scale)); /* 奖杯与#数字的间距 */
  text-align: left !important;
  /* 保留您文件里原有的字体大小、颜色等定义 */
}

/* 2. 定义前三名名次前的奖杯样式 (通用的发光与旋转) */
.leader-rank[data-rank="1"]::before,
.leader-rank[data-rank="2"]::before,
.leader-rank[data-rank="3"]::before {
  content: '🏆';
  display: inline-block;
  font-size: 1.1em; /* 奖杯比文字略大 */
  font-family: sans-serif; /* 确保在所有设备上正确渲染 emoji */
  /* 让奖杯也有动态浮动的赛博感 */
  animation: trophy-wobble 2.5s infinite ease-in-out alternate;
  will-change: transform;
}

/* 3. 分别定义前三名奖杯的专属 Neon Glow (光晕) 效果 */

/* 🥇 TOP 1 Gold Glow */
.leader-rank[data-rank="1"]::before {
  filter: drop-shadow(0 0 calc(8px * var(--ui-scale)) #ffd700);
}

/* 🥈 TOP 2 Platinum Glow */
.leader-rank[data-rank="2"]::before {
  filter: drop-shadow(0 0 calc(6px * var(--ui-scale)) #c0c0c0);
}

/* 🥉 TOP 3 Copper Glow */
.leader-rank[data-rank="3"]::before {
  filter: drop-shadow(0 0 calc(5px * var(--ui-scale)) #ff8c00);
}

/* 4. 定义奖杯微幅浮动动画，增加灵动感 */
@keyframes trophy-wobble {
  0% { transform: scale(1) rotate(-5deg); }
  50% { transform: scale(1.05) rotate(5deg); }
  100% { transform: scale(1) rotate(-5deg); }
}
/* 🥇 TOP 1: 纯金发光徽章特效 */
.leader-item[data-rank="1"] {
  border-left-color: #ffd700;
  background: linear-gradient(90deg, rgba(255, 215, 0, 0.15) 0%, rgba(2, 6, 23, 0.9) 100%);
}
.leader-item[data-rank="1"]:hover {
  border-color: #ffd700;
  box-shadow: 0 0 20px rgba(255, 215, 0, 0.5), inset 0 0 10px rgba(255, 215, 0, 0.2);
}
.leader-rank[data-rank="1"] {
  color: #ffd700;
  text-shadow: 0 0 12px #ffd700, 0 0 25px rgba(255, 215, 0, 0.8);
  font-size: calc(20px * var(--ui-scale) * var(--font-user, 1));
}

/* 🥈 TOP 2: 铂金/银色发光徽章特效 */
.leader-item[data-rank="2"] {
  border-left-color: #c0c0c0;
  background: linear-gradient(90deg, rgba(192, 192, 192, 0.1) 0%, rgba(2, 6, 23, 0.9) 100%);
}
.leader-item[data-rank="2"]:hover {
  border-color: #c0c0c0;
  box-shadow: 0 0 15px rgba(192, 192, 192, 0.5), inset 0 0 10px rgba(192, 192, 192, 0.2);
}
.leader-rank[data-rank="2"] {
  color: #e2e8f0;
  text-shadow: 0 0 10px #c0c0c0, 0 0 15px rgba(192, 192, 192, 0.6);
  font-size: calc(18px * var(--ui-scale) * var(--font-user, 1));
}

/* 🥉 TOP 3: 亮橙/铜色发光徽章特效 */
.leader-item[data-rank="3"] {
  border-left-color: #ff8c00;
  background: linear-gradient(90deg, rgba(255, 140, 0, 0.1) 0%, rgba(2, 6, 23, 0.9) 100%);
}
.leader-item[data-rank="3"]:hover {
  border-color: #ff8c00;
  box-shadow: 0 0 15px rgba(255, 140, 0, 0.5), inset 0 0 10px rgba(255, 140, 0, 0.2);
}
.leader-rank[data-rank="3"] {
  color: #ffb84d;
  text-shadow: 0 0 10px #ff8c00, 0 0 20px rgba(255, 140, 0, 0.6);
  font-size: calc(17px * var(--ui-scale) * var(--font-user, 1));
}
.leader-left {
  display: flex;
  align-items: center;
  gap: calc(12px * var(--ui-scale));
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
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.map-section-inner {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: row;
  align-items: stretch;
  position: relative;
}

/* 与地图同一外框：侧栏 + 画布共用压暗与描边 */
.map-frame {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  position: relative;
  isolation: isolate;
}

.map-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: row;
  align-items: stretch;
}

.map-team-dock {
  flex-shrink: 0;
  z-index: 8;
  display: flex;
  flex-direction: column;
  min-height: 0;
  box-sizing: border-box;
  padding: 6px 4px 6px 8px;
  overflow: hidden;
  background: rgba(3, 16, 42, 0.42);
  border-right: 1px solid rgba(120, 240, 255, 0.12);
}

.map-team-dock-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: calc(10px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(148, 163, 184, 0.55);
  padding: 4px 2px;
}

.map-team-dock-rows {
  flex: 1;
  min-height: 0;
  width: 100%;
}

.map-team-dock-row {
  display: flex;
  align-items: center;
  min-height: 0;
  padding: 2px 2px 2px 6px;
  border-left: 2px solid var(--team-line, rgba(120, 240, 255, 0.22));
  font-size: calc(11px * var(--ui-scale) * var(--font-user, 1));
  color: rgba(226, 232, 240, 0.88);
}

.dock-team-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.map-chart-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  position: relative;
}

/* 整块地图框（含左侧队伍条）边缘压暗 + 青边光 */
.map-frame::after {
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

.map-frame.map-impact {
  animation: mapImpactShake 560ms cubic-bezier(0.22, 0.61, 0.36, 1);
}

.map-frame.map-impact::before {
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

/* ==========================================
   💻 实时战况日志：黑客 CRT 终端扫描特效
========================================== */

/* 1. 终端容器：极暗底色 + 内部发光 + 绿色扫描线遮罩 */
.terminal-logs {
  height: calc(100% - 54px);
  overflow-y: auto;
  /* 极暗底色配合内发光，制造屏幕景深 */
  background: rgba(1, 8, 16, 0.9) !important;
  border: 1px solid rgba(0, 255, 170, 0.2) !important;
  box-shadow: inset 0 0 30px rgba(0, 255, 170, 0.05), 0 0 15px rgba(0, 255, 170, 0.1) !important;
  padding: 12px !important;
  /* 专属的终端横向扫描线背景叠加 */
  background-image: repeating-linear-gradient(
    0deg,
    rgba(0, 255, 170, 0.04),
    rgba(0, 255, 170, 0.04) 1px,
    transparent 1px,
    transparent 3px
  ) !important;
}

/* 2. 终端单行文本：投屏加大字号，略减弱闪烁避免眩晕 */
.log-line {
  font-family: 'Courier New', Courier, 'Roboto Mono', monospace !important;
  font-weight: 600;
  font-size: calc(11px * var(--ui-scale) * var(--font-user, 1));
  letter-spacing: 0.4px;
  line-height: 1.55;
  margin-bottom: 8px;
  animation: text-flicker 6s infinite normal ease-in-out;
}

/* 3. CRT 文字闪烁关键帧 */
@keyframes text-flicker {
  0%   { opacity: 0.95; }
  48%  { opacity: 0.95; }
  50%  { opacity: 0.6;  } /* 偶尔闪亮或暗淡一次 */
  52%  { opacity: 1;    }
  100% { opacity: 0.95; }
}

/* 4. 重新定义高对比度日志颜色与强烈文字外发光 */
.log-system {
  color: #00ffaa !important; /* 骇客电子绿 */
  text-shadow: 0 0 8px rgba(0, 255, 170, 0.6) !important;
}

.log-judge {
  color: #fffc00 !important; /* 裁判干预：霓虹黄 */
  text-shadow: 0 0 8px rgba(255, 252, 0, 0.6) !important;
}

.log-event {
  color: #ff0055 !important; /* 攻击事件：高亮警示红（原为平淡青色） */
  text-shadow: 0 0 8px rgba(255, 0, 85, 0.6) !important;
}

.log-normal {
  color: #00f3ff !important; /* 目标与普通文本：高亮青色 */
  text-shadow: 0 0 6px rgba(0, 243, 255, 0.4) !important;
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
  font-size: calc(13px * var(--ui-scale) * var(--font-user, 1));
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

/* ==========================================
   🚀 视觉大改版：赛博朋克与指挥舱特效追加
========================================== */

/* 1. 全局背景强化：加入暗色网格与深蓝环境光 */
.screen-root {
  background-color: #020617; /* 极暗的蓝灰底色 */
  background-image: 
    radial-gradient(circle at 50% 50%, rgba(14, 38, 74, 0.4) 0%, rgba(2, 6, 23, 1) 80%),
    linear-gradient(rgba(0, 243, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(0, 243, 255, 0.03) 1px, transparent 1px);
  background-size: 100% 100%, 30px 30px, 30px 30px;
  background-position: center center;
}

/* 2. 左右数据面板：切角边框与内发光 (替代原本的 cyber-panel) */
.cyber-panel {
  background: rgba(3, 11, 26, 0.65);
  border: 1px solid rgba(0, 243, 255, 0.15);
  box-shadow: inset 0 0 20px rgba(0, 243, 255, 0.05), 0 4px 12px rgba(0, 0, 0, 0.5);
  position: relative;
  /* 左上和右下切角设计 */
  clip-path: polygon(
    15px 0, 100% 0, 
    100% calc(100% - 15px), calc(100% - 15px) 100%, 
    0 100%, 0 15px
  );
  backdrop-filter: blur(4px);
}
/* 给面板四个角增加高亮瞄准星修饰 */
.cyber-panel::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  border: 2px solid transparent;
  background: linear-gradient(135deg, #00f3ff 10px, transparent 10px) top left,
              linear-gradient(-135deg, #00f3ff 10px, transparent 10px) top right,
              linear-gradient(45deg, #00f3ff 10px, transparent 10px) bottom left,
              linear-gradient(-45deg, #00f3ff 10px, transparent 10px) bottom right;
  background-size: 50% 50%;
  background-repeat: no-repeat;
  opacity: 0.6;
}

/* 3. 顶部标题霓虹发光优化 */
.topbar-title.glow-text {
  text-shadow: 0 0 10px rgba(0, 243, 255, 0.6), 0 0 20px rgba(0, 243, 255, 0.4), 0 0 30px rgba(0, 243, 255, 0.2);
  background: linear-gradient(to bottom, #ffffff, #82e9ff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* 4. 右上角战术时钟专属样式 */
.topbar-right {
  gap: 15px;
}
.topbar-countdown {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(255, 42, 42, 0.1);
  border: 1px solid rgba(255, 42, 42, 0.3);
  padding: 6px 16px;
  clip-path: polygon(12px 0, 100% 0, 100% calc(100% - 12px), calc(100% - 12px) 100%, 0 100%, 0 12px);
  box-shadow: inset 0 0 15px rgba(255, 42, 42, 0.15);
}
.countdown-icon {
  width: 10px;
  height: 10px;
  background-color: #ff2a2a;
  border-radius: 50%;
  box-shadow: 0 0 10px #ff2a2a;
  animation: pulse-red 1.2s infinite;
}
.countdown-content {
  display: flex;
  flex-direction: column;
}
.countdown-label {
  font-size: calc(10px * var(--ui-scale));
  color: #ff8b8b;
  letter-spacing: 1.5px;
}
.countdown-value {
  font-size: calc(18px * var(--ui-scale));
  color: #ff2a2a;
  text-shadow: 0 0 8px rgba(255, 42, 42, 0.8);
}
.divider-line {
  width: 2px;
  height: 35px;
  background: linear-gradient(to bottom, transparent, rgba(0, 243, 255, 0.8), transparent);
  transform: skewX(-15deg);
  margin: 0 10px;
}
.signal-bars {
  display: flex;
  gap: 4px;
  align-items: flex-end;
  height: 28px;
  margin-left: 5px;
}
.signal-bars .topbar-bar,
.signal-bars .topbar-bar-mini {
  background-color: #00f3ff;
  box-shadow: 0 0 8px #00f3ff;
  width: 4px;
}
.signal-bars .topbar-bar {
  animation: scan 1.8s infinite ease-in-out alternate;
}
.signal-bars .topbar-bar-mini {
  animation: scan 1.2s infinite ease-in-out alternate;
}
.signal-bars .topbar-bar-mini.delayed {
  animation-delay: 0.6s;
}

/* 5. 关键帧动画定义 */
@keyframes pulse-red {
  0% { transform: scale(0.85); opacity: 0.5; box-shadow: 0 0 0 0 rgba(255, 42, 42, 0.6); }
  70% { transform: scale(1.15); opacity: 1; box-shadow: 0 0 0 6px rgba(255, 42, 42, 0); }
  100% { transform: scale(0.85); opacity: 0.5; box-shadow: 0 0 0 0 rgba(255, 42, 42, 0); }
}
@keyframes scan {
  0% { height: 20%; opacity: 0.3; }
  100% { height: 100%; opacity: 1; }
}

/* 6. 覆盖原有的字体定义，强化电竞/科技感 */
.font-cyber {
  font-family: 'Rajdhani', 'Orbitron', 'Roboto Mono', monospace;
}

</style>

