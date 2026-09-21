package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/resume"
	"Q-Solver/pkg/screen"
	"Q-Solver/pkg/shortcut"
	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/state"
	"Q-Solver/pkg/task"
	"context"
	"runtime/debug"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	configManager *config.ConfigManager
	stateManager  *state.StateManager
	taskManager   *task.TaskCoordinator

	llmService      *llm.Service
	resumeService   *resume.Service
	shortcutService *shortcut.Service
	screenService   *screen.Service
	solver          *solution.Solver
}

func NewApp() *App {
	configManager := config.NewConfigManager()

	return &App{
		configManager: configManager,
		stateManager:  state.NewStateManager(),
		taskManager:   task.NewTaskCoordinator(),
		screenService: screen.NewService(),
	}
}

// recoverPanics 包装 panic 恢复：panic 会直接杀死整个进程（前端表现为"闪退"），
// 捕获后写日志并继续运行，避免一道题的异常拖垮整个应用
func (a *App) recoverPanics(where string) {
	if r := recover(); r != nil {
		logger.Printf("[panic] %s: %v\n%s", where, r, debug.Stack())
	}
}

// SafeGo 启动一个带 panic 恢复的 goroutine
func (a *App) SafeGo(where string, fn func()) {
	go func() {
		defer a.recoverPanics(where)
		fn()
	}()
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.configManager.Load(); err != nil {
		logger.Printf("加载配置失败: %v", err)
	}

	cfg := a.configManager.Get()
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		runtime.WindowSetSize(ctx, cfg.WindowWidth, cfg.WindowHeight)
		logger.Printf("应用保存的窗口尺寸: %dx%d", cfg.WindowWidth, cfg.WindowHeight)
	}

	// 恢复折叠工具条状态（需先解除 Min/Max 钳制才能缩小窗口）
	if cfg.ToolbarCollapsed {
		runtime.WindowSetMinSize(ctx, 0, 0)
		runtime.WindowSetMaxSize(ctx, 0, 0)
		runtime.WindowSetSize(ctx, ToolbarWidth, ToolbarHeight)
		if cfg.ToolbarX > 0 || cfg.ToolbarY > 0 {
			runtime.WindowSetPosition(ctx, cfg.ToolbarX, cfg.ToolbarY)
		}
		a.stateManager.SetToolbarCollapsed(true)
		logger.Printf("恢复折叠工具条: %dx%d @(%d,%d)", ToolbarWidth, ToolbarHeight, cfg.ToolbarX, cfg.ToolbarY)
	}

	a.stateManager.Startup(ctx, a.EmitEvent)
	a.screenService.Startup(ctx)

	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	a.solver = solution.NewSolver(a.llmService.GetProvider())
	a.resumeService = resume.NewService(a.configManager.Get(), a.configManager)

	a.shortcutService = shortcut.NewService(a, a.configManager.Get().Shortcuts, func(callback func(map[string]shortcut.KeyBinding)) {
		a.configManager.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
			callback(newConfig.Shortcuts)
		})
	})
	a.shortcutService.Start()

	// 恢复快捷键屏蔽模式（持久化状态，默认开启）
	a.shortcutService.SetShield(cfg.ShieldEnabled)

	a.configManager.Subscribe(a.onConfigChanged)
	a.stateManager.UpdateInitStatus(state.StatusReady)
}

func (a *App) onConfigChanged(newConfig config.Config, oldConfig config.Config) {
	if a.solver != nil {
		a.solver.SetProvider(a.llmService.GetProvider())
	}

	if !newConfig.KeepContext && a.solver != nil {
		a.solver.ClearHistory()
	}

	logger.Println("配置已更新并应用")
}

func (a *App) OnShutdown(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("[panic] shutdown: %v\n%s", r, debug.Stack())
		}
	}()

	// 折叠态下兜底保存工具条位置（拖动后可能尚未持久化）
	if a.stateManager.GetToolbarCollapsed() {
		x, y := runtime.WindowGetPosition(ctx)
		a.UpdateToolbarPosition(x, y)
	}
	a.stateManager.Shutdown()

	if a.shortcutService != nil {
		a.shortcutService.Stop()
	}
	if err := a.configManager.Save(); err != nil {
		logger.Printf("保存配置失败: %v", err)
	}
}

func (a *App) EmitEvent(eventName string, data ...interface{}) {
	runtime.EventsEmit(a.ctx, eventName, data...)
}

func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}
