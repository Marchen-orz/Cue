package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/platform"
	"Q-Solver/pkg/screen"
)

// GetScreenshotPreview 获取截图预览（预览即最终截图）
// screenshotMode 传空串时回落到配置值；"window" 作为遗留值按 region 处理
func (a *App) GetScreenshotPreview(quality int, sharpen float64, grayscale bool, noCompression bool, screenshotMode string, screenIndex int) (screen.PreviewResult, error) {
	cfg := a.configManager.Get()
	mode := normalizeScreenshotMode(screenshotMode, cfg.ScreenshotMode)
	if screenshotMode == "" {
		screenIndex = cfg.ScreenIndex
	}
	return a.screenService.CapturePreview(screen.CaptureOptions{
		Quality:       quality,
		Sharpen:       sharpen,
		Grayscale:     grayscale,
		NoCompression: noCompression,
		Mode:          mode,
		ScreenIndex:   screenIndex,
		Region: screen.Region{
			X: cfg.RegionX, Y: cfg.RegionY, W: cfg.RegionW, H: cfg.RegionH,
		},
	})
}

// normalizeScreenshotMode 归一化截图模式
func normalizeScreenshotMode(mode, fallback string) string {
	if mode == "" {
		mode = fallback
	}
	if mode == "window" {
		mode = "region"
	}
	return mode
}

// ListDisplays 枚举显示器（供全屏模式选择截取哪个屏幕）
func (a *App) ListDisplays() []screen.DisplayInfo {
	return screen.ListDisplays()
}

// StartRegionSelect 打开全屏框选遮罩，成功后把区域写入配置并广播事件；
// 返回框选结果（取消/失败返回 nil）
func (a *App) StartRegionSelect() *screen.Region {
	region, err := screen.SelectRegion()
	if err != nil {
		a.EmitEvent("region-select-cancelled", err.Error())
		return nil
	}

	a.configManager.Patch(func(cfg *config.Config) {
		cfg.RegionX = region.X
		cfg.RegionY = region.Y
		cfg.RegionW = region.W
		cfg.RegionH = region.H
	})
	a.EmitEvent("region-selected", map[string]int{
		"x": region.X, "y": region.Y, "w": region.W, "h": region.H,
	})
	return &region
}

// CheckScreenCapturePermission 检查截图权限 (macOS)
func (a *App) CheckScreenCapturePermission() bool {
	return platform.CheckScreenCaptureAccess()
}

// RequestScreenCapturePermission 请求截图权限 (macOS)
func (a *App) RequestScreenCapturePermission() bool {
	return platform.RequestScreenCaptureAccess()
}

// OpenScreenCaptureSettings 打开系统设置的屏幕录制权限页面 (macOS)
func (a *App) OpenScreenCaptureSettings() {
	platform.OpenScreenCaptureSettings()
}

// CheckMicrophoneAccess 检查麦克风权限状态 (macOS)
// 返回: 0=未决定, 1=已授权, 2=已拒绝
func (a *App) CheckMicrophoneAccess() int {
	return platform.CheckMicrophoneAccess()
}

// RequestMicrophoneAccess 请求麦克风权限 (macOS)
func (a *App) RequestMicrophoneAccess() {
	platform.RequestMicrophoneAccess()
}

// OpenMicrophoneSettings 打开系统设置的麦克风权限页面 (macOS)
func (a *App) OpenMicrophoneSettings() {
	platform.OpenMicrophoneSettings()
}
