package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// 更新相关常量
const (
	UpdateCheckURL      = "https://api.github.com/repos/your-org/scum_client/releases/latest" // GitHub API获取最新版本
	UpdateDownloadURL   = "https://github.com/your-org/scum_client/releases/download"         // GitHub下载基础URL
	UpdateTempDir       = "temp_update"                                                       // 临时更新目录
	UpdateBackupSuffix  = ".backup"                                                           // 备份文件后缀
	UpdateRetryCount    = 3                                                                   // 更新重试次数
	UpdateTimeoutSecond = 300                                                                 // 更新超时时间（秒）

	// 更新状态
	UpdateStatusChecking    = "checking"    // 检查更新中
	UpdateStatusDownloading = "downloading" // 下载中
	UpdateStatusInstalling  = "installing"  // 安装中
	UpdateStatusCompleted   = "completed"   // 更新完成
	UpdateStatusFailed      = "failed"      // 更新失败
	UpdateStatusNoUpdate    = "no_update"   // 无需更新
)

// SelfUpdater scum_client自我更新器
type SelfUpdater struct {
	currentVersion string
	updateURL      string
	tempDir        string
}

// CheckForUpdates 检查更新
func (u *SelfUpdater) CheckForUpdates() (version string, downloadURL string, err error) {
	fmt.Printf("Checking for updates from: %s\n", u.updateURL)

	// TODO: 实现实际的更新检查逻辑
	// 1. 获取当前版本
	// 2. 从GitHub API获取最新版本
	// 3. 比较版本号
	// 4. 如果有新版本，返回版本号和下载URL

	return "", "", nil // 暂时返回无更新
}

// DownloadUpdate 下载更新
func (u *SelfUpdater) DownloadUpdate(downloadURL string) (string, error) {
	fmt.Printf("Downloading update from: %s\n", downloadURL)

	// 创建临时目录
	tempDir := filepath.Join(".", u.tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// 下载文件
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 保存到临时文件
	updateFile := filepath.Join(tempDir, "scum_client_update.exe")
	out, err := os.Create(updateFile)
	if err != nil {
		return "", fmt.Errorf("failed to create update file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save update file: %w", err)
	}

	fmt.Printf("Update downloaded successfully: %s\n", updateFile)
	return updateFile, nil
}

// InstallUpdate 安装更新
func (u *SelfUpdater) InstallUpdate(updateFile string) error {
	fmt.Printf("Installing update from: %s\n", updateFile)

	// 获取当前执行文件路径
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// 备份当前文件
	backupFile := currentExe + UpdateBackupSuffix
	if err := u.copyFile(currentExe, backupFile); err != nil {
		return fmt.Errorf("failed to backup current executable: %w", err)
	}

	fmt.Printf("Current executable backed up to: %s\n", backupFile)

	// 替换执行文件
	if err := u.copyFile(updateFile, currentExe); err != nil {
		// 如果替换失败，尝试恢复备份
		fmt.Printf("Failed to replace executable, attempting to restore backup...\n")
		if restoreErr := u.copyFile(backupFile, currentExe); restoreErr != nil {
			fmt.Printf("Failed to restore backup: %v\n", restoreErr)
		}
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	fmt.Printf("Update installed successfully\n")

	// 清理临时文件
	os.Remove(updateFile)
	os.Remove(filepath.Dir(updateFile))

	return nil
}

// copyFile 复制文件
func (u *SelfUpdater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// 复制文件权限
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// RestartSelf 重启程序
func (u *SelfUpdater) RestartSelf() error {
	fmt.Printf("Restarting scum_client...\n")

	// 获取当前执行文件路径和参数
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// 启动新进程
	cmd := exec.Command(currentExe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart client: %w", err)
	}

	fmt.Printf("New client process started, exiting current process...\n")
	os.Exit(0)
	return nil
}

// PerformSelfUpdate 执行完整的自我更新流程
func (u *SelfUpdater) PerformSelfUpdate() error {
	fmt.Printf("Starting self-update process...\n")

	// 1. 检查更新
	latestVersion, downloadURL, err := u.CheckForUpdates()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if latestVersion == "" {
		fmt.Printf("No updates available\n")
		return nil
	}

	fmt.Printf("New version available: %s\n", latestVersion)

	// 2. 下载更新
	updateFile, err := u.DownloadUpdate(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// 3. 安装更新
	if err := u.InstallUpdate(updateFile); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	// 4. 重启程序
	return u.RestartSelf()
}
