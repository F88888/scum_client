# OCR 编码问题修复说明

## 问题描述
在运行 `ocr_setup.bat` 时出现如下错误：
```
'?if' 不是内部或外部命令，也不是可运行的程序
或批处理文件。
'e.bat' 不是内部或外部命令，也不是可运行的程序
或批处理文件。
```

这是由于批处理文件编码问题导致的字符解析错误。

## 解决方案

### 1. 编码修复
- 将 `ocr_setup.bat` 改为使用 UTF-8 编码（添加 `chcp 65001`）
- 简化批处理脚本，使用纯英文提示
- 分离复杂的 Python 代码到独立的 `download_model.py` 文件

### 2. 架构改进
```
assets/
├── ocr_setup.bat      # 简化的批处理文件 (UTF-8 编码)
├── ocr_server.py      # OCR HTTP 服务
└── download_model.py  # 独立的模型下载脚本
```

### 3. 新的批处理文件特点
- **编码安全**: 使用 `chcp 65001` 设置 UTF-8 编码
- **纯英文**: 避免中文字符导致的编码问题
- **模块化**: 复杂操作分离到 Python 脚本
- **错误处理**: 更好的错误检查和提示

### 4. 使用方法
现在程序会自动提取以下文件：
- `ocr_setup.bat` (编码安全的设置脚本)
- `ocr_server.py` (OCR HTTP 服务)
- `download_model.py` (模型下载脚本)

### 5. 手动运行（如果自动脚本仍有问题）
```bash
# 1. 程序会自动下载内置 Python 到 py_embed 目录

# 2. 手动安装依赖（如果自动安装失败）
py_embed\Scripts\python.exe -m pip install paddlepaddle==2.5.2 -i https://pypi.tuna.tsinghua.edu.cn/simple
py_embed\Scripts\python.exe -m pip install paddleocr flask requests pillow -i https://pypi.tuna.tsinghua.edu.cn/simple

# 4. 创建模型目录
mkdir paddle_models

# 5. 下载模型（可选）
cd paddle_models
python ..\download_model.py
cd ..
```

### 6. 验证修复
运行程序后应该看到：
```
正在提取 OCR 必需文件...
已提取文件: ocr_setup.bat
已提取文件: ocr_server.py
已提取文件: download_model.py
检查 OCR 服务状态...
开始设置 OCR 环境...
Setting up PaddleOCR environment...
```

如果仍有编码问题，请使用手动安装方法。
