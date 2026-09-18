<template>
  <div class="top-bar-wrapper" style="--wails-draggable: drag">
    <div class="top-bar">
      <div class="bar-left">
        <div class="logo">Q</div>

        <!-- 主操作：截图 / 搜题 -->
        <button
          class="bar-btn primary"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'screenshot', 'alert-flash': alertAction === 'screenshot' }"
          data-toolbar-action="screenshot"
          title="截图"
          @click="triggerAction('screenshot')"
        >
          <Icon name="camera" :size="16" />
          <span v-if="screenshotCount > 0" class="badge">{{ screenshotCount }}</span>
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('screenshot') }"></span>
        </button>

        <button
          class="bar-btn primary"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'send', 'alert-flash': alertAction === 'send' }"
          data-toolbar-action="send"
          title="搜题"
          @click="triggerAction('send')"
        >
          <Icon v-if="solution.isLoading" name="loader" :size="15" spinning class="send-loading" />
          <Icon v-else name="send" :size="16" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('send') }"></span>
        </button>

        <div class="bar-divider" />

        <!-- 内容滚动（按住/悬浮保持连续触发） -->
        <button
          class="bar-btn"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'scroll_up' }"
          data-toolbar-action="scroll_up"
          title="上滚内容"
          @pointerdown.prevent="startHold('scroll_up')"
          @pointerup="stopHold"
          @pointerleave="stopHold"
        >
          <Icon name="arrow-up" :size="14" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('scroll_up') }"></span>
        </button>

        <button
          class="bar-btn"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'scroll_down' }"
          data-toolbar-action="scroll_down"
          title="下滚内容"
          @pointerdown.prevent="startHold('scroll_down')"
          @pointerup="stopHold"
          @pointerleave="stopHold"
        >
          <Icon name="arrow-down" :size="14" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('scroll_down') }"></span>
        </button>

        <div class="bar-divider" />

        <!-- 折叠 / 展开 -->
        <button
          class="bar-btn"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'toggle-expand' }"
          data-toolbar-action="toggle-expand"
          :title="ui.isCollapsed ? '展开面板' : '折叠为工具条'"
          @click="toggleExpand"
        >
          <Icon :name="ui.isCollapsed ? 'chevron-up' : 'chevron-down'" :size="15" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('toggle-expand') }"></span>
        </button>
      </div>

      <!-- 关闭按钮始终显示；设置仅展开态可见 -->
      <div class="bar-right">
        <button class="bar-btn" v-show="!ui.isCollapsed" @click="$emit('openSettings')" title="设置">
          <Icon name="settings" :size="15" />
        </button>

        <button class="bar-btn bar-btn-quit" @click="ui.quit" title="退出">
          <Icon name="power" :size="15" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSolutionStore } from '../stores/solution'
import { useSettingsStore } from '../stores/settings'
import { collapseToolbar, expandToolbar } from '../services/toolbar'
import { on } from '../services/events'
import { api } from '../services/api'
import { WindowGetPosition } from '../../wailsjs/runtime/runtime'
import Icon from './Icon.vue'

defineEmits(['openSettings'])

const ui = useUIStore()
const solution = useSolutionStore()
const settingsStore = useSettingsStore()

// ==================== 动作触发 ====================

function triggerAction(action) {
  switch (action) {
    case 'screenshot':
      api.triggerScreenshot()
      break
    case 'send':
      api.triggerSend()
      // 搜题后默认展开面板
      if (ui.isCollapsed) expandToolbar()
      break
    case 'scroll_up':
      api.scrollContent('up')
      break
    case 'scroll_down':
      api.scrollContent('down')
      break
  }
}

function toggleExpand() {
  if (ui.isCollapsed) {
    expandToolbar()
  } else {
    collapseToolbar()
  }
}

// ==================== 滚动按钮连续触发 ====================

const SCROLL_ACTIONS = new Set(['scroll_up', 'scroll_down'])
const HOLD_INTERVAL = 150        // 非穿透：按住重复间隔
const HOVER_REPEAT_INTERVAL = 200 // 穿透：悬浮保持重复间隔

let holdTimer = null         // 非穿透按住重复
let hoverRepeatTimer = null  // 穿透悬浮保持重复

function startHold(action) {
  if (!SCROLL_ACTIONS.has(action)) return
  stopHold()
  triggerAction(action)
  holdTimer = setInterval(() => triggerAction(action), HOLD_INTERVAL)
}

function stopHold() {
  if (holdTimer) {
    clearInterval(holdTimer)
    holdTimer = null
  }
}

function startHoverRepeat(action) {
  stopHoverRepeat()
  hoverRepeatTimer = setInterval(() => {
    // 光标事件断流（如窗口被隐藏）时自动停止
    if (performance.now() - lastEventAt > 500) {
      stopHoverRepeat()
      return
    }
    triggerAction(action)
  }, HOVER_REPEAT_INTERVAL)
}

function stopHoverRepeat() {
  if (hoverRepeatTimer) {
    clearInterval(hoverRepeatTimer)
    hoverRepeatTimer = null
  }
}

// ==================== 截图徽标 ====================

const screenshotCount = ref(0)

on('screenshot-taken', (_, count) => { screenshotCount.value = count || 0 })
on('screenshot-removed', (_, count) => { screenshotCount.value = count || 0 })
on('screenshots-cleared', () => { screenshotCount.value = 0 })

// 发送后清空截图缓冲
on('start-solving', () => { screenshotCount.value = 0 })

// ==================== 按钮反馈（替代胶囊态 toast 文字） ====================

const alertAction = ref(null)
let alertTimer = null

function flashAlert(action) {
  alertAction.value = action
  if (alertTimer) clearTimeout(alertTimer)
  alertTimer = setTimeout(() => { alertAction.value = null }, 2000)
}

on('toast', (msg) => {
  if (!ui.isCollapsed || !msg) return
  // 截图相关错误闪截图按钮，其余闪搜题按钮
  flashAlert(String(msg).includes('截图') ? 'screenshot' : 'send')
})

on('solution-error', () => {
  if (ui.isCollapsed) flashAlert('send')
})

// ==================== 穿透模式：悬浮自动触发 ====================

// 触发时长可在设置中调节（500ms 下限保护）
const HOVER_TRIGGER_MS = computed(() => Math.max(500, settingsStore.settings.hoverTriggerMs || 3000))

const progressMap = ref({}) // action -> 0~1
let currentAction = null
let accumulated = 0
let lastEventAt = 0
let firedAction = null // 已触发后锁存，命中变化前不再连发

function resetHoverState() {
  currentAction = null
  accumulated = 0
  lastEventAt = 0
  firedAction = null
  ui.hoverToolbarAction = null
  progressMap.value = {}
  stopHoverRepeat()
}

function progressOf(action) {
  return progressMap.value[action] || 0
}

function onCursorPosition(payload) {
  if (!ui.isClickThrough || !payload) return

  const dpr = window.devicePixelRatio || 1
  const cssX = payload.x / dpr
  const cssY = payload.y / dpr

  // 只关心工具条区域，越界即视为未命中
  if (cssY > 60 || cssX < 0 || cssX >= window.innerWidth) {
    if (currentAction !== null || firedAction !== null) resetHoverState()
    return
  }

  const el = document.elementFromPoint(cssX, cssY)
  const action = el?.closest('[data-toolbar-action]')?.dataset.toolbarAction || null

  const now = performance.now()

  // 命中目标变化：复位计时并重新武装
  if (action !== currentAction) {
    stopHoverRepeat()
    firedAction = null
    currentAction = action
    accumulated = 0
    ui.hoverToolbarAction = action
    progressMap.value = {}
  }

  if (!action || firedAction) {
    lastEventAt = now
    return
  }

  if (lastEventAt) {
    accumulated += Math.min(now - lastEventAt, 250) // 防单次大跳（事件丢失/断流）
  }
  lastEventAt = now

  progressMap.value = { [action]: Math.min(1, accumulated / HOVER_TRIGGER_MS.value) }

  if (accumulated >= HOVER_TRIGGER_MS.value) {
    firedAction = action
    progressMap.value = {}
    triggerAction(action)
    // 滚动按钮：悬浮保持连续触发，移开才停止
    if (SCROLL_ACTIONS.has(action)) startHoverRepeat(action)
  }
}

watch(() => ui.isClickThrough, () => resetHoverState())

// ==================== 折叠态下拖动后的位置持久化 ====================

// Wails 原生拖动结束后 JS 收不到可靠事件，用 1s 轮询对比窗口位置兜底
let lastSavedPos = null
let posTimer = null

async function pollPosition() {
  if (!ui.isCollapsed) return
  try {
    const pos = await WindowGetPosition()
    if (!lastSavedPos) {
      lastSavedPos = pos
      return
    }
    if (pos.x !== lastSavedPos.x || pos.y !== lastSavedPos.y) {
      lastSavedPos = pos
      api.updateToolbarPosition(pos.x, pos.y)
    }
  } catch (_) { /* 忽略轮询错误 */ }
}

watch(() => ui.isCollapsed, (collapsed) => {
  if (collapsed) {
    lastSavedPos = null
    WindowGetPosition().then(pos => { lastSavedPos = pos }).catch(() => {})
  } else {
    resetHoverState()
  }
})

onMounted(() => {
  on('cursor-position', onCursorPosition)
  posTimer = setInterval(pollPosition, 1000)
})

onUnmounted(() => {
  if (posTimer) clearInterval(posTimer)
  if (alertTimer) clearTimeout(alertTimer)
  stopHold()
  stopHoverRepeat()
})
</script>

<style scoped>
.top-bar-wrapper {
  padding: var(--sp-1) var(--sp-1) 0;
  pointer-events: auto;
  display: flex;
  user-select: none;
  width: 100%;
  position: relative;
  z-index: 50;
  flex-shrink: 0;
}
.top-bar {
  background: var(--surface-elevated);
  backdrop-filter: blur(24px);
  height: 38px;
  padding: 0 var(--sp-1-5);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-1);
  pointer-events: auto;
  box-shadow: var(--shadow-md), 0 0 0 1px var(--border-subtle);
  border: 1px solid var(--border-default);
  transition: border-color var(--duration-base) ease;
  white-space: nowrap;
  width: 100%;
}
.top-bar:hover { border-color: var(--border-hover); }
.bar-left {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
}
.logo {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-gradient);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 800;
  color: white;
  flex-shrink: 0;
  letter-spacing: -0.5px;
  margin-right: var(--sp-0-5);
}
.bar-right {
  display: flex;
  align-items: center;
  gap: 2px;
}
.bar-divider {
  width: 1px;
  height: 14px;
  background: var(--border-default);
  margin: 0 var(--sp-0-5);
  flex-shrink: 0;
}
.bar-btn {
  position: relative;
  width: 28px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-full);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  flex-shrink: 0;
  overflow: visible;
  /* 按钮本身可点击；其余区域（logo/间隙/空白处）随顶栏拖动 */
  --wails-draggable: no-drag;
}
.bar-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
/* 主操作按钮：更大更醒目 */
.bar-btn.primary {
  width: 36px;
  height: 28px;
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-primary);
}
.bar-btn.primary:hover,
.bar-btn.primary.sim-hover {
  background: var(--surface-card-hover);
  border-color: var(--accent-border);
  color: var(--accent);
}
.bar-btn-quit:hover { color: var(--color-error); }

/* 截图数量徽标 */
.badge {
  position: absolute;
  top: -5px;
  right: -5px;
  min-width: 13px;
  height: 13px;
  padding: 0 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent);
  color: white;
  font-size: 9px;
  font-weight: var(--weight-bold);
  border-radius: var(--radius-full);
  line-height: 1;
  pointer-events: none;
  box-shadow: 0 0 0 2px var(--surface-elevated);
}

/* 错误反馈（替代胶囊态 toast 文字） */
.alert-flash {
  animation: alertShake 0.4s ease;
  color: var(--color-error) !important;
  border-color: var(--color-error) !important;
}
@keyframes alertShake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-2px); }
  75% { transform: translateX(2px); }
}

/* 搜题中：spinner 用主题色并降低不透明度，视觉更柔和 */
.send-loading {
  color: var(--accent);
  opacity: 0.85;
}

/* 悬浮触发进度条（穿透模式下模拟 hover 的倒计时可视化） */
.tb-progress {
  position: absolute;
  left: 22%;
  right: 22%;
  bottom: 1px;
  height: 2px;
  border-radius: 1px;
  background: var(--accent);
  transform-origin: left center;
  transform: scaleX(var(--hover-progress, 0));
  opacity: 0.9;
  pointer-events: none;
}
</style>
