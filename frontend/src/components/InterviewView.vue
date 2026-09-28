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
/* 不透明模式：视图作用域内把透明度分量变量拉满（透明度=0），
   覆盖全局 applyTransparency 的值，不影响其他视图与全局设置 */
.interview-view.opaque {
  --text-alpha: 1;
  --surface-elevated-a: 1;
  --surface-card-a: 1;
  --surface-card-hover-a: 1;
  --code-bg-a: 1;
  --code-header-a: 1;
}
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
