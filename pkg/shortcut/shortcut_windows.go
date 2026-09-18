//go:build windows

package shortcut

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type Manager struct {
	hHook           uintptr
	hMouseHook      uintptr
	recordingKeyFor string
	maxComboKeys    map[uint32]bool
	heldKeys        map[uint32]bool
	swallowedDown   map[uint32]bool // 已吞掉 keydown 的键，松开时同步吞 keyup
	swallowedMods   map[uint32]swallowedModInfo // 屏蔽模式拦截的修饰键（尚未补发）

	// Shortcuts 会被钩子线程读取、配置订阅并发写入，必须持锁访问
	shortcutsMu sync.RWMutex

	// 键盘屏蔽模式：开启后钩子吞掉系统所有键盘输入（原子读写，钩子线程与业务 goroutine 并发）
	shieldOn atomic.Bool
	// OnShieldChange 屏蔽状态变化回调（可能从钩子线程发起）
	OnShieldChange func(bool)

	// Callbacks
	OnTrigger           func(action string)
	OnRecord            func(action string, keyName string, comboID string)
	OnRecordingComplete func(action string, keyName string, comboID string)
	OnError             func(msg string)

	// Configuration
	Shortcuts map[string]KeyBinding
}

var globalManager *Manager

func NewManager() *Manager {
	return &Manager{
		heldKeys:      make(map[uint32]bool),
		maxComboKeys:  make(map[uint32]bool),
		swallowedDown: make(map[uint32]bool),
		swallowedMods: make(map[uint32]swallowedModInfo),
		Shortcuts:     make(map[string]KeyBinding),
	}
}

func (m *Manager) Start() {
	globalManager = m
	go m.installHooks()
}

// SetShield 开启/关闭键盘屏蔽模式（屏蔽系统所有键盘输入，按 shield 快捷键切换）
func (m *Manager) SetShield(on bool) {
	m.shieldOn.Store(on)
	if on {
		m.heldKeys = make(map[uint32]bool)
		m.swallowedDown = make(map[uint32]bool)
		m.swallowedMods = make(map[uint32]swallowedModInfo)
	} else if len(m.swallowedMods) > 0 {
		// 关闭时若还有被拦截的修饰键未释放，先补发，避免系统快捷键状态错乱
		m.reinjectSwallowedModifiers()
	}
	logger.Printf("[shield] 键盘屏蔽模式: %v", on)
	if m.OnShieldChange != nil {
		go m.OnShieldChange(on)
	}
}

// IsShieldOn 查询键盘屏蔽模式状态
func (m *Manager) IsShieldOn() bool {
	return m.shieldOn.Load()
}

// ReplaceShortcuts 用新配置整体替换快捷键表（配置订阅并发调用，须持锁）
func (m *Manager) ReplaceShortcuts(shortcuts map[string]KeyBinding) {
	m.shortcutsMu.Lock()
	defer m.shortcutsMu.Unlock()
	for k := range m.Shortcuts {
		delete(m.Shortcuts, k)
	}
	for k, v := range shortcuts {
		m.Shortcuts[k] = v
	}
}

// GetShortcuts 返回快捷键表副本
func (m *Manager) GetShortcuts() map[string]KeyBinding {
	m.shortcutsMu.RLock()
	defer m.shortcutsMu.RUnlock()
	out := make(map[string]KeyBinding, len(m.Shortcuts))
	for k, v := range m.Shortcuts {
		out[k] = v
	}
	return out
}

func (m *Manager) Stop() {
	if m.hHook != 0 {
		if platform.UnhookWindowsHookEx(m.hHook) {
			logger.Println("卸载键盘Hook成功")
		} else {
			logger.Println("卸载键盘Hook失败")
		}
		m.hHook = 0
	}
	if m.hMouseHook != 0 {
		if platform.UnhookWindowsHookEx(m.hMouseHook) {
			logger.Println("卸载鼠标Hook成功")
		} else {
			logger.Println("卸载鼠标Hook失败")
		}
		m.hMouseHook = 0
	}
	globalManager = nil
}

func (m *Manager) StartRecording(action string) {
	m.recordingKeyFor = action
	m.maxComboKeys = make(map[uint32]bool)
	logger.Printf("开始录制快捷键: %s\n", action)
}

func (m *Manager) StopRecording() {
	m.recordingKeyFor = ""
	logger.Println("停止录制快捷键")
}

func (m *Manager) installHooks() {
	// 获取模块句柄
	hMod := platform.GetModuleHandle("")

	// 创建键盘回调
	kbdCallback := syscall.NewCallback(keyboardHookProc)
	// 安装键盘钩子
	m.hHook = platform.SetWindowsHookEx(platform.WH_KEYBOARD_LL, kbdCallback, hMod, 0)
	if m.hHook == 0 {
		logger.Println("[hook] 安装键盘钩子失败")
	} else {
		logger.Printf("[hook] 键盘钩子已安装 hHook=0x%x", m.hHook)
	}

	// 创建鼠标回调
	mouseCallback := syscall.NewCallback(mouseHookProc)
	// 安装鼠标钩子
	m.hMouseHook = platform.SetWindowsHookEx(platform.WH_MOUSE_LL, mouseCallback, hMod, 0)
	if m.hMouseHook == 0 {
		logger.Println("安装鼠标钩子失败")
	}

	if m.hHook == 0 && m.hMouseHook == 0 {
		return
	}

	// 消息循环
	var msg platform.MSG
	for {
		n := platform.GetMessage(&msg, 0, 0, 0)
		if n == -1 {
			// GetMessage 出错：循环一旦停止泵消息，低级钩子会被系统停用，按键不再被屏蔽
			logger.Println("[hook] GetMessage 返回 -1，消息循环退出（钩子将失效）")
			return
		}
		if n <= 0 {
			return
		}
	}
}

// 这里解释了为什么只能吞掉第二个键，所以导致丢失焦点的问题：（其实是因为alt键的问题）
// 第一个键（Alt）按下：
// 记录：heldKeys = {Alt}
// 判断：有快捷键是只按 Alt 的吗？ -> 没有。
// 结果：放行（Chrome 收到 Alt）。
// 第二个键（~）按下：
// 记录：heldKeys = {Alt, ~}
// 判断：有快捷键是 Alt + ~ 的吗？ -> 有！
// 结果：拦截（return 1，Chrome 收不到 ~）。
//
// 屏蔽模式（shield 开启时）的三层拦截：
//  1. 完整命中组合 -> 立即吞掉并触发动作（与屏蔽关闭时相同），浏览器全程不可见
//  2. 组合前缀（如 Ctrl+J 的 Ctrl 单独按下）-> 吞掉，使组合从按下第一刻就隐身
//  3. 前缀之后按下了不属于任何组合的普通键（如 Ctrl+C 的 C）-> 把已吞掉的修饰键
//     用 SendInput 补发回去再放行该键，保证 Ctrl+C/Ctrl+V 等系统快捷键照常工作，
//     不会漏出"裸字母"（Ctrl 被吞后浏览器会把 C 当普通字符输入）
// 屏蔽关闭时：仅完整命中的组合被吞掉（前缀修饰键会泄漏，见上面的 Alt 分析）。
// 单独的字母/数字/方向键等永远放行——它们只是普通打字，不属于组合前缀。
func keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if globalManager == nil {
		return 0
	}
	// 只有当 nCode >= 0 时才处理消息，否则直接放行
	if nCode >= 0 {
		// 将 lParam 指针转换为键盘钩子结构体
		kbd := (*platform.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))

		// 注入事件（我们自己的 SendInput 补发，或 AutoHotkey 等第三方）直接放行：
		// 既避免拦截自己补发的键，也防止补发事件污染 heldKeys 状态
		if kbd.Flags&platform.LLKHF_INJECTED != 0 {
			return platform.CallNextHookEx(globalManager.hHook, nCode, wParam, lParam)
		}

		// 监听按下事件 (WM_KEYDOWN) 或 系统按键按下 (WM_SYSKEYDOWN，比如按住 Alt 时)
		if wParam == platform.WM_KEYDOWN || wParam == platform.WM_SYSKEYDOWN {
			globalManager.heldKeys[kbd.VkCode] = true
			if onKeysChanged() {
				// 记录被吞的按下键，松开时同步吞掉 keyup，避免浏览器收到"只有 keyup"的残缺序列
				globalManager.swallowedDown[kbd.VkCode] = true
				logger.Printf("[hook] down vk=%d -> 命中快捷键，已吞掉", kbd.VkCode)
				return 1
			}

			// 屏蔽模式下的组合前缀拦截
			if globalManager.shieldOn.Load() {
				if isModifierVK(kbd.VkCode) && (isPartOfAnyShortcut() || globalManager.hasSwallowedModifier()) {
					// 修饰键且正处在组合前缀过程中：吞掉，让组合整体隐身
					globalManager.swallowedDown[kbd.VkCode] = true
					globalManager.swallowedMods[kbd.VkCode] = swallowedModInfo{
						scan: byte(kbd.ScanCode),
						ext:  kbd.Flags&platform.LLKHF_EXTENDED != 0,
					}
					logger.Printf("[hook] down vk=%d -> 屏蔽模式拦截组合前缀修饰键", kbd.VkCode)
					return 1
				}
				if !isModifierVK(kbd.VkCode) && globalManager.hasSwallowedModifier() {
					// 前缀之后按了不构成组合的普通键：补发修饰键再放行该键，
					// 保证 Ctrl+C/Ctrl+V 等系统快捷键在屏蔽期间照常可用
					globalManager.reinjectSwallowedModifiers()
					globalManager.injectKey(kbd.VkCode, byte(kbd.ScanCode), kbd.Flags&platform.LLKHF_EXTENDED != 0)
					logger.Printf("[hook] down vk=%d -> 补发修饰键后放行（非组合按键）", kbd.VkCode)
					return 1
				}
			}

			logger.Printf("[hook] down vk=%d -> 放行", kbd.VkCode)
		}
		// 处理松开事件
		if wParam == platform.WM_KEYUP || wParam == platform.WM_SYSKEYUP {
			// 1. 从 map 中移除该键
			delete(globalManager.heldKeys, kbd.VkCode)

			// 录制模式下，松开按键也要检查是否结束录制
			if globalManager.recordingKeyFor != "" {
				if len(globalManager.heldKeys) == 0 {
					finishRecording()
				}
				return 1 // 录制期间吞掉所有按键
			}

			// 2. 之前被吞掉 keydown 的键，keyup 也一并吞掉
			if globalManager.swallowedDown[kbd.VkCode] {
				delete(globalManager.swallowedDown, kbd.VkCode)
				delete(globalManager.swallowedMods, kbd.VkCode)
				logger.Printf("[hook] up vk=%d -> 已吞掉（配套 keydown）", kbd.VkCode)
				return 1
			}
		}
	}

	// 如果不是我们要拦截的键，或者 nCode < 0，必须调用 CallNextHookEx
	// 否则会导致系统键盘卡死或其他人无法使用键盘
	return platform.CallNextHookEx(globalManager.hHook, nCode, wParam, lParam)
}

// modifierVKs 修饰键 VK -> 补发时使用的通用 VK（Ctrl 162/163 -> 17，Alt 164/165 -> 18，Shift 160/161 -> 16）
var modifierVKs = map[uint32]uint32{
	160: 16, 161: 16,
	162: 17, 163: 17,
	164: 18, 165: 18,
	91: 91, 92: 92,
}

func isModifierVK(vk uint32) bool {
	_, ok := modifierVKs[vk]
	return ok
}

// isPartOfAnyShortcut 判断当前按住的键是否全部属于某个已配置快捷键的组合
// （即用户正在按该快捷键的前缀过程；完整组合已在 onKeysChanged 中处理）
func isPartOfAnyShortcut() bool {
	globalManager.shortcutsMu.RLock()
	defer globalManager.shortcutsMu.RUnlock()
	if len(globalManager.heldKeys) == 0 {
		return false
	}
	for _, binding := range globalManager.Shortcuts {
		if binding.ComboID == "" {
			continue
		}
		if comboCovers(binding.ComboID, globalManager.heldKeys) {
			return true
		}
	}
	return false
}

// comboCovers 判断 held 中的每个键是否都属于 comboID（"74+162" 形式）所表示的组合
func comboCovers(comboID string, held map[uint32]bool) bool {
	var comboSet map[uint32]struct{}
	for _, part := range strings.Split(comboID, "+") {
		vk, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return false
		}
		if comboSet == nil {
			comboSet = make(map[uint32]struct{})
		}
		comboSet[uint32(vk)] = struct{}{}
	}
	if comboSet == nil {
		return false
	}
	for vk := range held {
		if _, ok := comboSet[vk]; !ok {
			return false
		}
	}
	return true
}

// swallowedModInfo 被拦截的修饰键的扫描码信息（用于 SendInput 补发）
type swallowedModInfo struct {
	scan byte
	ext  bool
}

// hasSwallowedModifier 是否有被拦截且尚未补发的修饰键处于按下状态
func (m *Manager) hasSwallowedModifier() bool {
	return len(m.swallowedMods) > 0
}

// reinjectSwallowedModifiers 把被拦截的修饰键用 SendInput 补发回去（系统快捷键依赖它们）
func (m *Manager) reinjectSwallowedModifiers() {
	vks := make([]int, 0, len(m.swallowedMods))
	for vk := range m.swallowedMods {
		vks = append(vks, int(vk))
	}
	sort.Ints(vks)
	for _, vkInt := range vks {
		vk := uint32(vkInt)
		info := m.swallowedMods[vk]
		flags := uint32(0)
		if info.ext {
			flags = platform.KEYEVENTF_EXTENDEDKEY
		}
		platform.KeybdEvent(byte(vk), info.scan, flags, 0)
		// 补发后修饰键对系统可见，其物理 keyup 需要正常放行，从吞键表移除
		delete(m.swallowedDown, vk)
		delete(m.swallowedMods, vk)
	}
}

// injectKey 用 SendInput 补发一个按键按下（注入事件会被上方 INJECTED 检查直接放行）
func (m *Manager) injectKey(vk uint32, scan byte, ext bool) {
	flags := uint32(0)
	if ext {
		flags = platform.KEYEVENTF_EXTENDEDKEY
	}
	platform.KeybdEvent(byte(vk), scan, flags, 0)
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if globalManager == nil {
		return 0
	}
	if nCode >= 0 {
		mouseStruct := (*platform.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		var vkCode uint32
		isDown := false
		isUp := false

		switch wParam {
		case platform.WM_XBUTTONDOWN:
			isDown = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		case platform.WM_XBUTTONUP:
			isUp = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		}

		if vkCode != 0 {
			if isDown {
				globalManager.heldKeys[vkCode] = true
				if onKeysChanged() {
					return 1
				}
			} else if isUp {
				delete(globalManager.heldKeys, vkCode)
				// 录制模式下，松开按键也要检查是否结束录制
				if globalManager.recordingKeyFor != "" {
					if len(globalManager.heldKeys) == 0 {
						finishRecording()
					}
					return 1
				}
			}
		}
	}
	return platform.CallNextHookEx(globalManager.hMouseHook, nCode, wParam, lParam)
}

func onKeysChanged() bool {
	if globalManager == nil {
		return false
	}

	// --- 录制模式 ---
	if globalManager.recordingKeyFor != "" {
		// 更新最大按键组合
		if len(globalManager.heldKeys) >= len(globalManager.maxComboKeys) {
			globalManager.maxComboKeys = make(map[uint32]bool)
			for k, v := range globalManager.heldKeys {
				globalManager.maxComboKeys[k] = v
			}
		}

		// 实时发给前端显示
		readableName := GetReadableName(globalManager.maxComboKeys)
		if globalManager.OnRecord != nil {
			globalManager.OnRecord(globalManager.recordingKeyFor, readableName, GetComboID(globalManager.maxComboKeys))
		}
		return true // 吞掉按键
	}

	// --- 正常模式 ---
	// 将当前按下的所有键生成 ID，去配置里查（Shortcuts 可能被配置订阅并发替换，须持锁）
	currentComboID := GetComboID(globalManager.heldKeys)
	globalManager.shortcutsMu.RLock()
	var matchedAction string
	for action, savedComboID := range globalManager.Shortcuts {
		if savedComboID.ComboID == currentComboID {
			matchedAction = action
			break
		}
	}
	globalManager.shortcutsMu.RUnlock()

	if matchedAction != "" {
		// 检查是否包含 Alt 键 (VK_MENU=18, VK_LMENU=164, VK_RMENU=165)
		// 或者 Win 键 (VK_LWIN=91, VK_RWIN=92)
		// 如果包含这些键，且其他键被我们吞掉了，Windows 会认为用户只按了 Alt/Win，从而激活菜单栏或开始菜单
		hasAlt := globalManager.heldKeys[18] || globalManager.heldKeys[164] || globalManager.heldKeys[165]
		hasWin := globalManager.heldKeys[91] || globalManager.heldKeys[92]

		if hasAlt || hasWin {
			// 模拟按下并松开 Ctrl 键，防止 Windows 激活菜单栏/开始菜单
			// VK_CONTROL = 0x11, KEYEVENTF_KEYUP = 0x0002
			platform.KeybdEvent(platform.VK_CONTROL, 0, 0, 0)
			platform.KeybdEvent(platform.VK_CONTROL, 0, 2, 0)
		}

		if globalManager.OnTrigger != nil {
			globalManager.OnTrigger(matchedAction)
		}
		return true // 吞掉按键
	}
	return false
}

func finishRecording() {
	if globalManager == nil || globalManager.recordingKeyFor == "" {
		return
	}

	// 如果没有按任何键（比如直接点击录制然后点别的），忽略
	if len(globalManager.maxComboKeys) == 0 {
		globalManager.recordingKeyFor = ""
		return
	}

	comboID := GetComboID(globalManager.maxComboKeys)
	readableName := GetReadableName(globalManager.maxComboKeys)
	action := globalManager.recordingKeyFor

	// 退出录制模式
	globalManager.recordingKeyFor = ""
	globalManager.maxComboKeys = nil

	// 异步调用回调，避免阻塞 Hook 线程
	go func() {
		if globalManager.OnRecordingComplete != nil {
			globalManager.OnRecordingComplete(action, readableName, comboID)
		}
	}()
}
