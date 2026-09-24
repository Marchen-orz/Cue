package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

// ToggleReview 切换存储复盘模式并持久化（F4 快捷键 / 工具条按钮）
func (a *App) ToggleReview() {
	a.SetReview(!a.review.IsEnabled())
}

// SetReview 设置存储复盘模式并持久化
func (a *App) SetReview(on bool) {
	a.review.SetEnabled(on)
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.ReviewEnabled = on
	})
	a.EmitEvent("review-state", on)
}

// GetReviewState 返回存储复盘模式当前状态（前端启动时查询）
func (a *App) GetReviewState() bool {
	return a.review.IsEnabled()
}

// SetReviewDir 设置复盘存储根目录并持久化（传空恢复默认文档目录）
func (a *App) SetReviewDir(dir string) {
	a.review.SetBaseDir(dir)
	a.configManager.Patch(func(cfg *config.Config) {
		cfg.ReviewDir = dir
	})
	a.EmitEvent("review-state", a.review.IsEnabled())
}

// GetReviewDir 返回复盘存储根目录（空表示使用默认文档目录）
func (a *App) GetReviewDir() string {
	return a.configManager.Get().ReviewDir
}

// OpenReviewDir 在资源管理器中打开复盘存储根目录
func (a *App) OpenReviewDir() {
	dir := a.configManager.Get().ReviewDir
	if dir == "" {
		dir = a.review.DefaultDir()
	}
	a.SafeGo("OpenReviewDir", func() {
		_ = platform.OpenInExplorer(dir)
	})
}

// SelectReviewDir 弹出系统文件夹选择对话框，选中即保存并生效
// 返回最终生效路径（取消返回空串）
func (a *App) SelectReviewDir() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择复盘存储位置",
	})
	if err != nil || dir == "" {
		return ""
	}
	a.SetReviewDir(dir)
	return dir
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
