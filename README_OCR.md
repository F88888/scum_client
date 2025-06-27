# SCUM Client - PaddleOCR 集成说明

## 概述

SCUM Client 现已集成 PaddleOCR 作为图片文字识别引擎，替换了原有的 OCR 系统。新系统提供更高的识别精度、更好的性能和完全自动化的环境配置。

## 主要特性

- ✅ **完全自动化**: 自动下载和配置 PaddleOCR 环境
- ✅ **高精度识别**: 使用 PP-OCRv4 英文移动模型
- ✅ **本地服务**: 无需外部 API，完全离线运行
- ✅ **智能管理**: 自动启动、停止和重启 OCR 服务
- ✅ **详细日志**: 完整的识别过程和结果记录

## 快速开始

### 1. 首次安装

运行自动化安装脚本：
```bash
ocr_setup.bat
```

这个脚本会自动：
- 检查 Python 环境
- 创建虚拟环境
- 安装 PaddleOCR 和依赖
- 下载英文识别模型 (约 10MB)

### 2. 编译程序

```bash
go build -o scum_client.exe
```

### 3. 启动程序

```bash
start.bat
```

程序会自动：
- 检查 OCR 环境
- 启动 OCR 服务
- 开始游戏监控

## 系统要求

### 必需环境
- **操作系统**: Windows 10/11
- **Python**: 3.8 或更高版本
- **内存**: 至少 2GB 可用内存
- **磁盘**: 至少 500MB 可用空间

### Python 安装
如果没有 Python，请从官网下载安装：
- 下载地址: https://www.python.org/downloads/
- **重要**: 安装时勾选 "Add Python to PATH"

## 文件结构

安装完成后的目录结构：
```
scum_client/
├── ocr_env/                     # Python 虚拟环境
│   ├── Scripts/
│   │   ├── python.exe
│   │   └── activate.bat
│   └── Lib/
├── paddle_models/               # PaddleOCR 模型
│   └── en_PP-OCRv4_mobile_rec_infer/
├── logs/                        # 日志文件
│   ├── scum_client_2024-01-01.log
│   └── ocr_service.log
├── ocr_setup.bat               # 环境设置脚本
├── ocr_server.py               # OCR HTTP 服务
├── start.bat                   # 程序启动脚本
├── config.yaml                 # 配置文件
└── scum_client.exe             # 主程序
```

## 使用说明

### 正常启动

运行 `start.bat` 或直接运行 `scum_client.exe`：

```
=======================================
        SCUM Client 启动程序
=======================================

=== SCUM Client 启动 ===
检查 OCR 服务状态...
正在启动 OCR 服务...
等待 OCR 服务初始化...
OCR 服务启动成功
OCR 服务已就绪
开始机器人主循环...
```

### 手动管理 OCR 环境

如果需要重新设置环境：
```bash
# 删除现有环境
rmdir /s ocr_env
rmdir /s paddle_models

# 重新运行设置
ocr_setup.bat
```

## API 接口

OCR 服务启动后提供以下 HTTP 接口：

### 健康检查
```
GET http://127.0.0.1:1224/health
```

### 文字识别
```
POST http://127.0.0.1:1224/api/ocr
Content-Type: application/json

{
    "Base64": "图片的Base64编码",
    "target_text": "期望识别的文本"  // 可选
}
```

响应格式：
```json
{
    "code": 100,                    // 100=成功, 200=识别到文字但非目标, 其他=错误
    "data": "识别结果文字",
    "message": "状态信息"
}
```

### 服务信息
```
GET http://127.0.0.1:1224/
```

## 故障排查

### 常见问题

#### 1. "未找到 Python"
**解决方案**:
- 安装 Python 3.8+ 版本
- 确保 Python 已添加到 PATH 环境变量
- 重启命令行窗口

#### 2. "创建虚拟环境失败"
**解决方案**:
```bash
# 手动创建虚拟环境
python -m venv ocr_env

# 激活环境
ocr_env\Scripts\activate.bat

# 安装依赖
pip install paddleocr flask pillow requests -i https://pypi.tuna.tsinghua.edu.cn/simple
```

#### 3. "模型下载失败"
**解决方案**:
- 检查网络连接
- 手动下载模型文件：
  ```bash
  # 进入模型目录
  cd paddle_models
  
  # 使用浏览器下载
  # https://paddle-model-ecology.bj.bcebos.com/paddlex/official_inference_model/paddle3.0.0/en_PP-OCRv4_mobile_rec_infer.tar
  
  # 解压
  tar -xf en_PP-OCRv4_mobile_rec_infer.tar
  ```

#### 4. "OCR 服务启动失败"
**解决方案**:
- 检查端口 1224 是否被占用
- 查看 OCR 服务日志：`logs/ocr_service.log`
- 手动启动服务：
  ```bash
  ocr_env\Scripts\python.exe ocr_server.py
  ```

#### 5. "图片识别错误"
**解决方案**:
- 检查游戏窗口位置和大小
- 确认游戏是英文界面
- 查看识别日志判断是否识别结果不准确

### 日志查看

#### 主程序日志
```bash
# 查看最新日志
type logs\scum_client_2024-01-01.log

# 实时监控日志
powershell Get-Content logs\scum_client_2024-01-01.log -Wait
```

#### OCR 服务日志
```bash
type logs\ocr_service.log
```

### 性能监控

#### 检查 OCR 服务状态
在浏览器中访问：http://127.0.0.1:1224/health

#### 查看服务详情
在浏览器中访问：http://127.0.0.1:1224/

## 高级配置

### 修改识别模型

如需使用其他模型，修改 `ocr_server.py`：
```python
# 在 initialize_paddleocr() 函数中修改
rec_model_dir = "paddle_models/your_custom_model"
```

### 调整服务端口

修改 `ocr_server.py` 和相关 Go 代码中的端口配置：
```python
# ocr_server.py 最后几行
app.run(
    host='127.0.0.1',
    port=1224,  # 修改此处
    debug=False,
    threaded=True
)
```

### 优化识别参数

在 `ocr_server.py` 中调整 PaddleOCR 初始化参数：
```python
ocr = PaddleOCR(
    use_angle_cls=True,
    lang='en',
    rec_model_dir=rec_model_dir,
    show_log=False,
    # 添加其他参数
    use_gpu=False,  # 是否使用GPU
    det_db_thresh=0.3,  # 检测阈值
    rec_char_type='en'  # 字符类型
)
```

## 技术详情

### 架构设计
```
Go Client (scum_client.exe)
    ↓ HTTP API
Python OCR Service (ocr_server.py)
    ↓ 调用
PaddleOCR Library
    ↓ 使用
en_PP-OCRv4_mobile_rec Model
```

### 通信流程
1. Go 客户端截取游戏界面区域
2. 转换为灰度图片并编码为 Base64
3. 通过 HTTP POST 发送给 OCR 服务
4. OCR 服务使用 PaddleOCR 进行识别
5. 返回识别结果给 Go 客户端
6. Go 客户端根据结果执行相应操作

### 模型信息
- **模型名称**: en_PP-OCRv4_mobile_rec
- **模型类型**: 英文文字识别
- **模型大小**: 约 10MB
- **特点**: 轻量级、移动端优化、适合英文界面

## 更新日志

### v1.0.0 (当前版本)
- ✅ 集成 PaddleOCR 替换原有 OCR 系统
- ✅ 实现自动化环境配置
- ✅ 添加 OCR 服务生命周期管理
- ✅ 优化图片识别 API 接口
- ✅ 完善日志系统和错误处理

## 支持与反馈

如遇到问题或需要帮助：
1. 查看日志文件确定具体错误
2. 检查环境配置是否正确
3. 确认 Python 和依赖是否正常安装
4. 验证模型文件是否完整下载

---

**注意**: 本系统专为 SCUM 游戏英文界面设计，在其他应用中可能需要调整识别参数。 