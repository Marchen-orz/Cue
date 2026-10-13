<template>
  <div class="main-interface interview-view" :class="{ visible: ui.mainVisible, opaque: isOpaque }">
    <!-- 内容面板（表面背景跟随全局透明度；不透明模式在视图作用域拉满 alpha 分量） -->
    <div class="iv-panel">
      <!-- 状态栏 -->
      <div class="iv-status">
        <span class="iv-status-dot" :class="{ on: interview.isListening, pulse: interview.isListening && !interview.isAnswering }"></span>
        <span class="iv-status-text">
          {{ statusText }}
        </span>
        <span class="iv-mode-tag" v-if="interview.isOn">{{ modeLabel }}</span>
        <div class="iv-status-actions">
          <button
            class="iv-action-btn"
            @click="isOpaque = !isOpaque"
            :title="isOpaque ? '恢复跟随全局透明度' : '临时切换为不透明（不改全局设置）'"
          >{{ isOpaque ? '透明' : '不透明' }}</button>
          <button
            v-if="interview.isOn && interview.isListening"
            class="iv-action-btn accent"
            @click="triggerAnswer"
            :disabled="interview.isAnswering"
            :title="isManualMode ? '用当前字幕立即回答（F3）' : '不等停顿，立即回答当前字幕'"
          >立即回答</button>
          <button
            v-if="interview.isOn"
            class="iv-action-btn"
            @click="togglePause"
          >{{ interview.isListening ? '暂停收音' : '继续收音' }}</button>
          <button
            v-if="interview.isOn"
            class="iv-action-btn danger"
            @click="stopInterview"
            title="结束会话并清空对话"
          >结束</button>
        </div>
      </div>

      <!-- 对话区 -->
      <div class="iv-content" id="interview-content">
        <!-- 历史问答对 -->
        <div v-for="(round, idx) in interview.rounds" :key="idx" class="iv-round">
          <div v-if="round.question" class="iv-question">
            <span class="iv-question-label">问</span>
            <div class="iv-question-text">{{ round.question }}</div>
          </div>
          <div v-if="round.skills && round.skills.length" class="iv-skills">
            <Icon name="sparkles" :size="12" />
            <span v-for="n in round.skills" :key="n" class="iv-skill-chip">{{ n }}</span>
          </div>
          <div v-if="round.html" class="iv-answer md-body">
            <div class="ai-response" v-html="round.html"></div>
            <span v-if="interview.isAnswering && idx === interview.rounds.length - 1" class="stream-cursor">▍</span>
          </div>
        </div>

        <!-- 实时字幕（收音中，尚未回答） -->
        <div v-if="interview.isListening && interview.transcript" class="iv-transcript">
          <span class="iv-transcript-label">字幕</span>
          <span class="iv-transcript-text">{{ interview.transcript }}</span>
        </div>

        <!-- 错误 -->
        <div v-if="interview.error" class="iv-error">
          <Icon name="alert-triangle" :size="14" class="iv-error-icon" />
          <span>{{ interview.error.msg }}</span>
          <button class="iv-error-close" @click="interview.error = null">
            <Icon name="x" :size="12" />
          </button>
        </div>

        <!-- 空状态 -->
        <div v-if="!interview.isOn && interview.rounds.length === 0" class="iv-empty">
          <Icon name="mic" :size="28" class="iv-empty-icon" />
          <p class="iv-empty-title">面试模式</p>
          <p class="iv-empty-desc">
            开启后自动监听{{ isSystemAudio ? '系统音频' : '麦克风' }}，
            {{ isManualMode ? '面试官提问后按 F3 或点击"立即回答"' : '识别到问题后自动作答' }}。
          </p>
          <button class="iv-start-btn" @click="startInterview" :disabled="!hasASRKey">
            {{ hasASRKey ? '开启面试模式' : '请先在设置中配置语音识别 Key' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import { useInterviewStore } from '../stores/interview'
import { api } from '../services/api'
import Icon from './Icon.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()
const interview = useInterviewStore()

// 面板不透明开关（临时，不写回全局透明度设置）
const isOpaque = ref(false)

const isSystemAudio = computed(() => settingsStore.settings.interviewAudioSource !== 'mic')
const isManualMode = computed(() => settingsStore.settings.interviewMode === 'manual' || !isSystemAudio.value)
const modeLabel = computed(() => {
  if (!interview.isOn) return ''
  const src = isSystemAudio.value ? '系统音频' : '麦克风'
  return `${src} · ${isManualMode.value ? '手动' : '自动'}`
})
const hasASRKey = computed(() => !!settingsStore.settings.asrApiKey)
const statusText = computed(() => {
  if (!interview.isOn) return '面试模式未开启'
  if (interview.isAnswering) return '正在作答…'
  return interview.isListening ? '正在聆听…' : '收音已暂停'
})

function startInterview() {
  interview.reset()
  api.startInterview()
}

// 收音中点停止 = 暂停（保留对话）；会话结束后开启会新建会话
function togglePause() {
  if (interview.isListening) {
    api.pauseInterview()
  } else {
    api.resumeInterview()
  }
}

function stopInterview() {
  api.stopInterview()
}

function triggerAnswer() {
  api.triggerInterviewAnswer()
}

// 回答/字幕更新时自动滚到底部
watch(() => interview.rounds.map(r => r.html).join('|'), () => {
  nextTick(() => {
    const el = document.getElementById('interview-content')
    if (el) el.scrollTop = el.scrollHeight
  })
})
watch(() => interview.transcript, () => {
  if (!interview.isAnswering) {
    nextTick(() => {
      const el = document.getElementById('interview-content')
      if (el) el.scrollTop = el.scrollHeight
    })
  }
})
</script>

<style scoped>
/* 与 SolveView 的 .main-interface 一致：占满主区、入场动画、跟随全局透明度 */
.interview-view {
  flex: 1;
  min-height: 0;
  padding: var(--sp-1-5) var(--sp-2) var(--sp-2);
  pointer-events: auto;
  display: flex;
  flex-direction: column;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity 0.35s var(--ease-out), transform 0.35s var(--ease-out);
}
.interview-view.visible {
  opacity: var(--text-alpha, 1);
  transform: translateY(0);
}
/* 不透明模式的变量覆盖见文件末尾的非 scoped 样式块：
   合成色（--surface-elevated 等）在 :root 声明处已完成 var() 替换，
   子元素继承的是替换结果，仅覆盖 *-a 分量无法让其重新求值，
   必须在组件元素上重新声明合成变量本身（按主题给出 alpha=1 的色值） */
/* 内容面板：跟随全局透明度的表面（同 SolveView 的 right-panel）。
   面板自身不滚动，滚动发生在内部的 .iv-content */
.iv-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--surface-elevated);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-subtle);
  box-shadow: none;
  transition: background-color var(--duration-base) ease, border-color var(--duration-base) ease;
}
.iv-status {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-4);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.iv-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border-default);
  flex-shrink: 0;
}
.iv-status-dot.on {
  background: var(--color-success);
  box-shadow: 0 0 6px var(--color-success);
}
.iv-status-dot.pulse {
  animation: ivPulse 1.6s ease-in-out infinite;
}
@keyframes ivPulse {
  0%, 100% { opacity: var(--text-alpha, 1); }
  50% { opacity: calc(var(--text-alpha, 1) * 0.4); }
}
.iv-status-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
}
.iv-mode-tag {
  font-size: var(--text-xs);
  color: var(--text-muted);
  padding: 1px 8px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-full);
}
.iv-status-actions {
  margin-left: auto;
  display: flex;
  gap: var(--sp-2);
}
.iv-action-btn {
  padding: 4px 14px;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-default);
  background: var(--surface-card);
  color: var(--text-primary);
  font-size: var(--text-xs);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.iv-action-btn:hover:not(:disabled) {
  background: var(--surface-card-hover);
  border-color: var(--border-hover);
}
.iv-action-btn.accent {
  background: var(--accent);
  border-color: var(--accent);
  color: white;
}
.iv-action-btn.accent:hover:not(:disabled) {
  background: var(--accent-hover);
}
.iv-action-btn.danger:hover:not(:disabled) {
  color: var(--color-error);
  border-color: var(--color-error);
}
.iv-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.iv-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--sp-3) var(--sp-4);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}
.iv-round {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}
.iv-skills {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sp-1);
  color: var(--text-muted);
}
.iv-skill-chip {
  padding: 1px var(--sp-2);
  font-size: var(--text-xs);
  color: var(--accent);
  background: var(--accent-bg, var(--surface-card));
  border: 1px solid var(--accent-border, var(--border-default));
  border-radius: var(--radius-full);
}
.iv-transcript {
  display: flex;
  align-items: baseline;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}
.iv-transcript-label,
.iv-question-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  flex-shrink: 0;
}
.iv-transcript-text {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  line-height: var(--leading-relaxed);
}
.iv-question {
  padding: var(--sp-2) var(--sp-3);
  background: var(--accent-bg, var(--surface-card));
  border: 1px solid var(--accent-border, var(--border-default));
  border-radius: var(--radius-md);
}
.iv-question-text {
  margin-top: 2px;
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  line-height: var(--leading-relaxed);
}
.iv-error {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--color-error-bg, rgba(220, 38, 38, 0.08));
  border: 1px solid var(--color-error-border, rgba(220, 38, 38, 0.3));
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  color: var(--color-error);
}
.iv-error-icon { flex-shrink: 0; }
.iv-error-close {
  margin-left: auto;
  background: transparent;
  border: none;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 2px;
}
.iv-answer {
  font-size: var(--text-sm);
  min-width: 0;
  overflow-wrap: break-word;
}
/* 长内容撑破 flex 布局的兜底：代码块/URL 强制换行 */
.iv-answer :deep(pre) {
  white-space: pre-wrap;
  word-break: break-all;
}
.iv-empty {
  margin: auto;
  text-align: center;
  max-width: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-2);
}
.iv-empty-icon {
  color: var(--text-muted);
  opacity: 0.6;
}
.iv-empty-title {
  font-size: var(--text-base);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
}
.iv-empty-desc {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  line-height: var(--leading-relaxed);
}
.iv-start-btn {
  margin-top: var(--sp-2);
  padding: 8px 24px;
  border-radius: var(--radius-full);
  border: none;
  background: var(--accent);
  color: white;
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.iv-start-btn:hover:not(:disabled) {
  background: var(--accent-hover);
}
.iv-start-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

<style>
/*
 * 不透明模式 token 覆盖（非 scoped：需要 <html> 上的 data-theme 祖先选择器）。
 *
 * 原理：主题色板变量（--surface-elevated / --text-primary 等）在 :root 处
 * 以 rgba(R,G,B,var(--xxx-a)) 形式声明，var() 替换发生在声明元素上，
 * 子元素继承到的已是替换完毕的半透明色。因此必须在这里针对组件元素
 * 重新声明合成变量本身（alpha 拉满 = 1），主题差异用 data-theme 区分。
 * 只影响面试视图内部，其他视图与全局透明度设置不受影响。
 */
.interview-view.opaque {
  --text-alpha: 1;
  --surface-elevated: rgba(17, 26, 40, 1);
  --surface-card: rgba(21, 32, 49, 1);
  --surface-card-hover: rgba(28, 42, 63, 1);
  --text-primary: rgba(234, 243, 252, 1);
  --text-secondary: rgba(152, 170, 194, 1);
  --text-tertiary: rgba(152, 170, 194, 0.55);
  --text-muted: rgba(98, 116, 142, 1);
  --accent-text: rgba(56, 189, 248, 1);
  --accent-muted: rgba(14, 165, 233, 0.14);
  --accent-border: rgba(14, 165, 233, 0.30);
  --accent-secondary-muted: rgba(45, 212, 191, 0.12);
  --accent-secondary-border: rgba(45, 212, 191, 0.28);
  --code-bg: rgba(12, 19, 30, 1);
  --code-header-bg: rgba(18, 27, 42, 1);
}
[data-theme="light"] .interview-view.opaque {
  --text-alpha: 1;
  --surface-elevated: rgba(255, 255, 255, 1);
  --surface-card: rgba(235, 245, 253, 1);
  --surface-card-hover: rgba(222, 237, 251, 1);
  --text-primary: rgba(12, 28, 52, 1);
  --text-secondary: rgba(62, 84, 112, 1);
  --text-tertiary: rgba(62, 84, 112, 0.60);
  --text-muted: rgba(128, 150, 176, 1);
  --accent-text: rgba(2, 132, 199, 1);
  --accent-muted: rgba(2, 132, 199, 0.08);
  --accent-border: rgba(2, 132, 199, 0.22);
  --accent-secondary-muted: rgba(13, 148, 136, 0.07);
  --accent-secondary-border: rgba(13, 148, 136, 0.18);
  --code-bg: rgba(238, 246, 253, 1);
  --code-header-bg: rgba(228, 240, 251, 1);
}
</style>
