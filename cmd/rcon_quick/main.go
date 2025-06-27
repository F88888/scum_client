package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"qq_client/rcon"
	"strings"
	"time"
)

var (
	address     = flag.String("addr", "127.0.0.1:7777", "SCUM服务器RCON地址")
	password    = flag.String("pass", "", "RCON密码")
	command     = flag.String("cmd", "", "要执行的单个命令")
	batch       = flag.String("batch", "", "批量命令文件路径")
	interactive = flag.Bool("i", false, "交互模式")
)

func main() {
	flag.Parse()

	fmt.Println("=== SCUM RCON 快速命令工具 ===")

	if *password == "" {
		fmt.Print("请输入RCON密码: ")
		reader := bufio.NewReader(os.Stdin)
		pwd, _ := reader.ReadString('\n')
		*password = strings.TrimSpace(pwd)
	}

	// 连接RCON服务器
	fmt.Printf("正在连接到 %s...\n", *address)
	client, err := rcon.NewRCONClient(*address, *password)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	fmt.Println("连接成功！")

	quickCmd := rcon.NewQuickCommands(client)

	// 根据参数选择执行模式
	switch {
	case *command != "":
		// 单个命令模式
		executeSingleCommand(client, *command)
	case *batch != "":
		// 批量命令模式
		executeBatchFromFile(quickCmd, *batch)
	case *interactive:
		// 交互模式
		quickCmd.InteractiveMode()
	default:
		// 默认进入交互模式
		fmt.Println("未指定命令，进入交互模式...")
		quickCmd.InteractiveMode()
	}
}

// executeSingleCommand 执行单个命令
func executeSingleCommand(client *rcon.RCONClient, command string) {
	start := time.Now()
	result, err := client.ExecuteCommand(command)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
		return
	}

	fmt.Printf("命令: %s\n", command)
	fmt.Printf("耗时: %v\n", elapsed)
	fmt.Printf("结果:\n%s\n", result)
}

// executeBatchFromFile 从文件批量执行命令
func executeBatchFromFile(quickCmd *rcon.QuickCommands, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("打开批量命令文件失败: %v", err)
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			commands = append(commands, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("读取批量命令文件失败: %v", err)
	}

	fmt.Printf("准备执行 %d 个命令...\n", len(commands))
	start := time.Now()

	results, err := quickCmd.ExecuteBatch(commands)
	elapsed := time.Since(start)

	fmt.Printf("批量执行完成，总耗时: %v\n", elapsed)
	fmt.Printf("平均每个命令耗时: %v\n", elapsed/time.Duration(len(commands)))

	// 输出结果
	for i, result := range results {
		fmt.Printf("\n--- 命令 %d: %s ---\n", i+1, commands[i])
		fmt.Println(result)
	}

	if err != nil {
		fmt.Printf("批量执行过程中出现错误: %v\n", err)
	}
}
