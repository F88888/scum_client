@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ========================================
echo PaddleOCR 模型修复工具
echo ========================================
echo.

REM 检查是否存在 Python 环境
set "PYTHON_CMD="
if exist "py_embed\Scripts\python.exe" (
    set "PYTHON_CMD=py_embed\Scripts\python.exe"
    echo 使用内置 Python: !PYTHON_CMD!
) else if exist "py_embed\python.exe" (
    set "PYTHON_CMD=py_embed\python.exe"
    echo 使用内置 Python: !PYTHON_CMD!
) else if exist "ocr_env\Scripts\python.exe" (
    set "PYTHON_CMD=ocr_env\Scripts\python.exe"
    echo 使用虚拟环境 Python: !PYTHON_CMD!
) else (
    echo 错误: 未找到可用的 Python 环境
    echo 请先运行 ocr_setup.bat 或 ocr_setup_simple.bat
    pause
    exit /b 1
)

echo.
echo 第一步: 检查当前模型状态
echo ----------------------------------------
if exist "check_models.py" (
    "!PYTHON_CMD!" check_models.py
) else (
    echo 未找到 check_models.py 脚本
)

echo.
echo 第二步: 下载自定义模型
echo ----------------------------------------
if exist "download_model.py" (
    if exist "paddle_models\en_PP-OCRv4_mobile_rec_infer" (
        echo 自定义模型已存在，跳过下载
    ) else (
        echo 开始下载自定义模型...
        "!PYTHON_CMD!" download_model.py
        if !errorlevel! equ 0 (
            echo ✓ 自定义模型下载成功
        ) else (
            echo ✗ 自定义模型下载失败
        )
    )
) else (
    echo 未找到 download_model.py 脚本
)

echo.
echo 第三步: 验证修复结果
echo ----------------------------------------
if exist "paddle_models\en_PP-OCRv4_mobile_rec_infer" (
    echo ✓ 自定义模型存在: paddle_models\en_PP-OCRv4_mobile_rec_infer
    echo ✓ 下次启动 OCR 服务将使用自定义模型，避免重复下载
) else (
    echo ✗ 自定义模型不存在
    echo ! 将使用系统默认模型（首次使用会自动下载）
)

echo.
echo ========================================
echo 修复完成！
echo ========================================
echo.
echo 说明:
echo - 如果存在自定义模型，OCR 服务会优先使用，避免重复下载
echo - 如果没有自定义模型，会使用系统缓存的默认模型
echo - 如果需要清理所有缓存，运行: python check_models.py --clean
echo.
pause
