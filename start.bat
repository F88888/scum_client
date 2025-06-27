@echo off
echo =======================================
echo         SCUM Client 启动程序
echo =======================================
echo.

:: 检查是否存在配置文件
if not exist "config.yaml" (
    echo 错误: 未找到配置文件 config.yaml
    echo 请确保配置文件存在并正确配置
    pause
    exit /b 1
)

:: 检查是否存在可执行文件
if not exist "scum_client.exe" (
    echo 错误: 未找到可执行文件 scum_client.exe
    echo 请先编译程序: go build -o scum_client.exe
    pause
    exit /b 1
)

echo 正在启动 SCUM Client...
echo.
echo 提示:
echo - 程序会自动检查并设置 OCR 环境
echo - 如果首次运行，会自动下载 PaddleOCR 模型
echo - 按 Ctrl+C 可以安全退出程序
echo.

:: 启动程序
scum_client.exe

echo.
echo 程序已退出
pause 