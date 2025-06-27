package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var ocrProcess *exec.Cmd
var ocrServiceRunning = false

// StartOCRService 启动 OCR 服务
func StartOCRService() error {
	// 检查服务是否已经运行
	if IsOCRServiceRunning() {
		fmt.Println("OCR 服务已经在运行")
		return nil
	}

	// 检查虚拟环境是否存在
	if !checkOCREnvironment() {
		fmt.Println("OCR 环境未设置，请先运行 ocr_setup.bat")
		return fmt.Errorf("OCR 环境未设置")
	}

	fmt.Println("正在启动 OCR 服务...")

	// 构建Python可执行文件路径
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join("ocr_env", "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join("ocr_env", "bin", "python")
	}

	// 检查Python可执行文件是否存在
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		return fmt.Errorf("Python 虚拟环境未找到: %s", pythonExe)
	}

	// 启动 OCR 服务
	ocrProcess = exec.Command(pythonExe, "ocr_server.py")

	// 在Windows下隐藏命令行窗口
	if runtime.GOOS == "windows" {
		ocrProcess.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	// 重定向输出到日志文件
	logFile, err := os.OpenFile("logs/ocr_service.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("无法创建OCR服务日志文件: %v\n", err)
	} else {
		ocrProcess.Stdout = logFile
		ocrProcess.Stderr = logFile
	}

	err = ocrProcess.Start()
	if err != nil {
		return fmt.Errorf("启动 OCR 服务失败: %v", err)
	}

	// 等待服务启动
	fmt.Println("等待 OCR 服务初始化...")
	maxWait := 30 // 最多等待30秒
	for i := 0; i < maxWait; i++ {
		time.Sleep(1 * time.Second)
		if IsOCRServiceRunning() {
			fmt.Println("OCR 服务启动成功")
			ocrServiceRunning = true
			return nil
		}
		fmt.Printf("等待中... (%d/%d)\n", i+1, maxWait)
	}

	// 超时后杀死进程
	if ocrProcess != nil && ocrProcess.Process != nil {
		ocrProcess.Process.Kill()
	}
	return fmt.Errorf("OCR 服务启动超时")
}

// IsOCRServiceRunning 检查 OCR 服务是否运行
func IsOCRServiceRunning() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:1224/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// StopOCRService 停止 OCR 服务
func StopOCRService() {
	if ocrProcess != nil && ocrProcess.Process != nil {
		fmt.Println("正在停止 OCR 服务...")

		// 在Windows下使用taskkill
		if runtime.GOOS == "windows" {
			cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", ocrProcess.Process.Pid), "/T", "/F")
			cmd.Run()
		} else {
			ocrProcess.Process.Kill()
		}

		ocrProcess.Wait()
		fmt.Println("OCR 服务已停止")
	}
	ocrServiceRunning = false
}

// checkOCREnvironment 检查 OCR 环境是否已设置
func checkOCREnvironment() bool {
	// 检查虚拟环境目录
	if _, err := os.Stat("ocr_env"); os.IsNotExist(err) {
		return false
	}

	// 检查 Python 可执行文件
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join("ocr_env", "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join("ocr_env", "bin", "python")
	}

	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		return false
	}

	// 检查 OCR 服务脚本
	if _, err := os.Stat("ocr_server.py"); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetupOCREnvironment 设置 OCR 环境
func SetupOCREnvironment() error {
	fmt.Println("开始设置 OCR 环境...")

	// 检查是否在 Windows 系统
	if runtime.GOOS != "windows" {
		return fmt.Errorf("目前只支持 Windows 系统")
	}

	// 检查 ocr_setup.bat 是否存在
	setupScript := "ocr_setup.bat"
	if _, err := os.Stat(setupScript); os.IsNotExist(err) {
		return fmt.Errorf("安装脚本不存在: %s", setupScript)
	}

	// 执行安装脚本
	cmd := exec.Command("cmd", "/C", setupScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("OCR 环境设置失败: %v", err)
	}

	fmt.Println("OCR 环境设置完成")
	return nil
}

// EnsureOCRService 确保 OCR 服务运行
func EnsureOCRService() error {
	// 如果服务已经运行，直接返回
	if IsOCRServiceRunning() {
		return nil
	}

	// 检查环境是否已设置
	if !checkOCREnvironment() {
		fmt.Println("检测到 OCR 环境未设置，正在自动设置...")
		if err := SetupOCREnvironment(); err != nil {
			return fmt.Errorf("自动设置 OCR 环境失败: %v", err)
		}
	}

	// 启动服务
	return StartOCRService()
}

// RestartOCRService 重启 OCR 服务
func RestartOCRService() error {
	fmt.Println("正在重启 OCR 服务...")
	StopOCRService()
	time.Sleep(2 * time.Second)
	return StartOCRService()
}

// GetOCRServiceStatus 获取 OCR 服务状态
func GetOCRServiceStatus() map[string]interface{} {
	status := map[string]interface{}{
		"environment_ready": checkOCREnvironment(),
		"service_running":   IsOCRServiceRunning(),
		"process_alive":     ocrProcess != nil && ocrProcess.Process != nil,
	}

	// 尝试获取服务详细信息
	if IsOCRServiceRunning() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://127.0.0.1:1224/")
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				status["service_info"] = strings.TrimSpace(string(body))
			}
		}
	}

	return status
}
