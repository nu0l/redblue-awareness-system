<template>
  <div class="html-panel-root">
    <!-- 攻防态势指标 -->
    <div v-if="variant === 'attack_metric_cards'" class="metric-grid">
      <div class="metric-card">
        <div class="metric-label">战术命中累计</div>
        <div class="metric-num">{{ totalAttackHits }}</div>
        <div class="metric-hint">次</div>
      </div>
      <div class="metric-card accent-purple">
        <div class="metric-label">受攻击地域覆盖</div>
        <div class="metric-num">{{ regionCount }}</div>
        <div class="metric-hint">个节点</div>
      </div>
      <div class="metric-card accent-amber">
        <div class="metric-label">参战队伍</div>
        <div class="metric-num">{{ teamCount }}</div>
        <div class="metric-hint">支</div>
      </div>
    </div>

    <!-- 红蓝阵营总分 -->
    <div v-else-if="variant === 'team_camp_totals'" class="camp-split">
      <div class="camp-side camp-red">
        <div class="camp-icon" aria-hidden="true">◆</div>
        <div class="camp-title">红方阵营总分</div>
        <div class="camp-score">{{ redTotalScore.toLocaleString() }}</div>
      </div>
      <div class="camp-vs">VS</div>
      <div class="camp-side camp-blue">
        <div class="camp-icon" aria-hidden="true">◇</div>
        <div class="camp-title">蓝方阵营总分</div>
        <div class="camp-score">{{ blueTotalScore.toLocaleString() }}</div>
      </div>
    </div>

    <!-- 战术类型 TOP -->
    <div v-else-if="variant === 'top_attack_types_list'" class="attack-list-wrap">
      <div v-if="!topTypes.length" class="empty-hint">暂无战术统计数据</div>
      <ul v-else class="attack-rank-list">
        <li v-for="(row, idx) in topTypes" :key="row.name" class="attack-rank-row" :class="`rk-${idx + 1}`">
          <span class="rk-badge">{{ idx + 1 }}</span>
          <span class="rk-name">{{ row.name }}</span>
          <span class="rk-val">{{ row.value }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { AttackStatDTO, TeamDTO } from "../shared/types";
import type { ScreenModuleId } from "../shared/screenLayout";

const props = defineProps<{
  variant: ScreenModuleId;
  attackStats: AttackStatDTO[];
  regionAttackStats: AttackStatDTO[];
  teams: TeamDTO[];
}>();

const totalAttackHits = computed(() =>
  (props.attackStats ?? []).reduce((s, x) => s + Number(x.value ?? 0), 0)
);

const regionCount = computed(() => (props.regionAttackStats ?? []).filter((x) => Number(x.value ?? 0) > 0).length);

const teamCount = computed(() => (props.teams ?? []).length);

const redTotalScore = computed(() =>
  (props.teams ?? []).filter((t) => t.type === "red").reduce((s, t) => s + Number(t.score ?? 0), 0)
);

const blueTotalScore = computed(() =>
  (props.teams ?? []).filter((t) => t.type === "blue").reduce((s, t) => s + Number(t.score ?? 0), 0)
);

const topTypes = computed(() => {
  const list = [...(props.attackStats ?? [])].sort((a, b) => b.value - a.value).slice(0, 8);
  return list;
});
</script>

<style scoped>
.html-panel-root {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  font-size: calc(15px * var(--font-user, 1) * var(--ui-scale, 1));
}

.metric-grid {
  display: flex;
  flex-direction: column;
  gap: calc(10px * var(--ui-scale, 1));
  height: 100%;
  justify-content: center;
}

.metric-card {
  position: relative;
  padding: calc(12px * var(--ui-scale, 1)) calc(14px * var(--ui-scale, 1));
  border-radius: 12px;
  border: 1px solid rgba(56, 189, 248, 0.35);
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.95) 0%, rgba(8, 15, 35, 0.88) 100%);
  box-shadow: 0 0 24px rgba(0, 243, 255, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  overflow: hidden;
}

.metric-card::before {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(0, 243, 255, 0.06), transparent 55%);
  pointer-events: none;
}

.metric-card.accent-purple {
  border-color: rgba(168, 85, 247, 0.45);
}
.metric-card.accent-amber {
  border-color: rgba(251, 191, 36, 0.4);
}

.metric-label {
  font-size: 0.85em;
  color: rgba(148, 163, 184, 0.95);
  letter-spacing: 0.04em;
  margin-bottom: 6px;
}

.metric-num {
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  font-size: calc(1.85em * var(--ui-scale, 1));
  font-weight: 800;
  color: #e2e8f0;
  text-shadow: 0 0 20px rgba(0, 243, 255, 0.25);
  line-height: 1.15;
}

.metric-hint {
  font-size: 0.75em;
  color: rgba(148, 163, 184, 0.85);
  margin-top: 4px;
}

.camp-split {
  display: flex;
  align-items: stretch;
  gap: calc(10px * var(--ui-scale, 1));
  height: 100%;
  min-height: 160px;
}

.camp-side {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 12px;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.camp-red {
  background: linear-gradient(160deg, rgba(127, 29, 29, 0.45) 0%, rgba(15, 23, 42, 0.9) 100%);
  box-shadow: 0 0 28px rgba(248, 113, 113, 0.15);
}

.camp-blue {
  background: linear-gradient(160deg, rgba(30, 58, 138, 0.5) 0%, rgba(15, 23, 42, 0.9) 100%);
  box-shadow: 0 0 28px rgba(56, 189, 248, 0.12);
}

.camp-icon {
  font-size: 1.2em;
  opacity: 0.85;
  margin-bottom: 6px;
}

.camp-red .camp-icon {
  color: #fca5a5;
}
.camp-blue .camp-icon {
  color: #7dd3fc;
}

.camp-title {
  font-size: 0.82em;
  color: rgba(226, 232, 240, 0.88);
  margin-bottom: 8px;
  text-align: center;
}

.camp-score {
  font-family: ui-monospace, Menlo, monospace;
  font-size: calc(1.65em * var(--ui-scale, 1));
  font-weight: 800;
  letter-spacing: 0.02em;
}

.camp-red .camp-score {
  color: #fecaca;
  text-shadow: 0 0 18px rgba(248, 113, 113, 0.45);
}

.camp-blue .camp-score {
  color: #bae6fd;
  text-shadow: 0 0 18px rgba(56, 189, 248, 0.4);
}

.camp-vs {
  align-self: center;
  font-weight: 800;
  font-size: 0.95em;
  color: rgba(148, 163, 184, 0.9);
  letter-spacing: 0.15em;
}

.attack-list-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 2px;
}

.empty-hint {
  text-align: center;
  color: rgba(148, 163, 184, 0.9);
  padding: 24px 8px;
  font-size: 0.95em;
}

.attack-rank-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.attack-rank-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: calc(10px * var(--ui-scale, 1)) 12px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(56, 189, 248, 0.18);
}

.attack-rank-row.rk-1 {
  border-color: rgba(251, 191, 36, 0.55);
  background: linear-gradient(90deg, rgba(251, 191, 36, 0.12), rgba(15, 23, 42, 0.7));
}
.attack-rank-row.rk-2 {
  border-color: rgba(148, 163, 184, 0.45);
}
.attack-rank-row.rk-3 {
  border-color: rgba(180, 83, 9, 0.45);
}

.rk-badge {
  flex: 0 0 auto;
  width: 1.75em;
  height: 1.75em;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  font-weight: 800;
  font-size: 0.95em;
  background: rgba(0, 243, 255, 0.15);
  color: #22d3ee;
}

.rk-1 .rk-badge {
  background: rgba(251, 191, 36, 0.25);
  color: #fde68a;
}

.rk-name {
  flex: 1;
  min-width: 0;
  font-size: 0.95em;
  font-weight: 600;
  color: #e2e8f0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rk-val {
  flex: 0 0 auto;
  font-family: ui-monospace, Menlo, monospace;
  font-size: 1em;
  font-weight: 700;
  color: #94a3b8;
}
</style>
