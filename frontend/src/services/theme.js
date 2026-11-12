/**
 * 主题管理服务
 * 三档：light / dark / system（跟随操作系统明暗设置，系统切换时实时同步）。
 * 偏好值（themeMode）通过后端 config 持久化，localStorage 作为启动快速缓存（防闪烁）；
 * 实际生效的明暗（currentTheme）= 偏好值解析后的 light / dark，写入 <html data-theme>。
 * 默认主题：light
 */
import { ref, computed, watch } from 'vue'

const THEME_KEY = 'cue-theme'
const LEGACY_THEME_KEY = 'q-solver-theme'

export const THEME_MODES = ['light', 'dark', 'system']

function isThemeMode(v) {
  return THEME_MODES.includes(v)
}

function loadModeFromCache() {
  if (typeof localStorage !== 'undefined') {
    const cached = localStorage.getItem(THEME_KEY) || localStorage.getItem(LEGACY_THEME_KEY)
    if (isThemeMode(cached)) return cached
  }
  return 'light'
}

// 启动时先从 localStorage 快速加载（避免白→黑闪烁），后续由 settings store 同步后端值
export const themeMode = ref(loadModeFromCache())

// 系统明暗偏好：WebView 的 prefers-color-scheme 跟随操作系统，变化时触发 change 事件
const systemQuery = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null
const systemDark = ref(systemQuery ? systemQuery.matches : false)
if (systemQuery) {
  systemQuery.addEventListener('change', (e) => { systemDark.value = e.matches })
}

// 实际生效的主题（light / dark）
export const currentTheme = computed(() => {
  if (themeMode.value === 'system') return systemDark.value ? 'dark' : 'light'
  return themeMode.value
})

export function setThemeMode(mode) {
  if (isThemeMode(mode)) themeMode.value = mode
}

// light → dark → system → light
export function cycleThemeMode() {
  const next = THEME_MODES[(THEME_MODES.indexOf(themeMode.value) + 1) % THEME_MODES.length]
  themeMode.value = next
}

export function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme)
  // 触发自定义事件，通知其他模块
  window.dispatchEvent(new CustomEvent('theme-changed', { detail: { theme } }))
}

// 偏好值写入 localStorage 作为快速缓存
watch(themeMode, (mode) => {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(THEME_KEY, mode)
  }
}, { immediate: true })

// 实际明暗变化（手动切换或系统明暗变化）时应用到页面
watch(currentTheme, (val) => applyTheme(val), { immediate: true })
