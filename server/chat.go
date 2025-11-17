package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"qq_client/global"
	"qq_client/util"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var flagsRegexp, _ = regexp.Compile("Page (\\d+)/(\\d+)")
var updateClipboard = map[string]bool{
	"#listflags 1 true":         true,
	"#ListSpawnedVehicles true": true,
	"#dumpallsquadsinfolist":    true,
	"#ListPlayers true":         true,
}

// 日志文件句柄
var logFile *os.File
var logger *log.Logger

// 定时指令状态追踪
var lastPeriodicCommandTime time.Time
var currentPeriodicCommand string
var lastClipboardContent string

// 批量指令获取缓存
var commandBatch []string
var lastCommandFetchTime time.Time

// 执行性能优化相关变量
var commandStats map[string]*CommandStats
var lastResponseTimes map[string]time.Duration
var preProcessedCommands map[string]string

// CommandStats 指令执行统计
type CommandStats struct {
	TotalExecutions int
	AverageTime     time.Duration
	LastExecuteTime time.Time
	SuccessRate     float64
}

// 初始化日志系统
func init() {
	// 创建logs目录
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	// 创建日志文件，按日期命名
	logFileName := fmt.Sprintf("scum_client_%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logsDir, logFileName)

	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("创建日志文件失败: %v\n", err)
		return
	}

	// 创建多重写入器，同时输出到控制台和文件
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
	logger.Printf("=== SCUM Client 启动 ===")

	// 初始化性能优化相关变量
	commandStats = make(map[string]*CommandStats)
	lastResponseTimes = make(map[string]time.Duration)
	preProcessedCommands = make(map[string]string)
	lastPeriodicCommandTime = time.Now()
}

// 统一的日志函数
func logInfo(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[INFO] "+format, v...)
	} else {
		fmt.Printf("[INFO] "+format+"\n", v...)
	}
}

func logError(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[ERROR] "+format, v...)
	} else {
		fmt.Printf("[ERROR] "+format+"\n", v...)
	}
}

func logDebug(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("[DEBUG] "+format, v...)
	} else {
		fmt.Printf("[DEBUG] "+format+"\n", v...)
	}
}

// 检查是否在聊天界面的更可靠方法
func isChatInterfaceOpen(hand syscall.Handle) bool {
	// 检查MUTE按钮是否存在（聊天界面的标志）
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "MUTE") == nil {
		return true
	}
	return false
}

// 获取当前聊天模式
func getCurrentChatMode(hand syscall.Handle) string {
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "GLOBAL") == nil {
		return "GLOBAL"
	}
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "LOCAL") == nil {
		return "LOCAL"
	}
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "ADMIN") == nil {
		return "ADMIN"
	}
	return "UNKNOWN"
}

// 优化的聊天框激活函数
func ensureChatBoxActive(hand syscall.Handle) bool {
	logDebug("开始检查聊天框状态")

	// 设置窗口为前台
	util.SetForegroundWindow(hand)
	time.Sleep(200 * time.Millisecond)

	// 首先检查是否已经在聊天界面
	if !isChatInterfaceOpen(hand) {
		logDebug("聊天界面未打开，尝试按T键激活")

		// 先按ESC确保退出任何菜单
		_ = robotgo.KeyTap("escape")
		time.Sleep(100 * time.Millisecond)

		// 按T激活聊天
		_ = robotgo.KeyTap("t")
		time.Sleep(500 * time.Millisecond)

		// 验证是否成功激活
		if !isChatInterfaceOpen(hand) {
			logError("按T后仍无法激活聊天界面")
			return false
		}
		logDebug("聊天界面已激活")
	} else {
		logDebug("聊天界面已经处于激活状态")
	}

	// 检查并切换到GLOBAL模式
	currentMode := getCurrentChatMode(hand)
	logDebug("当前聊天模式: %s", currentMode)

	maxAttempts := 5
	for i := 0; i < maxAttempts && currentMode != "GLOBAL"; i++ {
		logDebug("尝试切换到GLOBAL模式，当前: %s，尝试次数: %d", currentMode, i+1)

		switch currentMode {
		case "LOCAL":
			_ = robotgo.KeyTap("tab")
			time.Sleep(300 * time.Millisecond)
		case "ADMIN":
			_ = robotgo.KeyTap("tab")
			time.Sleep(150 * time.Millisecond)
			_ = robotgo.KeyTap("tab")
			time.Sleep(300 * time.Millisecond)
		case "UNKNOWN":
			logError("未知聊天模式，尝试按tab切换")
			_ = robotgo.KeyTap("tab")
			time.Sleep(300 * time.Millisecond)
		}

		// 重新检查模式
		currentMode = getCurrentChatMode(hand)
	}

	if currentMode == "GLOBAL" {
		logDebug("聊天框已切换到GLOBAL模式")
		return true
	} else {
		logError("无法切换到GLOBAL模式，当前模式: %s", currentMode)
		return false
	}
}

// 清空剪贴板并验证
func clearClipboard() error {
	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		_ = clipboard.WriteAll("")
		time.Sleep(50 * time.Millisecond)

		if content, err := clipboard.ReadAll(); err == nil && content == "" {
			logDebug("剪贴板已清空")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	logError("清空剪贴板失败，尝试了%d次", maxAttempts)
	return errors.New("无法清空剪贴板")
}

// 写入剪贴板并验证
func writeToClipboard(text string) error {
	maxAttempts := 8
	for i := 0; i < maxAttempts; i++ {
		_ = clipboard.WriteAll(text)
		time.Sleep(100 * time.Millisecond)

		if content, err := clipboard.ReadAll(); err == nil && content == text {
			logDebug("剪贴板写入成功: %s", text[:min(50, len(text))])
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	logError("剪贴板写入失败，尝试了%d次", maxAttempts)
	return errors.New("剪贴板写入失败")
}

// 辅助函数：获取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// preProcessCommand 预处理指令，减少执行时的处理时间
func preProcessCommand(text string) string {
	if processed, exists := preProcessedCommands[text]; exists {
		return processed
	}

	var commandToSend string
	if regexpList := global.ExtractLocationRegexp.FindAllString(text, -1); len(regexpList) == 6 {
		commandToSend = regexpList[0]
	} else {
		commandToSend = text
	}

	// 缓存预处理结果
	preProcessedCommands[text] = commandToSend
	return commandToSend
}

// getOptimalWaitTime 根据历史数据获取最优等待时间
func getOptimalWaitTime(command string) time.Duration {
	if lastTime, exists := lastResponseTimes[command]; exists {
		// 基于历史响应时间优化等待时间
		switch command {
		case "#ListPlayers true":
			return time.Duration(min(int(lastTime*12/10), int(3*time.Second))) // 历史时间*1.2，最多3秒
		case "#ListSpawnedVehicles true":
			return time.Duration(min(int(lastTime*11/10), int(2*time.Second))) // 历史时间*1.1，最多2秒
		case "#dumpallsquadsinfolist":
			return time.Duration(min(int(lastTime*13/10), int(4*time.Second))) // 历史时间*1.3，最多4秒
		default:
			if strings.Contains(command, "#listflags") {
				return time.Duration(min(int(lastTime*12/10), int(3*time.Second)))
			}
		}
	}

	// 默认等待时间（优化后的较短时间）
	switch command {
	case "#ListPlayers true":
		return 1200 * time.Millisecond // 从2秒减少到1.2秒
	case "#ListSpawnedVehicles true":
		return 1000 * time.Millisecond // 从1.5秒减少到1秒
	case "#dumpallsquadsinfolist":
		return 2500 * time.Millisecond // 从3秒减少到2.5秒
	default:
		if strings.Contains(command, "#listflags") {
			return 1500 * time.Millisecond // 从2秒减少到1.5秒
		}
		return 800 * time.Millisecond // 从1秒减少到0.8秒
	}
}

// updateCommandStats 更新指令执行统计
func updateCommandStats(command string, duration time.Duration, success bool) {
	if stats, exists := commandStats[command]; exists {
		stats.TotalExecutions++
		// 计算平均响应时间
		stats.AverageTime = (stats.AverageTime*time.Duration(stats.TotalExecutions-1) + duration) / time.Duration(stats.TotalExecutions)
		stats.LastExecuteTime = time.Now()

		// 更新成功率
		if success {
			stats.SuccessRate = (stats.SuccessRate*float64(stats.TotalExecutions-1) + 1) / float64(stats.TotalExecutions)
		} else {
			stats.SuccessRate = stats.SuccessRate * float64(stats.TotalExecutions-1) / float64(stats.TotalExecutions)
		}
	} else {
		commandStats[command] = &CommandStats{
			TotalExecutions: 1,
			AverageTime:     duration,
			LastExecuteTime: time.Now(),
			SuccessRate: func() float64 {
				if success {
					return 1.0
				}
				return 0.0
			}(),
		}
	}

	// 更新历史响应时间
	if success {
		lastResponseTimes[command] = duration
	}
}

// fastClipboardOperation 快速剪贴板操作
func fastClipboardOperation(text string) error {
	// 使用更激进的重试策略
	maxAttempts := 3
	for i := 0; i < maxAttempts; i++ {
		_ = clipboard.WriteAll(text)

		// 减少验证时间
		time.Sleep(30 * time.Millisecond)

		if content, err := clipboard.ReadAll(); err == nil && content == text {
			return nil
		}

		if i < maxAttempts-1 {
			time.Sleep(20 * time.Millisecond)
		}
	}

	return errors.New("快速剪贴板操作失败")
}

// parallelSquadSend 并行发送squad数据，不阻塞主流程
func parallelSquadSend(body map[string]interface{}) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logError("并行发送squad数据时发生panic: %v", r)
			}
		}()

		var err error
		var byteBody []byte
		var req *http.Request
		var resp *http.Response

		// 使用更短的超时时间
		httpClient := &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				DisableKeepAlives:   false,
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
			},
		}

		if byteBody, err = json.Marshal(&body); err == nil {
			if req, err = http.NewRequest("POST", fmt.Sprintf(
				"%s/api/v1/squad", global.ScumConfig.ServerUrl), bytes.NewReader(byteBody)); err == nil {
				req.Header.Set("Content-Type", "application/json")
				if resp, err = httpClient.Do(req); err == nil {
					_, _ = io.ReadAll(resp.Body)
					resp.Body.Close()
				}
			}
		}

		if err != nil {
			logDebug("并行发送squad数据失败: %v", err)
		}
	}()
}

// run
// @author: [Fantasia](https://www.npc0.com)
// @function: run
// @description: 获取运行命令
func run() string {
	// init
	var err error
	var bodyBytes []byte
	var req *http.Request
	var resp *http.Response
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	if req, err = http.NewRequest("GET", fmt.Sprintf(
		"%s/api/v1/run?id=%d", global.ScumConfig.ServerUrl, global.ScumConfig.ServerID), nil); err == nil {
		req.Header.Set("Content-Type", "application/json")
		if resp, err = httpClient.Do(req); err == nil {
			if bodyBytes, err = io.ReadAll(resp.Body); err == nil {
				return string(bodyBytes)
			}
		}
	}
	return ""
}

// getAllPendingCommands
// @author: [Fantasia](https://www.npc0.com)
// @function: getAllPendingCommands
// @description: 获取当前服务器的所有待处理指令
func getAllPendingCommands() []string {
	// 检查缓存是否过期（30秒刷新一次）
	if time.Since(lastCommandFetchTime) < 30*time.Second && len(commandBatch) > 0 {
		return commandBatch
	}

	// init
	var err error
	var bodyBytes []byte
	var req *http.Request
	var resp *http.Response
	var commands []string

	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	// 获取批量指令
	if req, err = http.NewRequest("GET", fmt.Sprintf(
		"%s/api/v1/run/batch?id=%d", global.ScumConfig.ServerUrl, global.ScumConfig.ServerID), nil); err == nil {
		req.Header.Set("Content-Type", "application/json")
		if resp, err = httpClient.Do(req); err == nil {
			if bodyBytes, err = io.ReadAll(resp.Body); err == nil {
				// 解析JSON数组
				if err = json.Unmarshal(bodyBytes, &commands); err == nil {
					commandBatch = commands
					lastCommandFetchTime = time.Now()
					logDebug("批量获取到 %d 条指令", len(commands))
					return commands
				}
			}
		}
	}

	// 如果批量获取失败，回退到单条获取
	if singleCommand := run(); singleCommand != "" {
		commands = []string{singleCommand}
		commandBatch = commands
		lastCommandFetchTime = time.Now()
	}

	return commands
}

// executePeriodicCommands
// @author: [Fantasia](https://www.npc0.com)
// @function: executePeriodicCommands
// @description: 执行定时指令（每分钟执行的三个固定指令）- 高速优化版本
func executePeriodicCommands(hwnd syscall.Handle) {
	// 检查是否到了执行时间（每分钟执行一次）
	if time.Since(lastPeriodicCommandTime) < 60*time.Second {
		return
	}

	// 自建服务器和命令行服务器不执行这三个固定指令（由scum_run自动推送）
	if global.ScumConfig.FtpProvider == global.FtpProviderSelfBuilt || global.ScumConfig.FtpProvider == global.FtpProviderCommandLine {
		logDebug("自建服务器或命令行服务器类型，跳过定时指令执行（由scum_run自动推送）")
		lastPeriodicCommandTime = time.Now()
		return
	}

	startTime := time.Now()
	logInfo("开始高速执行定时指令...")

	// 定义三个固定指令
	periodicCommands := []string{
		"#ListPlayers true",
		"#ListSpawnedVehicles true",
		"#dumpallsquadsinfolist",
	}

	// 激活聊天框
	if !ensureChatBoxActive(hwnd) {
		logError("无法激活聊天框进行定时指令执行")
		return
	}

	successCount := 0
	// 高速依次执行每个指令
	for i, command := range periodicCommands {
		logInfo("高速执行定时指令 [%d/%d]: %s", i+1, len(periodicCommands), command)

		// 发送指令
		if out, err := Send(hwnd, command); err != nil {
			logError("定时指令执行失败 %s: %v", command, err)
			continue
		} else if out != "" {
			successCount++
			// 并行发送结果到服务器，不阻塞主流程
			var mode string
			switch command {
			case "#ListPlayers true":
				mode = "user"
			case "#ListSpawnedVehicles true":
				mode = "spawned"
			case "#dumpallsquadsinfolist":
				mode = "all_group"
			}

			if mode != "" {
				// 使用并行发送，提升性能
				parallelSquadSend(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": mode,
					"info": out,
				})
				logDebug("定时指令结果已并行发送: %s, 数据长度: %d", command, len(out))
			}
		}

		// 动态调整指令间隔（根据执行成功率）
		if i < len(periodicCommands)-1 {
			if successCount > 1 && float64(successCount)/float64(i+1) > 0.8 {
				time.Sleep(1200 * time.Millisecond) // 成功率高时减少间隔
			} else {
				time.Sleep(1800 * time.Millisecond) // 从2秒减少到1.8秒
			}
		}
	}

	// 关闭聊天框
	duration := time.Since(startTime)
	logInfo("定时指令执行完毕，成功: %d/%d，耗时: %v，关闭聊天框",
		successCount, len(periodicCommands), duration)

	_ = robotgo.KeyTap("escape")
	time.Sleep(200 * time.Millisecond) // 从300ms减少到200ms

	// 更新最后执行时间
	lastPeriodicCommandTime = time.Now()
}

// ChatMonitorWithActivation
// @author: [Fantasia](https://www.npc0.com)
// @function: ChatMonitorWithActivation
// @description: 带激活功能的聊天监控 - 高速优化版本
func ChatMonitorWithActivation(hwnd syscall.Handle) {
	logInfo("开始智能聊天监控（高速按需激活模式）...")

	for {
		// 获取所有待处理指令
		commands := getAllPendingCommands()

		if len(commands) > 0 {
			batchStartTime := time.Now()
			logInfo("检测到 %d 条待处理指令，启动高速批量处理", len(commands))

			// 激活聊天框
			if !ensureChatBoxActive(hwnd) {
				logError("无法激活聊天框")
				time.Sleep(2 * time.Second) // 从5秒减少到2秒
				continue
			}

			// 高速批量执行指令
			successCount := 0
			for i, command := range commands {
				if command == "" {
					continue
				}

				logInfo("高速执行指令 [%d/%d]: %s", i+1, len(commands), command)

				var out string
				var err error
				if out, err = Send(hwnd, command); err != nil {
					logError("指令执行失败: %v", err)
					// 快速重试一次
					time.Sleep(300 * time.Millisecond) // 从1秒减少到300ms
					if out, err = Send(hwnd, command); err != nil {
						logError("快速重试失败: %v", err)
						continue
					}
				}

				// 异步处理指令结果，不阻塞主流程
				if out != "" {
					go func(result string) {
						SaveChat(result)
						logDebug("指令结果已异步保存，长度: %d", len(result))
					}(out)
				}

				successCount++

				// 动态调整指令间隔
				if i < len(commands)-1 {
					// 根据指令类型和成功率动态调整间隔
					if successCount > 5 && float64(successCount)/float64(i+1) > 0.8 {
						time.Sleep(200 * time.Millisecond) // 成功率高时减少间隔
					} else {
						time.Sleep(350 * time.Millisecond) // 从500ms减少到350ms
					}
				}
			}

			// 执行完所有指令后关闭聊天框
			batchDuration := time.Since(batchStartTime)
			logInfo("批量指令执行完毕，成功: %d/%d，耗时: %v，关闭聊天框",
				successCount, len(commands), batchDuration)

			_ = robotgo.KeyTap("escape")
			time.Sleep(200 * time.Millisecond) // 从300ms减少到200ms

			// 清空指令缓存
			commandBatch = []string{}
			lastCommandFetchTime = time.Time{}
		}

		// 执行定时指令
		executePeriodicCommands(hwnd)

		// 动态等待时间（根据当前负载调整）
		if len(commands) > 10 {
			time.Sleep(1500 * time.Millisecond) // 高负载时减少检查频率
		} else {
			time.Sleep(2500 * time.Millisecond) // 从3秒减少到2.5秒
		}
	}
}

// squad
// @author: [Fantasia](https://www.npc0.com)
// @function: Send
// @description: 发送回调 - 保持兼容性，内部使用并行发送
func squad(body map[string]interface{}) {
	// 使用并行发送提升性能
	parallelSquadSend(body)
}

// SaveChat
// @author: [Fantasia](https://www.npc0.com)
// @function: SaveChat
// @description: 回写聊天信息
func SaveChat(text string) {
	// init
	var err error
	var byteBody []byte
	var req *http.Request
	var resp *http.Response
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	if byteBody, err = json.Marshal(&text); err == nil {
		if req, err = http.NewRequest("POST", fmt.Sprintf(
			"%s/api/v1/recycling", global.ScumConfig.ServerUrl), bytes.NewReader(byteBody)); err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err = httpClient.Do(req); err == nil {
				_, _ = io.ReadAll(resp.Body)
			}
		}
	}
}

// Send
// @author: [Fantasia](https://www.npc0.com)
// @function: Send
// @description: 发送命令 - 高速优化版本
func Send(hwnd syscall.Handle, text string) (out string, err error) {
	startTime := time.Now()
	logInfo("开始发送指令: %s", text)

	// 验证输入参数
	if strings.Contains(text, "502 Bad Gateway") {
		logDebug("检测到502错误，跳过处理")
		return "", nil
	}

	// 设置窗口为前台并确保聊天框激活
	if !ensureChatBoxActive(hwnd) {
		logError("无法激活聊天框")
		return "", errors.New("无法激活聊天框")
	}

	// 使用预处理的指令，减少正则匹配时间
	commandToSend := preProcessCommand(text)

	// 验证指令长度
	if len(commandToSend) > 200 {
		logError("指令长度超限: %d", len(commandToSend))
		return "", errors.New("指令长度超限")
	}

	// 第一步：快速清空剪贴板（减少等待时间）
	_ = clipboard.WriteAll("")
	time.Sleep(30 * time.Millisecond) // 从原来的多次验证改为快速操作

	// 第二步：快速清空输入框（优化时序）
	robotgo.MoveClick(82, 319, "", false)
	time.Sleep(80 * time.Millisecond) // 从150ms减少到80ms
	_ = robotgo.KeyTap("a", "ctrl")
	time.Sleep(30 * time.Millisecond) // 从50ms减少到30ms
	_ = robotgo.KeyTap("delete")
	time.Sleep(80 * time.Millisecond) // 从150ms减少到80ms

	// 第三步：快速写入指令到剪贴板
	if err = fastClipboardOperation(commandToSend); err != nil {
		logError("快速剪贴板写入失败，回退到标准方式: %v", err)
		// 回退到标准方式
		if err = writeToClipboard(commandToSend); err != nil {
			logError("写入剪贴板失败: %v", err)
			return "", err
		}
	}

	// 第四步：快速粘贴并发送指令
	_ = robotgo.KeyTap("v", "ctrl")
	time.Sleep(120 * time.Millisecond) // 从200ms减少到120ms
	_ = robotgo.KeyTap("enter")

	logInfo("指令已发送: %s", commandToSend)

	// 第五步：对于需要返回结果的指令，智能等待响应
	if _, needsResponse := updateClipboard[commandToSend]; needsResponse {
		logInfo("等待指令响应: %s", commandToSend)

		// 立即清空剪贴板，准备接收游戏返回的结果
		_ = clipboard.WriteAll("")
		time.Sleep(20 * time.Millisecond) // 减少等待时间

		// 使用智能等待时间
		waitTime := getOptimalWaitTime(commandToSend)
		logDebug("智能等待响应时间: %v", waitTime)

		// 分段等待，提前检查响应
		waitSteps := 4
		stepTime := waitTime / time.Duration(waitSteps)

		for step := 0; step < waitSteps; step++ {
			time.Sleep(stepTime)

			// 每个步骤都检查一次响应
			if out, err = clipboard.ReadAll(); err == nil && out != "" && out != commandToSend {
				responseTime := time.Since(startTime)
				logInfo("快速获取响应 (步骤%d/%d)，长度: %d，耗时: %v", step+1, waitSteps, len(out), responseTime)
				updateCommandStats(commandToSend, responseTime, true)
				return out, nil
			}
		}

		// 如果分段等待没有结果，进行快速重试
		maxAttempts := 3 // 从5次减少到3次
		for i := 0; i < maxAttempts; i++ {
			if out, err = clipboard.ReadAll(); err == nil && out != "" && out != commandToSend {
				responseTime := time.Since(startTime)
				logInfo("重试获取响应成功，长度: %d，耗时: %v", len(out), responseTime)
				updateCommandStats(commandToSend, responseTime, true)
				return out, nil
			}

			// 减少重试间隔
			if i < maxAttempts-1 {
				time.Sleep(200 * time.Millisecond) // 从500ms减少到200ms
			}
		}

		// 快速检查聊天框状态
		if !isChatInterfaceOpen(hwnd) {
			logError("聊天框丢失")
			updateCommandStats(commandToSend, time.Since(startTime), false)
			return "", errors.New("聊天框状态异常")
		}

		// 最后一次快速尝试
		if out, err = clipboard.ReadAll(); err == nil && out != "" && out != commandToSend {
			responseTime := time.Since(startTime)
			logInfo("最终获取到响应，长度: %d，耗时: %v", len(out), responseTime)
			updateCommandStats(commandToSend, responseTime, true)
			return out, nil
		}

		logError("未获取到有效响应，剪贴板内容: %s", out)
		updateCommandStats(commandToSend, time.Since(startTime), false)
	} else {
		// 对于不需要响应的指令，记录执行时间
		updateCommandStats(commandToSend, time.Since(startTime), true)
	}

	return out, nil
}

// ChatMonitor
// @author: [Fantasia](https://www.npc0.com)
// @function: ChatMonitor
// @description: 聊天监控信息 - 优化版本
func ChatMonitor(hwnd syscall.Handle) {
	// init
	var i int
	var err error
	var out string

	logInfo("开始聊天监控...")

	// 初始化传送指令
	_, _ = Send(hwnd, "#Teleport 0 0 0")

	for {
		// 延时
		time.Sleep(150 * time.Millisecond)

		// 获取并执行服务器指令
		if command := run(); command != "" {
			logInfo("收到服务器指令: %s", command)
			if out, err = Send(hwnd, command); err != nil {
				logError("执行指令失败: %v，尝试重试", err)
				// 重试一次
				time.Sleep(1 * time.Second)
				if out, err = Send(hwnd, command); err != nil {
					logError("重试失败，退出监控: %v", err)
					return
				}
			}
			logInfo("指令执行完成")
		}

		// 定时获取载具和玩家信息（每15次循环 = 约2.25秒）
		if i%15 == 0 {
			logDebug("开始获取载具和玩家信息...")

			// 获取载具列表
			if out, err = Send(hwnd, "#ListSpawnedVehicles true"); err != nil {
				logError("获取载具列表失败: %v", err)
				return
			} else if out != "" {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "spawned",
					"info": out,
				})
				logDebug("载具信息已发送，数据长度: %d", len(out))
			}

			// 获取玩家列表
			if out, err = Send(hwnd, "#ListPlayers true"); err != nil {
				logError("获取玩家列表失败: %v", err)
				return
			} else if out != "" {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "user",
					"info": out,
				})
				logDebug("玩家信息已发送，数据长度: %d", len(out))
			}
		}

		// 定时获取领地和队伍信息（每150次循环 = 约22.5秒）
		if i%150 == 0 && i > 0 {
			logDebug("开始获取领地和队伍信息...")

			// 获取领地信息
			var flagNum = 1
			for {
				flagCommand := fmt.Sprintf("#listflags %d true", flagNum)
				if out, err = Send(hwnd, flagCommand); err != nil {
					logError("获取领地信息失败 %s: %v", flagCommand, err)
					return
				} else {
					// 发送领地信息
					squad(map[string]interface{}{
						"id":   global.ScumConfig.ServerID,
						"mode": "flags",
						"info": out,
					})

					// 判断是否为最后一页
					if lenList := flagsRegexp.FindAllStringSubmatch(out, 1); len(lenList) > 0 {
						if lenList[0][1] == lenList[0][2] {
							logDebug("领地信息获取完成，共%s页", lenList[0][2])
							break
						}
					}

					// 检查输出长度，如果太短可能是错误
					if len(out) < 10 {
						logError("领地信息输出异常: %s", out)
						break
					}

					flagNum++
					// 添加页面间隔时间，避免过快请求
					time.Sleep(500 * time.Millisecond)
				}
			}

			// 获取队伍信息
			if out, err = Send(hwnd, "#dumpallsquadsinfolist"); err != nil {
				logError("获取队伍信息失败: %v", err)
				return
			} else if out != "" {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "all_group",
					"info": out,
				})
				logDebug("队伍信息已发送，数据长度: %d", len(out))
			}
		}

		// 循环计数器递增
		i++
		if i > 1000 {
			i = 0
		}
	}
}

// 程序退出时关闭日志文件
func CloseLog() {
	if logFile != nil {
		logFile.Close()
	}
}
