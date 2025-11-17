package util

import (
	"fmt"
	"syscall"
	"time"
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
	procClientToScreen      = user32.NewProc("ClientToScreen")
	procSetCursorPos        = user32.NewProc("SetCursorPos")
	procSendInput           = user32.NewProc("SendInput")
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

// POINT 结构体用于坐标转换
type POINT struct {
	X, Y int32
}

// INPUT 结构体用于 SendInput
type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
}

// MOUSEINPUT 鼠标输入结构体
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// ClickWindow
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindow
// @description: 向指定窗口发送鼠标点击消息（使用窗口内坐标，优先使用真实鼠标输入）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func ClickWindow(hwnd syscall.Handle, x, y int) bool {
	// 先尝试使用真实鼠标输入（适用于游戏窗口）
	if clickWindowWithRealInput(hwnd, x, y) {
		return true
	}

	// 如果真实输入失败，回退到窗口消息方式
	return clickWindowWithMessage(hwnd, x, y)
}

// clickWindowWithRealInput 使用真实鼠标输入进行点击（适用于游戏窗口）
// @description: 将窗口坐标转换为屏幕坐标，然后使用 SendInput 发送真实的鼠标点击事件
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func clickWindowWithRealInput(hwnd syscall.Handle, x, y int) bool {
	// 确保窗口可见并激活
	if !EnsureWindowVisible(hwnd) {
		fmt.Printf("[ClickWindow] 窗口不可见或无法激活\n")
		return false
	}

	// 再次确保窗口在前台（游戏窗口可能需要多次尝试）
	SetForegroundWindow(hwnd)
	time.Sleep(50 * time.Millisecond)

	// 将窗口客户区坐标转换为屏幕坐标
	var pt POINT
	pt.X = int32(x)
	pt.Y = int32(y)
	ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		fmt.Printf("[ClickWindow] 坐标转换失败\n")
		return false
	}

	screenX := int(pt.X)
	screenY := int(pt.Y)

	// 移动鼠标到目标位置
	ret, _, _ = procSetCursorPos.Call(uintptr(screenX), uintptr(screenY))
	if ret == 0 {
		fmt.Printf("[ClickWindow] 移动鼠标失败\n")
		return false
	}

	// 等待一小段时间确保鼠标移动完成
	time.Sleep(10 * time.Millisecond)

	// 使用 SendInput 发送鼠标按下和释放事件
	const (
		INPUT_MOUSE          = 0
		MOUSEEVENTF_LEFTDOWN = 0x0002
		MOUSEEVENTF_LEFTUP   = 0x0004
	)

	// 发送鼠标按下事件
	var inputDown INPUT
	inputDown.Type = INPUT_MOUSE
	inputDown.Mi.DwFlags = MOUSEEVENTF_LEFTDOWN
	ret1, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&inputDown)), uintptr(unsafe.Sizeof(INPUT{})))

	// 短暂延迟（模拟真实点击）
	time.Sleep(20 * time.Millisecond)

	// 发送鼠标释放事件
	var inputUp INPUT
	inputUp.Type = INPUT_MOUSE
	inputUp.Mi.DwFlags = MOUSEEVENTF_LEFTUP
	ret2, _, _ := procSendInput.Call(1, uintptr(unsafe.Pointer(&inputUp)), uintptr(unsafe.Sizeof(INPUT{})))

	success := ret1 != 0 && ret2 != 0
	if success {
		fmt.Printf("[ClickWindow] 真实鼠标点击成功: 窗口坐标 (%d, %d) -> 屏幕坐标 (%d, %d)\n", x, y, screenX, screenY)
	} else {
		fmt.Printf("[ClickWindow] 真实鼠标点击失败: 窗口坐标 (%d, %d), SendInput返回: %d, %d\n", x, y, ret1, ret2)
	}
	return success
}

// clickWindowWithMessage 使用窗口消息进行点击（回退方案）
// @description: 使用 PostMessage 发送窗口消息（适用于普通窗口）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func clickWindowWithMessage(hwnd syscall.Handle, x, y int) bool {
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
		fmt.Printf("[ClickWindow] 窗口消息点击成功: 坐标 (%d, %d), WM_LBUTTONDOWN返回: %d, WM_LBUTTONUP返回: %d\n", x, y, ret1, ret2)
	} else {
		fmt.Printf("[ClickWindow] 窗口消息点击失败: 坐标 (%d, %d), WM_LBUTTONDOWN返回: %d, WM_LBUTTONUP返回: %d\n", x, y, ret1, ret2)
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
