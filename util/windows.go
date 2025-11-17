package util

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetClassName        = user32.NewProc("GetClassNameW")
	procGetWindowText       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procPostMessage         = user32.NewProc("PostMessageW")
	procIsWindow            = user32.NewProc("IsWindow")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
)

// FindWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: FindWindow
// @description: 查找窗口句柄
// @param: className, windowName string 类名和窗口名
// @return: syscall.Handle
func FindWindow(className, windowName string) syscall.Handle {
	lpClassName, e2 := syscall.UTF16PtrFromString(className)
	lpWindowName, e3 := syscall.UTF16PtrFromString(windowName)
	// 调用FindWindowW
	r0, _, e1 := procFindWindowW.Call(uintptr(unsafe.Pointer(lpClassName)), uintptr(unsafe.Pointer(lpWindowName)))
	fmt.Println(e1, e2, e3)
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

// GetForegroundWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: GetForegroundWindow
// @description: 获取前台窗口句柄
// @return: syscall.Handle
func GetForegroundWindow() syscall.Handle {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return syscall.Handle(hwnd)
}

// GetClassName
// @author: [Fantasia](https://www.npc0.com)
// @function: GetClassName
// @description: 获取句柄类名
// @return: syscall.Handle
func GetClassName(hwnd syscall.Handle) string {
	var className [256]uint16
	_, _, _ = procGetClassName.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&className[0])),
		uintptr(len(className)),
	)
	return syscall.UTF16ToString(className[:])
}

// GetWindowText
// @author: [Fantasia](https://www.npc0.com)
// @function: GetWindowText
// @description: 获取句柄标题
// @return: syscall.Handle
func GetWindowText(hwnd syscall.Handle) string {
	length, _, _ := procGetWindowTextLength.Call(uintptr(hwnd))
	var text [256]uint16
	_, _, _ = procGetWindowText.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&text[0])),
		uintptr(length+1),
	)
	return syscall.UTF16ToString(text[:])
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

// ClickWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindow
// @description: 向指定窗口发送鼠标点击消息（使用窗口内坐标）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func ClickWindow(hwnd syscall.Handle, x, y int) bool {
	const (
		WM_LBUTTONDOWN = 0x0201
		WM_LBUTTONUP   = 0x0202
	)

	// 将坐标打包到lParam中：lParam = (y << 16) | (x & 0xFFFF)
	lParam := uintptr((uint32(y) << 16) | (uint32(x) & 0xFFFF))

	// 发送鼠标按下消息
	ret1, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONDOWN,
		0, // wParam: MK_LBUTTON (0x0001) 可以设为0，因为PostMessage不需要
		lParam,
	)

	// 发送鼠标释放消息
	ret2, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONUP,
		0,
		lParam,
	)

	success := ret1 != 0 && ret2 != 0
	if success {
		fmt.Printf("[ClickWindow] 点击成功: 坐标 (%d, %d), WM_LBUTTONDOWN返回: %d, WM_LBUTTONUP返回: %d\n", x, y, ret1, ret2)
	} else {
		fmt.Printf("[ClickWindow] 点击失败: 坐标 (%d, %d), WM_LBUTTONDOWN返回: %d, WM_LBUTTONUP返回: %d\n", x, y, ret1, ret2)
	}
	return success
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

// EnsureWindowVisible
// @author: [Fantasia](https://www.npc0.com)
// @function: EnsureWindowVisible
// @description: 确保窗口可见且未最小化，如果不可见则尝试恢复
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool 是否成功
func EnsureWindowVisible(hwnd syscall.Handle) bool {
	if hwnd == 0 {
		return false
	}

	// 检查窗口句柄是否有效
	if !IsWindow(hwnd) {
		return false
	}

	// 如果窗口最小化，恢复窗口
	if IsIconic(hwnd) {
		const SW_RESTORE = 9
		ShowWindow(hwnd, SW_RESTORE)
	}

	// 如果窗口不可见，显示窗口
	if !IsWindowVisible(hwnd) {
		const SW_SHOW = 5
		ShowWindow(hwnd, SW_SHOW)
	}

	// 尝试将窗口置于前台
	SetForegroundWindow(hwnd)

	// 再次检查窗口是否可见
	return IsWindowVisible(hwnd) && !IsIconic(hwnd)
}
