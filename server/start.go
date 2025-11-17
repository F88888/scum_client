package server

import (
	"github.com/go-vgo/robotgo"
	"os/exec"
	"qq_client/global"
	"qq_client/util"
	"syscall"
	"time"
)

// 错误计算
var errorNumber int
var errorNumber2 int

// 窗口位置缓存
var lastWindowX, lastWindowY int = -1, -1
var lastWindowWidth, lastWindowHeight int = -1, -1

// 添加聊天框状态追踪
var lastChatState = "UNKNOWN"
var chatStateStableCount = 0

// 添加待处理指令队列状态
var hasPendingCommands = false

// 添加配置替换标记
var configReplaced = false

// isWindowInCorrectPosition 检查窗口是否在正确位置
func isWindowInCorrectPosition(hand syscall.Handle) bool {
	if hand == 0 {
		return false
	}

	// 获取当前窗口位置
	// 这里可以添加具体的窗口位置检查逻辑
	// 暂时返回 false 以触发位置设置
	return false
}

// setWindowPositionOnce 只在必要时设置窗口位置
func setWindowPositionOnce(hand syscall.Handle) {
	// 如果位置已经正确，跳过设置
	if lastWindowX == global.GameWindowX && lastWindowY == global.GameWindowY &&
		lastWindowWidth == global.GameWindowWidth && lastWindowHeight == global.GameWindowHeight {
		return
	}

	logInfo("设置窗口位置和大小...")
	util.MoveWindow(hand, global.GameWindowX, global.GameWindowY, global.GameWindowWidth, global.GameWindowHeight)

	// 更新缓存
	lastWindowX, lastWindowY = global.GameWindowX, global.GameWindowY
	lastWindowWidth, lastWindowHeight = global.GameWindowWidth, global.GameWindowHeight

	// 等待窗口稳定
	time.Sleep(500 * time.Millisecond)
}

// ErrorReboot
// @author: [Fantasia](https://www.npc0.com)
// @function: ErrorReboot
// @description: 错误重启
func ErrorReboot() {
	// 判断错误次数
	if errorNumber > 15 || errorNumber2 > 100 {
		// 错误次数大于15，重启游戏
		logError("错误次数过多 (errorNumber: %d, errorNumber2: %d)，重启游戏", errorNumber, errorNumber2)
		cmd := exec.Command("taskkill", "/IM", "SCUM.exe", "/F")
		_ = cmd.Run()
		// 重置错误计数器
		errorNumber = 0
		errorNumber2 = 0
		// 重置窗口位置缓存
		lastWindowX, lastWindowY = -1, -1
		lastWindowWidth, lastWindowHeight = -1, -1
		// 重置聊天状态
		lastChatState = "UNKNOWN"
		chatStateStableCount = 0
		// 重置指令队列状态
		hasPendingCommands = false
		// 重置配置替换标记
		configReplaced = false
		// 清空文本位置缓存
		util.ClearTextPositionCache()
	}
}

// 检查是否有待处理的服务器指令
func checkPendingCommands() bool {
	commands := getAllPendingCommands()
	return len(commands) > 0
}

// 检查游戏当前状态
func checkGameState(hand syscall.Handle) string {
	// 1. 检查是否在登录页面
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "CONTINUE") == nil {
		return "LOGIN"
	}

	// 2. 检查是否在加载界面
	if util.SpecifiedCoordinateColor(hand, 427, 142) == "ffffff" && util.SpecifiedCoordinateColor(hand, 438, 153) == "ffffff" {
		return "LOADING"
	}

	// 3. 检查是否在聊天界面
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand, "MUTE") == nil {
		// 进一步检查聊天模式
		currentMode := getCurrentChatMode(hand)
		return "GAME_" + currentMode
	}

	// 4. 检查是否在游戏主界面（没有聊天框激活）
	// 这里可以通过检查游戏界面的其他特征来确认是否在游戏中
	// 比如检查血条、生命值等游戏UI元素
	if isInGameInterface(hand) {
		return "GAME_MAIN"
	}

	return "UNKNOWN"
}

// 检查是否在游戏主界面（没有聊天框）
func isInGameInterface(hand syscall.Handle) bool {
	// 检查游戏界面的特征，这里可以添加更多的检测逻辑
	// 比如检查生命值条、饥饿度等UI元素的存在
	// 暂时通过排除法，如果不在登录、加载、聊天界面，且游戏在运行，则认为在游戏主界面
	return true
}

// Start
// @author: [Fantasia](https://www.npc0.com)
// @function: 启动服务主逻辑
// @description: 机器人登录检测主逻辑 - 优化版本
func Start() {
	// init
	var ok bool
	var err error
	var hand syscall.Handle

	ErrorReboot()
	logDebug("检查游戏状态...")

	// 判断是否有scum游戏进程
	if ok, err = util.CheckIfAppRunning("SCUM"); err != nil || !ok {
		// 启动游戏
		cmd := exec.Command("cmd", "/C", "start", "", "steam://rungameid/513710")
		logInfo("游戏未启动，正在启动游戏...")
		_ = cmd.Start()
		errorNumber++
		// 延时30秒等待游戏启动
		time.Sleep(30 * time.Second)
		return
	}

	// 查找窗口句柄
	if hand = util.FindWindow("UnrealWindow", "SCUM  "); hand == 0 {
		cmd := exec.Command("cmd", "/C", "start", "", "steam://rungameid/513710")
		logError("游戏窗口未找到，重新启动游戏...")
		_ = cmd.Start()
		// 延时120秒等待游戏完全加载
		time.Sleep(120 * time.Second)
		errorNumber++
		return
	}

	logDebug("找到游戏窗口，开始状态检测...")

	// 游戏成功启动后替换配置文件（只执行一次）
	if !configReplaced {
		logInfo("检测到游戏成功启动，正在替换SCUM配置文件...")
		if err := util.ReplaceSCUMConfig(); err != nil {
			logError("替换SCUM配置文件失败: %v", err)
		} else {
			logInfo("SCUM配置文件替换完成")
		}
		configReplaced = true
	}

	// 只在必要时设置游戏窗口大小和位置
	setWindowPositionOnce(hand)

	// 设置游戏窗口置顶
	util.SetForegroundWindow(hand)
	time.Sleep(200 * time.Millisecond)

	// 获取当前游戏状态
	currentState := checkGameState(hand)
	logDebug("当前游戏状态: %s", currentState)

	// 检查是否有待处理的指令
	hasPendingCommands = checkPendingCommands()

	// 根据状态进行相应处理
	switch {
	case currentState == "LOGIN":
		logInfo("检测到登录界面，验证机器人状态...")
		util.SendKeyToWindow(hand, 0x0D)
		time.Sleep(100 * time.Millisecond)
		util.SendKeyToWindow(hand, 0x0D)

		if util.SpecifiedCoordinateColor(hand, 97, 142) != "ffffff" {
			// 没有机器人,切换机器人模式
			logInfo("未检测到机器人模式，正在切换...")
			if err = robotgo.KeyTap("d", "ctrl"); err != nil {
				logError("切换机器人模式失败: %v", err)
				errorNumber++
				return
			}
			// 延时等待切换完成
			time.Sleep(1 * time.Second)
			errorNumber++
		}

		// 点击登录
		logInfo("开始登录...")
		if err = util.ClickTextCenter(hand, "CONTINUE"); err != nil {
			logError("点击CONTINUE失败: %v", err)
			errorNumber++
		}
		logInfo("点击登录成功...")
		time.Sleep(1 * time.Second)
		return

	case currentState == "LOADING":
		// 在加载界面，等待
		logDebug("检测到加载界面，等待加载完成...")
		time.Sleep(1 * time.Second)
		errorNumber2++
		return

	case currentState == "GAME_MAIN":
		// 在游戏主界面，检查是否有待处理的指令
		if hasPendingCommands {
			logInfo("检测到游戏主界面，有待处理指令，激活聊天监控...")
			// 重置错误计数器
			errorNumber2 = 0
			errorNumber = 0
			// 激活聊天并开始监控
			ChatMonitorWithActivation(hand)
		} else {
			// 没有待处理指令，保持在游戏主界面，定期检查
			logDebug("游戏主界面，无待处理指令，定期检查...")
			// 执行定时任务（每分钟的三个固定指令）
			executePeriodicCommands(hand)
			time.Sleep(5 * time.Second) // 短暂等待后再次检查
		}
		return

	case currentState == "GAME_GLOBAL":
		// 已经在GLOBAL模式，可以直接启动监控
		logInfo("检测到GLOBAL模式，启动聊天监控...")
		// 重置错误计数器
		errorNumber2 = 0
		errorNumber = 0
		ChatMonitor(hand)
		return

	case currentState == "GAME_LOCAL":
		// 在LOCAL模式，需要切换到GLOBAL
		logInfo("检测到LOCAL模式，切换聊天模式...")
		_ = robotgo.KeyTap("tab")
		time.Sleep(300 * time.Millisecond)

		// 验证是否切换成功
		if getCurrentChatMode(hand) == "GLOBAL" {
			logInfo("成功切换到GLOBAL模式，启动聊天监控...")
			errorNumber2 = 0
			errorNumber = 0
			ChatMonitor(hand)
			return
		}
		return

	case currentState == "GAME_ADMIN":
		// 在ADMIN模式，切换到GLOBAL模式
		logInfo("检测到ADMIN模式，切换到GLOBAL模式...")
		_ = robotgo.KeyTap("tab")
		time.Sleep(150 * time.Millisecond)
		_ = robotgo.KeyTap("tab")
		time.Sleep(300 * time.Millisecond)

		// 验证是否切换成功
		if getCurrentChatMode(hand) == "GLOBAL" {
			logInfo("成功切换到GLOBAL模式，启动聊天监控...")
			errorNumber2 = 0
			errorNumber = 0
			ChatMonitor(hand)
			return
		}
		return

	case currentState == "GAME_UNKNOWN":
		// 在游戏界面但聊天模式未知，尝试激活聊天
		logInfo("检测到游戏界面，聊天模式未知，尝试激活聊天功能...")

		// 检查状态稳定性，避免频繁按T
		if lastChatState == currentState {
			chatStateStableCount++
		} else {
			chatStateStableCount = 0
			lastChatState = currentState
		}

		// 只有在状态稳定且计数较低时才按T
		if chatStateStableCount < 3 {
			// 先按ESC确保退出任何菜单
			_ = robotgo.KeyTap("escape")
			time.Sleep(200 * time.Millisecond)

			// 按T激活聊天
			_ = robotgo.KeyTap("t")
			time.Sleep(500 * time.Millisecond)
		} else {
			logError("聊天状态持续未知，可能需要手动干预")
			errorNumber2++
		}
		return

	default:
		// 未知状态
		logError("检测到未知游戏状态: %s", currentState)

		// 检查状态稳定性
		if lastChatState == currentState {
			chatStateStableCount++
		} else {
			chatStateStableCount = 0
			lastChatState = currentState
		}

		// 只有在连续出现问题时才重新设置窗口位置
		if chatStateStableCount > 5 {
			logInfo("状态持续异常，重新设置窗口位置...")
			lastWindowX, lastWindowY = -1, -1 // 重置缓存，强制重新设置
			setWindowPositionOnce(hand)
			chatStateStableCount = 0
		}

		errorNumber2++
		return
	}
}
