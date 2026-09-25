<template>
  <div class="top-bar-wrapper" style="--wails-draggable: drag">
    <div class="top-bar">
      <div class="bar-left">
        <div class="logo" :class="{ 'logo-stealth': ui.isCollapsed && ui.isStealthMode }">Q</div>

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
          :title="solution.isSolving ? '停止搜题' : '搜题'"
          @click="triggerAction('send')"
        >
          <Icon v-if="solution.isSolving" name="square" :size="15" class="send-stop" />
          <Icon v-else name="send" :size="16" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('send') }"></span>
        </button>

        <!-- 内容滚动（仅展开态；按住/悬浮保持连续触发） -->
        <template v-if="!ui.isCollapsed">
          <div class="bar-divider" />

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
        </template>

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

      <!-- 右侧：隐身/状态灯/主题/设置仅展开态显示；键盘屏蔽与退出始终显示 -->
      <div class="bar-right">
        <template v-if="!ui.isCollapsed">
          <button
            class="bar-btn stealth-btn"
            :class="{ 'stealth-on': ui.isStealthMode, 'sim-hover': ui.hoverToolbarAction === 'stealth' }"
            data-toolbar-action="stealth"
            :title="ui.isStealthMode ? '隐身模式已开启（点击关闭）' : '隐身模式已关闭（点击开启）'"
            @click="triggerAction('stealth')"
          >
            <Icon :name="ui.isStealthMode ? 'eye-off' : 'eye'" :size="15" />
            <span class="tb-progress" :style="{ '--hover-progress': progressOf('stealth') }"></span>
          </button>

          <div class="bar-divider" />

          <button class="bar-btn" ref="statusBtnRef" @click="toggleStatusPanel" :title="'状态: ' + settingsStore.statusText">
            <span class="status-dot" :class="statusClass"></span>
          </button>

          <ThemeToggle />

          <button class="bar-btn" @click="$emit('openSettings')" title="设置">
            <Icon name="settings" :size="15" />
          </button>
        </template>

        <button
          class="bar-btn shield-btn"
          :class="{ 'shield-on': ui.isShieldOn, 'sim-hover': ui.hoverToolbarAction === 'shield' }"
          data-toolbar-action="shield"
          :title="ui.isShieldOn ? '快捷键屏蔽已开启（F6 关闭）' : '快捷键屏蔽已关闭（F6 开启）'"
          @click="triggerAction('shield')"
        >
          <Icon name="keyboard" :size="15" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('shield') }"></span>
        </button>

        <button
          class="bar-btn review-btn"
          :class="{ 'review-on': ui.isReviewOn, 'sim-hover': ui.hoverToolbarAction === 'review' }"
          data-toolbar-action="review"
          :title="ui.isReviewOn ? '存储复盘已开启（F4 关闭）' : '存储复盘已关闭（F4 开启）'"
          @click="triggerAction('review')"
        >
          <Icon name="file" :size="15" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('review') }"></span>
        </button>

        <button
          class="bar-btn bar-btn-quit"
          :class="{ 'sim-hover': ui.hoverToolbarAction === 'quit' }"
          data-toolbar-action="quit"
          @click="ui.quit"
          title="退出"
        >
          <Icon name="power" :size="15" />
          <span class="tb-progress" :style="{ '--hover-progress': progressOf('quit') }"></span>
        </button>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="panel-fade">
      <div class="status-panel" v-if="showStatusPanel" :style="panelStyle" @click.stop>
        <div class="sp-header">
          <span class="sp-title">运行状态</span>
          <button class="sp-close" @click="showStatusPanel = false">
            <Icon name="x" :size="14" />
          </button>
        </div>
        <div class="sp-body">
          <div class="sp-row">
            <span class="sp-label">API Key</span>
            <span class="sp-value" :class="settingsStore.settings.apiKey ? 'ok' : 'warn'">
              {{ settingsStore.settings.apiKey ? '已配置' : '未配置' }}
            </span>
          </div>
          <div class="sp-row">
            <span class="sp-label">Base URL</span>
            <span class="sp-value model">{{ settingsStore.settings.baseURL || 'https://api.openai.com/v1' }}</span>
          </div>
          <div class="sp-row">
            <span class="sp-label">使用模型</span>
            <span class="sp-value model">{{ settingsStore.settings.model || '未设置' }}</span>
          </div>
          <div class="sp-row">
            <span class="sp-label">隐身模式</span>
            <span class="sp-value" :class="ui.isStealthMode ? 'ok' : 'err'">{{ ui.isStealthMode ? '已开启' : '已关闭' }}</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSolutionStore } from '../stores/solution'
import { useSettingsStore } from '../stores/settings'
import { collapseToolbar, expandToolbar } from '../services/toolbar'
import { on } from '../services/events'
import { api } from '../services/api'
import { WindowGetPosition } from '../../wailsjs/runtime/runtime'
import Icon from './Icon.vue'
import ThemeToggle from './ThemeToggle.vue'

defineEmits(['openSettings'])

const ui = useUIStore()
const solution = useSolutionStore()
const settingsStore = useSettingsStore()

// ==================== 状态指示灯 ====================

const statusClass = computed(() => {
  const text = settingsStore.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成' || text.includes('思考') || text.includes('复制')) return 'connected'
  if (text.includes('未配置')) return 'unconfigured'
  if (text.includes('无效') || text.includes('Key')) return 'invalid-key'
  if (text.includes('失败') || text.includes('出错')) return 'disconnected'
  return 'unconfigured'
})

const showStatusPanel = ref(false)
const statusBtnRef = ref(null)
const panelStyle = reactive({ top: '0px', left: '0px' })

function toggleStatusPanel() {
  if (showStatusPanel.value) {
    showStatusPanel.value = false
    return
  }
  if (statusBtnRef.value) {
    const rect = statusBtnRef.value.getBoundingClientRect()
    panelStyle.top = `${rect.bottom + 8}px`
    const pw = 280
    let left = rect.left + rect.width / 2 - pw / 2
    if (left + pw > window.innerWidth - 10) left = window.innerWidth - pw - 10
    if (left < 10) left = 10
    panelStyle.left = `${left}px`
    showStatusPanel.value = true
  }
}

function handleClickOutside(e) {
  if (showStatusPanel.value && statusBtnRef.value && !statusBtnRef.value.contains(e.target)) {
    showStatusPanel.value = false
  }
}

// ==================== 动作触发 ====================

function triggerAction(action) {
  switch (action) {
    case 'screenshot':
      api.triggerScreenshot()
      break
    case 'send':
      // 搜题/流式输出进行中：点击/悬浮 = 停止；空闲 = 发起新搜题
      if (solution.isSolving) {
        api.cancelTask()
        ui.showToast('已停止搜题', 'success')
      } else {
        api.triggerSend()
        // 搜题后默认展开面板
        if (ui.isCollapsed) expandToolbar()
      }
      break
    case 'scroll_up':
      api.scrollContent('up')
      break
    case 'scroll_down':
      api.scrollContent('down')
      break
    case 'stealth':
      api.toggleVisibility()
      break
    case 'shield':
      api.toggleShield()
      break
    case 'review':
      api.toggleReview()
      break
    case 'toggle-expand':
      toggleExpand()
      break
    case 'quit':
      ui.quit()
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
let requireLeave = false // 布局切换（折叠/展开）后要求鼠标先移开再重新武装

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
  let action = el?.closest('[data-toolbar-action]')?.dataset.toolbarAction || null

  const now = performance.now()

  // 折叠/展开瞬间光标可能正好停在某按钮上（如关闭按钮），
  // 强制要求先移出按钮区域一次，才重新开始悬浮计时，防止误触
  if (requireLeave) {
    if (action === null) {
      requireLeave = false
    } else {
      if (currentAction !== null) resetHoverState()
      lastEventAt = now
      return
    }
  }

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

// 键盘屏蔽状态变化（F6 或点击按钮）时同步图标
on('shield-state', (on) => {
  ui.isShieldOn = !!on
  if (on) {
    ui.showToast('快捷键屏蔽已开启：软件快捷键及其组合键对外不可见（F6 恢复）', 'warning', 3000)
  } else {
    ui.showToast('快捷键屏蔽已关闭', 'success')
  }
})

// 存储复盘状态变化（F4 或点击按钮）时同步图标
on('review-state', (on) => {
  ui.isReviewOn = !!on
  if (on) {
    ui.showToast('存储复盘已开启：截图与解答将存入文档/Q-Solver/复盘（F4 关闭）', 'success', 3000)
  } else {
    ui.showToast('存储复盘已关闭', 'info')
  }
})

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
  }
  // 布局切换后按钮位置全变：光标停留处可能是别的按钮（如关闭），
  // 要求先移出按钮区域一次再重新武装悬浮计时，防止折叠瞬间误触
  requireLeave = true
  resetHoverState()
})

onMounted(() => {
  on('cursor-position', onCursorPosition)
  posTimer = setInterval(pollPosition, 1000)
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  if (posTimer) clearInterval(posTimer)
  if (alertTimer) clearTimeout(alertTimer)
  stopHold()
  stopHoverRepeat()
  document.removeEventListener('click', handleClickOutside)
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
  box-shadow: 0 0 0 1px var(--border-subtle);
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
  transition: box-shadow var(--duration-fast) ease, outline-color var(--duration-fast) ease;
}

/* 折叠态隐身标识：Q 图标外圈虚线光环（主题色） */
.logo-stealth {
  outline: 1.5px dashed var(--accent);
  outline-offset: 2px;
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

/* 隐身模式标识：开启时虚线描边 */
.stealth-btn {
  border: 1px dashed transparent;
}
.stealth-btn.stealth-on {
  border-color: var(--accent);
  color: var(--accent);
  background: transparent;
}
.stealth-btn.stealth-on:hover,
.stealth-btn.stealth-on.sim-hover {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--surface-card-hover);
}

/* 键盘屏蔽模式：开启时红色警示 */
.shield-btn.shield-on {
  color: var(--color-error);
  animation: shieldPulse 1.2s ease-in-out infinite;
}
@keyframes shieldPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}

/* 存储复盘模式：开启时主题色高亮 */
.review-btn.review-on {
  color: var(--accent);
  border-color: var(--accent);
}

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

/* 搜题中：停止按钮用红色醒目提示 */
.send-stop {
  color: var(--color-error);
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

/* ---- 状态指示灯与状态面板 ---- */
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: all var(--duration-fast) ease;
}
.status-dot.connected { background: var(--color-success); box-shadow: 0 0 6px var(--color-success); }
.status-dot.unconfigured { background: var(--color-warning); box-shadow: 0 0 6px var(--color-warning); }
.status-dot.invalid-key,
.status-dot.disconnected { background: var(--color-error); box-shadow: 0 0 6px var(--color-error); }
.status-panel {
  position: fixed;
  width: 280px;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  backdrop-filter: blur(24px);
  z-index: var(--z-tooltip);
  overflow: hidden;
  pointer-events: auto;
  transform-origin: top center;
}
.sp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--border-subtle);
}
.sp-title { font-size: var(--text-sm); font-weight: var(--weight-bold); color: var(--text-primary); }
.sp-close {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.sp-close:hover { background: var(--surface-card-hover); color: var(--text-primary); }
.sp-body { padding: var(--sp-2) var(--sp-4) var(--sp-3); }
.sp-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-2) 0;
  border-bottom: 1px solid var(--border-subtle);
}
.sp-row:last-child { border-bottom: none; }
.sp-label { font-size: var(--text-xs); color: var(--text-muted); }
.sp-value {
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  font-family: var(--font-mono);
  white-space: nowrap;
  color: var(--text-primary);
}
.sp-value.ok { color: var(--color-success); }
.sp-value.warn { color: var(--color-warning); }
.sp-value.err { color: var(--color-error); }
.sp-value.model { max-width: 160px; overflow: hidden; text-overflow: ellipsis; }
.panel-fade-enter-active { transition: all 0.2s var(--ease-out); }
.panel-fade-leave-active { transition: all 0.15s ease-in; pointer-events: none; }
.panel-fade-enter-from, .panel-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.96);
}
</style>
