package util

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procMoveWindow               = user32.NewProc("MoveWindow")
	procPostMessage              = user32.NewProc("PostMessageW")
	procIsWindow                 = user32.NewProc("IsWindow")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShowWindow               = user32.NewProc("ShowWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSendInput                = user32.NewProc("SendInput")
	procSetCursorPos             = user32.NewProc("SetCursorPos")
	procClientToScreen           = user32.NewProc("ClientToScreen")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSetFocus                 = user32.NewProc("SetFocus")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

// POINT 结构体
type POINT struct {
	X, Y int32
}

// INPUT 结构体 (必须是28字节)
type INPUT struct {
	Type uint32
	_    [4]byte // 对齐填充
	Mi   MOUSEINPUT
}

// MOUSEINPUT 结构体 (20字节)
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// 输入类型常量
const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1
)

// 鼠标事件标志
const (
	MOUSEEVENTF_MOVE       = 0x0001
	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
	MOUSEEVENTF_ABSOLUTE   = 0x8000
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
// @description: 向指定窗口发送鼠标点击消息（窗口内坐标）- 旧版PostMessage方法（不推荐用于游戏）
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

// ClientToScreen
// @author: [Fantasia](https://www.npc0.com)
// @function: ClientToScreen
// @description: 将窗口客户区坐标转换为屏幕坐标
// @param: hwnd syscall.Handle 窗口句柄, x, y int 客户区坐标
// @return: screenX, screenY int 屏幕坐标, success bool
func ClientToScreen(hwnd syscall.Handle, x, y int) (int, int, bool) {
	point := POINT{X: int32(x), Y: int32(y)}
	ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&point)))
	return int(point.X), int(point.Y), ret != 0
}

// SetCursorPos
// @author: [Fantasia](https://www.npc0.com)
// @function: SetCursorPos
// @description: 设置光标位置（屏幕坐标）
// @param: x, y int 屏幕坐标
// @return: bool
func SetCursorPos(x, y int) bool {
	ret, _, _ := procSetCursorPos.Call(uintptr(x), uintptr(y))
	return ret != 0
}

// GetCursorPos
// @author: [Fantasia](https://www.npc0.com)
// @function: GetCursorPos
// @description: 获取当前光标位置（屏幕坐标）
// @return: x, y int 屏幕坐标, success bool
func GetCursorPos() (int, int, bool) {
	var point POINT
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	return int(point.X), int(point.Y), ret != 0
}

// SendInput
// @author: [Fantasia](https://www.npc0.com)
// @function: SendInput
// @description: 发送硬件级别的输入事件
// @param: inputs []INPUT 输入事件数组
// @return: uint32 实际发送成功的事件数量
func SendInput(inputs []INPUT) uint32 {
	ret, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)

	// 调试信息
	if ret == 0 {
		fmt.Printf("[DEBUG] SendInput失败: %v, 结构体大小: %d 字节\n", err, unsafe.Sizeof(inputs[0]))
	} else {
		fmt.Printf("[DEBUG] SendInput成功: 发送了 %d/%d 个事件\n", ret, len(inputs))
	}

	return uint32(ret)
}

// MouseClick
// @author: [Fantasia](https://www.npc0.com)
// @function: MouseClick
// @description: 在当前鼠标位置执行鼠标点击（硬件级别）
// @return: bool
func MouseClick() bool {
	inputs := []INPUT{
		// 按下左键
		{
			Type: INPUT_MOUSE,
			Mi: MOUSEINPUT{
				Dx:          0,
				Dy:          0,
				MouseData:   0,
				DwFlags:     MOUSEEVENTF_LEFTDOWN,
				Time:        0,
				DwExtraInfo: 0,
			},
		},
		// 释放左键
		{
			Type: INPUT_MOUSE,
			Mi: MOUSEINPUT{
				Dx:          0,
				Dy:          0,
				MouseData:   0,
				DwFlags:     MOUSEEVENTF_LEFTUP,
				Time:        0,
				DwExtraInfo: 0,
			},
		},
	}

	sent := SendInput(inputs)
	return sent == uint32(len(inputs))
}

// GetWindowThreadProcessId
// @author: [Fantasia](https://www.npc0.com)
// @function: GetWindowThreadProcessId
// @description: 获取窗口所属的线程ID和进程ID
// @param: hwnd syscall.Handle 窗口句柄
// @return: threadId, processId uint32
func GetWindowThreadProcessId(hwnd syscall.Handle) (uint32, uint32) {
	var processId uint32
	threadId, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processId)),
	)
	return uint32(threadId), processId
}

// GetCurrentThreadId
// @author: [Fantasia](https://www.npc0.com)
// @function: GetCurrentThreadId
// @description: 获取当前线程ID
// @return: uint32
func GetCurrentThreadId() uint32 {
	ret, _, _ := procGetCurrentThreadId.Call()
	return uint32(ret)
}

// AttachThreadInput
// @author: [Fantasia](https://www.npc0.com)
// @function: AttachThreadInput
// @description: 附加或分离两个线程的输入处理
// @param: idAttach, idAttachTo uint32 线程ID, fAttach bool 是否附加
// @return: bool
func AttachThreadInput(idAttach, idAttachTo uint32, fAttach bool) bool {
	attach := 0
	if fAttach {
		attach = 1
	}
	ret, _, _ := procAttachThreadInput.Call(
		uintptr(idAttach),
		uintptr(idAttachTo),
		uintptr(attach),
	)
	return ret != 0
}

// SetFocus
// @author: [Fantasia](https://www.npc0.com)
// @function: SetFocus
// @description: 设置窗口焦点
// @param: hwnd syscall.Handle 窗口句柄
// @return: bool
func SetFocus(hwnd syscall.Handle) bool {
	ret, _, _ := procSetFocus.Call(uintptr(hwnd))
	return ret != 0
}

// ClickWindowPosition
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindowPosition
// @description: 点击窗口指定位置（使用硬件级别输入，适用于游戏窗口）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: error
func ClickWindowPosition(hwnd syscall.Handle, x, y int) error {
	fmt.Printf("[DEBUG] 开始点击流程 - 窗口内坐标: (%d, %d)\n", x, y)

	// 1. 确保窗口可见且未最小化
	if IsIconic(hwnd) {
		fmt.Println("[DEBUG] 窗口已最小化，正在恢复...")
		ShowWindow(hwnd, 9) // SW_RESTORE = 9
		time.Sleep(100 * time.Millisecond)
	}

	// 2. 将窗口置于前台（SendInput需要窗口具有焦点）
	fmt.Println("[DEBUG] 设置窗口为前台窗口...")
	BringWindowToTop(hwnd)
	SetForegroundWindow(hwnd)
	time.Sleep(100 * time.Millisecond)

	// 3. 附加线程输入以确保焦点设置生效（对游戏窗口很重要）
	windowThreadId, _ := GetWindowThreadProcessId(hwnd)
	currentThreadId := GetCurrentThreadId()
	fmt.Printf("[DEBUG] 窗口线程ID: %d, 当前线程ID: %d\n", windowThreadId, currentThreadId)

	if windowThreadId != currentThreadId && windowThreadId != 0 {
		fmt.Println("[DEBUG] 附加线程输入...")
		if AttachThreadInput(currentThreadId, windowThreadId, true) {
			defer AttachThreadInput(currentThreadId, windowThreadId, false)
			time.Sleep(50 * time.Millisecond)
		}
	}

	// 4. 设置焦点到窗口
	fmt.Println("[DEBUG] 设置窗口焦点...")
	SetFocus(hwnd)
	time.Sleep(50 * time.Millisecond)

	// 5. 将客户区坐标转换为屏幕坐标
	screenX, screenY, success := ClientToScreen(hwnd, x, y)
	if !success {
		return fmt.Errorf("坐标转换失败")
	}
	fmt.Printf("[DEBUG] 坐标转换成功 - 屏幕坐标: (%d, %d)\n", screenX, screenY)

	// 6. 移动鼠标到目标位置
	if !SetCursorPos(screenX, screenY) {
		return fmt.Errorf("移动鼠标失败")
	}
	fmt.Printf("[DEBUG] 鼠标已移动到屏幕坐标: (%d, %d)\n", screenX, screenY)

	// 验证鼠标位置
	curX, curY, _ := GetCursorPos()
	fmt.Printf("[DEBUG] 验证鼠标当前位置: (%d, %d)\n", curX, curY)

	// 短暂延迟，让鼠标移动生效
	time.Sleep(100 * time.Millisecond)

	// 7. 执行鼠标点击（硬件级别）
	fmt.Println("[DEBUG] 准备执行硬件级别点击...")
	if !MouseClick() {
		// SendInput失败，尝试备选方案：使用PostMessage
		fmt.Println("[WARN] SendInput点击失败，尝试使用PostMessage备选方案...")
		return ClickWindowPositionFallback(hwnd, x, y)
	}

	fmt.Println("[DEBUG] 硬件级别点击成功")
	return nil
}

// ClickWindowPositionFallback
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickWindowPositionFallback
// @description: 点击窗口指定位置的备选方案（使用PostMessage）
// @param: hwnd syscall.Handle 窗口句柄, x, y int 窗口内坐标
// @return: error
func ClickWindowPositionFallback(hwnd syscall.Handle, x, y int) error {
	fmt.Printf("[DEBUG] 使用PostMessage备选方案点击坐标: (%d, %d)\n", x, y)

	const (
		WM_LBUTTONDOWN = 0x0201
		WM_LBUTTONUP   = 0x0202
		MK_LBUTTON     = 0x0001
	)

	// 将坐标编码为LPARAM
	lParam := uintptr(x) | (uintptr(y) << 16)

	// 发送鼠标按下消息
	ret1, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONDOWN,
		MK_LBUTTON,
		lParam,
	)

	time.Sleep(30 * time.Millisecond)

	// 发送鼠标释放消息
	ret2, _, _ := procPostMessage.Call(
		uintptr(hwnd),
		WM_LBUTTONUP,
		0,
		lParam,
	)

	if ret1 != 0 && ret2 != 0 {
		fmt.Println("[DEBUG] PostMessage点击成功")
		return nil
	}

	return fmt.Errorf("PostMessage点击也失败了")
}
