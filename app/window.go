package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"
)

// ToggleVisibility 切换可见性
func (a *App) ToggleVisibility() {
	a.stateManager.ToggleVisibility()
}

// ToggleClickThrough 切换鼠标穿透
func (a *App) ToggleClickThrough() {
	a.stateManager.ToggleClickThrough()
}

// ToggleShield 切换快捷键屏蔽模式并持久化（F6 快捷键 / 工具条按钮 / 设置开关）
func (a *App) ToggleShield() {
	a.shortcutService.ToggleShield()
	a.persistShieldState()
}

// SetShield 设置快捷键屏蔽模式并持久化（设置界面开关）
func (a *App) SetShield(on bool) {
	a.shortcutService.SetShield(on)
	a.persistShieldState()
}

// GetShieldState 返回快捷键屏蔽模式当前状态（前端启动时查询）
func (a *App) GetShieldState() bool {
	if a.shortcutService == nil {
		return false
	}
	return a.shortcutService.IsShieldOn()
}

// persistShieldState 把屏蔽状态写入配置持久化
func (a *App) persistShieldState() {
	on := a.shortcutService.IsShieldOn()
	if err := a.configManager.Patch(func(cfg *config.Config) {
		cfg.ShieldEnabled = on
	}); err != nil {
		logger.Printf("保存屏蔽状态失败: %v", err)
	}
}

// MoveWindow 移动窗口
func (a *App) MoveWindow(dx, dy int) {
	a.stateManager.MoveWindow(dx, dy)
}

// RestoreFocus 恢复焦点
func (a *App) RestoreFocus() {
	a.stateManager.RestoreFocus()
}

// RemoveFocus 移除焦点
func (a *App) RemoveFocus() {
	a.stateManager.RemoveFocus()
}

// SetWindowAlwaysOnTop 设置窗口是否置顶
func (a *App) SetWindowAlwaysOnTop(alwaysOnTop bool) {
	hwnd := a.stateManager.GetHwnd()
	if hwnd == 0 {
		return
	}
	if alwaysOnTop {
		platform.SetWindowLevel(hwnd, platform.WindowLevelFloating)
	} else {
		platform.SetWindowLevel(hwnd, platform.WindowLevelNormal)
	}
}
