<template>
  <TopBar @openSettings="settingsStore.openSettings" />

  <WelcomeView
    v-if="!ui.hasStarted && solution.history.length === 0 && !ui.showInterview"
    v-show="!ui.isCollapsed"
  />
  <SolveView
    v-if="(ui.hasStarted || solution.history.length > 0) && !ui.showInterview"
    v-show="!ui.isCollapsed"
  />
  <InterviewView
    v-if="ui.showInterview"
    v-show="!ui.isCollapsed"
  />

  <ScreenshotDock v-show="!ui.isCollapsed" />
  <SettingsModal />

  <div id="toast-container">
    <div v-for="(t, i) in ui.toasts" :key="t.id || i" class="toast" :class="[t.type, { show: t.show }]">
      <Icon :name="toastIcon(t.type)" :size="14" class="toast-icon" />
      <span>{{ t.text }}</span>
    </div>
  </div>

  <ResizeHandle v-show="!ui.isCollapsed" />
</template>

<script setup>
import { onMounted, watch } from 'vue'
import TopBar from './components/TopBar.vue'
import WelcomeView from './components/WelcomeView.vue'
import SolveView from './components/SolveView.vue'
import InterviewView from './components/InterviewView.vue'
import ScreenshotDock from './components/ScreenshotDock.vue'
import SettingsModal from './components/SettingsModal.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import Icon from './components/Icon.vue'

import { useUIStore } from './stores/ui'
import { useSettingsStore } from './stores/settings'
import { useSolutionStore } from './stores/solution'
import { useInterviewStore } from './stores/interview'
import { on } from './services/events'
import { api } from './services/api'
import { applyModelCache } from './utils/modelCapabilities'
import { expandToolbar, collapseToolbar } from './services/toolbar'
import { initCodeBlockInteractions } from './utils/markdown-latex'

const ui = useUIStore()
const settingsStore = useSettingsStore()
const solution = useSolutionStore()
const interview = useInterviewStore()

// 打开设置时若处于折叠态，自动展开（折叠态下无法操作设置弹窗）
watch(() => ui.showSettings, (open) => {
  if (open && ui.isCollapsed) expandToolbar()
})

function toastIcon(type) {
  if (type === 'error') return 'x-circle'
  if (type === 'success') return 'check-circle'
  if (type === 'warning') return 'alert-triangle'
  return 'info'
}

onMounted(() => {
  initCodeBlockInteractions()

  api.getInitStatus().then(s => { ui.initStatus = s })
  api.getShieldState().then(v => { ui.isShieldOn = !!v })
  on('init-status', (s) => { ui.initStatus = s })

  // 异步合并最新模型能力表缓存（后端启动时已后台拉取；无缓存/失败静默跳过）
  api.getModelCapabilitiesCache().then(applyModelCache).catch(() => {})

  settingsStore.loadSettings().then(() => {
    settingsStore.resetStatus()
    // 恢复折叠工具条 UI（窗口尺寸 Go 侧 Startup 已缩好）
    if (settingsStore.settings.toolbarCollapsed) {
      ui.isCollapsed = true
    }
  })

  on('key-recorded', (data) => {
    if (data && data.action) {
      if (settingsStore.tempShortcuts[data.action]) {
        settingsStore.tempShortcuts[data.action].keyName = data.keyName
        settingsStore.tempShortcuts[data.action].vkCode = data.comboID
      } else {
        settingsStore.tempShortcuts[data.action] = { keyName: data.keyName, vkCode: data.comboID }
      }
      if (settingsStore.recordingAction === data.action) {
        settingsStore.recordingText = data.keyName
      }
    }
  })

  on('shortcut-error', async (msg) => {
    ui.showToast(msg, 'error', 2000)
    const targetAction = settingsStore.recordingAction
    settingsStore.recordingAction = null
    settingsStore.recordingText = ''
    api.stopRecordingKey()
    if (!targetAction) return
    try {
      if (settingsStore.shortcuts[targetAction]?.keyName) {
        settingsStore.tempShortcuts[targetAction] = JSON.parse(JSON.stringify(settingsStore.shortcuts[targetAction]))
      } else {
        delete settingsStore.tempShortcuts[targetAction]
      }
    } catch (e) {
      console.error('回滚快捷键配置失败', e)
    }
  })

  on('shortcut-saved', (action) => {
    if (settingsStore.recordingAction === action) {
      settingsStore.recordingAction = null
      ui.showToast('快捷键已保存', 'success')
    }
  })

  on('user-message', (screenshot) => solution.setUserScreenshot(screenshot))

  on('start-solving', () => {
    // 面试视图打开时发起新搜题：切回解题视图（面试会话继续在后台运行）
    ui.showInterview = false
    proceedWithSolve()
  })

  function proceedWithSolve() {
    solution.errorState.show = false
    solution.isSolving = true
    ui.flash('solve')
    settingsStore.statusText = '正在思考...'
    settingsStore.statusIcon = '...'
    ui.mainVisible = true
    ui.hasStarted = true
    const s = settingsStore.settings
    if (s.keepContext && solution.history.length > 0 && solution.activeHistoryIndex === 0) {
      solution.isLoading = false
      solution.isAppending = true
      setTimeout(() => {
        const el = document.getElementById('content')
        if (el) el.scrollTop = el.scrollHeight
      }, 50)
    } else {
      solution.isLoading = true
      solution.isAppending = false
    }
  }

  on('toggle-visibility', (isVisibleToCapture) => {
    ui.flash('toggle')
    ui.isStealthMode = isVisibleToCapture
    ui.showToast(isVisibleToCapture ? '隐身模式已开启' : '隐身模式已关闭', isVisibleToCapture ? 'info' : 'success')
  })

  on('solution', (data) => {
    settingsStore.statusText = '解题完成'
    settingsStore.statusIcon = '✓'
    solution.handleSolution(data)
  })

  on('copy-code', () => {
    const old = settingsStore.statusText
    settingsStore.statusText = '已复制'
    setTimeout(() => (settingsStore.statusText = old), 2000)
  })

  on('click-through-state', (enabled) => {
    ui.isClickThrough = enabled
    const el = document.getElementById('main-interface')
    if (el) el.style.pointerEvents = enabled ? 'none' : 'auto'
  })

  // 连续滚动（快捷键长按系统自动重复 / 工具条长按）：用瞬时滚动，
  // smooth 动画在 30Hz 的重复触发下会不断重启动画导致几乎滚不动；
  // 连续触发时步长渐增，长按越久滚越快
  let lastScrollAt = 0
  let scrollStep = 60
  on('scroll-content', (direction) => {
    const el = document.getElementById('content')
    if (!el) return
    const now = Date.now()
    if (now - lastScrollAt < 250) {
      scrollStep = Math.min(scrollStep + 30, 240)
    } else {
      scrollStep = 60
    }
    lastScrollAt = now
    el.scrollBy({ top: direction === 'up' ? -scrollStep : scrollStep, behavior: 'auto' })
  })

  // 快捷键调节全局透明度（与设置页滑块同一变量，静默持久化）
  on('opacity-adjust', (direction) => {
    settingsStore.adjustTransparency(direction)
  })

  on('solution-stream-start', () => {
    ui.hasStarted = true
    solution.handleStreamStart(settingsStore.settings.keepContext)
  })

  on('solution-stream-chunk', (token) => solution.handleStreamChunk(token))
  on('solution-stream-thinking', (token) => solution.handleThinkingChunk(token))

  on('solution-error', (rawErrMsg) => {
    // 任何错误（含用户主动停止）都意味着解题过程结束
    solution.isSolving = false
    if (rawErrMsg && (rawErrMsg.includes('context canceled') || rawErrMsg.includes('canceled'))) {
      handleUserCancellation()
      return
    }
    let title = '请求出错'
    let desc = rawErrMsg || '未知错误'
    const icon = 'x-circle'
    try {
      const errObj = JSON.parse(rawErrMsg)
      if (errObj.message) desc = errObj.message
      if (errObj.statusCode) title = `错误 ${errObj.statusCode}`
    } catch (_) {}
    settingsStore.statusText = '出错'
    settingsStore.statusIcon = '!'
    const usedInline = solution.handleInlineError({ title, desc, icon })
    if (!usedInline) {
      Object.assign(solution.errorState, { show: true, title, desc, icon, rawError: rawErrMsg, showDetails: false })
      solution.isLoading = false
      solution.isAppending = false
      solution.shouldOverwriteHistory = true
    }
  })

  function handleUserCancellation() {
    if (solution.history.length > 0 && solution.activeHistoryIndex === 0) {
      const current = solution.history[0]
      if (settingsStore.settings.keepContext && current.rounds?.length > 1) {
        current.rounds.pop()
        solution.setStreamBuffer('')
        solution.isAppending = true
        solution.isLoading = false
        solution.shouldOverwriteHistory = false
      } else {
        if (current.rounds?.length) current.rounds[0].aiResponse = ''
        solution.setStreamBuffer('')
        solution.isLoading = true
        settingsStore.statusText = '正在思考...'
        settingsStore.statusIcon = '...'
        solution.shouldOverwriteHistory = true
      }
    }
  }

  on('require-api-key', () => {
    if (!ui.showSettings) settingsStore.openSettings()
    ui.activeTab = 'api'
    ui.showToast('请先配置 API Key', 'warning')
  })

  on('open-settings', (tab) => {
    if (!ui.showSettings) settingsStore.openSettings()
    if (tab) ui.activeTab = tab
  })

  on('toggle-toolbar', () => {
    if (ui.isCollapsed) {
      expandToolbar()
    } else {
      collapseToolbar()
    }
  })

  // 快捷键搜题：折叠态下自动展开面板（Go 侧 TriggerSend 发出）
  on('expand-toolbar', () => {
    if (ui.isCollapsed) expandToolbar()
  })

  // 快捷键切换存储复盘模式
  on('toggle-review', () => {
    api.toggleReview()
  })

  // ==================== 面试模式 ====================

  // 会话建立（全新开始，清空旧对话）；结束则清空并回到常规视图。
  // 注意必须同步 interview store 的 isOn——面板与 F5 都依赖它判断三态
  on('interview-state', (running) => {
    ui.isInterviewOn = !!running
    interview.setState(!!running)
    if (running) {
      interview.reset()
      ui.showInterview = true
      if (ui.isCollapsed) expandToolbar()
      settingsStore.statusText = '聆听中'
      settingsStore.statusIcon = '●'
    } else {
      // 结束 = 清空对话并回到常规视图（与"结束"按钮语义一致）
      interview.reset()
      ui.showInterview = false
      settingsStore.statusText = '就绪'
      settingsStore.statusIcon = '●'
    }
  })

  // 收音开始/暂停
  on('interview-listening', (listening) => {
    interview.setListening(listening)
    ui.isInterviewListening = !!listening
    if (interview.isOn) {
      settingsStore.statusText = listening ? '聆听中' : '已暂停'
      settingsStore.statusIcon = listening ? '●' : '❚❚'
    }
  })

  on('interview-transcript', (text) => {
    interview.transcript = text || ''
  })

  on('interview-question', (text) => {
    interview.handleQuestion(text)
  })

  on('interview-skill', (names) => interview.handleSkills(names))

  // 截图搜题命中技能：toast 提示本题加载了哪些技能
  on('skill-matched', (names) => {
    if (names && names.length) ui.showToast('已加载技能：' + names.join('、'), 'info', 3000)
  })

  on('interview-stream-start', () => {
    ui.showInterview = true
    interview.handleStreamStart()
  })

  on('interview-stream-chunk', (token) => interview.handleStreamChunk(token))
  on('interview-stream-thinking', (token) => interview.handleThinkingChunk(token))

  on('interview-answer', (full) => {
    interview.handleAnswer(full)
    settingsStore.statusText = '已作答'
    settingsStore.statusIcon = '✓'
  })

  // 模型判定转写中没有提问：撤销刚建的那一轮，不展示
  on('interview-skip', () => {
    interview.handleSkip()
    settingsStore.statusText = '聆听中'
    settingsStore.statusIcon = '●'
  })

  // ASR 断线重连提示（区别于真正的错误：重连成功后自动撤销）
  let asrReconnecting = false

  on('interview-error', (msg) => {
    asrReconnecting = msg.startsWith('语音识别连接断开')
    interview.handleError(msg)
    settingsStore.statusText = asrReconnecting ? '重连中' : '出错'
    settingsStore.statusIcon = '!'
  })

  // ASR 重连成功：撤销重连提示，恢复聆听状态
  on('interview-asr-recover', () => {
    if (!asrReconnecting) return
    asrReconnecting = false
    if (interview.error && interview.error.msg.startsWith('语音识别连接断开')) {
      interview.error = null
    }
    if (interview.isOn && interview.isListening && !interview.isAnswering) {
      settingsStore.statusText = '聆听中'
      settingsStore.statusIcon = '●'
    }
  })

  // 快捷键：F5 切换（未开始→开始；收音中→暂停；暂停→继续）
  on('toggle-interview', () => {
    if (!interview.isOn) {
      ui.showInterview = true
      api.startInterview()
    } else if (interview.isListening) {
      api.pauseInterview()
    } else {
      api.resumeInterview()
    }
  })

  on('interview-answer-trigger', () => {
    api.triggerInterviewAnswer()
  })

  // 缺少语音识别 Key：打开设置并跳到面试 tab
  on('require-asr-key', () => {
    if (!ui.showSettings) settingsStore.openSettings()
    ui.activeTab = 'interview'
    ui.showToast('请先配置语音识别 API Key', 'warning')
  })

  // 前端重载后恢复面试会话状态（后端会话可能仍在运行）
  api.getInterviewState().then(st => {
    if (st && st.running) {
      interview.setState(true)
      interview.setListening(!!st.listening)
      ui.isInterviewOn = true
      ui.isInterviewListening = !!st.listening
      ui.showInterview = true
      if (ui.isCollapsed) expandToolbar()
      settingsStore.statusText = st.listening ? '聆听中' : '已暂停'
      settingsStore.statusIcon = st.listening ? '●' : '❚❚'
    }
  }).catch(() => {})

  on('toast', (msg) => {
    ui.showToast(msg)
  })

  document.addEventListener('keydown', event => {
    if (
      event.key === 'F12' ||
      (event.ctrlKey && event.shiftKey && event.key === 'I') ||
      (event.ctrlKey && event.shiftKey && event.key === 'J') ||
      (event.ctrlKey && event.key === 'U')
    ) {
      event.preventDefault()
    }
  })
})
</script>

<style scoped>
.disclaimer {
  text-align: center;
  font-size: 10px;
  color: var(--text-muted);
  padding: 2px 0 0;
  pointer-events: none;
  user-select: none;
  flex-shrink: 0;
  letter-spacing: 0.3px;
  opacity: 0.7;
}
</style>
