package util

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
)

// EnhancedCommandExecutor 增强命令执行器
type EnhancedCommandExecutor struct {
	inputManager *EnhancedInputManager
	hwnd         syscall.Handle

	// 执行配置
	defaultInputMethod InputMethod
	defaultChatMethod  ChatActivationMethod

	// 性能优化
	commandCache  map[string]*CommandExecution
	lastExecution time.Time

	// 执行状态
	isInChat          bool
	lastChatMode      string
	consecutiveErrors int

	// 统计信息
	totalCommands   int
	successCommands int
	totalTime       time.Duration
}

// CommandExecution 命令执行记录
type CommandExecution struct {
	Command       string
	InputMethod   InputMethod
	ChatMethod    ChatActivationMethod
	ExecutionTime time.Duration
	Success       bool
	Result        string
	Timestamp     time.Time
	RetryCount    int
}

// CommandConfig 命令配置
type CommandConfig struct {
	PreferredInput   InputMethod
	PreferredChat    ChatActivationMethod
	ExpectedWaitTime time.Duration
	NeedsResponse    bool
	Priority         int
}

// NewEnhancedCommandExecutor 创建增强命令执行器
func NewEnhancedCommandExecutor(hwnd syscall.Handle) *EnhancedCommandExecutor {
	inputManager := NewEnhancedInputManager(hwnd)

	return &EnhancedCommandExecutor{
		inputManager:       inputManager,
		hwnd:               hwnd,
		defaultInputMethod: INPUT_CLIPBOARD_PASTE,
		defaultChatMethod:  CHAT_ACTIVATE_T_KEY,
		commandCache:       make(map[string]*CommandExecution),
		isInChat:           false,
		lastChatMode:       "UNKNOWN",
	}
}

// ExecuteCommand 执行单个命令（智能优化版）
func (ece *EnhancedCommandExecutor) ExecuteCommand(command string) (*CommandExecution, error) {
	start := time.Now()
	execution := &CommandExecution{
		Command:   command,
		Timestamp: start,
	}

	// 预处理命令
	processedCommand := ece.preprocessCommand(command)
	config := ece.getCommandConfig(processedCommand)

	// 选择最优输入和激活方法
	inputMethod := ece.selectOptimalInputMethod(processedCommand, config)
	chatMethod := ece.selectOptimalChatMethod(processedCommand, config)

	execution.InputMethod = inputMethod
	execution.ChatMethod = chatMethod

	// 执行命令
	result, err := ece.executeWithMethod(processedCommand, inputMethod, chatMethod, config)

	// 记录执行结果
	execution.ExecutionTime = time.Since(start)
	execution.Success = (err == nil)
	execution.Result = result

	// 更新统计信息
	ece.updateExecutionStats(execution)

	// 缓存执行记录
	ece.commandCache[processedCommand] = execution

	if err != nil {
		ece.consecutiveErrors++
		return execution, fmt.Errorf("命令执行失败: %v", err)
	}

	ece.consecutiveErrors = 0
	return execution, nil
}

// ExecuteBatch 批量执行命令（优化版）
func (ece *EnhancedCommandExecutor) ExecuteBatch(commands []string) ([]*CommandExecution, error) {
	batchStart := time.Now()
	var executions []*CommandExecution
	var errors []string

	fmt.Printf("开始批量执行 %d 个命令\n", len(commands))

	// 预先激活聊天框，避免每个命令都激活
	if err := ece.ensureChatActive(); err != nil {
		return nil, fmt.Errorf("批量执行前激活聊天框失败: %v", err)
	}

	for i, cmd := range commands {
		fmt.Printf("执行批量命令 [%d/%d]: %s\n", i+1, len(commands), cmd)

		execution, err := ece.ExecuteCommand(cmd)
		executions = append(executions, execution)

		if err != nil {
			errors = append(errors, fmt.Sprintf("命令 %d: %v", i+1, err))

			// 如果连续错误太多，停止执行
			if ece.consecutiveErrors > 3 {
				fmt.Printf("连续错误过多，停止批量执行\n")
				break
			}
		}

		// 动态调整命令间隔
		if i < len(commands)-1 {
			interval := ece.calculateDynamicInterval(execution, i, len(commands))
			time.Sleep(interval)
		}
	}

	batchDuration := time.Since(batchStart)
	fmt.Printf("批量执行完成，总耗时: %v，成功: %d/%d\n",
		batchDuration, len(executions)-len(errors), len(executions))

	if len(errors) > 0 {
		return executions, fmt.Errorf("批量执行有 %d 个错误: %s", len(errors), strings.Join(errors, "; "))
	}

	return executions, nil
}

// executeWithMethod 使用指定方法执行命令
func (ece *EnhancedCommandExecutor) executeWithMethod(command string, inputMethod InputMethod, chatMethod ChatActivationMethod, config *CommandConfig) (string, error) {
	// 1. 确保窗口激活
	SetForegroundWindow(ece.hwnd)
	time.Sleep(50 * time.Millisecond)

	// 2. 激活聊天框（如果还未激活）
	if !ece.isInChat {
		if err := ece.inputManager.ActivateChat(chatMethod); err != nil {
			return "", fmt.Errorf("激活聊天框失败: %v", err)
		}
		ece.isInChat = true
		time.Sleep(100 * time.Millisecond)
	}

	// 3. 发送命令文本
	if err := ece.inputManager.SendText(command, inputMethod); err != nil {
		return "", fmt.Errorf("发送命令文本失败: %v", err)
	}

	// 4. 发送回车键执行命令
	if err := ece.inputManager.SendEnter(); err != nil {
		return "", fmt.Errorf("发送回车键失败: %v", err)
	}

	fmt.Printf("命令已发送: %s (方法: %d)\n", command, inputMethod)

	// 5. 等待并获取结果（如果需要）
	var result string
	if config.NeedsResponse {
		result = ece.waitForCommandResult(command, config.ExpectedWaitTime)
	}

	return result, nil
}

// ensureChatActive 确保聊天框处于激活状态
func (ece *EnhancedCommandExecutor) ensureChatActive() error {
	if ece.isInChat {
		return nil // 已经激活
	}

	// 尝试激活聊天框
	err := ece.inputManager.ActivateChat(ece.defaultChatMethod)
	if err != nil {
		// 尝试备选方法
		for _, method := range []ChatActivationMethod{CHAT_ACTIVATE_T_KEY, CHAT_ACTIVATE_SLASH_KEY, CHAT_ACTIVATE_WINDOW_MSG} {
			if method == ece.defaultChatMethod {
				continue
			}
			if err = ece.inputManager.ActivateChat(method); err == nil {
				break
			}
		}
	}

	if err == nil {
		ece.isInChat = true
		time.Sleep(200 * time.Millisecond) // 等待聊天框稳定
	}

	return err
}

// selectOptimalInputMethod 选择最优输入方法
func (ece *EnhancedCommandExecutor) selectOptimalInputMethod(command string, config *CommandConfig) InputMethod {
	// 1. 如果配置中指定了首选方法，使用配置的方法
	if config.PreferredInput != INPUT_HYBRID {
		return config.PreferredInput
	}

	// 2. 基于命令特性选择方法
	if len(command) > 100 {
		// 长命令优先使用剪贴板
		return INPUT_CLIPBOARD_PASTE
	}

	if strings.Contains(command, "#") && len(command) < 50 {
		// 短的系统命令使用模拟按键
		return INPUT_SIMULATE_KEY
	}

	// 3. 基于历史性能选择
	if ece.consecutiveErrors > 1 {
		// 连续错误时使用更可靠的方法
		return INPUT_WINDOW_MSG
	}

	// 4. 使用输入管理器的最优方法
	return ece.inputManager.GetOptimalMethod()
}

// selectOptimalChatMethod 选择最优聊天激活方法
func (ece *EnhancedCommandExecutor) selectOptimalChatMethod(command string, config *CommandConfig) ChatActivationMethod {
	// 1. 如果配置中指定了首选方法
	if config.PreferredChat != CHAT_ACTIVATE_T_KEY {
		return config.PreferredChat
	}

	// 2. 基于命令类型选择
	if strings.HasPrefix(command, "#") {
		// 系统命令可以用任何方式激活
		return CHAT_ACTIVATE_T_KEY
	}

	if strings.HasPrefix(command, "/") {
		// 如果是斜杠命令，直接用斜杠激活
		return CHAT_ACTIVATE_SLASH_KEY
	}

	// 3. 默认使用T键
	return CHAT_ACTIVATE_T_KEY
}

// getCommandConfig 获取命令配置
func (ece *EnhancedCommandExecutor) getCommandConfig(command string) *CommandConfig {
	// 预定义命令配置
	configs := map[string]*CommandConfig{
		"#ListPlayers": {
			PreferredInput:   INPUT_CLIPBOARD_PASTE,
			PreferredChat:    CHAT_ACTIVATE_T_KEY,
			ExpectedWaitTime: 1500 * time.Millisecond,
			NeedsResponse:    true,
			Priority:         1,
		},
		"#ListSpawnedVehicles": {
			PreferredInput:   INPUT_CLIPBOARD_PASTE,
			PreferredChat:    CHAT_ACTIVATE_T_KEY,
			ExpectedWaitTime: 1200 * time.Millisecond,
			NeedsResponse:    true,
			Priority:         1,
		},
		"#dumpallsquadsinfolist": {
			PreferredInput:   INPUT_CLIPBOARD_PASTE,
			PreferredChat:    CHAT_ACTIVATE_T_KEY,
			ExpectedWaitTime: 2000 * time.Millisecond,
			NeedsResponse:    true,
			Priority:         1,
		},
		"#SetTime": {
			PreferredInput:   INPUT_SIMULATE_KEY,
			PreferredChat:    CHAT_ACTIVATE_T_KEY,
			ExpectedWaitTime: 200 * time.Millisecond,
			NeedsResponse:    false,
			Priority:         2,
		},
	}

	// 精确匹配
	if config, exists := configs[command]; exists {
		return config
	}

	// 前缀匹配
	for prefix, config := range configs {
		if strings.HasPrefix(command, prefix) {
			return config
		}
	}

	// 默认配置
	return &CommandConfig{
		PreferredInput:   INPUT_HYBRID,
		PreferredChat:    CHAT_ACTIVATE_T_KEY,
		ExpectedWaitTime: 800 * time.Millisecond,
		NeedsResponse:    false,
		Priority:         3,
	}
}

// waitForCommandResult 等待命令结果
func (ece *EnhancedCommandExecutor) waitForCommandResult(command string, expectedWait time.Duration) string {
	// 清空剪贴板准备接收结果
	clipboard.WriteAll("")
	time.Sleep(50 * time.Millisecond)

	// 分段等待，提前检查结果
	steps := 4
	stepTime := expectedWait / time.Duration(steps)

	for step := 0; step < steps; step++ {
		time.Sleep(stepTime)

		if result, err := clipboard.ReadAll(); err == nil && result != "" && result != command {
			fmt.Printf("获取到命令结果，长度: %d\n", len(result))
			return result
		}
	}

	// 最终尝试
	if result, err := clipboard.ReadAll(); err == nil && result != "" && result != command {
		return result
	}

	return ""
}

// calculateDynamicInterval 计算动态间隔时间
func (ece *EnhancedCommandExecutor) calculateDynamicInterval(lastExecution *CommandExecution, currentIndex, totalCount int) time.Duration {
	baseInterval := 300 * time.Millisecond

	// 基于执行时间调整
	if lastExecution.ExecutionTime > 2*time.Second {
		baseInterval += 200 * time.Millisecond
	} else if lastExecution.ExecutionTime < 500*time.Millisecond {
		baseInterval -= 100 * time.Millisecond
	}

	// 基于成功率调整
	if ece.consecutiveErrors > 0 {
		baseInterval += time.Duration(ece.consecutiveErrors) * 200 * time.Millisecond
	}

	// 基于进度调整（后期减少间隔）
	progress := float64(currentIndex) / float64(totalCount)
	if progress > 0.7 && ece.consecutiveErrors == 0 {
		baseInterval = time.Duration(float64(baseInterval) * 0.8)
	}

	// 最小间隔限制
	if baseInterval < 100*time.Millisecond {
		baseInterval = 100 * time.Millisecond
	}

	return baseInterval
}

// preprocessCommand 预处理命令
func (ece *EnhancedCommandExecutor) preprocessCommand(command string) string {
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

// updateExecutionStats 更新执行统计
func (ece *EnhancedCommandExecutor) updateExecutionStats(execution *CommandExecution) {
	ece.totalCommands++
	ece.totalTime += execution.ExecutionTime

	if execution.Success {
		ece.successCommands++
	}

	ece.lastExecution = execution.Timestamp
}

// CloseChatIfOpen 关闭聊天框（如果打开）
func (ece *EnhancedCommandExecutor) CloseChatIfOpen() error {
	if !ece.isInChat {
		return nil
	}

	err := ece.inputManager.SendEscape()
	if err == nil {
		ece.isInChat = false
		time.Sleep(100 * time.Millisecond)
	}

	return err
}

// GetExecutionStats 获取执行统计信息
func (ece *EnhancedCommandExecutor) GetExecutionStats() map[string]interface{} {
	successRate := 0.0
	averageTime := time.Duration(0)

	if ece.totalCommands > 0 {
		successRate = float64(ece.successCommands) / float64(ece.totalCommands)
		averageTime = ece.totalTime / time.Duration(ece.totalCommands)
	}

	return map[string]interface{}{
		"total_commands":     ece.totalCommands,
		"success_commands":   ece.successCommands,
		"success_rate":       successRate,
		"average_time":       averageTime.String(),
		"consecutive_errors": ece.consecutiveErrors,
		"is_in_chat":         ece.isInChat,
		"input_method_stats": ece.inputManager.GetMethodStats(),
	}
}

// ResetStats 重置统计信息
func (ece *EnhancedCommandExecutor) ResetStats() {
	ece.totalCommands = 0
	ece.successCommands = 0
	ece.totalTime = 0
	ece.consecutiveErrors = 0
	ece.commandCache = make(map[string]*CommandExecution)
}

// SetDefaultMethods 设置默认方法
func (ece *EnhancedCommandExecutor) SetDefaultMethods(inputMethod InputMethod, chatMethod ChatActivationMethod) {
	ece.defaultInputMethod = inputMethod
	ece.defaultChatMethod = chatMethod
}
