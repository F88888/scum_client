package util

import (
	"archive/zip"
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

const (
	// 国内外下载源
	embeddedPythonURLCN   = "https://scum.npc0.com/python-3.12.10-embed-amd64.zip"
	getPipURLCN           = "https://scum.npc0.com/get-pip.py"
	embeddedPythonURLOff  = "https://www.python.org/ftp/python/3.12.10/python-3.12.10-embed-amd64.zip"
	getPipURLOff          = "https://bootstrap.pypa.io/get-pip.py"
	pypiTsinghuaIndex     = "https://pypi.tuna.tsinghua.edu.cn/simple"
	pypiTsinghuaHost      = "pypi.tuna.tsinghua.edu.cn"
	regionEnvKey          = "SCUM_REGION" // CN / INTL
	regionChinaFlagEnvKey = "SCUM_CN"     // 1/true/yes 表示中国区

	embedDir = "py_embed"
)

func ensureDir(dir string) error {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// fastReachable 尝试快速访问指定 URL，判断可达
func fastReachable(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

// shouldUseChinaMirror 根据环境变量/时区/网络探测选择是否使用国内镜像
func shouldUseChinaMirror() bool {
	// 环境变量强制
	if strings.EqualFold(os.Getenv(regionEnvKey), "CN") {
		return true
	}
	if strings.EqualFold(os.Getenv(regionEnvKey), "INTL") {
		return false
	}
	s := strings.ToLower(os.Getenv(regionChinaFlagEnvKey))
	if s == "1" || s == "true" || s == "yes" {
		return true
	}

	// 时区为 +8 作为弱指示
	_, offset := time.Now().Zone()
	if offset == 8*3600 {
		return true
	}

	// 网络探测：清华源是否可达
	if fastReachable(pypiTsinghuaIndex, 1500*time.Millisecond) {
		return true
	}
	return false
}

// selectDownloadSources 选择下载源与镜像策略
func selectDownloadSources() (embeddedURL, getPipURL string, useChina bool) {
	useChina = shouldUseChinaMirror()
	if useChina {
		return embeddedPythonURLCN, getPipURLCN, true
	}
	return embeddedPythonURLOff, getPipURLOff, false
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %s -> http %d", url, resp.StatusCode)
	}
	if err := ensureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := ensureDir(destDir); err != nil {
		return err
	}
	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法压缩条目路径: %s", fpath)
		}
		if f.FileInfo().IsDir() {
			if err := ensureDir(fpath); err != nil {
				return err
			}
			continue
		}
		if err := ensureDir(filepath.Dir(fpath)); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}

func enableImportSite(pthPath string) error {
	data, err := os.ReadFile(pthPath)
	if err != nil {
		return err
	}
	content := string(data)
	changed := false

	// 确保当前目录与标准库、site-packages 路径
	if !strings.Contains(content, "\n.\n") && !strings.HasSuffix(content, "\n.\n") && !strings.HasSuffix(content, ".\n") {
		content += "\n.\n"
		changed = true
	}
	if !strings.Contains(content, "Lib\n") {
		content += "Lib\n"
		changed = true
	}
	if !strings.Contains(content, "Lib\\site-packages\n") {
		content += "Lib\\site-packages\n"
		changed = true
	}

	// 启用 import site
	if strings.Contains(content, "#import site") {
		content = strings.ReplaceAll(content, "#import site", "import site")
		changed = true
	} else if !strings.Contains(content, "import site") {
		content += "import site\n"
		changed = true
	}

	if changed {
		return os.WriteFile(pthPath, []byte(content), 0644)
	}
	return nil
}

func ensureEmbeddedPython() (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("仅 Windows 支持内置 Python")
	}

	// 选择下载源
	embeddedURL, getPipURL, useChina := selectDownloadSources()

	pythonExe := filepath.Join(embedDir, "python.exe")
	if _, err := os.Stat(pythonExe); err == nil {
		abs, _ := filepath.Abs(pythonExe)
		return abs, nil
	}
	fmt.Println("未检测到内置 Python，开始自动下载并解压...")
	zipPath := filepath.Join(embedDir, "python-embed.zip")
	if err := downloadFile(embeddedURL, zipPath); err != nil {
		return "", fmt.Errorf("下载 Python 失败: %v", err)
	}
	if err := unzip(zipPath, embedDir); err != nil {
		return "", fmt.Errorf("解压 Python 失败: %v", err)
	}

	// 处理 _pth，开启 site 与路径
	var pth string
	if matches, _ := filepath.Glob(filepath.Join(embedDir, "python*._pth")); len(matches) > 0 {
		pth = matches[0]
	} else if matches, _ := filepath.Glob(filepath.Join(embedDir, "*._pth")); len(matches) > 0 {
		pth = matches[0]
	}
	if pth != "" {
		if err := enableImportSite(pth); err != nil {
			fmt.Printf("警告: 启用 import site 失败: %v\n", err)
		} else {
			fmt.Printf("已更新 _pth 文件: %s\n", filepath.Base(pth))
		}
	} else {
		fmt.Println("警告: 未找到 _pth 文件，将继续尝试安装 pip")
	}

	// 再次确认 python.exe 是否存在
	if _, err := os.Stat(pythonExe); err != nil {
		alts, _ := filepath.Glob(filepath.Join(embedDir, "*python*.exe"))
		if len(alts) == 0 {
			entries, _ := os.ReadDir(embedDir)
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			return "", fmt.Errorf("python.exe 未找到，目录内容: %v", strings.Join(names, ", "))
		}
		pythonExe = alts[0]
	}

	absPython, _ := filepath.Abs(pythonExe)
	fmt.Printf("使用内置 Python: %s\n", absPython)

	// 下载 get-pip.py
	getPipPath := filepath.Join(embedDir, "get-pip.py")
	if err := downloadFile(getPipURL, getPipPath); err != nil {
		return "", fmt.Errorf("下载 get-pip.py 失败: %v", err)
	}

	// 安装 pip（使用绝对路径）
	cmd := exec.Command(absPython, "get-pip.py", "--no-warn-script-location")
	cmd.Dir = embedDir
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("安装 pip 失败: %v", err)
	}

	// 中国区：配置 pip 全局镜像为清华源
	if useChina {
		cfg1 := exec.Command(absPython, "-m", "pip", "config", "set", "global.index-url", pypiTsinghuaIndex)
		cfg1.Dir = embedDir
		if runtime.GOOS == "windows" {
			cfg1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		cfg1.Stdout = os.Stdout
		cfg1.Stderr = os.Stderr
		_ = cfg1.Run()

		cfg2 := exec.Command(absPython, "-m", "pip", "config", "set", "global.trusted-host", pypiTsinghuaHost)
		cfg2.Dir = embedDir
		if runtime.GOOS == "windows" {
			cfg2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		cfg2.Stdout = os.Stdout
		cfg2.Stderr = os.Stderr
		_ = cfg2.Run()
	}

	// 安装依赖
	fmt.Println("正在安装 PaddlePaddle 及 PaddleOCR 依赖... (首次可能较慢)")
	pipArgs := []string{"-m", "pip", "install", "--no-warn-script-location"}
	if useChina {
		pipArgs = append(pipArgs, "-i", pypiTsinghuaIndex, "--trusted-host", pypiTsinghuaHost)
	}
	pipArgs = append(pipArgs, "paddlepaddle==3.0.0", "paddleocr", "flask", "requests", "pillow")

	pipCmd := exec.Command(absPython, pipArgs...)
	pipCmd.Dir = embedDir
	if runtime.GOOS == "windows" {
		pipCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	pipCmd.Stdout = os.Stdout
	pipCmd.Stderr = os.Stderr
	if err := pipCmd.Run(); err != nil {
		return "", fmt.Errorf("安装依赖失败: %v", err)
	}

	// 创建 Scripts 目录结构以保持一致性
	scriptsDir := filepath.Join(embedDir, "Scripts")
	if err := ensureDir(scriptsDir); err != nil {
		fmt.Printf("警告: 创建 Scripts 目录失败: %v\n", err)
	} else {
		// 在 Scripts 目录中创建 python.exe 的副本
		scriptsPythonExe := filepath.Join(scriptsDir, "python.exe")
		if _, err := os.Stat(scriptsPythonExe); os.IsNotExist(err) {
			// 复制 python.exe 到 Scripts 目录
			if err := copyFile(absPython, scriptsPythonExe); err != nil {
				fmt.Printf("警告: 创建 Scripts/python.exe 失败: %v\n", err)
			} else {
				fmt.Printf("已创建 Scripts/python.exe\n")
			}
		}

		// 复制必要的 DLL 文件到 Scripts 目录，解决 DLL 加载问题
		dllFiles := []string{"python312.dll", "python3.dll"}
		for _, dllFile := range dllFiles {
			srcDLL := filepath.Join(embedDir, dllFile)
			dstDLL := filepath.Join(scriptsDir, dllFile)
			if _, err := os.Stat(srcDLL); err == nil && !fileExists(dstDLL) {
				if err := copyFile(srcDLL, dstDLL); err != nil {
					fmt.Printf("警告: 复制 %s 失败: %v\n", dllFile, err)
				} else {
					fmt.Printf("已复制 %s 到 Scripts 目录\n", dllFile)
				}
			}
		}

		fmt.Printf("已创建 Scripts 目录结构: %s\n", scriptsDir)
	}

	return absPython, nil
}

// StartOCRService 启动 OCR 服务
func StartOCRService() error {
	// 检查服务是否已经运行
	if IsOCRServiceRunning() {
		fmt.Println("OCR 服务已经在运行")
		return nil
	}

	// 检查环境：若存在内置 Python 则视为已就绪，否则检查虚拟环境
	if !(runtime.GOOS == "windows" && fileExists(filepath.Join(embedDir, "python.exe"))) {
		if !checkOCREnvironment() {
			fmt.Println("OCR 环境未设置，请先运行 ocr_setup.bat")
			return fmt.Errorf("OCR 环境未设置")
		}
	}

	fmt.Println("正在启动 OCR 服务...")

	// 构建Python可执行文件路径（优先使用内置 Python）
	var pythonExe string
	if runtime.GOOS == "windows" {
		// 优先使用内置 Python（Scripts 目录结构）
		embeddedScripts := filepath.Join(embedDir, "Scripts", "python.exe")
		if _, err := os.Stat(embeddedScripts); err == nil {
			pythonExe = embeddedScripts
		} else {
			// 回退到直接目录结构
			embedded := filepath.Join(embedDir, "python.exe")
			if _, err := os.Stat(embedded); err == nil {
				pythonExe = embedded
			} else {
				pythonExe = filepath.Join("ocr_env", "Scripts", "python.exe")
			}
		}
	} else {
		pythonExe = filepath.Join("ocr_env", "bin", "python")
	}

	// 检查Python可执行文件是否存在
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		return fmt.Errorf("未找到可用的 Python 解释器: %s", pythonExe)
	}

	// 启动 OCR 服务
	ocrProcess = exec.Command(pythonExe, "ocr_server.py")

	// 设置环境变量，确保能找到 python312.dll
	if runtime.GOOS == "windows" {
		// 获取当前工作目录
		currentDir, _ := os.Getwd()
		embedDirAbs := filepath.Join(currentDir, embedDir)

		// 设置环境变量
		env := os.Environ()
		// 将 py_embed 目录添加到 PATH 前面，确保找到 DLL
		pathEnv := fmt.Sprintf("PATH=%s;%s", embedDirAbs, os.Getenv("PATH"))
		env = append(env, pathEnv)

		// 设置 PYTHONPATH 确保模块能被找到
		pythonPath := fmt.Sprintf("PYTHONPATH=%s;%s",
			filepath.Join(embedDirAbs, "Lib"),
			filepath.Join(embedDirAbs, "Lib", "site-packages"))
		env = append(env, pythonPath)

		ocrProcess.Env = env
		ocrProcess.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	// 创建管道用于同时输出到控制台和日志文件
	_ = ensureDir("logs")
	logFile, err := os.OpenFile("logs/ocr_service.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("无法创建OCR服务日志文件: %v\n", err)
		// 即使无法创建日志文件，也继续启动服务，只输出到控制台
		ocrProcess.Stdout = os.Stdout
		ocrProcess.Stderr = os.Stderr
	} else {
		// 使用 MultiWriter 同时输出到控制台和日志文件
		multiOut := io.MultiWriter(os.Stdout, logFile)
		multiErr := io.MultiWriter(os.Stderr, logFile)
		ocrProcess.Stdout = multiOut
		ocrProcess.Stderr = multiErr
		defer logFile.Close()
	}

	err = ocrProcess.Start()
	if err != nil {
		return fmt.Errorf("启动 OCR 服务失败: %v", err)
	}

	// 等待服务启动
	fmt.Println("等待 OCR 服务初始化...")
	fmt.Println("========== OCR 服务启动日志 ==========")
	maxWait := 60 // 最多等待60秒
	for i := 0; i < maxWait; i++ {
		time.Sleep(1 * time.Second)
		if IsOCRServiceRunning() {
			fmt.Println("========== OCR 服务启动成功 ==========")
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

	// 优先尝试下载并准备内置 Python（仅 Windows）
	if runtime.GOOS == "windows" {
		if _, err := ensureEmbeddedPython(); err == nil {
			fmt.Println("已准备好内置 Python 环境")
			return nil
		} else {
			fmt.Printf("内置 Python 准备失败，回退到批处理安装: %v\n", err)
		}
	}

	// 检查是否在 Windows 系统
	if runtime.GOOS != "windows" {
		return fmt.Errorf("目前只支持 Windows 系统")
	}

	// 检查安装脚本是否存在，优先使用简化版本
	var setupScript string
	if _, err := os.Stat("ocr_setup_simple.bat"); err == nil {
		setupScript = "ocr_setup_simple.bat"
		fmt.Println("使用简化版安装脚本")
	} else if _, err := os.Stat("ocr_setup.bat"); err == nil {
		setupScript = "ocr_setup.bat"
		fmt.Println("使用标准安装脚本")
	} else {
		return fmt.Errorf("安装脚本不存在: ocr_setup.bat 或 ocr_setup_simple.bat")
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
	if !(runtime.GOOS == "windows" && fileExists(filepath.Join(embedDir, "python.exe"))) {
		if !checkOCREnvironment() {
			fmt.Println("检测到 OCR 环境未设置，正在自动设置...")
			if err := SetupOCREnvironment(); err != nil {
				return fmt.Errorf("自动设置 OCR 环境失败: %v", err)
			}
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
		"environment_ready": checkOCREnvironment() || (runtime.GOOS == "windows" && fileExists(filepath.Join(embedDir, "python.exe"))),
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
