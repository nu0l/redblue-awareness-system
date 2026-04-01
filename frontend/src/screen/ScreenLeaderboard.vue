<template>
  <div class="leaderboard-wrap">
    <div v-if="topThree.length" class="leader-podium">
      <div
        v-for="(team, idx) in topThree"
        :key="`podium-${team.id}`"
        class="podium-row"
        :class="[`place-${idx + 1}`, team.type === 'red' ? 'side-red' : 'side-blue']"
        :data-rank="rankOf(team.id)"
      >
        <div class="podium-medal" aria-hidden="true">
          <span class="trophy-emoji">{{ podiumEmoji[idx] }}</span>
        </div>
        <div class="podium-main">
          <div class="podium-rank-line">
            <span class="podium-rank-label">TOP {{ idx + 1 }}</span>
            <span class="podium-name" :class="team.type === 'red' ? 'name-red' : 'name-blue'">
              {{ team.name }}
            </span>
          </div>
          <div
            class="podium-score font-cyber"
            :class="[
              team.type === 'red' ? 'score-red' : 'score-blue',
              scorePulse[team.id] ? 'score-pulse' : '',
            ]"
          >
            {{ team.score.toLocaleString() }}
            <span class="podium-pts">分</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="restTeams.length" ref="restViewportEl" class="leader-list leader-rest-list">
      <div class="leader-track" :class="restAnimating ? 'anim' : ''" :style="restTrackStyle">
        <div
          v-for="(team, idx) in restWindow"
          :key="`${team.id}-r-${idx}`"
          class="leader-item leader-rest-item"
          :data-rank="rankOf(team.id)"
        >
          <div class="leader-left">
            <span class="leader-rank font-cyber" :data-rank="rankOf(team.id)">#{{ rankOf(team.id) }}</span>
            <span class="leader-name" :class="team.type === 'red' ? 'name-red' : 'name-blue'">{{ team.name }}</span>
          </div>
          <div
            class="leader-score font-cyber"
            :class="[
              team.type === 'red' ? 'score-red' : 'score-blue',
              scorePulse[team.id] ? 'score-pulse' : '',
            ]"
          >
            {{ team.score.toLocaleString() }}
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="!topThree.length" class="leader-empty muted">暂无队伍数据</div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import type { TeamDTO } from "../shared/types";

const props = defineProps<{
  sortedTeams: TeamDTO[];
  scorePulse: Record<number, number>;
}>();

const podiumEmoji = ["🥇", "🥈", "🥉"];

const TOP_FIXED = 3;

const restViewportEl = ref<HTMLElement | null>(null);
const restStartIndex = ref(0);
const restShift = ref(0);
const restAnimating = ref(false);
const restItemStepPx = ref(56);
const restVisibleCount = ref(6);
const restShouldScroll = computed(() => {
  const n = restTeams.value.length;
  if (!n) return false;
  const visible = Math.min(Math.max(1, restVisibleCount.value), n);
  return n > visible;
});

const ranks = computed(() => {
  const m = new Map<number, number>();
  props.sortedTeams.forEach((t, i) => m.set(t.id, i + 1));
  return m;
});

function rankOf(teamId: number): number {
  return ranks.value.get(teamId) ?? 0;
}

const topThree = computed(() => props.sortedTeams.slice(0, TOP_FIXED));
const restTeams = computed(() => props.sortedTeams.slice(TOP_FIXED));

const restWindow = computed(() => {
  const arr = restTeams.value;
  const n = arr.length;
  if (!n) return [];
  const visible = Math.min(Math.max(1, restVisibleCount.value), n);
  if (n <= visible) return arr;
  const start = ((restStartIndex.value % n) + n) % n;
  const out: TeamDTO[] = [];
  for (let i = 0; i < visible + 1; i++) {
    out.push(arr[(start + i) % n]!);
  }
  return out;
});

const restTrackStyle = computed(() => ({
  transform: `translateY(${-restShift.value * restItemStepPx.value}px)`,
}));

let restTimer: number | undefined;
let restResizeObs: ResizeObserver | undefined;

function measureStep() {
  try {
    const viewport = restViewportEl.value;
    if (!viewport) return;
    const el = viewport.querySelector(".leader-rest-item") as HTMLElement | null;
    if (!el) return;
    const styles = window.getComputedStyle(el);
    const mb = Number.parseFloat(styles.marginBottom || "0") || 0;
    const step = el.getBoundingClientRect().height + mb;
    if (Number.isFinite(step) && step > 6) restItemStepPx.value = step;

    const vh = viewport.getBoundingClientRect().height;
    const avail = Math.max(1, vh - 2);
    const visible = Math.floor(avail / Math.max(1, restItemStepPx.value));
    restVisibleCount.value = Math.max(1, Math.min(12, visible || 1));

    // 根据最新可见行数，立即启停滚动（避免缩放后不滚）。
    if (restTeams.value.length > Math.min(Math.max(1, restVisibleCount.value), restTeams.value.length)) {
      startRestLoop();
    } else if (restTimer) {
      window.clearInterval(restTimer);
      restTimer = undefined;
    }
  } catch {
    // ignore
  }
}

function startRestLoop() {
  if (restTimer) window.clearInterval(restTimer);
  restTimer = window.setInterval(() => {
    const arr = restTeams.value;
    const n = arr.length;
    const visible = Math.min(Math.max(1, restVisibleCount.value), n);
    if (n <= visible) return;
    if (restAnimating.value) return;
    restAnimating.value = true;
    restShift.value = 1;
    window.setTimeout(() => {
      restStartIndex.value = (restStartIndex.value + 1) % n;
      restAnimating.value = false;
      restShift.value = 0;
    }, 520);
  }, 2600);
}

watch(
  () => restTeams.value.length,
  () => {
    restStartIndex.value = 0;
    restShift.value = 0;
    restAnimating.value = false;
    void nextTick(() => measureStep());
    if (restShouldScroll.value) startRestLoop();
    else if (restTimer) window.clearInterval(restTimer);
  }
);

watch(
  () => props.sortedTeams.map((t) => t.id).join(","),
  () => {
    // 仅在“队伍排序/数量”变化时重置滚动；分数刷新不要打断滚动动画。
    restStartIndex.value = 0;
    restShift.value = 0;
    void nextTick(() => measureStep());
  }
);

onMounted(() => {
  void nextTick(() => measureStep());
  if (restShouldScroll.value) startRestLoop();
  try {
    if ("ResizeObserver" in window) {
      restResizeObs = new ResizeObserver(() => {
        measureStep();
        if (restShouldScroll.value) startRestLoop();
        else if (restTimer) window.clearInterval(restTimer);
      });
      if (restViewportEl.value) restResizeObs.observe(restViewportEl.value);
    }
  } catch {
    // ignore
  }
});

onBeforeUnmount(() => {
  if (restTimer) window.clearInterval(restTimer);
  try {
    restResizeObs?.disconnect();
  } catch {
    // ignore
  }
});
</script>

<style scoped>
.leaderboard-wrap {
  display: flex;
  flex-direction: column;
  gap: calc(10px * var(--ui-scale, 1));
  min-height: 0;
  flex: 1;
  font-size: calc(13px * var(--font-user, 1) * var(--ui-scale, 1));
}

.leader-podium {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: calc(8px * var(--ui-scale, 1));
}

.podium-row {
  display: flex;
  align-items: stretch;
  gap: 8px;
  padding: calc(7px * var(--ui-scale, 1)) calc(9px * var(--ui-scale, 1));
  border-radius: 11px;
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.35);
}

.podium-row::after {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  opacity: 0.5;
  background: linear-gradient(120deg, transparent 30%, rgba(255, 255, 255, 0.06) 50%, transparent 70%);
}

.place-1 {
  border-width: 2px;
  border-color: rgba(251, 191, 36, 0.75);
  background: linear-gradient(135deg, rgba(120, 53, 15, 0.55) 0%, rgba(15, 23, 42, 0.92) 45%, rgba(15, 23, 42, 0.95) 100%);
  box-shadow: 0 0 36px rgba(251, 191, 36, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.12);
  transform: scale(1.005);
}
.place-2 {
  border-color: rgba(148, 163, 184, 0.55);
  background: linear-gradient(135deg, rgba(51, 65, 85, 0.45) 0%, rgba(15, 23, 42, 0.92) 100%);
}
.place-3 {
  border-color: rgba(180, 83, 9, 0.55);
  background: linear-gradient(135deg, rgba(120, 53, 15, 0.35) 0%, rgba(15, 23, 42, 0.92) 100%);
}

.podium-medal {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  width: calc(34px * var(--ui-scale, 1));
  min-width: 30px;
}

.trophy-emoji {
  font-size: calc(1.28em * var(--ui-scale, 1));
  line-height: 1;
  filter: drop-shadow(0 0 12px rgba(255, 215, 0, 0.45));
}

.place-2 .trophy-emoji {
  filter: drop-shadow(0 0 10px rgba(203, 213, 225, 0.4));
}
.place-3 .trophy-emoji {
  filter: drop-shadow(0 0 10px rgba(253, 186, 116, 0.35));
}

.podium-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
}

.podium-rank-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

.podium-rank-label {
  font-size: 0.66em;
  font-weight: 800;
  letter-spacing: 0.12em;
  color: rgba(148, 163, 184, 0.95);
}

.place-1 .podium-rank-label {
  color: #fde68a;
  text-shadow: 0 0 12px rgba(251, 191, 36, 0.5);
}

.podium-name {
  font-weight: 800;
  font-size: 0.86em;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.podium-score {
  font-size: calc(0.98em * var(--ui-scale, 1));
  font-weight: 800;
  letter-spacing: 0.06em;
}

.podium-pts {
  font-size: 0.55em;
  font-weight: 600;
  margin-left: 4px;
  opacity: 0.75;
}

.leader-rest-list {
  flex: 1 1 auto;
  min-height: 0;
}

.leader-empty {
  padding: 16px;
  text-align: center;
  font-size: 0.95em;
}
.muted {
  color: rgba(148, 163, 184, 0.85);
}

.leader-list {
  overflow: hidden;
  padding-right: 4px;
  min-height: 0;
}
.leader-track {
  will-change: transform;
}
.leader-track.anim {
  transition: transform 0.52s ease;
}

.leader-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: calc(9px * var(--ui-scale, 1)) calc(11px * var(--ui-scale, 1));
  margin-bottom: calc(7px * var(--ui-scale, 1));
  border-radius: 8px;
  border: 1px solid rgba(56, 189, 248, 0.2);
  background: linear-gradient(90deg, rgba(15, 23, 42, 0.72), rgba(8, 12, 28, 0.5));
}
.leader-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.leader-rank {
  flex: 0 0 auto;
  font-size: 0.88em;
  color: rgba(0, 243, 255, 0.85);
  letter-spacing: 0.06em;
}
.leader-name {
  font-weight: 650;
  font-size: 0.95em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.name-red {
  color: #ff8a8a;
}
.name-blue {
  color: #7dd3fc;
}
.leader-score {
  flex: 0 0 auto;
  font-size: 1em;
  letter-spacing: 0.04em;
}
.score-red {
  color: #ff6b6b;
  text-shadow: 0 0 10px rgba(255, 107, 107, 0.35);
}
.score-blue {
  color: #5ecbff;
  text-shadow: 0 0 10px rgba(94, 203, 255, 0.35);
}
.score-pulse {
  animation: scorePulseAnim 0.85s ease;
}
@keyframes scorePulseAnim {
  0% {
    transform: scale(1);
  }
  40% {
    transform: scale(1.06);
  }
  100% {
    transform: scale(1);
  }
}
.font-cyber {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}
</style>
