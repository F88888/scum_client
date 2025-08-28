@echo off
echo === SCUM Client OCR 构建脚本 ===
echo.

echo 1. 检查必需文件...
if not exist "assets\ocr_setup.bat" (
    echo ❌ 缺少 assets\ocr_setup.bat 文件
    pause
    exit /b 1
)

if not exist "assets\ocr_setup_simple.bat" (
    echo ❌ 缺少 assets\ocr_setup_simple.bat 文件
    pause
    exit /b 1
)

if not exist "assets\ocr_server.py" (
    echo ❌ 缺少 assets\ocr_server.py 文件
    pause
    exit /b 1
)

if not exist "assets\download_model.py" (
    echo ❌ 缺少 assets\download_model.py 文件
    pause
    exit /b 1
)

if not exist "config.yaml" (
    echo ❌ 缺少 config.yaml 文件
    pause
    exit /b 1
)

echo ✅ 所有必需文件存在

echo.
echo 2. 创建并测试嵌入文件功能...
echo package main > test_embed.go
echo. >> test_embed.go
echo import ( >> test_embed.go
echo 	"embed" >> test_embed.go
echo 	"fmt" >> test_embed.go
echo 	"os" >> test_embed.go
echo ^) >> test_embed.go
echo. >> test_embed.go
echo //go:embed assets/ocr_setup.bat assets/ocr_setup_simple.bat assets/ocr_server.py assets/download_model.py >> test_embed.go
echo var testFiles embed.FS >> test_embed.go
echo. >> test_embed.go
echo func main() { >> test_embed.go
echo 	fmt.Println("测试嵌入文件功能...") >> test_embed.go
echo 	files := []string{"assets/ocr_setup.bat", "assets/ocr_setup_simple.bat", "assets/ocr_server.py", "assets/download_model.py"} >> test_embed.go
echo 	for _, fileName := range files { >> test_embed.go
echo 		content, err := testFiles.ReadFile(fileName) >> test_embed.go
echo 		if err != nil { >> test_embed.go
echo 			fmt.Printf("❌ 读取失败: %%s\n", fileName) >> test_embed.go
echo 			os.Exit(1) >> test_embed.go
echo 		} >> test_embed.go
echo 		fmt.Printf("✅ 成功读取: %%s (%%d 字节)\n", fileName, len(content)) >> test_embed.go
echo 	} >> test_embed.go
echo } >> test_embed.go

go build -o test_embed.exe test_embed.go
if %errorlevel% neq 0 (
    echo ❌ 测试程序编译失败
    pause
    exit /b 1
)

echo 运行嵌入文件测试...
test_embed.exe
if %errorlevel% neq 0 (
    echo ❌ 嵌入文件测试失败
    pause
    exit /b 1
)

echo ✅ 嵌入文件功能正常

echo.
echo 3. 构建主程序...
go build -o scum_client.exe main.go
if %errorlevel% neq 0 (
    echo ❌ 主程序编译失败
    pause
    exit /b 1
)

echo ✅ 主程序编译成功

echo.
echo 4. 清理测试文件...
if exist "test_embed.exe" del test_embed.exe
if exist "test_embed.go" del test_embed.go

echo.
echo === 构建完成 ===
echo.
echo 生成的可执行文件: scum_client.exe
echo.
echo 该可执行文件已包含以下嵌入文件:
echo   - config.yaml (配置文件)
echo   - ocr_setup.bat (OCR 环境设置脚本)
echo   - ocr_server.py (OCR HTTP 服务)
echo   - download_model.py (模型下载脚本)
echo.
echo 使用说明:
echo   1. 确保系统已安装 Python 3.8+
echo      - 如未安装，运行: scripts\install_python.ps1 (需管理员权限)
echo      - 或手动下载: https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe
echo   2. 将 scum_client.exe 复制到任意目录
echo   3. 双击运行，程序会自动提取必需的 OCR 文件
echo   4. 首次运行会自动设置 OCR 环境
echo   5. 后续运行将直接启动 OCR 服务
echo.
echo 故障排查:
echo   - 如遇到 "Python not found" 错误，运行: scripts\check_python.bat
echo   - 查看详细安装指南: docs\PYTHON_INSTALLATION_GUIDE.md
echo.
pause
