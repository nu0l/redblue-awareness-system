<template>
  <div class="admin-root">
    <div v-if="!isAuthed" class="admin-login-overlay">
      <div class="login-orb orb-a" aria-hidden="true"></div>
      <div class="login-orb orb-b" aria-hidden="true"></div>
      <div class="admin-login-shell">
        <section class="admin-login-brand">
          <div class="admin-login-brand-kicker">CareerCompass · Inspired Layout</div>
          <h2 class="admin-login-brand-title">RedBlue Command Center</h2>
          <p class="admin-login-brand-desc">
            统一管理场次、战队、回放与可视化联动，让登录页和后台都具备一致的产品化体验。
          </p>
          <div class="admin-login-brand-tags">
            <span class="login-tag">WebSocket 实时联动</span>
            <span class="login-tag">攻防演练指挥</span>
            <span class="login-tag">事件可追溯回放</span>
            <span class="login-tag">Glassmorphism UI</span>
          </div>
          <ul class="login-feature-list">
            <li>细粒度模块导航：对抗演练 / 大屏视觉 / 广播回放 / 数据维护</li>
            <li>统一配色与卡片系统：登录、顶部栏、侧边导航视觉一致</li>
            <li>重点操作突出：创建场次、推送指令、全库重置风险隔离</li>
          </ul>
          <div class="login-slogan-rotator">
            <span class="rotator-label">NOW SHOWING</span>
            <span class="rotator-text">{{ activeSlogan }}</span>
          </div>
        </section>

        <section class="admin-login-card" :class="{ 'login-error': loginErrorPulse }">
          <div class="admin-login-card-head">
            <div class="admin-login-card-title">管理员登录</div>
            <div class="admin-login-card-sub">请使用后台账号继续访问 Command Center</div>
          </div>
          <div class="admin-login-mascot" :class="{ peeking: loginPeek }" aria-hidden="true">
            <div class="mascot-head">
              <div class="mascot-eye eye-left"></div>
              <div class="mascot-eye eye-right"></div>
              <div class="mascot-mouth"></div>
            </div>
            <div class="mascot-hand hand-left"></div>
            <div class="mascot-hand hand-right"></div>
          </div>
          <el-form :model="loginForm" label-position="top" @submit.prevent>
            <el-form-item label="用户名" class="neon-input">
              <el-input v-model="loginForm.username" placeholder="ADMIN_USERNAME" @keyup.enter="doLogin" />
            </el-form-item>
            <el-form-item label="密码" class="neon-input">
              <el-input
                v-model="loginForm.password"
                type="password"
                show-password
                placeholder="ADMIN_PASSWORD"
                @focus="loginPeek = true"
                @blur="loginPeek = false"
                @keyup.enter="doLogin"
              />
            </el-form-item>
          </el-form>
          <div class="admin-login-actions">
            <el-button type="primary" class="admin-login-submit" :loading="loginSubmitting" @click="doLogin">Continue</el-button>
          </div>
        </section>
      </div>
      <div class="admin-login-hint">
      </div>
    </div>

    <header class="admin-header">
      <div class="admin-header-left">
        <div class="admin-logo"></div>
        <div>
          <div class="admin-title font-cyber">红蓝对抗指挥中心 · Admin Workspace</div>
          <div class="admin-subtitle">Inspired by CareerCompass login/dashboard visual language</div>
        </div>
      </div>

      <div class="admin-header-right">
        <el-select
          v-model="matchId"
          filterable
          placeholder="选择场次"
          style="width: 260px"
          @change="onMatchChange"
        >
          <el-option v-for="m in matches" :key="m.id" :label="m.id" :value="m.id" />
        </el-select>
        <el-button type="primary" @click="createMatch">创建新场次</el-button>
        <el-select v-model="selectedTemplateId" clearable placeholder="套用模板创建" style="width: 180px">
          <el-option v-for="t in templates" :key="t.id" :label="t.name" :value="t.id" />
        </el-select>

        <el-button
          :disabled="!matchId"
          plain
          @click="copyMatchId"
        >复制 match_id</el-button>
        <el-button
          :disabled="!matchId"
          @click="openScreen"
        >跳转大屏</el-button>
        <el-button
          :disabled="!matchId"
          plain
          @click="openLeaderboard"
        >得分总榜</el-button>
        <el-button type="danger" plain @click="doLogout">退出登录</el-button>
      </div>
    </header>

    <div class="admin-body">
      <div class="admin-shell">
        <aside class="admin-sidebar" aria-label="功能分类">
          <div class="sidebar-head">
            <div class="sidebar-head-title">功能导航</div>
            <div class="sidebar-head-sub">按模块切换，避免单页堆叠</div>
          </div>

          <button
            type="button"
            class="nav-card"
            :class="{ active: activeSection === 'combat' }"
            @click="activeSection = 'combat'"
          >
            <span class="nav-card-ico" aria-hidden="true">CB</span>
            <div class="nav-card-body">
              <div class="nav-card-title">对抗演练</div>
              <div class="nav-card-desc">攻击飞线、裁判加扣分</div>
            </div>
          </button>

          <button
            type="button"
            class="nav-card"
            :class="{ active: activeSection === 'screen' }"
            @click="activeSection = 'screen'"
          >
            <span class="nav-card-ico" aria-hidden="true">VS</span>
            <div class="nav-card-body">
              <div class="nav-card-title">大屏与视觉</div>
              <div class="nav-card-desc">底图、标题、总榜与音频</div>
            </div>
          </button>

          <button
            type="button"
            class="nav-card"
            :class="{ active: activeSection === 'broadcast' }"
            @click="activeSection = 'broadcast'"
          >
            <span class="nav-card-ico" aria-hidden="true">BC</span>
            <div class="nav-card-body">
              <div class="nav-card-title">广播与回放</div>
              <div class="nav-card-desc">全局通知、大屏复盘</div>
            </div>
          </button>

          <button
            type="button"
            class="nav-card nav-card-warn"
            :class="{ active: activeSection === 'maintenance' }"
            @click="activeSection = 'maintenance'"
          >
            <span class="nav-card-ico" aria-hidden="true">MT</span>
            <div class="nav-card-body">
              <div class="nav-card-title">数据维护</div>
              <div class="nav-card-desc">重置场次或全库</div>
            </div>
          </button>

          <div class="sidebar-divider" role="presentation"></div>

          <button
            type="button"
            class="nav-card"
            :class="{ active: activeSection === 'teams' }"
            @click="activeSection = 'teams'"
          >
            <span class="nav-card-ico" aria-hidden="true">TM</span>
            <div class="nav-card-body">
              <div class="nav-card-title">队伍与复盘</div>
              <div class="nav-card-desc">参演队伍、事件流</div>
            </div>
          </button>

          <button
            type="button"
            class="nav-card"
            :class="{ active: activeSection === 'ops' }"
            @click="activeSection = 'ops'"
          >
            <span class="nav-card-ico" aria-hidden="true">OP</span>
            <div class="nav-card-body">
              <div class="nav-card-title">运营与分析</div>
              <div class="nav-card-desc">模板、工单、KPI与审计</div>
            </div>
          </button>
        </aside>

        <main class="admin-main">
          <!-- 对抗演练 -->
          <div v-show="activeSection === 'combat'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">对抗演练</h2>
              <p class="section-sub">触发大屏飞线动画，并按规则自动计分；裁判可手动调整比分。</p>
            </div>
          <div class="grid">
            <el-card shadow="hover" class="card">
              <template #header>
                <div class="card-title">触发攻击飞线与自动计分</div>
              </template>
              <el-form :model="attackForm" label-width="100px">
                <el-form-item label="攻击战队">
                  <el-select v-model="attackForm.team_id" style="width: 100%" @change="genAttackMessage">
                    <el-option
                      v-for="t in teams"
                      :key="t.id"
                      :label="t.name"
                      :value="t.id"
                      :disabled="t.type !== 'red'"
                    />
                  </el-select>
                </el-form-item>

                <el-form-item label="攻击队员">
                  <el-select
                    v-model="attackForm.member"
                    style="width: 100%"
                    :disabled="currentAttackMembers.length === 0"
                  >
                    <el-option v-for="m in currentAttackMembers" :key="m" :label="m" :value="m" />
                  </el-select>
                </el-form-item>

                <el-form-item label="攻击状态">
                  <el-select v-model="attackForm.status" style="width: 100%">
                    <el-option label="攻击成功" value="success" />
                    <el-option v-if="hasBlueTeam" label="防守成功" value="defense_success" />
                    <el-option v-if="hasBlueTeam" label="溯源成功" value="trace_success" />
                  </el-select>
                </el-form-item>

                <el-form-item label="攻击来源">
                  <el-radio-group v-model="attackForm.source_mode" size="small">
                    <el-radio-button label="city">源节点</el-radio-button>
                    <el-radio-button label="team">攻击队伍</el-radio-button>
                  </el-radio-group>
                </el-form-item>

                <el-form-item v-if="attackForm.source_mode === 'city'" label="源节点">
                  <el-select v-model="attackForm.source_city" style="width: 100%">
                    <el-option v-for="c in currentCities" :key="c" :label="c" :value="c" />
                  </el-select>
                </el-form-item>

                <el-form-item v-else label="来源队伍">
                  <el-select v-model="attackForm.source_team_id" style="width: 100%">
                    <el-option v-for="t in redTeams" :key="t.id" :label="t.name" :value="t.id" />
                  </el-select>
                </el-form-item>

                <el-form-item label="目标靶向">
                  <el-select v-model="attackForm.target_city" style="width: 100%">
                    <el-option v-for="c in currentCities" :key="c" :label="c" :value="c" />
                  </el-select>
                </el-form-item>

                <el-form-item label="单位名称">
                  <el-input
                    v-model="attackForm.target_unit"
                    placeholder="选填：用于拼成 [区县-单位]"
                    clearable
                  />
                </el-form-item>

                <el-form-item label="战术手段">
                  <el-select v-model="attackForm.attack_type" style="width: 100%">
                    <el-option v-for="a in attackTypes" :key="a" :label="a" :value="a" />
                  </el-select>

                  <div class="attack-custom">
                    <div class="attack-custom-row">
                      <el-input v-model="customAttackTypeInput" size="small" placeholder="输入新的战术手段名称" />
                      <el-button size="small" type="primary" @click="addAttackType">添加</el-button>
                    </div>
                    <div class="attack-custom-tags">
                      <el-tag
                        v-for="t in attackTypes"
                        :key="t"
                        closable
                        :disable-transitions="false"
                        @close="removeAttackType(t)"
                        class="tag"
                      >
                        {{ t }}
                      </el-tag>
                    </div>
                  </div>
                </el-form-item>

                <el-form-item label="实战加分">
                  <el-input-number v-model="attackForm.score_change" :min="0" :step="100" style="width: 100%" />
                </el-form-item>

                <el-form-item label="大屏播报" class="mt8">
                  <el-input
                    v-model="attackForm.message"
                    type="textarea"
                    :rows="2"
                    :readonly="autoMessage"
                  />
                  <div class="auto-msg-row">
                    <span class="auto-msg-label">自动生成</span>
                    <el-switch v-model="autoMessage" size="small" />
                  </div>
                </el-form-item>

                <el-form-item class="mt8">
                  <el-button
                    type="danger"
                    style="width: 240px"
                    :loading="isSubmitting"
                    :disabled="!attackReady"
                    @click="submitCommand('attack_success', attackForm)"
                  >
                    发射飞线指令
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>

            <el-card shadow="hover" class="card">
              <template #header>
                <div class="card-title">裁判手动加扣分</div>
              </template>
              <div class="row">
                <div class="col">
                  <div class="label">队伍</div>
                  <el-select v-model="manualScore.team_id" style="width: 100%">
                    <el-option v-for="t in teams" :key="t.id" :label="t.name" :value="t.id" />
                  </el-select>
                </div>
                <div class="col">
                  <div class="label">分数变化</div>
                  <el-input-number v-model="manualScore.score_change" :step="50" style="width: 100%" />
                </div>
              </div>
              <div style="height: 12px"></div>
              <div class="label">原因说明</div>
              <el-input v-model="manualScore.reason" placeholder="例如：提交 Flag 01 / 违规扣分" />
              <div class="attack-custom-tags" style="margin-top: 8px">
                <el-tag class="tag" @click="manualScore.reason = '提交有效攻击证据'" style="cursor: pointer">提交有效攻击证据</el-tag>
                <el-tag class="tag" @click="manualScore.reason = '违规操作扣分'" style="cursor: pointer">违规操作扣分</el-tag>
                <el-tag class="tag" @click="manualScore.reason = '溯源命中加分'" style="cursor: pointer">溯源命中加分</el-tag>
              </div>
              <div style="height: 12px"></div>
              <el-button
                type="primary"
                :loading="isSubmitting"
                :disabled="!manualReady"
                @click="submitManualScore"
                style="width: 240px"
              >
                强制判定
              </el-button>
            </el-card>
          </div>
          </div>

          <div v-show="activeSection === 'screen'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">大屏与视觉</h2>
              <p class="section-sub">底图模式、标题、得分总榜与背景、主办信息、音频资源。</p>
            </div>
          <div class="grid grid-single">
            <el-card shadow="hover" class="card small">
              <template #header>
                <div class="card-title">视觉与悬念控制台</div>
              </template>

              <div class="row-between mb12">
                <div>大屏底图模式</div>
                <el-radio-group v-model="mapMode" @change="switchMap">
                  <el-radio-button label="china">全国态势</el-radio-button>
                  <el-radio-button label="taizhou">泰州市节点</el-radio-button>
                </el-radio-group>
              </div>

              <div class="row-between mb12">
                <div>大屏隐藏排行榜（制造悬念）</div>
                <el-switch
                  v-model="leaderboardVisible"
                  active-color="#00f3ff"
                  inactive-color="#4b5563"
                  @change="togglePanel"
                />
              </div>

              <div class="row-between mb12">
                <div>大屏标题</div>
                <el-input v-model="screenTitle" size="small" style="width: 280px" placeholder="实战化红蓝对抗演练指挥中心" />
              </div>

              <el-button
                type="primary"
                class="w-full"
                :disabled="!matchId"
                :loading="isSubmitting"
                @click="saveScreenTitle"
              >
                保存标题
              </el-button>

              <div style="height: 14px"></div>

              <div class="mb8" style="font-size: 13px; color: var(--el-text-color-secondary)">得分总榜全屏背景（本场次）</div>
              <div class="row" style="flex-wrap: wrap; gap: 8px; align-items: center">
                <el-upload
                  :http-request="uploadLeaderboardBgHttpRequest"
                  :show-file-list="false"
                  accept="image/png,image/jpeg,image/webp"
                  :disabled="!matchId"
                  :on-success="onLeaderboardBgUploadSuccess"
                  :on-error="onLeaderboardBgUploadError"
                >
                  <el-button type="primary" size="small" :disabled="!matchId">上传背景图</el-button>
                </el-upload>
                <el-button size="small" :disabled="!matchId || !leaderboardBgUrl" @click="clearLeaderboardBackground">
                  恢复默认
                </el-button>
              </div>
              <div class="row-between mb12" style="margin-top: 10px">
                <div>总榜主区透明度</div>
                <div style="display: flex; align-items: center; gap: 10px; width: 280px">
                  <el-slider v-model="leaderboardMainAlpha" :min="0" :max="1" :step="0.01" style="flex: 1" />
                  <span class="mono" style="width: 44px; text-align: right">{{ leaderboardMainAlpha.toFixed(2) }}</span>
                </div>
              </div>
              <el-button
                size="small"
                class="w-full"
                :disabled="!matchId"
                :loading="isSubmitting"
                @click="saveLeaderboardStyle"
              >
                保存总榜透明度
              </el-button>
              <div v-if="leaderboardBgPreview" style="margin-top: 10px">
                <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 6px">当前自定义图预览</div>
                <div
                  style="
                    max-height: 88px;
                    border-radius: 8px;
                    overflow: hidden;
                    border: 1px solid var(--el-border-color);
                  "
                >
                  <a :href="leaderboardBgPreview" target="_blank" rel="noopener noreferrer" title="点击查看原图">
                    <img
                      :src="leaderboardBgPreview"
                      alt="得分总榜背景预览"
                      style="width: 100%; height: 88px; object-fit: cover; display: block; cursor: zoom-in"
                    />
                  </a>
                </div>
                <div class="mono" style="margin-top: 6px; font-size: 11px; color: var(--el-text-color-secondary)">
                  引用地址：{{ leaderboardBgPreview }}
                </div>
              </div>

              <div style="height: 14px"></div>

              <div class="row-between mb12">
                <div>主办方</div>
                <el-input v-model="screenOrganizer" size="small" style="width: 280px" placeholder="例如：某某战区"></el-input>
              </div>

              <div class="row-between mb12">
                <div>支撑方</div>
                <el-input v-model="screenSupporter" size="small" style="width: 280px" placeholder="例如：某某单位"></el-input>
              </div>

              <el-button
                type="primary"
                class="w-full"
                :disabled="!matchId"
                :loading="isSubmitting"
                @click="saveScreenCredits"
              >
                保存主办/支撑
              </el-button>

              <div style="height: 14px"></div>

              <div class="mb8" style="font-size: 13px; color: var(--el-text-color-secondary)">音频控制（本场次）</div>
              <div class="row-between mb12">
                <div>背景音乐 URL</div>
                <el-input
                  v-model="bgmUrl"
                  size="small"
                  style="width: 280px"
                  placeholder="例如：https://xxx.com/bgm.mp3 或 /uploads/match/a.mp3"
                />
              </div>
              <div class="row" style="gap: 8px; margin-top: -6px; margin-bottom: 10px">
                <el-upload
                  :http-request="uploadBgmHttpRequest"
                  :show-file-list="false"
                  accept=".mp3,.wav,.ogg,.m4a,.aac,audio/*"
                  :disabled="!matchId"
                  :on-success="onBgmUploadSuccess"
                  :on-error="onAudioUploadError"
                >
                  <el-button size="small" :disabled="!matchId">上传背景音乐</el-button>
                </el-upload>
                <el-button size="small" :disabled="!bgmUrl.trim()" @click="testBgm">测试播放</el-button>
                <el-button size="small" :disabled="!audioPlaying.bgm" @click="stopBgm">停止</el-button>
              </div>
              <div class="row-between mb12">
                <div>背景音乐开关</div>
                <el-switch v-model="bgmEnabled" />
              </div>
              <div class="row-between mb12">
                <div>攻击成功音效 URL</div>
                <el-input
                  v-model="successSfxUrl"
                  size="small"
                  style="width: 280px"
                  placeholder="例如：https://xxx.com/sfx.mp3 或 /uploads/match/hit.mp3"
                />
              </div>
              <div class="row" style="gap: 8px; margin-top: -6px; margin-bottom: 10px">
                <el-upload
                  :http-request="uploadSuccessSfxHttpRequest"
                  :show-file-list="false"
                  accept=".mp3,.wav,.ogg,.m4a,.aac,audio/*"
                  :disabled="!matchId"
                  :on-success="onSuccessSfxUploadSuccess"
                  :on-error="onAudioUploadError"
                >
                  <el-button size="small" :disabled="!matchId">上传成功音效</el-button>
                </el-upload>
                <el-button size="small" :disabled="!successSfxUrl.trim()" @click="testSuccessSfx">测试播放</el-button>
              </div>
              <div class="row-between mb12">
                <div>攻击成功音效开关</div>
                <el-switch v-model="successSfxEnabled" />
              </div>
              <el-button
                type="primary"
                class="w-full"
                :disabled="!matchId"
                :loading="isSubmitting"
                @click="saveAudioConfig"
              >
                保存音频配置
              </el-button>
            </el-card>
          </div>
          </div>

          <div v-show="activeSection === 'broadcast'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">广播与回放</h2>
              <p class="section-sub">向大屏推送系统广播；从指定 Seq 回放现场事件。</p>
            </div>
          <div class="grid">
            <el-card shadow="hover" class="card small">
              <template #header>
                <div class="card-title">全局系统广播推送</div>
              </template>
              <el-input v-model="broadcastMsg" type="textarea" :rows="3" placeholder="演练进入最后1小时冲刺..." />
              <div class="attack-custom-tags" style="margin-top: 8px">
                <el-tag
                  v-for="tpl in broadcastTemplates"
                  :key="tpl"
                  class="tag"
                  @click="broadcastMsg = tpl"
                  style="cursor: pointer"
                >
                  {{ tpl }}
                </el-tag>
              </div>
              <div style="height: 12px"></div>
              <el-button type="warning" class="w-full" :loading="isSubmitting" @click="submitBroadcast">
                发送全局通知
              </el-button>
            </el-card>

            <el-card shadow="hover" class="card small">
              <template #header>
                <div class="card-title">回放控制（从 Seq 开始）</div>
              </template>
              <div class="row">
                <div class="col">
                  <div class="label">起始 Seq</div>
                  <el-input-number v-model="replayFromSeq" :min="1" :step="1" style="width: 100%" />
                </div>
                <div class="col">
                  <div class="label">倍速</div>
                  <el-select v-model="replaySpeedOnScreen" style="width: 100%">
                    <el-option :value="1" label="1x" />
                    <el-option :value="2" label="2x" />
                    <el-option :value="4" label="4x" />
                    <el-option :value="8" label="8x" />
                  </el-select>
                </div>
              </div>
              <div style="height: 12px"></div>
              <el-button
                type="primary"
                class="w-full"
                :disabled="!matchId"
                :loading="isSubmitting"
                @click="startReplayOnScreen"
              >
                开始回放
              </el-button>
              <div style="height: 8px"></div>
              <el-button class="w-full" :disabled="!matchId" :loading="isSubmitting" @click="exitReplayOnScreen">
                返回实时
              </el-button>
            </el-card>
          </div>
          </div>

          <div v-show="activeSection === 'maintenance'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">数据维护</h2>
              <p class="section-sub">危险操作：重置当前场次或清空全部数据，请谨慎。</p>
            </div>
          <div class="grid grid-single">
            <el-card shadow="hover" class="card small">
              <template #header>
                <div class="card-title">数据维护（重置）</div>
              </template>
              <el-button type="danger" class="w-full" :disabled="isSubmitting" :loading="resetSubmitting" @click="resetAllDB">
                重置全部数据
              </el-button>
              <div style="height: 10px"></div>
              <el-button
                type="danger"
                plain
                class="w-full"
                :disabled="!matchId || isSubmitting"
                :loading="resetSubmitting"
                @click="resetCurrentMatch"
              >
                重置当前场次
              </el-button>
            </el-card>
          </div>
          </div>

          <div v-show="activeSection === 'teams'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">队伍与复盘</h2>
              <p class="section-sub">维护参演队伍与成员，查看历史事件流并按 Seq 回放。</p>
            </div>
          <el-card shadow="hover" class="card">
            <template #header>
              <div class="row-between">
                <div class="card-title">参演队伍配置管理</div>
                <el-button type="primary" @click="openAddTeamDialog">新增队伍</el-button>
              </div>
            </template>
            <div class="row mb12">
              <div class="col">
                <el-input v-model="teamImportCSV" type="textarea" :rows="3" placeholder="CSV: name,type,members(成员用|分隔)" />
              </div>
            </div>
            <div class="row mb12">
              <el-button :disabled="!matchId" @click="importTeamsByCSV">CSV批量导入队伍</el-button>
              <el-button :disabled="!matchId" @click="bulkSyncTeams">批量同步当前表格</el-button>
            </div>

            <el-table :data="teams" style="width: 100%">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column label="成员" min-width="220">
                <template #default="{ row }">
                  <span class="mono">{{ (row.members ?? []).join(", ") || "-" }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="name" label="队伍名称" min-width="160" />
              <el-table-column prop="type" label="阵营" width="140">
                <template #default="{ row }">
                  <el-tag :type="row.type === 'red' ? 'danger' : 'primary'" effect="dark">
                    {{ row.type === 'red' ? '红队' : '蓝队' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="score" label="当前比分" width="140">
                <template #default="{ row }">
                  <span class="score">{{ row.score }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="180" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" @click="openEditTeamDialog(row)">编辑</el-button>
                  <el-button size="small" type="danger" plain @click="deleteTeam(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>

            <div style="height: 16px"></div>

            <div class="replay-block">
              <div class="row-between">
                <div>
                  <div class="card-title">历史复盘（事件流）</div>
                  <div class="hint">拉取并按序展示事件（后续可把飞线/图表也做成回放渲染）。</div>
                </div>
                <el-button :disabled="!matchId" @click="loadReplayEvents" :loading="replayLoading">
                  加载事件
                </el-button>
              </div>
              <div class="row" style="margin-top: 10px">
                <div class="col">
                  <el-input v-model="eventFilter.attack_type" placeholder="按战术类型过滤" clearable />
                </div>
                <div class="col">
                  <el-select v-model="eventFilter.status" placeholder="状态过滤" clearable style="width: 100%">
                    <el-option label="攻击成功" value="success" />
                    <el-option label="防守成功" value="defense_success" />
                    <el-option label="溯源成功" value="trace_success" />
                  </el-select>
                </div>
                <div class="col">
                  <el-input-number v-model="eventFilter.min_score" :step="50" placeholder="最小分值" style="width: 100%" />
                </div>
              </div>
              <el-table :data="replayEvents" style="width: 100%; margin-top: 12px" height="260">
                <el-table-column prop="seq" label="Seq" width="70" />
                <el-table-column prop="event_type" label="事件类型" width="180" />
                <el-table-column prop="created_at" label="时间" width="150" />
                <el-table-column label="payload" min-width="320">
                  <template #default="{ row }">
                    <span class="mono">{{ row.payload_preview }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="140">
                  <template #default="{ row }">
                    <el-button
                      size="small"
                      type="primary"
                      :disabled="!matchId"
                      @click="() => { replayFromSeq = row.seq; startReplayOnScreen(); }"
                    >
                      从此回放
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
          </div>

          <div v-show="activeSection === 'ops'" class="admin-section">
            <div class="section-header">
              <h2 class="section-title">运营与分析</h2>
              <p class="section-sub">场次模板、任务工单、回放书签、KPI 指标与审计日志。</p>
            </div>
            <div class="grid">
              <el-card shadow="hover" class="card">
                <template #header>
                  <div class="row-between">
                    <div class="card-title">场次模板系统</div>
                    <el-button size="small" @click="loadTemplates">刷新</el-button>
                  </div>
                </template>
                <div class="row mb12">
                  <div class="col"><el-input v-model="templateForm.name" placeholder="模板名称" /></div>
                  <div class="col"><el-input v-model="templateForm.id" placeholder="模板ID(可选)" /></div>
                </div>
                <div class="row mb12">
                  <div class="col"><el-input v-model="templateForm.map_type" placeholder="地图模式" /></div>
                  <div class="col"><el-input v-model="templateForm.attack_types_csv" placeholder="战术类型(逗号分隔)" /></div>
                </div>
                <el-button type="primary" @click="saveTemplate" :loading="isSubmitting">保存模板</el-button>
                <div style="height: 12px" />
                <el-table :data="templates" height="220">
                  <el-table-column prop="id" label="模板ID" min-width="120" />
                  <el-table-column prop="name" label="名称" min-width="120" />
                  <el-table-column prop="version" label="版本" width="80" />
                  <el-table-column prop="map_type" label="地图" width="100" />
                </el-table>
              </el-card>
              <el-card shadow="hover" class="card">
                <template #header>
                  <div class="row-between">
                    <div class="card-title">任务工单与回放书签</div>
                    <el-button size="small" :disabled="!matchId" @click="loadTasks">刷新</el-button>
                  </div>
                </template>
                <div class="row mb12">
                  <div class="col"><el-input v-model="taskForm.title" placeholder="任务标题" /></div>
                  <div class="col"><el-input v-model="taskForm.assignee" placeholder="负责人" /></div>
                </div>
                <div class="row mb12">
                  <div class="col"><el-input v-model="taskForm.category" placeholder="分类" /></div>
                  <div class="col">
                    <el-select v-model="taskForm.status" style="width: 100%">
                      <el-option label="待处理" value="todo" />
                      <el-option label="处理中" value="doing" />
                      <el-option label="已完成" value="done" />
                    </el-select>
                  </div>
                </div>
                <el-button type="primary" :disabled="!matchId" @click="createTask">创建工单</el-button>
                <div style="height: 12px" />
                <el-table :data="tasks" height="150">
                  <el-table-column prop="title" label="任务" min-width="140" />
                  <el-table-column prop="status" label="状态" width="100" />
                  <el-table-column prop="assignee" label="负责人" width="120" />
                </el-table>
                <el-divider />
                <div class="row mb12">
                  <div class="col"><el-input-number v-model="bookmarkForm.seq" :min="1" style="width: 100%" /></div>
                  <div class="col"><el-input v-model="bookmarkForm.title" placeholder="书签标题" /></div>
                </div>
                <el-button :disabled="!matchId" @click="createBookmark">保存书签</el-button>
              </el-card>
            </div>
            <div class="grid">
              <el-card shadow="hover" class="card">
                <template #header>
                  <div class="row-between">
                    <div class="card-title">KPI 与趋势</div>
                    <el-button size="small" :disabled="!matchId" @click="loadKpi">刷新</el-button>
                  </div>
                </template>
                <div class="row">
                  <div class="col"><div class="label">总事件数</div><div class="score">{{ kpi.total_events ?? 0 }}</div></div>
                  <div class="col"><div class="label">有效攻击率</div><div class="score">{{ formatRate(kpi.effective_attack_rate) }}</div></div>
                  <div class="col"><div class="label">溯源成功率</div><div class="score">{{ formatRate(kpi.trace_success_rate) }}</div></div>
                  <div class="col"><div class="label">净分差</div><div class="score">{{ kpi.net_score_diff ?? 0 }}</div></div>
                </div>
                <div style="height: 12px" />
                <el-table :data="trendRows" height="180">
                  <el-table-column prop="dimension" label="维度" min-width="100" />
                  <el-table-column prop="key" label="项" min-width="120" />
                  <el-table-column prop="value" label="数值" min-width="120" />
                </el-table>
              </el-card>
              <el-card shadow="hover" class="card">
                <template #header>
                  <div class="row-between">
                    <div class="card-title">复盘报告与审计日志</div>
                    <el-button size="small" :disabled="!matchId" @click="loadAuditLogs">刷新</el-button>
                  </div>
                </template>
                <div class="row mb12">
                  <el-select v-model="reportMode" style="width: 180px">
                    <el-option label="领导简版" value="leader" />
                    <el-option label="技术详版" value="tech" />
                  </el-select>
                  <el-button type="primary" :disabled="!matchId" @click="generateReport">生成 Markdown 报告</el-button>
                  <el-button :disabled="!matchId" @click="downloadReportPDF">导出 PDF</el-button>
                </div>
                <div class="mono" style="margin-top: 10px; max-height: 160px; overflow: auto; white-space: pre-wrap;">{{ reportMarkdown }}</div>
                <el-divider />
                <el-table :data="auditLogs" height="180">
                  <el-table-column prop="actor" label="操作人" width="110" />
                  <el-table-column prop="module" label="模块" width="110" />
                  <el-table-column prop="action" label="动作" min-width="120" />
                  <el-table-column prop="created_at" label="时间" min-width="120" />
                </el-table>
              </el-card>
            </div>
          </div>

          <!-- 新增/编辑队伍弹窗 -->
          <el-dialog
            v-model="teamDialogVisible"
            :title="isEditMode ? '编辑队伍' : '新增队伍'"
            width="520px"
          >
            <el-form :model="currentTeam" label-width="100px">
              <el-form-item label="队伍名称">
                <el-input v-model="currentTeam.name" />
              </el-form-item>
              <el-form-item label="阵营类型">
                <el-radio-group v-model="currentTeam.type">
                  <el-radio-button label="red">红队（攻击方）</el-radio-button>
                  <el-radio-button label="blue">蓝队（防守方）</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="成员列表">
                <el-input
                  v-model="currentTeam.membersText"
                  type="textarea"
                  :rows="3"
                  placeholder="一行一个成员/或用逗号分隔"
                />
              </el-form-item>
              <el-form-item label="初始分数">
                <el-input-number v-model="currentTeam.score" :step="100" style="width: 100%" />
              </el-form-item>
            </el-form>

            <template #footer>
              <el-button @click="teamDialogVisible = false">取消</el-button>
              <el-button type="primary" @click="saveTeam">确定并同步</el-button>
            </template>
          </el-dialog>
        </main>
      </div>
    </div>
    <audio ref="bgmPreviewAudioEl" loop preload="none"></audio>
    <audio ref="sfxPreviewAudioEl" preload="none"></audio>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";
import type { UploadRequestOptions } from "element-plus";
import type { TeamDTO } from "../shared/types";

// @ts-expect-error: Vetur may not infer correct TS module settings for import.meta
const deployMode = String((import.meta as any).env?.VITE_DEPLOY_MODE ?? "proxy").trim().toLowerCase();
const directApiBase = String((import.meta as any).env?.VITE_DIRECT_API_BASE ?? "http://127.0.0.1:8080").trim();
const rawApiBase = String((import.meta as any).env?.VITE_API_BASE ?? "").trim();
const apiBaseEnvRaw =
  deployMode === "direct" && (rawApiBase === "" || rawApiBase === "/api" || rawApiBase === "/api/")
    ? directApiBase
    : (rawApiBase || (deployMode === "direct" ? directApiBase : "/api"));
const apiBaseEnv = apiBaseEnvRaw.replace(/\/$/, "");
const apiBaseRoot = apiBaseEnv.endsWith("/api") ? apiBaseEnv.slice(0, -4) : apiBaseEnv;
const siteOrigin = window.location.origin;
const absoluteApiRoot = /^https?:\/\//i.test(apiBaseRoot) ? apiBaseRoot : siteOrigin;

// 这里沿用旧版城市字典（后续建议改为后台下发城市点位/坐标）。
const CITIES_CHINA = [
  "北京",
  "天津",
  "石家庄",
  "太原",
  "呼和浩特",
  "沈阳",
  "长春",
  "哈尔滨",
  "南京",
  "杭州",
  "合肥",
  "福州",
  "南昌",
  "济南",
  "郑州",
  "武汉",
  "长沙",
  "广州",
  "成都",
  "西安",
  "兰州",
  "西宁",
  "银川",
  "乌鲁木齐",
  "拉萨",
  "贵阳",
  "昆明",
  "南宁",
  "海口",
  "重庆",
  "上海",
];
const CITIES_TAIZHOU = ["市区", "海陵区", "高港区", "姜堰区", "兴化市", "靖江市", "泰兴市"];

// web 漏洞分类：用于 admin 选择 attack_type，并驱动大屏的战术统计/雷达维度
const attackTypes = ref<string[]>([
  "业务逻辑漏洞",
  "弱口令爆破",
  "账号撞库/凭证填充",
  "验证码绕过",

  "命令执行(RCE)",
  "SQL注入",
  "模板注入/表达式注入",
  "反序列化漏洞",
  "XXE外部实体注入",

  "XSS跨站脚本",
  "DOM型XSS",
  "CSRF跨站请求伪造",
  "点击劫持",

  "SSRF服务端请求伪造",
  "开放重定向",

  "路径遍历/任意文件读取",
  "任意文件下载",
  "任意文件上传",
  "文件包含漏洞",

  "未授权访问",
  "授权绕过(越权访问)",
  "会话管理漏洞",

  "敏感信息泄露",
  "错误信息泄露",
  "接口信息泄露",

  "安全配置错误"
]);
const customAttackTypeInput = ref("");

const axiosClient = axios.create({ baseURL: apiBaseRoot || undefined });

const authToken = ref<string>(localStorage.getItem("rb_jwt") ?? "");
const isAuthed = computed(() => !!authToken.value);
const loginForm = reactive({ username: "", password: "" });
const loginSubmitting = ref(false);
const loginPeek = ref(false);
const loginErrorPulse = ref(false);
const sloganTimer = ref<number | null>(null);
const sloganIndex = ref(0);
const slogans = [
  "Blue Team Defense · Real-time Telemetry",
  "Red Team Attack Paths · Visualized",
  "Judge Actions · Event-driven Replay",
  "One Match ID · Multi-screen Sync",
];
const activeSlogan = computed(() => slogans[sloganIndex.value % slogans.length]);

if (authToken.value) {
  axiosClient.defaults.headers.common.Authorization = `Bearer ${authToken.value}`;
}

axiosClient.interceptors.response.use(
  (resp) => resp,
  (err: any) => {
    const status = err?.response?.status;
    if (status === 401) {
      authToken.value = "";
      localStorage.removeItem("rb_jwt");
      ElMessage.warning("未授权或登录已过期，请重新登录");
    }
    return Promise.reject(err);
  }
);

async function requestWithRetry<T>(runner: () => Promise<T>, retries = 2, delayMs = 350): Promise<T> {
  let lastErr: any;
  for (let i = 0; i <= retries; i++) {
    try {
      return await runner();
    } catch (e: any) {
      lastErr = e;
      const status = e?.response?.status;
      const retryable = !status || status >= 500;
      if (!retryable || i === retries) break;
      await new Promise((resolve) => setTimeout(resolve, delayMs * (i + 1)));
    }
  }
  throw lastErr;
}

async function doLogin() {
  loginSubmitting.value = true;
  try {
    const res = await axios.post(`${apiBaseRoot}/api/admin/login`, loginForm);
    const token = res.data?.token as string | undefined;
    if (!token) throw new Error("token missing");

    authToken.value = token;
    localStorage.setItem("rb_jwt", token);
    axiosClient.defaults.headers.common.Authorization = `Bearer ${token}`;

    ElMessage.success("登录成功");

    // 登录后再拉取初始化数据
    await fetchMatches();
    await fetchState();
    await fetchTeams();
    await loadTemplates();
    await loadTasks();
    await loadKpi();
    await loadAuditLogs();
  } catch (e: any) {
    loginErrorPulse.value = true;
    window.setTimeout(() => {
      loginErrorPulse.value = false;
    }, 420);
    ElMessage.error(e?.response?.data?.error ?? "登录失败");
  } finally {
    loginSubmitting.value = false;
  }
}

function doLogout() {
  authToken.value = "";
  loginForm.password = "";
  localStorage.removeItem("rb_jwt");
  delete axiosClient.defaults.headers.common.Authorization;
  ElMessage.success("已退出登录");
}

type AdminSection = "combat" | "screen" | "broadcast" | "maintenance" | "teams" | "ops";
const activeSection = ref<AdminSection>("combat");
const matches = ref<{ id: string; map_type: string; leaderboard_visible: boolean }[]>([]);
const matchId = ref("");
const screenTitle = ref("");
const screenOrganizer = ref("");
const screenSupporter = ref("");
const bgmUrl = ref("");
const bgmEnabled = ref(false);
const successSfxUrl = ref("");
const successSfxEnabled = ref(false);
const leaderboardMainAlpha = ref(0.14);
const bgmPreviewAudioEl = ref<HTMLAudioElement | null>(null);
const sfxPreviewAudioEl = ref<HTMLAudioElement | null>(null);
const audioPlaying = reactive({ bgm: false });
/** 本场次得分总榜自定义背景 API 路径，如 /uploads/.../leaderboard-bg.png */
const leaderboardBgUrl = ref("");
const leaderboardBgPreview = computed(() => {
  const u = leaderboardBgUrl.value.trim();
  if (!u) return "";
  if (u.startsWith("/")) return `${absoluteApiRoot}${u}`;
  return u;
});
const replaySpeedOnScreen = ref(8);
const templates = ref<any[]>([]);
const selectedTemplateId = ref("");
const templateForm = reactive({
  id: "",
  name: "",
  map_type: "china",
  attack_types_csv: "",
});
const tasks = ref<any[]>([]);
const taskForm = reactive({
  category: "attack",
  title: "",
  status: "todo",
  assignee: "",
});
const bookmarkForm = reactive({
  seq: 1,
  title: "",
});
const kpi = ref<any>({});
const trends = ref<Record<string, any[]>>({});
const reportMarkdown = ref("");
const auditLogs = ref<any[]>([]);
const reportMode = ref<"leader" | "tech">("leader");
const teamImportCSV = ref("name,type,members\n红队一,red,成员A|成员B\n蓝队一,blue,成员C|成员D");
const broadcastTemplates = ref([
  "演练进入最后30分钟，请各队聚焦关键目标。",
  "裁判提示：请同步提交证据材料。",
  "请各队确认当前比分与事件序列。",
]);

const teams = ref<TeamDTO[]>([]);
const redTeams = computed(() => teams.value.filter((t) => t.type === "red"));
const hasBlueTeam = computed(() => teams.value.some((t) => t.type === "blue"));
const currentAttackMembers = computed(() => {
  const t = teams.value.find((x) => x.id === attackForm.team_id);
  return (t?.members ?? []) as string[];
});
const currentCities = computed(() => (mapMode.value === "china" ? CITIES_CHINA : CITIES_TAIZHOU));

const mapMode = ref<"china" | "taizhou">("china");
const leaderboardVisible = ref(true);

const isSubmitting = ref(false);
const resetSubmitting = ref(false);
const broadcastMsg = ref("");
const replayFromSeq = ref(1);
const autoMessage = ref(true);

const attackReady = computed(() => {
  const hasSource = attackForm.source_mode === "team" ? attackForm.source_team_id > 0 : !!attackForm.source_city;
  return (
    !!matchId.value &&
    teams.value.length > 0 &&
    attackForm.team_id > 0 &&
    hasSource &&
    !!attackForm.target_city &&
    !!attackForm.attack_type &&
    !!attackForm.message
  );
});

const manualReady = computed(() => {
  return (
    !!matchId.value &&
    teams.value.length > 0 &&
    manualScore.team_id > 0 &&
    !!manualScore.reason.trim()
  );
});

const attackForm = reactive({
  team_id: 0,
  status: "success" as "success" | "defense_success" | "trace_success",
  source_mode: "city" as "city" | "team",
  source_city: "北京",
  source_team_id: 0,
  target_city: "上海",
  target_unit: "",
  attack_type: attackTypes.value[0],
  score_change: 0,
  member: "",
  message: "",
});

const manualScore = reactive({
  team_id: 0,
  score_change: 100,
  reason: "",
});

function requireMatch() {
  if (!matchId.value) {
    ElMessage.warning("请先选择或创建场次");
    return false;
  }
  return true;
}

async function fetchMatches() {
  const res = await axiosClient.get("/api/matches");
  matches.value = res.data.matches ?? [];
  if (!matchId.value && matches.value.length) {
    matchId.value = matches.value[0].id;
  }
}

async function fetchTeams() {
  if (!matchId.value) return;
  const res = await axiosClient.get(`/api/matches/${matchId.value}/teams`);
  teams.value = res.data.teams ?? [];
  // 默认填充
  const red = teams.value.find((t) => t.type === "red");
  if (red) {
    attackForm.team_id = red.id;
    attackForm.source_team_id = red.id;
    attackForm.member = (red.members ?? [])[0] ?? "";
  }
  const anyTeam = teams.value[0];
  if (anyTeam) manualScore.team_id = anyTeam.id;
  genAttackMessage();
}

async function fetchState() {
  if (!matchId.value) return;
  const res = await axiosClient.get(`/api/matches/${matchId.value}/state`);
  const s = res.data.state;
  mapMode.value = s.map_type === "taizhou" ? "taizhou" : "china";
  leaderboardVisible.value = s.leaderboard_visible;
  screenTitle.value = s.screen_title ?? "实战化红蓝对抗演练指挥中心";
  screenOrganizer.value = s.screen_organizer ?? "";
  screenSupporter.value = s.screen_supporter ?? "";
  bgmUrl.value = s.bgm_url ?? "";
  bgmEnabled.value = !!s.bgm_enabled;
  successSfxUrl.value = s.success_sfx_url ?? "";
  successSfxEnabled.value = !!s.success_sfx_enabled;
  leaderboardMainAlpha.value = Number.isFinite(Number(s.leaderboard_main_alpha))
    ? Math.max(0, Math.min(1, Number(s.leaderboard_main_alpha)))
    : 0.14;
  leaderboardBgUrl.value = s.leaderboard_bg_url ?? "";
}

function onLeaderboardBgUploadSuccess() {
  ElMessage.success("得分总榜背景已更新，在线榜单将自动同步");
  void fetchState();
}

function onLeaderboardBgUploadError() {
  ElMessage.error("上传失败，请检查图片格式（PNG/JPEG/WebP）与大小（≤8MB）");
}

async function uploadLeaderboardBgHttpRequest(options: UploadRequestOptions) {
  if (!matchId.value) return;
  const form = new FormData();
  form.append("file", options.file as File);
  try {
    const res = await axiosClient.post(`/api/matches/${matchId.value}/leaderboard_background`, form);
    options.onSuccess?.(res.data as any);
  } catch (e: any) {
    options.onError?.(e);
  }
}

function resolveAudioURL(raw: string) {
  const u = raw.trim();
  if (!u) return "";
  if (/^https?:\/\//i.test(u)) return u;
  if (u.startsWith("/")) return `${absoluteApiRoot}${u}`;
  return u;
}

function onBgmUploadSuccess(resp: any) {
  const u = String(resp?.url ?? "").trim();
  if (u) bgmUrl.value = u;
  bgmEnabled.value = true;
  void (async () => {
    try {
      await saveAudioConfig();
      ElMessage.success("背景音乐上传成功，已自动开启并保存");
      await fetchState();
    } catch (e: any) {
      ElMessage.error(e?.response?.data?.error ?? "背景音乐自动保存失败，请手动点“保存音频配置”");
    }
  })();
}

function onSuccessSfxUploadSuccess(resp: any) {
  const u = String(resp?.url ?? "").trim();
  if (u) successSfxUrl.value = u;
  successSfxEnabled.value = true;
  void (async () => {
    try {
      await saveAudioConfig();
      ElMessage.success("成功音效上传成功，已自动开启并保存");
      await fetchState();
    } catch (e: any) {
      ElMessage.error(e?.response?.data?.error ?? "成功音效自动保存失败，请手动点“保存音频配置”");
    }
  })();
}

function onAudioUploadError(err: any) {
  const msg =
    err?.response?.data?.error ||
    err?.response?.data ||
    err?.message ||
    "音频上传失败，请检查格式（mp3/wav/ogg/m4a/aac）和大小（≤20MB）";
  ElMessage.error(String(msg));
}

async function testBgm() {
  const url = resolveAudioURL(bgmUrl.value);
  if (!url) return;
  const el = bgmPreviewAudioEl.value;
  if (!el) return;
  try {
    if (el.src !== url) el.src = url;
    el.volume = 0.32;
    await el.play();
    audioPlaying.bgm = true;
  } catch {
    ElMessage.error("背景音乐播放失败，请检查 URL 或浏览器自动播放限制");
  }
}

function stopBgm() {
  const el = bgmPreviewAudioEl.value;
  if (!el) return;
  el.pause();
  el.currentTime = 0;
  audioPlaying.bgm = false;
}

async function testSuccessSfx() {
  const url = resolveAudioURL(successSfxUrl.value);
  if (!url) return;
  const el = sfxPreviewAudioEl.value;
  if (!el) return;
  try {
    if (el.src !== url) el.src = url;
    el.currentTime = 0;
    el.volume = 0.9;
    await el.play();
  } catch {
    ElMessage.error("成功音效播放失败，请检查 URL 或浏览器自动播放限制");
  }
}

async function uploadAudioByKind(options: UploadRequestOptions, kind: "bgm" | "success_sfx") {
  if (!matchId.value) return;
  const form = new FormData();
  form.append("file", options.file as File);
  try {
    const res = await axiosClient.post(`/api/matches/${matchId.value}/audio_upload/${kind}`, form);
    options.onSuccess?.(res.data as any);
  } catch (e: any) {
    options.onError?.(e);
  }
}

function uploadBgmHttpRequest(options: UploadRequestOptions) {
  return uploadAudioByKind(options, "bgm");
}

function uploadSuccessSfxHttpRequest(options: UploadRequestOptions) {
  return uploadAudioByKind(options, "success_sfx");
}

async function clearLeaderboardBackground() {
  if (!matchId.value) return;
  try {
    await axiosClient.delete(`/api/matches/${matchId.value}/leaderboard_background`);
    ElMessage.success("已恢复默认背景");
    await fetchState();
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "清除失败");
  }
}

function genAttackMessage() {
  const team = teams.value.find((t) => t.id === attackForm.team_id);
  if (!team || !attackForm.target_city) return;
  const member =
    attackForm.member?.trim() && attackForm.member.trim().length > 0
      ? attackForm.member.trim()
      : (team.members?.[0] ?? team.name ?? "未知队伍");
  const unit = attackForm.target_unit?.trim() ?? "";
  const districtUnit = unit ? `${attackForm.target_city}-${unit}` : attackForm.target_city;

  let statusText = "发起";
  let verb = "攻破";
  if (attackForm.status === "success") {
    statusText = "成功通过";
    verb = "攻破";
  } else if (attackForm.status === "defense_success") {
    statusText = "防守成功";
    verb = "成功拦截";
  } else if (attackForm.status === "trace_success") {
    statusText = "溯源成功";
    verb = "定位溯源";
  }

  // 大屏播报模板：
  // [队伍][队员]成功通过[战术手段]攻破[区县-单位]防御节点
  // 其中“单位”可由管理员在“单位名称”输入框自定义。
  const msg = `[${team.name}][${member}]${statusText}[${attackForm.attack_type}]${verb}[${districtUnit}]防御节点`;
  if (autoMessage.value) attackForm.message = msg;
}

watch(
  () => [
    attackForm.team_id,
    attackForm.source_mode,
    attackForm.source_team_id,
    attackForm.member,
    attackForm.target_unit,
    attackForm.status,
    attackForm.source_city,
    attackForm.target_city,
    attackForm.attack_type,
  ],
  () => genAttackMessage()
);

watch(
  () => attackForm.team_id,
  () => {
    const members = currentAttackMembers.value;
    if (!members.length) {
      attackForm.member = "";
      return;
    }
    if (!members.includes(attackForm.member)) attackForm.member = members[0];
    if (attackForm.source_mode === "team") attackForm.source_team_id = attackForm.team_id;
  }
);

watch(mapMode, (mode) => {
  const cities = mode === "china" ? CITIES_CHINA : CITIES_TAIZHOU;
  if (!cities.length) return;

  if (!cities.includes(attackForm.source_city)) {
    attackForm.source_city = cities[0];
  }
  if (!cities.includes(attackForm.target_city)) {
    attackForm.target_city = cities[1] ?? cities[0];
  }

  genAttackMessage();
});

watch(
  () => attackForm.source_mode,
  (mode) => {
    if (mode === "team") {
      if (!redTeams.value.find((t) => t.id === attackForm.source_team_id)) {
        attackForm.source_team_id = attackForm.team_id || redTeams.value[0]?.id || 0;
      }
    } else {
      const cities = mapMode.value === "china" ? CITIES_CHINA : CITIES_TAIZHOU;
      if (!cities.includes(attackForm.source_city)) attackForm.source_city = cities[0] ?? "";
    }
    genAttackMessage();
  }
);

function addAttackType() {
  const name = customAttackTypeInput.value.trim();
  if (!name) return ElMessage.warning("请输入战术手段名称");
  if (attackTypes.value.includes(name)) return ElMessage.warning("该战术手段已存在");
  attackTypes.value.push(name);
  customAttackTypeInput.value = "";
  attackForm.attack_type = name;
  genAttackMessage();
}

function removeAttackType(type: string) {
  if (attackTypes.value.length <= 1) return ElMessage.warning("至少保留一个战术手段");
  attackTypes.value = attackTypes.value.filter((t) => t !== type);
  if (attackForm.attack_type === type) {
    attackForm.attack_type = attackTypes.value[0] ?? "";
    genAttackMessage();
  }
}

async function createMatch() {
  isSubmitting.value = true;
  try {
    const res = await axiosClient.post("/api/matches", { template_id: selectedTemplateId.value || "" });
    const id = res.data.match_id;
    ElMessage.success("已创建新场次");
    matchId.value = id;
    await fetchMatches();
    await fetchState();
    await fetchTeams();
    activeSection.value = "combat";
  } finally {
    isSubmitting.value = false;
  }
}

function openScreen() {
  if (!matchId.value) return;
  const url = `${window.location.origin}/screen-vite.html?match_id=${encodeURIComponent(
    matchId.value
  )}&token=${encodeURIComponent(authToken.value)}`;
  window.open(url, "_blank");
}

/** 独立全屏：战队得分总榜（需同场 JWT） */
function openLeaderboard() {
  if (!matchId.value) return;
  const url = `${window.location.origin}/leaderboard-vite.html?match_id=${encodeURIComponent(
    matchId.value
  )}&token=${encodeURIComponent(authToken.value)}&api_base=${encodeURIComponent(apiBaseRoot || siteOrigin)}`;
  window.open(url, "_blank");
}

async function copyMatchId() {
  if (!matchId.value) return;
  try {
    await navigator.clipboard.writeText(matchId.value);
    ElMessage.success("match_id 已复制");
  } catch {
    // 兼容：不支持 clipboard 时走选中复制
    const el = document.createElement("textarea");
    el.value = matchId.value;
    document.body.appendChild(el);
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
    ElMessage.success("match_id 已复制");
  }
}

async function onMatchChange() {
  if (!matchId.value) return;
  await fetchState();
  await fetchTeams();
  await loadTasks();
  await loadKpi();
  await loadAuditLogs();
}

async function submitCommand(eventType: string, payload: any) {
  if (!requireMatch()) return;
  isSubmitting.value = true;
  try {
    await axiosClient.post(`/api/matches/${matchId.value}/command`, {
      event_type: eventType,
      data: payload,
    });
    ElMessage.success("指令已推送");
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "推送失败");
  } finally {
    isSubmitting.value = false;
  }
}

async function submitManualScore() {
  if (!requireMatch()) return;
  if (!manualScore.reason.trim()) {
    ElMessage.warning("请输入计分原因");
    return;
  }
  const team = teams.value.find((t) => t.id === manualScore.team_id);
  const message = `裁判判定：为 [${team?.name ?? "未知队伍"}] ${manualScore.score_change >= 0 ? "增加" : "扣除"} ${Math.abs(
    manualScore.score_change
  )} 分。原因：${manualScore.reason}`;
  await submitCommand("manual_score", {
    team_id: manualScore.team_id,
    score_change: manualScore.score_change,
    message,
    reason: manualScore.reason,
  });
}

async function submitBroadcast() {
  if (!requireMatch()) return;
  await submitCommand("system_broadcast", { message: broadcastMsg.value });
}

async function startReplayOnScreen() {
  if (!requireMatch()) return;
  if (replayFromSeq.value < 1) replayFromSeq.value = 1;
  await submitCommand("replay_start", { from_seq: replayFromSeq.value, speed: replaySpeedOnScreen.value });
}

async function exitReplayOnScreen() {
  if (!requireMatch()) return;
  await submitCommand("replay_exit", {});
}

async function switchMap() {
  await submitCommand("switch_map", { map_type: mapMode.value });
}

async function togglePanel() {
  await submitCommand("toggle_panel", { panel_id: "panel-leaderboard", visible: leaderboardVisible.value });
}

async function saveScreenTitle() {
  await submitCommand("set_screen_title", { title: screenTitle.value });
}

async function saveScreenCredits() {
  await submitCommand("set_screen_credits", { organizer: screenOrganizer.value, supporter: screenSupporter.value });
}

async function saveAudioConfig() {
  await submitCommand("set_audio_config", {
    bgm_url: bgmUrl.value.trim(),
    bgm_enabled: bgmEnabled.value,
    success_sfx_url: successSfxUrl.value.trim(),
    success_sfx_enabled: successSfxEnabled.value,
  });
}

async function saveLeaderboardStyle() {
  await submitCommand("set_leaderboard_style", { main_alpha: Number(leaderboardMainAlpha.value) });
}

async function resetCurrentMatch() {
  if (!matchId.value) return;
  try {
    await ElMessageBox.confirm("确定要重置当前场次吗？这会清空该场次的队伍与事件数据。", "确认重置", {
      type: "warning",
    });
  } catch {
    return;
  }

  resetSubmitting.value = true;
  try {
    await axiosClient.post("/api/admin/reset", { confirm: "redblue-reset", match_id: matchId.value });
    ElMessage.success("当前场次已重置");
    matchId.value = "";
    await fetchMatches();
    await fetchState();
    await fetchTeams();
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "重置失败");
  } finally {
    resetSubmitting.value = false;
  }
}

async function resetAllDB() {
  try {
    await ElMessageBox.confirm("确定要重置整个数据库吗？这会清空所有场次的队伍与事件数据。", "确认重置", {
      type: "warning",
    });
  } catch {
    return;
  }

  resetSubmitting.value = true;
  try {
    await axiosClient.post("/api/admin/reset", { confirm: "redblue-reset" });
    ElMessage.success("数据库已重置");
    matchId.value = "";
    await fetchMatches();
    await fetchState();
    await fetchTeams();
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "重置失败");
  } finally {
    resetSubmitting.value = false;
  }
}

// === 队伍 CRUD ===
const teamDialogVisible = ref(false);
const isEditMode = ref(false);
const currentTeam = reactive<{
  id: number | null;
  name: string;
  type: "red" | "blue";
  membersText: string;
  score: number;
}>({
  id: null,
  name: "",
  type: "red",
  membersText: "",
  score: 0,
});

function openAddTeamDialog() {
  isEditMode.value = false;
  currentTeam.id = null;
  currentTeam.name = "";
  currentTeam.type = "red";
  currentTeam.membersText = "";
  currentTeam.score = 0;
  teamDialogVisible.value = true;
}

function openEditTeamDialog(row: TeamDTO) {
  isEditMode.value = true;
  currentTeam.id = row.id;
  currentTeam.name = row.name;
  currentTeam.type = row.type;
  currentTeam.membersText = (row.members ?? []).join("\n");
  currentTeam.score = row.score;
  teamDialogVisible.value = true;
}

async function saveTeam() {
  if (!requireMatch()) return;
  if (!currentTeam.name.trim()) {
    ElMessage.warning("队伍名称不能为空");
    return;
  }

  try {
    const members = currentTeam.membersText
      .split(/[,，\n]/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (isEditMode.value && currentTeam.id != null) {
      await axiosClient.put(`/api/matches/${matchId.value}/teams/${currentTeam.id}`, {
        name: currentTeam.name,
        type: currentTeam.type,
        members,
        score: currentTeam.score,
      });
      ElMessage.success("修改成功");
    } else {
      const res = await axiosClient.post(`/api/matches/${matchId.value}/teams`, {
        name: currentTeam.name,
        type: currentTeam.type,
        members,
        score: currentTeam.score,
      });
      ElMessage.success("添加成功");
      // match 端会通过 teams_updated 推送同步到大屏
      // admin 端则需要自己刷新一次表格
      void res;
    }
    teamDialogVisible.value = false;
    await fetchTeams();
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "保存失败");
  }
}

async function deleteTeam(row: TeamDTO) {
  if (!requireMatch()) return;
  try {
    await axiosClient.delete(`/api/matches/${matchId.value}/teams/${row.id}`);
    ElMessage.success("队伍已删除");
    await fetchTeams();
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "删除失败");
  }
}

function formatEventPayloadPreview(payload: any) {
  try {
    const s = JSON.stringify(payload);
    return s.length > 140 ? s.slice(0, 140) + "..." : s;
  } catch {
    return String(payload);
  }
}

// === 回放事件 ===
const replayLoading = ref(false);
const replayEvents = ref<
  Array<{
    seq: number;
    event_type: string;
    created_at: number;
    payload_preview: string;
  }>
>([]);
const eventFilter = reactive({
  attack_type: "",
  status: "",
  min_score: 0,
});

async function loadReplayEvents() {
  if (!requireMatch()) return;
  replayLoading.value = true;
  try {
    const params: any = { from_seq: 1, limit: 5000 };
    if (eventFilter.attack_type.trim()) params.attack_type = eventFilter.attack_type.trim();
    if (eventFilter.status.trim()) params.status = eventFilter.status.trim();
    if (eventFilter.min_score > 0) params.min_score = eventFilter.min_score;
    const res = await requestWithRetry(() => axiosClient.get(`/api/matches/${matchId.value}/events_enhanced`, { params }));
    const evs = res.data.events ?? [];
    replayEvents.value = evs.map((ev: any) => {
      const payloadObj = ev.payload_raw ?? ev.PayloadRaw ?? ev.payloadRaw ?? ev.payload_json ?? ev.payload;
      return {
        seq: ev.seq ?? ev.Seq,
        event_type: ev.event_type ?? ev.EventType ?? ev.eventType,
        created_at: ev.timestamp ?? ev.Timestamp ?? ev.created_at,
        payload_preview: formatEventPayloadPreview(payloadObj),
      };
    });
    ElMessage.success("已加载事件");
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error ?? "加载失败");
  } finally {
    replayLoading.value = false;
  }
}

function formatRate(v: number | undefined) {
  const n = Number(v ?? 0);
  return `${(n * 100).toFixed(1)}%`;
}
const trendRows = computed(() =>
  Object.entries(trends.value ?? {}).flatMap(([dimension, list]) =>
    (list ?? []).map((x: any) => ({
      dimension,
      key: x.key,
      value: x.value,
    }))
  )
);

async function loadTemplates() {
  try {
    const res = await requestWithRetry(() => axiosClient.get("/api/match_templates"));
    templates.value = res.data.templates ?? [];
  } catch {
    ElMessage.error("模板加载失败");
  }
}

async function saveTemplate() {
  try {
    await axiosClient.post("/api/match_templates", {
      id: templateForm.id.trim(),
      name: templateForm.name.trim(),
      map_type: templateForm.map_type.trim() || "china",
      version: 1,
      attack_types: templateForm.attack_types_csv.split(/[,，]/).map((s) => s.trim()).filter(Boolean),
      cities: mapMode.value === "china" ? CITIES_CHINA : CITIES_TAIZHOU,
      score_rules: { attack_success: 100, trace_success: 80 },
      audio_config: { bgm_enabled: bgmEnabled.value, success_sfx_enabled: successSfxEnabled.value },
      change_log: "admin 创建/更新模板",
    });
    ElMessage.success("模板已保存");
    await loadTemplates();
  } catch {
    ElMessage.error("模板保存失败");
  }
}

async function loadTasks() {
  if (!matchId.value) return;
  try {
    const res = await requestWithRetry(() => axiosClient.get(`/api/matches/${matchId.value}/tasks`));
    tasks.value = res.data.tasks ?? [];
  } catch {
    ElMessage.error("工单加载失败");
  }
}

async function createTask() {
  if (!matchId.value) return;
  try {
    await axiosClient.post(`/api/matches/${matchId.value}/tasks`, {
      category: taskForm.category,
      title: taskForm.title,
      status: taskForm.status,
      assignee: taskForm.assignee,
      payload: { source: "admin" },
    });
    ElMessage.success("工单已创建");
    taskForm.title = "";
    await loadTasks();
  } catch {
    ElMessage.error("工单创建失败");
  }
}

async function createBookmark() {
  if (!matchId.value) return;
  try {
    await axiosClient.post(`/api/matches/${matchId.value}/bookmarks`, {
      seq: bookmarkForm.seq,
      title: bookmarkForm.title || `Seq ${bookmarkForm.seq}`,
      note: "管理员书签",
    });
    ElMessage.success("书签已保存");
  } catch {
    ElMessage.error("书签保存失败");
  }
}

async function loadKpi() {
  if (!matchId.value) return;
  try {
    const [kpiRes, trendRes] = await Promise.all([
      requestWithRetry(() => axiosClient.get(`/api/matches/${matchId.value}/analytics/kpi`)),
      requestWithRetry(() => axiosClient.get(`/api/matches/${matchId.value}/analytics/trends`)),
    ]);
    kpi.value = kpiRes.data.kpi ?? {};
    trends.value = trendRes.data.trends ?? {};
  } catch {
    ElMessage.error("KPI 加载失败");
  }
}

async function generateReport() {
  if (!matchId.value) return;
  try {
    const res = await axiosClient.get(`/api/matches/${matchId.value}/report`, { params: { mode: reportMode.value } });
    reportMarkdown.value = String(res.data.markdown ?? "");
  } catch {
    ElMessage.error("报告生成失败");
  }
}

async function downloadReportPDF() {
  if (!matchId.value) return;
  try {
    const res = await axiosClient.get(`/api/matches/${matchId.value}/report`, {
      params: { mode: reportMode.value, format: "pdf" },
      responseType: "blob",
    });
    const blob = new Blob([res.data], { type: "application/pdf" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${matchId.value}-${reportMode.value}-report.pdf`;
    a.click();
    URL.revokeObjectURL(url);
  } catch {
    ElMessage.error("PDF导出失败");
  }
}

async function loadAuditLogs() {
  if (!matchId.value) return;
  try {
    const res = await requestWithRetry(() => axiosClient.get(`/api/matches/${matchId.value}/audit_logs`));
    auditLogs.value = res.data.audit_logs ?? [];
  } catch {
    ElMessage.error("审计日志加载失败");
  }
}

function handleAdminHotkeys(e: KeyboardEvent) {
  if (!isAuthed.value) return;
  if (!(e.ctrlKey || e.metaKey)) return;
  const key = e.key.toLowerCase();
  if (key === "b") {
    e.preventDefault();
    if (broadcastMsg.value.trim()) void submitBroadcast();
  } else if (key === "r") {
    e.preventDefault();
    void startReplayOnScreen();
  } else if (key === "m") {
    e.preventDefault();
    if (audioPlaying.bgm) stopBgm();
    else void testBgm();
  }
}

async function importTeamsByCSV() {
  if (!matchId.value) return;
  try {
    await axiosClient.post(`/api/matches/${matchId.value}/teams/import`, { csv_text: teamImportCSV.value });
    ElMessage.success("CSV导入完成");
    await fetchTeams();
  } catch {
    ElMessage.error("CSV导入失败");
  }
}

async function bulkSyncTeams() {
  if (!matchId.value) return;
  try {
    await axiosClient.put(`/api/matches/${matchId.value}/teams/batch_update`, { teams: teams.value });
    ElMessage.success("批量同步完成");
    await fetchTeams();
  } catch {
    ElMessage.error("批量同步失败");
  }
}

onMounted(async () => {
  sloganTimer.value = window.setInterval(() => {
    sloganIndex.value = (sloganIndex.value + 1) % slogans.length;
  }, 2600);
  if (!isAuthed.value) return;
  await fetchMatches();
  await fetchState();
  await fetchTeams();
  await loadTemplates();
  window.addEventListener("keydown", handleAdminHotkeys);
});

onBeforeUnmount(() => {
  if (sloganTimer.value != null) {
    window.clearInterval(sloganTimer.value);
    sloganTimer.value = null;
  }
  window.removeEventListener("keydown", handleAdminHotkeys);
});
</script>

<style scoped>
.admin-root {
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.16), transparent 32%),
    radial-gradient(circle at bottom right, rgba(139, 92, 246, 0.16), transparent 34%),
    #edf2ff;
  color: #111827;
}
.admin-header {
  height: 72px;
  background: rgba(255, 255, 255, 0.72);
  border-bottom: 1px solid rgba(148, 163, 184, 0.22);
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(9px);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}
.admin-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.admin-logo {
  width: 12px;
  height: 32px;
  border-radius: 6px;
  background: linear-gradient(180deg, #2563eb, #7c3aed);
  box-shadow: 0 8px 16px rgba(59, 130, 246, 0.35);
}
.admin-title {
  font-size: 15px;
  font-weight: 700;
  color: #111827;
}
.admin-subtitle {
  margin-top: 2px;
  font-size: 11px;
  color: rgba(17, 24, 39, 0.48);
}
.admin-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.admin-body {
  padding: 0;
  height: calc(100vh - 72px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.admin-shell {
  flex: 1;
  display: flex;
  min-height: 0;
}

.admin-sidebar {
  width: 268px;
  flex: 0 0 268px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.8), rgba(248, 250, 252, 0.86));
  border-right: 1px solid rgba(148, 163, 184, 0.25);
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  box-shadow: 2px 0 12px rgba(17, 24, 39, 0.04);
}

.sidebar-head {
  padding: 8px 8px 4px;
  margin-bottom: 4px;
}
.sidebar-head-title {
  font-size: 13px;
  font-weight: 800;
  color: #111827;
  letter-spacing: 0.02em;
}
.sidebar-head-sub {
  margin-top: 4px;
  font-size: 11px;
  color: rgba(17, 24, 39, 0.45);
  line-height: 1.35;
}

.sidebar-divider {
  height: 1px;
  background: #e5e7eb;
  margin: 6px 4px 0;
}

.nav-card {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  text-align: left;
  padding: 12px 12px;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  background: #f9fafb;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background 0.15s ease,
    box-shadow 0.15s ease;
}
.nav-card:hover {
  border-color: rgba(37, 99, 235, 0.35);
  background: #ffffff;
  box-shadow: 0 2px 10px rgba(17, 24, 39, 0.06);
}
.nav-card.active {
  border-color: rgba(37, 99, 235, 0.55);
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.08), #ffffff);
  box-shadow: 0 2px 14px rgba(37, 99, 235, 0.12);
}
.nav-card-warn:not(.active):hover {
  border-color: rgba(220, 38, 38, 0.35);
}
.nav-card-warn.active {
  border-color: rgba(220, 38, 38, 0.45);
  background: linear-gradient(135deg, rgba(220, 38, 38, 0.06), #ffffff);
}

.nav-card-ico {
  flex: 0 0 auto;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.12);
  border: 1px solid rgba(37, 99, 235, 0.2);
  line-height: 1;
}
.nav-card-warn .nav-card-ico {
  color: #b91c1c;
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.18);
}
.nav-card-body {
  min-width: 0;
}
.nav-card-title {
  font-size: 14px;
  font-weight: 800;
  color: #111827;
}
.nav-card-desc {
  margin-top: 4px;
  font-size: 11px;
  color: rgba(17, 24, 39, 0.5);
  line-height: 1.35;
}

.admin-main {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 18px 20px 28px;
  background: transparent;
}

.admin-section {
  max-width: 1280px;
}
.section-header {
  margin-bottom: 16px;
}
.section-title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
  color: #111827;
  letter-spacing: 0.02em;
}
.section-sub {
  margin: 8px 0 0;
  font-size: 13px;
  color: rgba(17, 24, 39, 0.55);
  line-height: 1.45;
  max-width: 720px;
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}
.grid-single {
  grid-template-columns: 1fr;
  max-width: 960px;
}
@media (max-width: 1100px) {
  .admin-shell {
    flex-direction: column;
  }
  .admin-sidebar {
    width: 100%;
    flex: 0 0 auto;
    border-right: none;
    border-bottom: 1px solid #e5e7eb;
    flex-direction: row;
    flex-wrap: wrap;
    padding: 12px;
  }
  .sidebar-head {
    width: 100%;
  }
  .sidebar-divider {
    width: 100%;
    margin: 4px 0 8px;
  }
  .nav-card {
    flex: 1 1 200px;
    min-width: 160px;
  }
}
.card {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.84), rgba(248, 250, 252, 0.92));
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 14px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
  backdrop-filter: blur(4px);
}
.small {
  height: 100%;
}
.card-title {
  font-weight: 800;
  font-size: 16px;
}
.row {
  display: flex;
  gap: 16px;
}
.col {
  flex: 1;
}
.label {
  font-size: 12px;
  color: rgba(17, 24, 39, 0.6);
  margin-bottom: 6px;
}
.mt8 {
  margin-top: 8px;
}
.mb12 {
  margin-bottom: 12px;
}
.row-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.w-full {
  width: 100%;
}

.logo {
  font-size: 26px;
}
.score {
  font-weight: 700;
  color: #2563eb;
}

.replay-block .hint {
  color: rgba(17, 24, 39, 0.55);
  font-size: 12px;
  margin-top: 6px;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  word-break: break-word;
}

.attack-custom {
  margin-top: 10px;
}
.attack-custom-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.attack-custom-tags {
  margin-top: 8px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.tag {
  background: rgba(37, 99, 235, 0.08);
  border: 1px solid rgba(37, 99, 235, 0.18);
  color: #1d4ed8;
}

.auto-msg-row {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.auto-msg-label {
  font-size: 12px;
  color: rgba(17, 24, 39, 0.65);
}

.admin-login-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background:
    radial-gradient(circle at 8% 12%, rgba(59, 130, 246, 0.22), transparent 36%),
    radial-gradient(circle at 92% 88%, rgba(99, 102, 241, 0.22), transparent 38%),
    linear-gradient(135deg, rgba(15, 23, 42, 0.9), rgba(30, 41, 59, 0.9));
}
.login-orb {
  position: absolute;
  border-radius: 999px;
  filter: blur(2px);
  opacity: 0.55;
  pointer-events: none;
  animation: orbFloat 9s ease-in-out infinite;
}
.orb-a {
  width: 220px;
  height: 220px;
  left: 8%;
  top: 14%;
  background: radial-gradient(circle at 30% 30%, rgba(147, 197, 253, 0.9), rgba(59, 130, 246, 0.06));
}
.orb-b {
  width: 260px;
  height: 260px;
  right: 7%;
  bottom: 10%;
  animation-delay: 1.8s;
  background: radial-gradient(circle at 30% 30%, rgba(196, 181, 253, 0.9), rgba(124, 58, 237, 0.06));
}
.admin-login-shell {
  width: min(980px, 100%);
  min-height: 520px;
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  border-radius: 20px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.28);
  box-shadow: 0 20px 56px rgba(2, 6, 23, 0.5);
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(10px);
}
.admin-login-brand {
  padding: 48px 44px;
  color: #e2e8f0;
  background:
    radial-gradient(circle at 12% 20%, rgba(125, 211, 252, 0.24), transparent 38%),
    radial-gradient(circle at 85% 72%, rgba(167, 139, 250, 0.3), transparent 42%),
    linear-gradient(145deg, #0f172a, #1e293b);
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 16px;
}
.admin-login-brand-kicker {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: rgba(148, 163, 184, 0.92);
  text-transform: uppercase;
}
.admin-login-brand-title {
  margin: 0;
  font-size: 34px;
  line-height: 1.2;
  letter-spacing: 0.02em;
}
.admin-login-brand-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: rgba(226, 232, 240, 0.88);
}
.admin-login-brand-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.login-feature-list {
  margin: 6px 0 0;
  padding-left: 18px;
  color: rgba(226, 232, 240, 0.88);
  font-size: 12px;
  line-height: 1.55;
  display: grid;
  gap: 6px;
}
.login-slogan-rotator {
  margin-top: 8px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid rgba(125, 211, 252, 0.2);
  background: rgba(15, 23, 42, 0.32);
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rotator-label {
  font-size: 10px;
  color: rgba(125, 211, 252, 0.85);
  letter-spacing: 0.18em;
}
.rotator-text {
  font-size: 13px;
  color: #f8fafc;
  letter-spacing: 0.02em;
}
.login-tag {
  padding: 5px 10px;
  border-radius: 999px;
  font-size: 12px;
  color: #dbeafe;
  background: rgba(59, 130, 246, 0.22);
  border: 1px solid rgba(125, 211, 252, 0.32);
}
.admin-login-card {
  padding: 44px 38px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(248, 250, 252, 0.88));
  display: flex;
  flex-direction: column;
  justify-content: center;
  position: relative;
}
.admin-login-card.login-error {
  animation: loginShake 0.38s ease;
}
.admin-login-card-head {
  margin-bottom: 8px;
}
.admin-login-card-title {
  font-size: 28px;
  font-weight: 800;
  color: #0f172a;
}
.admin-login-card-sub {
  margin-top: 6px;
  font-size: 13px;
  color: rgba(15, 23, 42, 0.58);
}
.admin-login-actions {
  margin-top: 4px;
  display: flex;
}
.admin-login-mascot {
  margin: 2px auto 14px;
  position: relative;
  width: 96px;
  height: 78px;
  transition: transform 0.25s ease;
}
.admin-login-mascot.peeking {
  transform: translateY(-2px);
}
.mascot-head {
  position: absolute;
  left: 14px;
  top: 8px;
  width: 68px;
  height: 54px;
  border-radius: 22px 22px 20px 20px;
  background: linear-gradient(180deg, #fde68a, #f59e0b);
  border: 2px solid rgba(15, 23, 42, 0.16);
  box-shadow: 0 6px 16px rgba(245, 158, 11, 0.28);
}
.mascot-eye {
  position: absolute;
  top: 18px;
  width: 12px;
  height: 12px;
  background: #0f172a;
  border-radius: 50%;
}
.eye-left {
  left: 17px;
}
.eye-right {
  right: 17px;
}
.admin-login-mascot.peeking .mascot-eye {
  transform: scaleY(0.25);
  transform-origin: center;
}
.mascot-mouth {
  position: absolute;
  left: 50%;
  bottom: 12px;
  width: 18px;
  height: 8px;
  transform: translateX(-50%);
  border-bottom: 3px solid rgba(15, 23, 42, 0.85);
  border-radius: 0 0 16px 16px;
}
.mascot-hand {
  position: absolute;
  top: 46px;
  width: 20px;
  height: 20px;
  border-radius: 999px;
  background: #fbbf24;
  border: 2px solid rgba(15, 23, 42, 0.16);
  transition: transform 0.25s ease;
}
.hand-left {
  left: 4px;
}
.hand-right {
  right: 4px;
}
.admin-login-mascot.peeking .hand-left {
  transform: translateX(8px) translateY(-8px) rotate(-12deg);
}
.admin-login-mascot.peeking .hand-right {
  transform: translateX(-8px) translateY(-8px) rotate(12deg);
}
.admin-login-submit {
  width: 100%;
  height: 44px;
  font-weight: 700;
  letter-spacing: 0.06em;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.admin-login-submit:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(37, 99, 235, 0.35);
}
.admin-login-hint {
  margin-top: 14px;
  font-size: 12px;
  color: rgba(226, 232, 240, 0.9);
  text-align: center;
}
:deep(.neon-input .el-input__wrapper) {
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}
:deep(.neon-input .el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1px rgba(59, 130, 246, 0.65),
    0 0 0 4px rgba(59, 130, 246, 0.18) !important;
}
@keyframes loginShake {
  0%,
  100% {
    transform: translateX(0);
  }
  20% {
    transform: translateX(-7px);
  }
  40% {
    transform: translateX(7px);
  }
  60% {
    transform: translateX(-5px);
  }
  80% {
    transform: translateX(5px);
  }
}
@keyframes orbFloat {
  0%,
  100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-16px);
  }
}
@media (max-width: 900px) {
  .admin-login-shell {
    grid-template-columns: 1fr;
    min-height: auto;
  }
  .admin-login-brand {
    padding: 28px 24px;
  }
  .admin-login-card {
    padding: 28px 24px;
  }
  .admin-login-brand-title {
    font-size: 26px;
  }
}
</style>

<style>
body {
  background: #f6f7fb !important;
  color: #111827 !important;
  overflow: auto !important;
}
</style>
