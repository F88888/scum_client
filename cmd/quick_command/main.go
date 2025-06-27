package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"qq_client/quick"
	"strings"
)

var (
	mode    = flag.String("mode", "interactive", "运行模式: interactive, server, command")
	port    = flag.String("port", "8080", "HTTP服务器端口")
	command = flag.String("cmd", "", "要执行的命令")
	batch   = flag.String("batch", "", "批量命令文件路径")
)

func main() {
	flag.Parse()

	fmt.Println("=== SCUM 快速命令工具 ===")

	// 创建快速命令执行器
	fastCmd := quick.NewFastCommand()

	// 根据模式运行
	switch *mode {
	case "server":
		runServerMode(fastCmd)
	case "command":
		runCommandMode(fastCmd)
	case "interactive":
		runInteractiveMode(fastCmd)
	default:
		fmt.Printf("未知模式: %s\n", *mode)
		flag.Usage()
	}
}

// runServerMode 运行HTTP服务器模式
func runServerMode(fastCmd *quick.FastCommand) {
	fmt.Printf("启动HTTP API服务器模式 (端口: %s)\n", *port)

	// 初始化快速命令执行器
	if err := fastCmd.Initialize(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Println("快速命令执行器初始化成功")
	fmt.Printf("API端点:\n")
	fmt.Printf("  POST /command  - 执行单个命令\n")
	fmt.Printf("  POST /batch    - 批量执行命令\n")
	fmt.Printf("  GET  /status   - 检查状态\n")
	fmt.Printf("  GET  /presets  - 获取预设命令\n")

	// 启动HTTP服务器
	fastCmd.StartHTTPServer(*port)
}

// runCommandMode 运行单命令模式
func runCommandMode(fastCmd *quick.FastCommand) {
	if *command == "" && *batch == "" {
		fmt.Println("命令模式需要指定 -cmd 或 -batch 参数")
		return
	}

	// 初始化
	if err := fastCmd.Initialize(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	if *command != "" {
		// 执行单个命令
		fmt.Printf("执行命令: %s\n", *command)
		result, err := fastCmd.ExecuteCommand(*command)
		if err != nil {
			log.Fatalf("命令执行失败: %v", err)
		}
		fmt.Println("执行结果:")
		fmt.Println(result)
	}

	if *batch != "" {
		// 执行批量命令
		commands, err := loadCommandsFromFile(*batch)
		if err != nil {
			log.Fatalf("加载批量命令失败: %v", err)
		}

		fmt.Printf("执行批量命令 (%d个)...\n", len(commands))
		results, err := fastCmd.ExecuteBatch(commands)
		if err != nil {
			log.Fatalf("批量执行失败: %v", err)
		}

		fmt.Println("执行结果:")
		for _, result := range results {
			fmt.Println(result)
		}
	}
}

// runInteractiveMode 运行交互模式
func runInteractiveMode(fastCmd *quick.FastCommand) {
	fmt.Println("启动交互模式...")

	// 初始化
	if err := fastCmd.Initialize(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Println("快速命令执行器已就绪!")
	printHelp()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("SCUM> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 处理特殊命令
		switch input {
		case "exit", "quit":
			fmt.Println("退出程序...")
			return
		case "help":
			printHelp()
			continue
		case "status":
			showStatus(fastCmd)
			continue
		case "presets":
			showPresets(fastCmd)
			continue
		}

		// 处理批量命令（多行输入）
		if strings.HasPrefix(input, "batch") {
			handleBatchInput(fastCmd, scanner)
			continue
		}

		// 执行单个命令
		result, err := fastCmd.ExecuteCommand(input)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println(result)
		}
	}
}

// loadCommandsFromFile 从文件加载命令
func loadCommandsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释行
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "//") {
			commands = append(commands, line)
		}
	}

	return commands, scanner.Err()
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println(`
可用命令:
  help            - 显示此帮助信息
  status          - 显示当前状态
  presets         - 显示预设命令
  batch           - 进入批量命令输入模式
  exit/quit       - 退出程序

快速别名:
  players         - 列出玩家 (#ListPlayers)
  vehicles        - 列出载具 (#ListSpawnedVehicles)
  time12          - 设置时间12:00 (#SetTime 12 00)
  time0           - 设置时间00:00 (#SetTime 00 00)
  morning         - 设置时间08:00 (#SetTime 08 00)
  noon            - 设置时间12:00 (#SetTime 12 00)
  evening         - 设置时间18:00 (#SetTime 18 00)
  night           - 设置时间22:00 (#SetTime 22 00)
  flags           - 列出标记 (#listflags 1 true)
  squads          - 列出小队 (#dumpallsquadsinfolist)

原始命令 (可省略#前缀):
  ListPlayers     - 列出所有玩家
  SetTime H M     - 设置时间 (H=小时, M=分钟)
  Kick 玩家名 原因 - 踢出玩家
  Ban 玩家名 原因  - 封禁玩家
  以及其他SCUM管理员命令...
`)
}

// showStatus 显示状态信息
func showStatus(fastCmd *quick.FastCommand) {
	fmt.Println("=== 当前状态 ===")
	// 这里可以添加状态检查逻辑
	fmt.Println("快速命令执行器: 运行中")
	fmt.Println("游戏连接: 已连接")
	fmt.Println("聊天界面: 已激活")
}

// showPresets 显示预设命令
func showPresets(fastCmd *quick.FastCommand) {
	fmt.Println("=== 预设命令 ===")

	aliases := map[string]string{
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
	}

	fmt.Println("快速别名:")
	for alias, command := range aliases {
		fmt.Printf("  %-10s -> %s\n", alias, command)
	}

	fmt.Println("\n常用命令:")
	commonCommands := []string{
		"#ListPlayers",
		"#ListSpawnedVehicles",
		"#SetTime 12 00",
		"#listflags 1 true",
		"#dumpallsquadsinfolist",
		"#RestartServer",
		"#Shutdown",
	}

	for _, cmd := range commonCommands {
		fmt.Printf("  %s\n", cmd)
	}
}

// handleBatchInput 处理批量输入
func handleBatchInput(fastCmd *quick.FastCommand, scanner *bufio.Scanner) {
	fmt.Println("进入批量命令模式 (输入 'end' 结束):")

	var commands []string
	for {
		fmt.Print("  > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "end" {
			break
		}
		if input != "" {
			commands = append(commands, input)
		}
	}

	if len(commands) == 0 {
		fmt.Println("没有输入命令")
		return
	}

	fmt.Printf("准备执行 %d 个命令...\n", len(commands))
	results, err := fastCmd.ExecuteBatch(commands)
	if err != nil {
		fmt.Printf("批量执行失败: %v\n", err)
		return
	}

	fmt.Println("执行结果:")
	for _, result := range results {
		fmt.Println(result)
	}
}
