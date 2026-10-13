import { defineStore } from 'pinia'
import { reactive, ref, computed, watch } from 'vue'
import { api } from '../services/api'
import { useUIStore } from './ui'
import { currentTheme, setTheme } from '../services/theme'

export const useSettingsStore = defineStore('settings', () => {
  const ui = useUIStore()

  const settings = reactive({
    apiKey: '',
    baseURL: 'https://api.openai.com/v1',
    model: '',
    // 思考强度（仅搜题模式）：'' = 跟随模型/网关默认
    reasoningEffort: '',
    assistantModel: '',
    prompt: '',
    transparency: 0,
    keepContext: false,
    screenshotMode: 'fullscreen',
    compressionQuality: 80,
    sharpening: 0,
    grayscale: true,
    noCompression: false,
    // 展开态窗口尺寸（ResizeHandle 拖拽后保存，供折叠/展开切换使用）
    windowWidth: 1024,
    windowHeight: 768,
    // 折叠工具条状态（仅读取，持久化走独立的 Go 绑定）
    toolbarCollapsed: false,
    // 穿透模式悬浮触发时长（毫秒）
    hoverTriggerMs: 3000,
    // 截图：全屏模式显示器序号 / 框选区域（虚拟屏幕物理像素）
    screenIndex: 0,
    regionX: 0,
    regionY: 0,
    regionW: 0,
    regionH: 0,
    // 面试模式
    interviewAudioSource: 'system',
    interviewMode: 'auto',
    interviewModel: '',
    asrApiKey: '',
    asrModel: 'paraformer-realtime-v2',
    interviewSilenceMs: 800,
  })

  const tempSettings = reactive({ ...settings })
  const shortcuts = reactive({})
  const tempShortcuts = reactive({})
  // 知识库导入进行中（PDF 转写需调用模型，期间不允许关闭设置）
  const isKnowledgeImporting = ref(false)

  const isMacOS = ref(
    typeof navigator !== 'undefined' &&
    (navigator.platform.toLowerCase().includes('mac') ||
      navigator.userAgent.toLowerCase().includes('mac'))
  )

  const recordingAction = ref(null)
  const recordingText = ref('')
  const shortcutActions = [
    { action: 'screenshot', label: '截图', default: 'F8', macDefault: 'Cmd+1' },
    { action: 'send', label: '发送解题', default: 'Ctrl+J', macDefault: 'Cmd+J' },
    { action: 'delete', label: '删除截图', default: 'Ctrl+D', macDefault: 'Cmd+D' },
    { action: 'toggle', label: '隐藏/显示', default: 'F9', macDefault: 'Cmd+2' },
    { action: 'clickthrough', label: '鼠标穿透', default: 'F10', macDefault: 'Cmd+3' },
    { action: 'toggle_toolbar', label: '折叠/展开面板', default: 'F7', macDefault: 'Cmd+4' },
    { action: 'shield', label: '快捷键屏蔽', default: 'F6', macDefault: 'Cmd+5' },
    { action: 'review', label: '存储复盘', default: 'F4', macDefault: 'Cmd+6' },
    { action: 'interview', label: '面试模式', default: 'F5', macDefault: 'Cmd+7' },
    { action: 'interview_answer', label: '面试立即回答', default: 'F3', macDefault: 'Cmd+8' },
    { action: 'move_up', label: '向上移动', default: 'Alt+Up', macDefault: 'Cmd+Option+Up' },
    { action: 'move_down', label: '向下移动', default: 'Alt+Down', macDefault: 'Cmd+Option+Down' },
    { action: 'move_left', label: '向左移动', default: 'Alt+Left', macDefault: 'Cmd+Option+Left' },
    { action: 'move_right', label: '向右移动', default: 'Alt+Right', macDefault: 'Cmd+Option+Right' },
    { action: 'scroll_up', label: '向上滚动', default: 'Alt+PgUp', macDefault: 'Cmd+Option+Shift+Up' },
    { action: 'scroll_down', label: '向下滚动', default: 'Alt+PgDn', macDefault: 'Cmd+Option+Shift+Down' },
    { action: 'opacity_up', label: '增加透明度', default: 'Alt+Shift+Up', macDefault: 'Cmd+Shift+Up' },
    { action: 'opacity_down', label: '降低透明度', default: 'Alt+Shift+Down', macDefault: 'Cmd+Shift+Down' },
  ]

  const maskedKey = computed(() => {
    if (!settings.apiKey) return ''
    if (settings.apiKey.length < 8) return settings.apiKey
    return settings.apiKey.substring(0, 3) + '****' + settings.apiKey.substring(settings.apiKey.length - 4)
  })

  const solveShortcut = computed(() => shortcuts.screenshot?.keyName || shortcuts.solve?.keyName || 'F8')
  const sendShortcut = computed(() => shortcuts.send?.keyName || 'Ctrl+J')
  const deleteShortcut = computed(() => shortcuts.delete?.keyName || 'Ctrl+D')
  const toggleShortcut = computed(() => shortcuts.toggle?.keyName || 'F9')

  const statusText = ref('就绪')
  const statusIcon = ref('●')

  function resetStatus() {
    if (!settings.apiKey) {
      statusText.value = '未配置'
      statusIcon.value = '!'
    } else {
      statusText.value = '已连接'
      statusIcon.value = '✓'
    }
  }

  watch(() => settings.apiKey, () => resetStatus(), { immediate: true })

  watch(() => tempSettings.transparency, (newVal) => {
    applyTransparency(1.0 - newVal)
  })

  window.addEventListener('theme-changed', () => {
    requestAnimationFrame(() => {
      applyTransparency(1.0 - (tempSettings.transparency ?? 0))
    })
  })

  let themeInitialized = false
  watch(currentTheme, () => {
    if (!themeInitialized) {
      themeInitialized = true
      return
    }
    saveSettingsSilent()
  })

  // 切换快捷键屏蔽模式（Go 侧持久化，状态经 shield-state 事件同步到 ui store）
  function toggleShield() {
    api.toggleShield()
  }

  // 设置界面开关调用：直接置位
  function setShield(on) {
    api.setShield(on)
  }

  // 切换存储复盘模式（Go 侧持久化，状态经 review-state 事件同步到 ui store）
  function toggleReview() {
    api.toggleReview()
  }

  // 设置界面开关调用：直接置位
  function setReview(on) {
    api.setReview(on)
  }

  // 快捷键调节全局透明度：settings.transparency（0 全不透明 ~ 1 全透明），
  // 与设置页滑块同一变量；连按只应用即时生效，防抖后静默持久化
  let opacitySaveTimer = null
  function adjustTransparency(direction) {
    const step = 0.05
    const cur = settings.transparency ?? 0
    const next = Math.min(1, Math.max(0, +(cur + (direction === 'up' ? step : -step)).toFixed(2)))
    if (next === cur) return
    settings.transparency = next
    tempSettings.transparency = next
    applyTransparency(1.0 - next)
    ui.showToast(`窗口透明度 ${Math.round(next * 100)}%`, 'info')
    if (opacitySaveTimer) clearTimeout(opacitySaveTimer)
    opacitySaveTimer = setTimeout(() => { saveSettingsSilent() }, 600)
  }

  function applyTransparency(opacity) {
    const root = document.documentElement
    if (!root) return
    const style = getComputedStyle(root)
    const base = style.getPropertyValue('--surface-base').trim()
    let r, g, b
    if (base.startsWith('rgba')) {
      const parts = base.match(/rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*[\d.]+\s*\)/)
      if (parts) {
        r = parts[1]
        g = parts[2]
        b = parts[3]
      }
    }
    if (!r) {
      const theme = root.getAttribute('data-theme')
      if (theme === 'light') {
        r = '252'
        g = '252'
        b = '255'
      } else {
        r = '10'
        g = '10'
        b = '14'
      }
    }
    root.style.setProperty('--app-bg-r', r)
    root.style.setProperty('--app-bg-g', g)
    root.style.setProperty('--app-bg-b', b)
    root.style.setProperty('--app-bg-a', String(opacity))
    root.style.setProperty('--app-panel-a', String(Math.max(0, opacity * 0.5)))
    // 文字透明度跟随全局透明度（下限 0.15，避免完全看不见）；
    // 不再做深色 text-shadow 补偿——文字淡出后阴影会比文字更显眼
    root.style.setProperty('--text-shadow-a', '0')
    root.style.setProperty('--text-alpha', String(Math.max(0.15, opacity)))

    const isLight = root.getAttribute('data-theme') === 'light'
    root.style.setProperty('--surface-elevated-a', String(opacity * (isLight ? 0.96 : 0.92)))
    root.style.setProperty('--surface-card-a', String(opacity * (isLight ? 0.78 : 0.80)))
    root.style.setProperty('--surface-card-hover-a', String(opacity * (isLight ? 0.86 : 0.88)))
    // 代码块背景跟随全局透明度（保持各主题默认不透明度作为上限）
    root.style.setProperty('--code-bg-a', String(opacity * (isLight ? 1 : 0.90)))
    root.style.setProperty('--code-header-a', String(opacity * (isLight ? 1 : 0.95)))
  }

  async function loadSettings() {
    try {
      const backendConfig = await api.getSettings()
      applyConfig(backendConfig)
      if (backendConfig.shortcuts) Object.assign(shortcuts, backendConfig.shortcuts)
      if (backendConfig.theme) setTheme(backendConfig.theme)
      if (settings.apiKey && (!settings.model || settings.model === 'auto')) {
        await fetchModels(settings.apiKey)
        if (ui.availableModels.length > 0 && !settings.model) {
          settings.model = ui.availableModels[0]
          tempSettings.model = ui.availableModels[0]
        }
      }
    } catch (e) {
      console.error('loadSettings error', e)
    }
  }

  function applyConfig(config) {
    settings.apiKey = config.apiKey || ''
    settings.baseURL = config.baseURL || 'https://api.openai.com/v1'
    settings.model = config.model || ''
    settings.reasoningEffort = config.reasoningEffort || ''
    settings.assistantModel = config.assistantModel || ''
    settings.prompt = config.prompt || ''
    settings.compressionQuality = config.compressionQuality || 80
    settings.sharpening = config.sharpening || 0
    settings.grayscale = config.grayscale !== undefined ? config.grayscale : true
    settings.noCompression = config.noCompression || false
    settings.keepContext = config.keepContext || false
    settings.screenshotMode = config.screenshotMode || 'fullscreen'
    if (settings.screenshotMode === 'window') settings.screenshotMode = 'region'
    settings.windowWidth = config.windowWidth || 1024
    settings.windowHeight = config.windowHeight || 768
    settings.toolbarCollapsed = config.toolbarCollapsed || false
    settings.hoverTriggerMs = config.hoverTriggerMs || 3000
    settings.screenIndex = config.screenIndex || 0
    settings.regionX = config.regionX || 0
    settings.regionY = config.regionY || 0
    settings.regionW = config.regionW || 0
    settings.regionH = config.regionH || 0
    // 面试模式
    settings.interviewAudioSource = config.interviewAudioSource || 'system'
    settings.interviewMode = config.interviewMode || 'auto'
    settings.interviewModel = config.interviewModel || ''
    settings.asrApiKey = config.asrApiKey || ''
    settings.asrModel = config.asrModel || 'paraformer-realtime-v2'
    settings.interviewSilenceMs = config.interviewSilenceMs || 800
    // 屏蔽状态以 Go 侧运行时为准（Startup 已应用），此处仅同步显示
    ui.isShieldOn = !!config.shieldEnabled

    const opacity = config.opacity !== undefined ? config.opacity : 1.0
    settings.transparency = 1.0 - opacity
    applyTransparency(opacity)
    Object.assign(tempSettings, JSON.parse(JSON.stringify(settings)))
  }

  async function fetchModels(apiKey) {
    if (!apiKey) return false
    ui.isLoadingModels = true
    try {
      const models = await api.getModels(apiKey)
      if (models && models.length > 0) {
        ui.availableModels = models
        if (!tempSettings.model || !models.includes(tempSettings.model)) {
          tempSettings.model = models[0]
        }
        return true
      }
      ui.availableModels = []
      return false
    } catch (e) {
      console.error('获取模型列表失败', e)
      ui.availableModels = []
      let errorMsg = '获取模型列表失败'
      try {
        const errObj = JSON.parse(e.message || e)
        if (errObj.message) errorMsg = errObj.message
      } catch (_) {
        errorMsg = e.message || '获取模型列表失败'
      }
      ui.showToast(errorMsg, 'error')
      return false
    } finally {
      ui.isLoadingModels = false
    }
  }

  async function refreshModels() {
    if (!tempSettings.apiKey) {
      ui.showToast('请先填写 API Key', 'warning')
      return
    }
    const success = await fetchModels(tempSettings.apiKey)
    if (success && ui.availableModels.length > 0) {
      ui.showToast(`已加载 ${ui.availableModels.length} 个模型`, 'success')
    }
  }

  async function testConnection() {
    if (!tempSettings.model) {
      ui.showToast('请先选择模型', 'warning')
      return
    }
    ui.isTestingConnection = true
    ui.connectionStatus = null
    try {
      const currentConfig = JSON.parse(JSON.stringify(settings))
      settings.baseURL = tempSettings.baseURL
      const result = await api.testConnection(tempSettings.apiKey, tempSettings.model)
      Object.assign(settings, currentConfig)
      if (result === '') {
        ui.connectionStatus = { type: 'success', icon: 'check-circle', message: `模型 ${tempSettings.model} 连接成功` }
        ui.showToast('连接测试成功', 'success')
      } else {
        ui.connectionStatus = { type: 'error', icon: 'alert-triangle', message: result }
        ui.showToast('连接测试失败', 'error')
      }
    } catch (e) {
      ui.connectionStatus = { type: 'error', icon: 'alert-triangle', message: e.message || '连接测试失败' }
    } finally {
      ui.isTestingConnection = false
    }
  }

  function buildConfigToSave(sourceSettings, sourceShortcuts) {
    return {
      apiKey: sourceSettings.apiKey,
      baseURL: sourceSettings.baseURL,
      model: sourceSettings.model,
      reasoningEffort: sourceSettings.reasoningEffort || '',
      assistantModel: sourceSettings.assistantModel,
      prompt: '',
      opacity: 1.0 - sourceSettings.transparency,
      keepContext: sourceSettings.keepContext,
      screenshotMode: sourceSettings.screenshotMode,
      compressionQuality: sourceSettings.compressionQuality,
      sharpening: sourceSettings.sharpening,
      grayscale: sourceSettings.grayscale,
      noCompression: sourceSettings.noCompression,
      hoverTriggerMs: sourceSettings.hoverTriggerMs,
      shieldEnabled: ui.isShieldOn,
      screenIndex: sourceSettings.screenIndex,
      regionX: sourceSettings.regionX,
      regionY: sourceSettings.regionY,
      regionW: sourceSettings.regionW,
      regionH: sourceSettings.regionH,
      shortcuts: sourceShortcuts,
      theme: currentTheme.value,
      interviewAudioSource: sourceSettings.interviewAudioSource,
      interviewMode: sourceSettings.interviewMode,
      interviewModel: sourceSettings.interviewModel,
      asrApiKey: sourceSettings.asrApiKey,
      asrModel: sourceSettings.asrModel,
      interviewSilenceMs: sourceSettings.interviewSilenceMs,
    }
  }

  async function saveSettings() {
    try {
      if (!tempSettings.model && tempSettings.apiKey) {
        ui.showToast('正在自动获取模型...', 'info')
        await fetchModels(tempSettings.apiKey)
        if (!tempSettings.model && ui.availableModels.length > 0) {
          tempSettings.model = ui.availableModels[0]
        }
      }

      Object.assign(shortcuts, JSON.parse(JSON.stringify(tempShortcuts)))

      const err = await api.syncSettings(JSON.stringify(buildConfigToSave(tempSettings, tempShortcuts)))
      if (err) {
        ui.showToast(err, 'error')
      } else {
        ui.showToast('设置已保存', 'success')
        Object.assign(settings, tempSettings)
        resetStatus()
        closeSettings()
      }
    } catch (e) {
      console.error('保存设置失败', e)
      ui.showToast('保存失败', 'error')
    }
  }

  function openSettings() {
    api.restoreFocus()
    Object.assign(tempSettings, JSON.parse(JSON.stringify(settings)))
    Object.assign(tempShortcuts, JSON.parse(JSON.stringify(shortcuts)))
    ui.connectionStatus = null
    if (settings.apiKey && ui.availableModels.length === 0) {
      fetchModels(settings.apiKey)
    }
    ui.showSettings = true
  }

  function closeSettings() {
    if (isKnowledgeImporting.value) {
      ui.showToast('知识库文档正在导入中，请稍候...', 'warning')
      return
    }
    api.removeFocus()
    ui.showSettings = false
    if (recordingAction.value) api.stopRecordingKey()
    recordingAction.value = null
    recordingText.value = ''
    resetTempSettings()
  }

  function resetTempSettings() {
    Object.assign(tempSettings, settings)
    applyTransparency(1.0 - settings.transparency)
  }

  function recordKey(action) {
    if (isMacOS.value) return
    recordingAction.value = action
    recordingText.value = '请按键...'
    api.startRecordingKey(action)
  }

  async function saveSettingsSilent() {
    try {
      await api.syncSettings(JSON.stringify(buildConfigToSave(settings, shortcuts)))
    } catch (e) {
      console.error('静默保存失败', e)
    }
  }

  return {
    settings, tempSettings,
    shortcuts, tempShortcuts,
    isKnowledgeImporting,
    isMacOS,
    recordingAction, recordingText, shortcutActions,
    maskedKey, solveShortcut, sendShortcut, deleteShortcut, toggleShortcut,
    statusText, statusIcon, resetStatus,
    loadSettings, fetchModels, refreshModels, testConnection,
    saveSettings, saveSettingsSilent, openSettings, closeSettings, resetTempSettings,
    toggleShield, setShield, toggleReview, setReview,
    recordKey,
    adjustTransparency,
    applyTransparency,
  }
})
