<template>
  <div class="main-interface interview-view" :class="{ visible: ui.mainVisible }">
    <!-- 状态栏 -->
    <div class="iv-panel">
      <span class="iv-status-dot" :class="{ on: interview.isOn, pulse: interview.isOn && !interview.isAnswering }"></span>
      <span class="iv-status-text">
        {{ interview.isOn ? (interview.isAnswering ? '正在作答…' : '正在聆听…') : '面试模式未开启' }}
      </span>
      <span class="iv-mode-tag" v-if="interview.isOn">{{ modeLabel }}</span>
      <div class="iv-status-actions">
        <button
          v-if="interview.isOn"
          class="iv-action-btn accent"
          @click="triggerAnswer"
          :disabled="interview.isAnswering"
          :title="isManualMode ? '用当前字幕立即回答（F3）' : '不等停顿，立即回答当前字幕'"
        >立即回答</button>
        <button class="iv-action-btn" @click="toggleInterview">
          {{ interview.isOn ? '停止' : '开启面试模式' }}
        </button>
      </div>
    </div>

    <!-- 内容区 -->
    <div class="iv-content" id="interview-content">
      <!-- 实时字幕 -->
      <div v-if="interview.isOn && interview.transcript" class="iv-transcript">
        <span class="iv-transcript-label">字幕</span>
        <span class="iv-transcript-text">{{ interview.transcript }}</span>
      </div>

      <!-- 问题 -->
      <div v-if="interview.question" class="iv-question">
        <span class="iv-question-label">问题</span>
        <div class="iv-question-text">{{ interview.question }}</div>
      </div>

      <!-- 错误 -->
      <div v-if="interview.error" class="iv-error">
        <Icon name="alert-triangle" :size="14" class="iv-error-icon" />
        <span>{{ interview.error.msg }}</span>
        <button class="iv-error-close" @click="interview.error = null">
          <Icon name="x" :size="12" />
        </button>
      </div>

      <!-- 答案 -->
      <div v-if="interview.answerHtml" class="iv-answer md-body">
        <div class="ai-response" v-html="interview.answerHtml"></div>
        <span v-if="interview.isAnswering" class="stream-cursor">▍</span>
      </div>

      <!-- 空状态 -->
      <div v-if="!interview.isOn && !interview.answerHtml" class="iv-empty">
        <Icon name="mic" :size="28" class="iv-empty-icon" />
        <p class="iv-empty-title">面试模式</p>
        <p class="iv-empty-desc">
          开启后自动监听{{ isSystemAudio ? '系统音频' : '麦克风' }}，
          {{ isManualMode ? '面试官提问后按 F3 或点击"立即回答"' : '识别到问题后自动作答' }}。
        </p>
        <button class="iv-start-btn" @click="toggleInterview" :disabled="!hasASRKey">
          {{ hasASRKey ? '开启面试模式' : '请先在设置中配置语音识别 Key' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, watch, nextTick } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import { useInterviewStore } from '../stores/interview'
import { api } from '../services/api'
import Icon from './Icon.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()
const interview = useInterviewStore()

const isSystemAudio = computed(() => settingsStore.settings.interviewAudioSource !== 'mic')
const isManualMode = computed(() => settingsStore.settings.interviewMode === 'manual' || !isSystemAudio.value)
const modeLabel = computed(() => {
  if (!interview.isOn) return ''
  const src = isSystemAudio.value ? '系统音频' : '麦克风'
  return `${src} · ${isManualMode.value ? '手动' : '自动'}`
})
const hasASRKey = computed(() => !!settingsStore.settings.asrApiKey)

function toggleInterview() {
  if (interview.isOn) {
    api.stopInterview()
    interview.setState(false)
  } else {
    interview.reset()
    api.startInterview()
  }
}

function triggerAnswer() {
  api.triggerInterviewAnswer()
}

// 回答/字幕更新时自动滚到底部
watch(() => interview.answerHtml, () => {
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
  gap: 0;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity 0.35s var(--ease-out), transform 0.35s var(--ease-out);
}
.interview-view.visible {
  opacity: var(--text-alpha, 1);
  transform: translateY(0);
}
/* 内容面板：跟随全局透明度的表面（同 SolveView 的 right-panel） */
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
.iv-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.iv-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--sp-3) var(--sp-4);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
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
