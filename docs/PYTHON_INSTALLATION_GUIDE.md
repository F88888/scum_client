# Python 安装指南

## ⚠️ 问题描述
运行程序时出现：
```
Error: Python not found! Please install Python 3.8+
```

这表示你的系统中没有安装 Python 或者 Python 没有正确添加到 PATH 环境变量中。

## 📥 解决方案

### 方法一：官方网站下载（推荐）

1. **下载 Python 3.12.11**：
   - **64位系统**：[python-3.12.11-amd64.exe](https://www.python.org/ftp/python/3.12.11/python-3.12.11-amd64.exe)
   - **32位系统**：[python-3.12.11.exe](https://www.python.org/ftp/python/3.12.11/python-3.12.11.exe)

2. **安装步骤**：
   - 双击下载的安装包
   - ⚠️ **重要**：勾选 "Add Python to PATH" ✅
   - 点击 "Install Now"
   - 等待安装完成

3. **验证安装**：
   - 关闭并重新打开命令提示符
   - 运行：`python --version`
   - 应该显示：`Python 3.12.11`

### 方法二：Microsoft Store

1. 打开 Microsoft Store
2. 搜索 "Python 3.12"
3. 点击安装
4. 安装完成后重启命令提示符

### 方法三：Winget (Windows 10/11)

打开命令提示符或 PowerShell，运行：
```bash
winget install Python.Python.3.12
```

### 方法四：Chocolatey

如果你已经安装了 Chocolatey：
```bash
choco install python
```

## 🔧 安装后验证

运行我们提供的 Python 检查脚本：
```bash
scripts\check_python.bat
```

或手动验证：
```bash
python --version
python -m pip --version
```

## ⚡ 快速测试

安装完成后，重新运行你的程序：
```bash
scum_client.exe
```

应该看到：
```
正在提取 OCR 必需文件...
已提取文件: ocr_setup.bat
已提取文件: ocr_server.py
已提取文件: download_model.py
检查 OCR 服务状态...
开始设置 OCR 环境...
Checking for Python installation...
✅ Python found: python
✅ Version: 3.12.11
```

## 🛠️ 故障排查

### 问题：安装后仍显示 "Python not found"

**解决方案**：
1. **重启命令提示符**（重要！）
2. 检查 PATH 环境变量：
   ```bash
   echo %PATH%
   ```
   应该包含类似 `C:\Users\YourName\AppData\Local\Programs\Python\Python312\` 的路径

3. 如果 PATH 中没有 Python，手动添加：
   - 右键 "此电脑" → 属性 → 高级系统设置
   - 环境变量 → 系统变量 → Path → 编辑
   - 添加 Python 安装目录

### 问题：虚拟环境创建失败

**解决方案**：
```bash
# 手动安装到内置 Python 环境
py_embed\Scripts\python.exe -m pip install paddlepaddle==2.5.2
py_embed\Scripts\python.exe -m pip install paddleocr flask requests pillow
```

### 问题：pip 安装失败

**解决方案**：
1. 尝试不使用镜像源：
   ```bash
   pip install paddleocr flask requests pillow
   ```

2. 升级 pip：
   ```bash
   python -m pip install --upgrade pip
   ```

3. 使用不同的镜像源：
   ```bash
   pip install paddleocr flask requests pillow -i https://pypi.org/simple
   ```

## 📞 获取帮助

如果仍有问题：
1. 运行 `scripts\check_python.bat` 检查 Python 状态
2. 查看 `logs\ocr_service.log` 获取详细错误信息
3. 尝试手动运行 `ocr_setup.bat` 查看具体错误

---

**注意**：Python 3.8+ 是最低要求，但推荐使用 Python 3.12.11 以获得最佳兼容性。
