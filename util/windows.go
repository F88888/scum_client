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
