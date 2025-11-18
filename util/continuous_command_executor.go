package util

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
)

// ContinuousCommandExecutor 连续命令执行器
// 专门优化：按T激活聊天框后，连续输入多个命令，每个命令用回车分隔
type ContinuousCommandExecutor struct {
	inputManager *EnhancedInputManager
	hwnd         syscall.Handle

	// 连续执行状态
	isChatSessionActive bool
	sessionStartTime    time.Time
	sessionCommands     []string

	// 配置
	defaultInputMethod InputMethod
	sessionTimeout     time.Duration
	commandInterval    time.Duration

	// 统计
	sessionStats *SessionStats
}

// SessionStats 会话统计
type SessionStats struct {
	TotalSessions      int
	TotalCommands      int
	SuccessCommands    int
	AverageSessionTime time.Duration
	LastSessionTime    time.Time
}

// NewContinuousCommandExecutor 创建连续命令执行器
func NewContinuousCommandExecutor(hwnd syscall.Handle) *ContinuousCommandExecutor {
	return &ContinuousCommandExecutor{
		hwnd:               hwnd,
		inputManager:       NewEnhancedInputManager(hwnd),
		defaultInputMethod: INPUT_CLIPBOARD_PASTE,
		sessionTimeout:     30 * time.Second,
		commandInterval:    200 * time.Millisecond,
		sessionStats:       &SessionStats{},
	}
}

// StartContinuousSession 开始连续命令会话
func (cce *ContinuousCommandExecutor) StartContinuousSession() error {
	if cce.isChatSessionActive {
		fmt.Println("会话已经激活，将重新开始")
		cce.EndContinuousSession()
	}

	fmt.Println("开始连续命令会话...")

	// 设置窗口为前台 - 已注释：使用句柄操作不需要窗口置顶
	// SetForegroundWindow(cce.hwnd)
	// time.Sleep(100 * time.Millisecond)

	// 激活聊天框（只激活一次）
	if err := cce.inputManager.ActivateChat(CHAT_ACTIVATE_T_KEY); err != nil {
		return fmt.Errorf("激活聊天框失败: %v", err)
	}

	// 等待聊天框稳定
	time.Sleep(300 * time.Millisecond)

	// 标记会话开始
	cce.isChatSessionActive = true
	cce.sessionStartTime = time.Now()
	cce.sessionCommands = []string{}

	fmt.Println("✓ 连续命令会话已激活，可以开始输入命令")
	return nil
}

// AddCommandToContinuousSession 在连续会话中添加命令
func (cce *ContinuousCommandExecutor) AddCommandToContinuousSession(command string) error {
	if !cce.isChatSessionActive {
		return fmt.Errorf("连续会话未激活，请先调用 StartContinuousSession()")
	}

	// 检查会话超时
	if time.Since(cce.sessionStartTime) > cce.sessionTimeout {
		fmt.Println("会话超时，重新激活...")
		if err := cce.StartContinuousSession(); err != nil {
			return fmt.Errorf("重新激活会话失败: %v", err)
		}
	}

	// 预处理命令
	processedCommand := cce.preprocessCommand(command)

	fmt.Printf("在连续会话中添加命令: %s\n", processedCommand)

	// 发送命令文本（不重新激活聊天框）
	if err := cce.inputManager.SendText(processedCommand, cce.defaultInputMethod); err != nil {
		return fmt.Errorf("发送命令文本失败: %v", err)
	}

	// 发送回车键执行命令
	if err := cce.inputManager.SendEnter(); err != nil {
		return fmt.Errorf("发送回车键失败: %v", err)
	}

	// 记录命令
	cce.sessionCommands = append(cce.sessionCommands, processedCommand)

	// 等待命令执行完成的间隔
	time.Sleep(cce.commandInterval)

	fmt.Printf("✓ 命令已发送: %s\n", processedCommand)
	return nil
}

// ExecuteContinuousBatch 批量执行连续命令（优化版）
func (cce *ContinuousCommandExecutor) ExecuteContinuousBatch(commands []string) error {
	if len(commands) == 0 {
		return fmt.Errorf("命令列表为空")
	}

	start := time.Now()
	fmt.Printf("开始连续批量执行 %d 个命令...\n", len(commands))

	// 开始连续会话
	if err := cce.StartContinuousSession(); err != nil {
		return fmt.Errorf("启动连续会话失败: %v", err)
	}

	successCount := 0
	errorCount := 0

	// 连续发送所有命令
	for i, cmd := range commands {
		fmt.Printf("[%d/%d] 发送命令: %s\n", i+1, len(commands), cmd)

		if err := cce.AddCommandToContinuousSession(cmd); err != nil {
			fmt.Printf("✗ 命令失败: %v\n", err)
			errorCount++

			// 如果连续失败太多，重新激活会话
			if errorCount >= 3 {
				fmt.Println("连续失败过多，重新激活会话...")
				if err := cce.StartContinuousSession(); err != nil {
					return fmt.Errorf("重新激活会话失败: %v", err)
				}
				errorCount = 0
			}
		} else {
			successCount++
		}

		// 动态调整命令间隔
		if i < len(commands)-1 {
			interval := cce.calculateDynamicInterval(successCount, errorCount, i, len(commands))
			time.Sleep(interval)
		}
	}

	// 结束会话（自动按ESC关闭聊天框）
	cce.EndContinuousSession()

	duration := time.Since(start)
	fmt.Printf("连续批量执行完成: %d/%d成功, 耗时: %v, 聊天框已关闭\n",
		successCount, len(commands), duration)

	// 更新统计
	cce.updateSessionStats(len(commands), successCount, duration)

	if errorCount > 0 {
		return fmt.Errorf("批量执行有 %d 个命令失败", errorCount)
	}

	return nil
}

// EndContinuousSession 结束连续命令会话
func (cce *ContinuousCommandExecutor) EndContinuousSession() {
	if !cce.isChatSessionActive {
		return
	}

	fmt.Println("结束连续命令会话，按ESC关闭聊天框...")

	// 发送ESC关闭聊天框
	cce.inputManager.SendEscape()
	time.Sleep(200 * time.Millisecond)

	// 记录会话统计
	sessionDuration := time.Since(cce.sessionStartTime)
	fmt.Printf("✓ 会话结束 - 执行了%d个命令，耗时: %v，聊天框已关闭\n",
		len(cce.sessionCommands), sessionDuration)

	// 重置状态
	cce.isChatSessionActive = false
	cce.sessionCommands = []string{}
}

// calculateDynamicInterval 计算动态间隔时间
func (cce *ContinuousCommandExecutor) calculateDynamicInterval(successCount, errorCount, currentIndex, totalCount int) time.Duration {
	baseInterval := cce.commandInterval

	// 基于成功率调整
	if errorCount > 0 {
		// 有错误时增加间隔
		baseInterval += time.Duration(errorCount) * 100 * time.Millisecond
	} else if successCount > 5 {
		// 连续成功时减少间隔
		baseInterval = time.Duration(float64(baseInterval) * 0.8)
	}

	// 基于进度调整
	progress := float64(currentIndex) / float64(totalCount)
	if progress > 0.5 && errorCount == 0 {
		// 后半段且无错误时可以加速
		baseInterval = time.Duration(float64(baseInterval) * 0.9)
	}

	// 最小间隔限制
	if baseInterval < 100*time.Millisecond {
		baseInterval = 100 * time.Millisecond
	}

	// 最大间隔限制
	if baseInterval > 1*time.Second {
		baseInterval = 1 * time.Second
	}

	return baseInterval
}

// preprocessCommand 预处理命令
func (cce *ContinuousCommandExecutor) preprocessCommand(command string) string {
	command = strings.TrimSpace(command)

	// 处理命令别名
	aliases := map[string]string{
		"players":   "#ListPlayers true",
		"vehicles":  "#ListSpawnedVehicles true",
		"squads":    "#dumpallsquadsinfolist",
		"time12":    "#SetTime 12 00",
		"time0":     "#SetTime 00 00",
		"morning":   "#SetTime 08 00",
		"noon":      "#SetTime 12 00",
		"evening":   "#SetTime 18 00",
		"night":     "#SetTime 22 00",
		"sunrise":   "#SetTime 06 00",
		"sunset":    "#SetTime 20 00",
		"midnight":  "#SetTime 00 00",
		"flags":     "#listflags 1 true",
		"save":      "#Save",
		"restart":   "#RestartServer",
		"shutdown":  "#Shutdown",
		"godmode":   "#SetGodMode true",
		"nogodmode": "#SetGodMode false",
		"weather0":  "#SetWeather 0",
		"weather1":  "#SetWeather 1",
		"sunny":     "#SetWeather 0",
		"storm":     "#SetWeather 1",
	}

	if alias, exists := aliases[strings.ToLower(command)]; exists {
		return alias
	}

	return command
}

// updateSessionStats 更新会话统计
func (cce *ContinuousCommandExecutor) updateSessionStats(totalCommands, successCommands int, duration time.Duration) {
	cce.sessionStats.TotalSessions++
	cce.sessionStats.TotalCommands += totalCommands
	cce.sessionStats.SuccessCommands += successCommands
	cce.sessionStats.LastSessionTime = time.Now()

	// 更新平均会话时间
	if cce.sessionStats.TotalSessions == 1 {
		cce.sessionStats.AverageSessionTime = duration
	} else {
		cce.sessionStats.AverageSessionTime =
			(cce.sessionStats.AverageSessionTime*time.Duration(cce.sessionStats.TotalSessions-1) + duration) /
				time.Duration(cce.sessionStats.TotalSessions)
	}
}

// GetSessionStats 获取会话统计信息
func (cce *ContinuousCommandExecutor) GetSessionStats() *SessionStats {
	return cce.sessionStats
}

// SetCommandInterval 设置命令间隔时间
func (cce *ContinuousCommandExecutor) SetCommandInterval(interval time.Duration) {
	cce.commandInterval = interval
}

// SetSessionTimeout 设置会话超时时间
func (cce *ContinuousCommandExecutor) SetSessionTimeout(timeout time.Duration) {
	cce.sessionTimeout = timeout
}

// SetDefaultInputMethod 设置默认输入方法
func (cce *ContinuousCommandExecutor) SetDefaultInputMethod(method InputMethod) {
	cce.defaultInputMethod = method
}

// IsSessionActive 检查会话是否激活
func (cce *ContinuousCommandExecutor) IsSessionActive() bool {
	return cce.isChatSessionActive
}

// GetCurrentSessionInfo 获取当前会话信息
func (cce *ContinuousCommandExecutor) GetCurrentSessionInfo() map[string]interface{} {
	info := map[string]interface{}{
		"is_active":        cce.isChatSessionActive,
		"default_method":   cce.defaultInputMethod,
		"command_interval": cce.commandInterval.String(),
		"session_timeout":  cce.sessionTimeout.String(),
	}

	if cce.isChatSessionActive {
		info["session_start_time"] = cce.sessionStartTime
		info["session_duration"] = time.Since(cce.sessionStartTime).String()
		info["commands_in_session"] = len(cce.sessionCommands)
		info["session_commands"] = cce.sessionCommands
	}

	return info
}

// ExecuteQuickSequence 执行快速命令序列（预定义的常用组合）
func (cce *ContinuousCommandExecutor) ExecuteQuickSequence(sequenceName string) error {
	sequences := map[string][]string{
		"status_check": {
			"#ListPlayers true",
			"#ListSpawnedVehicles true",
			"#dumpallsquadsinfolist",
		},
		"server_reset": {
			"#SetTime 12 00",
			"#SetWeather 0",
			"#Save",
		},
		"morning_routine": {
			"#SetTime 08 00",
			"#SetWeather 0",
			"#ListPlayers true",
		},
		"evening_routine": {
			"#SetTime 18 00",
			"#SetWeather 0",
			"#Save",
		},
		"admin_check": {
			"#ListPlayers true",
			"#ListSpawnedVehicles true",
			"#listflags 1 true",
			"#dumpallsquadsinfolist",
		},
	}

	commands, exists := sequences[sequenceName]
	if !exists {
		return fmt.Errorf("未知的命令序列: %s", sequenceName)
	}

	fmt.Printf("执行预定义序列: %s (%d个命令)\n", sequenceName, len(commands))
	return cce.ExecuteContinuousBatch(commands)
}

// WaitForCommandResponse 等待命令响应（适用于需要返回结果的命令）
func (cce *ContinuousCommandExecutor) WaitForCommandResponse(command string, timeout time.Duration) (string, error) {
	if !cce.isChatSessionActive {
		return "", fmt.Errorf("连续会话未激活")
	}

	// 清空剪贴板准备接收结果
	clipboard.WriteAll("")
	time.Sleep(50 * time.Millisecond)

	// 发送命令
	if err := cce.AddCommandToContinuousSession(command); err != nil {
		return "", err
	}

	// 等待结果
	start := time.Now()
	for time.Since(start) < timeout {
		if result, err := clipboard.ReadAll(); err == nil && result != "" && result != command {
			fmt.Printf("获取到命令响应，长度: %d\n", len(result))
			return result, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf("等待命令响应超时: %v", timeout)
}
