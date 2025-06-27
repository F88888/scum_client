package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// SCUMRconClient SCUM RCON客户端
type SCUMRconClient struct {
	conn     net.Conn
	host     string
	port     string
	password string
}

// NewSCUMRconClient 创建新的RCON客户端
func NewSCUMRconClient(host, port, password string) *SCUMRconClient {
	return &SCUMRconClient{
		host:     host,
		port:     port,
		password: password,
	}
}

// Connect 连接到SCUM服务器RCON端口
func (r *SCUMRconClient) Connect() error {
	var err error
	r.conn, err = net.DialTimeout("tcp", r.host+":"+r.port, 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接RCON失败: %v", err)
	}

	fmt.Printf("成功连接到SCUM RCON服务器 %s:%s\n", r.host, r.port)
	return nil
}

// SendCommand 发送命令到SCUM服务器
func (r *SCUMRconClient) SendCommand(command string) (string, error) {
	if r.conn == nil {
		return "", fmt.Errorf("未连接到服务器")
	}

	// 发送命令
	_, err := r.conn.Write([]byte(command + "\n"))
	if err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 读取响应
	reader := bufio.NewReader(r.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	return strings.TrimSpace(response), nil
}

// ExecuteAdminCommand 执行管理员命令
func (r *SCUMRconClient) ExecuteAdminCommand(command string) (string, error) {
	// 添加管理员命令前缀
	if !strings.HasPrefix(command, "#") {
		command = "#" + command
	}

	fmt.Printf("执行命令: %s\n", command)
	result, err := r.SendCommand(command)
	if err != nil {
		return "", err
	}

	fmt.Printf("命令结果: %s\n", result)
	return result, nil
}

// Close 关闭连接
func (r *SCUMRconClient) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// BatchExecuteCommands 批量执行命令
func (r *SCUMRconClient) BatchExecuteCommands(commands []string) map[string]string {
	results := make(map[string]string)

	for _, cmd := range commands {
		result, err := r.ExecuteAdminCommand(cmd)
		if err != nil {
			results[cmd] = "ERROR: " + err.Error()
		} else {
			results[cmd] = result
		}
		// 命令间隔
		time.Sleep(100 * time.Millisecond)
	}

	return results
}

// QuickCommands 快速执行常用命令
func (r *SCUMRconClient) QuickCommands() {
	commands := []string{
		"ListPlayers",
		"ListSpawnedVehicles",
		"SetTime 12",
		"SetWeather 0",
	}

	fmt.Println("=== 快速执行常用命令 ===")
	results := r.BatchExecuteCommands(commands)

	for cmd, result := range results {
		fmt.Printf("[%s] -> %s\n", cmd, result)
	}
}

func main() {
	// 使用示例
	client := NewSCUMRconClient("localhost", "7777", "your_rcon_password")

	if err := client.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// 交互式命令行
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("SCUM RCON客户端已启动，输入管理员命令（输入 'quit' 退出）:")
	fmt.Println("示例命令: ListPlayers, SetTime 12, Teleport 100 100 100")

	for {
		fmt.Print("SCUM> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}

		if input == "quick" {
			client.QuickCommands()
			continue
		}

		if input == "" {
			continue
		}

		result, err := client.ExecuteAdminCommand(input)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("结果: %s\n", result)
		}
	}

	fmt.Println("RCON客户端已退出")
}
