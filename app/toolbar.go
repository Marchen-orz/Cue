package app

import "Q-Solver/pkg/config"

// 折叠工具条尺寸（前端 services/toolbar.js 保持同值）
const (
	ToolbarWidth  = 212
	ToolbarHeight = 50
)

// SetToolbarCollapsed 设置工具条折叠状态并持久化（x/y 为窗口左上角物理像素屏幕坐标）
func (a *App) SetToolbarCollapsed(collapsed bool, x, y int) {
	a.stateManager.SetToolbarCollapsed(collapsed)
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.ToolbarCollapsed = collapsed
		cfg.ToolbarX = x
		cfg.ToolbarY = y
	})
}

// UpdateToolbarPosition 更新折叠工具条位置（拖动/移动后持久化）
func (a *App) UpdateToolbarPosition(x, y int) {
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.ToolbarX = x
		cfg.ToolbarY = y
	})
}
