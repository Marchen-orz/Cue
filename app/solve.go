package app

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/solution"
	"context"
	"sync"
)

const MaxScreenshots = 3

// screenshotBuffer 截图缓冲：会被钩子触发 goroutine 与前端绑定 goroutine
// 并发读写，必须持锁访问
var (
	screenshotMu     sync.Mutex
	screenshotBuffer []string
)

func appendScreenshot(b64 string) {
	screenshotMu.Lock()
	defer screenshotMu.Unlock()
	screenshotBuffer = append(screenshotBuffer, b64)
}

func takeScreenshots() []string {
	screenshotMu.Lock()
	defer screenshotMu.Unlock()
	out := make([]string, len(screenshotBuffer))
	copy(out, screenshotBuffer)
	screenshotBuffer = nil
	return out
}

func (a *App) TriggerScreenshot() {
	defer a.recoverPanics("TriggerScreenshot")
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略截图：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	if screenshotCount() >= MaxScreenshots {
		a.EmitEvent("toast", "最多截图 3 张图片，请先发送或删除")
		return
	}

	// 框选区域模式但尚未设置区域：先弹出框选，取消则放弃本次截图
	if normalizeScreenshotMode(cfg.ScreenshotMode, "") == "region" && cfg.RegionW <= 0 {
		if a.StartRegionSelect() == nil {
			a.EmitEvent("toast", "已取消框选")
			return
		}
	}

	previewResult, err := a.GetScreenshotPreview(
		cfg.CompressionQuality,
		cfg.Sharpening,
		cfg.Grayscale,
		cfg.NoCompression,
		cfg.ScreenshotMode,
		cfg.ScreenIndex,
	)
	if err != nil {
		logger.Printf("截图失败: %v\n", err)
		a.EmitEvent("toast", "截图失败: "+err.Error())
		return
	}

	appendScreenshot(previewResult.Base64)
	a.EmitEvent("screenshot-taken", previewResult.Base64, screenshotCount())
}

func screenshotCount() int {
	screenshotMu.Lock()
	defer screenshotMu.Unlock()
	return len(screenshotBuffer)
}

func (a *App) RemoveScreenshot(index int) {
	defer a.recoverPanics("RemoveScreenshot")
	screenshotMu.Lock()
	if index < 0 || index >= len(screenshotBuffer) {
		screenshotMu.Unlock()
		return
	}
	screenshotBuffer = append(screenshotBuffer[:index], screenshotBuffer[index+1:]...)
	count := len(screenshotBuffer)
	screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

func (a *App) RemoveLastScreenshot() {
	screenshotMu.Lock()
	if len(screenshotBuffer) == 0 {
		screenshotMu.Unlock()
		return
	}
	index := len(screenshotBuffer) - 1
	screenshotBuffer = screenshotBuffer[:index]
	count := len(screenshotBuffer)
	screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

func (a *App) ClearScreenshots() {
	screenshotMu.Lock()
	screenshotBuffer = nil
	screenshotMu.Unlock()
	a.EmitEvent("screenshots-cleared")
}

func (a *App) TriggerSend() {
	defer a.recoverPanics("TriggerSend")
	// 快捷键触发路径不经过前端按钮，折叠态下通知前端展开面板
	a.EmitEvent("expand-toolbar")

	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	screenshotMu.Lock()
	if len(screenshotBuffer) == 0 {
		screenshotMu.Unlock()
		// 框选区域模式但尚未设置区域：先弹出框选，取消则放弃发送
		if normalizeScreenshotMode(cfg.ScreenshotMode, "") == "region" && cfg.RegionW <= 0 {
			if a.StartRegionSelect() == nil {
				a.EmitEvent("toast", "已取消框选")
				return
			}
		}
		previewResult, err := a.GetScreenshotPreview(
			cfg.CompressionQuality,
			cfg.Sharpening,
			cfg.Grayscale,
			cfg.NoCompression,
			cfg.ScreenshotMode,
			cfg.ScreenIndex,
		)
		if err != nil {
			logger.Printf("截图失败: %v\n", err)
			a.EmitEvent("toast", "截图失败: "+err.Error())
			return
		}
		appendScreenshot(previewResult.Base64)
	} else {
		screenshotMu.Unlock()
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略重复触发：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	screenshots := takeScreenshots()

	a.EmitEvent("start-solving")
	a.EmitEvent("user-message", screenshots[0])

	ctx, taskID := a.taskManager.StartTask("solve")
	a.SafeGo("solve", func() {
		defer a.taskManager.CompleteTask(taskID)
		a.solveInternal(ctx, screenshots)
	})
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, screenshots []string) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return false
	}

	req := solution.Request{
		Config:      cfg,
		Screenshots: screenshots,
	}

	cb := solution.Callbacks{
		EmitEvent: a.EmitEvent,
	}

	return a.solver.Solve(ctx, req, cb)
}

func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}
