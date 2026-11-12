package app

import (
	"Q-Solver/pkg/knowledge"
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
	a.triggerSolveFlow(false)
}

// TriggerContinue 把当前截图（没有就先截一张）发到「当前会话」，追加为新的一轮，
// 模型会看到同一会话里之前的问答，用于追问场景
func (a *App) TriggerContinue() {
	defer a.recoverPanics("TriggerContinue")
	a.triggerSolveFlow(true)
}

func (a *App) triggerSolveFlow(continueSession bool) {
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

	// 复盘模式：题目截图落盘，分配题号
	if a.review.IsEnabled() {
		if _, _, err := a.review.beginQuestion(screenshots); err != nil {
			logger.Printf("[review] 复盘截图保存失败: %v", err)
		}
	}

	a.EmitEvent("start-solving", continueSession)
	a.EmitEvent("user-message", screenshots[0])

	ctx, taskID := a.taskManager.StartTask("solve")
	a.SafeGo("solve", func() {
		defer a.taskManager.CompleteTask(taskID)
		a.solveInternal(ctx, screenshots, continueSession)
	})
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, screenshots []string, continueSession bool) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return false
	}

	req := solution.Request{
		Config:          cfg,
		Screenshots:     screenshots,
		KnowledgePrompt: a.knowledge.Prompt(knowledge.ScopeSolve),
		SkillPrompt:     a.pickSkills(ctx, cfg.Model, "", screenshots, "skill-matched"),
		ReasoningEffort: cfg.ReasoningEffort,
		Continue:        continueSession,
	}

	cb := solution.Callbacks{
		EmitEvent: a.EmitEvent,
	}

	// 复盘模式：拦截解答完成事件，把解答 markdown 写入题目文档
	if a.review.IsEnabled() {
		origEmit := cb.EmitEvent
		cb.EmitEvent = func(name string, data ...interface{}) {
			if name == "solution" && len(data) > 0 {
				if md, ok := data[0].(string); ok {
					if err := a.review.saveSolution(md); err != nil {
						logger.Printf("[review] 复盘解答保存失败: %v", err)
					}
				}
			}
			origEmit(name, data...)
		}
	}

	return a.solver.Solve(ctx, req, cb)
}

func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}

// ClearSolveContext 清空「继续当前会话」依赖的后端对话历史。
// 前端删除当前这条历史记录（列表里最新的一条）后调用，避免再按继续
// 快捷键时又把已删除的会话内容接回去
func (a *App) ClearSolveContext() {
	if a.solver != nil {
		a.solver.ClearHistory()
	}
}
