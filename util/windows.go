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
	// procSendMessage 已在 input_methods.go 中声明，这里不重复声明
	procSendInput        = user32.NewProc("SendInput")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procClientToScreen   = user32.NewProc("ClientToScreen")
	procIsWindow         = user32.NewProc("IsWindow")
	procIsWindowVisible  = user32.NewProc("IsWindowVisible")
	procIsIconic         = user32.NewProc("IsIconic")
	procShowWindow       = user32.NewProc("ShowWindow")
	procSetActiveWindow  = user32.NewProc("SetActiveWindow")
	procBringWindowToTop = user32.NewProc("BringWindowToTop")
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

// POINT 结构体用于坐标转换
type POINT struct {
	X, Y int32
}

// ClientToScreen
// @author: [Fantasia](https://www.npc0.com)
// @function: ClientToScreen
// @description: 将窗口客户区坐标转换为屏幕坐标
// @param: hwnd syscall.Handle 窗口句柄, x, y int 客户区坐标
// @return: (int, int) 屏幕坐标
func ClientToScreen(hwnd syscall.Handle, x, y int) (int, int) {
	pt := POINT{X: int32(x), Y: int32(y)}
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

// GetCursorPos
// @author: [Fantasia](https://www.npc0.com)
// @function: GetCursorPos
// @description: 获取当前鼠标光标位置
// @return: (int, int) 屏幕坐标
func GetCursorPos() (int, int) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

// SetCursorPos
// @author: [Fantasia](https://www.npc0.com)
// @function: SetCursorPos
// @description: 设置鼠标光标位置
// @param: x, y int 屏幕坐标
// @return: bool
func SetCursorPos(x, y int) bool {
	ret, _, _ := procSetCursorPos.Call(uintptr(x), uintptr(y))
	return ret != 0
}

// MOUSEINPUT 结构体
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KEYBDINPUT 结构体
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
	Unused      [8]byte
}

// INPUT 结构体
type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
	_    [8]byte // padding to match the largest union member
}

// ClickWindowUsingSendMessage
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindowUsingSendMessage
// @description: 使用SendMessage向窗口发送鼠标点击消息（同步方式）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func ClickWindowUsingSendMessage(hwnd syscall.Handle, x, y int) bool {
	const (
		WM_MOUSEMOVE   = 0x0200
		WM_LBUTTONDOWN = 0x0201
		WM_LBUTTONUP   = 0x0202
	)

	// 将坐标打包到lParam中
	lParam := uintptr((uint32(y) << 16) | (uint32(x) & 0xFFFF))

	// 先发送鼠标移动消息
	procSendMessage.Call(
		uintptr(hwnd),
		WM_MOUSEMOVE,
		0,
		lParam,
	)

	// 发送鼠标按下消息
	ret1, _, _ := procSendMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONDOWN,
		1, // MK_LBUTTON
		lParam,
	)

	// 发送鼠标释放消息
	ret2, _, _ := procSendMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONUP,
		0,
		lParam,
	)

	success := ret1 != 0 && ret2 != 0
	if success {
		fmt.Printf("[ClickWindowUsingSendMessage] 点击成功: 坐标 (%d, %d)\n", x, y)
	} else {
		fmt.Printf("[ClickWindowUsingSendMessage] 点击失败: 坐标 (%d, %d)\n", x, y)
	}
	return success
}

// ClickWindowUsingInput
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindowUsingInput
// @description: 使用SendInput模拟物理鼠标点击（最接近真实用户操作）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func ClickWindowUsingInput(hwnd syscall.Handle, x, y int) bool {
	const (
		INPUT_MOUSE          = 0
		MOUSEEVENTF_MOVE     = 0x0001
		MOUSEEVENTF_ABSOLUTE = 0x8000
		MOUSEEVENTF_LEFTDOWN = 0x0002
		MOUSEEVENTF_LEFTUP   = 0x0004
	)

	// 将窗口坐标转换为屏幕坐标
	screenX, screenY := ClientToScreen(hwnd, x, y)

	// 保存当前鼠标位置
	oldX, oldY := GetCursorPos()

	// 移动鼠标到目标位置
	if !SetCursorPos(screenX, screenY) {
		fmt.Printf("[ClickWindowUsingInput] 移动鼠标失败: 目标坐标 (%d, %d)\n", screenX, screenY)
		return false
	}

	// 创建鼠标按下事件
	inputDown := INPUT{
		Type: INPUT_MOUSE,
		Mi: MOUSEINPUT{
			Dx:        0,
			Dy:        0,
			MouseData: 0,
			DwFlags:   MOUSEEVENTF_LEFTDOWN,
			Time:      0,
		},
	}

	// 创建鼠标释放事件
	inputUp := INPUT{
		Type: INPUT_MOUSE,
		Mi: MOUSEINPUT{
			Dx:        0,
			Dy:        0,
			MouseData: 0,
			DwFlags:   MOUSEEVENTF_LEFTUP,
			Time:      0,
		},
	}

	// 发送鼠标按下事件
	ret1, _, _ := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&inputDown)),
		uintptr(unsafe.Sizeof(inputDown)),
	)

	// 发送鼠标释放事件
	ret2, _, _ := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&inputUp)),
		uintptr(unsafe.Sizeof(inputUp)),
	)

	// 恢复鼠标位置（可选）
	// SetCursorPos(oldX, oldY)

	success := ret1 != 0 && ret2 != 0
	if success {
		fmt.Printf("[ClickWindowUsingInput] 物理点击成功: 窗口坐标 (%d, %d) -> 屏幕坐标 (%d, %d), 原鼠标位置 (%d, %d)\n",
			x, y, screenX, screenY, oldX, oldY)
	} else {
		fmt.Printf("[ClickWindowUsingInput] 物理点击失败: ret1=%d, ret2=%d\n", ret1, ret2)
	}
	return success
}

// ClickWindowEnhanced
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindowEnhanced
// @description: 增强版窗口点击，尝试多种方式确保点击成功
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: bool
func ClickWindowEnhanced(hwnd syscall.Handle, x, y int) bool {
	// 1. 确保窗口可见并激活
	if !EnsureWindowVisible(hwnd) {
		fmt.Printf("[ClickWindowEnhanced] 窗口不可见，无法点击\n")
		return false
	}

	// 2. 将窗口置于前台
	SetForegroundWindow(hwnd)
	BringWindowToTop(hwnd)

	// 短暂延迟，等待窗口激活
	// time.Sleep(50 * time.Millisecond) // 如果需要可以取消注释

	fmt.Printf("[ClickWindowEnhanced] 尝试点击坐标 (%d, %d)...\n", x, y)

	// 3. 首先尝试使用 SendInput（最可靠，模拟真实鼠标操作）
	if ClickWindowUsingInput(hwnd, x, y) {
		fmt.Printf("[ClickWindowEnhanced] SendInput 方式成功\n")
		return true
	}

	// 4. 如果 SendInput 失败，尝试 SendMessage
	if ClickWindowUsingSendMessage(hwnd, x, y) {
		fmt.Printf("[ClickWindowEnhanced] SendMessage 方式成功\n")
		return true
	}

	// 5. 最后尝试 PostMessage（原有方式）
	if ClickWindow(hwnd, x, y) {
		fmt.Printf("[ClickWindowEnhanced] PostMessage 方式成功\n")
		return true
	}

	fmt.Printf("[ClickWindowEnhanced] 所有点击方式均失败\n")
	return false
}
