@echo off
echo 正在设置 PaddleOCR 环境...
echo.

:: 检查 Python 是否已安装
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: 未找到 Python！请先安装 Python 3.8 或更高版本
    echo 下载地址: https://www.python.org/downloads/
    pause
    exit /b 1
)

:: 创建虚拟环境目录
if not exist "ocr_env" (
    echo 创建 Python 虚拟环境...
    python -m venv ocr_env
    if %errorlevel% neq 0 (
        echo 创建虚拟环境失败！
        pause
        exit /b 1
    )
)

:: 激活虚拟环境
echo 激活虚拟环境...
call ocr_env\Scripts\activate.bat

:: 升级 pip
echo 升级 pip...
python -m pip install --upgrade pip -i https://pypi.tuna.tsinghua.edu.cn/simple

:: 安装 PaddleOCR
echo 安装 PaddleOCR...
python -m pip install paddleocr -i https://pypi.tuna.tsinghua.edu.cn/simple
if %errorlevel% neq 0 (
    echo PaddleOCR 安装失败！
    pause
    exit /b 1
)

:: 安装其他依赖
echo 安装额外依赖...
python -m pip install flask requests pillow -i https://pypi.tuna.tsinghua.edu.cn/simple

:: 创建模型目录
if not exist "paddle_models" (
    echo 创建模型目录...
    mkdir paddle_models
)

:: 检查模型是否已存在
if not exist "paddle_models\en_PP-OCRv4_mobile_rec_infer" (
    echo 下载英文识别模型...
    cd paddle_models
    
    :: 使用 Python 下载模型
    python -c "
import urllib.request
import tarfile
import os

print('正在下载 en_PP-OCRv4_mobile_rec_infer 模型...')
url = 'https://paddle-model-ecology.bj.bcebos.com/paddlex/official_inference_model/paddle3.0.0/en_PP-OCRv4_mobile_rec_infer.tar'
filename = 'en_PP-OCRv4_mobile_rec_infer.tar'

try:
    urllib.request.urlretrieve(url, filename)
    print('模型下载完成，正在解压...')
    
    with tarfile.open(filename, 'r') as tar:
        tar.extractall('.')
    
    os.remove(filename)
    print('模型解压完成！')
except Exception as e:
    print(f'下载失败: {e}')
    exit(1)
"
    
    if %errorlevel% neq 0 (
        echo 模型下载失败！
        pause
        exit /b 1
    )
    
    cd ..
) else (
    echo 英文识别模型已存在，跳过下载
)

echo.
echo ================================
echo PaddleOCR 环境设置完成！
echo ================================
echo 虚拟环境路径: ocr_env
echo 模型路径: paddle_models
echo.
echo 下次运行程序时会自动启动 OCR 服务
pause 