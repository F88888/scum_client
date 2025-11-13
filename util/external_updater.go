package util

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

// ExternalUpdaterConfig 外部更新器配置
type ExternalUpdaterConfig struct {
	CurrentExePath string   // 当前程序路径
	UpdateURL      string   // 更新下载URL
	Args           []string // 程序启动参数
}

// CreateExternalUpdater 创建外部更新器脚本
func CreateExternalUpdater(config ExternalUpdaterConfig) error {
	var scriptContent string
	var scriptName string

	if runtime.GOOS == "windows" {
		scriptName = "scum_client_updater.bat"
		scriptContent = fmt.Sprintf(`@echo off
echo Starting SCUM Client updater...

:: 等待主程序完全退出
timeout /t 3 /nobreak >nul

:: 下载新版本
echo Downloading update from %s...
powershell -Command "Invoke-WebRequest -Uri '%s' -OutFile 'scum_client_new.exe'"

if not exist "scum_client_new.exe" (
    echo Download failed!
    pause
    exit /b 1
)

:: 备份当前版本
if exist "%s" (
    echo Backing up current version...
    copy "%s" "%s.backup" >nul
    if errorlevel 1 (
        echo Backup failed!
        del "scum_client_new.exe" >nul
        pause
        exit /b 1
    )
)

:: 替换程序文件
echo Installing update...
copy "scum_client_new.exe" "%s" >nul
if errorlevel 1 (
    echo Installation failed! Restoring backup...
    copy "%s.backup" "%s" >nul
    del "scum_client_new.exe" >nul
    pause
    exit /b 1
)

:: 清理临时文件
del "scum_client_new.exe" >nul
del "%s.backup" >nul

:: 重启程序
echo Restarting SCUM Client...
start "" "%s" %s

:: 删除更新器脚本自己
del "%%%%~f0"
`, config.UpdateURL, config.UpdateURL, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, formatArgsForBatch(config.Args))
	} else {
		scriptName = "scum_client_updater.sh"
		scriptContent = fmt.Sprintf(`#!/bin/bash
echo "Starting SCUM Client updater..."

# 等待主程序完全退出
sleep 3

# 下载新版本
echo "Downloading update from %s..."
if ! curl -L -o "scum_client_new" "%s"; then
    echo "Download failed!"
    exit 1
fi

# 备份当前版本
if [ -f "%s" ]; then
    echo "Backing up current version..."
    if ! cp "%s" "%s.backup"; then
        echo "Backup failed!"
        rm -f "scum_client_new"
        exit 1
    fi
fi

# 替换程序文件
echo "Installing update..."
if ! cp "scum_client_new" "%s"; then
    echo "Installation failed! Restoring backup..."
    cp "%s.backup" "%s" 2>/dev/null
    rm -f "scum_client_new"
    exit 1
fi

# 设置执行权限
chmod +x "%s"

# 清理临时文件
rm -f "scum_client_new"
rm -f "%s.backup"

# 重启程序
echo "Restarting SCUM Client..."
nohup "%s" %s > /dev/null 2>&1 &

# 删除更新器脚本自己
rm -f "$0"
`, config.UpdateURL, config.UpdateURL, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, config.CurrentExePath, formatArgsForShell(config.Args))
	}

	// 写入脚本文件 - 确保使用 Windows 风格的换行符
	windowsContent := strings.ReplaceAll(scriptContent, "\n", "\r\n")
	if err := os.WriteFile(scriptName, []byte(windowsContent), 0644); err != nil {
		return fmt.Errorf("failed to create updater script: %w", err)
	}

	return nil
}

// ExecuteExternalUpdate 执行外部更新
func ExecuteExternalUpdate(config ExternalUpdaterConfig) error {
	// 1. 创建更新器脚本
	if err := CreateExternalUpdater(config); err != nil {
		return fmt.Errorf("failed to create updater script: %w", err)
	}

	// 2. 启动更新器脚本
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "scum_client_updater.bat")
	} else {
		cmd = exec.Command("bash", "scum_client_updater.sh")
	}

	// 分离进程，让更新器独立运行
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false, // 显示更新器窗口，让用户能看到更新进度
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	}
	// 注意：在 macOS/Linux 上不需要设置 SysProcAttr
	// 因为脚本中已经使用了 nohup 命令，它会自动分离进程

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start updater: %w", err)
	}

	return nil
}

// formatArgsForBatch 为Windows批处理格式化参数
func formatArgsForBatch(args []string) string {
	result := ""
	for _, arg := range args {
		if result != "" {
			result += " "
		}
		// 如果参数包含空格，需要加引号
		if containsSpaceChar(arg) {
			result += `"` + arg + `"`
		} else {
			result += arg
		}
	}
	return result
}

// formatArgsForShell 为Shell脚本格式化参数
func formatArgsForShell(args []string) string {
	result := ""
	for _, arg := range args {
		if result != "" {
			result += " "
		}
		// 转义特殊字符
		if containsSpaceChar(arg) || containsSpecialChars(arg) {
			result += `"` + arg + `"`
		} else {
			result += arg
		}
	}
	return result
}

// containsSpaceChar 检查字符串是否包含空格
func containsSpaceChar(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}

// containsSpecialChars 检查字符串是否包含特殊字符
func containsSpecialChars(s string) bool {
	specialChars := `$&|><;"'`
	for _, r := range s {
		for _, sc := range specialChars {
			if r == sc {
				return true
			}
		}
	}
	return false
}
