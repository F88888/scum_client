package main

import (
	"fmt"
	"log"
	"qq_client/quick"
	"qq_client/util"
	"time"
)

// 演示增强输入功能的使用方法
func main() {
	fmt.Println("=== SCUM Client 增强输入功能演示 ===")

	// 创建快速命令执行器
	fastCmd := quick.NewFastCommand()

	// 初始化
	if err := fastCmd.Initialize(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Println("✓ 快速命令执行器初始化完成")

	// 演示1: 基本命令执行（使用增强模式）
	fmt.Println("\n--- 演示1: 基本命令执行 ---")
	result, err := fastCmd.ExecuteCommand("players")
	if err != nil {
		fmt.Printf("✗ 命令执行失败: %v\n", err)
	} else {
		fmt.Printf("✓ 命令执行成功: %s\n", result)
	}

	// 演示2: 批量命令执行
	fmt.Println("\n--- 演示2: 批量命令执行 ---")
	commands := []string{
		"#ListPlayers true",
		"#ListSpawnedVehicles true",
		"#SetTime 12 00",
		"sunny",
	}

	results, err := fastCmd.ExecuteBatch(commands)
	if err != nil {
		fmt.Printf("✗ 批量执行有错误: %v\n", err)
	}
	for _, result := range results {
		fmt.Printf("✓ %s\n", result)
	}

	// 演示3: 直接使用增强输入管理器
	fmt.Println("\n--- 演示3: 直接使用增强输入管理器 ---")
	demoEnhancedInputManager()

	// 演示4: 直接使用增强命令执行器
	fmt.Println("\n--- 演示4: 直接使用增强命令执行器 ---")
	demoEnhancedCommandExecutor()

	// 演示5: 不同输入方法性能对比
	fmt.Println("\n--- 演示5: 输入方法性能对比 ---")
	demoPerformanceComparison()

	fmt.Println("\n=== 演示完成 ===")
}

// 演示增强输入管理器的直接使用
func demoEnhancedInputManager() {
	// 查找SCUM窗口
	hwnd := util.FindWindow("UnrealWindow", "SCUM  ")
	if hwnd == 0 {
		fmt.Println("✗ 未找到SCUM游戏窗口")
		return
	}

	// 创建输入管理器
	inputManager := util.NewEnhancedInputManager(hwnd)

	fmt.Println("✓ 输入管理器创建成功")

	// 测试不同的聊天激活方式
	fmt.Println("  测试T键激活...")
	if err := inputManager.ActivateChat(util.CHAT_ACTIVATE_T_KEY); err != nil {
		fmt.Printf("  ✗ T键激活失败: %v\n", err)
	} else {
		fmt.Println("  ✓ T键激活成功")
	}

	time.Sleep(500 * time.Millisecond)

	// 测试不同的文本输入方式
	testCommands := []string{
		"#ListPlayers",
		"这是一个测试命令",
		"#SetTime 12 00",
	}

	inputMethods := []util.InputMethod{
		util.INPUT_CLIPBOARD_PASTE,
		util.INPUT_SIMULATE_KEY,
		util.INPUT_WINDOW_MSG,
	}

	methodNames := []string{
		"剪贴板粘贴",
		"模拟按键",
		"窗口消息",
	}

	for i, method := range inputMethods {
		fmt.Printf("  测试%s方式...\n", methodNames[i])

		for _, cmd := range testCommands {
			start := time.Now()
			if err := inputManager.SendText(cmd, method); err != nil {
				fmt.Printf("    ✗ 发送'%s'失败: %v\n", cmd, err)
			} else {
				duration := time.Since(start)
				fmt.Printf("    ✓ 发送'%s'成功 (耗时: %v)\n", cmd, duration)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	// 发送ESC关闭聊天框
	inputManager.SendEscape()
}

// 演示增强命令执行器的直接使用
func demoEnhancedCommandExecutor() {
	// 查找SCUM窗口
	hwnd := util.FindWindow("UnrealWindow", "SCUM  ")
	if hwnd == 0 {
		fmt.Println("✗ 未找到SCUM游戏窗口")
		return
	}

	// 创建增强命令执行器
	executor := util.NewEnhancedCommandExecutor(hwnd)

	fmt.Println("✓ 增强命令执行器创建成功")

	// 测试单个命令执行
	testCommands := []string{
		"players",
		"vehicles",
		"time12",
		"#Save",
	}

	for _, cmd := range testCommands {
		fmt.Printf("  执行命令: %s\n", cmd)
		execution, err := executor.ExecuteCommand(cmd)
		if err != nil {
			fmt.Printf("    ✗ 执行失败: %v\n", err)
		} else {
			fmt.Printf("    ✓ 执行成功 - 耗时: %v, 方法: %d, 成功: %v\n",
				execution.ExecutionTime, execution.InputMethod, execution.Success)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 测试批量命令执行
	fmt.Println("  执行批量命令...")
	batchCommands := []string{
		"#ListPlayers true",
		"#ListSpawnedVehicles true",
		"#SetTime 18 00",
	}

	executions, err := executor.ExecuteBatch(batchCommands)
	if err != nil {
		fmt.Printf("    ✗ 批量执行失败: %v\n", err)
	} else {
		fmt.Printf("    ✓ 批量执行完成，共%d个命令\n", len(executions))
		for i, execution := range executions {
			fmt.Printf("      命令%d: %v (耗时: %v)\n",
				i+1, execution.Success, execution.ExecutionTime)
		}
	}

	// 获取执行统计
	stats := executor.GetExecutionStats()
	fmt.Printf("  执行统计: %+v\n", stats)

	// 关闭聊天框
	executor.CloseChatIfOpen()
}

// 演示不同输入方法的性能对比
func demoPerformanceComparison() {
	// 查找SCUM窗口
	hwnd := util.FindWindow("UnrealWindow", "SCUM  ")
	if hwnd == 0 {
		fmt.Println("✗ 未找到SCUM游戏窗口")
		return
	}

	inputManager := util.NewEnhancedInputManager(hwnd)

	// 激活聊天框
	if err := inputManager.ActivateChat(util.CHAT_ACTIVATE_T_KEY); err != nil {
		fmt.Printf("✗ 无法激活聊天框: %v\n", err)
		return
	}

	testText := "#ListPlayers true"
	methods := []util.InputMethod{
		util.INPUT_SIMULATE_KEY,
		util.INPUT_CLIPBOARD_PASTE,
		util.INPUT_WINDOW_MSG,
	}

	methodNames := []string{
		"模拟按键",
		"剪贴板粘贴",
		"窗口消息",
	}

	fmt.Printf("  测试文本: %s\n", testText)
	fmt.Println("  性能对比结果:")

	for i, method := range methods {
		var totalTime time.Duration
		var successCount int
		testCount := 3

		for j := 0; j < testCount; j++ {
			start := time.Now()
			err := inputManager.SendText(testText, method)
			duration := time.Since(start)

			if err == nil {
				successCount++
				totalTime += duration
			}

			time.Sleep(200 * time.Millisecond) // 间隔时间
		}

		if successCount > 0 {
			avgTime := totalTime / time.Duration(successCount)
			fmt.Printf("    %s: 平均耗时 %v, 成功率 %d/%d\n",
				methodNames[i], avgTime, successCount, testCount)
		} else {
			fmt.Printf("    %s: 全部失败\n", methodNames[i])
		}
	}

	// 获取方法统计
	stats := inputManager.GetMethodStats()
	fmt.Println("  详细统计:")
	for method, stat := range stats {
		fmt.Printf("    方法%d: 成功%d次, 失败%d次, 可靠性%.2f, 平均时间%v\n",
			method, stat.SuccessCount, stat.FailureCount,
			stat.Reliability, stat.AverageTime)
	}

	// 关闭聊天框
	inputManager.SendEscape()
}
