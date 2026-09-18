import {
  WindowGetPosition,
  WindowSetPosition,
  WindowSetSize,
  WindowSetMinSize,
  WindowSetMaxSize,
} from '../../wailsjs/runtime/runtime'
import { api } from './api'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'

// 与 Go 侧 app/toolbar.go 的常量保持一致
export const TOOLBAR_WIDTH = 236
export const TOOLBAR_HEIGHT = 50

// 展开态的尺寸钳制（与 main.go / ResizeHandle.vue 一致）
const EXPAND_MIN_WIDTH = 840
const EXPAND_MIN_HEIGHT = 700
const EXPAND_MAX_WIDTH = 1024
const EXPAND_MAX_HEIGHT = 768

function clamp(v, min, max) {
  return Math.min(max, Math.max(min, v))
}

/**
 * 折叠为悬浮工具条。
 * 顺序红线：必须先解除 Min/Max 钳制再缩小窗口，否则会被静默钳回 840x700。
 */
export async function collapseToolbar() {
  const ui = useUIStore()
  if (ui.isCollapsed) return

  try {
    const pos = await WindowGetPosition()

    // 先持久化（折叠前记录原窗口位置作为工具条位置）
    await api.setToolbarCollapsed(true, pos.x, pos.y)

    WindowSetMinSize(0, 0)
    WindowSetMaxSize(0, 0)
    WindowSetSize(TOOLBAR_WIDTH, TOOLBAR_HEIGHT)
  } catch (e) {
    console.error('折叠工具条失败', e)
    return
  }

  ui.isCollapsed = true
  ui.hoverToolbarAction = null
}

/**
 * 展开为完整面板，以工具条当前位置为锚点（按屏幕工作区钳制，防止越界）。
 */
export async function expandToolbar() {
  const ui = useUIStore()
  const settingsStore = useSettingsStore()
  if (!ui.isCollapsed) return

  try {
    const s = settingsStore.settings
    // 展开尺寸：优先用户上次拖拽保存的尺寸
    const width = clamp(s.windowWidth || EXPAND_MAX_WIDTH, EXPAND_MIN_WIDTH, EXPAND_MAX_WIDTH)
    const height = clamp(s.windowHeight || EXPAND_MAX_HEIGHT, EXPAND_MIN_HEIGHT, EXPAND_MAX_HEIGHT)

    const pos = await WindowGetPosition()
    const dpr = window.devicePixelRatio || 1

    // WindowGetPosition 是物理像素，screen 工作区是 CSS 像素
    const cssX = pos.x / dpr
    const cssY = pos.y / dpr
    const nx = clamp(cssX, 0, Math.max(0, window.screen.availWidth - width))
    const ny = clamp(cssY, 0, Math.max(0, window.screen.availHeight - height))
    const physX = Math.round(nx * dpr)
    const physY = Math.round(ny * dpr)

    // 先恢复钳制再放大（与折叠相反的顺序）
    WindowSetPosition(physX, physY)
    WindowSetMinSize(EXPAND_MIN_WIDTH, EXPAND_MIN_HEIGHT)
    WindowSetMaxSize(EXPAND_MAX_WIDTH, EXPAND_MAX_HEIGHT)
    WindowSetSize(width, height)

    await api.setToolbarCollapsed(false, physX, physY)
  } catch (e) {
    console.error('展开面板失败', e)
    return
  }

  ui.isCollapsed = false
  ui.hoverToolbarAction = null
}
