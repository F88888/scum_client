package quick

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qq_client/util"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-vgo/robotgo"
)

// FastCommand 快速命令执行器（升级版）
type FastCommand struct {
	hwnd         syscall.Handle
	isReady      bool
	lastCommand  time.Time
	commandQueue []string

	// 集成增强功能
	enhancedExecutor   *util.EnhancedCommandExecutor
	inputManager       *util.EnhancedInputManager
	continuousExecutor *util.ContinuousCommandExecutor

	// 配置选项
	preferredInputMethod    util.InputMethod
	preferredChatMethod     util.ChatActivationMethod
	enableSmartMode         bool
	enableBatchOptimization bool
	enableContinuousMode    bool
}

// CommandRequest HTTP API请求结构
type CommandRequest struct {
	Command  string   `json:"command"`
	Commands []string `json:"commands"`
}

// CommandResponse HTTP API响应结构
type CommandResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Result   string `json:"result,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// NewFastCommand 创建快速命令执行器（增强版）
func NewFastCommand() *FastCommand {
	fc := &FastCommand{
		commandQueue:            make([]string, 0),
		preferredInputMethod:    util.INPUT_CLIPBOARD_PASTE,
		preferredChatMethod:     util.CHAT_ACTIVATE_T_KEY,
		enableSmartMode:         true,
		enableBatchOptimization: true,
		enableContinuousMode:    true,
	}
	return fc
}

// Initialize 初始化快速命令执行器（增强版）
func (fc *FastCommand) Initialize() error {
	// 查找SCUM游戏窗口
	fc.hwnd = util.FindWindow("UnrealWindow", "SCUM  ")
	if fc.hwnd == 0 {
		return fmt.Errorf("未找到SCUM游戏窗口")
	}

	// 初始化增强组件
	fc.enhancedExecutor = util.NewEnhancedCommandExecutor(fc.hwnd)
	fc.inputManager = util.NewEnhancedInputManager(fc.hwnd)
	fc.continuousExecutor = util.NewContinuousCommandExecutor(fc.hwnd)

	// 设置默认方法
	fc.enhancedExecutor.SetDefaultMethods(fc.preferredInputMethod, fc.preferredChatMethod)
	fc.continuousExecutor.SetDefaultInputMethod(fc.preferredInputMethod)

	// 设置窗口为前台并准备聊天界面
	if err := fc.prepareForCommands(); err != nil {
		return fmt.Errorf("准备命令执行环境失败: %v", err)
	}

	fc.isReady = true
	fmt.Printf("FastCommand 初始化完成，启用智能模式: %v，批量优化: %v，连续模式: %v\n",
		fc.enableSmartMode, fc.enableBatchOptimization, fc.enableContinuousMode)
	return nil
}

// prepareForCommands 准备命令执行环境
func (fc *FastCommand) prepareForCommands() error {
	// 设置窗口为前台
	util.SetForegroundWindow(fc.hwnd)
	time.Sleep(100 * time.Millisecond)

	// 检查是否在聊天界面
	if !fc.isChatActive() {
		// 激活聊天界面
		robotgo.KeyTap("t")
		time.Sleep(200 * time.Millisecond)

		// 验证聊天界面是否激活
		if !fc.isChatActive() {
			return fmt.Errorf("无法激活聊天界面")
		}
	}

	// 确保在GLOBAL模式
	return fc.ensureGlobalMode()
}

// isChatActive 检查聊天界面是否激活
func (fc *FastCommand) isChatActive() bool {
	return util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(30, 310, 61, 325, "MUTE") == nil
}

// ensureGlobalMode 确保在GLOBAL聊天模式
func (fc *FastCommand) ensureGlobalMode() error {
	maxAttempts := 3
	for i := 0; i < maxAttempts; i++ {
		if fc.getCurrentChatMode() == "GLOBAL" {
			return nil
		}

		// 按Tab切换模式
		robotgo.KeyTap("tab")
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("无法切换到GLOBAL模式")
}

// getCurrentChatMode 获取当前聊天模式
func (fc *FastCommand) getCurrentChatMode() string {
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(233, 308, 267, 327, "GLOBAL") == nil {
		return "GLOBAL"
	}
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(237, 309, 268, 328, "LOCAL") == nil {
		return "LOCAL"
	}
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(233, 308, 267, 327, "ADMIN") == nil {
		return "ADMIN"
	}
	return "UNKNOWN"
}

// ExecuteCommand 执行单个命令（增强版）
func (fc *FastCommand) ExecuteCommand(command string) (string, error) {
	if !fc.isReady {
		if err := fc.Initialize(); err != nil {
			return "", err
		}
	}

	start := time.Now()

	// 使用增强执行器（如果启用智能模式）
	if fc.enableSmartMode && fc.enhancedExecutor != nil {
		execution, err := fc.enhancedExecutor.ExecuteCommand(command)
		duration := time.Since(start)
		fc.lastCommand = time.Now()

		if err != nil {
			return "", fmt.Errorf("增强执行失败: %v", err)
		}

		result := fmt.Sprintf("命令执行完成 [耗时: %v] [方法: %d]\n结果: %s",
			duration, execution.InputMethod, execution.Result)
		return result, nil
	}

	// 回退到传统方法
	return fc.executeCommandLegacy(command)
}

// executeCommandLegacy 传统执行方法（保持兼容性）
func (fc *FastCommand) executeCommandLegacy(command string) (string, error) {
	start := time.Now()

	// 预处理命令
	processedCommand := fc.preprocessCommand(command)

	// 清空聊天输入框（使用Ctrl+A选中全部然后删除）
	robotgo.KeyTap("a", "ctrl")
	time.Sleep(10 * time.Millisecond)
	robotgo.KeyTap("delete")
	time.Sleep(10 * time.Millisecond)

	// 快速输入命令
	if err := fc.fastType(processedCommand); err != nil {
		return "", fmt.Errorf("输入命令失败: %v", err)
	}

	// 发送命令
	robotgo.KeyTap("enter")
	time.Sleep(50 * time.Millisecond)

	// 等待命令执行并获取结果
	result := fc.waitForResult(processedCommand)

	duration := time.Since(start)
	fc.lastCommand = time.Now()

	return fmt.Sprintf("命令执行完成 [耗时: %v]\n结果: %s", duration, result), nil
}

// ExecuteBatch 批量执行命令（增强版）
func (fc *FastCommand) ExecuteBatch(commands []string) ([]string, error) {
	if !fc.isReady {
		if err := fc.Initialize(); err != nil {
			return nil, err
		}
	}

	start := time.Now()

	// 使用连续执行（如果启用连续模式）
	if fc.enableContinuousMode && fc.continuousExecutor != nil {
		err := fc.continuousExecutor.ExecuteContinuousBatch(commands)

		var results []string
		if err != nil {
			results = append(results, fmt.Sprintf("连续执行失败: %v", err))
		} else {
			results = append(results, "连续执行成功")
		}

		// 获取统计信息
		stats := fc.continuousExecutor.GetSessionStats()
		totalDuration := time.Since(start)
		summary := fmt.Sprintf("连续批量执行完成: %d命令, 总成功: %d, 总耗时: %v, 聊天框已关闭",
			len(commands), stats.SuccessCommands, totalDuration)

		results = append([]string{summary}, results...)

		if err != nil {
			return results, err
		}
		return results, nil
	}

	// 使用增强批量执行（如果启用批量优化）
	if fc.enableBatchOptimization && fc.enhancedExecutor != nil {
		executions, err := fc.enhancedExecutor.ExecuteBatch(commands)

		var results []string
		successCount := 0

		for i, execution := range executions {
			if execution.Success {
				successCount++
				results = append(results, fmt.Sprintf("命令 %d: %s [成功] [耗时: %v] [方法: %d]",
					i+1, execution.Command, execution.ExecutionTime, execution.InputMethod))
			} else {
				results = append(results, fmt.Sprintf("命令 %d: %s [失败]",
					i+1, execution.Command))
			}
		}

		totalDuration := time.Since(start)
		summary := fmt.Sprintf("增强批量执行完成: %d/%d成功, 总耗时: %v, 平均耗时: %v",
			successCount, len(commands), totalDuration, totalDuration/time.Duration(len(commands)))

		results = append([]string{summary}, results...)

		if err != nil {
			return results, fmt.Errorf("批量执行有错误: %v", err)
		}
		return results, nil
	}

	// 回退到传统批量执行
	return fc.executeBatchLegacy(commands)
}

// executeBatchLegacy 传统批量执行方法（保持兼容性）
func (fc *FastCommand) executeBatchLegacy(commands []string) ([]string, error) {
	var results []string
	start := time.Now()

	for i, cmd := range commands {
		// 预处理命令
		processedCommand := fc.preprocessCommand(cmd)

		// 清空输入框
		robotgo.KeyTap("a", "ctrl")
		time.Sleep(8 * time.Millisecond)
		robotgo.KeyTap("delete")
		time.Sleep(8 * time.Millisecond)

		// 快速输入并发送
		if err := fc.fastType(processedCommand); err != nil {
			results = append(results, fmt.Sprintf("命令 %d 输入失败: %v", i+1, err))
			continue
		}

		robotgo.KeyTap("enter")

		// 批量模式下缩短等待时间
		time.Sleep(30 * time.Millisecond)

		result := fmt.Sprintf("命令 %d: %s [已发送]", i+1, processedCommand)
		results = append(results, result)

		// 命令间隔时间最小化
		if i < len(commands)-1 {
			time.Sleep(20 * time.Millisecond)
		}
	}

	totalDuration := time.Since(start)
	summary := fmt.Sprintf("传统批量执行完成: %d个命令, 总耗时: %v, 平均耗时: %v",
		len(commands), totalDuration, totalDuration/time.Duration(len(commands)))

	results = append([]string{summary}, results...)
	return results, nil
}

// fastType 快速输入文本（优化版本）
func (fc *FastCommand) fastType(text string) error {
	// 使用剪贴板方式快速输入长文本
	if len(text) > 20 {
		return fc.typeViaClipboard(text)
	}

	// 短文本直接键盘输入，最小化延迟
	for _, char := range text {
		robotgo.TypeStr(string(char))
		time.Sleep(5 * time.Millisecond) // 最小延迟
	}

	return nil
}

// typeViaClipboard 通过剪贴板快速输入文本
func (fc *FastCommand) typeViaClipboard(text string) error {
	// 备份当前剪贴板内容
	originalClipboard, _ := robotgo.ReadAll()

	// 写入文本到剪贴板
	robotgo.WriteAll(text)
	time.Sleep(10 * time.Millisecond)

	// 粘贴
	robotgo.KeyTap("v", "ctrl")
	time.Sleep(20 * time.Millisecond)

	// 恢复剪贴板内容
	if originalClipboard != "" {
		robotgo.WriteAll(originalClipboard)
	}

	return nil
}

// preprocessCommand 预处理命令
func (fc *FastCommand) preprocessCommand(command string) string {
	command = strings.TrimSpace(command)

	// 处理命令别名
	aliases := fc.getCommandAliases()
	if alias, exists := aliases[command]; exists {
		return alias
	}

	// 如果不以#开头，自动添加
	if !strings.HasPrefix(command, "#") {
		return "#" + command
	}

	return command
}

// getCommandAliases 获取命令别名映射
func (fc *FastCommand) getCommandAliases() map[string]string {
	return map[string]string{
		"players":  "#ListPlayers",
		"vehicles": "#ListSpawnedVehicles",
		"time12":   "#SetTime 12 00",
		"time0":    "#SetTime 00 00",
		"flags":    "#listflags 1 true",
		"squads":   "#dumpallsquadsinfolist",
		"morning":  "#SetTime 08 00",
		"noon":     "#SetTime 12 00",
		"evening":  "#SetTime 18 00",
		"night":    "#SetTime 22 00",
		"restart":  "#RestartServer",
		"shutdown": "#Shutdown",
	}
}

// waitForResult 等待命令执行结果
func (fc *FastCommand) waitForResult(command string) string {
	// 根据命令类型决定等待时间
	waitTime := fc.getWaitTimeForCommand(command)
	time.Sleep(waitTime)

	// 尝试从聊天记录或系统反馈中获取结果
	// 这里可以根据实际需要实现结果解析逻辑
	return "命令已发送"
}

// getWaitTimeForCommand 根据命令获取最优等待时间
func (fc *FastCommand) getWaitTimeForCommand(command string) time.Duration {
	// 根据命令类型返回最小等待时间
	fastCommands := map[string]time.Duration{
		"#SetTime":               50 * time.Millisecond,
		"#ListPlayers":           200 * time.Millisecond,
		"#ListSpawnedVehicles":   300 * time.Millisecond,
		"#listflags":             500 * time.Millisecond,
		"#dumpallsquadsinfolist": 400 * time.Millisecond,
	}

	for prefix, duration := range fastCommands {
		if strings.HasPrefix(command, prefix) {
			return duration
		}
	}

	// 默认等待时间
	return 100 * time.Millisecond
}

// StartHTTPServer 启动HTTP API服务器
func (fc *FastCommand) StartHTTPServer(port string) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 单个命令执行端点
	r.POST("/command", fc.handleSingleCommand)

	// 批量命令执行端点
	r.POST("/batch", fc.handleBatchCommands)

	// 状态检查端点
	r.GET("/status", fc.handleStatus)

	// 预设命令端点
	r.GET("/presets", fc.handlePresets)

	// 新增增强功能端点
	r.GET("/stats", fc.handleStats)
	r.POST("/config", fc.handleConfig)
	r.GET("/methods", fc.handleMethods)
	r.POST("/test", fc.handleTest)

	// 连续执行功能端点
	r.POST("/continuous/start", fc.handleContinuousStart)
	r.POST("/continuous/add", fc.handleContinuousAdd)
	r.POST("/continuous/batch", fc.handleContinuousBatch)
	r.POST("/continuous/end", fc.handleContinuousEnd)
	r.GET("/continuous/status", fc.handleContinuousStatus)
	r.POST("/continuous/sequence", fc.handleContinuousSequence)

	fmt.Printf("快速命令HTTP服务器启动在端口 %s\n", port)
	fmt.Printf("增强功能已启用 - 智能模式: %v, 批量优化: %v, 连续模式: %v\n",
		fc.enableSmartMode, fc.enableBatchOptimization, fc.enableContinuousMode)
	r.Run(":" + port)
}

// handleSingleCommand 处理单个命令请求
func (fc *FastCommand) handleSingleCommand(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Message: "请求格式错误: " + err.Error(),
		})
		return
	}

	start := time.Now()
	result, err := fc.ExecuteCommand(req.Command)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, CommandResponse{
			Success:  false,
			Message:  "命令执行失败: " + err.Error(),
			Duration: duration.String(),
		})
		return
	}

	c.JSON(http.StatusOK, CommandResponse{
		Success:  true,
		Message:  "命令执行成功",
		Result:   result,
		Duration: duration.String(),
	})
}

// handleBatchCommands 处理批量命令请求
func (fc *FastCommand) handleBatchCommands(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Message: "请求格式错误: " + err.Error(),
		})
		return
	}

	start := time.Now()
	results, err := fc.ExecuteBatch(req.Commands)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, CommandResponse{
			Success:  false,
			Message:  "批量命令执行失败: " + err.Error(),
			Duration: duration.String(),
		})
		return
	}

	c.JSON(http.StatusOK, CommandResponse{
		Success:  true,
		Message:  "批量命令执行完成",
		Result:   strings.Join(results, "\n"),
		Duration: duration.String(),
	})
}

// handleStatus 处理状态检查请求
func (fc *FastCommand) handleStatus(c *gin.Context) {
	status := map[string]interface{}{
		"ready":        fc.isReady,
		"last_command": fc.lastCommand,
		"window_found": fc.hwnd != 0,
		"chat_active":  fc.isChatActive(),
		"chat_mode":    fc.getCurrentChatMode(),
	}

	c.JSON(http.StatusOK, status)
}

// handlePresets 处理预设命令请求
func (fc *FastCommand) handlePresets(c *gin.Context) {
	presets := map[string]interface{}{
		"aliases": fc.getCommandAliases(),
		"common_commands": []string{
			"#ListPlayers",
			"#ListSpawnedVehicles",
			"#SetTime 12 00",
			"#listflags 1 true",
			"#dumpallsquadsinfolist",
		},
		"time_presets": map[string]string{
			"morning": "#SetTime 08 00",
			"noon":    "#SetTime 12 00",
			"evening": "#SetTime 18 00",
			"night":   "#SetTime 22 00",
		},
	}

	c.JSON(http.StatusOK, presets)
}

// handleStats 处理统计信息请求
func (fc *FastCommand) handleStats(c *gin.Context) {
	stats := map[string]interface{}{
		"ready":                  fc.isReady,
		"last_command":           fc.lastCommand,
		"smart_mode_enabled":     fc.enableSmartMode,
		"batch_optimization":     fc.enableBatchOptimization,
		"preferred_input_method": fc.preferredInputMethod,
		"preferred_chat_method":  fc.preferredChatMethod,
	}

	// 如果增强执行器可用，获取详细统计
	if fc.enhancedExecutor != nil {
		enhancedStats := fc.enhancedExecutor.GetExecutionStats()
		stats["enhanced_stats"] = enhancedStats
	}

	c.JSON(http.StatusOK, stats)
}

// handleConfig 处理配置更新请求
func (fc *FastCommand) handleConfig(c *gin.Context) {
	var config struct {
		PreferredInputMethod    *util.InputMethod          `json:"preferred_input_method,omitempty"`
		PreferredChatMethod     *util.ChatActivationMethod `json:"preferred_chat_method,omitempty"`
		EnableSmartMode         *bool                      `json:"enable_smart_mode,omitempty"`
		EnableBatchOptimization *bool                      `json:"enable_batch_optimization,omitempty"`
	}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "配置格式错误: " + err.Error(),
		})
		return
	}

	// 更新配置
	if config.PreferredInputMethod != nil {
		fc.preferredInputMethod = *config.PreferredInputMethod
		if fc.enhancedExecutor != nil {
			fc.enhancedExecutor.SetDefaultMethods(fc.preferredInputMethod, fc.preferredChatMethod)
		}
	}

	if config.PreferredChatMethod != nil {
		fc.preferredChatMethod = *config.PreferredChatMethod
		if fc.enhancedExecutor != nil {
			fc.enhancedExecutor.SetDefaultMethods(fc.preferredInputMethod, fc.preferredChatMethod)
		}
	}

	if config.EnableSmartMode != nil {
		fc.enableSmartMode = *config.EnableSmartMode
	}

	if config.EnableBatchOptimization != nil {
		fc.enableBatchOptimization = *config.EnableBatchOptimization
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已更新",
		"config": map[string]interface{}{
			"preferred_input_method":    fc.preferredInputMethod,
			"preferred_chat_method":     fc.preferredChatMethod,
			"enable_smart_mode":         fc.enableSmartMode,
			"enable_batch_optimization": fc.enableBatchOptimization,
		},
	})
}

// handleMethods 处理输入方法信息请求
func (fc *FastCommand) handleMethods(c *gin.Context) {
	methods := map[string]interface{}{
		"input_methods": map[string]int{
			"SIMULATE_KEY":    int(util.INPUT_SIMULATE_KEY),
			"WINDOW_MSG":      int(util.INPUT_WINDOW_MSG),
			"UI_AUTOMATION":   int(util.INPUT_UI_AUTOMATION),
			"CLIPBOARD_PASTE": int(util.INPUT_CLIPBOARD_PASTE),
			"HYBRID":          int(util.INPUT_HYBRID),
		},
		"chat_methods": map[string]int{
			"T_KEY":      int(util.CHAT_ACTIVATE_T_KEY),
			"SLASH_KEY":  int(util.CHAT_ACTIVATE_SLASH_KEY),
			"WINDOW_MSG": int(util.CHAT_ACTIVATE_WINDOW_MSG),
		},
		"descriptions": map[string]string{
			"SIMULATE_KEY":    "模拟按键输入",
			"WINDOW_MSG":      "窗口消息直接发送",
			"UI_AUTOMATION":   "UI自动化接口",
			"CLIPBOARD_PASTE": "剪贴板粘贴",
			"HYBRID":          "智能选择最佳方式",
			"T_KEY":           "T键激活聊天",
			"SLASH_KEY":       "/键激活命令",
		},
	}

	// 如果输入管理器可用，获取方法统计
	if fc.inputManager != nil {
		methodStats := fc.inputManager.GetMethodStats()
		methods["method_stats"] = methodStats
	}

	c.JSON(http.StatusOK, methods)
}

// handleTest 处理测试请求
func (fc *FastCommand) handleTest(c *gin.Context) {
	var req struct {
		InputMethod util.InputMethod          `json:"input_method"`
		ChatMethod  util.ChatActivationMethod `json:"chat_method"`
		TestText    string                    `json:"test_text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "测试请求格式错误: " + err.Error(),
		})
		return
	}

	if req.TestText == "" {
		req.TestText = "#ListPlayers"
	}

	if !fc.isReady {
		if err := fc.Initialize(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "初始化失败: " + err.Error(),
			})
			return
		}
	}

	start := time.Now()

	// 测试聊天激活
	chatErr := fc.inputManager.ActivateChat(req.ChatMethod)

	var inputErr error
	if chatErr == nil {
		// 测试文本输入
		inputErr = fc.inputManager.SendText(req.TestText, req.InputMethod)
	}

	duration := time.Since(start)

	c.JSON(http.StatusOK, gin.H{
		"success":         (chatErr == nil && inputErr == nil),
		"chat_activation": (chatErr == nil),
		"text_input":      (inputErr == nil),
		"duration":        duration.String(),
		"chat_error": func() string {
			if chatErr != nil {
				return chatErr.Error()
			} else {
				return ""
			}
		}(),
		"input_error": func() string {
			if inputErr != nil {
				return inputErr.Error()
			} else {
				return ""
			}
		}(),
		"input_method": req.InputMethod,
		"chat_method":  req.ChatMethod,
		"test_text":    req.TestText,
	})
}

// handleContinuousStart 处理开始连续会话请求
func (fc *FastCommand) handleContinuousStart(c *gin.Context) {
	if !fc.isReady {
		if err := fc.Initialize(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "初始化失败: " + err.Error(),
			})
			return
		}
	}

	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	err := fc.continuousExecutor.StartContinuousSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "启动连续会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "连续会话已启动",
		"session_info": fc.continuousExecutor.GetCurrentSessionInfo(),
	})
}

// handleContinuousAdd 处理在连续会话中添加命令请求
func (fc *FastCommand) handleContinuousAdd(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求格式错误: " + err.Error(),
		})
		return
	}

	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	start := time.Now()
	err := fc.continuousExecutor.AddCommandToContinuousSession(req.Command)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"message":  "添加命令失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "命令已添加到连续会话",
		"command":      req.Command,
		"duration":     duration.String(),
		"session_info": fc.continuousExecutor.GetCurrentSessionInfo(),
	})
}

// handleContinuousBatch 处理连续批量执行请求
func (fc *FastCommand) handleContinuousBatch(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求格式错误: " + err.Error(),
		})
		return
	}

	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	start := time.Now()
	err := fc.continuousExecutor.ExecuteContinuousBatch(req.Commands)
	duration := time.Since(start)

	stats := fc.continuousExecutor.GetSessionStats()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"message":  "连续批量执行失败: " + err.Error(),
			"duration": duration.String(),
			"stats":    stats,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "连续批量执行成功",
		"commands_count": len(req.Commands),
		"duration":       duration.String(),
		"stats":          stats,
	})
}

// handleContinuousEnd 处理结束连续会话请求
func (fc *FastCommand) handleContinuousEnd(c *gin.Context) {
	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	sessionInfo := fc.continuousExecutor.GetCurrentSessionInfo()
	fc.continuousExecutor.EndContinuousSession()

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"message":            "连续会话已结束",
		"final_session_info": sessionInfo,
	})
}

// handleContinuousStatus 处理连续会话状态查询请求
func (fc *FastCommand) handleContinuousStatus(c *gin.Context) {
	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	status := map[string]interface{}{
		"is_session_active":       fc.continuousExecutor.IsSessionActive(),
		"current_session":         fc.continuousExecutor.GetCurrentSessionInfo(),
		"session_stats":           fc.continuousExecutor.GetSessionStats(),
		"continuous_mode_enabled": fc.enableContinuousMode,
	}

	c.JSON(http.StatusOK, status)
}

// handleContinuousSequence 处理预定义序列执行请求
func (fc *FastCommand) handleContinuousSequence(c *gin.Context) {
	var req struct {
		SequenceName string `json:"sequence_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求格式错误: " + err.Error(),
		})
		return
	}

	if fc.continuousExecutor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "连续执行器未可用",
		})
		return
	}

	start := time.Now()
	err := fc.continuousExecutor.ExecuteQuickSequence(req.SequenceName)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"message":       "执行预定义序列失败: " + err.Error(),
			"sequence_name": req.SequenceName,
			"duration":      duration.String(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "预定义序列执行成功",
		"sequence_name": req.SequenceName,
		"duration":      duration.String(),
		"stats":         fc.continuousExecutor.GetSessionStats(),
	})
}
