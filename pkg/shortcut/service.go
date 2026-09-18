package shortcut

import (
	"Q-Solver/pkg/logger"
	"fmt"
	"runtime"
)

type Service struct {
	manager  *Manager
	delegate ServiceDelegate
}

// SubscribeFunc 配置变更订阅函数类型
type SubscribeFunc func(callback func(shortcuts map[string]KeyBinding))

func NewService(delegate ServiceDelegate, initialShortcuts map[string]KeyBinding, subscribe SubscribeFunc) *Service {
	s := &Service{
		manager:  NewManager(),
		delegate: delegate,
	}
	// 异步分发触发：钩子回调若同步执行业务逻辑（如截图需数百毫秒），
	// 超过系统 LowLevelHooksTimeout 后 Windows 会静默移除钩子，
	// 导致之后所有快捷键失效且不再屏蔽按键
	s.manager.OnTrigger = func(action string) {
		go s.handleTrigger(action)
	}
	s.manager.OnShieldChange = func(on bool) {
		s.delegate.EmitEvent("shield-state", on)
	}
	s.manager.OnRecord = s.handleRecord
	s.manager.OnRecordingComplete = s.handleRecordingComplete
	s.manager.OnError = func(msg string) {}

	// 初始化快捷键
	s.manager.ReplaceShortcuts(initialShortcuts)

	// 自注册配置变更回调
	subscribe(func(shortcuts map[string]KeyBinding) {
		s.manager.ReplaceShortcuts(shortcuts)
		logger.Println("快捷键配置已更新")
	})

	return s
}

func (s *Service) Start() {
	s.manager.Start()
}

func (s *Service) Stop() {
	s.manager.Stop()
}

func (s *Service) GetShortcuts() map[string]KeyBinding {
	return s.manager.GetShortcuts()
}

func (s *Service) SetShortcuts(shortcuts map[string]KeyBinding) {
	s.manager.ReplaceShortcuts(shortcuts)
}

func (s *Service) StartRecording(action string) {
	// macOS 不支持热键录制
	if runtime.GOOS == "darwin" {
		s.delegate.EmitEvent("toast", "macOS 不支持自定义快捷键，请使用预设快捷键")
		return
	}
	s.manager.StartRecording(action)
}

// ToggleShield 切换键盘屏蔽模式
func (s *Service) ToggleShield() {
	s.manager.SetShield(!s.manager.IsShieldOn())
}

// IsShieldOn 查询键盘屏蔽模式状态
func (s *Service) IsShieldOn() bool {
	return s.manager.IsShieldOn()
}

func (s *Service) StopRecording() {
	s.manager.StopRecording()
}

func (s *Service) handleTrigger(action string) {
	switch action {
	case "screenshot":
		logger.Println("触发截图")
		s.delegate.TriggerScreenshot()
	case "send":
		logger.Println("触发发送")
		s.delegate.TriggerSend()
	case "delete":
		logger.Println("触发删除截图")
		s.delegate.TriggerDeleteScreenshot()
	case "toggle":
		logger.Println("切换可见性")
		s.delegate.ToggleVisibility()
	case "clickthrough":
		logger.Println("切换鼠标穿透")
		s.delegate.ToggleClickThrough()
	case "toggle_toolbar":
		logger.Println("切换折叠工具条")
		s.delegate.EmitEvent("toggle-toolbar")
	case "shield":
		logger.Println("切换键盘屏蔽")
		s.delegate.ToggleShield()
	case "move_up":
		s.delegate.MoveWindow(0, -10)
	case "move_down":
		s.delegate.MoveWindow(0, 10)
	case "move_left":
		s.delegate.MoveWindow(-10, 0)
	case "move_right":
		s.delegate.MoveWindow(10, 0)
	case "scroll_up":
		s.delegate.ScrollContent("up")
	case "scroll_down":
		s.delegate.ScrollContent("down")
	}
}

func (s *Service) handleRecord(action string, keyName string, comboID string) {
	s.delegate.EmitEvent("key-recorded", map[string]string{
		"action":  action,
		"keyName": keyName,
		"comboID": comboID,
	})
	logger.Printf("快捷键录制完成: %s -> %s->%s\n", action, keyName, comboID)
	s.manager.shortcutsMu.Lock()
	s.manager.Shortcuts[action] = KeyBinding{
		ComboID: comboID,
		KeyName: keyName,
	}
	s.manager.shortcutsMu.Unlock()
}

func (s *Service) handleRecordingComplete(action string, keyName string, comboID string) {
	// 检查冲突
	conflict := false
	conflictAction := ""
	s.manager.shortcutsMu.RLock()
	for act, binding := range s.manager.Shortcuts {
		if binding.ComboID == comboID && act != action {
			conflict = true
			conflictAction = act
			break
		}
	}
	s.manager.shortcutsMu.RUnlock()

	if conflict {
		s.delegate.EmitEvent("shortcut-error", fmt.Sprintf("快捷键冲突：该组合键已被“%s”占用，请重新输入", conflictAction))
		logger.Printf("快捷键录制失败，冲突: %s -> %s\n", action, keyName)
		// 重新开始录制
		s.manager.StartRecording(action)
		return
	}
}
