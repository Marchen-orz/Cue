//go:build windows

package screen

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"

	"Q-Solver/pkg/platform"
)

// Region 框选区域（虚拟屏幕物理像素坐标）
type Region struct {
	X, Y, W, H int
}

// Empty 区域是否为空（未设置或已取消）
func (r Region) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// Normalized 返回规范化（左上/右下）后的矩形
func (r Region) Normalized() Region {
	x, y, w, h := r.X, r.Y, r.W, r.H
	if w < 0 {
		x += w
		w = -w
	}
	if h < 0 {
		y += h
		h = -h
	}
	return Region{X: x, Y: y, W: w, H: h}
}

// 框选遮罩窗状态（消息循环与窗口过程运行在同一线程，无需加锁）
var selectorState struct {
	hwnd      uintptr
	selecting bool
	startX    int // 拖拽起点（虚拟屏幕坐标）
	startY    int
	curX      int // 当前光标（虚拟屏幕坐标）
	curY      int
	done      bool
	resultCh  chan Region
}

var (
	selectActive       atomic.Bool
	selectorClassUTF16 = func() *uint16 {
		p, _ := syscall.UTF16PtrFromString("QSolverRegionSelector")
		return p
	}()
	selectorWndProcPtr = syscall.NewCallback(selectorWndProc)
)

// 虚拟屏幕原点（窗口客户区坐标 -> 虚拟屏幕坐标的偏移）
var vsX, vsY int32

// SelectRegion 打开全屏框选遮罩，阻塞直到用户完成拖选或取消。
// 返回的 Region 为虚拟屏幕物理像素坐标；取消时返回空 Region。
func SelectRegion() (Region, error) {
	if !selectActive.CompareAndSwap(false, true) {
		return Region{}, fmt.Errorf("框选已在进行中")
	}
	defer selectActive.Store(false)

	resultCh := make(chan Region, 1)
	go runSelector(resultCh)

	region := <-resultCh
	if region.Empty() {
		return Region{}, fmt.Errorf("已取消框选")
	}
	return region.Normalized(), nil
}

// runSelector 在独立系统线程上运行遮罩窗与消息循环
func runSelector(resultCh chan Region) {
	// 窗口消息循环必须固定在一条系统线程上
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	selectorState.resultCh = resultCh
	selectorState.done = false
	selectorState.selecting = false

	hInstance := platform.GetModuleHandle("")

	var wc platform.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.Style = platform.CS_HREDRAW | platform.CS_VREDRAW
	wc.LpfnWndProc = selectorWndProcPtr
	wc.HInstance = syscall.Handle(hInstance)
	wc.HCursor = syscall.Handle(platform.LoadCursorW(0, platform.IDC_CROSS))
	wc.LpszClassName = uintptr(unsafe.Pointer(selectorClassUTF16))

	// 每次框选都重新注册窗口类：上一次会话退出时未 UnregisterClass，
	// 同名类已存在，RegisterClassExW 会失败返回 0——忽略该错误即可
	// （类仍是上一次注册的同一窗口过程，行为一致）
	platform.RegisterClassExW(&wc)

	vsX = platform.GetSystemMetrics(platform.SM_XVIRTUALSCREEN)
	vsY = platform.GetSystemMetrics(platform.SM_YVIRTUALSCREEN)
	vsW := platform.GetSystemMetrics(platform.SM_CXVIRTUALSCREEN)
	vsH := platform.GetSystemMetrics(platform.SM_CYVIRTUALSCREEN)

	hwnd := platform.CreateWindowExW(
		platform.WS_EX_LAYERED|platform.WS_EX_TOPMOST|platform.WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(selectorClassUTF16)),
		0,
		platform.WS_POPUP,
		vsX, vsY, vsW, vsH,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		selectorState.resultCh <- Region{}
		return
	}
	selectorState.hwnd = hwnd

	// 半透明黑遮罩（整体 40% 不透明度，选框在 WM_PAINT 中自绘）
	platform.SetLayeredWindowAttributes(hwnd, 0, 100, platform.LWA_ALPHA)

	platform.ShowWindow(hwnd, platform.SW_SHOW)
	platform.SetForegroundWindow(hwnd)

	// 消息循环：排队消息（鼠标/键盘/WM_PAINT）必须经 DispatchMessage
	// 分发到窗口过程，缺少分发会导致遮罩完全无响应
	var msg platform.MSG
	for {
		n := platform.GetMessage(&msg, 0, 0, 0)
		if n <= 0 {
			break
		}
		platform.TranslateMessage(&msg)
		platform.DispatchMessage(&msg)
	}
}

// selectorResult 汇总并投递结果（只投一次）
func selectorResult(region Region) {
	if selectorState.done {
		return
	}
	selectorState.done = true
	selectorState.resultCh <- region
}

// toVirtualX/Y 客户区坐标 -> 虚拟屏幕坐标
func toVirtualX(clientX int32) int { return int(vsX + clientX) }
func toVirtualY(clientY int32) int { return int(vsY + clientY) }

// selectorWndProc 遮罩窗窗口过程
func selectorWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case platform.WM_LBUTTONDOWN:
		selectorState.selecting = true
		selectorState.startX = toVirtualX(loSigned(lParam))
		selectorState.startY = toVirtualY(hiSigned(lParam))
		selectorState.curX, selectorState.curY = selectorState.startX, selectorState.startY
		platform.SetCapture(hwnd)
		return 0

	case platform.WM_MOUSEMOVE:
		if selectorState.selecting {
			selectorState.curX = toVirtualX(loSigned(lParam))
			selectorState.curY = toVirtualY(hiSigned(lParam))
			platform.InvalidateRect(hwnd, nil, false)
		}
		return 0

	case platform.WM_LBUTTONUP:
		if !selectorState.selecting {
			return 0
		}
		selectorState.selecting = false
		platform.ReleaseCapture()
		selectorState.curX = toVirtualX(loSigned(lParam))
		selectorState.curY = toVirtualY(hiSigned(lParam))

		region := Region{
			X: selectorState.startX, Y: selectorState.startY,
			W: selectorState.curX - selectorState.startX,
			H: selectorState.curY - selectorState.startY,
		}.Normalized()
		if region.W < 3 || region.H < 3 {
			// 过小的选区视为误触，按取消处理
			selectorResult(Region{})
		} else {
			selectorResult(region)
		}
		platform.DestroyWindow(hwnd)
		return 0

	case platform.WM_RBUTTONDOWN:
		selectorResult(Region{})
		platform.DestroyWindow(hwnd)
		return 0

	case platform.WM_KEYDOWN:
		if wParam == platform.VK_ESCAPE {
			selectorResult(Region{})
			platform.DestroyWindow(hwnd)
			return 0
		}

	case platform.WM_PAINT:
		paintOverlay(hwnd)
		return 0

	case platform.WM_ERASEBKGND:
		return 1

	case platform.WM_CLOSE:
		selectorResult(Region{})
		platform.DestroyWindow(hwnd)
		return 0

	case platform.WM_DESTROY:
		selectorResult(Region{})
		platform.PostQuitMessage(0)
		return 0
	}

	return platform.DefWindowProcW(hwnd, msg, wParam, lParam)
}

// paintOverlay 绘制遮罩与选框：整体暗色，选区高亮，白色边框
func paintOverlay(hwnd uintptr) {
	var ps platform.PAINTSTRUCT
	hdc := platform.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer platform.EndPaint(hwnd, &ps)

	vsW := platform.GetSystemMetrics(platform.SM_CXVIRTUALSCREEN)
	vsH := platform.GetSystemMetrics(platform.SM_CYVIRTUALSCREEN)
	full := platform.RECT{Left: 0, Top: 0, Right: int32(vsW), Bottom: int32(vsH)}

	// 全屏暗色底（窗口级 alpha 再叠加）
	darkBrush := platform.CreateSolidBrush(0x00000000)
	platform.FillRect(hdc, &full, darkBrush)
	platform.DeleteObject(darkBrush)

	if !selectorState.selecting {
		return
	}

	// 选区矩形（客户区坐标）
	x1, y1 := int32(selectorState.startX-int(vsX)), int32(selectorState.startY-int(vsY))
	x2, y2 := int32(selectorState.curX-int(vsX)), int32(selectorState.curY-int(vsY))
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	selRect := platform.RECT{Left: x1, Top: y1, Right: x2, Bottom: y2}

	// 选区内浅色提亮
	innerBrush := platform.CreateSolidBrush(0x00303030)
	platform.FillRect(hdc, &selRect, innerBrush)
	platform.DeleteObject(innerBrush)

	// 白色边框（FrameRect 用白画刷描边）
	whiteBrush := platform.CreateSolidBrush(0x00FFFFFF)
	platform.FrameRect(hdc, &selRect, whiteBrush)
	platform.DeleteObject(whiteBrush)
}
// loSigned/hiSigned 从 lParam 提取有符号 16 位坐标
func loSigned(lp uintptr) int32 { return int32(int16(uint16(lp & 0xffff))) }
func hiSigned(lp uintptr) int32 { return int32(int16(uint16((lp >> 16) & 0xffff))) }
