package util

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
)

var (
	user32DLL = syscall.NewLazyDLL("user32.dll")
	ole32DLL  = syscall.NewLazyDLL("ole32.dll")

	// 窗口消息相关API
	procSendMessage = user32DLL.NewProc("SendMessageW")

	// 按键相关API
	procKeybd_event = user32DLL.NewProc("keybd_event")

	// UI自动化相关
	procCoInitialize   = ole32DLL.NewProc("CoInitialize")
	procCoUninitialize = ole32DLL.NewProc("CoUninitialize")
)

// Windows消息常量
const (
	WM_SETTEXT = 0x000C

	// 虚拟键码
	VK_RETURN  = 0x0D
	VK_ESCAPE  = 0x1B
	VK_T       = 0x54
	VK_SLASH   = 0xBF
	VK_CONTROL = 0x11
	VK_V       = 0x56
)

// InputMethod 输入方式枚举
type InputMethod int

const (
	INPUT_SIMULATE_KEY    InputMethod = iota // 模拟按键
	INPUT_WINDOW_MSG                         // 窗口消息
	INPUT_UI_AUTOMATION                      // UI自动化
	INPUT_CLIPBOARD_PASTE                    // 剪贴板粘贴
	INPUT_HYBRID                             // 混合模式（智能选择）
)

// ChatActivationMethod 聊天框激活方式
type ChatActivationMethod int

const (
	CHAT_ACTIVATE_T_KEY      ChatActivationMethod = iota // T键激活
	CHAT_ACTIVATE_SLASH_KEY                              // /键激活命令
	CHAT_ACTIVATE_WINDOW_MSG                             // 直接窗口消息
)

// EnhancedInputManager 增强型输入管理器
type EnhancedInputManager struct {
	hwnd            syscall.Handle
	chatInputHwnd   syscall.Handle
	lastInputMethod InputMethod
	fallbackChain   []InputMethod

	// 性能统计
	methodStats map[InputMethod]*MethodStats

	// 配置选项
	enableFallback bool
	timeoutMs      int
	retryCount     int
}

// MethodStats 方法统计信息
type MethodStats struct {
	SuccessCount int
	FailureCount int
	AverageTime  time.Duration
	LastUsed     time.Time
	Reliability  float64
}

// NewEnhancedInputManager 创建增强型输入管理器
func NewEnhancedInputManager(mainHwnd syscall.Handle) *EnhancedInputManager {
	eim := &EnhancedInputManager{
		hwnd:           mainHwnd,
		enableFallback: true,
		timeoutMs:      5000,
		retryCount:     3,
		methodStats:    make(map[InputMethod]*MethodStats),
		fallbackChain: []InputMethod{
			INPUT_CLIPBOARD_PASTE,
			INPUT_WINDOW_MSG,
			INPUT_SIMULATE_KEY,
			INPUT_UI_AUTOMATION,
		},
	}

	// 初始化统计信息
	for _, method := range eim.fallbackChain {
		eim.methodStats[method] = &MethodStats{
			Reliability: 1.0,
		}
	}

	// 查找聊天输入框句柄
	eim.findChatInputHandle()

	return eim
}

// findChatInputHandle 查找聊天输入框句柄
func (eim *EnhancedInputManager) findChatInputHandle() {
	// 通过EnumChildWindows遍历子窗口找到聊天输入框
	// 这里简化实现，实际应用中需要根据具体的窗口类名和ID查找
	eim.chatInputHwnd = eim.hwnd // 暂时使用主窗口句柄
}

// ActivateChat 激活聊天框（支持多种方式）
func (eim *EnhancedInputManager) ActivateChat(method ChatActivationMethod) error {
	start := time.Now()
	var err error

	// 设置窗口为前台 - 已注释：使用句柄操作不需要窗口置顶
	// SetForegroundWindow(eim.hwnd)
	// time.Sleep(100 * time.Millisecond)

	switch method {
	case CHAT_ACTIVATE_T_KEY:
		err = eim.activateChatWithTKey()
	case CHAT_ACTIVATE_SLASH_KEY:
		err = eim.activateChatWithSlashKey()
	case CHAT_ACTIVATE_WINDOW_MSG:
		err = eim.activateChatWithWindowMsg()
	default:
		err = eim.activateChatWithTKey() // 默认使用T键
	}

	if err == nil {
		fmt.Printf("聊天框激活成功，方式: %d，耗时: %v\n", method, time.Since(start))
	}

	return err
}

// activateChatWithTKey 使用T键激活聊天框
func (eim *EnhancedInputManager) activateChatWithTKey() error {
	// 方法1：使用robotgo
	err := robotgo.KeyTap("t")
	if err != nil {
		// 方法2：使用Windows API直接发送按键
		return eim.sendVirtualKey(VK_T)
	}
	return nil
}

// activateChatWithSlashKey 使用/键激活命令输入
func (eim *EnhancedInputManager) activateChatWithSlashKey() error {
	// 方法1：使用robotgo
	err := robotgo.KeyTap("/")
	if err != nil {
		// 方法2：使用Windows API
		return eim.sendVirtualKey(VK_SLASH)
	}
	return nil
}

// activateChatWithWindowMsg 使用窗口消息激活聊天框
func (eim *EnhancedInputManager) activateChatWithWindowMsg() error {
	if eim.chatInputHwnd == 0 {
		return fmt.Errorf("聊天输入框句柄未找到")
	}

	// 发送焦点消息到聊天输入框
	ret, _, _ := procSendMessage.Call(
		uintptr(eim.chatInputHwnd),
		uintptr(0x0007), // WM_SETFOCUS
		0,
		0,
	)

	if ret == 0 {
		return fmt.Errorf("发送焦点消息失败")
	}

	return nil
}

// sendVirtualKey 发送虚拟按键
func (eim *EnhancedInputManager) sendVirtualKey(vk uint8) error {
	// 按下
	procKeybd_event.Call(
		uintptr(vk),
		0,
		0,
		0,
	)

	// 释放
	procKeybd_event.Call(
		uintptr(vk),
		0,
		2, // KEYEVENTF_KEYUP
		0,
	)

	return nil
}

// SendText 发送文本（智能选择输入方式）
func (eim *EnhancedInputManager) SendText(text string, preferredMethod InputMethod) error {
	if text == "" {
		return fmt.Errorf("文本不能为空")
	}

	start := time.Now()
	var err error
	var methodUsed InputMethod

	// 如果指定了首选方法，先尝试首选方法
	if preferredMethod != INPUT_HYBRID {
		methodUsed = preferredMethod
		err = eim.sendTextWithMethod(text, preferredMethod)
	}

	// 如果首选方法失败且启用了回退机制，尝试回退方案
	if err != nil && eim.enableFallback {
		for _, method := range eim.fallbackChain {
			if method == preferredMethod {
				continue // 跳过已经尝试过的方法
			}

			fmt.Printf("尝试回退方案: %d\n", method)
			if err = eim.sendTextWithMethod(text, method); err == nil {
				methodUsed = method
				break
			}
		}
	}

	// 更新统计信息
	eim.updateMethodStats(methodUsed, time.Since(start), err == nil)

	if err == nil {
		eim.lastInputMethod = methodUsed
		fmt.Printf("文本发送成功，方式: %d，耗时: %v\n", methodUsed, time.Since(start))
	}

	return err
}

// sendTextWithMethod 使用指定方法发送文本
func (eim *EnhancedInputManager) sendTextWithMethod(text string, method InputMethod) error {
	switch method {
	case INPUT_SIMULATE_KEY:
		return eim.sendTextWithSimulateKey(text)
	case INPUT_WINDOW_MSG:
		return eim.sendTextWithWindowMsg(text)
	case INPUT_UI_AUTOMATION:
		return eim.sendTextWithUIAutomation(text)
	case INPUT_CLIPBOARD_PASTE:
		return eim.sendTextWithClipboardPaste(text)
	default:
		return fmt.Errorf("不支持的输入方法: %d", method)
	}
}

// sendTextWithSimulateKey 使用模拟按键发送文本
func (eim *EnhancedInputManager) sendTextWithSimulateKey(text string) error {
	// 清空输入框
	if err := eim.clearInputBox(); err != nil {
		return fmt.Errorf("清空输入框失败: %v", err)
	}

	// 逐字符输入
	for _, char := range text {
		if err := eim.typeCharacter(char); err != nil {
			return fmt.Errorf("输入字符失败: %v", err)
		}
		time.Sleep(5 * time.Millisecond) // 短暂延迟避免输入过快
	}

	return nil
}

// sendTextWithWindowMsg 使用窗口消息发送文本
func (eim *EnhancedInputManager) sendTextWithWindowMsg(text string) error {
	if eim.chatInputHwnd == 0 {
		return fmt.Errorf("聊天输入框句柄未找到")
	}

	// 转换为UTF16字符串
	utf16Text, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return fmt.Errorf("转换UTF16失败: %v", err)
	}

	// 使用WM_SETTEXT直接设置文本
	ret, _, _ := procSendMessage.Call(
		uintptr(eim.chatInputHwnd),
		WM_SETTEXT,
		0,
		uintptr(unsafe.Pointer(utf16Text)),
	)

	if ret == 0 {
		return fmt.Errorf("发送WM_SETTEXT消息失败")
	}

	return nil
}

// sendTextWithUIAutomation 使用UI自动化发送文本
func (eim *EnhancedInputManager) sendTextWithUIAutomation(text string) error {
	// 初始化COM
	procCoInitialize.Call(0)
	defer procCoUninitialize.Call()

	// 这里应该实现UI Automation API的调用
	// 由于实现较复杂，暂时返回未实现错误
	return fmt.Errorf("UI自动化方法暂未实现")
}

// sendTextWithClipboardPaste 使用剪贴板粘贴发送文本
func (eim *EnhancedInputManager) sendTextWithClipboardPaste(text string) error {
	// 备份当前剪贴板内容
	originalClipboard, _ := clipboard.ReadAll()

	// 清空输入框
	if err := eim.clearInputBox(); err != nil {
		return fmt.Errorf("清空输入框失败: %v", err)
	}

	// 将文本写入剪贴板
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("写入剪贴板失败: %v", err)
	}

	// 等待剪贴板操作完成
	time.Sleep(50 * time.Millisecond)

	// 模拟Ctrl+V粘贴
	if err := eim.simulateCtrlV(); err != nil {
		// 恢复剪贴板内容
		clipboard.WriteAll(originalClipboard)
		return fmt.Errorf("粘贴操作失败: %v", err)
	}

	// 验证粘贴是否成功
	time.Sleep(100 * time.Millisecond)

	// 恢复剪贴板内容
	clipboard.WriteAll(originalClipboard)

	return nil
}

// clearInputBox 清空输入框
func (eim *EnhancedInputManager) clearInputBox() error {
	// 方法1：Ctrl+A + Delete
	if err := eim.simulateCtrlA(); err == nil {
		time.Sleep(10 * time.Millisecond)
		return eim.sendVirtualKey(0x2E) // VK_DELETE
	}

	// 方法2：使用窗口消息清空
	if eim.chatInputHwnd != 0 {
		utf16Empty, _ := syscall.UTF16PtrFromString("")
		procSendMessage.Call(
			uintptr(eim.chatInputHwnd),
			WM_SETTEXT,
			0,
			uintptr(unsafe.Pointer(utf16Empty)),
		)
	}

	return nil
}

// simulateCtrlA 模拟Ctrl+A全选
func (eim *EnhancedInputManager) simulateCtrlA() error {
	// 按下Ctrl
	procKeybd_event.Call(uintptr(VK_CONTROL), 0, 0, 0)
	time.Sleep(10 * time.Millisecond)

	// 按下A
	procKeybd_event.Call(uintptr(0x41), 0, 0, 0) // VK_A
	time.Sleep(10 * time.Millisecond)

	// 释放A
	procKeybd_event.Call(uintptr(0x41), 0, 2, 0)
	time.Sleep(10 * time.Millisecond)

	// 释放Ctrl
	procKeybd_event.Call(uintptr(VK_CONTROL), 0, 2, 0)

	return nil
}

// simulateCtrlV 模拟Ctrl+V粘贴
func (eim *EnhancedInputManager) simulateCtrlV() error {
	// 按下Ctrl
	procKeybd_event.Call(uintptr(VK_CONTROL), 0, 0, 0)
	time.Sleep(10 * time.Millisecond)

	// 按下V
	procKeybd_event.Call(uintptr(VK_V), 0, 0, 0)
	time.Sleep(10 * time.Millisecond)

	// 释放V
	procKeybd_event.Call(uintptr(VK_V), 0, 2, 0)
	time.Sleep(10 * time.Millisecond)

	// 释放Ctrl
	procKeybd_event.Call(uintptr(VK_CONTROL), 0, 2, 0)

	return nil
}

// typeCharacter 输入单个字符
func (eim *EnhancedInputManager) typeCharacter(char rune) error {
	// 获取字符的虚拟键码
	vk := eim.charToVirtualKey(char)
	if vk == 0 {
		// 对于无法映射的字符，使用Unicode输入
		return eim.typeUnicodeChar(char)
	}

	// 检查是否需要Shift键
	needShift := eim.needShiftForChar(char)

	if needShift {
		// 按下Shift
		procKeybd_event.Call(uintptr(0x10), 0, 0, 0) // VK_SHIFT
		time.Sleep(5 * time.Millisecond)
	}

	// 按下字符键
	procKeybd_event.Call(uintptr(vk), 0, 0, 0)
	time.Sleep(5 * time.Millisecond)

	// 释放字符键
	procKeybd_event.Call(uintptr(vk), 0, 2, 0)

	if needShift {
		time.Sleep(5 * time.Millisecond)
		// 释放Shift
		procKeybd_event.Call(uintptr(0x10), 0, 2, 0)
	}

	return nil
}

// charToVirtualKey 字符转虚拟键码
func (eim *EnhancedInputManager) charToVirtualKey(char rune) uint8 {
	if char >= 'a' && char <= 'z' {
		return uint8(char - 'a' + 0x41) // A-Z
	}
	if char >= 'A' && char <= 'Z' {
		return uint8(char)
	}
	if char >= '0' && char <= '9' {
		return uint8(char)
	}

	// 特殊字符映射
	specialChars := map[rune]uint8{
		' ':  0x20, // VK_SPACE
		'#':  0x33, // '3' key (need shift for #)
		'/':  VK_SLASH,
		'.':  0xBE, // VK_OEM_PERIOD
		',':  0xBC, // VK_OEM_COMMA
		'-':  0xBD, // VK_OEM_MINUS
		'=':  0xBB, // VK_OEM_PLUS
		'[':  0xDB, // VK_OEM_4
		']':  0xDD, // VK_OEM_6
		'\\': 0xDC, // VK_OEM_5
		';':  0xBA, // VK_OEM_1
		'\'': 0xDE, // VK_OEM_7
		'`':  0xC0, // VK_OEM_3
	}

	if vk, exists := specialChars[char]; exists {
		return vk
	}

	return 0
}

// needShiftForChar 检查字符是否需要Shift键
func (eim *EnhancedInputManager) needShiftForChar(char rune) bool {
	shiftChars := "!@#$%^&*()_+{}|:\"<>?ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, c := range shiftChars {
		if c == char {
			return true
		}
	}
	return false
}

// typeUnicodeChar 输入Unicode字符
func (eim *EnhancedInputManager) typeUnicodeChar(char rune) error {
	// 使用Alt + 数字键盘输入Unicode字符
	// 这里简化实现，实际应该将Unicode转换为数字键序列
	// 对于复杂字符，建议使用剪贴板方式
	return fmt.Errorf("Unicode字符输入暂未实现: %c", char)
}

// SendEnter 发送回车键
func (eim *EnhancedInputManager) SendEnter() error {
	return eim.sendVirtualKey(VK_RETURN)
}

// SendEscape 发送ESC键
func (eim *EnhancedInputManager) SendEscape() error {
	return eim.sendVirtualKey(VK_ESCAPE)
}

// updateMethodStats 更新方法统计信息
func (eim *EnhancedInputManager) updateMethodStats(method InputMethod, duration time.Duration, success bool) {
	stats := eim.methodStats[method]
	if stats == nil {
		stats = &MethodStats{Reliability: 1.0}
		eim.methodStats[method] = stats
	}

	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	// 更新平均时间
	totalExecutions := stats.SuccessCount + stats.FailureCount
	if totalExecutions == 1 {
		stats.AverageTime = duration
	} else {
		stats.AverageTime = (stats.AverageTime*time.Duration(totalExecutions-1) + duration) / time.Duration(totalExecutions)
	}

	// 更新可靠性
	stats.Reliability = float64(stats.SuccessCount) / float64(totalExecutions)
	stats.LastUsed = time.Now()
}

// GetOptimalMethod 获取最优输入方法
func (eim *EnhancedInputManager) GetOptimalMethod() InputMethod {
	var bestMethod InputMethod = INPUT_CLIPBOARD_PASTE
	var bestScore float64 = 0

	for method, stats := range eim.methodStats {
		// 计算综合得分：可靠性 * 时间因子
		timeFactor := 1.0
		if stats.AverageTime > 0 {
			timeFactor = 1.0 / (float64(stats.AverageTime.Milliseconds()) / 100.0)
		}

		score := stats.Reliability * timeFactor
		if score > bestScore {
			bestScore = score
			bestMethod = method
		}
	}

	return bestMethod
}

// GetMethodStats 获取方法统计信息
func (eim *EnhancedInputManager) GetMethodStats() map[InputMethod]*MethodStats {
	return eim.methodStats
}

// SetFallbackChain 设置回退链
func (eim *EnhancedInputManager) SetFallbackChain(chain []InputMethod) {
	eim.fallbackChain = chain
}

// EnableFallback 启用/禁用回退机制
func (eim *EnhancedInputManager) EnableFallback(enable bool) {
	eim.enableFallback = enable
}
