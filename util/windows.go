package util

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procPostMessage         = user32.NewProc("PostMessageW")
	procIsWindow            = user32.NewProc("IsWindow")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
)

// FindWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: FindWindow
// @description: 查找窗口句柄
// @param: className, windowName string 类名和窗口名
// @return: syscall.Handle
func FindWindow(className, windowName string) syscall.Handle {
	lpClassName, _ := syscall.UTF16PtrFromString(className)
	lpWindowName, _ := syscall.UTF16PtrFromString(windowName)
	// 调用FindWindowW
	r0, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(lpClassName)), uintptr(unsafe.Pointer(lpWindowName)))
	return syscall.Handle(r0)
}

// SetForegroundWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: SetForegroundWindow
// @description: 设置窗口置顶
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func SetForegroundWindow(hwnd syscall.Handle) bool {
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	syscall.Syscall(procSetForegroundWindow.Addr(), 1, uintptr(hwnd), 0, 0)
	return ret != 0
}

// MoveWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: MoveWindow
// @description: 设置窗口大小并移动
// @param: hwnd syscall.Handle, x, y, width, height int
// @return: bool
func MoveWindow(hwnd syscall.Handle, x, y, width, height int) bool {
	ret, _, _ := procMoveWindow.Call(
		uintptr(hwnd),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(1),
	)
	return ret != 0
}

// SendKeyToWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: SendKeyToWindow
// @description: 向指定窗口发送按键消息
// @param: hwnd syscall.Handle 窗口句柄, vkCode uint16 虚拟键码
// @return: bool
func SendKeyToWindow(hwnd syscall.Handle, vkCode uint16) bool {
	const (
		WM_KEYDOWN = 0x0100
		WM_KEYUP   = 0x0101
	)

	// 发送按键按下消息
	ret1, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_KEYDOWN,
		uintptr(vkCode),
		0,
	)

	// 发送按键释放消息
	ret2, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_KEYUP,
		uintptr(vkCode),
		0,
	)

	return ret1 != 0 && ret2 != 0
}

// IsWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: IsWindow
// @description: 检查窗口句柄是否有效
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func IsWindow(hwnd syscall.Handle) bool {
	ret, _, _ := procIsWindow.Call(uintptr(hwnd))
	return ret != 0
}

// IsWindowVisible
// @author: [Fantasia](https://www.npc0.com)
// @function: IsWindowVisible
// @description: 检查窗口是否可见
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func IsWindowVisible(hwnd syscall.Handle) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

// IsIconic
// @author: [Fantasia](https://www.npc0.com)
// @function: IsIconic
// @description: 检查窗口是否最小化
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func IsIconic(hwnd syscall.Handle) bool {
	ret, _, _ := procIsIconic.Call(uintptr(hwnd))
	return ret != 0
}

// ShowWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: ShowWindow
// @description: 显示或隐藏窗口
// @param: hwnd syscall.Handle 窗口句柄, nCmdShow int 显示命令（SW_RESTORE=9, SW_SHOW=5）
// @return: bool
func ShowWindow(hwnd syscall.Handle, nCmdShow int) bool {
	ret, _, _ := procShowWindow.Call(uintptr(hwnd), uintptr(nCmdShow))
	return ret != 0
}

// BringWindowToTop
// @author: [Fantasia](https://www.npc0.com)
// @function: BringWindowToTop
// @description: 将窗口置于顶层
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func BringWindowToTop(hwnd syscall.Handle) bool {
	ret, _, _ := procBringWindowToTop.Call(uintptr(hwnd))
	return ret != 0
}

// SendMouseClickToWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: SendMouseClickToWindow
// @description: 向指定窗口发送鼠标点击消息（窗口内坐标）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func SendMouseClickToWindow(hwnd syscall.Handle, x, y int) bool {
	const (
		WM_LBUTTONDOWN = 0x0201
		WM_LBUTTONUP   = 0x0202
		MK_LBUTTON     = 0x0001
	)

	// 将坐标编码为LPARAM（低16位为x，高16位为y）
	lParam := uintptr(x) | (uintptr(y) << 16)

	// 发送鼠标按下消息
	ret1, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONDOWN,
		MK_LBUTTON,
		lParam,
	)

	// 发送鼠标释放消息
	ret2, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONUP,
		0,
		lParam,
	)

	return ret1 != 0 && ret2 != 0
}
