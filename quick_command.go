package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// QuickCommandExecutor 快速命令执行器
type QuickCommandExecutor struct {
	hwnd        syscall.Handle
	isReady     bool
	lastCommand time.Time
}

// NewQuickCommandExecutor 创建快速命令执行器
func NewQuickCommandExecutor(hwnd syscall.Handle) *QuickCommandExecutor {
	return &QuickCommandExecutor{
		hwnd:        hwnd,
		isReady:     false,
		lastCommand: time.Now(),
	}
}

// 常用命令预设
var QuickCommands = map[string]string{
	"players":   "#ListPlayers true",
	"vehicles":  "#ListSpawnedVehicles true",
	"squads":    "#dumpallsquadsinfolist",
	"time12":    "#SetTime 12",
	"time0":     "#SetTime 0",
	"sunny":     "#SetWeather 0",
	"storm":     "#SetWeather 1",
	"godmode":   "#SetGodMode true",
	"nogodmode": "#SetGodMode false",
	"save":      "#Save",
}

// 批量命令预设
var BatchCommands = map[string][]string{
	"status": {
		"#ListPlayers true",
		"#ListSpawnedVehicles true",
		"#dumpallsquadsinfolist",
	},
	"reset": {
		"#SetTime 12",
		"#SetWeather 0",
		"#Save",
	},
	"cleanup": {
		"#DestroyAllVehicles please",
		"#Save",
	},
}

// FastSendCommand 快速发送命令（优化版本）
func (qce *QuickCommandExecutor) FastSendCommand(command string) (string, error) {
	// 检查命令间隔，避免过于频繁
	if time.Since(qce.lastCommand) < 200*time.Millisecond {
		time.Sleep(200 * time.Millisecond)
	}

	// 1. 快速激活游戏窗口
	user32 := syscall.NewLazyDLL("user32.dll")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")
	setForegroundWindow.Call(uintptr(qce.hwnd))

	// 2. 最小延迟激活聊天框
	time.Sleep(50 * time.Millisecond)
	keybd_event := user32.NewProc("keybd_event")

	// 按下Enter激活聊天
	keybd_event.Call(0x0D, 0, 0, 0) // 按下
	keybd_event.Call(0x0D, 0, 2, 0) // 释放
	time.Sleep(100 * time.Millisecond)

	// 3. 快速输入命令
	for _, char := range command {
		if char == ' ' {
			keybd_event.Call(0x20, 0, 0, 0) // 空格按下
			keybd_event.Call(0x20, 0, 2, 0) // 空格释放
		} else {
			// 转换字符为虚拟键码并发送
			vk := charToVK(char)
			if vk != 0 {
				keybd_event.Call(uintptr(vk), 0, 0, 0)
				keybd_event.Call(uintptr(vk), 0, 2, 0)
			}
		}
		time.Sleep(5 * time.Millisecond) // 最小字符间隔
	}

	// 4. 发送命令
	time.Sleep(50 * time.Millisecond)
	keybd_event.Call(0x0D, 0, 0, 0) // Enter按下
	keybd_event.Call(0x0D, 0, 2, 0) // Enter释放

	// 5. 等待结果（根据命令类型调整等待时间）
	waitTime := getCommandWaitTime(command)
	time.Sleep(waitTime)

	// 6. 读取结果（使用OCR或剪贴板）
	result := readCommandResult()

	// 7. 关闭聊天框
	keybd_event.Call(0x1B, 0, 0, 0) // ESC按下
	keybd_event.Call(0x1B, 0, 2, 0) // ESC释放

	qce.lastCommand = time.Now()
	return result, nil
}

// ExecuteQuickCommand 执行预设快速命令
func (qce *QuickCommandExecutor) ExecuteQuickCommand(alias string) (string, error) {
	command, exists := QuickCommands[alias]
	if !exists {
		return "", fmt.Errorf("未知的快速命令: %s", alias)
	}

	fmt.Printf("执行快速命令: %s -> %s\n", alias, command)
	return qce.FastSendCommand(command)
}

// ExecuteBatchCommands 执行批量命令
func (qce *QuickCommandExecutor) ExecuteBatchCommands(batchName string) map[string]string {
	commands, exists := BatchCommands[batchName]
	if !exists {
		return map[string]string{"error": "未知的批量命令: " + batchName}
	}

	results := make(map[string]string)
	fmt.Printf("执行批量命令组: %s (%d个命令)\n", batchName, len(commands))

	for i, cmd := range commands {
		fmt.Printf("[%d/%d] 执行: %s\n", i+1, len(commands), cmd)
		result, err := qce.FastSendCommand(cmd)
		if err != nil {
			results[cmd] = "ERROR: " + err.Error()
		} else {
			results[cmd] = result
		}

		// 批量命令间适当间隔
		if i < len(commands)-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	return results
}

// 字符转虚拟键码
func charToVK(char rune) uint8 {
	if char >= 'a' && char <= 'z' {
		return uint8(char - 'a' + 0x41)
	}
	if char >= 'A' && char <= 'Z' {
		return uint8(char)
	}
	if char >= '0' && char <= '9' {
		return uint8(char)
	}

	// 特殊字符映射
	switch char {
	case '#':
		return 0x33 // '3' key with shift
	case ' ':
		return 0x20
	case '.':
		return 0xBE
	case '-':
		return 0xBD
	}

	return 0
}

// 根据命令类型获取等待时间
func getCommandWaitTime(command string) time.Duration {
	switch {
	case strings.Contains(command, "ListPlayers"):
		return 500 * time.Millisecond
	case strings.Contains(command, "ListSpawnedVehicles"):
		return 300 * time.Millisecond
	case strings.Contains(command, "dumpallsquadsinfolist"):
		return 800 * time.Millisecond
	case strings.Contains(command, "SetTime"):
		return 100 * time.Millisecond
	case strings.Contains(command, "SetWeather"):
		return 100 * time.Millisecond
	case strings.Contains(command, "Save"):
		return 1000 * time.Millisecond
	default:
		return 400 * time.Millisecond
	}
}

// 读取命令结果
func readCommandResult() string {
	// 这里可以实现OCR读取或剪贴板读取
	// 暂时返回占位符
	return "命令执行完成"
}

// HTTPCommandServer HTTP命令服务器
func StartHTTPCommandServer(qce *QuickCommandExecutor) {
	http.HandleFunc("/quick", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Command string `json:"command"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "读取请求失败", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "解析JSON失败", http.StatusBadRequest)
			return
		}

		result, err := qce.ExecuteQuickCommand(req.Command)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := map[string]string{
			"result": result,
			"status": "success",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			BatchName string `json:"batch_name"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "读取请求失败", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "解析JSON失败", http.StatusBadRequest)
			return
		}

		results := qce.ExecuteBatchCommands(req.BatchName)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	fmt.Println("HTTP命令服务器启动在端口 :8080")
	fmt.Println("快速命令示例:")
	fmt.Println("curl -X POST http://localhost:8080/quick -d '{\"command\":\"players\"}'")
	fmt.Println("curl -X POST http://localhost:8080/batch -d '{\"batch_name\":\"status\"}'")

	http.ListenAndServe(":8080", nil)
}

// 使用示例
func QuickCommandExample() {
	// 查找SCUM窗口
	hwnd := findSCUMWindow()
	if hwnd == 0 {
		fmt.Println("未找到SCUM游戏窗口")
		return
	}

	// 创建快速命令执行器
	qce := NewQuickCommandExecutor(hwnd)

	// 启动HTTP服务器（在后台goroutine中）
	go StartHTTPCommandServer(qce)

	// 示例：执行快速命令
	fmt.Println("=== 快速命令示例 ===")

	// 获取玩家列表
	result, _ := qce.ExecuteQuickCommand("players")
	fmt.Printf("玩家列表: %s\n", result)

	// 设置时间为中午
	qce.ExecuteQuickCommand("time12")

	// 设置晴天
	qce.ExecuteQuickCommand("sunny")

	// 执行状态检查批量命令
	fmt.Println("\n=== 批量命令示例 ===")
	results := qce.ExecuteBatchCommands("status")
	for cmd, result := range results {
		fmt.Printf("[%s] -> %s\n", cmd, result)
	}
}

func findSCUMWindow() syscall.Handle {
	// 查找SCUM窗口的实现
	// 这里使用占位符，实际实现需要调用Windows API
	return syscall.Handle(12345) // 占位符
}
